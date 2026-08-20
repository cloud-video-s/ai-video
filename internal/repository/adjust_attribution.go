package repository

import (
	"context"
	"errors"
	"strings"

	"ai-video/internal/gen/model"

	"gorm.io/gen/field"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	AdjustMatchStatusPendingApp      = 1
	AdjustMatchStatusPendingCallback = 2
	AdjustMatchStatusFused           = 3
)

type AdjustAttributionRepo struct{}

func NewAdjustAttributionRepo() *AdjustAttributionRepo {
	return &AdjustAttributionRepo{}
}

func (r *AdjustAttributionRepo) Create(ctx context.Context, item *model.VideoAdjustAttribution) error {
	q := qFrom(ctx).VideoAdjustAttribution
	return q.WithContext(ctx).Create(item)
}

func (r *AdjustAttributionRepo) GetByADID(
	ctx context.Context, adid string, lock bool,
) (*model.VideoAdjustAttribution, error) {
	q := qFrom(ctx).VideoAdjustAttribution
	dao := q.WithContext(ctx).Where(q.AdjustADID.Eq(adid))
	if lock {
		dao = dao.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	return dao.First()
}

func (r *AdjustAttributionRepo) GetByUserID(
	ctx context.Context, userID uint64, lock bool,
) (*model.VideoAdjustAttribution, error) {
	q := qFrom(ctx).VideoAdjustAttribution
	dao := q.WithContext(ctx).Where(q.UserID.Eq(userID))
	if lock {
		dao = dao.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	return dao.First()
}

func (r *AdjustAttributionRepo) Update(ctx context.Context, id uint64, updates map[string]any) error {
	if len(updates) == 0 {
		return nil
	}
	q := qFrom(ctx).VideoAdjustAttribution
	_, err := q.WithContext(ctx).Where(q.ID.Eq(id)).Updates(updates)
	return err
}

func (r *AdjustAttributionRepo) ResolveChannelID(ctx context.Context, channelCode string) (uint64, error) {
	channelCode = strings.TrimSpace(channelCode)
	if channelCode == "" {
		return 0, nil
	}
	q := qFrom(ctx).VideoChannel
	row, err := q.WithContext(ctx).Where(q.ChannelCode.Eq(channelCode)).First()
	if err != nil {
		return 0, err
	}
	return row.ID, nil
}

func (r *AdjustAttributionRepo) ResolveMedia(ctx context.Context, network string) (uint64, uint64, error) {
	network = strings.TrimSpace(network)
	if network == "" || strings.EqualFold(network, "organic") {
		return 0, 0, nil
	}
	q := qFrom(ctx).VideoMedium
	row, err := q.WithContext(ctx).
		Where(field.NewUnsafeFieldRaw("INSTR(LOWER(?), LOWER(`name`)) > 0", network)).
		Where(q.Status.Eq(1)).
		Order(q.ID.Asc()).First()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, 0, nil
	}
	if err != nil {
		return 0, 0, err
	}
	return row.ID, row.AdjustPartnerID, nil
}

// ForgetDevice removes the device-level fusion payload while retaining the
// already selected video_user_attribution snapshot.
func (r *AdjustAttributionRepo) ForgetDevice(ctx context.Context, adid string) error {
	fusion, err := r.GetByADID(ctx, adid, true)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	q := qFrom(ctx).VideoAdjustAttribution
	_, err = q.WithContext(ctx).Unscoped().Where(q.ID.Eq(fusion.ID)).Delete(fusion)
	return err
}

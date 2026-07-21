package repository

import (
	"context"
	"strings"
	"time"

	"ai-video/internal/domain"
	"ai-video/internal/gen/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UserAttributionRepo struct{}

func NewUserAttributionRepo() *UserAttributionRepo { return &UserAttributionRepo{} }

type UserAttributionListFilter struct {
	Keyword     string
	ChannelCode string
	Event       string
	Reached     *bool
	StartedAt   *time.Time
	EndedAt     *time.Time
}

func (r *UserAttributionRepo) PageList(
	ctx context.Context, page, pageSize int, filter *UserAttributionListFilter,
) ([]model.VideoUserAttribution, int64, error) {
	db := dbFrom(ctx).Model(&model.VideoUserAttribution{}).
		Joins("JOIN video_user AS attribution_user ON attribution_user.id = video_user_attribution.user_id AND attribution_user.deleted_at IS NULL")
	if filter != nil {
		if filter.Keyword != "" {
			keyword := "%" + filter.Keyword + "%"
			db = db.Where(
				"video_user_attribution.oaid LIKE ? OR video_user_attribution.imei LIKE ? OR "+
					"video_user_attribution.android_id LIKE ? OR video_user_attribution.ip LIKE ? OR "+
					"attribution_user.username LIKE ? OR attribution_user.imei LIKE ?",
				keyword, keyword, keyword, keyword, keyword, keyword,
			)
		}
		if filter.ChannelCode != "" {
			db = db.Where(
				"video_user_attribution.channel_code = ? OR (video_user_attribution.channel_code = '' AND attribution_user.channel_id = ?)",
				filter.ChannelCode, filter.ChannelCode,
			)
		}
		if filter.StartedAt != nil {
			db = db.Where("video_user_attribution.attributed_at >= ?", *filter.StartedAt)
		}
		if filter.EndedAt != nil {
			db = db.Where("video_user_attribution.attributed_at <= ?", *filter.EndedAt)
		}
		if filter.Reached != nil {
			if column := reachedColumn(filter.Event); column != "" {
				value := interface{}(*filter.Reached)
				if filter.Event == domain.AttributionEventActivation || filter.Event == domain.AttributionEventKeyBehavior {
					value = uint32(0)
					if *filter.Reached {
						value = uint32(1)
					}
				}
				db = db.Where("attribution_user."+column+" = ?", value)
			}
		}
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.VideoUserAttribution
	err := db.Select("video_user_attribution.*").Preload("User").
		Order("video_user_attribution.id DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}

func (r *UserAttributionRepo) GetByID(ctx context.Context, id uint64, lock bool) (*model.VideoUserAttribution, error) {
	var item model.VideoUserAttribution
	db := dbFrom(ctx).Preload("User")
	if lock {
		db = db.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	if err := db.First(&item, id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *UserAttributionRepo) Update(ctx context.Context, item *model.VideoUserAttribution) error {
	return dbFrom(ctx).Model(item).Select(
		"ChannelCode", "OAID", "IMEI", "AndroidID", "IP", "UserAgent", "AttributedAt", "Remark",
	).Updates(item).Error
}

func (r *UserAttributionRepo) UpsertDevice(
	ctx context.Context, userID uint64, updates map[string]interface{},
) error {
	row := model.VideoUserAttribution{UserID: userID}
	if err := dbFrom(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		DoNothing: true,
	}).Create(&row).Error; err != nil {
		return err
	}
	if len(updates) == 0 {
		return nil
	}
	return dbFrom(ctx).Model(&model.VideoUserAttribution{}).Where("user_id = ?", userID).Updates(updates).Error
}

func (r *UserAttributionRepo) IncrementEvent(
	ctx context.Context, id uint64, column string, now time.Time,
) error {
	allowed := map[string]bool{
		"activation_callback_count": true, "activation_deduct_count": true,
		"key_behavior_callback_count": true, "key_behavior_deduct_count": true,
		"payment_callback_count": true, "payment_deduct_count": true,
		"first_payment_callback_count": true, "first_payment_deduct_count": true,
		"registration_callback_count": true, "registration_deduct_count": true,
	}
	if !allowed[column] {
		return gorm.ErrInvalidField
	}
	return dbFrom(ctx).Model(&model.VideoUserAttribution{}).Where("id = ?", id).Updates(map[string]interface{}{
		column:             gorm.Expr(column + " + 1"),
		"last_operated_at": now,
	}).Error
}

func (r *UserAttributionRepo) SyncUsers(ctx context.Context) (int64, error) {
	var total int64
	var cursor uint64
	for {
		var users []model.VideoUser
		err := dbFrom(ctx).Where(
			"id > ? AND NOT EXISTS (SELECT 1 FROM video_user_attribution a WHERE a.user_id = video_user.id)",
			cursor,
		).Order("id ASC").Limit(500).Find(&users).Error
		if err != nil {
			return total, err
		}
		if len(users) == 0 {
			return total, nil
		}
		rows := make([]model.VideoUserAttribution, 0, len(users))
		for i := range users {
			attributedAt := users[i].AttributionClickedAt
			if !attributedAt.IsZero() {
				attributedAt = users[i].FirstOpenedAt
			}
			rows = append(rows, model.VideoUserAttribution{
				UserID: users[i].ID, ChannelCode: users[i].ChannelID,
				IMEI: users[i].IMEI, IP: users[i].LastLoginIP, AttributedAt: attributedAt,
			})
			cursor = users[i].ID
		}
		result := dbFrom(ctx).Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}},
			DoNothing: true,
		}).CreateInBatches(&rows, 500)
		if result.Error != nil {
			return total, result.Error
		}
		total += result.RowsAffected
	}
}

func reachedColumn(event string) string {
	switch strings.TrimSpace(event) {
	case domain.AttributionEventActivation:
		return "activated"
	case domain.AttributionEventKeyBehavior:
		return "key_behavior_met"
	case domain.AttributionEventPayment:
		return "payment_met"
	case domain.AttributionEventFirstPayment:
		return "first_payment_met"
	case domain.AttributionEventRegistration:
		return "registered"
	default:
		return ""
	}
}

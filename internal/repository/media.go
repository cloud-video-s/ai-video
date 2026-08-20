package repository

import (
	"context"

	"ai-video/internal/gen/model"
)

// MediaOption is the read-only projection used by channel media selectors.
// Keep the select list explicit because the generated VideoMedium model may
// contain fields that are not present in every deployed video_media schema.
type MediaOption struct {
	ID              uint64 `gorm:"column:id" json:"id"`
	Name            string `gorm:"column:name" json:"name"`
	AdjustPartnerID uint64 `gorm:"column:adjust_partner_id" json:"adjust_partner_id"`
}

type MediaRepo struct{}

func NewMediaRepo() *MediaRepo {
	return &MediaRepo{}
}

func (r *MediaRepo) ListOptions(ctx context.Context) ([]*model.VideoMedium, error) {
	q := qFrom(ctx).VideoMedium
	find, err := q.WithContext(ctx).Where(q.Status.Eq(1)).Find()
	return find, err
}

func (r *MediaRepo) ExistsByName(ctx context.Context, name string) (bool, error) {
	q := qFrom(ctx).VideoMedium
	count, err := q.WithContext(ctx).Where(q.Status.Eq(1)).Where(q.Name.Eq(name)).Count()
	return count > 0, err
}

func (r *MediaRepo) GetByName(ctx context.Context, name string) (*model.VideoMedium, error) {
	q := qFrom(ctx).VideoMedium
	return q.WithContext(ctx).Where(q.Status.Eq(1), q.Name.Eq(name)).First()
}

func (r *MediaRepo) GetLikeName(ctx context.Context, name string) (*model.VideoMedium, error) {
	q := qFrom(ctx).VideoMedium
	return q.WithContext(ctx).Where(q.Status.Eq(1)).Where(q.Name.Like("%" + name + "%")).First()
}

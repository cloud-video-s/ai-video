package repository

import (
	"context"

	"ai-video/internal/gen/model"

	"gorm.io/gen/field"
)

type PlatformRepo struct {
	BaseRepo[model.VideoPlatform]
}

func NewPlatformRepo() *PlatformRepo { return &PlatformRepo{} }

type PlatformListFilter struct {
	Keyword string
	Status  *int32
}

func (r *PlatformRepo) PageList(ctx context.Context, page, pageSize int, filter *PlatformListFilter) ([]model.VideoPlatform, int64, error) {
	q := qFrom(ctx).VideoPlatform
	dao := q.WithContext(ctx)
	if filter != nil {
		if filter.Status != nil {
			dao = dao.Where(q.Status.Eq(*filter.Status))
		}
		if filter.Keyword != "" {
			keyword := "%" + filter.Keyword + "%"
			dao = dao.Where(field.Or(q.Name.Like(keyword), q.Code.Like(keyword), q.BaseURL.Like(keyword)))
		}
	}
	total, err := dao.Count()
	if err != nil {
		return nil, 0, err
	}
	rows, err := dao.Order(q.ID.Desc()).Offset((page - 1) * pageSize).Limit(pageSize).Find()
	return valuesOf(rows), total, err
}

func (r *PlatformRepo) ListOptions(ctx context.Context) ([]model.VideoPlatform, error) {
	q := qFrom(ctx).VideoPlatform
	rows, err := q.WithContext(ctx).Where(q.Status.Eq(1)).Order(q.Name.Asc(), q.ID.Asc()).Find()
	return valuesOf(rows), err
}

func (r *PlatformRepo) GetByCode(ctx context.Context, code string) (*model.VideoPlatform, error) {
	q := qFrom(ctx).VideoPlatform
	return q.WithContext(ctx).Where(q.Code.Eq(code)).First()
}

func (r *PlatformRepo) UpdateFields(ctx context.Context, item *model.VideoPlatform) error {
	q := qFrom(ctx).VideoPlatform
	_, err := q.WithContext(ctx).Where(q.ID.Eq(item.ID)).Select(
		q.Name, q.Code, q.BaseURL, q.Description, q.Status,
	).Updates(item)
	return err
}

func (r *PlatformRepo) ModelCount(ctx context.Context, platformID int64) (int64, error) {
	q := qFrom(ctx).VideoModel
	return q.WithContext(ctx).Where(q.PlatformID.Eq(platformID)).Count()
}

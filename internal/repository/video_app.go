package repository

import (
	"context"
	"strings"

	"ai-video/internal/gen/model"

	"gorm.io/gen/field"
)

type VideoAppRepo struct{ BaseRepo[model.VideoApp] }

func NewVideoAppRepo() *VideoAppRepo { return &VideoAppRepo{} }

type VideoAppListFilter struct {
	Keyword string
	AppCode string
	Status  *uint32
}

func (r *VideoAppRepo) PageList(ctx context.Context, page, pageSize int, filter *VideoAppListFilter) ([]model.VideoApp, int64, error) {
	q := qFrom(ctx).VideoApp
	dao := q.WithContext(ctx)
	if filter != nil {
		if filter.AppCode != "" {
			dao = dao.Where(q.AppCode.Eq(filter.AppCode))
		}
		if filter.Status != nil {
			dao = dao.Where(q.Status.Eq(uint8(*filter.Status)))
		}
		if keyword := strings.TrimSpace(filter.Keyword); keyword != "" {
			value := "%" + keyword + "%"
			dao = dao.Where(field.Or(q.Name.Like(value), q.AppCode.Like(value), q.Description.Like(value)))
		}
	}
	total, err := dao.Count()
	if err != nil {
		return nil, 0, err
	}
	rows, err := dao.Order(q.Sort.Asc(), q.ID.Desc()).Offset((page - 1) * pageSize).Limit(pageSize).Find()
	return valuesOf(rows), total, err
}

func (r *VideoAppRepo) GetByAppCode(ctx context.Context, appID uint64) (*model.VideoApp, error) {
	q := qFrom(ctx).VideoApp
	return q.WithContext(ctx).Where(q.ID.Eq(appID)).First()
}

func (r *VideoAppRepo) ListOptions(ctx context.Context) ([]model.VideoApp, error) {
	q := qFrom(ctx).VideoApp
	rows, err := q.WithContext(ctx).Order(q.Sort.Asc(), q.ID.Asc()).Find()
	return valuesOf(rows), err
}

func (r *VideoAppRepo) PackageCount(ctx context.Context, appCode string) (int64, error) {
	q := qFrom(ctx).VideoPackage
	return q.WithContext(ctx).Where(q.AppCode.Eq(appCode)).Count()
}

func (r *VideoAppRepo) UpdateFields(ctx context.Context, app *model.VideoApp) error {
	q := qFrom(ctx).VideoApp
	_, err := q.WithContext(ctx).Where(q.ID.Eq(app.ID)).Select(q.Name, q.AppCode, q.Status, q.Sort, q.Description).Updates(app)
	return err
}

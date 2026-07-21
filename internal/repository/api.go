package repository

import (
	"context"

	"ai-video/internal/gen/model"
)

type ApiRepo struct{}

func NewApiRepo() *ApiRepo {
	return &ApiRepo{}
}

func (d *ApiRepo) Create(ctx context.Context, api *model.VideoAPI) error {
	return qFrom(ctx).VideoAPI.WithContext(ctx).UnderlyingDB().Create(api).Error
}

func (d *ApiRepo) GetByID(ctx context.Context, id uint) (*model.VideoAPI, error) {
	var api model.VideoAPI
	q := qFrom(ctx).VideoAPI
	err := q.WithContext(ctx).Where(q.ID.Eq(uint64(id))).UnderlyingDB().First(&api).Error
	if err != nil {
		return nil, err
	}
	return &api, nil
}

func (d *ApiRepo) Update(ctx context.Context, api *model.VideoAPI) error {
	q := qFrom(ctx).VideoAPI
	return q.WithContext(ctx).Where(q.ID.Eq(uint64(api.ID))).UnderlyingDB().Save(api).Error
}

func (d *ApiRepo) Delete(ctx context.Context, id uint) error {
	q := qFrom(ctx).VideoAPI
	return q.WithContext(ctx).Where(q.ID.Eq(uint64(id))).UnderlyingDB().Delete(&model.VideoAPI{}).Error
}

func (d *ApiRepo) PageList(ctx context.Context, page, pageSize int, _ *QueryOptions) ([]model.VideoAPI, int64, error) {
	q := qFrom(ctx).VideoAPI
	dao := q.WithContext(ctx).Order(q.Group.Asc(), q.ID.Asc())
	total, err := dao.Count()
	if err != nil {
		return nil, 0, err
	}
	var apis []model.VideoAPI
	if err := dao.Offset((page - 1) * pageSize).Limit(pageSize).Scan(&apis); err != nil {
		return nil, 0, err
	}
	return apis, total, nil
}

func (d *ApiRepo) ListAll(ctx context.Context) ([]model.VideoAPI, error) {
	var apis []model.VideoAPI
	q := qFrom(ctx).VideoAPI
	if err := q.WithContext(ctx).Order(q.Group.Asc(), q.ID.Asc()).Scan(&apis); err != nil {
		return nil, err
	}
	return apis, nil
}

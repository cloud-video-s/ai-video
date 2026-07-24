package repository

import (
	"context"

	"ai-video/internal/gen/model"

	"gorm.io/gen/field"
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
	return Transaction(ctx, func(txCtx context.Context) error {
		q := qFrom(txCtx)
		relation := q.VideoMenuAPI
		if _, err := relation.WithContext(txCtx).Unscoped().
			Where(relation.VideoAPIID.Eq(uint64(id))).Delete(); err != nil {
			return err
		}
		api := q.VideoAPI
		_, err := api.WithContext(txCtx).Where(api.ID.Eq(uint64(id))).Delete()
		return err
	})
}

type APIListFilter struct {
	Group   string
	Method  string
	Keyword string
}

func (d *ApiRepo) PageList(ctx context.Context, page, pageSize int, filter *APIListFilter) ([]model.VideoAPI, int64, error) {
	q := qFrom(ctx).VideoAPI
	dao := q.WithContext(ctx)
	if filter != nil {
		if filter.Group != "" {
			dao = dao.Where(q.Group.Eq(filter.Group))
		}
		if filter.Method != "" {
			dao = dao.Where(q.Method.Eq(filter.Method))
		}
		if filter.Keyword != "" {
			keyword := "%" + filter.Keyword + "%"
			dao = dao.Where(field.Or(
				q.Path.Like(keyword), q.Group.Like(keyword), q.Description.Like(keyword),
			))
		}
	}
	total, err := dao.Count()
	if err != nil {
		return nil, 0, err
	}
	var apis []model.VideoAPI
	if err := dao.Order(q.Group.Asc(), q.ID.Asc()).
		Offset((page - 1) * pageSize).Limit(pageSize).Scan(&apis); err != nil {
		return nil, 0, err
	}
	return apis, total, nil
}

func (d *ApiRepo) Exists(ctx context.Context, path, method string, excludeID uint64) (bool, error) {
	q := qFrom(ctx).VideoAPI
	dao := q.WithContext(ctx).Where(q.Path.Eq(path), q.Method.Eq(method))
	if excludeID != 0 {
		dao = dao.Where(q.ID.Neq(excludeID))
	}
	count, err := dao.Count()
	return count > 0, err
}

func (d *ApiRepo) ListAll(ctx context.Context) ([]model.VideoAPI, error) {
	var apis []model.VideoAPI
	q := qFrom(ctx).VideoAPI
	if err := q.WithContext(ctx).Order(q.Group.Asc(), q.ID.Asc()).Scan(&apis); err != nil {
		return nil, err
	}
	return apis, nil
}

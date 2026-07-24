package repository

import (
	"context"

	"ai-video/internal/gen/model"

	"gorm.io/gen/field"
)

type VIPSubscriptionLevelRepo struct {
	BaseRepo[model.VideoVipSubscriptionLevel]
}

func NewVIPSubscriptionLevelRepo() *VIPSubscriptionLevelRepo {
	return &VIPSubscriptionLevelRepo{}
}

type VIPSubscriptionLevelListFilter struct {
	Status  *uint32
	Keyword string
}

func (r *VIPSubscriptionLevelRepo) PageList(ctx context.Context, page, pageSize int, filter *VIPSubscriptionLevelListFilter) ([]model.VideoVipSubscriptionLevel, int64, error) {
	q := qFrom(ctx).VideoVipSubscriptionLevel
	dao := q.WithContext(ctx)
	if filter != nil {
		if filter.Status != nil {
			dao = dao.Where(q.Status.Eq(*filter.Status))
		}
		if filter.Keyword != "" {
			keyword := "%" + filter.Keyword + "%"
			dao = dao.Where(field.Or(q.Level.Like(keyword), q.Description.Like(keyword)))
		}
	}
	total, err := dao.Count()
	if err != nil {
		return nil, 0, err
	}
	rows, err := dao.Order(q.Sort.Asc(), q.ID.Desc()).Offset((page - 1) * pageSize).Limit(pageSize).Find()
	return valuesOf(rows), total, err
}

func (r *VIPSubscriptionLevelRepo) ListOptions(ctx context.Context) ([]model.VideoVipSubscriptionLevel, error) {
	q := qFrom(ctx).VideoVipSubscriptionLevel
	rows, err := q.WithContext(ctx).Where(q.Status.Eq(1)).Order(q.Sort.Asc(), q.ID.Asc()).Find()
	return valuesOf(rows), err
}

func (r *VIPSubscriptionLevelRepo) UpdateFields(ctx context.Context, item *model.VideoVipSubscriptionLevel) error {
	q := qFrom(ctx).VideoVipSubscriptionLevel
	_, err := q.WithContext(ctx).Where(q.ID.Eq(item.ID)).Select(
		q.Level, q.Description, q.Status, q.Sort,
	).Updates(item)
	return err
}

func (r *VIPSubscriptionLevelRepo) SubscriptionCount(ctx context.Context, id uint64) (int64, error) {
	q := qFrom(ctx).VideoVipSubscription
	return q.WithContext(ctx).Where(q.LevelID.Eq(id)).Count()
}

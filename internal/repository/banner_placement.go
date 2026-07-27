package repository

import (
	"context"

	"ai-video/internal/gen/model"

	"gorm.io/gen/field"
)

type BannerPlacementRepo struct {
	BaseRepo[model.VideoBannerPlacement]
}

func NewBannerPlacementRepo() *BannerPlacementRepo { return &BannerPlacementRepo{} }

type BannerPlacementListFilter struct {
	Status  *int8
	Keyword string
}

func (r *BannerPlacementRepo) PageList(ctx context.Context, page, pageSize int, filter *BannerPlacementListFilter) ([]model.VideoBannerPlacement, int64, error) {
	q := qFrom(ctx).VideoBannerPlacement
	dao := q.WithContext(ctx)
	if filter != nil {
		if filter.Status != nil {
			dao = dao.Where(q.Status.Eq(*filter.Status))
		}
		if filter.Keyword != "" {
			keyword := "%" + filter.Keyword + "%"
			dao = dao.Where(field.Or(q.PlacementName.Like(keyword), q.PlacementKey.Like(keyword), q.Description.Like(keyword)))
		}
	}
	total, err := dao.Count()
	if err != nil {
		return nil, 0, err
	}
	rows, err := dao.Order(q.Sort.Asc(), q.ID.Desc()).Offset((page - 1) * pageSize).Limit(pageSize).Find()
	return valuesOf(rows), total, err
}

func (r *BannerPlacementRepo) ListOptions(ctx context.Context) ([]model.VideoBannerPlacement, error) {
	q := qFrom(ctx).VideoBannerPlacement
	rows, err := q.WithContext(ctx).Where(q.Status.Eq(1)).Order(q.Sort.Asc(), q.ID.Asc()).Find()
	return valuesOf(rows), err
}

func (r *BannerPlacementRepo) GetByKey(ctx context.Context, key string) (*model.VideoBannerPlacement, error) {
	q := qFrom(ctx).VideoBannerPlacement
	return q.WithContext(ctx).Where(q.PlacementKey.Eq(key)).First()
}

func (r *BannerPlacementRepo) UpdateFields(ctx context.Context, item *model.VideoBannerPlacement) error {
	q := qFrom(ctx).VideoBannerPlacement
	_, err := q.WithContext(ctx).Where(q.ID.Eq(item.ID)).Select(
		q.PlacementName, q.PlacementKey, q.Description, q.CoverImage, q.Sort, q.Status,
	).Updates(item)
	return err
}

func (r *BannerPlacementRepo) BannerCount(ctx context.Context, placementKey string) (int64, error) {
	q := qFrom(ctx).VideoBannerPlacementAssociation
	return q.WithContext(ctx).Where(q.PlacementKey.Eq(placementKey)).Count()
}

func (r *BannerPlacementRepo) RenameAssociationKey(ctx context.Context, oldKey, newKey string) error {
	if oldKey == newKey {
		return nil
	}
	q := qFrom(ctx).VideoBannerPlacementAssociation
	_, err := q.WithContext(ctx).Where(q.PlacementKey.Eq(oldKey)).Update(q.PlacementKey, newKey)
	return err
}

func (r *BannerPlacementRepo) ListForBannerIDs(ctx context.Context, bannerIDs []uint64) (map[uint64][]model.VideoBannerPlacement, error) {
	result := make(map[uint64][]model.VideoBannerPlacement, len(bannerIDs))
	bannerIDs = uniqueUint64s(bannerIDs)
	if len(bannerIDs) == 0 {
		return result, nil
	}

	q := qFrom(ctx)
	relationDAO := q.VideoBannerPlacementAssociation
	relations, err := relationDAO.WithContext(ctx).
		Where(relationDAO.BannerID.In(bannerIDs...)).
		Order(relationDAO.BannerID.Asc(), relationDAO.PlacementKey.Asc()).Find()
	if err != nil {
		return nil, err
	}
	keys := make([]string, 0, len(relations))
	for _, relation := range relations {
		keys = append(keys, relation.PlacementKey)
	}
	keys = sortedUniqueStrings(keys)
	if len(keys) == 0 {
		return result, nil
	}

	placementDAO := q.VideoBannerPlacement
	placements, err := placementDAO.WithContext(ctx).Where(placementDAO.PlacementKey.In(keys...)).Find()
	if err != nil {
		return nil, err
	}
	byKey := make(map[string]model.VideoBannerPlacement, len(placements))
	for _, placement := range placements {
		byKey[placement.PlacementKey] = *placement
	}
	for _, relation := range relations {
		if placement, ok := byKey[relation.PlacementKey]; ok {
			result[relation.BannerID] = append(result[relation.BannerID], placement)
		}
	}
	return result, nil
}

package repository

import (
	"context"

	"ai-video/internal/gen/model"

	"gorm.io/gen/field"
)

type DisplayPositionRepo struct {
	BaseRepo[model.VideoDisplayPosition]
}

func NewDisplayPositionRepo() *DisplayPositionRepo { return &DisplayPositionRepo{} }

type DisplayPositionListFilter struct {
	Status  *int8
	Keyword string
}

func (r *DisplayPositionRepo) PageList(ctx context.Context, page, pageSize int, filter *DisplayPositionListFilter) ([]model.VideoDisplayPosition, int64, error) {
	q := qFrom(ctx).VideoDisplayPosition
	dao := q.WithContext(ctx)
	if filter != nil {
		if filter.Status != nil {
			dao = dao.Where(q.Status.Eq(*filter.Status))
		}
		if filter.Keyword != "" {
			keyword := "%" + filter.Keyword + "%"
			dao = dao.Where(field.Or(q.PositionName.Like(keyword), q.PositionKey.Like(keyword), q.Description.Like(keyword)))
		}
	}
	total, err := dao.Count()
	if err != nil {
		return nil, 0, err
	}
	rows, err := dao.Order(q.Sort.Asc(), q.ID.Desc()).Offset((page - 1) * pageSize).Limit(pageSize).Find()
	return valuesOf(rows), total, err
}

func (r *DisplayPositionRepo) ListOptions(ctx context.Context) ([]model.VideoDisplayPosition, error) {
	q := qFrom(ctx).VideoDisplayPosition
	rows, err := q.WithContext(ctx).Where(q.Status.Eq(1)).Order(q.Sort.Asc(), q.ID.Asc()).Find()
	return valuesOf(rows), err
}

func (r *DisplayPositionRepo) GetByKey(ctx context.Context, key string) (*model.VideoDisplayPosition, error) {
	q := qFrom(ctx).VideoDisplayPosition
	return q.WithContext(ctx).Where(q.PositionKey.Eq(key)).First()
}

func (r *DisplayPositionRepo) GetTemplatePlacementByKey(ctx context.Context, key string) (*model.VideoTemplatePlacement, error) {
	q := qFrom(ctx).VideoTemplatePlacement
	return q.WithContext(ctx).Where(q.PlacementKey.Eq(key)).First()
}

func (r *DisplayPositionRepo) UpdateFields(ctx context.Context, item *model.VideoDisplayPosition) error {
	q := qFrom(ctx).VideoDisplayPosition
	_, err := q.WithContext(ctx).Where(q.ID.Eq(item.ID)).Select(
		q.PositionName, q.PositionKey, q.Description, q.CoverImage, q.Sort, q.Status,
	).Updates(item)
	return err
}

func (r *DisplayPositionRepo) BannerCount(ctx context.Context, positionKey string) (int64, error) {
	q := qFrom(ctx).VideoBannerPlacementAssociation
	return q.WithContext(ctx).Where(q.PlacementKey.Eq(positionKey)).Count()
}

func (r *DisplayPositionRepo) TemplateTypeCount(ctx context.Context, positionKey string) (int64, error) {
	q := qFrom(ctx).VideoTemplateTypeDisplayPosition
	return q.WithContext(ctx).Where(q.PositionKey.Eq(positionKey)).Count()
}

func (r *DisplayPositionRepo) TemplateDisplayConfigCount(ctx context.Context, positionKey string) (int64, error) {
	q := qFrom(ctx).VideoTemplatePlacementConfig
	return q.WithContext(ctx).Where(q.PlacementKey.Eq(positionKey)).Count()
}

func (r *DisplayPositionRepo) RenameTemplateTypePositionKey(ctx context.Context, oldKey, newKey string) error {
	if oldKey == newKey {
		return nil
	}
	q := qFrom(ctx)
	typePosition := q.VideoTemplateTypeDisplayPosition
	if _, err := typePosition.WithContext(ctx).Where(typePosition.PositionKey.Eq(oldKey)).
		Update(typePosition.PositionKey, newKey); err != nil {
		return err
	}
	bannerPosition := q.VideoBannerPlacementAssociation
	if _, err := bannerPosition.WithContext(ctx).Where(bannerPosition.PlacementKey.Eq(oldKey)).
		Update(bannerPosition.PlacementKey, newKey); err != nil {
		return err
	}
	templatePosition := q.VideoTemplatePlacementConfig
	_, err := templatePosition.WithContext(ctx).Where(templatePosition.PlacementKey.Eq(oldKey)).
		Update(templatePosition.PlacementKey, newKey)
	return err
}

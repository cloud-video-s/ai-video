package repository

import (
	"context"

	"ai-video/internal/gen/model"

	"gorm.io/gen/field"
)

type TemplateDisplayConfigRepo struct {
	BaseRepo[model.VideoTemplatePlacementConfig]
}

func NewTemplateDisplayConfigRepo() *TemplateDisplayConfigRepo {
	return &TemplateDisplayConfigRepo{}
}

// Create 显式写入 Status，确保 0 值不会被数据库默认值替换。
func (r *TemplateDisplayConfigRepo) Create(ctx context.Context, item *model.VideoTemplatePlacementConfig) error {
	q := qFrom(ctx).VideoTemplatePlacementConfig
	return q.WithContext(ctx).Select(
		q.TemplateID, q.PlacementKey, q.Sort, q.Status, q.Description, q.CreatedAt, q.UpdatedAt,
	).Create(item)
}

type TemplateDisplayConfigListFilter struct {
	TemplateID          uint64
	VideoTemplateTypeID uint64
	PositionKey         string
	Status              *int8
	Keyword             string
}

func (r *TemplateDisplayConfigRepo) PageList(ctx context.Context, page, pageSize int, filter *TemplateDisplayConfigListFilter) ([]model.VideoTemplatePlacementConfig, int64, error) {
	q := qFrom(ctx)
	config := q.VideoTemplatePlacementConfig
	template := q.VideoTemplate
	placement := q.VideoTemplatePlacement
	dao := config.WithContext(ctx).
		LeftJoin(template, template.ID.EqCol(config.TemplateID)).
		LeftJoin(placement, placement.PlacementKey.EqCol(config.PlacementKey))
	if filter != nil {
		if filter.TemplateID != 0 {
			dao = dao.Where(config.TemplateID.Eq(filter.TemplateID))
		}
		if filter.VideoTemplateTypeID != 0 {
			dao = dao.Where(template.VideoTemplateTypeID.Eq(filter.VideoTemplateTypeID))
		}
		if filter.PositionKey != "" {
			dao = dao.Where(config.PlacementKey.Eq(filter.PositionKey))
		}
		if filter.Status != nil {
			dao = dao.Where(config.Status.Eq(uint8(*filter.Status)))
		}
		if filter.Keyword != "" {
			keyword := "%" + filter.Keyword + "%"
			dao = dao.Where(field.Or(
				config.Description.Like(keyword), template.Name.Like(keyword),
				placement.PlacementName.Like(keyword), placement.PlacementKey.Like(keyword),
			))
		}
	}
	total, err := dao.Count()
	if err != nil {
		return nil, 0, err
	}
	rows, err := dao.Preload(config.Template, config.Template.VideoTemplateType, config.Placement).
		Order(config.Sort.Desc(), config.ID.Desc()).Offset((page - 1) * pageSize).Limit(pageSize).Find()
	return valuesOf(rows), total, err
}

func (r *TemplateDisplayConfigRepo) GetDetail(ctx context.Context, id uint64) (*model.VideoTemplatePlacementConfig, error) {
	q := qFrom(ctx).VideoTemplatePlacementConfig
	return q.WithContext(ctx).Preload(q.Template, q.Template.VideoTemplateType, q.Placement).
		Where(q.ID.Eq(id)).First()
}

func (r *TemplateDisplayConfigRepo) PairExists(ctx context.Context, templateID uint64, positionKey string, excludeID uint64) (bool, error) {
	q := qFrom(ctx).VideoTemplatePlacementConfig
	dao := q.WithContext(ctx).Where(q.TemplateID.Eq(templateID), q.PlacementKey.Eq(positionKey))
	if excludeID != 0 {
		dao = dao.Where(q.ID.Neq(excludeID))
	}
	count, err := dao.Count()
	return count > 0, err
}

func (r *TemplateDisplayConfigRepo) UpdateFields(ctx context.Context, item *model.VideoTemplatePlacementConfig) error {
	q := qFrom(ctx).VideoTemplatePlacementConfig
	_, err := q.WithContext(ctx).Where(q.ID.Eq(item.ID)).Select(
		q.TemplateID, q.PlacementKey, q.Sort, q.Status, q.Description,
	).Updates(item)
	return err
}

type ClientTemplateDisplayTargets struct {
	PositionKey string
	CountryCode string
	AppCode     string
	PackageCode string
	VersionCode string
}

func (r *TemplateDisplayConfigRepo) ListForClient(ctx context.Context, targets ClientTemplateDisplayTargets) ([]model.VideoTemplatePlacementConfig, error) {
	// 分类的国家、展示位置和 APP/包/版本规则由 TemplateTypeRepo 统一解析。
	types, err := NewTemplateTypeRepo().ListForClient(ctx, ClientTemplateTypeTargets{
		PositionKey: targets.PositionKey, CountryCode: targets.CountryCode,
		AppCode: targets.AppCode, PackageCode: targets.PackageCode, VersionCode: targets.VersionCode,
	})
	if err != nil {
		return nil, err
	}
	if len(types) == 0 {
		return []model.VideoTemplatePlacementConfig{}, nil
	}
	typeIDs := make([]uint64, 0, len(types))
	for i := range types {
		typeIDs = append(typeIDs, types[i].ID)
	}

	q := qFrom(ctx)
	config := q.VideoTemplatePlacementConfig
	template := q.VideoTemplate
	placement := q.VideoTemplatePlacement
	rows, err := config.WithContext(ctx).
		Join(template, template.ID.EqCol(config.TemplateID)).
		Join(placement, placement.PlacementKey.EqCol(config.PlacementKey)).
		Where(
			config.PlacementKey.Eq(targets.PositionKey), config.Status.Eq(1),
			template.Status.Eq(1), template.VideoTemplateTypeID.In(typeIDs...), placement.Status.Eq(1),
		).
		Preload(config.Template, config.Template.VideoTemplateType, config.Placement).
		Order(config.Sort.Desc(), template.Sort.Desc(), template.UsageCount.Desc(),
			template.LikeCount.Desc(), template.ViewCount.Desc(), template.ID.Desc()).Find()
	return valuesOf(rows), err
}

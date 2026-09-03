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
	ListSort       ListSort
	TemplateID     uint64
	TemplateTypeID uint64
	PositionKey    string
	Status         *int8
	Keyword        string
}

type TemplateDisplayConfigRecord struct {
	model.VideoTemplatePlacementConfig
	PositionKey     string                      `json:"position_key"`
	Remark          string                      `json:"remark"`
	Template        *TemplateRecord             `json:"template,omitempty"`
	DisplayPosition *model.VideoDisplayPosition `json:"display_position,omitempty"`
}

func (r *TemplateDisplayConfigRepo) PageList(ctx context.Context, page, pageSize int, filter *TemplateDisplayConfigListFilter) ([]TemplateDisplayConfigRecord, int64, error) {
	q := qFrom(ctx)
	config := q.VideoTemplatePlacementConfig
	template := q.VideoTemplate
	placement := q.VideoDisplayPosition
	dao := config.WithContext(ctx).
		LeftJoin(template, template.ID.EqCol(config.TemplateID)).
		LeftJoin(placement, placement.PositionKey.EqCol(config.PlacementKey))
	if filter != nil {
		if filter.TemplateID != 0 {
			dao = dao.Where(config.TemplateID.Eq(filter.TemplateID))
		}
		if filter.TemplateTypeID != 0 {
			dao = dao.Where(template.TemplateTypeID.Eq(filter.TemplateTypeID))
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
				placement.PositionName.Like(keyword), placement.PositionKey.Like(keyword),
			))
		}
	}
	total, err := dao.Count()
	if err != nil {
		return nil, 0, err
	}
	listSort := ListSort{}
	if filter != nil {
		listSort = filter.ListSort
	}
	order := orderForList(listSort, map[string]field.OrderExpr{"id": config.ID, "sort": config.Sort}, config.ID, config.Sort.Desc(), config.ID.Desc())
	rows, err := dao.Order(order...).Offset((page - 1) * pageSize).Limit(pageSize).Find()
	if err != nil {
		return nil, 0, err
	}
	records, err := r.loadRecords(ctx, valuesOf(rows))
	return records, total, err
}

func (r *TemplateDisplayConfigRepo) GetDetail(ctx context.Context, id uint64) (*TemplateDisplayConfigRecord, error) {
	q := qFrom(ctx).VideoTemplatePlacementConfig
	item, err := q.WithContext(ctx).Where(q.ID.Eq(id)).First()
	if err != nil {
		return nil, err
	}
	records, err := r.loadRecords(ctx, []model.VideoTemplatePlacementConfig{*item})
	if err != nil {
		return nil, err
	}
	return &records[0], nil
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
}

func (r *TemplateDisplayConfigRepo) ListForClient(ctx context.Context, targets ClientTemplateDisplayTargets) ([]*model.VideoTemplatePlacementConfig, error) {
	q := qFrom(ctx)
	config := q.VideoTemplatePlacementConfig
	template := q.VideoTemplate
	placement := q.VideoDisplayPosition
	rows, err := config.WithContext(ctx).Preload(config.Template, template.AIModel).
		Join(template, template.ID.EqCol(config.TemplateID)).
		Join(placement, placement.PositionKey.EqCol(config.PlacementKey)).
		Where(
			config.PlacementKey.Eq(targets.PositionKey), config.Status.Eq(1),
			template.Status.Eq(1), placement.Status.Eq(1),
		).
		Order(config.Sort.Desc(), template.Sort.Desc(), template.UsageCount.Desc(),
			template.LikeCount.Desc(), template.ViewCount.Desc(), template.ID.Desc()).Find()
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *TemplateDisplayConfigRepo) loadRecords(ctx context.Context, items []model.VideoTemplatePlacementConfig) ([]TemplateDisplayConfigRecord, error) {
	result := make([]TemplateDisplayConfigRecord, 0, len(items))
	if len(items) == 0 {
		return result, nil
	}
	templateIDs := make([]uint64, 0, len(items))
	positionKeys := make([]string, 0, len(items))
	for i := range items {
		templateIDs = append(templateIDs, items[i].TemplateID)
		positionKeys = append(positionKeys, items[i].PlacementKey)
	}
	q := qFrom(ctx)
	templateRows, err := q.VideoTemplate.WithContext(ctx).Preload(q.VideoTemplate.AIModel).
		Where(q.VideoTemplate.ID.In(templateIDs...)).Find()
	if err != nil {
		return nil, err
	}
	templates, err := NewTemplateRepo().loadRecords(ctx, templateValues(templateRows))
	if err != nil {
		return nil, err
	}
	templateByID := make(map[uint64]*TemplateRecord, len(templates))
	for i := range templates {
		templateByID[templates[i].ID] = &templates[i]
	}
	positionRows, err := q.VideoDisplayPosition.WithContext(ctx).
		Where(q.VideoDisplayPosition.PositionKey.In(positionKeys...)).Find()
	if err != nil {
		return nil, err
	}
	positionByKey := make(map[string]*model.VideoDisplayPosition, len(positionRows))
	for _, row := range positionRows {
		if row != nil {
			positionByKey[row.PositionKey] = row
		}
	}
	for i := range items {
		result = append(result, TemplateDisplayConfigRecord{
			VideoTemplatePlacementConfig: items[i],
			PositionKey:                  items[i].PlacementKey,
			Remark:                       items[i].Description,
			Template:                     templateByID[items[i].TemplateID],
			DisplayPosition:              positionByKey[items[i].PlacementKey],
		})
	}
	return result, nil
}

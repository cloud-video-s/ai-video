package repository

import (
	"context"
	"time"

	"ai-video/internal/domain"
	"ai-video/internal/gen/model"

	"gorm.io/datatypes"
	"gorm.io/gen/field"
	"gorm.io/gorm"
)

const videoToolConfigTable = "video_tool_config"

type toolConfigRecord struct {
	ID              uint64
	Name            string
	Icon            string
	BackgroundImage string
	ToolType        uint8
	ToolsType       string
	ModelID         int64
	ConfigType      uint8
	ConfigData      datatypes.JSON
	BadgeImage      string
	Sort            int64
	Prompt          string
	Status          int8
	CreatedAt       time.Time
	UpdatedAt       time.Time
	DeletedAt       gorm.DeletedAt
}

func (*toolConfigRecord) TableName() string { return videoToolConfigTable }

type ToolConfigRepo struct{}

func NewToolConfigRepo() *ToolConfigRepo { return &ToolConfigRepo{} }

type ToolConfigListFilter struct {
	ListSort ListSort
	Status   *int8
	Keyword  string
}

func (r *ToolConfigRepo) PageList(ctx context.Context, page, pageSize int, filter *ToolConfigListFilter) ([]domain.ToolConfig, int64, error) {
	q := qFrom(ctx).VideoToolConfig
	dao := q.WithContext(ctx)
	if filter != nil {
		if filter.Status != nil {
			dao = dao.Where(q.Status.Eq(*filter.Status))
		}
		if filter.Keyword != "" {
			dao = dao.Where(q.Name.Like("%" + filter.Keyword + "%"))
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
	order := orderForList(listSort, map[string]field.OrderExpr{
		"id": q.ID, "sort": q.Sort, "status": q.Status, "updated_at": q.UpdatedAt,
	}, q.ID, q.Sort.Asc(), q.ID.Desc())
	rows, err := dao.Order(order...).Offset((page - 1) * pageSize).Limit(pageSize).Find()
	if err != nil {
		return nil, 0, err
	}
	items := toolConfigValuesFromModels(rows)
	if err := r.attachModelNames(ctx, items); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *ToolConfigRepo) ListOptions(ctx context.Context) ([]*model.VideoToolConfig, error) {
	items, err := r.ListForClient(ctx)
	if err != nil {
		return nil, err
	}

	//if err := r.attachModelNames(ctx, items); err != nil {
	//	return nil, err
	//}
	return items, nil
}

// ListForClient returns every online, non-deleted tool in display order.
func (r *ToolConfigRepo) ListForClient(ctx context.Context) ([]*model.VideoToolConfig, error) {
	q := qFrom(ctx).VideoToolConfig
	rows, err := q.WithContext(ctx).Preload(q.Model).Where(q.Status.Eq(1)).Find()
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// GetEnabledByID returns an online, non-deleted tool for task generation.
// Model and platform availability are validated separately by ModelRepo so a
// disabled dependency cannot be invoked through a stale tool configuration.
func (r *ToolConfigRepo) GetEnabledByID(ctx context.Context, id uint64) (*model.VideoToolConfig, error) {
	q := qFrom(ctx).VideoToolConfig
	return q.WithContext(ctx).Where(q.ID.Eq(id), q.Status.Eq(1)).First()
}

func (r *ToolConfigRepo) GetByID(ctx context.Context, id uint64) (*domain.ToolConfig, error) {
	var row toolConfigRecord
	if err := qFrom(ctx).UnderlyingDB().First(&row, id).Error; err != nil {
		return nil, err
	}
	items := []domain.ToolConfig{*toolConfigValue(&row)}
	if err := r.attachModelNames(ctx, items); err != nil {
		return nil, err
	}
	return &items[0], nil
}

func (r *ToolConfigRepo) GetByName(ctx context.Context, name string) (*domain.ToolConfig, error) {
	var row toolConfigRecord
	if err := qFrom(ctx).UnderlyingDB().Where("name = ?", name).First(&row).Error; err != nil {
		return nil, err
	}
	return toolConfigValue(&row), nil
}

func (r *ToolConfigRepo) Create(ctx context.Context, item *domain.ToolConfig) error {
	row := toolConfigRecordFromDomain(item)
	if err := qFrom(ctx).UnderlyingDB().Create(&row).Error; err != nil {
		return err
	}
	*item = *toolConfigValue(&row)
	return nil
}

func (r *ToolConfigRepo) UpdateFields(ctx context.Context, item *domain.ToolConfig) error {
	row := toolConfigRecordFromDomain(item)
	result := qFrom(ctx).UnderlyingDB().Model(&toolConfigRecord{}).Where("id = ?", item.ID).
		Select(
			"name", "icon", "background_image", "tool_type", "tools_type", "model_id", "config_type",
			"config_data", "badge_image", "sort", "prompt", "status",
		).Updates(&row)
	return result.Error
}

func (r *ToolConfigRepo) UpdateStatus(ctx context.Context, id uint64, status int8) error {
	return qFrom(ctx).UnderlyingDB().Model(&toolConfigRecord{}).Where("id = ?", id).Update("status", status).Error
}

func (r *ToolConfigRepo) Delete(ctx context.Context, id uint64) error {
	return qFrom(ctx).UnderlyingDB().Delete(&toolConfigRecord{}, id).Error
}

func (r *ToolConfigRepo) ModelAvailable(ctx context.Context, id int64, modelType uint8) (bool, error) {
	var count int64
	err := qFrom(ctx).UnderlyingDB().Table("video_model").
		Where("id = ? AND model_type = ? AND status = ? AND deleted_at IS NULL", id, modelType, 1).
		Count(&count).Error
	return count > 0, err
}

func (r *ToolConfigRepo) ListAvailableModels(ctx context.Context, modelType uint8) ([]domain.ToolModelOption, error) {
	db := qFrom(ctx).UnderlyingDB().Table("video_model").
		Select("id", "name", "model_type").
		Where("status = ? AND deleted_at IS NULL", 1)
	if modelType != 0 {
		db = db.Where("model_type = ?", modelType)
	}
	var rows []domain.ToolModelOption
	if err := db.Order("name ASC, id ASC").Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *ToolConfigRepo) attachModelNames(ctx context.Context, items []domain.ToolConfig) error {
	if len(items) == 0 {
		return nil
	}
	ids := make([]int64, 0, len(items))
	seen := make(map[int64]struct{}, len(items))
	for i := range items {
		if items[i].ModelID == 0 {
			continue
		}
		if _, ok := seen[items[i].ModelID]; ok {
			continue
		}
		seen[items[i].ModelID] = struct{}{}
		ids = append(ids, items[i].ModelID)
	}
	if len(ids) == 0 {
		return nil
	}
	q := qFrom(ctx).VideoModel
	rows, err := q.WithContext(ctx).Select(q.ID, q.Name).Where(q.ID.In(ids...)).Find()
	if err != nil {
		return err
	}
	names := make(map[int64]string, len(rows))
	for _, row := range rows {
		names[row.ID] = row.Name
	}
	for i := range items {
		items[i].ModelName = names[items[i].ModelID]
	}
	return nil
}

func toolConfigRecordFromDomain(item *domain.ToolConfig) toolConfigRecord {
	if item == nil {
		return toolConfigRecord{}
	}
	return toolConfigRecord{
		ID: item.ID, Name: item.Name, Icon: item.Icon, BackgroundImage: item.BackgroundImage,
		ToolType: item.ToolType, ToolsType: item.ToolsType, ModelID: item.ModelID, ConfigType: item.ConfigType,
		ConfigData: datatypes.JSON(item.ConfigData), BadgeImage: item.BadgeImage,
		Sort: item.Sort, Prompt: item.Prompt, Status: item.Status,
		CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt,
	}
}

func toolConfigValue(row *toolConfigRecord) *domain.ToolConfig {
	if row == nil {
		return nil
	}
	return &domain.ToolConfig{
		ID: row.ID, Name: row.Name, Icon: row.Icon, BackgroundImage: row.BackgroundImage,
		ToolType: row.ToolType, ToolsType: row.ToolsType, ModelID: row.ModelID, ConfigType: row.ConfigType,
		ConfigData: append([]byte(nil), row.ConfigData...), BadgeImage: row.BadgeImage,
		Sort: row.Sort, Prompt: row.Prompt, Status: row.Status,
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
}

func toolConfigValuesFromModels(rows []*model.VideoToolConfig) []domain.ToolConfig {
	result := make([]domain.ToolConfig, 0, len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		result = append(result, domain.ToolConfig{
			ID: row.ID, Name: row.Name, Icon: row.Icon, BackgroundImage: row.BackgroundImage,
			ToolType: row.ToolType, ToolsType: row.ToolsType, ModelID: row.ModelID,
			ConfigType: row.ConfigType, ConfigData: append([]byte(nil), row.ConfigData...),
			BadgeImage: row.BadgeImage, Sort: row.Sort, Prompt: row.Prompt, Status: row.Status,
			CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
		})
	}
	return result
}

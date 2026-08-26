package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"ai-video/internal/domain"
	"ai-video/internal/gen/model"
	"ai-video/internal/repository"

	"gorm.io/gorm"
)

type ToolConfigService struct {
	repo *repository.ToolConfigRepo
}

func NewToolConfigService() *ToolConfigService {
	return &ToolConfigService{repo: repository.NewToolConfigRepo()}
}

type ListToolConfigRequest struct {
	ListSortRequest
	Status  *int8  `form:"status" binding:"omitempty,oneof=0 1"`
	Keyword string `form:"keyword"`
}

type ListToolModelOptionRequest struct {
	ToolType uint8 `form:"tool_type" binding:"required,oneof=1 2"`
}

type ToolConfigPayload struct {
	Name            string          `json:"name" binding:"required,max=128"`
	Icon            string          `json:"icon" binding:"required,max=1024"`
	BackgroundImage string          `json:"background_image" binding:"required,max=1024"`
	ToolType        uint8           `json:"tool_type" binding:"required,oneof=1 2"`
	ToolsType       string          `json:"tools_type" binding:"required"`
	ModelID         int64           `json:"model_id" binding:"required,gt=0"`
	ConfigType      uint8           `json:"config_type" binding:"required,oneof=1 2 3 4"`
	ConfigData      json.RawMessage `json:"config_data" binding:"required"`
	BadgeImage      string          `json:"badge_image" binding:"max=1024"`
	Sort            int64           `json:"sort" binding:"gte=0,lte=999999"`
	Prompt          string          `json:"prompt" binding:"max=10000"`
	Status          int8            `json:"status" binding:"oneof=0 1"`
}

type ToolReferenceImageOption struct {
	Name  string `json:"name"`
	Image string `json:"image"`
	Sort  int64  `json:"sort"`
}

type ToolRatioOption struct {
	Name  string `json:"name"`
	Value string `json:"value"`
	Sort  int64  `json:"sort"`
}

type ToolAgeRange struct {
	Min *int `json:"min"`
	Max *int `json:"max"`
}

type toolReferenceImageConfig struct {
	ReferenceImages []ToolReferenceImageOption `json:"reference_images"`
}

type toolRatioConfig struct {
	RatioOptions []ToolRatioOption `json:"ratio_options"`
}

type toolAgeConfig struct {
	AgeRange ToolAgeRange `json:"age_range"`
}

type ToolConfigStatusPayload struct {
	Status int8 `json:"status" binding:"oneof=0 1"`
}

var validToolsTypes = map[string]struct{}{
	"enhance":       {},
	"outpaint":      {},
	"hairstyle":     {},
	"age_transform": {},
	"body_reshape":  {},
	"colorful":      {},
	"makeup":        {},
	"outfit":        {},
	"pose_transfer": {},
}

func (s *ToolConfigService) List(ctx context.Context, page, pageSize int, req *ListToolConfigRequest) ([]domain.ToolConfig, int64, error) {
	return s.repo.PageList(ctx, page, pageSize, &repository.ToolConfigListFilter{
		ListSort: req.listSort(), Status: req.Status, Keyword: strings.TrimSpace(req.Keyword),
	})
}

func (s *ToolConfigService) ListOptions(ctx context.Context) ([]domain.ToolConfig, error) {
	options, err := s.repo.ListOptions(ctx)
	if err != nil {
		return []domain.ToolConfig{}, err
	}
	return applyToolConfig(options), nil
}

func (s *ToolConfigService) ListModelOptions(ctx context.Context, toolType uint8) ([]domain.ToolModelOption, error) {
	if toolType < domain.ToolTypeImageGeneration || toolType > domain.ToolTypeVideoGeneration {
		return nil, errors.New("工具类型无效")
	}
	return s.repo.ListAvailableModels(ctx, toolType)
}

func (s *ToolConfigService) GetByID(ctx context.Context, id uint64) (*domain.ToolConfig, error) {
	item, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, notFoundOr(err, "工具配置不存在")
	}
	return item, nil
}

func (s *ToolConfigService) Create(ctx context.Context, req *ToolConfigPayload) (*domain.ToolConfig, error) {
	if err := s.prepareAndValidate(ctx, req, 0); err != nil {
		return nil, err
	}
	item := &domain.ToolConfig{}
	applyToolConfigPayload(item, req)
	if err := s.repo.Create(ctx, item); err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, item.ID)
}

func (s *ToolConfigService) Update(ctx context.Context, id uint64, req *ToolConfigPayload) (*domain.ToolConfig, error) {
	item, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, notFoundOr(err, "工具配置不存在")
	}
	if err := s.prepareAndValidate(ctx, req, id); err != nil {
		return nil, err
	}
	applyToolConfigPayload(item, req)
	if err := s.repo.UpdateFields(ctx, item); err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, id)
}

func (s *ToolConfigService) UpdateStatus(ctx context.Context, id uint64, status int8) (*domain.ToolConfig, error) {
	if status != 0 && status != 1 {
		return nil, errors.New("工具状态只能为在线或下线")
	}
	if _, err := s.repo.GetByID(ctx, id); err != nil {
		return nil, notFoundOr(err, "工具配置不存在")
	}
	if err := s.repo.UpdateStatus(ctx, id, status); err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, id)
}

func (s *ToolConfigService) Delete(ctx context.Context, id uint64) error {
	if _, err := s.repo.GetByID(ctx, id); err != nil {
		return notFoundOr(err, "工具配置不存在")
	}
	return s.repo.Delete(ctx, id)
}

func (s *ToolConfigService) prepareAndValidate(ctx context.Context, req *ToolConfigPayload, currentID uint64) error {
	req.Name = strings.TrimSpace(req.Name)
	req.Icon = strings.TrimSpace(req.Icon)
	req.BackgroundImage = strings.TrimSpace(req.BackgroundImage)
	req.ToolsType = strings.TrimSpace(req.ToolsType)
	req.BadgeImage = strings.TrimSpace(req.BadgeImage)
	req.Prompt = strings.TrimSpace(req.Prompt)
	if req.Name == "" {
		return errors.New("工具名称不能为空")
	}
	if req.Icon == "" {
		return errors.New("工具图标不能为空")
	}
	if req.BackgroundImage == "" {
		return errors.New("工具背景图不能为空")
	}
	if req.ToolType < domain.ToolTypeImageGeneration || req.ToolType > domain.ToolTypeVideoGeneration {
		return errors.New("工具类型无效")
	}
	if _, ok := validToolsTypes[req.ToolsType]; !ok {
		return errors.New("所属功能无效")
	}
	if req.ConfigType < domain.ToolConfigTypeNone || req.ConfigType > domain.ToolConfigTypeRatio {
		return errors.New("工具配置类型无效")
	}
	modelExists, err := s.repo.ModelAvailable(ctx, req.ModelID, req.ToolType)
	if err != nil {
		return err
	}
	if !modelExists {
		return errors.New("关联模型不存在、未启用或与工具类型不匹配")
	}
	if err := normalizeToolConfigData(req); err != nil {
		return err
	}
	if req.Sort < 0 || req.Sort > 999999 {
		return errors.New("排序值必须在 0 到 999999 之间")
	}
	if req.Status != 0 && req.Status != 1 {
		return errors.New("工具状态只能为在线或下线")
	}
	existing, err := s.repo.GetByName(ctx, req.Name)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if existing.ID != currentID {
		return errors.New("工具名称已存在")
	}
	return nil
}

func applyToolConfigPayload(item *domain.ToolConfig, req *ToolConfigPayload) {
	item.Name = req.Name
	item.Icon = req.Icon
	item.BackgroundImage = req.BackgroundImage
	item.ToolType = req.ToolType
	item.ToolsType = strings.TrimSpace(req.ToolsType)
	item.ModelID = req.ModelID
	item.ConfigType = req.ConfigType
	item.ConfigData = append([]byte(nil), req.ConfigData...)
	item.BadgeImage = req.BadgeImage
	item.Sort = req.Sort
	item.Prompt = req.Prompt
	item.Status = req.Status
}

func normalizeToolConfigData(req *ToolConfigPayload) error {
	switch req.ConfigType {
	case domain.ToolConfigTypeNone:
		req.ConfigData = json.RawMessage(`{}`)
		return nil
	case domain.ToolConfigTypeReferenceImage:
		var config toolReferenceImageConfig
		if err := json.Unmarshal(req.ConfigData, &config); err != nil {
			return errors.New("参考图配置格式错误")
		}
		if len(config.ReferenceImages) == 0 || len(config.ReferenceImages) > 20 {
			return errors.New("参考图配置数量必须在 1 到 20 之间")
		}
		for i := range config.ReferenceImages {
			option := &config.ReferenceImages[i]
			option.Name = strings.TrimSpace(option.Name)
			option.Image = strings.TrimSpace(option.Image)
			if option.Name == "" || option.Image == "" {
				return errors.New("参考图名称和图片不能为空")
			}
			if option.Sort < 0 || option.Sort > 999999 {
				return errors.New("参考图排序值必须在 0 到 999999 之间")
			}
		}
		normalized, err := json.Marshal(config)
		if err != nil {
			return err
		}
		req.ConfigData = normalized
		return nil
	case domain.ToolConfigTypeRatio:
		var config toolRatioConfig
		if err := json.Unmarshal(req.ConfigData, &config); err != nil {
			return errors.New("比例配置格式错误")
		}
		if len(config.RatioOptions) == 0 || len(config.RatioOptions) > 20 {
			return errors.New("比例配置数量必须在 1 到 20 之间")
		}
		for i := range config.RatioOptions {
			option := &config.RatioOptions[i]
			option.Name = strings.TrimSpace(option.Name)
			option.Value = strings.TrimSpace(option.Value)
			if option.Name == "" || option.Value == "" {
				return errors.New("比例名称和参数不能为空")
			}
			if option.Sort < 0 || option.Sort > 999999 {
				return errors.New("比例排序值必须在 0 到 999999 之间")
			}
		}
		normalized, err := json.Marshal(config)
		if err != nil {
			return err
		}
		req.ConfigData = normalized
		return nil
	case domain.ToolConfigTypeAge:
		var config toolAgeConfig
		if err := json.Unmarshal(req.ConfigData, &config); err != nil {
			return errors.New("年龄配置格式错误")
		}
		if config.AgeRange.Min == nil || config.AgeRange.Max == nil {
			return errors.New("年龄最小值和最大值不能为空")
		}
		if *config.AgeRange.Min < 0 || *config.AgeRange.Max > 200 || *config.AgeRange.Min > *config.AgeRange.Max {
			return errors.New("年龄范围必须满足 0 ≤ 最小值 ≤ 最大值 ≤ 200")
		}
		normalized, err := json.Marshal(config)
		if err != nil {
			return err
		}
		req.ConfigData = normalized
		return nil
	default:
		return errors.New("工具配置类型无效")
	}
}

func applyToolConfig(rows []*model.VideoToolConfig) (data []domain.ToolConfig) {
	for _, item := range rows {
		data = append(data, domain.ToolConfig{
			Name:            item.Name,
			Icon:            item.Icon,
			BackgroundImage: item.BackgroundImage,
			ToolType:        item.ToolType,
			ToolsType:       item.ToolsType,
			ModelID:         item.ModelID,
			ConfigType:      item.ConfigType,
			ConfigData:      json.RawMessage(item.ConfigData),
			BadgeImage:      item.BadgeImage,
			Sort:            item.Sort,
			Status:          item.Status,
			Prompt:          item.Prompt,
		})
	}
	return data
}

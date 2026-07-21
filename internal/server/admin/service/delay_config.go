package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"ai-video/internal/app"
	"ai-video/internal/gen/model"
	"ai-video/internal/repository"

	"gorm.io/gorm"
)

type DelayConfigService struct {
	repo *repository.DelayConfigRepo
}

func NewDelayConfigService() *DelayConfigService {
	return &DelayConfigService{repo: repository.NewDelayConfigRepo()}
}

type ListDelayConfigRequest struct {
	Group   string
	Keyword string
}

type CreateDelayConfigRequest struct {
	Group   string `json:"group" binding:"required,max=64"`
	Key     string `json:"key" binding:"required,max=128"`
	Value   string `json:"value" binding:"required,max=64"`
	Type    string `json:"type" binding:"required,oneof=string int bool"`
	Options string `json:"options" binding:"max=255"`
	Remark  string `json:"remark" binding:"max=255"`
	Sort    int    `json:"sort"`
}

type UpdateDelayConfigRequest struct {
	Group   string `json:"group" binding:"required,max=64"`
	Value   string `json:"value" binding:"required,max=64"`
	Type    string `json:"type" binding:"required,oneof=string int bool"`
	Options string `json:"options" binding:"max=255"`
	Remark  string `json:"remark" binding:"max=255"`
	Sort    int64  `json:"sort"`
}

type DelayConfigValueItem struct {
	Key   string `json:"key" binding:"required"`
	Value string `json:"value"`
}

func (s *DelayConfigService) List(ctx context.Context, page, pageSize int, req *ListDelayConfigRequest) ([]model.VideoDelayConfig, int64, error) {
	return s.repo.PageList(ctx, page, pageSize, &repository.DelayConfigListFilter{
		Group: strings.TrimSpace(req.Group), Keyword: strings.TrimSpace(req.Keyword),
	})
}

func (s *DelayConfigService) ListGroups(ctx context.Context) ([]string, error) {
	return s.repo.ListGroups(ctx)
}

func (s *DelayConfigService) GetByID(ctx context.Context, id uint) (*model.VideoDelayConfig, error) {
	config, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, notFoundOr(err, "延迟配置不存在")
	}
	return config, nil
}

func (s *DelayConfigService) Create(ctx context.Context, req *CreateDelayConfigRequest) error {
	key := strings.TrimSpace(req.Key)
	exists, err := s.repo.ExistsByKey(ctx, key)
	if err != nil {
		return err
	}
	if exists {
		return errors.New("配置键已存在")
	}
	if err := validateDelayConfigValue(req.Type, req.Value, req.Options); err != nil {
		return err
	}
	config := &model.VideoDelayConfig{
		Group: strings.TrimSpace(req.Group), Key: key, Value: strings.TrimSpace(req.Value),
		Type: req.Type, Options: strings.TrimSpace(req.Options), Remark: strings.TrimSpace(req.Remark), Sort: int64(req.Sort),
	}
	if err := s.repo.Create(ctx, config); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return errors.New("配置键已存在")
		}
		return err
	}
	return nil
}

func (s *DelayConfigService) Update(ctx context.Context, id uint, req *UpdateDelayConfigRequest) error {
	config, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return notFoundOr(err, "延迟配置不存在")
	}
	if err := validateDelayConfigValue(req.Type, req.Value, req.Options); err != nil {
		return err
	}
	config.Group = strings.TrimSpace(req.Group)
	config.Value = strings.TrimSpace(req.Value)
	config.Type = req.Type
	config.Options = strings.TrimSpace(req.Options)
	config.Remark = strings.TrimSpace(req.Remark)
	config.Sort = req.Sort
	return s.repo.Update(ctx, config)
}

func (s *DelayConfigService) Delete(ctx context.Context, id uint) error {
	if _, err := s.repo.GetByID(ctx, id); err != nil {
		return notFoundOr(err, "延迟配置不存在")
	}
	return s.repo.HardDelete(ctx, id)
}

func (s *DelayConfigService) BatchUpdateValues(ctx context.Context, items []DelayConfigValueItem) error {
	return repository.Transaction(ctx, func(ctx context.Context) error {
		for _, item := range items {
			key := strings.TrimSpace(item.Key)
			if key == "" {
				return errors.New("配置键不能为空")
			}
			config, err := s.repo.GetByKey(ctx, key)
			if err != nil {
				return notFoundOr(err, "延迟配置不存在: "+key)
			}
			value := strings.TrimSpace(item.Value)
			if err := validateDelayConfigValue(config.Type, value, config.Options); err != nil {
				return fmt.Errorf("配置 %s: %w", key, err)
			}
			if err := s.repo.UpdateValue(ctx, key, value); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *DelayConfigService) SyncFromFile() error {
	return app.SeedOBDelayConfig(app.DefaultOBDelayConfigPath)
}

func validateDelayConfigValue(typ, value, options string) error {
	typ = strings.TrimSpace(typ)
	value = strings.TrimSpace(value)
	if value == "" {
		return errors.New("配置值不能为空")
	}
	switch typ {
	case "string":
	case "int":
		if _, err := strconv.Atoi(value); err != nil {
			return errors.New("int 类型配置值必须是整数")
		}
	case "bool":
		if value != "0" && value != "1" && value != "true" && value != "false" {
			return errors.New("bool 类型配置值必须是 0、1、true 或 false")
		}
	default:
		return fmt.Errorf("不支持的配置类型: %s", typ)
	}
	if strings.TrimSpace(options) == "" {
		return nil
	}
	var allowed []string
	if err := json.Unmarshal([]byte(options), &allowed); err != nil {
		return errors.New("options 必须是字符串 JSON 数组")
	}
	for _, option := range allowed {
		if value == option {
			return nil
		}
	}
	return errors.New("配置值不在 options 允许范围内")
}

package service

import (
	"context"
	"errors"
	"strings"

	"ai-video/internal/gen/model"
	"ai-video/internal/repository"

	"gorm.io/gorm"
)

type TemplateDisplayConfigService struct {
	repo         *repository.TemplateDisplayConfigRepo
	templateRepo *repository.TemplateRepo
	positionRepo *repository.DisplayPositionRepo
}

func NewTemplateDisplayConfigService() *TemplateDisplayConfigService {
	return &TemplateDisplayConfigService{
		repo: repository.NewTemplateDisplayConfigRepo(), templateRepo: repository.NewTemplateRepo(),
		positionRepo: repository.NewDisplayPositionRepo(),
	}
}

type ListTemplateDisplayConfigRequest struct {
	TemplateID          uint64 `form:"template_id"`
	VideoTemplateTypeID uint64 `form:"video_template_type_id"`
	PositionKey         string `form:"position_key" binding:"omitempty,max=64"`
	Status              *int8  `form:"status" binding:"omitempty,oneof=0 1"`
	Keyword             string `form:"keyword" binding:"omitempty,max=128"`
}

type TemplateDisplayConfigPayload struct {
	TemplateID   uint64 `json:"template_id" binding:"required"`
	PlacementKey string `json:"placement_key" binding:"required,max=64"`
	Sort         int    `json:"sort"`
	Status       int8   `json:"status" binding:"oneof=0 1"`
	Description  string `json:"description" binding:"max=500"`
}

func (s *TemplateDisplayConfigService) List(ctx context.Context, page, pageSize int, req *ListTemplateDisplayConfigRequest) ([]repository.TemplateDisplayConfigRecord, int64, error) {
	return s.repo.PageList(ctx, page, pageSize, &repository.TemplateDisplayConfigListFilter{
		TemplateID: req.TemplateID, VideoTemplateTypeID: req.VideoTemplateTypeID,
		PositionKey: strings.TrimSpace(req.PositionKey), Status: req.Status,
		Keyword: strings.TrimSpace(req.Keyword),
	})
}

func (s *TemplateDisplayConfigService) GetByID(ctx context.Context, id uint64) (*repository.TemplateDisplayConfigRecord, error) {
	item, err := s.repo.GetDetail(ctx, id)
	if err != nil {
		return nil, notFoundOr(err, "模板展示配置不存在")
	}
	return item, nil
}

func (s *TemplateDisplayConfigService) Create(ctx context.Context, req *TemplateDisplayConfigPayload) (*repository.TemplateDisplayConfigRecord, error) {
	if err := s.prepare(ctx, req, 0); err != nil {
		return nil, err
	}
	item := &model.VideoTemplatePlacementConfig{}
	applyTemplateDisplayConfigPayload(item, req)
	if err := s.repo.Create(ctx, item); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, errors.New("该模板已配置到此展示位置")
		}
		return nil, err
	}
	return s.repo.GetDetail(ctx, item.ID)
}

func (s *TemplateDisplayConfigService) Update(ctx context.Context, id uint64, req *TemplateDisplayConfigPayload) (*repository.TemplateDisplayConfigRecord, error) {
	item, err := s.repo.GetDetail(ctx, id)
	if err != nil {
		return nil, notFoundOr(err, "模板展示配置不存在")
	}
	if err := s.prepare(ctx, req, id); err != nil {
		return nil, err
	}
	applyTemplateDisplayConfigPayload(&item.VideoTemplatePlacementConfig, req)
	if err := s.repo.UpdateFields(ctx, &item.VideoTemplatePlacementConfig); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, errors.New("该模板已配置到此展示位置")
		}
		return nil, err
	}
	return s.repo.GetDetail(ctx, item.ID)
}

func (s *TemplateDisplayConfigService) Delete(ctx context.Context, id uint64) error {
	if _, err := s.repo.GetDetail(ctx, id); err != nil {
		return notFoundOr(err, "模板展示配置不存在")
	}
	// Configurations are replaceable mapping rows. Hard deletion allows the
	// same template-position pair to be configured again later without the
	// soft-deleted row colliding with the unique index.
	return s.repo.HardDelete(ctx, uint(id))
}

func (s *TemplateDisplayConfigService) prepare(ctx context.Context, req *TemplateDisplayConfigPayload, currentID uint64) error {
	req.PlacementKey = strings.TrimSpace(req.PlacementKey)
	req.Description = strings.TrimSpace(req.Description)
	if _, err := s.templateRepo.GetWithType(ctx, req.TemplateID); err != nil {
		return notFoundOr(err, "模板不存在")
	}
	position, err := s.positionRepo.GetByKey(ctx, req.PlacementKey)
	if err != nil {
		return notFoundOr(err, "展示位置不存在")
	}
	if position.Status != 1 && req.Status == 1 {
		return errors.New("展示位置已禁用，不能启用该配置")
	}
	exists, err := s.repo.PairExists(ctx, req.TemplateID, req.PlacementKey, currentID)
	if err != nil {
		return err
	}
	if exists {
		return errors.New("该模板已配置到此展示位置")
	}
	return nil
}

func applyTemplateDisplayConfigPayload(item *model.VideoTemplatePlacementConfig, req *TemplateDisplayConfigPayload) {
	item.TemplateID = req.TemplateID
	item.PlacementKey = req.PlacementKey
	item.Sort = uint(req.Sort)
	item.Status = uint8(req.Status)
	item.Description = req.Description
}

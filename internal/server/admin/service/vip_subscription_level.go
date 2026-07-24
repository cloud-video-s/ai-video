package service

import (
	"context"
	"errors"
	"strings"

	"ai-video/internal/gen/model"
	"ai-video/internal/repository"
)

type VIPSubscriptionLevelService struct {
	repo *repository.VIPSubscriptionLevelRepo
}

func NewVIPSubscriptionLevelService() *VIPSubscriptionLevelService {
	return &VIPSubscriptionLevelService{repo: repository.NewVIPSubscriptionLevelRepo()}
}

type ListVIPSubscriptionLevelRequest struct {
	Status  *uint32 `form:"status" binding:"omitempty,oneof=0 1"`
	Keyword string  `form:"keyword" binding:"max=255"`
}

type VIPSubscriptionLevelPayload struct {
	Level       string `json:"level" binding:"required,max=255"`
	Description string `json:"description"`
	Status      uint32 `json:"status" binding:"oneof=0 1"`
	Sort        uint64 `json:"sort"`
}

type VIPSubscriptionLevelStatusPayload struct {
	Status *uint32 `json:"status" binding:"required,oneof=0 1"`
}

func (s *VIPSubscriptionLevelService) List(ctx context.Context, page, pageSize int, req *ListVIPSubscriptionLevelRequest) ([]model.VideoVipSubscriptionLevel, int64, error) {
	return s.repo.PageList(ctx, page, pageSize, &repository.VIPSubscriptionLevelListFilter{
		Status: req.Status, Keyword: strings.TrimSpace(req.Keyword),
	})
}

func (s *VIPSubscriptionLevelService) ListOptions(ctx context.Context) ([]model.VideoVipSubscriptionLevel, error) {
	return s.repo.ListOptions(ctx)
}

func (s *VIPSubscriptionLevelService) GetByID(ctx context.Context, id uint64) (*model.VideoVipSubscriptionLevel, error) {
	item, err := s.repo.GetByID(ctx, uint(id))
	if err != nil {
		return nil, notFoundOr(err, "VIP 等级不存在")
	}
	return item, nil
}

func (s *VIPSubscriptionLevelService) Create(ctx context.Context, req *VIPSubscriptionLevelPayload) (*model.VideoVipSubscriptionLevel, error) {
	if err := validateVIPSubscriptionLevelPayload(req); err != nil {
		return nil, err
	}
	item := &model.VideoVipSubscriptionLevel{}
	applyVIPSubscriptionLevelPayload(item, req)
	if err := s.repo.Create(ctx, item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *VIPSubscriptionLevelService) Update(ctx context.Context, id uint64, req *VIPSubscriptionLevelPayload) (*model.VideoVipSubscriptionLevel, error) {
	item, err := s.repo.GetByID(ctx, uint(id))
	if err != nil {
		return nil, notFoundOr(err, "VIP 等级不存在")
	}
	if err := validateVIPSubscriptionLevelPayload(req); err != nil {
		return nil, err
	}
	applyVIPSubscriptionLevelPayload(item, req)
	if err := s.repo.UpdateFields(ctx, item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *VIPSubscriptionLevelService) UpdateStatus(ctx context.Context, id uint64, status uint32) error {
	item, err := s.repo.GetByID(ctx, uint(id))
	if err != nil {
		return notFoundOr(err, "VIP 等级不存在")
	}
	item.Status = status
	return s.repo.Update(ctx, item, "Status")
}

func (s *VIPSubscriptionLevelService) Delete(ctx context.Context, id uint64) error {
	if _, err := s.repo.GetByID(ctx, uint(id)); err != nil {
		return notFoundOr(err, "VIP 等级不存在")
	}
	count, err := s.repo.SubscriptionCount(ctx, id)
	if err != nil {
		return err
	}
	if count > 0 {
		return errors.New("该 VIP 等级仍被订阅套餐使用，无法删除")
	}
	return s.repo.Delete(ctx, uint(id))
}

func validateVIPSubscriptionLevelPayload(req *VIPSubscriptionLevelPayload) error {
	if strings.TrimSpace(req.Level) == "" {
		return errors.New("VIP 等级名称不能为空")
	}
	return nil
}

func applyVIPSubscriptionLevelPayload(item *model.VideoVipSubscriptionLevel, req *VIPSubscriptionLevelPayload) {
	item.Level = strings.TrimSpace(req.Level)
	item.Description = strings.TrimSpace(req.Description)
	item.Status = req.Status
	item.Sort = req.Sort
}

package service

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"ai-video/internal/gen/model"
	"ai-video/internal/repository"

	"gorm.io/gorm"
)

var bannerPlacementKeyPattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

type BannerPlacementService struct {
	repo *repository.BannerPlacementRepo
}

func NewBannerPlacementService() *BannerPlacementService {
	return &BannerPlacementService{repo: repository.NewBannerPlacementRepo()}
}

type ListBannerPlacementRequest struct {
	Status  *int8  `form:"status" binding:"omitempty,oneof=0 1"`
	Keyword string `form:"keyword" binding:"max=255"`
}

type BannerPlacementPayload struct {
	PlacementName string `json:"placement_name" binding:"required,max=128"`
	PlacementKey  string `json:"placement_key" binding:"required,max=64"`
	Description   string `json:"description" binding:"max=500"`
	CoverImage    string `json:"cover_image" binding:"required,max=1024"`
	Sort          int64  `json:"sort" binding:"gte=0"`
	Status        int8   `json:"status" binding:"oneof=0 1"`
}

func (s *BannerPlacementService) List(ctx context.Context, page, pageSize int, req *ListBannerPlacementRequest) ([]model.VideoBannerPlacement, int64, error) {
	return s.repo.PageList(ctx, page, pageSize, &repository.BannerPlacementListFilter{
		Status: req.Status, Keyword: strings.TrimSpace(req.Keyword),
	})
}

func (s *BannerPlacementService) ListOptions(ctx context.Context) ([]model.VideoBannerPlacement, error) {
	return s.repo.ListOptions(ctx)
}

func (s *BannerPlacementService) GetByID(ctx context.Context, id uint64) (*model.VideoBannerPlacement, error) {
	item, err := s.repo.GetByID(ctx, uint(id))
	if err != nil {
		return nil, notFoundOr(err, "Banner 位置不存在")
	}
	return item, nil
}

func (s *BannerPlacementService) Create(ctx context.Context, req *BannerPlacementPayload) (*model.VideoBannerPlacement, error) {
	if err := s.prepareAndValidate(ctx, req, 0); err != nil {
		return nil, err
	}
	item := &model.VideoBannerPlacement{}
	applyBannerPlacementPayload(item, req)
	if err := s.repo.Create(ctx, item); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, errors.New("Banner 位置标识已存在")
		}
		return nil, err
	}
	return item, nil
}

func (s *BannerPlacementService) Update(ctx context.Context, id uint64, req *BannerPlacementPayload) (*model.VideoBannerPlacement, error) {
	item, err := s.repo.GetByID(ctx, uint(id))
	if err != nil {
		return nil, notFoundOr(err, "Banner 位置不存在")
	}
	if err := s.prepareAndValidate(ctx, req, id); err != nil {
		return nil, err
	}
	oldKey := item.PlacementKey
	applyBannerPlacementPayload(item, req)
	if err := repository.Transaction(ctx, func(ctx context.Context) error {
		if err := s.repo.UpdateFields(ctx, item); err != nil {
			return err
		}
		return s.repo.RenameAssociationKey(ctx, oldKey, item.PlacementKey)
	}); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, errors.New("Banner 位置标识已存在")
		}
		return nil, err
	}
	return item, nil
}

func (s *BannerPlacementService) Delete(ctx context.Context, id uint64) error {
	item, err := s.repo.GetByID(ctx, uint(id))
	if err != nil {
		return notFoundOr(err, "Banner 位置不存在")
	}
	count, err := s.repo.BannerCount(ctx, item.PlacementKey)
	if err != nil {
		return err
	}
	if count > 0 {
		return errors.New("该位置仍被 Banner 使用，无法删除")
	}
	return s.repo.Delete(ctx, uint(id))
}

func (s *BannerPlacementService) prepareAndValidate(ctx context.Context, req *BannerPlacementPayload, currentID uint64) error {
	req.PlacementName = strings.TrimSpace(req.PlacementName)
	req.PlacementKey = strings.TrimSpace(req.PlacementKey)
	req.Description = strings.TrimSpace(req.Description)
	req.CoverImage = strings.TrimSpace(req.CoverImage)
	if req.PlacementName == "" || req.CoverImage == "" {
		return errors.New("Banner 位置名称和封面图不能为空")
	}
	if !bannerPlacementKeyPattern.MatchString(req.PlacementKey) {
		return errors.New("Banner 位置标识只能包含字母、数字、下划线和中划线")
	}
	item, err := s.repo.GetByKey(ctx, req.PlacementKey)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if item.ID != currentID {
		return errors.New("Banner 位置标识已存在")
	}
	return nil
}

func applyBannerPlacementPayload(item *model.VideoBannerPlacement, req *BannerPlacementPayload) {
	item.PlacementName = req.PlacementName
	item.PlacementKey = req.PlacementKey
	item.Description = req.Description
	item.CoverImage = req.CoverImage
	item.Sort = req.Sort
	item.Status = req.Status
}

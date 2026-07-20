package service

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"ai-video/internal/model"
	"ai-video/internal/repository"

	"gorm.io/gorm"
)

var displayPositionKeyPattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

type DisplayPositionService struct {
	repo *repository.DisplayPositionRepo
}

func NewDisplayPositionService() *DisplayPositionService {
	return &DisplayPositionService{repo: repository.NewDisplayPositionRepo()}
}

type ListDisplayPositionRequest struct {
	Status  *int8  `form:"status" binding:"omitempty,oneof=0 1"`
	Keyword string `form:"keyword"`
}

type DisplayPositionPayload struct {
	PositionName string `json:"position_name" binding:"required,max=128"`
	PositionKey  string `json:"position_key" binding:"required,max=64"`
	Description  string `json:"description" binding:"max=500"`
	CoverImage   string `json:"cover_image" binding:"required,max=1024"`
	Sort         int    `json:"sort"`
	Status       int8   `json:"status" binding:"oneof=0 1"`
}

func (s *DisplayPositionService) List(ctx context.Context, page, pageSize int, req *ListDisplayPositionRequest) ([]model.VideoDisplayPosition, int64, error) {
	return s.repo.PageList(ctx, page, pageSize, &repository.DisplayPositionListFilter{
		Status: req.Status, Keyword: strings.TrimSpace(req.Keyword),
	})
}

func (s *DisplayPositionService) ListOptions(ctx context.Context) ([]model.VideoDisplayPosition, error) {
	return s.repo.ListOptions(ctx)
}

func (s *DisplayPositionService) GetByID(ctx context.Context, id uint64) (*model.VideoDisplayPosition, error) {
	item, err := s.repo.GetByID(ctx, uint(id))
	if err != nil {
		return nil, notFoundOr(err, "展示位置不存在")
	}
	return item, nil
}

func (s *DisplayPositionService) Create(ctx context.Context, req *DisplayPositionPayload) (*model.VideoDisplayPosition, error) {
	if err := s.validateKey(ctx, strings.TrimSpace(req.PositionKey), 0); err != nil {
		return nil, err
	}
	item := &model.VideoDisplayPosition{}
	applyDisplayPositionPayload(item, req)
	if err := s.repo.Create(ctx, item); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, errors.New("位置标识已存在")
		}
		return nil, err
	}
	return item, nil
}

func (s *DisplayPositionService) Update(ctx context.Context, id uint64, req *DisplayPositionPayload) (*model.VideoDisplayPosition, error) {
	item, err := s.repo.GetByID(ctx, uint(id))
	if err != nil {
		return nil, notFoundOr(err, "展示位置不存在")
	}
	if err := s.validateKey(ctx, strings.TrimSpace(req.PositionKey), id); err != nil {
		return nil, err
	}
	oldKey := item.PositionKey
	applyDisplayPositionPayload(item, req)
	if err := repository.Transaction(ctx, func(ctx context.Context) error {
		if err := s.repo.UpdateFields(ctx, item); err != nil {
			return err
		}
		return s.repo.RenameTemplateTypePositionKey(ctx, oldKey, item.PositionKey)
	}); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, errors.New("位置标识已存在")
		}
		return nil, err
	}
	return item, nil
}

func (s *DisplayPositionService) Delete(ctx context.Context, id uint64) error {
	item, err := s.repo.GetByID(ctx, uint(id))
	if err != nil {
		return notFoundOr(err, "展示位置不存在")
	}
	bannerCount, err := s.repo.BannerCount(ctx, item.PositionKey)
	if err != nil {
		return err
	}
	if bannerCount > 0 {
		return errors.New("该展示位置仍被 Banner 使用，无法删除")
	}
	typeCount, err := s.repo.TemplateTypeCount(ctx, item.PositionKey)
	if err != nil {
		return err
	}
	if typeCount > 0 {
		return errors.New("该展示位置仍被模板分类使用，无法删除")
	}
	return s.repo.Delete(ctx, uint(id))
}

func (s *DisplayPositionService) validateKey(ctx context.Context, key string, currentID uint64) error {
	if !displayPositionKeyPattern.MatchString(key) {
		return errors.New("位置标识只能包含字母、数字、下划线和中划线")
	}
	item, err := s.repo.GetByKey(ctx, key)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if item.ID != currentID {
		return errors.New("位置标识已存在")
	}
	return nil
}

func applyDisplayPositionPayload(item *model.VideoDisplayPosition, req *DisplayPositionPayload) {
	item.PositionName = strings.TrimSpace(req.PositionName)
	item.PositionKey = strings.TrimSpace(req.PositionKey)
	item.Description = strings.TrimSpace(req.Description)
	item.CoverImage = strings.TrimSpace(req.CoverImage)
	item.Sort = req.Sort
	item.Status = req.Status
}

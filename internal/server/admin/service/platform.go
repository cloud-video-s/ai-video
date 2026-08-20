package service

import (
	"context"
	"errors"
	"net/url"
	"regexp"
	"strings"

	"ai-video/internal/gen/model"
	"ai-video/internal/repository"

	"gorm.io/gorm"
)

var platformCodePattern = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

type PlatformService struct{ repo *repository.PlatformRepo }

func NewPlatformService() *PlatformService {
	return &PlatformService{repo: repository.NewPlatformRepo()}
}

type ListPlatformRequest struct {
	ListSortRequest
	Keyword string `form:"keyword"`
	Status  *int32 `form:"status" binding:"omitempty,oneof=0 1"`
}

type PlatformPayload struct {
	Name        string `json:"name" binding:"required,max=64"`
	Code        string `json:"code" binding:"required,max=32"`
	BaseURL     string `json:"base_url" binding:"required,max=255"`
	Description string `json:"description" binding:"max=255"`
	Status      int32  `json:"status" binding:"oneof=0 1"`
}

func (s *PlatformService) List(ctx context.Context, page, pageSize int, req *ListPlatformRequest) ([]model.VideoPlatform, int64, error) {
	return s.repo.PageList(ctx, page, pageSize, &repository.PlatformListFilter{
		ListSort: req.listSort(),
		Keyword:  strings.TrimSpace(req.Keyword), Status: req.Status,
	})
}

func (s *PlatformService) ListOptions(ctx context.Context) ([]model.VideoPlatform, error) {
	return s.repo.ListOptions(ctx)
}

func (s *PlatformService) Get(ctx context.Context, id int64) (*model.VideoPlatform, error) {
	item, err := s.repo.GetByID(ctx, uint(id))
	if err != nil {
		return nil, notFoundOr(err, "平台不存在")
	}
	return item, nil
}

func (s *PlatformService) Create(ctx context.Context, req *PlatformPayload) (*model.VideoPlatform, error) {
	if err := s.validatePayload(ctx, req, 0); err != nil {
		return nil, err
	}
	item := &model.VideoPlatform{}
	applyPlatformPayload(item, req)
	if err := s.repo.Create(ctx, item); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, errors.New("平台编码已存在")
		}
		return nil, err
	}
	return item, nil
}

func (s *PlatformService) Update(ctx context.Context, id int64, req *PlatformPayload) (*model.VideoPlatform, error) {
	item, err := s.repo.GetByID(ctx, uint(id))
	if err != nil {
		return nil, notFoundOr(err, "平台不存在")
	}
	if err := s.validatePayload(ctx, req, id); err != nil {
		return nil, err
	}
	applyPlatformPayload(item, req)
	if err := s.repo.UpdateFields(ctx, item); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, errors.New("平台编码已存在")
		}
		return nil, err
	}
	return item, nil
}

func (s *PlatformService) Delete(ctx context.Context, id int64) error {
	if _, err := s.repo.GetByID(ctx, uint(id)); err != nil {
		return notFoundOr(err, "平台不存在")
	}
	count, err := s.repo.ModelCount(ctx, id)
	if err != nil {
		return err
	}
	if count > 0 {
		return errors.New("该平台仍关联模型，请先删除或迁移关联模型")
	}
	// BaseRepo.Delete uses GORM's DeletedAt scope, so platform deletion is
	// always a soft delete.
	return s.repo.Delete(ctx, uint(id))
}

func (s *PlatformService) validatePayload(ctx context.Context, req *PlatformPayload, currentID int64) error {
	req.Name = strings.TrimSpace(req.Name)
	req.Code = strings.TrimSpace(req.Code)
	req.BaseURL = strings.TrimRight(strings.TrimSpace(req.BaseURL), "/")
	req.Description = strings.TrimSpace(req.Description)
	if req.Name == "" || !platformCodePattern.MatchString(req.Code) {
		return errors.New("平台名称不能为空，平台编码只能包含字母、数字、点、下划线和中划线")
	}
	parsed, err := url.Parse(req.BaseURL)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.User != nil {
		return errors.New("平台基础域名必须是有效且不含用户凭据的 HTTP(S) 地址")
	}
	existing, err := s.repo.GetByCode(ctx, req.Code)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if existing.ID != currentID {
		return errors.New("平台编码已存在")
	}
	return nil
}

func applyPlatformPayload(item *model.VideoPlatform, req *PlatformPayload) {
	item.Name = req.Name
	item.Code = req.Code
	item.BaseURL = req.BaseURL
	item.Description = req.Description
	item.Status = req.Status
}

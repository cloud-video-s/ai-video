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

type VideoAppService struct{ repo *repository.VideoAppRepo }

func NewVideoAppService() *VideoAppService {
	return &VideoAppService{repo: repository.NewVideoAppRepo()}
}

type ListVideoAppRequest struct {
	Keyword string  `form:"keyword" binding:"max=255"`
	AppCode string  `form:"app_code" binding:"max=60"`
	Status  *uint32 `form:"status" binding:"omitempty,oneof=0 1"`
}

type VideoAppPayload struct {
	Name        string `json:"name" binding:"required,max=255"`
	AppCode     string `json:"app_code" binding:"required,max=60"`
	Status      uint32 `json:"status" binding:"oneof=0 1"`
	Sort        uint32 `json:"sort" binding:"max=999999"`
	Description string `json:"description" binding:"max=10000"`
}

var videoAppCodePattern = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

func (s *VideoAppService) List(ctx context.Context, page, pageSize int, req *ListVideoAppRequest) ([]model.VideoApp, int64, error) {
	return s.repo.PageList(ctx, page, pageSize, &repository.VideoAppListFilter{
		Keyword: strings.TrimSpace(req.Keyword), AppCode: strings.TrimSpace(req.AppCode), Status: req.Status,
	})
}

func (s *VideoAppService) GetByID(ctx context.Context, id uint64) (*model.VideoApp, error) {
	item, err := s.repo.GetByID(ctx, uint(id))
	if err != nil {
		return nil, notFoundOr(err, "应用不存在")
	}
	return item, nil
}

func (s *VideoAppService) ListOptions(ctx context.Context) ([]model.VideoApp, error) {
	return s.repo.ListOptions(ctx)
}

func (s *VideoAppService) Create(ctx context.Context, req *VideoAppPayload) (*model.VideoApp, error) {
	if err := s.validatePayload(ctx, req, 0); err != nil {
		return nil, err
	}
	item := &model.VideoApp{}
	applyVideoAppPayload(item, req)
	if err := s.repo.Create(ctx, item); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, errors.New("应用标识已存在")
		}
		return nil, err
	}
	return item, nil
}

func (s *VideoAppService) Update(ctx context.Context, id uint64, req *VideoAppPayload) (*model.VideoApp, error) {
	item, err := s.repo.GetByID(ctx, uint(id))
	if err != nil {
		return nil, notFoundOr(err, "应用不存在")
	}
	if err := s.validatePayload(ctx, req, id); err != nil {
		return nil, err
	}
	//if item.AppCode != strings.TrimSpace(req.AppID) {
	//	count, err := s.repo.PackageCount(ctx, item.AppCode)
	//	if err != nil {
	//		return nil, err
	//	}
	//	if count > 0 {
	//		return nil, errors.New("该应用仍有关联安装包，不能修改应用标识")
	//	}
	//}
	applyVideoAppPayload(item, req)
	if err := s.repo.UpdateFields(ctx, item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *VideoAppService) Delete(ctx context.Context, id uint64) error {
	item, err := s.repo.GetByID(ctx, uint(id))
	if err != nil {
		return notFoundOr(err, "应用不存在")
	}
	count, err := s.repo.PackageCount(ctx, item.AppCode)
	if err != nil {
		return err
	}
	if count > 0 {
		return errors.New("该应用仍有关联安装包，请先删除安装包")
	}
	return s.repo.Delete(ctx, uint(id))
}

func (s *VideoAppService) validatePayload(ctx context.Context, req *VideoAppPayload, currentID uint64) error {
	if strings.TrimSpace(req.Name) == "" {
		return errors.New("应用名称不能为空")
	}
	//code := strings.TrimSpace(req.AppID)
	//if !videoAppCodePattern.MatchString(code) {
	//	return errors.New("应用标识只能包含字母、数字、点、下划线和中划线")
	//}
	existing, err := s.repo.GetByAppCode(ctx, req.AppCode)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if existing.ID != currentID {
		return errors.New("应用标识已存在")
	}
	return nil
}

func applyVideoAppPayload(item *model.VideoApp, req *VideoAppPayload) {
	item.Name = strings.TrimSpace(req.Name)
	//item.AppCode = strings.TrimSpace(req.AppID)
	//item.Status = req.Status
	//item.Sort = req.Sort
	item.Description = strings.TrimSpace(req.Description)
}

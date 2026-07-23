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

var packageCodePattern = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

type PackageService struct {
	repo                *repository.PackageRepo
	appRepo             *repository.VideoAppRepo
	vipSubscriptionRepo *repository.VIPSubscriptionRepo
}

func NewPackageService() *PackageService {
	return &PackageService{
		repo: repository.NewPackageRepo(), appRepo: repository.NewVideoAppRepo(),
		vipSubscriptionRepo: repository.NewVIPSubscriptionRepo(),
	}
}

type ListPackageRequest struct {
	AppCode     string  `form:"app_code" binding:"max=50"`
	PackageCode string  `form:"package_code" binding:"max=128"`
	SystemType  *uint32 `form:"system_type" binding:"omitempty,oneof=1 2"`
	Status      *int8   `form:"status" binding:"omitempty,oneof=0 1"`
	Keyword     string  `form:"keyword" binding:"max=255"`
}

type PackagePayload struct {
	PackageName string `json:"package_name" binding:"required,max=128"`
	PackageCode string `json:"package_code" binding:"required,max=128"`
	AppID       uint64 `json:"app_id" binding:"required,max=50"`
	Description string `json:"description" binding:"max=10000"`
	Sort        int64  `json:"sort" binding:"min=0,max=999999"`
	Status      uint8  `json:"status" binding:"oneof=0 1"`
	SystemType  uint8  `json:"system_type" binding:"required,oneof=1 2"`
}

func (s *PackageService) List(ctx context.Context, page, pageSize int, req *ListPackageRequest) ([]model.VideoPackage, int64, error) {
	return s.repo.PageList(ctx, page, pageSize, &repository.PackageListFilter{
		AppCode: strings.TrimSpace(req.AppCode), PackageCode: strings.TrimSpace(req.PackageCode),
		SystemType: req.SystemType, Status: req.Status, Keyword: strings.TrimSpace(req.Keyword),
	})
}

func (s *PackageService) GetByID(ctx context.Context, id uint64) (*model.VideoPackage, error) {
	item, err := s.repo.GetByID(ctx, uint(id))
	if err != nil {
		return nil, notFoundOr(err, "安装包不存在")
	}
	return item, nil
}

func (s *PackageService) ListOptions(ctx context.Context) ([]model.VideoPackage, error) {
	return s.repo.ListOptions(ctx)
}

func (s *PackageService) Create(ctx context.Context, req *PackagePayload) (*model.VideoPackage, error) {
	if err := s.validatePayload(ctx, req, 0); err != nil {
		return nil, err
	}
	item := &model.VideoPackage{}
	applyPackagePayload(item, req)
	if err := s.repo.Create(ctx, item); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, errors.New("包标识码已存在")
		}
		return nil, err
	}
	return item, nil
}

func (s *PackageService) Update(ctx context.Context, id uint64, req *PackagePayload) (*model.VideoPackage, error) {
	item, err := s.repo.GetByID(ctx, uint(id))
	if err != nil {
		return nil, notFoundOr(err, "安装包不存在")
	}
	if item.PackageCode != strings.TrimSpace(req.PackageCode) {
		return nil, errors.New("包标识码创建后不可修改")
	}
	if err := s.validatePayload(ctx, req, id); err != nil {
		return nil, err
	}
	applyPackagePayload(item, req)
	if err := s.repo.UpdateFields(ctx, item); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, errors.New("包标识码已存在")
		}
		return nil, err
	}
	return item, nil
}

func (s *PackageService) Delete(ctx context.Context, id uint64) error {
	if _, err := s.repo.GetByID(ctx, uint(id)); err != nil {
		return notFoundOr(err, "安装包不存在")
	}
	count, err := s.repo.VersionCount(ctx, id)
	if err != nil {
		return err
	}
	if count > 0 {
		return errors.New("该安装包仍存在版本记录，请先删除版本")
	}
	count, err = s.repo.TemplateCount(ctx, id)
	if err != nil {
		return err
	}
	if count > 0 {
		return errors.New("该安装包仍被视频模板使用，无法删除")
	}
	count, err = s.repo.PointsPackageCount(ctx, id)
	if err != nil {
		return err
	}
	if count > 0 {
		return errors.New("该安装包仍被积分套餐使用，无法删除")
	}
	count, err = s.vipSubscriptionRepo.PackageCount(ctx, id)
	if err != nil {
		return err
	}
	if count > 0 {
		return errors.New("该安装包仍被 VIP 订阅套餐使用，无法删除")
	}
	return s.repo.Delete(ctx, uint(id))
}

func (s *PackageService) validatePayload(ctx context.Context, req *PackagePayload, currentID uint64) error {
	name := strings.TrimSpace(req.PackageName)
	code := strings.TrimSpace(req.PackageCode)
	if name == "" || code == "" || req.AppID == 0 {
		return errors.New("包名称、包标识码和所属应用不能为空")
	}
	if !packageCodePattern.MatchString(code) {
		return errors.New("包标识码只能包含字母、数字、点、下划线和中划线")
	}
	if _, err := s.appRepo.GetByAppCode(ctx, req.AppID); err != nil {
		return notFoundOr(err, "所属应用不存在")
	}
	existing, err := s.repo.GetByCode(ctx, code)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if existing.ID != currentID {
		return errors.New("包标识码已存在")
	}
	return nil
}

func applyPackagePayload(item *model.VideoPackage, req *PackagePayload) {
	item.PackageName = strings.TrimSpace(req.PackageName)
	item.PackageCode = strings.TrimSpace(req.PackageCode)
	//item.AppCode = strings.TrimSpace(req.AppCode)
	item.Description = strings.TrimSpace(req.Description)
	item.Sort = req.Sort
	item.Status = int8(req.Status)
	item.SystemType = req.SystemType
}

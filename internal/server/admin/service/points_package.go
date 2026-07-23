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

var (
	pointsProductIDPattern = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)
	resourceTypePattern    = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,31}$`)
)

type PointsPackageService struct {
	repo        *repository.PointsPackageRepo
	packageRepo *repository.PackageRepo
	channelRepo *repository.ChannelRepo
}

func NewPointsPackageService() *PointsPackageService {
	return &PointsPackageService{
		repo: repository.NewPointsPackageRepo(), packageRepo: repository.NewPackageRepo(),
		channelRepo: repository.NewChannelRepo(),
	}
}

type ListPointsPackageRequest struct {
	PackageID    uint64 `form:"package_id"`
	ChannelID    uint64 `form:"channel_id"`
	System       string `form:"system" binding:"max=32"`
	UserType     int    `form:"user_type" binding:"omitempty,oneof=1 2"`
	ResourceType string `form:"resource_type" binding:"max=32"`
	Status       *int8  `form:"status" binding:"omitempty,oneof=0 1"`
	Keyword      string `form:"keyword" binding:"max=255"`
}

type PointsPackagePayload struct {
	ProductCode   string   `json:"product_code" binding:"required,max=191"`
	Name          string   `json:"name" binding:"required,max=128"`
	PackageID     uint64   `json:"package_id" binding:"required"`
	Systems       []string `json:"systems" binding:"required,min=1,max=10,dive,required,max=32"`
	UserTypes     []int    `json:"user_types" binding:"required,min=1,max=2,dive,oneof=1 2"`
	ChannelIDs    []uint64 `json:"channel_ids" binding:"max=100,dive,gt=0"`
	ResourceType  string   `json:"resource_type" binding:"required,max=32"`
	Points        uint64   `json:"points" binding:"required"`
	Currency      string   `json:"currency" binding:"required,len=3"`
	SalePrice     float64  `json:"sale_price" binding:"gte=0"`
	ActualRevenue float64  `json:"actual_revenue" binding:"gte=0"`
	OriginalPrice float64  `json:"original_price" binding:"gte=0"`
	BadgeText     string   `json:"badge_text" binding:"max=64"`
	Description   string   `json:"description" binding:"max=1000"`
	ButtonText    string   `json:"button_text" binding:"max=128"`
	IsDefault     bool     `json:"is_default"`
	Status        int8     `json:"status" binding:"oneof=0 1"`
	Sort          int      `json:"sort"`
}

type PointsPackageStatusPayload struct {
	Status int8 `json:"status" binding:"oneof=0 1"`
}

func (s *PointsPackageService) List(ctx context.Context, page, pageSize int, req *ListPointsPackageRequest) ([]model.VideoPointsPackage, int64, error) {
	return s.repo.PageList(ctx, page, pageSize, &repository.PointsPackageListFilter{
		PackageID: req.PackageID, ChannelID: req.ChannelID, System: strings.ToLower(strings.TrimSpace(req.System)),
		UserType: req.UserType, ResourceType: strings.ToLower(strings.TrimSpace(req.ResourceType)),
		Status: req.Status, Keyword: strings.TrimSpace(req.Keyword),
	})
}

func (s *PointsPackageService) ListOptions(ctx context.Context) ([]model.VideoPointsPackage, error) {
	return s.repo.ListOptions(ctx)
}

func (s *PointsPackageService) GetByID(ctx context.Context, id uint64) (*model.VideoPointsPackage, error) {
	item, err := s.repo.GetDetail(ctx, id)
	if err != nil {
		return nil, notFoundOr(err, "积分套餐不存在")
	}
	return item, nil
}

func (s *PointsPackageService) Create(ctx context.Context, req *PointsPackagePayload) (*model.VideoPointsPackage, error) {
	if err := s.prepareAndValidate(ctx, req, 0); err != nil {
		return nil, err
	}
	item := &model.VideoPointsPackage{}
	applyPointsPackagePayload(item, req)
	err := repository.Transaction(ctx, func(ctx context.Context) error {
		if err := s.repo.Create(ctx, item); err != nil {
			return err
		}
		if err := s.repo.ReplaceTargets(ctx, item, req.PackageID, req.ChannelIDs); err != nil {
			return err
		}
		if item.IsDefault == 1 {
			return s.repo.ClearDefaults(ctx, req.PackageID, item.ResourceType, item.ID)
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, errors.New("产品 ID 已存在，每个积分套餐必须唯一")
		}
		return nil, err
	}
	return s.repo.GetDetail(ctx, item.ID)
}

func (s *PointsPackageService) Update(ctx context.Context, id uint64, req *PointsPackagePayload) (*model.VideoPointsPackage, error) {
	item, err := s.repo.GetDetail(ctx, id)
	if err != nil {
		return nil, notFoundOr(err, "积分套餐不存在")
	}
	if err := s.prepareAndValidate(ctx, req, id); err != nil {
		return nil, err
	}
	if req.ProductCode != item.ProductCode {
		return nil, errors.New("产品 ID 创建后不可修改")
	}
	applyPointsPackagePayload(item, req)
	err = repository.Transaction(ctx, func(ctx context.Context) error {
		if err := s.repo.UpdateFields(ctx, item); err != nil {
			return err
		}
		if err := s.repo.ReplaceTargets(ctx, item, req.PackageID, req.ChannelIDs); err != nil {
			return err
		}
		if item.IsDefault == 1 {
			return s.repo.ClearDefaults(ctx, req.PackageID, item.ResourceType, item.ID)
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, errors.New("产品 ID 已存在，每个积分套餐必须唯一")
		}
		return nil, err
	}
	return s.repo.GetDetail(ctx, item.ID)
}

func (s *PointsPackageService) Delete(ctx context.Context, id uint64) error {
	if _, err := s.repo.GetDetail(ctx, id); err != nil {
		return notFoundOr(err, "积分套餐不存在")
	}
	return s.repo.DeleteWithTargets(ctx, id)
}

func (s *PointsPackageService) UpdateStatus(ctx context.Context, id uint64, status int8) error {
	if _, err := s.repo.GetDetail(ctx, id); err != nil {
		return notFoundOr(err, "积分套餐不存在")
	}
	return s.repo.UpdateStatus(ctx, id, status)
}

func (s *PointsPackageService) SetDefault(ctx context.Context, id uint64) error {
	item, err := s.repo.GetDetail(ctx, id)
	if err != nil {
		return notFoundOr(err, "积分套餐不存在")
	}
	return s.repo.SetDefault(ctx, item)
}

func (s *PointsPackageService) prepareAndValidate(ctx context.Context, req *PointsPackagePayload, currentID uint64) error {
	req.ProductCode = strings.TrimSpace(req.ProductCode)
	req.Name = strings.TrimSpace(req.Name)
	req.ResourceType = strings.ToLower(strings.TrimSpace(req.ResourceType))
	req.Currency = strings.ToUpper(strings.TrimSpace(req.Currency))
	if !pointsProductIDPattern.MatchString(req.ProductCode) {
		return errors.New("产品 ID 只能包含字母、数字、点、下划线和中划线")
	}
	if !resourceTypePattern.MatchString(req.ResourceType) {
		return errors.New("资源类型只能包含小写字母、数字、下划线和中划线")
	}
	if req.Points == 0 {
		return errors.New("赠送积分必须大于 0")
	}
	if err := validatePointsPackageMoney(req); err != nil {
		return err
	}
	var err error
	//if req.Systems, err = normalizeSystemTypes(req.Systems); err != nil {
	//	return err
	//}
	if req.UserTypes, err = normalizeUserTypes(req.UserTypes); err != nil {
		return err
	}
	appPackage, err := s.packageRepo.GetByID(ctx, uint(req.PackageID))
	if err != nil {
		return notFoundOr(err, "安装包不存在")
	}
	if appPackage.Status != 1 {
		return errors.New("所选安装包已禁用")
	}
	if req.ChannelIDs, err = normalizeTargetIDs(req.ChannelIDs, "渠道"); err != nil {
		return err
	}
	for _, id := range req.ChannelIDs {
		if _, err := s.channelRepo.GetByID(ctx, uint(id)); err != nil {
			return notFoundOr(err, "渠道不存在")
		}
	}
	existing, err := s.repo.GetByProductID(ctx, req.ProductCode)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if existing.ID != currentID {
		return errors.New("产品 ID 已存在，每个积分套餐必须唯一")
	}
	return nil
}

func validatePointsPackageMoney(req *PointsPackagePayload) error {
	const maxMoney = 9999999999.99
	if req.SalePrice < 0 || req.ActualRevenue < 0 || req.OriginalPrice < 0 ||
		req.SalePrice > maxMoney || req.ActualRevenue > maxMoney || req.OriginalPrice > maxMoney {
		return errors.New("金额必须在 0 到 9999999999.99 之间")
	}
	if req.SalePrice > 0 && req.ActualRevenue > req.SalePrice {
		return errors.New("实际收入不能高于销售金额")
	}
	if req.OriginalPrice > 0 && req.OriginalPrice < req.SalePrice {
		return errors.New("划线价不能低于销售金额")
	}
	return nil
}

func applyPointsPackagePayload(item *model.VideoPointsPackage, req *PointsPackagePayload) {
	item.ProductCode = req.ProductCode
	item.Name = req.Name
	//item.Systems = req.Systems
	//item.UserTypes = req.UserTypes
	item.ResourceType = req.ResourceType
	item.Points = req.Points
	item.Currency = req.Currency
	item.SalePrice = req.SalePrice
	item.ActualRevenue = req.ActualRevenue
	item.OriginalPrice = req.OriginalPrice
	item.BadgeText = strings.TrimSpace(req.BadgeText)
	item.Description = strings.TrimSpace(req.Description)
	item.ButtonText = strings.TrimSpace(req.ButtonText)
	//item.IsDefault = req.IsDefault
	//item.Status = req.Status
	//item.Sort = req.Sort
}

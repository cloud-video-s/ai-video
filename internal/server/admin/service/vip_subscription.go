package service

import (
	"context"
	"errors"
	"strings"

	"ai-video/internal/gen/model"
	"ai-video/internal/repository"

	"gorm.io/gorm"
)

type VIPSubscriptionService struct {
	repo *repository.VIPSubscriptionRepo
}

func NewVIPSubscriptionService() *VIPSubscriptionService {
	return &VIPSubscriptionService{repo: repository.NewVIPSubscriptionRepo()}
}

type ListVIPSubscriptionRequest struct {
	AppID          uint64 `form:"app_id"`
	PackageID      uint64 `form:"package_id"`
	VersionID      uint64 `form:"version_id"`
	CountryID      uint64 `form:"country_id"`
	PlacementKey   string `form:"placement_key" binding:"omitempty,max=100"`
	LevelID        int64  `form:"level_id"`
	PlanType       string `form:"plan_type" binding:"omitempty,oneof=normal trial paywall"`
	VipType        string `form:"vip_type" binding:"omitempty,oneof=android ios pc web"`
	DisplayMode    *int8  `form:"display_mode" binding:"omitempty,oneof=0 1"`
	Status         *int8  `form:"status" binding:"omitempty,oneof=0 1"`
	IsSubscription *bool  `form:"is_subscription"`
	Keyword        string `form:"keyword" binding:"max=255"`
}

// VIPSubscriptionPayload 与新模型保持一致，投放范围使用 APP、包、版本和国家关联表。
type VIPSubscriptionPayload struct {
	AppIDs                   []uint64 `json:"app_ids" binding:"max=100,dive,gt=0"`
	PackageIDs               []uint64 `json:"package_ids" binding:"required,min=1,max=100,dive,gt=0"`
	VersionIDs               []uint64 `json:"version_ids" binding:"max=100,dive,gt=0"`
	CountryIDs               []uint64 `json:"country_ids" binding:"max=250,dive,gt=0"`
	PlacementKey             string   `json:"placement_key" binding:"required,max=100"`
	LevelID                  int64    `json:"level_id" binding:"required,gt=0"`
	VipType                  string   `json:"vip_type" binding:"required,oneof=android ios pc web"`
	ProductCode              string   `json:"product_code" binding:"required,max=191"`
	Name                     string   `json:"name" binding:"required,max=128"`
	PlanType                 string   `json:"plan_type" binding:"required,oneof=normal trial paywall"`
	AppVersion               string   `json:"app_version" binding:"max=32"`
	Currency                 string   `json:"currency" binding:"required,len=3"`
	FirstSubscriptionPrice   float64  `json:"first_subscription_price" binding:"gte=0"`
	FirstSubscriptionRevenue float64  `json:"first_subscription_revenue" binding:"gte=0"`
	FirstBonusPoints         uint64   `json:"first_bonus_points"`
	OriginalPrice            float64  `json:"original_price" binding:"gte=0"`
	VIPDurationDays          uint     `json:"vip_duration_days"`
	TrialDays                uint     `json:"trial_days"`
	RenewalText              string   `json:"renewal_text" binding:"max=255"`
	BadgeText                string   `json:"badge_text" binding:"max=64"`
	AgreementDefaultChecked  bool     `json:"agreement_default_checked"`
	DisplayMode              int8     `json:"display_mode" binding:"oneof=0 1"`
	Status                   int8     `json:"status" binding:"oneof=0 1"`
	FreeTrial                bool     `json:"free_trial"`
	IsSubscription           bool     `json:"is_subscription"`
	IsDefault                bool     `json:"is_default"`
	SubscriptionDescription  string   `json:"subscription_description" binding:"max=500"`
	SubscriptionPrice        float64  `json:"subscription_price" binding:"gte=0"`
	SubscriptionRevenue      float64  `json:"subscription_revenue" binding:"gte=0"`
	SubscriptionPoints       uint64   `json:"subscription_points"`
	SubscriptionPeriod       string   `json:"subscription_period" binding:"max=64"`
	Sort                     int64    `json:"sort"`
	Description              string   `json:"description" binding:"max=1000"`
	Remark                   string   `json:"remark" binding:"max=1000"`
}

type VIPSubscriptionStatusPayload struct {
	Status int8 `json:"status" binding:"oneof=0 1"`
}

type VIPSubscriptionDisplayPayload struct {
	DisplayMode int8 `json:"display_mode" binding:"oneof=0 1"`
}

type CloneVIPSubscriptionRequest struct {
	ProductCode string `json:"product_code" binding:"required,max=191"`
	Name        string `json:"name" binding:"omitempty,max=128"`
}

func (s *VIPSubscriptionService) List(ctx context.Context, page, pageSize int, req *ListVIPSubscriptionRequest) ([]model.VideoVipSubscription, int64, error) {
	return s.repo.PageList(ctx, page, pageSize, &repository.VIPSubscriptionListFilter{
		AppID: req.AppID, PackageID: req.PackageID, VersionID: req.VersionID, CountryID: req.CountryID,
		PlacementKey: strings.TrimSpace(req.PlacementKey), LevelID: req.LevelID,
		PlanType: strings.TrimSpace(req.PlanType), VipType: strings.TrimSpace(req.VipType),
		DisplayMode: req.DisplayMode, Status: req.Status, IsSubscription: req.IsSubscription,
		Keyword: strings.TrimSpace(req.Keyword),
	})
}

func (s *VIPSubscriptionService) GetByID(ctx context.Context, id uint64) (*model.VideoVipSubscription, error) {
	item, err := s.repo.GetDetail(ctx, id)
	if err != nil {
		return nil, notFoundOr(err, "VIP 订阅套餐不存在")
	}
	return item, nil
}

func (s *VIPSubscriptionService) Create(ctx context.Context, req *VIPSubscriptionPayload) (*model.VideoVipSubscription, error) {
	if err := s.prepareAndValidate(ctx, req); err != nil {
		return nil, err
	}
	item := &model.VideoVipSubscription{}
	applyVIPSubscriptionPayload(item, req)
	err := repository.Transaction(ctx, func(txCtx context.Context) error {
		if err := s.repo.Create(txCtx, item); err != nil {
			return err
		}
		if err := s.repo.ReplaceTargets(txCtx, item, vipSubscriptionTargets(req)); err != nil {
			return err
		}
		if item.IsDefault == 1 {
			return s.clearDefaults(txCtx, req.PackageIDs, item.VipType, item.ID)
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, errors.New("该 VIP 类型下的产品编码已存在")
		}
		return nil, err
	}
	return s.repo.GetDetail(ctx, item.ID)
}

func (s *VIPSubscriptionService) Update(ctx context.Context, id uint64, req *VIPSubscriptionPayload) (*model.VideoVipSubscription, error) {
	item, err := s.repo.GetDetail(ctx, id)
	if err != nil {
		return nil, notFoundOr(err, "VIP 订阅套餐不存在")
	}
	if err := s.prepareAndValidate(ctx, req); err != nil {
		return nil, err
	}
	applyVIPSubscriptionPayload(item, req)
	err = repository.Transaction(ctx, func(txCtx context.Context) error {
		if err := s.repo.UpdateFields(txCtx, item); err != nil {
			return err
		}
		if err := s.repo.ReplaceTargets(txCtx, item, vipSubscriptionTargets(req)); err != nil {
			return err
		}
		if item.IsDefault == 1 {
			return s.clearDefaults(txCtx, req.PackageIDs, item.VipType, item.ID)
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, errors.New("该 VIP 类型下的产品编码已存在")
		}
		return nil, err
	}
	return s.repo.GetDetail(ctx, item.ID)
}

func (s *VIPSubscriptionService) clearDefaults(ctx context.Context, packageIDs []uint64, vipType string, exceptID uint64) error {
	for _, packageID := range packageIDs {
		if err := s.repo.ClearDefaults(ctx, packageID, vipType, exceptID); err != nil {
			return err
		}
	}
	return nil
}

func (s *VIPSubscriptionService) Delete(ctx context.Context, id uint64) error {
	if _, err := s.repo.GetDetail(ctx, id); err != nil {
		return notFoundOr(err, "VIP 订阅套餐不存在")
	}
	return s.repo.DeleteWithTargets(ctx, id)
}

func (s *VIPSubscriptionService) UpdateStatus(ctx context.Context, id uint64, status int8) error {
	if _, err := s.repo.GetDetail(ctx, id); err != nil {
		return notFoundOr(err, "VIP 订阅套餐不存在")
	}
	return s.repo.UpdateStatus(ctx, id, status)
}

func (s *VIPSubscriptionService) UpdateDisplayMode(ctx context.Context, id uint64, mode int8) error {
	if _, err := s.repo.GetDetail(ctx, id); err != nil {
		return notFoundOr(err, "VIP 订阅套餐不存在")
	}
	return s.repo.UpdateDisplayMode(ctx, id, mode)
}

func (s *VIPSubscriptionService) SetDefault(ctx context.Context, id uint64) error {
	item, err := s.repo.GetDetail(ctx, id)
	if err != nil {
		return notFoundOr(err, "VIP 订阅套餐不存在")
	}
	return s.repo.SetDefault(ctx, item)
}

func (s *VIPSubscriptionService) Clone(ctx context.Context, id uint64, req *CloneVIPSubscriptionRequest) (*model.VideoVipSubscription, error) {
	source, err := s.repo.GetDetail(ctx, id)
	if err != nil {
		return nil, notFoundOr(err, "VIP 订阅套餐不存在")
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = source.Name + "（副本）"
	}
	payload := vipSubscriptionPayloadFromModel(source)
	payload.ProductCode = strings.TrimSpace(req.ProductCode)
	payload.Name = name
	payload.IsDefault = false
	return s.Create(ctx, payload)
}

func (s *VIPSubscriptionService) prepareAndValidate(ctx context.Context, req *VIPSubscriptionPayload) error {
	var err error
	if req.AppIDs, err = normalizeTargetIDs(req.AppIDs, "APP"); err != nil {
		return err
	}
	if req.PackageIDs, err = normalizeTargetIDs(req.PackageIDs, "安装包"); err != nil {
		return err
	}
	if len(req.PackageIDs) == 0 {
		return errors.New("至少选择一个安装包")
	}
	if req.VersionIDs, err = normalizeTargetIDs(req.VersionIDs, "版本"); err != nil {
		return err
	}
	if req.CountryIDs, err = normalizeTargetIDs(req.CountryIDs, "国家"); err != nil {
		return err
	}
	req.PlacementKey = strings.TrimSpace(req.PlacementKey)
	req.VipType = strings.ToLower(strings.TrimSpace(req.VipType))
	req.ProductCode = strings.TrimSpace(req.ProductCode)
	req.Name = strings.TrimSpace(req.Name)
	req.PlanType = strings.ToLower(strings.TrimSpace(req.PlanType))
	req.AppVersion = strings.TrimSpace(req.AppVersion)
	req.Currency = strings.ToUpper(strings.TrimSpace(req.Currency))
	req.SubscriptionPeriod = strings.TrimSpace(req.SubscriptionPeriod)
	if req.PlacementKey == "" || req.LevelID <= 0 || req.ProductCode == "" || req.Name == "" {
		return errors.New("展示位置、会员等级、产品编码和 VIP 名称不能为空")
	}
	if _, err := s.repo.GetPlacementByKey(ctx, req.PlacementKey); err != nil {
		return notFoundOr(err, "VIP 展示位置不存在")
	}
	if _, err := s.repo.GetLevelByID(ctx, req.LevelID); err != nil {
		return notFoundOr(err, "VIP 会员等级不存在")
	}
	if req.FirstSubscriptionRevenue > req.FirstSubscriptionPrice && req.FirstSubscriptionPrice > 0 {
		return errors.New("首次订阅实际收入不能高于首次订阅金额")
	}
	if req.SubscriptionRevenue > req.SubscriptionPrice && req.SubscriptionPrice > 0 {
		return errors.New("续订实际收入不能高于续订金额")
	}
	if req.OriginalPrice > 0 && req.OriginalPrice < req.FirstSubscriptionPrice {
		return errors.New("划线金额不能低于首次订阅金额")
	}
	if req.FreeTrial && req.TrialDays == 0 {
		return errors.New("开启免费体验时，试用天数必须大于 0")
	}
	return nil
}

func applyVIPSubscriptionPayload(item *model.VideoVipSubscription, req *VIPSubscriptionPayload) {
	item.VipType, item.ProductCode, item.Name = req.VipType, req.ProductCode, req.Name
	item.LevelID, item.PlacementKey, item.PlanType = req.LevelID, req.PlacementKey, req.PlanType
	item.AppVersion, item.Currency = req.AppVersion, req.Currency
	item.FirstSubscriptionPrice = req.FirstSubscriptionPrice
	item.FirstSubscriptionRevenue = req.FirstSubscriptionRevenue
	item.FirstBonusPoints, item.OriginalPrice = req.FirstBonusPoints, req.OriginalPrice
	item.VIPDurationDays, item.TrialDays = req.VIPDurationDays, req.TrialDays
	item.RenewalText, item.BadgeText = strings.TrimSpace(req.RenewalText), strings.TrimSpace(req.BadgeText)
	item.AgreementDefaultChecked = boolToInt8(req.AgreementDefaultChecked)
	item.DisplayMode, item.Status = req.DisplayMode, req.Status
	item.FreeTrial, item.IsSubscription, item.IsDefault = boolToInt8(req.FreeTrial), boolToInt8(req.IsSubscription), boolToInt8(req.IsDefault)
	item.SubscriptionDescription = strings.TrimSpace(req.SubscriptionDescription)
	item.SubscriptionPrice, item.SubscriptionRevenue = req.SubscriptionPrice, req.SubscriptionRevenue
	item.SubscriptionPoints, item.SubscriptionPeriod = req.SubscriptionPoints, req.SubscriptionPeriod
	item.Sort, item.Description, item.Remark = req.Sort, strings.TrimSpace(req.Description), strings.TrimSpace(req.Remark)
}

func vipSubscriptionTargets(req *VIPSubscriptionPayload) repository.VIPSubscriptionTargets {
	return repository.VIPSubscriptionTargets{
		AppIDs: req.AppIDs, PackageIDs: req.PackageIDs, VersionIDs: req.VersionIDs, CountryIDs: req.CountryIDs,
	}
}

func vipSubscriptionPayloadFromModel(item *model.VideoVipSubscription) *VIPSubscriptionPayload {
	return &VIPSubscriptionPayload{
		AppIDs: appIDs(item.Apps), PackageIDs: packageIDs(item.Packages), VersionIDs: versionIDs(item.Versions), CountryIDs: countryIDs(item.Countries),
		PlacementKey: item.PlacementKey, LevelID: item.LevelID, VipType: item.VipType, ProductCode: item.ProductCode,
		Name: item.Name, PlanType: item.PlanType, AppVersion: item.AppVersion, Currency: item.Currency,
		FirstSubscriptionPrice: item.FirstSubscriptionPrice, FirstSubscriptionRevenue: item.FirstSubscriptionRevenue,
		FirstBonusPoints: item.FirstBonusPoints, OriginalPrice: item.OriginalPrice,
		VIPDurationDays: item.VIPDurationDays, TrialDays: item.TrialDays,
		RenewalText: item.RenewalText, BadgeText: item.BadgeText,
		AgreementDefaultChecked: item.AgreementDefaultChecked == 1, DisplayMode: item.DisplayMode, Status: item.Status,
		FreeTrial: item.FreeTrial == 1, IsSubscription: item.IsSubscription == 1, IsDefault: item.IsDefault == 1,
		SubscriptionDescription: item.SubscriptionDescription, SubscriptionPrice: item.SubscriptionPrice,
		SubscriptionRevenue: item.SubscriptionRevenue, SubscriptionPoints: item.SubscriptionPoints,
		SubscriptionPeriod: item.SubscriptionPeriod, Sort: item.Sort, Description: item.Description, Remark: item.Remark,
	}
}

func appIDs(items []model.VideoApp) []uint64 {
	result := make([]uint64, len(items))
	for i := range items {
		result[i] = items[i].ID
	}
	return result
}

func packageIDs(items []model.VideoPackage) []uint64 {
	result := make([]uint64, len(items))
	for i := range items {
		result[i] = items[i].ID
	}
	return result
}

func versionIDs(items []model.VideoPackageVersion) []uint64 {
	result := make([]uint64, len(items))
	for i := range items {
		result[i] = items[i].ID
	}
	return result
}

func countryIDs(items []model.VideoCountry) []uint64 {
	result := make([]uint64, len(items))
	for i := range items {
		result[i] = items[i].ID
	}
	return result
}

func boolToInt8(value bool) int8 {
	if value {
		return 1
	}
	return 0
}

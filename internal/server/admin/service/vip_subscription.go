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
	AppCode        string `form:"app_code"`
	PackageCode    string `form:"package_code"`
	VersionCode    string `form:"version_code"`
	CountryCode    string `form:"country_code"`
	PlacementKey   string `form:"placement_key" binding:"omitempty,max=100"`
	LevelID        int64  `form:"level_id"`
	PlanType       string `form:"plan_type" binding:"omitempty,oneof=normal trial paywall"`
	VipType        uint64 `form:"vip_type" binding:"omitempty,oneof=android ios pc web"`
	DisplayMode    int8   `form:"display_mode" binding:"omitempty,oneof=0 1"`
	Status         int8   `form:"status" binding:"omitempty,oneof=0 1"`
	IsSubscription int8   `form:"is_subscription"`
	Keyword        string `form:"keyword" binding:"max=255"`
}

// VIPSubscriptionPayload 与新模型保持一致，投放范围使用 APP、包、版本和国家关联表。
type VIPSubscriptionPayload struct {
	AppCodes                 []string `json:"app_codes" binding:"max=100,dive,gt=0"`
	PackageCodes             []string `json:"package_codes" binding:"required,min=1,max=100,dive,gt=0"`
	VersionCodes             []string `json:"version_codes" binding:"max=100,dive,gt=0"`
	CountryCodes             []string `json:"country_codes" binding:"max=250,dive,gt=0"`
	LevelID                  uint64   `json:"level_id" binding:"required,gt=0"`
	VipType                  uint64   `json:"vip_type" binding:"required,oneof=android ios pc web"`
	SukCode                  string   `json:"suk_code" binding:"required,max=191"`
	Name                     string   `json:"name" binding:"required,max=128"`
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
	SubscriptionPeriod       uint32   `json:"subscription_period" binding:"max=64"`
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
	SukCode string `json:"suk_code" binding:"required,max=191"`
	Name    string `json:"name" binding:"omitempty,max=128"`
}

func (s *VIPSubscriptionService) List(ctx context.Context, page, pageSize int, req *ListVIPSubscriptionRequest) ([]repository.VIPSubscriptionRecord, int64, error) {
	return s.repo.PageList(ctx, page, pageSize, &repository.VIPSubscriptionListFilter{
		AppCode: req.AppCode, PackageCode: req.PackageCode, VersionCode: req.VersionCode, CountryCode: req.CountryCode,
		PlacementKey: strings.TrimSpace(req.PlacementKey), LevelID: req.LevelID,
		PlanType: strings.TrimSpace(req.PlanType), VipType: req.VipType,
		DisplayMode: req.DisplayMode, Status: req.Status, IsSubscription: req.IsSubscription,
		Keyword: strings.TrimSpace(req.Keyword),
	})
}

func (s *VIPSubscriptionService) GetByID(ctx context.Context, id uint64) (*repository.VIPSubscriptionRecord, error) {
	item, err := s.repo.GetDetail(ctx, id)
	if err != nil {
		return nil, notFoundOr(err, "VIP 订阅套餐不存在")
	}
	return item, nil
}

func (s *VIPSubscriptionService) Create(ctx context.Context, req *VIPSubscriptionPayload) (*repository.VIPSubscriptionRecord, error) {
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
			return s.clearDefaults(txCtx, req.PackageCodes, item.VipType, item.ID)
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

func (s *VIPSubscriptionService) Update(ctx context.Context, id uint64, req *VIPSubscriptionPayload) (*repository.VIPSubscriptionRecord, error) {
	item, err := s.repo.GetDetail(ctx, id)
	if err != nil {
		return nil, notFoundOr(err, "VIP 订阅套餐不存在")
	}
	if err := s.prepareAndValidate(ctx, req); err != nil {
		return nil, err
	}
	applyVIPSubscriptionPayload(&item.VideoVipSubscription, req)
	err = repository.Transaction(ctx, func(txCtx context.Context) error {
		if err := s.repo.UpdateFields(txCtx, &item.VideoVipSubscription); err != nil {
			return err
		}
		if err := s.repo.ReplaceTargets(txCtx, &item.VideoVipSubscription, vipSubscriptionTargets(req)); err != nil {
			return err
		}
		if item.IsDefault == 1 {
			return s.clearDefaults(txCtx, req.PackageCodes, item.VipType, item.ID)
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

func (s *VIPSubscriptionService) clearDefaults(ctx context.Context, packageCodes []string, vipType uint64, exceptID uint64) error {
	for _, packageCode := range packageCodes {
		if err := s.repo.ClearDefaults(ctx, packageCode, vipType, exceptID); err != nil {
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

func (s *VIPSubscriptionService) Clone(ctx context.Context, id uint64, req *CloneVIPSubscriptionRequest) (*repository.VIPSubscriptionRecord, error) {
	source, err := s.repo.GetDetail(ctx, id)
	if err != nil {
		return nil, notFoundOr(err, "VIP 订阅套餐不存在")
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = source.Name + "（副本）"
	}
	payload := vipSubscriptionPayloadFromModel(source)
	payload.SukCode = strings.TrimSpace(req.SukCode)
	payload.Name = name
	payload.IsDefault = false
	return s.Create(ctx, payload)
}

func (s *VIPSubscriptionService) prepareAndValidate(ctx context.Context, req *VIPSubscriptionPayload) error {
	var err error
	if req.AppCodes, err = normalizeTargetIDs(req.AppCodes, "APP"); err != nil {
		return err
	}
	if req.PackageCodes, err = normalizeTargetIDs(req.PackageCodes, "安装包"); err != nil {
		return err
	}
	if len(req.PackageCodes) == 0 {
		return errors.New("至少选择一个安装包")
	}
	if req.VersionCodes, err = normalizeTargetIDs(req.VersionCodes, "版本"); err != nil {
		return err
	}
	if req.CountryCodes, err = normalizeTargetIDs(req.CountryCodes, "国家"); err != nil {
		return err
	}
	req.VipType = req.VipType
	req.SukCode = strings.TrimSpace(req.SukCode)
	req.Name = strings.TrimSpace(req.Name)
	req.Currency = strings.ToUpper(strings.TrimSpace(req.Currency))
	if req.VipType <= 0 || req.LevelID <= 0 || req.SukCode == "" || req.Name == "" {
		return errors.New("展示位置、会员等级、产品编码和 VIP 名称不能为空")
	}
	if _, err := s.repo.GetLevelByID(ctx, int64(req.LevelID)); err != nil {
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
	item.VipType, item.SukCode, item.Name = req.VipType, req.SukCode, req.Name
	item.LevelID, item.Currency = req.LevelID, req.Currency
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
		AppCodes: req.AppCodes, PackageCodes: req.PackageCodes, VersionCdes: req.VersionCodes, CountryCode: req.CountryCodes,
	}
}

func vipSubscriptionPayloadFromModel(item *repository.VIPSubscriptionRecord) *VIPSubscriptionPayload {
	return &VIPSubscriptionPayload{
		AppCodes: appIDs(item.Apps), PackageCodes: packageIDs(item.Packages), VersionCodes: versionIDs(item.Versions), CountryCodes: countryIDs(item.Countries),
		LevelID: item.LevelID, VipType: item.VipType,
		Name: item.Name, Currency: item.Currency,
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

func appIDs(items []model.VideoApp) []string {
	result := make([]string, len(items))
	for i := range items {
		result[i] = items[i].AppCode
	}
	return result
}

func packageIDs(items []model.VideoPackage) []string {
	result := make([]string, len(items))
	for i := range items {
		result[i] = items[i].PackageCode
	}
	return result
}

func versionIDs(items []model.VideoPackageVersion) []string {
	result := make([]string, len(items))
	for i := range items {
		result[i] = items[i].VersionCode
	}
	return result
}

func countryIDs(items []model.VideoCountry) []string {
	result := make([]string, len(items))
	for i := range items {
		result[i] = items[i].Code
	}
	return result
}

func boolToInt8(value bool) int8 {
	if value {
		return 1
	}
	return 0
}

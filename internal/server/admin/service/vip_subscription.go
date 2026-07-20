package service

import (
	"context"
	"errors"
	"strings"

	"ai-video/internal/model"
	"ai-video/internal/repository"

	"gorm.io/gorm"
)

type VIPSubscriptionService struct {
	repo         *repository.VIPSubscriptionRepo
	packageRepo  *repository.PackageRepo
	positionRepo *repository.DisplayPositionRepo
	channelRepo  *repository.ChannelRepo
}

func NewVIPSubscriptionService() *VIPSubscriptionService {
	return &VIPSubscriptionService{
		repo: repository.NewVIPSubscriptionRepo(), packageRepo: repository.NewPackageRepo(),
		positionRepo: repository.NewDisplayPositionRepo(), channelRepo: repository.NewChannelRepo(),
	}
}

type ListVIPSubscriptionRequest struct {
	PackageID         uint64 `form:"package_id"`
	DisplayPositionID uint64 `form:"display_position_id"`
	ChannelID         uint64 `form:"channel_id"`
	PlanType          string `form:"plan_type" binding:"omitempty,oneof=normal trial paywall"`
	Platform          string `form:"platform" binding:"omitempty,oneof=android ios pc web"`
	DisplayMode       *int8  `form:"display_mode" binding:"omitempty,oneof=0 1"`
	Status            *int8  `form:"status" binding:"omitempty,oneof=0 1"`
	IsSubscription    *bool  `form:"is_subscription"`
	Keyword           string `form:"keyword" binding:"max=255"`
}

type VIPSubscriptionPayload struct {
	PackageID                uint64   `json:"package_id" binding:"required"`
	Platform                 string   `json:"platform" binding:"required,oneof=android ios pc web"`
	ProductID                string   `json:"product_id" binding:"required,max=191"`
	Name                     string   `json:"name" binding:"required,max=128"`
	VIPLevel                 string   `json:"vip_level" binding:"required,max=64"`
	PlanType                 string   `json:"plan_type" binding:"required,oneof=normal trial paywall"`
	DisplayPositionIDs       []uint64 `json:"display_position_ids" binding:"max=100,dive,gt=0"`
	ChannelIDs               []uint64 `json:"channel_ids" binding:"max=100,dive,gt=0"`
	ExcludedChannelIDs       []uint64 `json:"excluded_channel_ids" binding:"max=100,dive,gt=0"`
	AppVersion               string   `json:"app_version" binding:"max=32"`
	Currency                 string   `json:"currency" binding:"required,len=3"`
	FirstSubscriptionPrice   float64  `json:"first_subscription_price" binding:"gte=0"`
	FirstSubscriptionRevenue float64  `json:"first_subscription_revenue" binding:"gte=0"`
	FirstBonusPoints         uint64   `json:"first_bonus_points"`
	OriginalPrice            float64  `json:"original_price" binding:"gte=0"`
	VIPDurationDays          uint32   `json:"vip_duration_days"`
	TrialDays                uint32   `json:"trial_days"`
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
	Sort                     int      `json:"sort"`
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
	ProductID string `json:"product_id" binding:"required,max=191"`
	Name      string `json:"name" binding:"omitempty,max=128"`
}

func (s *VIPSubscriptionService) List(ctx context.Context, page, pageSize int, req *ListVIPSubscriptionRequest) ([]model.VideoVIPSubscription, int64, error) {
	return s.repo.PageList(ctx, page, pageSize, &repository.VIPSubscriptionListFilter{
		PackageID: req.PackageID, DisplayPositionID: req.DisplayPositionID, ChannelID: req.ChannelID,
		PlanType: strings.TrimSpace(req.PlanType), Platform: strings.TrimSpace(req.Platform), DisplayMode: req.DisplayMode,
		Status: req.Status, IsSubscription: req.IsSubscription, Keyword: strings.TrimSpace(req.Keyword),
	})
}

func (s *VIPSubscriptionService) GetByID(ctx context.Context, id uint64) (*model.VideoVIPSubscription, error) {
	item, err := s.repo.GetDetail(ctx, id)
	if err != nil {
		return nil, notFoundOr(err, "VIP 订阅套餐不存在")
	}
	return item, nil
}

func (s *VIPSubscriptionService) Create(ctx context.Context, req *VIPSubscriptionPayload) (*model.VideoVIPSubscription, error) {
	if err := s.prepareAndValidate(ctx, req); err != nil {
		return nil, err
	}
	item := &model.VideoVIPSubscription{}
	applyVIPSubscriptionPayload(item, req)
	err := repository.Transaction(ctx, func(ctx context.Context) error {
		if err := s.repo.Create(ctx, item); err != nil {
			return err
		}
		if err := s.repo.ReplaceTargets(ctx, item, vipSubscriptionTargets(req)); err != nil {
			return err
		}
		if item.IsDefault {
			return s.repo.ClearDefaults(ctx, item.PackageID, item.Platform, item.ID)
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, errors.New("该应用包和平台下的产品 ID 已存在")
		}
		return nil, err
	}
	return s.repo.GetDetail(ctx, item.ID)
}

func (s *VIPSubscriptionService) Update(ctx context.Context, id uint64, req *VIPSubscriptionPayload) (*model.VideoVIPSubscription, error) {
	item, err := s.repo.GetDetail(ctx, id)
	if err != nil {
		return nil, notFoundOr(err, "VIP 订阅套餐不存在")
	}
	if err := s.prepareAndValidate(ctx, req); err != nil {
		return nil, err
	}
	applyVIPSubscriptionPayload(item, req)
	err = repository.Transaction(ctx, func(ctx context.Context) error {
		if err := s.repo.UpdateFields(ctx, item); err != nil {
			return err
		}
		if err := s.repo.ReplaceTargets(ctx, item, vipSubscriptionTargets(req)); err != nil {
			return err
		}
		if item.IsDefault {
			return s.repo.ClearDefaults(ctx, item.PackageID, item.Platform, item.ID)
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, errors.New("该应用包和平台下的产品 ID 已存在")
		}
		return nil, err
	}
	return s.repo.GetDetail(ctx, item.ID)
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

func (s *VIPSubscriptionService) Clone(ctx context.Context, id uint64, req *CloneVIPSubscriptionRequest) (*model.VideoVIPSubscription, error) {
	source, err := s.repo.GetDetail(ctx, id)
	if err != nil {
		return nil, notFoundOr(err, "VIP 订阅套餐不存在")
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = source.Name + "（副本）"
	}
	payload := vipSubscriptionPayloadFromModel(source)
	payload.ProductID = strings.TrimSpace(req.ProductID)
	payload.Name = name
	payload.IsDefault = false
	return s.Create(ctx, payload)
}

func (s *VIPSubscriptionService) prepareAndValidate(ctx context.Context, req *VIPSubscriptionPayload) error {
	var err error
	req.DisplayPositionIDs, err = normalizeTargetIDs(req.DisplayPositionIDs, "展示位置")
	if err != nil {
		return err
	}
	req.ChannelIDs, err = normalizeTargetIDs(req.ChannelIDs, "渠道")
	if err != nil {
		return err
	}
	req.ExcludedChannelIDs, err = normalizeTargetIDs(req.ExcludedChannelIDs, "排除渠道")
	if err != nil {
		return err
	}
	packageItem, err := s.packageRepo.GetByID(ctx, uint(req.PackageID))
	if err != nil {
		return notFoundOr(err, "应用包不存在")
	}
	req.Platform = strings.ToLower(strings.TrimSpace(req.Platform))
	if len(packageItem.SystemTypes) > 0 && !containsString(packageItem.SystemTypes, req.Platform) {
		return errors.New("所选应用包不支持该平台")
	}
	for _, id := range req.DisplayPositionIDs {
		if _, err := s.positionRepo.GetByID(ctx, uint(id)); err != nil {
			return notFoundOr(err, "展示位置不存在")
		}
	}
	channelSet := make(map[uint64]struct{}, len(req.ChannelIDs))
	for _, id := range req.ChannelIDs {
		if _, err := s.channelRepo.GetByID(ctx, uint(id)); err != nil {
			return notFoundOr(err, "渠道不存在")
		}
		channelSet[id] = struct{}{}
	}
	for _, id := range req.ExcludedChannelIDs {
		if _, exists := channelSet[id]; exists {
			return errors.New("渠道和排除渠道不能选择同一项")
		}
		if _, err := s.channelRepo.GetByID(ctx, uint(id)); err != nil {
			return notFoundOr(err, "排除渠道不存在")
		}
	}
	req.ProductID = strings.TrimSpace(req.ProductID)
	req.Name = strings.TrimSpace(req.Name)
	req.VIPLevel = strings.TrimSpace(req.VIPLevel)
	req.PlanType = strings.ToLower(strings.TrimSpace(req.PlanType))
	req.AppVersion = strings.TrimSpace(req.AppVersion)
	req.Currency = strings.ToUpper(strings.TrimSpace(req.Currency))
	if req.FirstSubscriptionRevenue > req.FirstSubscriptionPrice && req.FirstSubscriptionPrice > 0 {
		return errors.New("首次订阅实际收入不能高于首次订阅金额")
	}
	if req.SubscriptionRevenue > req.SubscriptionPrice && req.SubscriptionPrice > 0 {
		return errors.New("续订实际收入不能高于续订金额")
	}
	return nil
}

func applyVIPSubscriptionPayload(item *model.VideoVIPSubscription, req *VIPSubscriptionPayload) {
	item.PackageID = req.PackageID
	item.Platform = req.Platform
	item.ProductID = req.ProductID
	item.Name = req.Name
	item.VIPLevel = req.VIPLevel
	item.PlanType = req.PlanType
	item.AppVersion = req.AppVersion
	item.Currency = req.Currency
	item.FirstSubscriptionPrice = req.FirstSubscriptionPrice
	item.FirstSubscriptionRevenue = req.FirstSubscriptionRevenue
	item.FirstBonusPoints = req.FirstBonusPoints
	item.OriginalPrice = req.OriginalPrice
	item.VIPDurationDays = req.VIPDurationDays
	item.TrialDays = req.TrialDays
	item.RenewalText = strings.TrimSpace(req.RenewalText)
	item.BadgeText = strings.TrimSpace(req.BadgeText)
	item.AgreementDefaultChecked = req.AgreementDefaultChecked
	item.DisplayMode = req.DisplayMode
	item.Status = req.Status
	item.FreeTrial = req.FreeTrial
	item.IsSubscription = req.IsSubscription
	item.IsDefault = req.IsDefault
	item.SubscriptionDescription = strings.TrimSpace(req.SubscriptionDescription)
	item.SubscriptionPrice = req.SubscriptionPrice
	item.SubscriptionRevenue = req.SubscriptionRevenue
	item.SubscriptionPoints = req.SubscriptionPoints
	item.SubscriptionPeriod = strings.TrimSpace(req.SubscriptionPeriod)
	item.Sort = req.Sort
	item.Description = strings.TrimSpace(req.Description)
	item.Remark = strings.TrimSpace(req.Remark)
}

func vipSubscriptionTargets(req *VIPSubscriptionPayload) repository.VIPSubscriptionTargets {
	return repository.VIPSubscriptionTargets{DisplayPositionIDs: req.DisplayPositionIDs, ChannelIDs: req.ChannelIDs, ExcludedChannelIDs: req.ExcludedChannelIDs}
}

func vipSubscriptionPayloadFromModel(item *model.VideoVIPSubscription) *VIPSubscriptionPayload {
	return &VIPSubscriptionPayload{
		PackageID: item.PackageID, Platform: item.Platform, ProductID: item.ProductID, Name: item.Name,
		VIPLevel: item.VIPLevel, PlanType: item.PlanType, DisplayPositionIDs: positionIDs(item.DisplayPositions),
		ChannelIDs: channelIDs(item.Channels), ExcludedChannelIDs: channelIDs(item.ExcludedChannels),
		AppVersion: item.AppVersion, Currency: item.Currency, FirstSubscriptionPrice: item.FirstSubscriptionPrice,
		FirstSubscriptionRevenue: item.FirstSubscriptionRevenue, FirstBonusPoints: item.FirstBonusPoints,
		OriginalPrice: item.OriginalPrice, VIPDurationDays: item.VIPDurationDays, TrialDays: item.TrialDays,
		RenewalText: item.RenewalText, BadgeText: item.BadgeText, AgreementDefaultChecked: item.AgreementDefaultChecked,
		DisplayMode: item.DisplayMode, Status: item.Status, FreeTrial: item.FreeTrial, IsSubscription: item.IsSubscription,
		IsDefault: item.IsDefault, SubscriptionDescription: item.SubscriptionDescription,
		SubscriptionPrice: item.SubscriptionPrice, SubscriptionRevenue: item.SubscriptionRevenue,
		SubscriptionPoints: item.SubscriptionPoints, SubscriptionPeriod: item.SubscriptionPeriod,
		Sort: item.Sort, Description: item.Description, Remark: item.Remark,
	}
}

func positionIDs(items []model.VideoDisplayPosition) []uint64 {
	result := make([]uint64, len(items))
	for i := range items {
		result[i] = items[i].ID
	}
	return result
}
func channelIDs(items []model.VideoChannel) []uint64 {
	result := make([]uint64, len(items))
	for i := range items {
		result[i] = items[i].ChannelID
	}
	return result
}
func containsString(values []string, target string) bool {
	for _, value := range values {
		if strings.EqualFold(strings.TrimSpace(value), target) {
			return true
		}
	}
	return false
}

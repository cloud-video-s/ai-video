package service

import (
	"context"
	"errors"
	"strings"

	"ai-video/internal/model"
	"ai-video/internal/repository"

	"gorm.io/gorm"
)

type TemplateTypeService struct {
	repo         *repository.TemplateTypeRepo
	positionRepo *repository.DisplayPositionRepo
	countryRepo  *repository.CountryRepo
	channelRepo  *repository.ChannelRepo
	packageRepo  *repository.PackageRepo
}

func NewTemplateTypeService() *TemplateTypeService {
	return &TemplateTypeService{
		repo:         repository.NewTemplateTypeRepo(),
		positionRepo: repository.NewDisplayPositionRepo(),
		countryRepo:  repository.NewCountryRepo(),
		channelRepo:  repository.NewChannelRepo(),
		packageRepo:  repository.NewPackageRepo(),
	}
}

type ListTemplateTypeRequest struct {
	Status      *int8  `form:"status" binding:"omitempty,oneof=0 1"`
	PositionKey string `form:"position_key" binding:"omitempty,max=255"`
	CountryID   uint64 `form:"country_id"`
	ChannelID   uint64 `form:"channel_id"`
	PackageID   uint64 `form:"package_id"`
	Keyword     string `form:"keyword"`
}

type TemplateTypePayload struct {
	CategoryName         string   `json:"category_name" binding:"required,max=128"`
	DisplayPositionKeys  []string `json:"display_position_keys" binding:"required,min=1,max=100,dive,required,max=64"`
	CountryIDs           []uint64 `json:"country_ids" binding:"max=100,dive,gt=0"`
	ChannelIDs           []uint64 `json:"channel_ids" binding:"max=100,dive,gt=0"`
	PackageIDs           []uint64 `json:"package_ids" binding:"max=100,dive,gt=0"`
	UserTypes            []int    `json:"user_types" binding:"required,min=1,max=2,dive,oneof=1 2"`
	SubscriptionStatuses []string `json:"subscription_statuses" binding:"required,min=1,max=2,dive,oneof=subscribed unsubscribed"`
	Sort                 int64    `json:"sort"`
	Status               int8     `json:"status" binding:"oneof=0 1"`
	Description          string   `json:"description" binding:"max=500"`
	legacyCountry        string
	legacyAppPackage     string
	legacyChannelID      string
	legacyPackageID      *uint64
}

func (s *TemplateTypeService) List(ctx context.Context, page, pageSize int, req *ListTemplateTypeRequest) ([]model.VideoTemplateType, int64, error) {
	return s.repo.PageList(ctx, page, pageSize, &repository.TemplateTypeListFilter{
		Status: req.Status, PositionKey: strings.TrimSpace(req.PositionKey),
		CountryID: req.CountryID, ChannelID: req.ChannelID, PackageID: req.PackageID,
		Keyword: strings.TrimSpace(req.Keyword),
	})
}

func (s *TemplateTypeService) ListOptions(ctx context.Context) ([]model.VideoTemplateType, error) {
	return s.repo.ListOptions(ctx)
}

func (s *TemplateTypeService) GetByID(ctx context.Context, id uint64) (*model.VideoTemplateType, error) {
	item, err := s.repo.GetDetail(ctx, id)
	if err != nil {
		return nil, notFoundOr(err, "模板分类不存在")
	}
	return item, nil
}

func (s *TemplateTypeService) Create(ctx context.Context, req *TemplateTypePayload) (*model.VideoTemplateType, error) {
	if err := s.preparePositionIDs(ctx, req); err != nil {
		return nil, err
	}
	item := &model.VideoTemplateType{}
	applyTemplateTypePayload(item, req)
	if err := repository.Transaction(ctx, func(ctx context.Context) error {
		if err := s.repo.Create(ctx, item); err != nil {
			return err
		}
		return s.repo.ReplaceTargets(ctx, item, templateTypeTargetIDs(req))
	}); err != nil {
		return nil, err
	}
	return s.repo.GetDetail(ctx, item.ID)
}

func (s *TemplateTypeService) Update(ctx context.Context, id uint64, req *TemplateTypePayload) (*model.VideoTemplateType, error) {
	item, err := s.repo.GetDetail(ctx, id)
	if err != nil {
		return nil, notFoundOr(err, "模板分类不存在")
	}
	if err := s.preparePositionIDs(ctx, req); err != nil {
		return nil, err
	}
	applyTemplateTypePayload(item, req)
	if err := repository.Transaction(ctx, func(ctx context.Context) error {
		if err := s.repo.UpdateFields(ctx, item); err != nil {
			return err
		}
		return s.repo.ReplaceTargets(ctx, item, templateTypeTargetIDs(req))
	}); err != nil {
		return nil, err
	}
	return s.repo.GetDetail(ctx, item.ID)
}

func (s *TemplateTypeService) Delete(ctx context.Context, id uint64) error {
	if _, err := s.repo.GetByID(ctx, uint(id)); err != nil {
		return notFoundOr(err, "模板分类不存在")
	}
	count, err := s.repo.TemplateCount(ctx, id)
	if err != nil {
		return err
	}
	if count > 0 {
		return errors.New("该模板分类下仍有模板，无法删除")
	}
	return s.repo.DeleteWithDisplayPositions(ctx, id)
}

func applyTemplateTypePayload(item *model.VideoTemplateType, req *TemplateTypePayload) {
	item.CategoryName = strings.TrimSpace(req.CategoryName)
	item.Sort = req.Sort
	item.Status = req.Status
	item.Description = strings.TrimSpace(req.Description)
	item.LegacyCountry = req.legacyCountry
	item.LegacyAppPackage = req.legacyAppPackage
	item.LegacyChannelID = req.legacyChannelID
	item.LegacyPackageID = req.legacyPackageID
	item.UserTypes = append([]int(nil), req.UserTypes...)
	item.SubscriptionStatuses = append([]string(nil), req.SubscriptionStatuses...)
	item.LegacyUserType = uint32(req.UserTypes[0])
	item.LegacyIsSubscribed = req.SubscriptionStatuses[0] == "subscribed"
}

func (s *TemplateTypeService) preparePositionIDs(ctx context.Context, req *TemplateTypePayload) error {
	keys := normalizeStringValues(req.DisplayPositionKeys)
	if len(keys) == 0 {
		return errors.New("请至少选择一个展示位置")
	}
	for _, key := range keys {
		position, err := s.positionRepo.GetByKey(ctx, key)
		if err != nil {
			return notFoundOr(err, "展示位置不存在")
		}
		if position.Status != 1 {
			return errors.New("所选展示位置中包含已禁用项")
		}
	}
	req.DisplayPositionKeys = keys
	var err error
	if req.CountryIDs, err = normalizeTargetIDs(req.CountryIDs, "国家"); err != nil {
		return err
	}
	if req.ChannelIDs, err = normalizeTargetIDs(req.ChannelIDs, "渠道"); err != nil {
		return err
	}
	if req.PackageIDs, err = normalizeTargetIDs(req.PackageIDs, "安装包"); err != nil {
		return err
	}
	for i, id := range req.CountryIDs {
		country, lookupErr := s.countryRepo.GetByID(ctx, uint(id))
		if lookupErr != nil {
			return notFoundOr(lookupErr, "国家不存在")
		}
		if country.Status != 1 {
			return errors.New("所选国家中包含已禁用项")
		}
		if i == 0 {
			req.legacyCountry = country.Code
		}
	}
	for i, id := range req.ChannelIDs {
		channel, lookupErr := s.channelRepo.GetByID(ctx, uint(id))
		if lookupErr != nil {
			return notFoundOr(lookupErr, "渠道不存在")
		}
		if channel.Status != 1 {
			return errors.New("所选渠道中包含已禁用项")
		}
		if i == 0 {
			req.legacyChannelID = channel.ChannelCode
		}
	}
	for i, id := range req.PackageIDs {
		appPackage, lookupErr := s.packageRepo.GetByID(ctx, uint(id))
		if lookupErr != nil {
			return notFoundOr(lookupErr, "安装包不存在")
		}
		if appPackage.Status != 1 {
			return errors.New("所选安装包中包含已禁用项")
		}
		if i == 0 {
			req.legacyAppPackage = appPackage.PackageCode
			legacyID := id
			req.legacyPackageID = &legacyID
		}
	}
	if req.UserTypes, err = normalizeUserTypes(req.UserTypes); err != nil {
		return err
	}
	if req.SubscriptionStatuses, err = normalizeSubscriptionStatuses(req.SubscriptionStatuses); err != nil {
		return err
	}
	return nil
}

func templateTypeTargetIDs(req *TemplateTypePayload) repository.TemplateTypeTargetIDs {
	return repository.TemplateTypeTargetIDs{
		DisplayPositionKeys: req.DisplayPositionKeys,
		CountryIDs:          req.CountryIDs, ChannelIDs: req.ChannelIDs, PackageIDs: req.PackageIDs,
	}
}

func normalizeStringValues(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

type TemplateService struct {
	repo        *repository.TemplateRepo
	typeRepo    *repository.TemplateTypeRepo
	countryRepo *repository.CountryRepo
	packageRepo *repository.PackageRepo
	channelRepo *repository.ChannelRepo
}

func NewTemplateService() *TemplateService {
	return &TemplateService{
		repo: repository.NewTemplateRepo(), typeRepo: repository.NewTemplateTypeRepo(),
		countryRepo: repository.NewCountryRepo(), packageRepo: repository.NewPackageRepo(),
		channelRepo: repository.NewChannelRepo(),
	}
}

type ListTemplateRequest struct {
	VideoTemplateTypeID uint64 `form:"video_template_type_id"`
	PositionKey         string `form:"position_key" binding:"omitempty,max=64"`
	CountryID           uint64 `form:"country_id"`
	PackageID           uint64 `form:"package_id"`
	ChannelID           uint64 `form:"channel_id"`
	UserType            uint8  `form:"user_type" binding:"omitempty,oneof=1 2"`
	SubscriptionStatus  string `form:"subscription_status" binding:"omitempty,oneof=subscribed unsubscribed"`
	TemplateType        string `form:"template_type"`
	Status              *int8  `form:"status" binding:"omitempty,oneof=0 1"`
	Keyword             string `form:"keyword"`
}

type TemplatePayload struct {
	VideoTemplateTypeID  uint64   `json:"video_template_type_id" binding:"required"`
	CountryIDs           []uint64 `json:"country_ids" binding:"max=100,dive,gt=0"`
	PackageIDs           []uint64 `json:"package_ids" binding:"max=100,dive,gt=0"`
	ChannelIDs           []uint64 `json:"channel_ids" binding:"max=100,dive,gt=0"`
	UserTypes            []int    `json:"user_types" binding:"required,min=1,max=2,dive,oneof=1 2"`
	SubscriptionStatuses []string `json:"subscription_statuses" binding:"required,min=1,max=2,dive,oneof=subscribed unsubscribed"`
	Name                 string   `json:"name" binding:"required,max=128"`
	TemplateType         string   `json:"template_type" binding:"required,max=32"`
	Sort                 int      `json:"sort"`
	CoverImage           string   `json:"cover_image" binding:"required,max=1024"`
	TemplateVideo        string   `json:"template_video" binding:"required,max=1024"`
	ThumbnailVideo       string   `json:"thumbnail_video" binding:"max=1024"`
	Prompt               string   `json:"prompt" binding:"max=65535"`
	Status               int8     `json:"status" binding:"oneof=0 1"`
	Description          string   `json:"description" binding:"max=500"`
}

func (s *TemplateService) List(ctx context.Context, page, pageSize int, req *ListTemplateRequest) ([]model.VideoTemplate, int64, error) {
	return s.repo.PageList(ctx, page, pageSize, &repository.TemplateListFilter{
		VideoTemplateTypeID: req.VideoTemplateTypeID,
		PositionKey:         strings.TrimSpace(req.PositionKey),
		CountryID:           req.CountryID, PackageID: req.PackageID,
		ChannelID: req.ChannelID, UserType: req.UserType, SubscriptionStatus: req.SubscriptionStatus,
		TemplateType: strings.TrimSpace(req.TemplateType), Status: req.Status,
		Keyword: strings.TrimSpace(req.Keyword),
	})
}

func (s *TemplateService) GetByID(ctx context.Context, id uint64) (*model.VideoTemplate, error) {
	item, err := s.repo.GetWithType(ctx, id)
	if err != nil {
		return nil, notFoundOr(err, "模板不存在")
	}
	return item, nil
}

func (s *TemplateService) Create(ctx context.Context, req *TemplatePayload) (*model.VideoTemplate, error) {
	if err := s.ensureTypeExists(ctx, req.VideoTemplateTypeID); err != nil {
		return nil, err
	}
	if err := s.prepareAndValidateTargets(ctx, req); err != nil {
		return nil, err
	}
	item := &model.VideoTemplate{}
	applyTemplatePayload(item, req)
	if err := repository.Transaction(ctx, func(ctx context.Context) error {
		if err := s.repo.Create(ctx, item); err != nil {
			return err
		}
		return s.repo.ReplaceTargets(ctx, item, targetIDsFromPayload(req))
	}); err != nil {
		return nil, err
	}
	return s.repo.GetWithType(ctx, item.ID)
}

func (s *TemplateService) Update(ctx context.Context, id uint64, req *TemplatePayload) (*model.VideoTemplate, error) {
	item, err := s.repo.GetWithType(ctx, id)
	if err != nil {
		return nil, notFoundOr(err, "模板不存在")
	}
	if err := s.ensureTypeExists(ctx, req.VideoTemplateTypeID); err != nil {
		return nil, err
	}
	if err := s.prepareAndValidateTargets(ctx, req); err != nil {
		return nil, err
	}
	applyTemplatePayload(item, req)
	if err := repository.Transaction(ctx, func(ctx context.Context) error {
		if err := s.repo.UpdateFields(ctx, item); err != nil {
			return err
		}
		return s.repo.ReplaceTargets(ctx, item, targetIDsFromPayload(req))
	}); err != nil {
		return nil, err
	}
	return s.repo.GetWithType(ctx, item.ID)
}

func (s *TemplateService) Delete(ctx context.Context, id uint64) error {
	if _, err := s.repo.GetWithType(ctx, id); err != nil {
		return notFoundOr(err, "模板不存在")
	}
	return s.repo.DeleteWithTargets(ctx, id)
}

func (s *TemplateService) ensureTypeExists(ctx context.Context, id uint64) error {
	_, err := s.typeRepo.GetByID(ctx, uint(id))
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return errors.New("模板分类不存在")
	}
	return err
}

func applyTemplatePayload(item *model.VideoTemplate, req *TemplatePayload) {
	item.VideoTemplateTypeID = req.VideoTemplateTypeID
	item.UserTypes = append([]int(nil), req.UserTypes...)
	item.SubscriptionStatuses = append([]string(nil), req.SubscriptionStatuses...)
	item.Name = strings.TrimSpace(req.Name)
	item.TemplateType = strings.TrimSpace(req.TemplateType)
	item.Sort = req.Sort
	item.CoverImage = strings.TrimSpace(req.CoverImage)
	item.TemplateVideo = strings.TrimSpace(req.TemplateVideo)
	item.ThumbnailVideo = strings.TrimSpace(req.ThumbnailVideo)
	item.Prompt = strings.TrimSpace(req.Prompt)
	item.Status = req.Status
	item.Description = strings.TrimSpace(req.Description)
}

func (s *TemplateService) prepareAndValidateTargets(ctx context.Context, req *TemplatePayload) error {
	var err error
	if req.CountryIDs, err = normalizeTargetIDs(req.CountryIDs, "国家"); err != nil {
		return err
	}
	if req.PackageIDs, err = normalizeTargetIDs(req.PackageIDs, "安装包"); err != nil {
		return err
	}
	if req.ChannelIDs, err = normalizeTargetIDs(req.ChannelIDs, "渠道"); err != nil {
		return err
	}
	if req.UserTypes, err = normalizeUserTypes(req.UserTypes); err != nil {
		return err
	}
	if req.SubscriptionStatuses, err = normalizeSubscriptionStatuses(req.SubscriptionStatuses); err != nil {
		return err
	}
	for _, id := range req.CountryIDs {
		if _, err := s.countryRepo.GetByID(ctx, uint(id)); err != nil {
			return notFoundOr(err, "国家不存在")
		}
	}
	for _, id := range req.PackageIDs {
		if _, err := s.packageRepo.GetByID(ctx, uint(id)); err != nil {
			return notFoundOr(err, "安装包不存在")
		}
	}
	for _, id := range req.ChannelIDs {
		if _, err := s.channelRepo.GetByID(ctx, uint(id)); err != nil {
			return notFoundOr(err, "渠道不存在")
		}
	}
	return nil
}

func normalizeTargetKeys(values []string, label string) ([]string, error) {
	if len(values) > 100 {
		return nil, errors.New(label + "最多选择 100 项")
	}
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, id := range values {
		if id == "" {
			return nil, errors.New(label + " ID 无效")
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	return result, nil
}

func normalizeUserTypes(values []int) ([]int, error) {
	result := make([]int, 0, 2)
	seen := map[int]bool{}
	for _, value := range values {
		if value != 1 && value != 2 {
			return nil, errors.New("用户类型无效")
		}
		if !seen[value] {
			seen[value] = true
			result = append(result, value)
		}
	}
	if len(result) == 0 {
		return nil, errors.New("请至少选择一种用户类型")
	}
	return result, nil
}

func normalizeTargetIDs(values []uint64, label string) ([]uint64, error) {
	if len(values) > 100 {
		return nil, errors.New(label + "最多选择 100 项")
	}
	result := make([]uint64, 0, len(values))
	seen := make(map[uint64]struct{}, len(values))
	for _, id := range values {
		if id == 0 {
			return nil, errors.New(label + " ID 无效")
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	return result, nil
}

func normalizeSubscriptionStatuses(values []string) ([]string, error) {
	result := make([]string, 0, 2)
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if value != "subscribed" && value != "unsubscribed" {
			return nil, errors.New("订阅状态无效")
		}
		if !seen[value] {
			seen[value] = true
			result = append(result, value)
		}
	}
	if len(result) == 0 {
		return nil, errors.New("请至少选择一种订阅状态")
	}
	return result, nil
}

func targetIDsFromPayload(req *TemplatePayload) repository.TemplateTargetIDs {
	return repository.TemplateTargetIDs{
		CountryIDs: req.CountryIDs, PackageIDs: req.PackageIDs, ChannelIDs: req.ChannelIDs,
	}
}

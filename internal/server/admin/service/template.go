package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"ai-video/internal/gen/model"
	"ai-video/internal/repository"

	"gorm.io/gorm"
)

type TemplateTypeService struct {
	repo         *repository.TemplateTypeRepo
	positionRepo *repository.DisplayPositionRepo
	countryRepo  *repository.CountryRepo
	appRepo      *repository.VideoAppRepo
	packageRepo  *repository.PackageRepo
	versionRepo  *repository.PackageVersionRepo
}

func NewTemplateTypeService() *TemplateTypeService {
	return &TemplateTypeService{
		repo:         repository.NewTemplateTypeRepo(),
		positionRepo: repository.NewDisplayPositionRepo(),
		countryRepo:  repository.NewCountryRepo(),
		appRepo:      repository.NewVideoAppRepo(),
		packageRepo:  repository.NewPackageRepo(),
		versionRepo:  repository.NewPackageVersionRepo(),
	}
}

type ListTemplateTypeRequest struct {
	Status      *int8  `form:"status" binding:"omitempty,oneof=0 1"`
	PositionKey string `form:"position_key" binding:"omitempty,max=255"`
	CountryID   uint64 `form:"country_id"`
	AppCode     string `form:"app_code" binding:"omitempty,max=50"`
	PackageCode string `form:"package_code" binding:"omitempty,max=50"`
	VersionCode string `form:"version_code" binding:"omitempty,max=50"`
	Keyword     string `form:"keyword"`
}

// TemplateTypeAppRulePayload 描述分类可投放的精确 APP、包与版本组合。
// app_rules 为空表示全部 APP/包/版本，不向关系表写入默认数据。
type TemplateTypeAppRulePayload struct {
	AppCode string `json:"app_code" binding:"required,max=50"`
}

type TemplateTypePayload struct {
	CategoryName         string                       `json:"category_name" binding:"required,max=128"`
	DisplayPositionKeys  []string                     `json:"display_position_keys" binding:"max=100,dive,required,max=64"`
	CountryCodes         []string                     `json:"country_codes" binding:"max=100,dive,gt=0"`
	AppRules             []TemplateTypeAppRulePayload `json:"app_rules" binding:"max=100,dive"`
	UserTypes            []int                        `json:"user_types" binding:"required,min=1,max=2,dive,oneof=1 2"`
	SubscriptionStatuses []string                     `json:"subscription_statuses" binding:"required,min=1,max=2,dive,oneof=subscribed unsubscribed"`
	Sort                 int64                        `json:"sort"`
	Status               int8                         `json:"status" binding:"oneof=0 1"`
	Description          string                       `json:"description" binding:"max=500"`
}

func (s *TemplateTypeService) List(ctx context.Context, page, pageSize int, req *ListTemplateTypeRequest) ([]model.VideoTemplateType, int64, error) {
	return s.repo.PageList(ctx, page, pageSize, &repository.TemplateTypeListFilter{
		Status: req.Status, PositionKey: strings.TrimSpace(req.PositionKey),
		CountryID: req.CountryID, AppCode: strings.TrimSpace(req.AppCode),
		PackageCode: strings.TrimSpace(req.PackageCode), VersionCode: strings.TrimSpace(req.VersionCode),
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
	if err := s.prepareTargets(ctx, req); err != nil {
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
	if err := s.prepareTargets(ctx, req); err != nil {
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
	userTypes, _ := json.Marshal(req.UserTypes)
	subscriptionStatuses, _ := json.Marshal(req.SubscriptionStatuses)
	item.UserTypes = string(userTypes)
	item.SubscriptionStatuses = string(subscriptionStatuses)
}

// prepareTargets 校验分类关系并统一去重。三个关系数组为空都表示“全部”。
func (s *TemplateTypeService) prepareTargets(ctx context.Context, req *TemplateTypePayload) error {
	keys := normalizeStringValues(req.DisplayPositionKeys)
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
	if req.CountryCodes, err = normalizeTargetIDs(req.CountryCodes, "国家"); err != nil {
		return err
	}
	for _, id := range req.CountryCodes {
		country, lookupErr := s.countryRepo.GetEnabledByCode(ctx, id)
		if lookupErr != nil {
			return notFoundOr(lookupErr, "国家不存在")
		}
		if country.Status != 1 {
			return errors.New("所选国家中包含已禁用项")
		}
	}
	normalizedRules := make([]TemplateTypeAppRulePayload, 0, len(req.AppRules))
	for _, rule := range req.AppRules {
		app, lookupErr := s.appRepo.GetByAppCode(ctx, rule.AppCode)
		if lookupErr != nil {
			return notFoundOr(lookupErr, "APP 不存在")
		}
		if app.Status != 1 {
			return errors.New("所选 APP 中包含已禁用项")
		}
		normalizedRules = append(normalizedRules, rule)
	}
	req.AppRules = normalizedRules
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
		CountryCodes:        req.CountryCodes,
		AppRules: func() []repository.TemplateTypeAppRule {
			rules := make([]repository.TemplateTypeAppRule, 0, len(req.AppRules))
			for _, rule := range req.AppRules {
				rules = append(rules, repository.TemplateTypeAppRule{
					AppCode: rule.AppCode,
				})
			}
			return rules
		}(),
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
	repo     *repository.TemplateRepo
	typeRepo *repository.TemplateTypeRepo
}

func NewTemplateService() *TemplateService {
	return &TemplateService{
		repo: repository.NewTemplateRepo(), typeRepo: repository.NewTemplateTypeRepo(),
	}
}

type ListTemplateRequest struct {
	VideoTemplateTypeID uint64 `form:"video_template_type_id"`
	PositionKey         string `form:"position_key" binding:"omitempty,max=64"`
	UserType            uint8  `form:"user_type" binding:"omitempty,oneof=1 2"`
	SubscriptionStatus  string `form:"subscription_status" binding:"omitempty,oneof=subscribed unsubscribed"`
	TemplateType        string `form:"template_type"`
	Status              *int8  `form:"status" binding:"omitempty,oneof=0 1"`
	Keyword             string `form:"keyword"`
}

type TemplatePayload struct {
	VideoTemplateTypeID  uint64   `json:"video_template_type_id" binding:"required"`
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
		TemplateType:        strings.TrimSpace(req.TemplateType), Status: req.Status,
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

func (s *TemplateService) ListOptions(ctx context.Context) ([]model.VideoTemplate, error) {
	return s.repo.ListOptions(ctx)
}

func (s *TemplateService) Create(ctx context.Context, req *TemplatePayload) (*model.VideoTemplate, error) {
	if err := s.ensureTypeExists(ctx, req.VideoTemplateTypeID); err != nil {
		return nil, err
	}
	if err := prepareTemplateAudience(req); err != nil {
		return nil, err
	}
	item := &model.VideoTemplate{}
	applyTemplatePayload(item, req)
	if err := s.repo.Create(ctx, item); err != nil {
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
	if err := prepareTemplateAudience(req); err != nil {
		return nil, err
	}
	applyTemplatePayload(item, req)
	if err := s.repo.UpdateFields(ctx, item); err != nil {
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
	//item.UserTypes = append([]int(nil), req.UserTypes...)
	//item.SubscriptionStatuses = append([]string(nil), req.SubscriptionStatuses...)
	item.Name = strings.TrimSpace(req.Name)
	item.TemplateType = strings.TrimSpace(req.TemplateType)
	item.Sort = int64(req.Sort)
	item.CoverImage = strings.TrimSpace(req.CoverImage)
	item.TemplateVideo = strings.TrimSpace(req.TemplateVideo)
	item.ThumbnailVideo = strings.TrimSpace(req.ThumbnailVideo)
	item.Prompt = strings.TrimSpace(req.Prompt)
	item.Status = req.Status
	item.Description = strings.TrimSpace(req.Description)
}

// prepareTemplateAudience 仅处理模板自身的用户类型和订阅状态字段。
// 国家、APP/包/版本由模板分类统一控制，展示位置由独立配置表控制。
func prepareTemplateAudience(req *TemplatePayload) error {
	var err error
	if req.UserTypes, err = normalizeUserTypes(req.UserTypes); err != nil {
		return err
	}
	if req.SubscriptionStatuses, err = normalizeSubscriptionStatuses(req.SubscriptionStatuses); err != nil {
		return err
	}
	return nil
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

func normalizeTargetIDs(values []string, label string) ([]string, error) {
	if len(values) > 100 {
		return nil, errors.New(label + "最多选择 100 项")
	}
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, id := range values {
		if id == "" {
			return nil, errors.New(label + " Code 无效")
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

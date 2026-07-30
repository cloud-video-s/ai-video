package service

import (
	"context"
	"errors"
	"strings"

	"ai-video/internal/domain"
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

// TemplateTypeAppRulePayload 描述分类可投放的 APP。
// app_rules 为空表示全部 APP，不向 APP 关系表写入默认数据。
type TemplateTypeAppRulePayload struct {
	AppCode string `json:"app_code" binding:"required,max=50"`
}

type TemplateTypePayload struct {
	CategoryName        string                       `json:"category_name" binding:"required,max=128"`
	DisplayPositionKeys []string                     `json:"display_position_keys" binding:"max=100,dive,required,max=64"`
	CountryCodes        []string                     `json:"country_codes" binding:"max=100,dive,gt=0"`
	AppRules            []TemplateTypeAppRulePayload `json:"app_rules" binding:"max=100,dive"`
	PackageCodes        []string                     `json:"package_codes" binding:"max=100,dive,required,max=128"`
	VersionCodes        []string                     `json:"version_codes" binding:"max=100,dive,required,max=50"`
	Sort                int64                        `json:"sort"`
	Status              int8                         `json:"status" binding:"oneof=0 1"`
	Description         string                       `json:"description" binding:"max=500"`
}

func (s *TemplateTypeService) List(ctx context.Context, page, pageSize int, req *ListTemplateTypeRequest) ([]repository.TemplateTypeRecord, int64, error) {
	return s.repo.PageList(ctx, page, pageSize, &repository.TemplateTypeListFilter{
		Status: req.Status, PositionKey: strings.TrimSpace(req.PositionKey),
		CountryID: req.CountryID, AppCode: strings.TrimSpace(req.AppCode),
		PackageCode: strings.TrimSpace(req.PackageCode), VersionCode: strings.TrimSpace(req.VersionCode),
		Keyword: strings.TrimSpace(req.Keyword),
	})
}

func (s *TemplateTypeService) ListOptions(ctx context.Context) ([]repository.TemplateTypeRecord, error) {
	return s.repo.ListOptions(ctx)
}

func (s *TemplateTypeService) GetByID(ctx context.Context, id uint64) (*repository.TemplateTypeRecord, error) {
	item, err := s.repo.GetDetail(ctx, id)
	if err != nil {
		return nil, notFoundOr(err, "模板分类不存在")
	}
	return item, nil
}

func (s *TemplateTypeService) Create(ctx context.Context, req *TemplateTypePayload) (*repository.TemplateTypeRecord, error) {
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

func (s *TemplateTypeService) Update(ctx context.Context, id uint64, req *TemplateTypePayload) (*repository.TemplateTypeRecord, error) {
	item, err := s.repo.GetDetail(ctx, id)
	if err != nil {
		return nil, notFoundOr(err, "模板分类不存在")
	}
	if err := s.prepareTargets(ctx, req); err != nil {
		return nil, err
	}
	applyTemplateTypePayload(&item.VideoTemplateType, req)
	if err := repository.Transaction(ctx, func(ctx context.Context) error {
		if err := s.repo.UpdateFields(ctx, &item.VideoTemplateType); err != nil {
			return err
		}
		return s.repo.ReplaceTargets(ctx, &item.VideoTemplateType, templateTypeTargetIDs(req))
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
}

// prepareTargets 校验分类关系并统一去重。任一关系数组为空都表示该维度选择“全部”。
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
	for index, code := range req.CountryCodes {
		req.CountryCodes[index] = strings.ToUpper(strings.TrimSpace(code))
	}
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
	selectedApps := make(map[string]struct{}, len(req.AppRules))
	for _, rule := range req.AppRules {
		rule.AppCode = strings.TrimSpace(rule.AppCode)
		if rule.AppCode == "" {
			return errors.New("APP 不能为空")
		}
		if _, exists := selectedApps[rule.AppCode]; exists {
			continue
		}
		app, lookupErr := s.appRepo.GetByAppCode(ctx, rule.AppCode)
		if lookupErr != nil {
			return notFoundOr(lookupErr, "APP 不存在")
		}
		if app.Status != 1 {
			return errors.New("所选 APP 中包含已禁用项")
		}
		selectedApps[rule.AppCode] = struct{}{}
		normalizedRules = append(normalizedRules, rule)
	}
	req.AppRules = normalizedRules

	req.PackageCodes = normalizeStringValues(req.PackageCodes)
	if len(req.PackageCodes) > 0 && len(req.AppRules) == 0 {
		return errors.New("选择安装包前请先选择 APP")
	}
	selectedPackages := make(map[string]struct{}, len(req.PackageCodes))
	for _, code := range req.PackageCodes {
		item, lookupErr := s.packageRepo.GetByCode(ctx, code)
		if lookupErr != nil {
			return notFoundOr(lookupErr, "安装包不存在")
		}
		if item.Status != 1 {
			return errors.New("所选安装包中包含已禁用项")
		}
		if _, exists := selectedApps[item.AppCode]; !exists {
			return errors.New("所选安装包不属于已选 APP")
		}
		selectedPackages[code] = struct{}{}
	}

	req.VersionCodes = normalizeStringValues(req.VersionCodes)
	if len(req.VersionCodes) > 0 && len(req.PackageCodes) == 0 {
		return errors.New("选择版本前请先选择安装包")
	}
	for _, versionCode := range req.VersionCodes {
		found := false
		enabled := false
		for packageCode := range selectedPackages {
			item, lookupErr := s.versionRepo.GetByPackageVersion(ctx, packageCode, versionCode)
			if errors.Is(lookupErr, gorm.ErrRecordNotFound) {
				continue
			}
			if lookupErr != nil {
				return lookupErr
			}
			found = true
			if item.Status == 1 {
				enabled = true
				break
			}
		}
		if !found {
			return errors.New("所选版本不属于已选安装包")
		}
		if !enabled {
			return errors.New("所选版本中包含已禁用项")
		}
	}
	return nil
}

func templateTypeTargetIDs(req *TemplateTypePayload) repository.TemplateTypeTargetIDs {
	return repository.TemplateTypeTargetIDs{
		DisplayPositionKeys: req.DisplayPositionKeys,
		CountryCodes:        req.CountryCodes,
		PackageCodes:        req.PackageCodes,
		VersionCodes:        req.VersionCodes,
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
	repo             *repository.TemplateRepo
	typeRepo         *repository.TemplateTypeRepo
	modelRepo        *repository.ModelRepo
	parameterService *TemplateModelParameterService
}

func NewTemplateService() *TemplateService {
	return &TemplateService{
		repo: repository.NewTemplateRepo(), typeRepo: repository.NewTemplateTypeRepo(),
		modelRepo: repository.NewModelRepo(), parameterService: NewTemplateModelParameterService(),
	}
}

type ListTemplateRequest struct {
	TemplateTypeID       uint64 `form:"template_type_id"`
	LegacyTemplateTypeID uint64 `form:"video_template_type_id"`
	ModelID              uint64 `form:"model_id"`
	PositionKey          string `form:"position_key" binding:"omitempty,max=64"`
	TemplateType         int64  `form:"template_type" binding:"omitempty,oneof=1 2"`
	Status               *int32 `form:"status" binding:"omitempty,oneof=0 1"`
	Keyword              string `form:"keyword"`
}

type TemplatePayload struct {
	TemplateTypeID       uint64                  `json:"template_type_id"`
	LegacyTemplateTypeID uint64                  `json:"video_template_type_id"`
	ModelID              uint64                  `json:"model_id" binding:"required"`
	Name                 string                  `json:"name" binding:"required,max=128"`
	TemplateType         int64                   `json:"template_type" binding:"required,oneof=1 2"`
	Sort                 int                     `json:"sort"`
	CoverImageURL        string                  `json:"cover_image_url" binding:"required,max=1024"`
	OriginalURL          string                  `json:"original_url" binding:"required,max=1024"`
	ThumbnailURL         string                  `json:"thumbnail_url" binding:"max=1024"`
	Prompt               string                  `json:"prompt" binding:"max=65535"`
	Status               int32                   `json:"status" binding:"oneof=0 1"`
	Description          string                  `json:"description" binding:"max=500"`
	ModelParameters      []ModelParameterPayload `json:"model_parameters" binding:"omitempty,max=100,dive"`
}

func (s *TemplateService) List(ctx context.Context, page, pageSize int, req *ListTemplateRequest) ([]repository.TemplateRecord, int64, error) {
	templateTypeID := req.TemplateTypeID
	if templateTypeID == 0 {
		templateTypeID = req.LegacyTemplateTypeID
	}
	return s.repo.PageList(ctx, page, pageSize, &repository.TemplateListFilter{
		TemplateTypeID: templateTypeID,
		ModelID:        req.ModelID,
		PositionKey:    strings.TrimSpace(req.PositionKey),
		TemplateType:   req.TemplateType, Status: req.Status,
		Keyword: strings.TrimSpace(req.Keyword),
	})
}

func (s *TemplateService) GetByID(ctx context.Context, id uint64) (*repository.TemplateRecord, error) {
	item, err := s.repo.GetWithType(ctx, id)
	if err != nil {
		return nil, notFoundOr(err, "模板不存在")
	}
	return item, nil
}

func (s *TemplateService) ListOptions(ctx context.Context) ([]repository.TemplateRecord, error) {
	return s.repo.ListOptions(ctx)
}

func (s *TemplateService) Create(ctx context.Context, req *TemplatePayload) (*repository.TemplateRecord, error) {
	normalizeTemplatePayload(req)
	if err := validateTemplatePayload(req); err != nil {
		return nil, err
	}
	if err := s.ensureTypeExists(ctx, req.TemplateTypeID); err != nil {
		return nil, err
	}
	if err := s.ensureModelMatchesTemplateType(ctx, req.ModelID, req.TemplateType); err != nil {
		return nil, err
	}
	parameters, err := s.parameterService.prepare(ctx, int64(req.ModelID), req.ModelParameters)
	if err != nil {
		return nil, err
	}
	item := &model.VideoTemplate{}
	applyTemplatePayload(item, req)
	if err := repository.Transaction(ctx, func(txCtx context.Context) error {
		if err := s.repo.Create(txCtx, item); err != nil {
			return err
		}
		return s.parameterService.repo.ReplaceForTemplate(txCtx, item.ID, parameters)
	}); err != nil {
		return nil, err
	}
	return s.repo.GetWithType(ctx, item.ID)
}

func (s *TemplateService) Update(ctx context.Context, id uint64, req *TemplatePayload) (*repository.TemplateRecord, error) {
	item, err := s.repo.GetWithType(ctx, id)
	if err != nil {
		return nil, notFoundOr(err, "模板不存在")
	}
	normalizeTemplatePayload(req)
	if err := validateTemplatePayload(req); err != nil {
		return nil, err
	}
	if err := s.ensureTypeExists(ctx, req.TemplateTypeID); err != nil {
		return nil, err
	}
	if err := s.ensureModelMatchesTemplateType(ctx, req.ModelID, req.TemplateType); err != nil {
		return nil, err
	}
	modelChanged := item.ModelID != req.ModelID
	var parameters []*model.VideoTemplateModelParameter
	if modelChanged || req.ModelParameters != nil {
		parameters, err = s.parameterService.prepare(ctx, int64(req.ModelID), req.ModelParameters)
		if err != nil {
			return nil, err
		}
	}
	applyTemplatePayload(&item.VideoTemplate, req)
	if err := repository.Transaction(ctx, func(txCtx context.Context) error {
		if err := s.repo.UpdateFields(txCtx, &item.VideoTemplate); err != nil {
			return err
		}
		if modelChanged || req.ModelParameters != nil {
			return s.parameterService.repo.ReplaceForTemplate(txCtx, item.ID, parameters)
		}
		return nil
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

func (s *TemplateService) ensureModelMatchesTemplateType(ctx context.Context, id uint64, templateType int64) error {
	item, err := s.modelRepo.GetByID(ctx, uint(id))
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return errors.New("关联模型不存在")
	}
	if err != nil {
		return err
	}
	return validateTemplateModelType(templateType, item.ModelType)
}

func validateTemplateModelType(templateType int64, modelType uint32) error {
	if templateType != int64(modelType) {
		return errors.New("模板种类必须与关联模型的模型类型一致")
	}
	return nil
}

func normalizeTemplatePayload(req *TemplatePayload) {
	if req == nil {
		return
	}
	if req.TemplateTypeID == 0 {
		req.TemplateTypeID = req.LegacyTemplateTypeID
	}
	req.Name = strings.TrimSpace(req.Name)
	req.CoverImageURL = strings.TrimSpace(req.CoverImageURL)
	req.OriginalURL = strings.TrimSpace(req.OriginalURL)
	req.ThumbnailURL = strings.TrimSpace(req.ThumbnailURL)
	req.Prompt = strings.TrimSpace(req.Prompt)
	req.Description = strings.TrimSpace(req.Description)
}

func validateTemplatePayload(req *TemplatePayload) error {
	if req == nil || req.TemplateTypeID == 0 || req.ModelID == 0 {
		return errors.New("模板分类和关联模型不能为空")
	}
	if req.Name == "" || (req.TemplateType != domain.VideoTemplateKindImage && req.TemplateType != domain.VideoTemplateKindVideo) || req.CoverImageURL == "" || req.OriginalURL == "" {
		return errors.New("模板名称、模板类型、封面和原始资源地址不能为空")
	}
	return nil
}

func applyTemplatePayload(item *model.VideoTemplate, req *TemplatePayload) {
	item.TemplateTypeID = req.TemplateTypeID
	item.ModelID = req.ModelID
	item.Name = strings.TrimSpace(req.Name)
	item.TemplateType = req.TemplateType
	item.Sort = int64(req.Sort)
	item.CoverImageURL = strings.TrimSpace(req.CoverImageURL)
	item.OriginalURL = strings.TrimSpace(req.OriginalURL)
	item.ThumbnailURL = strings.TrimSpace(req.ThumbnailURL)
	item.Prompt = strings.TrimSpace(req.Prompt)
	item.Status = req.Status
	item.Description = strings.TrimSpace(req.Description)
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

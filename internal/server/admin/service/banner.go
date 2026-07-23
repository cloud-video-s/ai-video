package service

import (
	"context"
	"errors"
	"net/url"
	"sort"
	"strings"

	"ai-video/internal/domain"
	"ai-video/internal/gen/model"
	"ai-video/internal/repository"

	"gorm.io/gorm"
)

type BannerService struct {
	repo         *repository.BannerRepo
	templateRepo *repository.TemplateRepo
	countryRepo  *repository.CountryRepo
	positionRepo *repository.DisplayPositionRepo
}

func NewBannerService() *BannerService {
	return &BannerService{
		repo: repository.NewBannerRepo(), templateRepo: repository.NewTemplateRepo(),
		countryRepo: repository.NewCountryRepo(), positionRepo: repository.NewDisplayPositionRepo(),
	}
}

type ListBannerRequest struct {
	PositionKey string `form:"position_key" binding:"omitempty,max=100"`
	CountryCode string `form:"country_code" binding:"omitempty,max=50"`
	AppCode     string `form:"app_code" binding:"omitempty,max=60"`
	PackageCode string `form:"package_code" binding:"omitempty,max=128"`
	VersionCode string `form:"version_code" binding:"omitempty,max=50"`
	JumpType    uint8  `form:"jump_type" binding:"omitempty,oneof=1 2 3 4"`
	Status      *int8  `form:"status" binding:"omitempty,oneof=0 1"`
	Keyword     string `form:"keyword" binding:"max=255"`
}

type BannerAppTargetPayload struct {
	AppCode      string   `json:"app_code" binding:"required,max=60"`
	PackageCode  string   `json:"package_code" binding:"required,max=128"`
	VersionCodes []string `json:"version_codes" binding:"max=100,dive,required,max=50"`
}

type BannerPayload struct {
	Name                string                   `json:"name" binding:"required,max=128"`
	CoverImage          string                   `json:"cover_image" binding:"required,max=1024"`
	DisplayPositionKeys []string                 `json:"display_position_keys" binding:"max=100,dive,required,max=64"`
	CountryCodes        []string                 `json:"country_codes" binding:"max=100,dive,gt=0"`
	AppTargets          []BannerAppTargetPayload `json:"app_targets" binding:"max=100,dive"`
	Remark              string                   `json:"remark" binding:"max=500"`
	Sort                uint64                   `json:"sort"`
	JumpType            uint8                    `json:"jump_type" binding:"required,oneof=1 2 3 4"`
	JumpURL             string                   `json:"jump_url" binding:"max=1024"`
	TemplateID          *uint64                  `json:"template_id"`
	Status              int8                     `json:"status" binding:"oneof=0 1"`
	SubscriptionStatus  uint8                    `json:"subscription_status" binding:"required,oneof=1 2 3"`
}

type BannerView struct {
	*model.VideoBanner
	AppTargets []repository.BannerAppTarget `json:"app_targets"`
}

func (s *BannerService) List(ctx context.Context, page, pageSize int, req *ListBannerRequest) ([]BannerView, int64, error) {
	items, total, err := s.repo.PageList(ctx, page, pageSize, &repository.BannerListFilter{
		CountryCode: strings.ToUpper(strings.TrimSpace(req.CountryCode)),
		AppCode:     strings.TrimSpace(req.AppCode), PackageCode: strings.TrimSpace(req.PackageCode),
		VersionCode: strings.TrimSpace(req.VersionCode), PositionKey: strings.TrimSpace(req.PositionKey),
		JumpType: req.JumpType, Status: req.Status, Keyword: strings.TrimSpace(req.Keyword),
	})
	if err != nil {
		return nil, 0, err
	}
	result, err := s.withAppTargets(ctx, items)
	return result, total, err
}

func (s *BannerService) GetByID(ctx context.Context, id uint64) (*BannerView, error) {
	item, err := s.repo.GetDetail(ctx, id)
	if err != nil {
		return nil, notFoundOr(err, "Banner 不存在")
	}
	items, err := s.withAppTargets(ctx, []model.VideoBanner{*item})
	if err != nil {
		return nil, err
	}
	return &items[0], nil
}

func (s *BannerService) DeliveryOptions(ctx context.Context) ([]repository.BannerDeliveryApp, error) {
	return s.repo.ListDeliveryOptions(ctx)
}

func (s *BannerService) Create(ctx context.Context, req *BannerPayload) (*BannerView, error) {
	if err := s.prepareAndValidate(ctx, req); err != nil {
		return nil, err
	}
	item := &model.VideoBanner{}
	applyBannerPayload(item, req)
	if err := repository.Transaction(ctx, func(ctx context.Context) error {
		if err := s.repo.Create(ctx, item); err != nil {
			return err
		}
		return s.repo.ReplaceTargets(ctx, item, bannerTargetIDs(req))
	}); err != nil {
		return nil, err
	}
	return s.GetByID(ctx, item.ID)
}

func (s *BannerService) Update(ctx context.Context, id uint64, req *BannerPayload) (*BannerView, error) {
	item, err := s.repo.GetDetail(ctx, id)
	if err != nil {
		return nil, notFoundOr(err, "Banner 不存在")
	}
	if err := s.prepareAndValidate(ctx, req); err != nil {
		return nil, err
	}
	applyBannerPayload(item, req)
	if err := repository.Transaction(ctx, func(ctx context.Context) error {
		if err := s.repo.UpdateFields(ctx, item); err != nil {
			return err
		}
		return s.repo.ReplaceTargets(ctx, item, bannerTargetIDs(req))
	}); err != nil {
		return nil, err
	}
	return s.GetByID(ctx, item.ID)
}

func (s *BannerService) Delete(ctx context.Context, id uint64) error {
	if _, err := s.repo.GetDetail(ctx, id); err != nil {
		return notFoundOr(err, "Banner 不存在")
	}
	return s.repo.DeleteWithTargets(ctx, id)
}

func (s *BannerService) prepareAndValidate(ctx context.Context, req *BannerPayload) error {
	var err error
	if req.DisplayPositionKeys, err = normalizeBannerPositionKeys(req.DisplayPositionKeys); err != nil {
		return err
	}
	for _, key := range req.DisplayPositionKeys {
		position, lookupErr := s.positionRepo.GetByKey(ctx, key)
		if lookupErr != nil {
			return notFoundOr(lookupErr, "展示位置不存在")
		}
		if position.Status != 1 {
			return errors.New("所选展示位置中包含已禁用项")
		}
	}
	if req.CountryCodes, err = normalizeTargetIDs(req.CountryCodes, "国家"); err != nil {
		return err
	}
	for _, code := range req.CountryCodes {
		country, lookupErr := s.countryRepo.GetEnabledByCode(ctx, code)
		if lookupErr != nil {
			return notFoundOr(lookupErr, "国家不存在")
		}
		if country.Status != 1 {
			return errors.New("所选国家中包含已禁用项")
		}
	}
	req.AppTargets, err = normalizeBannerAppTargets(req.AppTargets)
	if err != nil {
		return err
	}
	for _, target := range req.AppTargets {
		if err := s.repo.ValidateAppTarget(ctx, repository.BannerAppTargetInput{
			AppCode: target.AppCode, PackageCode: target.PackageCode, VersionCodes: target.VersionCodes,
		}); err != nil {
			return err
		}
	}
	if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.CoverImage) == "" {
		return errors.New("Banner 名称和封面图不能为空")
	}
	return s.validateJump(ctx, req)
}

func normalizeBannerPositionKeys(values []string) ([]string, error) {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if len(value) > 64 || !displayPositionKeyPattern.MatchString(value) {
			return nil, errors.New("展示位置标识格式有误")
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result, nil
}

func normalizeBannerAppTargets(values []BannerAppTargetPayload) ([]BannerAppTargetPayload, error) {
	type targetKey struct{ appCode, packageCode string }
	result := make([]BannerAppTargetPayload, 0, len(values))
	indexes := make(map[targetKey]int, len(values))
	allVersions := make(map[targetKey]bool, len(values))
	for _, value := range values {
		value.AppCode = strings.TrimSpace(value.AppCode)
		value.PackageCode = strings.TrimSpace(value.PackageCode)
		if value.AppCode == "" || value.PackageCode == "" {
			return nil, errors.New("应用和包不能为空")
		}
		value.VersionCodes = normalizeBannerVersionCodes(value.VersionCodes)
		key := targetKey{appCode: value.AppCode, packageCode: value.PackageCode}
		index, exists := indexes[key]
		if !exists {
			indexes[key] = len(result)
			allVersions[key] = len(value.VersionCodes) == 0
			result = append(result, value)
			continue
		}
		if allVersions[key] || len(value.VersionCodes) == 0 {
			result[index].VersionCodes = []string{}
			allVersions[key] = true
			continue
		}
		result[index].VersionCodes = normalizeBannerVersionCodes(append(result[index].VersionCodes, value.VersionCodes...))
	}
	return result, nil
}

func normalizeBannerVersionCodes(values []string) []string {
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
	sort.Strings(result)
	return result
}

func (s *BannerService) validateJump(ctx context.Context, req *BannerPayload) error {
	req.JumpURL = strings.TrimSpace(req.JumpURL)
	switch req.JumpType {
	case domain.BannerJumpTypeLink:
		if !validBannerLink(req.JumpURL) {
			return errors.New("链接跳转必须提供有效的绝对链接、深链或站内路径")
		}
		req.TemplateID = nil
	case domain.BannerJumpTypeTemplate:
		if req.TemplateID == nil || *req.TemplateID == 0 {
			return errors.New("模板跳转必须选择目标模板")
		}
		if _, err := s.templateRepo.GetWithType(ctx, *req.TemplateID); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("目标模板不存在")
			}
			return err
		}
		req.JumpURL = ""
	case domain.BannerJumpTypeTextToImage, domain.BannerJumpTypeTextToVideo:
		req.JumpURL = ""
		req.TemplateID = nil
	default:
		return errors.New("不支持的 Banner 跳转方式")
	}
	return nil
}

func validBannerLink(value string) bool {
	if strings.HasPrefix(value, "/") && !strings.HasPrefix(value, "//") {
		return true
	}
	parsed, err := url.Parse(value)
	return err == nil && parsed.IsAbs() && parsed.Scheme != "" && parsed.Host != ""
}

func applyBannerPayload(item *model.VideoBanner, req *BannerPayload) {
	item.Name = strings.TrimSpace(req.Name)
	item.CoverImage = strings.TrimSpace(req.CoverImage)
	item.Remark = strings.TrimSpace(req.Remark)
	item.Sort = req.Sort
	item.JumpType = req.JumpType
	item.JumpURL = req.JumpURL
	item.TemplateID = req.TemplateID
	item.Status = req.Status
	item.SubscriptionStatus = req.SubscriptionStatus
}

func bannerTargetIDs(req *BannerPayload) repository.BannerTargetIDs {
	return repository.BannerTargetIDs{
		DisplayPositionKeys: append([]string(nil), req.DisplayPositionKeys...),
		CountryCodes:        req.CountryCodes, AppTargets: bannerAppTargetInputs(req.AppTargets),
	}
}

func bannerAppTargetInputs(values []BannerAppTargetPayload) []repository.BannerAppTargetInput {
	result := make([]repository.BannerAppTargetInput, len(values))
	for i, value := range values {
		result[i] = repository.BannerAppTargetInput{
			AppCode: value.AppCode, PackageCode: value.PackageCode,
			VersionCodes: append([]string(nil), value.VersionCodes...),
		}
	}
	return result
}

func (s *BannerService) withAppTargets(ctx context.Context, items []model.VideoBanner) ([]BannerView, error) {
	ids := make([]uint64, len(items))
	for i := range items {
		ids[i] = items[i].ID
	}
	targets, err := s.repo.LoadAppTargets(ctx, ids)
	if err != nil {
		return nil, err
	}
	result := make([]BannerView, len(items))
	for i := range items {
		result[i] = BannerView{VideoBanner: &items[i], AppTargets: targets[items[i].ID]}
		if result[i].AppTargets == nil {
			result[i].AppTargets = []repository.BannerAppTarget{}
		}
	}
	return result, nil
}

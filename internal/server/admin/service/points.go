package service

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"time"

	"ai-video/internal/gen/model"
	"ai-video/internal/repository"

	"gorm.io/gorm"
)

var (
	pointsProductIDPattern = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)
	resourceTypePattern    = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,31}$`)
	currencyPattern        = regexp.MustCompile(`^[A-Z]{3}$`)
)

type PointsService struct {
	repo *repository.PointsRepo
}

func NewPointsService() *PointsService {
	return &PointsService{repo: repository.NewPointsRepo()}
}

type ListPointsRequest struct {
	ListSortRequest
	AppCode      string `form:"app_code" binding:"omitempty,max=60"`
	PackageCode  string `form:"package_code" binding:"omitempty,max=128"`
	VersionCode  string `form:"version_code" binding:"omitempty,max=50"`
	CountryCode  string `form:"country_code" binding:"omitempty,max=2"`
	ChannelCode  string `form:"channel_code" binding:"omitempty,max=64"`
	System       string `form:"system" binding:"max=32"`
	UserType     int    `form:"user_type" binding:"omitempty,oneof=1 2"`
	ResourceType string `form:"resource_type" binding:"max=32"`
	Status       *int8  `form:"status" binding:"omitempty,oneof=0 1"`
	Keyword      string `form:"keyword" binding:"max=255"`
}

type PointsPayload struct {
	AppCodes      []string `json:"app_codes" binding:"max=100,dive,gt=0"`
	PackageCodes  []string `json:"package_codes" binding:"required,min=1,max=100,dive,gt=0"`
	VersionCodes  []string `json:"version_codes" binding:"max=100,dive,gt=0"`
	CountryCodes  []string `json:"country_codes" binding:"max=100,dive,gt=0"`
	ChannelCodes  []string `json:"channel_codes" binding:"max=100,dive,gt=0"`
	ProductCode   string   `json:"product_code" binding:"required,max=191"`
	Name          string   `json:"name" binding:"required,max=128"`
	Systems       []string `json:"systems" binding:"required,min=1,max=10,dive,required,max=32"`
	UserTypes     []int    `json:"user_types" binding:"required,min=1,max=2,dive,oneof=1 2"`
	ResourceType  string   `json:"resource_type" binding:"required,max=32"`
	Points        uint64   `json:"points" binding:"required"`
	Currency      string   `json:"currency" binding:"required,len=3"`
	SalePrice     float64  `json:"sale_price" binding:"gte=0"`
	ActualRevenue float64  `json:"actual_revenue" binding:"gte=0"`
	OriginalPrice float64  `json:"original_price" binding:"gte=0"`
	Icon          string   `json:"icon" binding:"max=1024"`
	Description   string   `json:"description" binding:"max=1000"`
	ButtonText    string   `json:"button_text" binding:"max=128"`
	IsDefault     bool     `json:"is_default"`
	Status        int8     `json:"status" binding:"oneof=0 1"`
	Sort          int64    `json:"sort" binding:"gte=0"`
}

type PointsResponse struct {
	PointsPayload
	ID             uint64                       `json:"id"`
	Apps           []*model.VideoApp            `json:"apps"`
	Packages       []*model.VideoPackage        `json:"packages"`
	PackageVersion []*model.VideoPackageVersion `json:"package_version"`
	Country        []*model.VideoCountry        `json:"country"`
	Channels       []*model.VideoChannel        `json:"channels"`
	CreatedAt      time.Time                    `json:"created_at"`
	UpdatedAt      time.Time                    `json:"updated_at"`
}

type PointsStatusPayload struct {
	Status int8 `json:"status" binding:"oneof=0 1"`
}

func (s *PointsService) List(ctx context.Context, page, pageSize int, req *ListPointsRequest) ([]PointsResponse, int64, error) {
	items, total, err := s.repo.PageList(ctx, page, pageSize, &repository.PointsListFilter{
		ListSort: req.listSort(), AppCode: strings.TrimSpace(req.AppCode),
		PackageCode: strings.TrimSpace(req.PackageCode), VersionCode: strings.TrimSpace(req.VersionCode),
		CountryCode: strings.ToUpper(strings.TrimSpace(req.CountryCode)), ChannelCode: strings.TrimSpace(req.ChannelCode),
		System: strings.ToLower(strings.TrimSpace(req.System)), UserType: req.UserType,
		ResourceType: strings.ToLower(strings.TrimSpace(req.ResourceType)), Status: req.Status,
		Keyword: strings.TrimSpace(req.Keyword),
	})
	if err != nil {
		return nil, 0, err
	}
	return pointsResponses(items), total, nil
}

func (s *PointsService) ListOptions(ctx context.Context) ([]PointsResponse, error) {
	items, err := s.repo.ListOptions(ctx)
	if err != nil {
		return nil, err
	}
	return pointsResponses(items), nil
}

func (s *PointsService) GetByID(ctx context.Context, id uint64) (*PointsResponse, error) {
	item, err := s.repo.GetDetail(ctx, id)
	if err != nil {
		return nil, notFoundOr(err, "积分套餐不存在")
	}
	return pointsResponse(item), nil
}

func (s *PointsService) Create(ctx context.Context, req *PointsPayload) (*PointsResponse, error) {
	if err := s.prepareAndValidate(ctx, req, 0); err != nil {
		return nil, err
	}
	item := &model.VideoPoint{}
	applyPointsPayload(item, req)
	err := repository.Transaction(ctx, func(txCtx context.Context) error {
		if err := s.repo.Create(txCtx, item); err != nil {
			return err
		}
		if err := s.repo.ReplaceTargets(txCtx, item, pointsTargets(req)); err != nil {
			return err
		}
		if item.IsDefault == 1 {
			return s.clearDefaults(txCtx, req.PackageCodes, item.ResourceType, item.ID)
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, errors.New("产品 ID 已存在，每个积分套餐必须唯一")
		}
		return nil, err
	}
	created, err := s.repo.GetDetail(ctx, item.ID)
	if err != nil {
		return nil, err
	}
	return pointsResponse(created), nil
}

func (s *PointsService) Update(ctx context.Context, id uint64, req *PointsPayload) (*PointsResponse, error) {
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
	applyPointsPayload(item, req)
	err = repository.Transaction(ctx, func(txCtx context.Context) error {
		if err := s.repo.UpdateFields(txCtx, item); err != nil {
			return err
		}
		if err := s.repo.ReplaceTargets(txCtx, item, pointsTargets(req)); err != nil {
			return err
		}
		if item.IsDefault == 1 {
			return s.clearDefaults(txCtx, req.PackageCodes, item.ResourceType, item.ID)
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, errors.New("产品 ID 已存在，每个积分套餐必须唯一")
		}
		return nil, err
	}
	updated, err := s.repo.GetDetail(ctx, item.ID)
	if err != nil {
		return nil, err
	}
	return pointsResponse(updated), nil
}

func (s *PointsService) clearDefaults(ctx context.Context, packageCodes []string, resourceType string, exceptID uint64) error {
	for _, packageCode := range packageCodes {
		if err := s.repo.ClearDefaults(ctx, packageCode, resourceType, exceptID); err != nil {
			return err
		}
	}
	return nil
}

func (s *PointsService) Delete(ctx context.Context, id uint64) error {
	if _, err := s.repo.GetDetail(ctx, id); err != nil {
		return notFoundOr(err, "积分套餐不存在")
	}
	return s.repo.DeleteWithTargets(ctx, id)
}

func (s *PointsService) UpdateStatus(ctx context.Context, id uint64, status int8) error {
	if _, err := s.repo.GetDetail(ctx, id); err != nil {
		return notFoundOr(err, "积分套餐不存在")
	}
	return s.repo.UpdateStatus(ctx, id, status)
}

func (s *PointsService) SetDefault(ctx context.Context, id uint64) error {
	item, err := s.repo.GetDetail(ctx, id)
	if err != nil {
		return notFoundOr(err, "积分套餐不存在")
	}
	return s.repo.SetDefault(ctx, item)
}

func (s *PointsService) prepareAndValidate(ctx context.Context, req *PointsPayload, currentID uint64) error {
	var err error
	if req.AppCodes, err = normalizePointsTargetCodes(req.AppCodes, "APP", false); err != nil {
		return err
	}
	if req.PackageCodes, err = normalizePointsTargetCodes(req.PackageCodes, "安装包", false); err != nil {
		return err
	}
	if len(req.PackageCodes) == 0 {
		return errors.New("请至少选择一个安装包")
	}
	if req.VersionCodes, err = normalizePointsTargetCodes(req.VersionCodes, "版本", false); err != nil {
		return err
	}
	if req.CountryCodes, err = normalizePointsTargetCodes(req.CountryCodes, "国家", true); err != nil {
		return err
	}
	if req.ChannelCodes, err = normalizePointsTargetCodes(req.ChannelCodes, "渠道", false); err != nil {
		return err
	}
	req.ProductCode = strings.TrimSpace(req.ProductCode)
	req.Name = strings.TrimSpace(req.Name)
	req.ResourceType = strings.ToLower(strings.TrimSpace(req.ResourceType))
	req.Currency = strings.ToUpper(strings.TrimSpace(req.Currency))
	if !pointsProductIDPattern.MatchString(req.ProductCode) {
		return errors.New("产品 ID 只能包含字母、数字、点、下划线和中划线")
	}
	if req.Name == "" {
		return errors.New("积分名称不能为空")
	}
	if !resourceTypePattern.MatchString(req.ResourceType) {
		return errors.New("资源类型只能包含小写字母、数字、下划线和中划线")
	}
	if !currencyPattern.MatchString(req.Currency) {
		return errors.New("币种必须是三位大写字母代码")
	}
	if req.Points == 0 {
		return errors.New("赠送积分必须大于 0")
	}
	if err := validatePointsMoney(req); err != nil {
		return err
	}
	if req.Systems, err = normalizeSystemTypes(req.Systems); err != nil {
		return err
	}
	if req.UserTypes, err = normalizeUserTypes(req.UserTypes); err != nil {
		return err
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

func validatePointsMoney(req *PointsPayload) error {
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

func applyPointsPayload(item *model.VideoPoint, req *PointsPayload) {
	item.ProductCode = req.ProductCode
	item.Name = req.Name
	systems, _ := json.Marshal(req.Systems)
	userTypes, _ := json.Marshal(req.UserTypes)
	item.Systems = string(systems)
	item.UserTypes = string(userTypes)
	item.ResourceType = req.ResourceType
	item.Points = req.Points
	item.Currency = req.Currency
	item.SalePrice = req.SalePrice
	item.ActualRevenue = req.ActualRevenue
	item.OriginalPrice = req.OriginalPrice
	item.Icon = strings.TrimSpace(req.Icon)
	item.Description = strings.TrimSpace(req.Description)
	item.ButtonText = strings.TrimSpace(req.ButtonText)
	item.IsDefault = boolToInt8(req.IsDefault)
	item.Status = req.Status
	item.Sort = req.Sort
}

func pointsTargets(req *PointsPayload) repository.PointsTargets {
	return repository.PointsTargets{
		AppCodes: req.AppCodes, PackageCodes: req.PackageCodes, VersionCodes: req.VersionCodes,
		CountryCodes: req.CountryCodes, ChannelCodes: req.ChannelCodes,
	}
}

func pointsPayloadFromModel(item *model.VideoPoint) *PointsPayload {
	if item == nil {
		return nil
	}
	systems := make([]string, 0)
	_ = json.Unmarshal([]byte(item.Systems), &systems)
	userTypes := make([]int, 0)
	_ = json.Unmarshal([]byte(item.UserTypes), &userTypes)
	return &PointsPayload{
		AppCodes: pointsAppCodes(item.Apps), PackageCodes: pointsPackageCodes(item.Packages),
		VersionCodes: pointsVersionCodes(item.PackageVersion), CountryCodes: pointsCountryCodes(item.Country),
		ChannelCodes: pointsChannelCodes(item.Channels), ProductCode: item.ProductCode, Name: item.Name,
		Systems: systems, UserTypes: userTypes, ResourceType: item.ResourceType, Points: item.Points,
		Currency: item.Currency, SalePrice: item.SalePrice, ActualRevenue: item.ActualRevenue,
		OriginalPrice: item.OriginalPrice, Icon: item.Icon, Description: item.Description,
		ButtonText: item.ButtonText, IsDefault: item.IsDefault == 1, Status: item.Status, Sort: item.Sort,
	}
}

func pointsResponse(item *model.VideoPoint) *PointsResponse {
	if item == nil {
		return nil
	}
	return &PointsResponse{
		PointsPayload: *pointsPayloadFromModel(item), ID: item.ID, Apps: item.Apps, Packages: item.Packages,
		PackageVersion: item.PackageVersion, Country: item.Country, Channels: item.Channels,
		CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt,
	}
}

func pointsResponses(items []model.VideoPoint) []PointsResponse {
	result := make([]PointsResponse, 0, len(items))
	for i := range items {
		result = append(result, *pointsResponse(&items[i]))
	}
	return result
}

func normalizePointsTargetCodes(values []string, label string, uppercase bool) ([]string, error) {
	normalized := make([]string, len(values))
	for i, value := range values {
		value = strings.TrimSpace(value)
		if uppercase {
			value = strings.ToUpper(value)
		}
		normalized[i] = value
	}
	return normalizeTargetIDs(normalized, label)
}

func pointsAppCodes(items []*model.VideoApp) []string {
	result := make([]string, 0, len(items))
	for _, item := range items {
		if item != nil {
			result = append(result, item.AppCode)
		}
	}
	return result
}

func pointsPackageCodes(items []*model.VideoPackage) []string {
	result := make([]string, 0, len(items))
	for _, item := range items {
		if item != nil {
			result = append(result, item.PackageCode)
		}
	}
	return result
}

func pointsVersionCodes(items []*model.VideoPackageVersion) []string {
	result := make([]string, 0, len(items))
	for _, item := range items {
		if item != nil {
			result = append(result, item.VersionCode)
		}
	}
	return result
}

func pointsCountryCodes(items []*model.VideoCountry) []string {
	result := make([]string, 0, len(items))
	for _, item := range items {
		if item != nil {
			result = append(result, item.Code)
		}
	}
	return result
}

func pointsChannelCodes(items []*model.VideoChannel) []string {
	result := make([]string, 0, len(items))
	for _, item := range items {
		if item != nil {
			result = append(result, item.ChannelCode)
		}
	}
	return result
}

func normalizeSystemTypes(values []string) ([]string, error) {
	allowed := map[string]struct{}{
		"android": {}, "ios": {}, "pc": {}, "harmony": {}, "web": {}, "other": {},
	}
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if _, ok := allowed[value]; !ok {
			return nil, errors.New("系统类型仅支持 android、ios、pc、harmony、web 或 other")
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	if len(result) == 0 {
		return nil, errors.New("至少选择一个系统类型")
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

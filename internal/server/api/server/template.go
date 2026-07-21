package service

import (
	"errors"
	"sort"
	"strings"

	"ai-video/internal/domain"
	"ai-video/internal/gen/model"
	"ai-video/internal/repository"

	"github.com/gin-gonic/gin"
)

type ClientTemplateService struct {
	typeRepo     *repository.TemplateTypeRepo
	templateRepo *repository.TemplateRepo
	displayRepo  *repository.TemplateDisplayConfigRepo
	userRepo     *repository.AppUserRepo
	countryRepo  *repository.CountryRepo
	channelRepo  *repository.ChannelRepo
	packageRepo  *repository.PackageRepo
}

func NewClientTemplateService() *ClientTemplateService {
	return &ClientTemplateService{
		typeRepo: repository.NewTemplateTypeRepo(), templateRepo: repository.NewTemplateRepo(),
		displayRepo: repository.NewTemplateDisplayConfigRepo(),
		userRepo:    repository.NewAppUserRepo(), countryRepo: repository.NewCountryRepo(),
		channelRepo: repository.NewChannelRepo(), packageRepo: repository.NewPackageRepo(),
	}
}

// ClientTemplateRequest accepts explicit delivery targets for diagnostics and
// API clients. UserType and SubscriptionStatus may only repeat the authenticated
// user's current values; they cannot be used to elevate template visibility.
type ClientTemplateRequest struct {
	PositionKey        string `form:"position_key" binding:"required,max=64"`
	Country            string `form:"country" binding:"omitempty,max=64"`
	PackageCode        string `form:"package" binding:"omitempty,max=255"`
	PackageVersion     string `form:"package_version" binding:"omitempty,max=64"`
	Channel            string `form:"channel" binding:"omitempty,max=64"`
	UserType           uint32 `form:"user_type" binding:"omitempty,oneof=1 2"`
	SubscriptionStatus uint32 `form:"subscription_status" binding:"omitempty,oneof=1 2 3"`
	AccountBaseRequest
}

type ClientTemplateRecommendRequest struct {
	PositionKey string `form:"position_key" binding:"required,max=64"`
	AccountBaseRequest
}

type ClientTemplateDisplayRequest struct {
	PositionKey string `form:"position_key" binding:"required,max=64"`
	AccountBaseRequest
}

var ErrClientTemplateAudienceMismatch = errors.New("模板受众条件与当前登录用户不一致")

type ClientTemplateType struct {
	ID           uint64           `json:"id"`
	CategoryName string           `json:"category_name"`
	Description  string           `json:"description"`
	Sort         int64            `json:"sort"`
	Templates    []ClientTemplate `json:"templates"`
}

type ClientTemplate struct {
	ID                   uint64   `json:"id"`
	VideoTemplateTypeID  uint64   `json:"video_template_type_id"`
	Name                 string   `json:"name"`
	TemplateType         string   `json:"template_type"`
	CoverImage           string   `json:"cover_image"`
	TemplateVideo        string   `json:"template_video"`
	ThumbnailVideo       string   `json:"thumbnail_video"`
	Prompt               string   `json:"prompt"`
	Description          string   `json:"description"`
	UserTypes            []int    `json:"user_types"`
	SubscriptionStatuses []string `json:"subscription_statuses"`
	Sort                 int      `json:"sort"`
	UsageCount           uint64   `json:"usage_count"`
	FavoriteCount        uint64   `json:"favorite_count"`
	ViewCount            uint64   `json:"view_count"`
}

type ClientTemplateDisplayItem struct {
	ClientTemplate
	DisplayConfigID uint64 `json:"display_config_id"`
	PositionKey     string `json:"position_key"`
	DisplaySort     int    `json:"display_sort"`
}

// ListByPosition returns templates explicitly curated for a display position.
// Template, category, position and configuration status must all be enabled;
// the existing country, package, channel and user audience rules still apply.
func (s *ClientTemplateService) ListByPosition(ctx *gin.Context, userID uint64, req *ClientTemplateDisplayRequest) ([]ClientTemplateDisplayItem, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	GetCtxAccountBaseRequest(ctx, &req.AccountBaseRequest)

	countryCode := strings.ToUpper(strings.TrimSpace(req.DeviceCountry))
	if countryCode == "" {
		countryCode = strings.ToUpper(strings.TrimSpace(user.DeviceCountry))
	}
	var countryID uint64
	if countryCode != "" {
		if country, lookupErr := s.countryRepo.GetEnabledByCode(ctx, countryCode); lookupErr == nil {
			countryID = country.ID
		}
	}

	channelValue := strings.TrimSpace(req.ChannelID)
	if channelValue == "" {
		channelValue = strings.TrimSpace(user.ChannelID)
	}
	channels, err := s.channelRepo.ResolveEnabledTargets(ctx, channelValue, strings.TrimSpace(req.ChannelPackage))
	if err != nil {
		return nil, err
	}
	packages, err := s.packageRepo.ResolveEnabledTargets(ctx, strings.TrimSpace(req.AppPackage), strings.TrimSpace(req.AppVersion))
	if err != nil {
		return nil, err
	}

	subscriptionState := "unsubscribed"
	if user.SubscriptionStatus == domain.AppUserSubscriptionSubscribed {
		subscriptionState = "subscribed"
	}
	rows, err := s.displayRepo.ListForClient(ctx, repository.ClientTemplateDisplayTargets{
		PositionKey: strings.TrimSpace(req.PositionKey), CountryID: countryID,
		ChannelIDs: clientChannelIDs(channels), PackageIDs: clientPackageIDs(packages),
		UserType: user.UserType, SubscriptionState: subscriptionState,
	})
	if err != nil {
		return nil, err
	}
	result := make([]ClientTemplateDisplayItem, 0, len(rows))
	for i := range rows {
		result = append(result, ClientTemplateDisplayItem{
			ClientTemplate: mapClientTemplate(&rows[i].Template), DisplayConfigID: rows[i].ID,
			PositionKey: rows[i].DisplayPositionKey, DisplaySort: int(rows[i].Sort),
		})
	}
	return result, nil
}

func (s *ClientTemplateService) List(ctx *gin.Context, userID uint64, req *ClientTemplateRequest) ([]ClientTemplateType, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	GetCtxAccountBaseRequest(ctx, &req.AccountBaseRequest)
	countryCode := strings.ToUpper(strings.TrimSpace(req.Country))
	if countryCode == "" {
		countryCode = strings.ToUpper(strings.TrimSpace(req.DeviceCountry))
	}
	var countryID uint64
	if countryCode != "" {
		if country, lookupErr := s.countryRepo.GetEnabledByCode(ctx, countryCode); lookupErr == nil {
			countryID = country.ID
		}
	}

	channelValue := strings.TrimSpace(req.ChannelID)
	if channelValue == "" {
		channelValue = strings.TrimSpace(user.ChannelID)
	}
	channels, err := s.channelRepo.ResolveEnabledTargets(ctx, channelValue, strings.TrimSpace(req.ChannelPackage))
	if err != nil {
		return nil, err
	}

	packageCode := strings.TrimSpace(req.AppPackage)

	packageVersion := strings.TrimSpace(req.AppVersion)
	packages, err := s.packageRepo.ResolveEnabledTargets(ctx, packageCode, packageVersion)
	if err != nil {
		return nil, err
	}

	subscriptionState := "unsubscribed"
	if user.SubscriptionStatus == domain.AppUserSubscriptionSubscribed {
		subscriptionState = "subscribed"
	}
	types, err := s.typeRepo.ListForClient(ctx, repository.ClientTemplateTypeTargets{
		PositionKey: strings.TrimSpace(req.PositionKey), CountryID: countryID,
		ChannelIDs: clientChannelIDs(channels), PackageIDs: clientPackageIDs(packages),
		UserType: user.UserType, SubscriptionState: subscriptionState,
	})
	if err != nil {
		return nil, err
	}
	rows, err := s.templateRepo.ListForClient(ctx, repository.ClientTemplateTargets{
		TemplateTypeIDs: templateTypeIDs(types), CountryID: countryID,
		ChannelIDs: clientChannelIDs(channels), PackageIDs: clientPackageIDs(packages),
		UserType: user.UserType, SubscriptionStatus: subscriptionState,
	})
	if err != nil {
		return nil, err
	}
	return buildClientTemplateGroups(types, rows), nil
}

func (s *ClientTemplateService) Categories(ctx *gin.Context, userID uint64, req *ClientTemplateRequest) ([]ClientTemplateType, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	GetCtxAccountBaseRequest(ctx, &req.AccountBaseRequest)
	countryCode := strings.ToUpper(strings.TrimSpace(req.Country))
	if countryCode == "" {
		countryCode = strings.ToUpper(strings.TrimSpace(req.DeviceCountry))
	}
	var countryID uint64
	if countryCode != "" {
		if country, lookupErr := s.countryRepo.GetEnabledByCode(ctx, countryCode); lookupErr == nil {
			countryID = country.ID
		}
	}

	channelValue := strings.TrimSpace(req.ChannelID)
	if channelValue == "" {
		channelValue = strings.TrimSpace(user.ChannelID)
	}
	channels, err := s.channelRepo.ResolveEnabledTargets(ctx, channelValue, strings.TrimSpace(req.ChannelPackage))
	if err != nil {
		return nil, err
	}

	packageCode := strings.TrimSpace(req.AppPackage)

	packageVersion := strings.TrimSpace(req.AppVersion)
	packages, err := s.packageRepo.ResolveEnabledTargets(ctx, packageCode, packageVersion)
	if err != nil {
		return nil, err
	}

	subscriptionState := "unsubscribed"
	if user.SubscriptionStatus == domain.AppUserSubscriptionSubscribed {
		subscriptionState = "subscribed"
	}
	types, err := s.typeRepo.ListForClient(ctx, repository.ClientTemplateTypeTargets{
		PositionKey: strings.TrimSpace(req.PositionKey), CountryID: countryID,
		ChannelIDs: clientChannelIDs(channels), PackageIDs: clientPackageIDs(packages),
		UserType: user.UserType, SubscriptionState: subscriptionState,
	})
	if err != nil {
		return nil, err
	}
	rows, err := s.templateRepo.ListForClient(ctx, repository.ClientTemplateTargets{
		TemplateTypeIDs: templateTypeIDs(types), CountryID: countryID,
		ChannelIDs: clientChannelIDs(channels), PackageIDs: clientPackageIDs(packages),
		UserType: user.UserType, SubscriptionStatus: subscriptionState,
	})
	if err != nil {
		return nil, err
	}
	return buildClientTemplateGroups(types, rows), nil
}

func buildClientTemplateGroups(types []model.VideoTemplateType, rows []model.VideoTemplate) []ClientTemplateType {
	types = append([]model.VideoTemplateType(nil), types...)
	rows = append([]model.VideoTemplate(nil), rows...)
	sort.SliceStable(types, func(i, j int) bool {
		if types[i].Sort != types[j].Sort {
			return types[i].Sort > types[j].Sort
		}
		return types[i].ID > types[j].ID
	})
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].Sort != rows[j].Sort {
			return rows[i].Sort > rows[j].Sort
		}
		if rows[i].UsageCount != rows[j].UsageCount {
			return rows[i].UsageCount > rows[j].UsageCount
		}
		if rows[i].FavoriteCount != rows[j].FavoriteCount {
			return rows[i].FavoriteCount > rows[j].FavoriteCount
		}
		if rows[i].ViewCount != rows[j].ViewCount {
			return rows[i].ViewCount > rows[j].ViewCount
		}
		return rows[i].ID > rows[j].ID
	})
	templatesByType := make(map[uint64][]ClientTemplate, len(types))
	for i := range rows {
		item := rows[i]
		templatesByType[item.VideoTemplateTypeID] = append(templatesByType[item.VideoTemplateTypeID], mapClientTemplate(&item))
	}
	result := make([]ClientTemplateType, 0, len(types))
	for i := range types {
		item := types[i]
		templates := templatesByType[item.ID]
		if len(templates) == 0 {
			continue
		}
		result = append(result, ClientTemplateType{
			ID: item.ID, CategoryName: item.CategoryName, Description: item.Description,
			Sort:      item.Sort,
			Templates: templates,
		})
	}
	return result
}

func mapClientTemplate(item *model.VideoTemplate) ClientTemplate {
	return ClientTemplate{
		ID:                   item.ID,
		VideoTemplateTypeID:  item.VideoTemplateTypeID,
		Name:                 item.Name,
		TemplateType:         item.TemplateType,
		CoverImage:           item.CoverImage,
		TemplateVideo:        item.TemplateVideo,
		ThumbnailVideo:       item.ThumbnailVideo,
		Prompt:               item.Prompt,
		Description:          item.Description,
		UserTypes:            item.UserTypes,
		SubscriptionStatuses: item.SubscriptionStatuses,
		Sort:                 item.Sort,
		UsageCount:           item.UsageCount,
		FavoriteCount:        item.FavoriteCount,
		ViewCount:            item.ViewCount,
	}
}

func templateTypeIDs(items []model.VideoTemplateType) []uint64 {
	result := make([]uint64, len(items))
	for i := range items {
		result[i] = items[i].ID
	}
	return result
}

func templatePositionKeys(items []model.VideoDisplayPosition) []string {
	result := make([]string, 0, len(items))
	for i := range items {
		result = append(result, items[i].PositionKey)
	}
	return result
}

func (s *ClientTemplateService) Recommend(ctx *gin.Context, userID uint64, req *ClientTemplateRecommendRequest) ([]ClientTemplate, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	GetCtxAccountBaseRequest(ctx, &req.AccountBaseRequest)
	countryCode := strings.ToUpper(strings.TrimSpace(req.DeviceCountry))
	if countryCode == "" {
		countryCode = strings.ToUpper(strings.TrimSpace(user.DeviceCountry))
	}
	var countryID uint64
	if countryCode != "" {
		if country, lookupErr := s.countryRepo.GetEnabledByCode(ctx, countryCode); lookupErr == nil {
			countryID = country.ID
		}
	}

	channelValue := strings.TrimSpace(req.ChannelID)
	if channelValue == "" {
		channelValue = strings.TrimSpace(user.ChannelID)
	}
	channels, err := s.channelRepo.ResolveEnabledTargets(ctx, channelValue, strings.TrimSpace(req.ChannelPackage))
	if err != nil {
		return nil, err
	}
	packageCode := strings.TrimSpace(req.AppPackage)
	packageVersion := strings.TrimSpace(req.AppVersion)
	packages, err := s.packageRepo.ResolveEnabledTargets(ctx, packageCode, packageVersion)
	if err != nil {
		return nil, err
	}

	subscriptionState := "unsubscribed"
	if user.SubscriptionStatus == domain.AppUserSubscriptionSubscribed {
		subscriptionState = "subscribed"
	}
	rows, err := s.templateRepo.ListForClient(ctx, repository.ClientTemplateTargets{
		CountryID:  countryID,
		ChannelIDs: clientChannelIDs(channels), PackageIDs: clientPackageIDs(packages),
		UserType: user.UserType, SubscriptionStatus: subscriptionState,
	})
	if err != nil {
		return nil, err
	}
	var result []ClientTemplate
	for i := range rows {
		result = append(result, mapClientTemplate(&rows[i]))
	}
	return result, nil
}

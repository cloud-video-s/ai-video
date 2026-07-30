package service

import (
	"ai-video/internal/middleware"
	"errors"
	"fmt"
	"sort"
	"strings"

	"ai-video/internal/gen/model"
	"ai-video/internal/repository"

	"github.com/gin-gonic/gin"
)

type ClientTemplateService struct {
	typeRepo              *repository.TemplateTypeRepo
	templateRepo          *repository.TemplateRepo
	displayRepo           *repository.TemplateDisplayConfigRepo
	userRepo              *repository.AppUserRepo
	countryRepo           *repository.CountryRepo
	TemplateFavoriteRepo  *repository.TemplateFavoriteRepo
	modelRepo             *repository.ModelRepo
	templateParameterRepo *repository.TemplateModelParameterRepo
}

func NewClientTemplateService() *ClientTemplateService {
	return &ClientTemplateService{
		typeRepo: repository.NewTemplateTypeRepo(), templateRepo: repository.NewTemplateRepo(),
		displayRepo: repository.NewTemplateDisplayConfigRepo(),
		userRepo:    repository.NewAppUserRepo(), countryRepo: repository.NewCountryRepo(),
		modelRepo: repository.NewModelRepo(), templateParameterRepo: repository.NewTemplateModelParameterRepo(),
		TemplateFavoriteRepo: repository.NewTemplateFavoriteRepo(),
	}
}

// ClientTemplateRequest 使用公共客户端上下文匹配模板分类投放范围。
type ClientTemplateRequest struct {
	PositionKey string `form:"position_key"`
	AccountBaseRequest
}

type TemplateListRequest struct {
	Page           int    `form:"page" binding:"omitempty,min=1" default:"1"`
	PageSize       int    `form:"pageSize" binding:"omitempty,min=1" default:"10"`
	PositionKey    string `form:"position_key" binding:"required,max=64"`
	TemplateTypeId uint64 `form:"template_type_id" binding:"required,max=64"`
	AccountBaseRequest
}

type ClientTemplateRecommendRequest struct {
	PositionKey string `form:"position_key" binding:"required,max=64"`
	AccountBaseRequest
}

type TemplateInfoRequest struct {
	TemplateID uint64 `form:"template_id" binding:"required,max=64"`
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
	ID              uint64                         `json:"id"`
	TemplateTypeID  uint64                         `json:"video_template_type_id"`
	Name            string                         `json:"name"`
	TemplateType    int64                          `json:"template_type"`
	CoverImageURL   string                         `json:"cover_image_url"`
	OriginalURL     string                         `json:"original_url"`
	ThumbnailURL    string                         `json:"thumbnail_url"`
	Prompt          string                         `json:"prompt"`
	Description     string                         `json:"description"`
	Sort            int                            `json:"sort"`
	UsageCount      uint64                         `json:"usage_count"`
	FavoriteCount   uint64                         `json:"favorite_count"`
	ViewCount       uint64                         `json:"view_count"`
	IsFavorite      int                            `json:"is_favorite"`
	ModelID         uint64                         `json:"model_id"`
	ModelCode       string                         `json:"model_code"`
	ModelName       string                         `json:"model_name"`
	ModelParameters []ClientTemplateModelParameter `json:"model_parameters"`
}

type ClientTemplateDisplayItem struct {
	ClientTemplate
	DisplayConfigID uint64 `json:"display_config_id"`
	PositionKey     string `json:"position_key"`
	DisplaySort     int    `json:"display_sort"`
}

type ClientCategoriesRequest struct {
	TemplateID    uint64 `json:"template_id" binding:"required,max=64"`
	ComplaintType string `json:"complaint_type" binding:"required,max=64"`
	Content       string `json:"content"`
}

func (s *ClientTemplateService) List(ctx *gin.Context, req *ClientTemplateRequest) ([]ClientTemplateType, error) {
	GetCtxAccountBaseRequest(ctx, &req.AccountBaseRequest)
	countryCode := strings.ToUpper(strings.TrimSpace(req.ClientCountry))
	if countryCode == "" {
		countryCode = strings.ToUpper(strings.TrimSpace(req.ClientCountry))
	}

	types, err := s.typeRepo.ListForClient(ctx, repository.ClientTemplateTypeTargets{
		PositionKey: strings.TrimSpace(req.PositionKey), CountryCode: countryCode,
		AppCode: strings.TrimSpace(req.AppName), PackageCode: strings.TrimSpace(req.AppPackage),
		VersionCode: strings.TrimSpace(req.AppVersion),
	})
	if err != nil {
		return nil, err
	}
	rows, err := s.templateRepo.ListForClient(ctx, repository.ClientTemplateTargets{
		TemplateTypeIDs: templateTypeIDs(types),
	})
	if err != nil {
		return nil, err
	}
	configurations, err := s.loadTemplateModelConfigurations(ctx, rows)
	if err != nil {
		return nil, err
	}
	return buildClientTemplateGroups(types, rows, configurations), nil
}

func (s *ClientTemplateService) Categories(ctx *gin.Context, req *ClientTemplateRequest) ([]ClientTemplateType, error) {
	user, err := s.userRepo.GetByID(ctx, middleware.GetAPIUserID(ctx))
	if err != nil {
		return nil, err
	}

	GetCtxAccountBaseRequest(ctx, &req.AccountBaseRequest)
	countryCode := strings.ToUpper(strings.TrimSpace(req.ClientCountry))
	if countryCode == "" {
		countryCode = user.ClientCountry
	}

	types, err := s.typeRepo.ListForClient(ctx, repository.ClientTemplateTypeTargets{
		PositionKey: strings.TrimSpace(req.PositionKey), CountryCode: countryCode,
		AppCode: strings.TrimSpace(req.AppName), PackageCode: strings.TrimSpace(req.AppPackage),
		VersionCode: strings.TrimSpace(req.AppVersion),
	})
	if err != nil {
		return nil, err
	}
	rows, err := s.templateRepo.ListForClient(ctx, repository.ClientTemplateTargets{
		TemplateTypeIDs: templateTypeIDs(types),
	})
	if err != nil {
		return nil, err
	}
	configurations, err := s.loadTemplateModelConfigurations(ctx, rows)
	if err != nil {
		return nil, err
	}
	return buildClientTemplateGroups(types, rows, configurations), nil
}

func buildClientTemplateGroups(types []model.VideoTemplateType, rows []model.VideoTemplate, configurationArgs ...map[uint64]clientTemplateModelConfiguration) []ClientTemplateType {
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
		if rows[i].ViewCount != rows[j].ViewCount {
			return rows[i].ViewCount > rows[j].ViewCount
		}
		return rows[i].ID > rows[j].ID
	})
	templatesByType := make(map[uint64][]ClientTemplate, len(types))
	var configurations map[uint64]clientTemplateModelConfiguration
	requireConfiguration := len(configurationArgs) > 0
	if requireConfiguration {
		configurations = configurationArgs[0]
	}
	for i := range rows {
		item := rows[i]
		configuration, exists := configurations[item.ID]
		if requireConfiguration && !exists {
			continue
		}
		templatesByType[item.TemplateTypeID] = append(templatesByType[item.TemplateTypeID], mapClientTemplate(&item, configuration))
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

func mapClientTemplate(item *model.VideoTemplate, configurationArgs ...clientTemplateModelConfiguration) ClientTemplate {
	result := ClientTemplate{
		ID:             item.ID,
		TemplateTypeID: item.TemplateTypeID,
		Name:           item.Name,
		TemplateType:   item.TemplateType,
		CoverImageURL:  item.CoverImageURL,
		OriginalURL:    item.OriginalURL,
		ThumbnailURL:   item.ThumbnailURL,
		Prompt:         item.Prompt,
		Description:    item.Description,
		Sort:           int(item.Sort),
		UsageCount:     item.UsageCount,
		ViewCount:      item.ViewCount,
	}
	if len(configurationArgs) > 0 {
		configuration := configurationArgs[0]
		result.ModelID = configuration.ModelID
		result.ModelCode = configuration.ModelCode
		result.ModelName = configuration.ModelName
		result.ModelParameters = configuration.Parameters
	}
	if result.ModelParameters == nil {
		result.ModelParameters = []ClientTemplateModelParameter{}
	}
	return result
}

func templateTypeIDs(items []model.VideoTemplateType) []uint64 {
	result := make([]uint64, len(items))
	for i := range items {
		result[i] = items[i].ID
	}
	return result
}

func (s *ClientTemplateService) Recommend(ctx *gin.Context, req *ClientTemplateRecommendRequest) ([]ClientTemplate, error) {
	user, err := s.userRepo.GetByID(ctx, middleware.GetAPIUserID(ctx))
	if err != nil {
		return nil, err
	}
	GetCtxAccountBaseRequest(ctx, &req.AccountBaseRequest)
	countryCode := strings.ToUpper(strings.TrimSpace(req.ClientCountry))
	if countryCode == "" {
		countryCode = strings.ToUpper(strings.TrimSpace(user.ClientCountry))
	}
	rows, err := s.displayRepo.ListForClient(ctx, repository.ClientTemplateDisplayTargets{
		PositionKey: strings.TrimSpace(req.PositionKey), CountryCode: countryCode,
		AppCode: strings.TrimSpace(req.AppName), PackageCode: strings.TrimSpace(req.AppPackage),
		VersionCode: strings.TrimSpace(req.AppVersion),
	})
	if err != nil {
		return nil, err
	}
	templates := make([]model.VideoTemplate, 0, len(rows))
	for i := range rows {
		if rows[i].Template != nil {
			templates = append(templates, rows[i].Template.VideoTemplate)
		}
	}
	configurations, err := s.loadTemplateModelConfigurations(ctx, templates)
	if err != nil {
		return nil, err
	}
	result := make([]ClientTemplate, 0, len(rows))
	for i := range rows {
		if rows[i].Template != nil {
			template := &rows[i].Template.VideoTemplate
			if configuration, exists := configurations[template.ID]; exists {
				result = append(result, mapClientTemplate(template, configuration))
			}
		}
	}
	return result, nil
}

func (s *ClientTemplateService) CategoryTemplateList(ctx *gin.Context, req *TemplateListRequest) ([]ClientTemplate, error) {
	user, err := s.userRepo.GetByID(ctx, middleware.GetAPIUserID(ctx))
	if err != nil {
		return nil, err
	}
	GetCtxAccountBaseRequest(ctx, &req.AccountBaseRequest)
	countryCode := strings.ToUpper(strings.TrimSpace(req.ClientCountry))
	if countryCode == "" {
		countryCode = strings.ToUpper(strings.TrimSpace(user.ClientCountry))
	}

	page, pageSize := req.Page, req.PageSize
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	types, err := s.typeRepo.ListForClient(ctx, repository.ClientTemplateTypeTargets{
		PositionKey: strings.TrimSpace(req.PositionKey), CountryCode: countryCode,
		AppCode: strings.TrimSpace(req.AppName), PackageCode: strings.TrimSpace(req.AppPackage),
		VersionCode: strings.TrimSpace(req.AppVersion),
	})
	if err != nil {
		return nil, err
	}
	allowed := false
	for _, item := range types {
		if item.ID == req.TemplateTypeId {
			allowed = true
			break
		}
	}
	if !allowed {
		return []ClientTemplate{}, nil
	}
	rows, _, err := s.templateRepo.PageList(ctx, page, pageSize, &repository.TemplateListFilter{
		TemplateTypeID: req.TemplateTypeId,
	})
	if err != nil {
		return nil, err
	}
	templates := make([]model.VideoTemplate, 0, len(rows))
	for i := range rows {
		templates = append(templates, rows[i].VideoTemplate)
	}
	configurations, err := s.loadTemplateModelConfigurations(ctx, templates)
	if err != nil {
		return nil, err
	}
	result := make([]ClientTemplate, 0, len(templates))
	for i := range templates {
		if configuration, exists := configurations[templates[i].ID]; exists {
			result = append(result, mapClientTemplate(&templates[i], configuration))
		}
	}
	return result, nil
}

func (s *ClientTemplateService) ClientTemplateInfo(ctx *gin.Context, req *TemplateInfoRequest) (ClientTemplate, error) {
	template, err := s.templateRepo.GetTemplateID(ctx, req.TemplateID)
	if err != nil {
		return ClientTemplate{}, err
	}
	configurations, err := s.loadTemplateModelConfigurations(ctx, []model.VideoTemplate{*template})
	if err != nil {
		return ClientTemplate{}, err
	}
	configuration, exists := configurations[template.ID]
	if !exists {
		return ClientTemplate{}, errors.New("template model is unavailable")
	}
	resp := mapClientTemplate(template, configuration)
	if s.TemplateFavoriteRepo.GetUserFavorite(ctx, middleware.GetAPIUserID(ctx), template.ID) {
		resp.IsFavorite = 1
	}
	return resp, nil
}

func (s *ClientTemplateService) Complaint(ctx *gin.Context, req *ClientCategoriesRequest) (interface{}, error) {
	template, err := s.templateRepo.GetByID(ctx, uint(req.TemplateID))
	if err != nil {
		return nil, fmt.Errorf("get template id %d: %w", req.TemplateID, err)
	}
	if template.Status != 1 {
		return nil, errors.New("template is not active")
	}
	err = repository.QFrom(ctx).VideoUserTemplateComplaint.WithContext(ctx).Create(&model.VideoUserTemplateComplaint{
		TemplateID:    template.ID,
		UserID:        middleware.GetAPIUserID(ctx),
		ComplaintType: req.ComplaintType,
		Content:       req.Content,
	})
	if err != nil {
		return nil, fmt.Errorf("create complaint %d: %w", template.ID, err)
	}
	return nil, nil
}

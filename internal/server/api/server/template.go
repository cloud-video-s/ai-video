package service

import (
	"ai-video/internal/middleware"
	"errors"
	"fmt"
	"sort"
	"strings"

	"ai-video/internal/gen/model"
	"ai-video/internal/pkg/uploadruntime"
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
	modelParameterRepo    *repository.ModelParameterRepo
	templateParameterRepo *repository.TemplateModelParameterRepo
}

func NewClientTemplateService() *ClientTemplateService {
	return &ClientTemplateService{
		typeRepo: repository.NewTemplateTypeRepo(), templateRepo: repository.NewTemplateRepo(),
		displayRepo: repository.NewTemplateDisplayConfigRepo(),
		userRepo:    repository.NewAppUserRepo(), countryRepo: repository.NewCountryRepo(),
		modelRepo: repository.NewModelRepo(), modelParameterRepo: repository.NewModelParameterRepo(),
		templateParameterRepo: repository.NewTemplateModelParameterRepo(),
		TemplateFavoriteRepo:  repository.NewTemplateFavoriteRepo(),
	}
}

// ClientTemplateRequest 使用公共客户端上下文匹配模板分类投放范围。
type ClientTemplateRequest struct {
	BasePage
	PositionKey string `form:"position_key" json:"position_key"`
	AccountBaseRequest
}

type TemplateListRequest struct {
	BasePage
	PositionKey    string `form:"position_key" binding:"omitempty,max=64"`
	TemplateTypeId uint64 `form:"template_type_id" binding:"omitempty,max=64"`
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
	Icon         string           `json:"icon"`
	Description  string           `json:"description"`
	Sort         int64            `json:"sort"`
	Templates    []ClientTemplate `json:"templates"`
}

type ClientTemplate struct {
	ID             uint64 `json:"id"`
	TemplateTypeID uint64 `json:"template_type_id"`
	Name           string `json:"name"`
	TemplateType   int64  `json:"template_type"`
	Icon           string `json:"icon"`
	CoverImageURL  string `json:"cover_image_url"`
	OriginalURL    string `json:"original_url"`
	ThumbnailURL   string `json:"thumbnail_url"`
	Prompt         string `json:"prompt"`
	Description    string `json:"description"`
	Sort           int    `json:"sort"`
	UsageCount     uint64 `json:"usage_count"`
	FavoriteCount  uint64 `json:"favorite_count"`
	ViewCount      uint64 `json:"view_count"`
	IsFavorite     int    `json:"is_favorite"`
	ModelScore     int64  `json:"model_score"`
}

type ClientTemplateDisplayItem struct {
	ClientTemplate
	DisplayConfigID uint64 `json:"display_config_id"`
	PositionKey     string `json:"position_key"`
	DisplaySort     int    `json:"display_sort"`
}

type ClientCategoriesRequest struct {
	TemplateID    uint64 `json:"template_id" binding:"required,gt=0"`
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

func (s *ClientTemplateService) Categories(ctx *gin.Context, req *ClientTemplateRequest) (interface{}, error) {
	user, err := s.userRepo.GetByID(ctx, middleware.GetAPIUserID(ctx))
	if err != nil {
		return nil, err
	}
	GetCtxAccountBaseRequest(ctx, &req.AccountBaseRequest)
	countryCode := strings.ToUpper(strings.TrimSpace(req.ClientCountry))
	if countryCode == "" {
		countryCode = user.ClientCountry
	}
	if req.PageSize == 0 {
		req.PageSize = 5
	}
	types, count, err := s.typeRepo.GetLimitListClient(ctx, repository.ClientTemplateTypeTargets{
		PositionKey: strings.TrimSpace(req.PositionKey), CountryCode: countryCode,
		AppCode: strings.TrimSpace(req.AppName), PackageCode: strings.TrimSpace(req.AppPackage),
		VersionCode: strings.TrimSpace(req.AppVersion),
		Page:        req.Page,
		Limit:       req.PageSize,
	})
	if err != nil {
		return nil, err
	}
	var data []ClientTemplateType
	for _, item := range types {
		rows, _ := s.templateRepo.GetListForClient(ctx, repository.ClientTemplateTargets{
			TemplateTypeID: item.ID,
			Page:           1,
			PageSize:       10,
		})
		var templates []ClientTemplate
		for _, val := range rows {
			templates = append(templates, mapClientTemplate(&val))
		}
		data = append(data, ClientTemplateType{
			ID:           item.ID,
			CategoryName: item.CategoryName,
			Icon:         uploadruntime.PublicURL(item.Icon),
			Description:  item.Description,
			Sort:         item.Sort,
			Templates:    templates,
		})
	}
	return GetPageResponse(int64(req.Page), int64(req.PageSize), count, data)
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
	for i := range rows {
		templatesByType[rows[i].TemplateTypeID] = append(templatesByType[rows[i].TemplateTypeID], mapClientTemplate(&rows[i]))
	}
	result := make([]ClientTemplateType, 0, len(types))
	for i := range types {
		item := types[i]
		templates := templatesByType[item.ID]
		if len(templates) == 0 {
			continue
		}
		result = append(result, ClientTemplateType{
			ID: item.ID, CategoryName: item.CategoryName, Icon: uploadruntime.PublicURL(item.Icon), Description: item.Description,
			Sort:      item.Sort,
			Templates: templates,
		})
	}
	return result
}

func mapClientTemplate(item *model.VideoTemplate) ClientTemplate {
	result := ClientTemplate{
		ID:             item.ID,
		TemplateTypeID: item.TemplateTypeID,
		Name:           item.Name,
		TemplateType:   item.TemplateType,
		Icon:           uploadruntime.PublicURL(item.Icon),
		CoverImageURL:  uploadruntime.PublicURL(item.CoverImageURL),
		OriginalURL:    uploadruntime.PublicURL(item.OriginalURL),
		ThumbnailURL:   uploadruntime.PublicURL(item.ThumbnailURL),
		Prompt:         item.Prompt,
		Description:    item.Description,
		Sort:           int(item.Sort),
		UsageCount:     item.UsageCount,
		ViewCount:      item.ViewCount,
		ModelScore:     item.AIModel.Score,
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
	result := make([]ClientTemplate, 0, len(rows))
	for i := range rows {
		if rows[i].Template != nil {
			template := &rows[i].Template.VideoTemplate
			result = append(result, mapClientTemplate(template))
		}
	}
	return result, nil
}

func (s *ClientTemplateService) CategoryTemplateList(ctx *gin.Context, req *TemplateListRequest) (interface{}, error) {
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
	var templateTypeId []uint64
	if req.TemplateTypeId > 0 {
		templateTypeId = append(templateTypeId, req.TemplateTypeId)
	}
	if req.PositionKey != "" {
		types, err := s.typeRepo.ListForClient(ctx, repository.ClientTemplateTypeTargets{
			PositionKey: strings.TrimSpace(req.PositionKey), CountryCode: countryCode,
			AppCode: strings.TrimSpace(req.AppName), PackageCode: strings.TrimSpace(req.AppPackage),
			VersionCode: strings.TrimSpace(req.AppVersion),
		})
		if err != nil {
			return nil, err
		}
		for _, t := range types {
			if req.TemplateTypeId > 0 && t.ID == req.TemplateTypeId {
				continue
			}
			templateTypeId = append(templateTypeId, t.ID)
		}
	}
	templates, total, err := s.templateRepo.GetPageList(ctx, page, pageSize, &repository.TemplateListRequest{
		TemplateTypeID: templateTypeId,
		//PositionKey:    req.PositionKey,
	})
	if err != nil {
		return nil, err
	}
	result := make([]ClientTemplate, 0, len(templates))
	for i := range templates {
		result = append(result, mapClientTemplate(templates[i]))
	}
	return GetPageResponse(int64(req.Page), int64(req.PageSize), total, result)
}

func (s *ClientTemplateService) ClientTemplateInfo(ctx *gin.Context, req *TemplateInfoRequest) (ClientTemplate, error) {
	template, err := s.templateRepo.GetTemplateID(ctx, req.TemplateID)
	if err != nil {
		return ClientTemplate{}, err
	}
	resp := mapClientTemplate(template)
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

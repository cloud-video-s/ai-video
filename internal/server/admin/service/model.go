package service

import (
	"ai-video/internal/gen/model"
	"ai-video/internal/repository"
	"context"
	"errors"
	"net/url"
	"regexp"
	"strings"

	"gorm.io/gorm"
)

const modelSecretMask = "******"

var modelCodePattern = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

type ModelService struct {
	repo          *repository.ModelRepo
	platformRepo  *repository.PlatformRepo
	parameterRepo *repository.ModelParameterRepo
	taskRepo      *repository.UserGenerationTaskRepo
	templateRepo  *repository.TemplateRepo
}

func NewModelService() *ModelService {
	return &ModelService{
		repo: repository.NewModelRepo(), platformRepo: repository.NewPlatformRepo(),
		parameterRepo: repository.NewModelParameterRepo(), taskRepo: repository.NewUserGenerationTaskRepo(),
		templateRepo: repository.NewTemplateRepo(),
	}
}

type ListModelRequest struct {
	ListSortRequest
	Keyword       string   `form:"keyword"`
	PlatformID    *int64   `form:"platform_id" binding:"omitempty,gt=0"`
	ModelType     uint32   `form:"model_type"`
	ModelFeatures []uint32 `form:"model_features" collection_format:"csv" binding:"omitempty,dive,oneof=1 2 3 4"`
	Status        *uint32  `form:"status" binding:"omitempty,oneof=0 1"`
}

type ModelPayload struct {
	PlatformID     int64  `json:"platform_id" binding:"required,gt=0"`
	Name           string `json:"name" binding:"required,max=64"`
	Code           string `json:"code" binding:"required,max=32"`
	ModelType      uint32 `json:"model_type" binding:"required,oneof=1 2"`
	ModelFeatures  uint32 `json:"model_features" binding:"required,oneof=1 2 3 4"`
	Version        string `json:"version" binding:"required,max=16"`
	HostURL        string `json:"host_url" binding:"required,max=255"`
	SubmitEndpoint string `json:"submit_endpoint" binding:"required,max=255"`
	StatusEndpoint string `json:"status_endpoint" binding:"max=255"`
	RequestMethod  string `json:"request_method" binding:"required,oneof=GET POST"`
	AuthType       uint32 `json:"auth_type" binding:"required,oneof=1 2"`
	APIKey         string `json:"api_key" binding:"max=2048"`
	Score          int64  `json:"score" binding:"gte=0"`
	Icon           string `json:"icon" binding:"max=255"`
	Description    string `json:"description" binding:"max=255"`
	Status         uint32 `json:"status" binding:"oneof=0 1"`
}

type ModelPlatformView struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	Code    string `json:"code"`
	BaseURL string `json:"base_url"`
}

type ModelView struct {
	ID               int64              `json:"id"`
	PlatformID       int64              `json:"platform_id"`
	Platform         *ModelPlatformView `json:"platform"`
	Name             string             `json:"name"`
	Code             string             `json:"code"`
	ModelType        uint32             `json:"model_type"`
	ModelFeatures    uint32             `json:"model_features"`
	Version          string             `json:"version"`
	HostURL          string             `json:"host_url"`
	SubmitEndpoint   string             `json:"submit_endpoint"`
	StatusEndpoint   string             `json:"status_endpoint"`
	RequestMethod    string             `json:"request_method"`
	AuthType         uint32             `json:"auth_type"`
	APIKey           string             `json:"api_key"`
	APIKeyConfigured bool               `json:"api_key_configured"`
	Score            int64              `json:"score"`
	Icon             string             `json:"icon"`
	Description      string             `json:"description"`
	Status           uint32             `json:"status"`
	CreatedAt        string             `json:"created_at"`
	UpdatedAt        string             `json:"updated_at"`
}

func (s *ModelService) List(ctx context.Context, page, pageSize int, req *ListModelRequest) ([]ModelView, int64, error) {
	items, total, err := s.repo.PageList(ctx, page, pageSize, &repository.ModelListFilter{
		ListSort: req.listSort(),
		Keyword:  strings.TrimSpace(req.Keyword), PlatformID: req.PlatformID,
		ModelType: req.ModelType, ModelFeatures: req.ModelFeatures, Status: req.Status,
	})
	if err != nil {
		return nil, 0, err
	}
	ids := make([]int64, 0, len(items))
	for i := range items {
		ids = append(ids, items[i].ID)
	}
	configured, err := s.repo.APIKeyConfigured(ctx, ids)
	if err != nil {
		return nil, 0, err
	}
	result := make([]ModelView, 0, len(items))
	for i := range items {
		result = append(result, modelView(&items[i], configured[items[i].ID]))
	}
	return result, total, nil
}

func (s *ModelService) Get(ctx context.Context, id int64) (*ModelView, error) {
	item, err := s.repo.GetByIDWithPlatform(ctx, id)
	if err != nil {
		return nil, notFoundOr(err, "模型不存在")
	}
	configured, err := s.repo.APIKeyConfigured(ctx, []int64{id})
	if err != nil {
		return nil, err
	}
	view := modelView(item, configured[id])
	return &view, nil
}

func (s *ModelService) Create(ctx context.Context, req *ModelPayload) (*ModelView, error) {
	platform, err := s.validatePayload(ctx, req, 0, false)
	if err != nil {
		return nil, err
	}
	item := &model.VideoModel{}
	applyModelPayload(item, req)
	if err := repository.Transaction(ctx, func(txCtx context.Context) error {
		if err := s.repo.Create(txCtx, item); err != nil {
			return err
		}
		return s.repo.UpdateAPIKey(txCtx, item.ID, req.APIKey)
	}); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, errors.New("模型编码已存在")
		}
		return nil, err
	}
	item.Platform = *platform
	view := modelView(item, true)
	return &view, nil
}

func (s *ModelService) Update(ctx context.Context, id int64, req *ModelPayload) (*ModelView, error) {
	item, err := s.repo.GetByID(ctx, uint(id))
	if err != nil {
		return nil, notFoundOr(err, "模型不存在")
	}
	platform, err := s.validatePayload(ctx, req, id, true)
	if err != nil {
		return nil, err
	}
	if item.ModelType != req.ModelType {
		templateCount, err := s.templateRepo.CountByModel(ctx, uint64(id))
		if err != nil {
			return nil, err
		}
		if templateCount > 0 {
			return nil, errors.New("该模型已关联模板，不能修改模型类型；请先调整关联模板")
		}
	}
	applyModelPayload(item, req)
	if err := repository.Transaction(ctx, func(txCtx context.Context) error {
		if err := s.repo.UpdateFields(txCtx, item); err != nil {
			return err
		}
		if req.APIKey != "" && req.APIKey != modelSecretMask {
			return s.repo.UpdateAPIKey(txCtx, item.ID, req.APIKey)
		}
		return nil
	}); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, errors.New("模型编码已存在")
		}
		return nil, err
	}
	configured, err := s.repo.APIKeyConfigured(ctx, []int64{id})
	if err != nil {
		return nil, err
	}
	item.Platform = *platform
	view := modelView(item, configured[id])
	return &view, nil
}

func (s *ModelService) Delete(ctx context.Context, id int64) error {
	if _, err := s.repo.GetByID(ctx, uint(id)); err != nil {
		return notFoundOr(err, "模型不存在")
	}
	taskCount, err := s.taskRepo.CountByModel(ctx, uint64(id))
	if err != nil {
		return err
	}
	if taskCount > 0 {
		return errors.New("该模型已有用户生成任务，不能删除；可将其禁用")
	}
	templateCount, err := s.templateRepo.CountByModel(ctx, uint64(id))
	if err != nil {
		return err
	}
	if templateCount > 0 {
		return errors.New("该模型已关联模板，请先调整关联模板；也可以将模型禁用")
	}
	// Model and its parameter rows are soft-deleted in one transaction.
	return repository.Transaction(ctx, func(txCtx context.Context) error {
		if err := s.parameterRepo.SoftDeleteByModel(txCtx, id); err != nil {
			return err
		}
		return s.repo.Delete(txCtx, uint(id))
	})
}

func (s *ModelService) validatePayload(ctx context.Context, req *ModelPayload, currentID int64, updating bool) (*model.VideoPlatform, error) {
	req.Name = strings.TrimSpace(req.Name)
	req.Code = strings.TrimSpace(req.Code)
	req.Version = strings.TrimSpace(req.Version)
	req.HostURL = strings.TrimRight(strings.TrimSpace(req.HostURL), "/")
	req.SubmitEndpoint = strings.TrimSpace(req.SubmitEndpoint)
	req.StatusEndpoint = strings.TrimSpace(req.StatusEndpoint)
	req.RequestMethod = strings.ToUpper(strings.TrimSpace(req.RequestMethod))
	req.APIKey = strings.TrimSpace(req.APIKey)
	req.Icon = strings.TrimSpace(req.Icon)
	req.Description = strings.TrimSpace(req.Description)
	if req.Name == "" || req.ModelType == 0 || req.ModelFeatures == 0 || req.Version == "" || !modelCodePattern.MatchString(req.Code) {
		return nil, errors.New("模型名称、类型和版本不能为空，模型编码只能包含字母、数字、点、下划线和中划线")
	}
	if req.ModelFeatures > 4 {
		return nil, errors.New("模型类型必须是 1=通用、2=模板、3=生成模型或 4=工具")
	}
	parsed, err := url.Parse(req.HostURL)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.User != nil {
		return nil, errors.New("API 域名必须是有效且不含用户凭据的 HTTP(S) 地址")
	}
	for _, endpoint := range modelEndpoints(req) {
		parsedEndpoint, parseErr := url.Parse(endpoint)
		if parseErr != nil || parsedEndpoint.IsAbs() || !strings.HasPrefix(endpoint, "/") {
			return nil, errors.New("生成地址路由和查询地址路由必须是以 / 开头的相对路径")
		}
	}
	if !updating && (req.APIKey == "" || req.APIKey == modelSecretMask) {
		return nil, errors.New("新建模型必须配置密钥")
	}
	platform, err := s.platformRepo.GetByID(ctx, uint(req.PlatformID))
	if err != nil {
		return nil, notFoundOr(err, "关联平台不存在")
	}
	if platform.Status != 1 {
		return nil, errors.New("只能关联启用中的平台")
	}
	existing, err := s.repo.GetByCode(ctx, req.Code)
	if err == nil && existing.ID != currentID {
		return nil, errors.New("模型编码已存在")
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	return platform, nil
}

func modelEndpoints(req *ModelPayload) []string {
	endpoints := []string{req.SubmitEndpoint}
	if req.ModelType == 2 {
		endpoints = append(endpoints, req.StatusEndpoint)
	}
	return endpoints
}

func applyModelPayload(item *model.VideoModel, req *ModelPayload) {
	item.PlatformID = req.PlatformID
	item.Name, item.Code = req.Name, req.Code
	item.ModelType, item.ModelFeatures, item.Version = req.ModelType, req.ModelFeatures, req.Version
	item.HostURL = req.HostURL
	item.SubmitEndpoint, item.StatusEndpoint = req.SubmitEndpoint, req.StatusEndpoint
	item.RequestMethod, item.AuthType = req.RequestMethod, req.AuthType
	item.Score, item.Icon, item.Description, item.Status = req.Score, req.Icon, req.Description, req.Status
}

func modelView(item *model.VideoModel, apiKeyConfigured bool) ModelView {
	var platform *ModelPlatformView
	if item.Platform.ID != 0 {
		platform = &ModelPlatformView{ID: item.Platform.ID, Name: item.Platform.Name, Code: item.Platform.Code, BaseURL: item.Platform.BaseURL}
	}
	masked := ""
	if apiKeyConfigured {
		masked = modelSecretMask
	}
	return ModelView{
		ID: item.ID, PlatformID: item.PlatformID, Platform: platform,
		Name: item.Name, Code: item.Code, ModelType: item.ModelType, ModelFeatures: item.ModelFeatures, Version: item.Version,
		HostURL: item.HostURL, SubmitEndpoint: item.SubmitEndpoint, StatusEndpoint: item.StatusEndpoint,
		RequestMethod: item.RequestMethod, AuthType: item.AuthType, APIKey: masked,
		APIKeyConfigured: apiKeyConfigured, Score: item.Score, Icon: item.Icon, Description: item.Description, Status: item.Status,
		CreatedAt: item.CreatedAt.Format("2006-01-02 15:04:05"), UpdatedAt: item.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
}

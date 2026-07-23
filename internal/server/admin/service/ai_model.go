package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"regexp"
	"strings"

	"ai-video/internal/gen/model"
	"ai-video/internal/repository"

	"gorm.io/gorm"
)

const aiModelSecretMask = "******"

var aiModelCodePattern = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

type AIModelService struct{ repo *repository.AIModelRepo }

func NewAIModelService() *AIModelService { return &AIModelService{repo: repository.NewAIModelRepo()} }

type ListAIModelRequest struct {
	Keyword  string `form:"keyword" binding:"omitempty,max=255"`
	Provider string `form:"provider" binding:"omitempty,max=32"`
	Status   *int8  `form:"status" binding:"omitempty,oneof=0 1"`
}

// AIModelPayload 是模型配置中心的写入结构。更新时 APIKey 留空或传 ****** 表示保留原密钥。
type AIModelPayload struct {
	Code                string `json:"code" binding:"required,max=64"`
	Name                string `json:"name" binding:"required,max=128"`
	Provider            string `json:"provider" binding:"required,max=32"`
	ModelName           string `json:"model_name" binding:"required,max=128"`
	BaseURL             string `json:"base_url" binding:"required,max=512"`
	SubmitPath          string `json:"submit_path" binding:"required,max=255"`
	StatusPath          string `json:"status_path" binding:"required,max=255"`
	APIKey              string `json:"api_key" binding:"max=2048"`
	DefaultParameters   string `json:"default_parameters" binding:"max=20000"`
	HTTPTimeoutSeconds  uint32 `json:"http_timeout_seconds" binding:"min=1,max=600"`
	PollIntervalSeconds uint32 `json:"poll_interval_seconds" binding:"min=1,max=300"`
	TaskTimeoutSeconds  uint32 `json:"task_timeout_seconds" binding:"min=30,max=86400"`
	Status              int8   `json:"status" binding:"oneof=0 1"`
	Description         string `json:"description" binding:"max=1000"`
}

type AIModelView struct {
	ID                  uint64 `json:"id"`
	Code                string `json:"code"`
	Name                string `json:"name"`
	Provider            string `json:"provider"`
	ModelName           string `json:"model_name"`
	BaseURL             string `json:"base_url"`
	SubmitPath          string `json:"submit_path"`
	StatusPath          string `json:"status_path"`
	APIKey              string `json:"api_key"`
	APIKeyConfigured    bool   `json:"api_key_configured"`
	DefaultParameters   string `json:"default_parameters"`
	HTTPTimeoutSeconds  uint32 `json:"http_timeout_seconds"`
	PollIntervalSeconds uint32 `json:"poll_interval_seconds"`
	TaskTimeoutSeconds  uint32 `json:"task_timeout_seconds"`
	Status              int8   `json:"status"`
	Description         string `json:"description"`
	CreatedAt           string `json:"created_at"`
	UpdatedAt           string `json:"updated_at"`
}

func (s *AIModelService) List(ctx context.Context, page, pageSize int, req *ListAIModelRequest) ([]AIModelView, int64, error) {
	items, total, err := s.repo.PageList(ctx, page, pageSize, &repository.AIModelListFilter{
		Keyword: strings.TrimSpace(req.Keyword), Provider: strings.TrimSpace(req.Provider), Status: req.Status,
	})
	if err != nil {
		return nil, 0, err
	}
	result := make([]AIModelView, 0, len(items))
	for i := range items {
		result = append(result, aiModelView(&items[i]))
	}
	return result, total, nil
}

func (s *AIModelService) Get(ctx context.Context, id uint64) (*AIModelView, error) {
	item, err := s.repo.GetByID(ctx, uint(id))
	if err != nil {
		return nil, err
	}
	view := aiModelView(item)
	return &view, nil
}

func (s *AIModelService) Create(ctx context.Context, req *AIModelPayload) (*AIModelView, error) {
	if err := validateAIModelPayload(req, false); err != nil {
		return nil, err
	}
	if _, err := s.repo.GetByCode(ctx, req.Code); err == nil {
		return nil, errors.New("模型编码已存在")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	item := &model.VideoAiModel{}
	applyAIModelPayload(item, req, false)
	if item.Status == 1 && strings.TrimSpace(item.APIKey) == "" {
		return nil, errors.New("启用模型前必须配置 API Key")
	}
	if err := s.repo.Create(ctx, item); err != nil {
		return nil, err
	}
	view := aiModelView(item)
	return &view, nil
}

func (s *AIModelService) Update(ctx context.Context, id uint64, req *AIModelPayload) (*AIModelView, error) {
	item, err := s.repo.GetByID(ctx, uint(id))
	if err != nil {
		return nil, err
	}
	if err := validateAIModelPayload(req, true); err != nil {
		return nil, err
	}
	if existing, lookupErr := s.repo.GetByCode(ctx, req.Code); lookupErr == nil && existing.ID != id {
		return nil, errors.New("模型编码已存在")
	} else if lookupErr != nil && !errors.Is(lookupErr, gorm.ErrRecordNotFound) {
		return nil, lookupErr
	}
	applyAIModelPayload(item, req, true)
	if item.Status == 1 && strings.TrimSpace(item.APIKey) == "" {
		return nil, errors.New("启用模型前必须配置 API Key")
	}
	if err := s.repo.UpdateFields(ctx, item); err != nil {
		return nil, err
	}
	view := aiModelView(item)
	return &view, nil
}

func (s *AIModelService) Delete(ctx context.Context, id uint64) error {
	if _, err := s.repo.GetByID(ctx, uint(id)); err != nil {
		return err
	}
	count, err := s.repo.TaskCount(ctx, id)
	if err != nil {
		return err
	}
	if count > 0 {
		return errors.New("该模型已有生成任务，不能删除；可将其禁用")
	}
	return s.repo.Delete(ctx, uint(id))
}

func validateAIModelPayload(req *AIModelPayload, updating bool) error {
	req.Code = strings.TrimSpace(req.Code)
	req.Name = strings.TrimSpace(req.Name)
	req.Provider = strings.ToLower(strings.TrimSpace(req.Provider))
	req.ModelName = strings.TrimSpace(req.ModelName)
	req.BaseURL = strings.TrimRight(strings.TrimSpace(req.BaseURL), "/")
	req.SubmitPath = strings.TrimSpace(req.SubmitPath)
	req.StatusPath = strings.TrimSpace(req.StatusPath)
	req.APIKey = strings.TrimSpace(req.APIKey)
	req.DefaultParameters = strings.TrimSpace(req.DefaultParameters)
	req.Description = strings.TrimSpace(req.Description)
	if !aiModelCodePattern.MatchString(req.Code) {
		return errors.New("模型编码只能包含字母、数字、点、下划线和中划线")
	}
	if req.Provider != "modelverse" {
		return errors.New("当前仅支持 modelverse provider")
	}
	parsed, err := url.Parse(req.BaseURL)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.User != nil {
		return errors.New("Base URL 必须是有效的 HTTP(S) 地址且不能包含用户凭据")
	}
	for _, path := range []string{req.SubmitPath, req.StatusPath} {
		parsedPath, pathErr := url.Parse(path)
		if pathErr != nil || parsedPath.IsAbs() || !strings.HasPrefix(path, "/") {
			return errors.New("提交和状态路径必须是以 / 开头的相对路径")
		}
	}
	if req.DefaultParameters == "" {
		req.DefaultParameters = "{}"
	}
	var defaults map[string]interface{}
	if err := json.Unmarshal([]byte(req.DefaultParameters), &defaults); err != nil || defaults == nil {
		return errors.New("默认参数必须是 JSON 对象")
	}
	if !updating && req.Status == 1 && (req.APIKey == "" || req.APIKey == aiModelSecretMask) {
		return errors.New("启用模型前必须配置 API Key")
	}
	return nil
}

func applyAIModelPayload(item *model.VideoAiModel, req *AIModelPayload, updating bool) {
	item.Code, item.Name, item.Provider = req.Code, req.Name, req.Provider
	item.ModelName, item.BaseURL = req.ModelName, req.BaseURL
	item.SubmitPath, item.StatusPath = req.SubmitPath, req.StatusPath
	if !updating || (req.APIKey != "" && req.APIKey != aiModelSecretMask) {
		item.APIKey = req.APIKey
	}
	item.DefaultParameters = req.DefaultParameters
	item.HTTPTimeoutSeconds = req.HTTPTimeoutSeconds
	item.PollIntervalSeconds = req.PollIntervalSeconds
	item.TaskTimeoutSeconds = req.TaskTimeoutSeconds
	item.Status, item.Description = req.Status, req.Description
}

func aiModelView(item *model.VideoAiModel) AIModelView {
	configured := strings.TrimSpace(item.APIKey) != ""
	masked := ""
	if configured {
		masked = aiModelSecretMask
	}
	return AIModelView{
		ID: item.ID, Code: item.Code, Name: item.Name, Provider: item.Provider,
		ModelName: item.ModelName, BaseURL: item.BaseURL, SubmitPath: item.SubmitPath,
		StatusPath: item.StatusPath, APIKey: masked, APIKeyConfigured: configured,
		DefaultParameters: item.DefaultParameters, HTTPTimeoutSeconds: item.HTTPTimeoutSeconds,
		PollIntervalSeconds: item.PollIntervalSeconds, TaskTimeoutSeconds: item.TaskTimeoutSeconds,
		Status: item.Status, Description: item.Description,
		CreatedAt: item.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt: item.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
}

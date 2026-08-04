package service

import (
	"ai-video/internal/generation"
	"ai-video/internal/middleware"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"ai-video/internal/gen/model"
	"ai-video/internal/repository"

	"github.com/gin-gonic/gin"
)

type GenerationModelService struct {
	modelRepo     *repository.ModelRepo
	parameterRepo *repository.ModelParameterRepo
	userRepo      *repository.AppUserRepo
}

func NewGenerationModelService() *GenerationModelService {
	return &GenerationModelService{
		modelRepo:     repository.NewModelRepo(),
		parameterRepo: repository.NewModelParameterRepo(),
	}
}

type GenerationModelRequest struct {
	ModelType uint32 `form:"model_type" binding:"required,gt=0"`
}

type GenerationModelParameter struct {
	ParamKey      string        `json:"param_key"`
	DefaultValue  interface{}   `json:"default_value"`
	AllowedValues []interface{} `json:"allowed_values"`
	Description   string        `json:"description"`
	ParameterType uint32        `json:"parameter_type"`
	Constraints   string        `json:"constraints"`
	Alias         string        `json:"alias"`
	DisplayType   string        `json:"display_type"`
}

type GenerationModelView struct {
	Name        string                     `json:"name"`
	ModelCode   string                     `json:"model_code"`
	Score       uint64                     `json:"score"`
	Icon        string                     `json:"icon"`
	Description string                     `json:"description"`
	Parameters  []GenerationModelParameter `json:"parameter"`
}

type GenerationListRequest struct {
	BasePage
	TaskType uint32 `query:"task_type" json:"task_type" form:"task_type" binding:"omitempty,oneof=1 2 3" default:"1"`
	Status   uint32 `query:"status" json:"status" form:"status" binding:"omitempty,oneof=1 2 3" default:"0"`
}

func (s *GenerationModelService) GenerationList(ctx *gin.Context, req *GenerationListRequest) (*PageResult, error) {
	user, err := s.userRepo.GetByID(ctx, middleware.GetAPIUserID(ctx))
	if err != nil {
		return nil, err
	}
	task, total, err := s.parameterRepo.GetApiPageTask(ctx, user.ID, req.Page, req.PageSize, req.TaskType, req.Status)
	if err != nil {
		return nil, err
	}
	var data []generation.TaskView
	for _, item := range task {
		data = append(data, generation.ViewOf(item))
	}
	return GetPageResponse(int64(req.Page), int64(req.PageSize), total, data)
}

func (s *GenerationModelService) List(ctx context.Context, modelType uint32) ([]GenerationModelView, error) {
	models, err := s.modelRepo.ListEnabledByType(ctx, modelType)
	if err != nil {
		return nil, err
	}
	modelIDs := make([]int64, 0, len(models))
	for i := range models {
		modelIDs = append(modelIDs, models[i].ID)
	}
	parameters, err := s.parameterRepo.ListByModels(ctx, modelIDs)
	if err != nil {
		return nil, err
	}
	parametersByModel := make(map[int64][]GenerationModelParameter, len(models))
	for i := range parameters {
		view, err := generationModelParameterView(&parameters[i])
		if err != nil {
			return nil, err
		}
		parametersByModel[parameters[i].ModelID] = append(parametersByModel[parameters[i].ModelID], view)
	}
	result := make([]GenerationModelView, 0, len(models))
	for i := range models {
		items := parametersByModel[models[i].ID]
		if items == nil {
			items = []GenerationModelParameter{}
		}
		result = append(result, GenerationModelView{Name: models[i].Name,
			ModelCode:   models[i].Code,
			Score:       uint64(models[i].Score),
			Icon:        models[i].Icon,
			Description: models[i].Description,
			Parameters:  items})
	}
	return result, nil
}

func generationModelParameterView(item *model.VideoModelParameter) (GenerationModelParameter, error) {
	var defaultValue interface{}
	if value := strings.TrimSpace(item.DefaultValue); value != "" {
		if err := json.Unmarshal([]byte(value), &defaultValue); err != nil {
			return GenerationModelParameter{}, fmt.Errorf("parameter %s has invalid default_value JSON: %w", item.ParamKey, err)
		}
	}
	allowedValues := make([]interface{}, 0)
	if value := strings.TrimSpace(item.AllowedValues); value != "" {
		if err := json.Unmarshal([]byte(value), &allowedValues); err != nil {
			return GenerationModelParameter{}, fmt.Errorf("parameter %s has invalid allowed_values JSON: %w", item.ParamKey, err)
		}
		if allowedValues == nil {
			allowedValues = make([]interface{}, 0)
		}
	}
	constraints := make(map[string]interface{})
	if value := strings.TrimSpace(item.Constraints); value != "" {
		if err := json.Unmarshal([]byte(value), &constraints); err != nil {
			return GenerationModelParameter{}, fmt.Errorf("parameter %s has invalid constraints JSON: %w", item.ParamKey, err)
		}
		if constraints == nil {
			constraints = make(map[string]interface{})
		}
	}
	return GenerationModelParameter{
		ParamKey:     item.ParamKey,
		DefaultValue: defaultValue, AllowedValues: allowedValues,
		Description:   item.Description,
		ParameterType: item.ParameterType,
		Constraints:   item.Constraints,
		DisplayType:   item.DisplayType,
		Alias:         item.Alias_,
	}, nil
}

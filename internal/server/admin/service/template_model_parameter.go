package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"ai-video/internal/gen/model"
	"ai-video/internal/repository"

	"gorm.io/gorm"
)

type TemplateModelParameterService struct {
	templateRepo *repository.TemplateRepo
	modelRepo    *repository.ModelRepo
	modelParams  *repository.ModelParameterRepo
	repo         *repository.TemplateModelParameterRepo
}

func NewTemplateModelParameterService() *TemplateModelParameterService {
	return &TemplateModelParameterService{
		templateRepo: repository.NewTemplateRepo(), modelRepo: repository.NewModelRepo(),
		modelParams: repository.NewModelParameterRepo(), repo: repository.NewTemplateModelParameterRepo(),
	}
}

type TemplateModelParameterReplacePayload struct {
	Parameters []ModelParameterPayload `json:"parameters" binding:"max=100,dive"`
}

type TemplateModelSummary struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Code      string `json:"code"`
	ModelType uint32 `json:"model_type"`
	Version   string `json:"version"`
	Status    uint32 `json:"status"`
}

type TemplateModelParameterView struct {
	ID            int64                  `json:"id"`
	ModelID       int64                  `json:"model_id"`
	TemplateID    uint64                 `json:"template_id"`
	ParamKey      string                 `json:"param_key"`
	ValueType     string                 `json:"value_type"`
	ParameterType uint32                 `json:"parameter_type"`
	IsRequired    uint32                 `json:"is_required"`
	DefaultValue  interface{}            `json:"default_value"`
	AllowedValues []interface{}          `json:"allowed_values"`
	Constraints   map[string]interface{} `json:"constraints"`
	Description   string                 `json:"description"`
	SortOrder     uint32                 `json:"sort_order"`
	CreatedAt     string                 `json:"created_at"`
	UpdatedAt     string                 `json:"updated_at"`
}

type TemplateModelConfigurationView struct {
	TemplateID uint64                       `json:"template_id"`
	ModelID    uint64                       `json:"model_id"`
	Model      TemplateModelSummary         `json:"model"`
	Parameters []TemplateModelParameterView `json:"parameters"`
}

func (s *TemplateModelParameterService) List(ctx context.Context, templateID uint64) (*TemplateModelConfigurationView, error) {
	template, err := s.templateRepo.GetTemplateID(ctx, templateID)
	if err != nil {
		return nil, notFoundOr(err, "模板不存在")
	}
	return s.configurationView(ctx, template)
}

func (s *TemplateModelParameterService) Replace(ctx context.Context, templateID uint64, req *TemplateModelParameterReplacePayload) (*TemplateModelConfigurationView, error) {
	template, err := s.templateRepo.GetTemplateID(ctx, templateID)
	if err != nil {
		return nil, notFoundOr(err, "模板不存在")
	}
	if req == nil {
		return nil, errors.New("模板模型配置不能为空")
	}
	items, err := s.prepare(ctx, int64(template.ModelID), req.Parameters)
	if err != nil {
		return nil, err
	}
	if err := repository.Transaction(ctx, func(txCtx context.Context) error {
		return s.repo.ReplaceForTemplate(txCtx, template.ID, items)
	}); err != nil {
		return nil, err
	}
	return s.configurationView(ctx, template)
}

func (s *TemplateModelParameterService) configurationView(ctx context.Context, template *model.VideoTemplate) (*TemplateModelConfigurationView, error) {
	modelItem, err := s.modelRepo.GetByID(ctx, uint(template.ModelID))
	if err != nil {
		return nil, notFoundOr(err, "关联模型不存在")
	}
	items, err := s.repo.ListByTemplate(ctx, template.ID)
	if err != nil {
		return nil, err
	}
	parameters := make([]TemplateModelParameterView, 0, len(items))
	for i := range items {
		view, err := templateModelParameterView(&items[i])
		if err != nil {
			return nil, err
		}
		parameters = append(parameters, view)
	}
	return &TemplateModelConfigurationView{
		TemplateID: template.ID,
		ModelID:    template.ModelID,
		Model: TemplateModelSummary{
			ID: modelItem.ID, Name: modelItem.Name, Code: modelItem.Code,
			ModelType: modelItem.ModelType, Version: modelItem.Version, Status: modelItem.Status,
		},
		Parameters: parameters,
	}, nil
}

func (s *TemplateModelParameterService) prepare(ctx context.Context, modelID int64, payloads []ModelParameterPayload) ([]*model.VideoTemplateModelParameter, error) {
	if modelID <= 0 {
		return nil, errors.New("关联模型无效")
	}
	baseItems, err := s.modelParams.ListByModel(ctx, modelID)
	if err != nil {
		return nil, err
	}
	baseByKey := make(map[string]model.VideoModelParameter, len(baseItems))
	for i := range baseItems {
		baseByKey[baseItems[i].ParamKey] = baseItems[i]
	}
	result := make([]*model.VideoTemplateModelParameter, 0, len(payloads))
	seen := make(map[string]struct{}, len(payloads))
	for i := range payloads {
		payload := payloads[i]
		if err := normalizeAndValidateTemplateParameterPayload(&payload); err != nil {
			return nil, fmt.Errorf("模型配置[%d]: %w", i, err)
		}
		if _, exists := seen[payload.ParamKey]; exists {
			return nil, fmt.Errorf("模型配置字段 %s 重复", payload.ParamKey)
		}
		seen[payload.ParamKey] = struct{}{}
		base, exists := baseByKey[payload.ParamKey]
		if !exists {
			return nil, fmt.Errorf("模型配置字段 %s 不属于所选模型", payload.ParamKey)
		}
		if payload.ValueType != base.ParamType || payload.ParameterType != base.ParameterType {
			return nil, fmt.Errorf("模型配置字段 %s 的值类型或参数类型与模型定义不一致", payload.ParamKey)
		}
		if payload.ParameterType == ParameterTypeOption {
			if err := ensureTemplateAllowedValuesAreSubset(base, payload.AllowedValues); err != nil {
				return nil, fmt.Errorf("模型配置字段 %s: %w", payload.ParamKey, err)
			}
		}
		item, err := buildTemplateModelParameter(modelID, &payload)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, nil
}

func normalizeAndValidateTemplateParameterPayload(req *ModelParameterPayload) error {
	if req == nil {
		return errors.New("配置不能为空")
	}
	req.ParamKey = strings.TrimSpace(req.ParamKey)
	req.ValueType = strings.ToLower(strings.TrimSpace(req.ValueType))
	req.Description = strings.TrimSpace(req.Description)
	if !parameterKeyPattern.MatchString(req.ParamKey) {
		return errors.New("参数键格式无效")
	}
	switch req.ValueType {
	case "string", "integer", "number", "boolean", "object", "array":
	default:
		return errors.New("参数值类型无效")
	}
	switch req.ParameterType {
	case ParameterTypeOption:
		if req.ValueType == "object" || req.ValueType == "array" {
			return errors.New("选项参数不支持 object 或 array")
		}
		if len(req.AllowedValues) == 0 || req.DefaultValue == nil {
			return errors.New("选项参数必须配置可选值和默认值")
		}
		seen := make(map[string]struct{}, len(req.AllowedValues))
		defaultFound := false
		for _, value := range req.AllowedValues {
			if err := validateParameterValue(req.ValueType, value); err != nil {
				return err
			}
			encoded, _ := json.Marshal(value)
			if _, exists := seen[string(encoded)]; exists {
				return errors.New("可选值不能重复")
			}
			seen[string(encoded)] = struct{}{}
			defaultFound = defaultFound || parameterValuesEqual(value, req.DefaultValue)
		}
		if err := validateParameterValue(req.ValueType, req.DefaultValue); err != nil {
			return err
		}
		if !defaultFound {
			return errors.New("默认值必须属于可选值列表")
		}
		req.IsRequired = 0
		req.Constraints = map[string]interface{}{}
	case ParameterTypeRequest:
		if len(req.Constraints) == 0 {
			return errors.New("请求参数必须配置限制条件")
		}
		if req.DefaultValue != nil {
			if err := validateParameterValue(req.ValueType, req.DefaultValue); err != nil {
				return err
			}
		}
		if err := validateParameterConstraints(req.Constraints); err != nil {
			return err
		}
		req.AllowedValues = []interface{}{}
	default:
		return errors.New("参数类型必须为 1 或 2")
	}
	return nil
}

func ensureTemplateAllowedValuesAreSubset(base model.VideoModelParameter, values []interface{}) error {
	var allowed []interface{}
	if strings.TrimSpace(base.AllowedValues) != "" {
		if err := json.Unmarshal([]byte(base.AllowedValues), &allowed); err != nil {
			return fmt.Errorf("模型可选值 JSON 无效: %w", err)
		}
	}
	allowedSet := make(map[string]struct{}, len(allowed))
	for _, value := range allowed {
		encoded, _ := json.Marshal(value)
		allowedSet[string(encoded)] = struct{}{}
	}
	for _, value := range values {
		encoded, _ := json.Marshal(value)
		if _, exists := allowedSet[string(encoded)]; !exists {
			return fmt.Errorf("值 %s 不在模型允许范围内", string(encoded))
		}
	}
	return nil
}

func buildTemplateModelParameter(modelID int64, req *ModelParameterPayload) (*model.VideoTemplateModelParameter, error) {
	defaultValue := ""
	if req.DefaultValue != nil {
		encoded, err := json.Marshal(req.DefaultValue)
		if err != nil {
			return nil, err
		}
		defaultValue = string(encoded)
	}
	allowedValues, err := json.Marshal(req.AllowedValues)
	if err != nil {
		return nil, err
	}
	constraints := ""
	if req.Constraints != nil {
		encoded, err := json.Marshal(req.Constraints)
		if err != nil {
			return nil, err
		}
		constraints = string(encoded)
	}
	return &model.VideoTemplateModelParameter{
		ModelID: modelID, ParamKey: req.ParamKey, ParamType: req.ValueType,
		IsRequired: req.IsRequired, DefaultValue: defaultValue, AllowedValues: string(allowedValues),
		Description: req.Description, SortOrder: req.SortOrder, ParameterType: req.ParameterType,
		Constraints: constraints,
	}, nil
}

func templateModelParameterView(item *model.VideoTemplateModelParameter) (TemplateModelParameterView, error) {
	var defaultValue interface{}
	if value := strings.TrimSpace(item.DefaultValue); value != "" {
		if err := json.Unmarshal([]byte(value), &defaultValue); err != nil {
			return TemplateModelParameterView{}, fmt.Errorf("参数 %s 的默认值 JSON 无效: %w", item.ParamKey, err)
		}
	}
	allowedValues := make([]interface{}, 0)
	if value := strings.TrimSpace(item.AllowedValues); value != "" {
		if err := json.Unmarshal([]byte(value), &allowedValues); err != nil {
			return TemplateModelParameterView{}, fmt.Errorf("参数 %s 的可选值 JSON 无效: %w", item.ParamKey, err)
		}
	}
	constraints := make(map[string]interface{})
	if value := strings.TrimSpace(item.Constraints); value != "" {
		if err := json.Unmarshal([]byte(value), &constraints); err != nil {
			return TemplateModelParameterView{}, fmt.Errorf("参数 %s 的限制 JSON 无效: %w", item.ParamKey, err)
		}
	}
	return TemplateModelParameterView{
		ID: item.ID, ModelID: item.ModelID, TemplateID: item.TemplateID,
		ParamKey: item.ParamKey, ValueType: item.ParamType, ParameterType: item.ParameterType,
		IsRequired: item.IsRequired, DefaultValue: defaultValue, AllowedValues: allowedValues,
		Constraints: constraints, Description: item.Description, SortOrder: item.SortOrder,
		CreatedAt: item.CreatedAt.Format("2006-01-02 15:04:05"), UpdatedAt: item.UpdatedAt.Format("2006-01-02 15:04:05"),
	}, nil
}

func templateModelParameterNotFound(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return errors.New("模板模型配置不存在")
	}
	return err
}

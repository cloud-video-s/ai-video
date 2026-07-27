package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"regexp"
	"strings"

	"ai-video/internal/gen/model"
	"ai-video/internal/repository"

	"gorm.io/gorm"
)

const (
	ParameterTypeOption  uint32 = 1
	ParameterTypeRequest uint32 = 2
)

var parameterKeyPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_.-]{0,63}$`)

type ModelParameterService struct {
	modelRepo *repository.ModelRepo
	repo      *repository.ModelParameterRepo
}

func NewModelParameterService() *ModelParameterService {
	return &ModelParameterService{modelRepo: repository.NewModelRepo(), repo: repository.NewModelParameterRepo()}
}

type ModelParameterPayload struct {
	ParamKey      string                 `json:"param_key" binding:"required,max=64"`
	ValueType     string                 `json:"value_type" binding:"required,oneof=string integer number boolean object array"`
	ParameterType uint32                 `json:"parameter_type" binding:"required,oneof=1 2"`
	IsRequired    uint32                 `json:"is_required" binding:"oneof=0 1"`
	DefaultValue  interface{}            `json:"default_value"`
	AllowedValues []interface{}          `json:"allowed_values"`
	Constraints   map[string]interface{} `json:"constraints"`
	Description   string                 `json:"description" binding:"max=255"`
	SortOrder     uint32                 `json:"sort_order"`
}

type ModelParameterView struct {
	ID            int64                  `json:"id"`
	ModelID       int64                  `json:"model_id"`
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

func (s *ModelParameterService) List(ctx context.Context, modelID int64) ([]ModelParameterView, error) {
	if _, err := s.modelRepo.GetByID(ctx, uint(modelID)); err != nil {
		return nil, notFoundOr(err, "模型不存在")
	}
	items, err := s.repo.ListByModel(ctx, modelID)
	if err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(items))
	for i := range items {
		ids = append(ids, items[i].ID)
	}
	constraints, err := s.repo.ConstraintsByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	result := make([]ModelParameterView, 0, len(items))
	for i := range items {
		view, err := modelParameterView(&items[i], constraints[items[i].ID])
		if err != nil {
			return nil, err
		}
		result = append(result, view)
	}
	return result, nil
}

func (s *ModelParameterService) Create(ctx context.Context, modelID int64, req *ModelParameterPayload) (*ModelParameterView, error) {
	if _, err := s.modelRepo.GetByID(ctx, uint(modelID)); err != nil {
		return nil, notFoundOr(err, "模型不存在")
	}
	if err := s.validatePayload(ctx, modelID, req, 0); err != nil {
		return nil, err
	}
	item, constraintsJSON, err := buildModelParameter(modelID, req)
	if err != nil {
		return nil, err
	}
	if err := repository.Transaction(ctx, func(txCtx context.Context) error {
		if err := s.repo.Create(txCtx, item); err != nil {
			return err
		}
		return s.repo.UpdateConstraints(txCtx, item.ID, constraintsJSON)
	}); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, errors.New("该模型下参数字段已存在")
		}
		return nil, err
	}
	view, err := modelParameterView(item, constraintsJSON)
	return &view, err
}

func (s *ModelParameterService) Update(ctx context.Context, modelID, id int64, req *ModelParameterPayload) (*ModelParameterView, error) {
	item, err := s.repo.GetByModelAndID(ctx, modelID, id)
	if err != nil {
		return nil, notFoundOr(err, "模型配置不存在")
	}
	if err := s.validatePayload(ctx, modelID, req, id); err != nil {
		return nil, err
	}
	updated, constraintsJSON, err := buildModelParameter(modelID, req)
	if err != nil {
		return nil, err
	}
	updated.ID = item.ID
	updated.CreatedAt = item.CreatedAt
	if err := repository.Transaction(ctx, func(txCtx context.Context) error {
		if err := s.repo.UpdateFields(txCtx, updated); err != nil {
			return err
		}
		return s.repo.UpdateConstraints(txCtx, updated.ID, constraintsJSON)
	}); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, errors.New("该模型下参数字段已存在")
		}
		return nil, err
	}
	view, err := modelParameterView(updated, constraintsJSON)
	return &view, err
}

func (s *ModelParameterService) Delete(ctx context.Context, modelID, id int64) error {
	item, err := s.repo.GetByModelAndID(ctx, modelID, id)
	if err != nil {
		return notFoundOr(err, "模型配置不存在")
	}
	// Generated model contains DeletedAt, therefore this is a soft delete.
	return s.repo.Delete(ctx, uint(item.ID))
}

func (s *ModelParameterService) validatePayload(ctx context.Context, modelID int64, req *ModelParameterPayload, currentID int64) error {
	req.ParamKey = strings.TrimSpace(req.ParamKey)
	req.ValueType = strings.ToLower(strings.TrimSpace(req.ValueType))
	req.Description = strings.TrimSpace(req.Description)
	if !parameterKeyPattern.MatchString(req.ParamKey) {
		return errors.New("参数字段必须以字母或下划线开头，且只能包含字母、数字、点、下划线和中划线")
	}
	if req.ParameterType == ParameterTypeOption {
		if req.ValueType == "object" || req.ValueType == "array" {
			return errors.New("选项参数的值类型仅支持 string、integer、number 或 boolean")
		}
		if len(req.AllowedValues) == 0 {
			return errors.New("选项参数至少需要一个选择值")
		}
		if req.DefaultValue == nil {
			return errors.New("选项参数必须指定一个默认选择值")
		}
		seen := make(map[string]struct{}, len(req.AllowedValues))
		defaultFound := false
		for _, value := range req.AllowedValues {
			if err := validateParameterValue(req.ValueType, value); err != nil {
				return fmt.Errorf("选择值类型错误: %w", err)
			}
			canonical, _ := json.Marshal(value)
			key := string(canonical)
			if _, ok := seen[key]; ok {
				return errors.New("选择值不能重复")
			}
			seen[key] = struct{}{}
			if parameterValuesEqual(value, req.DefaultValue) {
				defaultFound = true
			}
		}
		if err := validateParameterValue(req.ValueType, req.DefaultValue); err != nil {
			return fmt.Errorf("默认选择值类型错误: %w", err)
		}
		if !defaultFound {
			return errors.New("默认选择值必须属于选择值列表")
		}
		req.IsRequired = 0
		req.Constraints = map[string]interface{}{}
	} else {
		if len(req.Constraints) == 0 {
			return errors.New("请求参数必须填写限制条件")
		}
		if req.DefaultValue != nil {
			if err := validateParameterValue(req.ValueType, req.DefaultValue); err != nil {
				return fmt.Errorf("默认值类型错误: %w", err)
			}
		}
		if err := validateParameterConstraints(req.Constraints); err != nil {
			return err
		}
		req.AllowedValues = []interface{}{}
	}
	existing, err := s.repo.GetByKey(ctx, modelID, req.ParamKey)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if existing.ID != currentID {
		return errors.New("该模型下参数字段已存在")
	}
	return nil
}

func validateParameterValue(valueType string, value interface{}) error {
	switch valueType {
	case "string":
		if _, ok := value.(string); !ok {
			return errors.New("值必须是字符串")
		}
	case "integer":
		number, ok := value.(float64)
		if !ok || math.Trunc(number) != number {
			return errors.New("值必须是整数")
		}
	case "number":
		if _, ok := value.(float64); !ok {
			return errors.New("值必须是数字")
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return errors.New("值必须是布尔值")
		}
	case "object":
		if _, ok := value.(map[string]interface{}); !ok {
			return errors.New("值必须是 JSON 对象")
		}
	case "array":
		if _, ok := value.([]interface{}); !ok {
			return errors.New("值必须是 JSON 数组")
		}
	default:
		return errors.New("不支持的值类型")
	}
	return nil
}

func validateParameterConstraints(constraints map[string]interface{}) error {
	for _, key := range []string{"min", "max"} {
		if value, ok := constraints[key]; ok {
			if _, ok := value.(float64); !ok {
				return fmt.Errorf("限制条件 %s 必须是数字", key)
			}
		}
	}
	for _, key := range []string{"min_length", "max_length"} {
		if value, ok := constraints[key]; ok {
			number, ok := value.(float64)
			if !ok || number < 0 || math.Trunc(number) != number {
				return fmt.Errorf("限制条件 %s 必须是非负整数", key)
			}
		}
	}
	if pattern, ok := constraints["pattern"]; ok {
		value, ok := pattern.(string)
		if !ok {
			return errors.New("限制条件 pattern 必须是字符串")
		}
		if _, err := regexp.Compile(value); err != nil {
			return errors.New("限制条件 pattern 不是有效的正则表达式")
		}
	}
	if min, minOK := constraints["min"].(float64); minOK {
		if max, maxOK := constraints["max"].(float64); maxOK && min > max {
			return errors.New("限制条件 min 不能大于 max")
		}
	}
	if min, minOK := constraints["min_length"].(float64); minOK {
		if max, maxOK := constraints["max_length"].(float64); maxOK && min > max {
			return errors.New("限制条件 min_length 不能大于 max_length")
		}
	}
	return nil
}

func buildModelParameter(modelID int64, req *ModelParameterPayload) (*model.VideoModelParameter, string, error) {
	defaultJSON := ""
	if req.DefaultValue != nil {
		encoded, err := json.Marshal(req.DefaultValue)
		if err != nil {
			return nil, "", err
		}
		defaultJSON = string(encoded)
	}
	allowedJSON, err := json.Marshal(req.AllowedValues)
	if err != nil {
		return nil, "", err
	}
	constraintsJSON, err := json.Marshal(req.Constraints)
	if err != nil {
		return nil, "", err
	}
	return &model.VideoModelParameter{
		ModelID: modelID, ParamKey: req.ParamKey, ParamType: req.ValueType,
		IsRequired: req.IsRequired, DefaultValue: defaultJSON, AllowedValues: string(allowedJSON),
		Description: req.Description, SortOrder: req.SortOrder, ParameterType: req.ParameterType,
	}, string(constraintsJSON), nil
}

func modelParameterView(item *model.VideoModelParameter, constraintsJSON string) (ModelParameterView, error) {
	var defaultValue interface{}
	if item.DefaultValue != "" {
		if err := json.Unmarshal([]byte(item.DefaultValue), &defaultValue); err != nil {
			return ModelParameterView{}, fmt.Errorf("参数 %s 的默认值 JSON 无效: %w", item.ParamKey, err)
		}
	}
	allowedValues := make([]interface{}, 0)
	if item.AllowedValues != "" {
		if err := json.Unmarshal([]byte(item.AllowedValues), &allowedValues); err != nil {
			return ModelParameterView{}, fmt.Errorf("参数 %s 的选择值 JSON 无效: %w", item.ParamKey, err)
		}
	}
	constraints := make(map[string]interface{})
	if constraintsJSON != "" {
		if err := json.Unmarshal([]byte(constraintsJSON), &constraints); err != nil {
			return ModelParameterView{}, fmt.Errorf("参数 %s 的限制条件 JSON 无效: %w", item.ParamKey, err)
		}
	}
	return ModelParameterView{
		ID: item.ID, ModelID: item.ModelID, ParamKey: item.ParamKey, ValueType: item.ParamType,
		ParameterType: item.ParameterType, IsRequired: item.IsRequired, DefaultValue: defaultValue,
		AllowedValues: allowedValues, Constraints: constraints, Description: item.Description,
		SortOrder: item.SortOrder, CreatedAt: item.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt: item.UpdatedAt.Format("2006-01-02 15:04:05"),
	}, nil
}

func parameterValuesEqual(left, right interface{}) bool {
	leftJSON, leftErr := json.Marshal(left)
	rightJSON, rightErr := json.Marshal(right)
	return leftErr == nil && rightErr == nil && string(leftJSON) == string(rightJSON)
}

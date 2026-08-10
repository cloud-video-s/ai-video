package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"ai-video/internal/gen/model"
	"ai-video/internal/modelparameter"
)

type ClientTemplateModelParameter struct {
	ParamKey            string                       `json:"param_key"`
	ValueType           string                       `json:"value_type"`
	ParameterType       uint32                       `json:"parameter_type"`
	IsRequired          uint32                       `json:"is_required"`
	DefaultValue        interface{}                  `json:"default_value"`
	AllowedValues       []interface{}                `json:"allowed_values"`
	AllowedValueOptions []modelparameter.ValueOption `json:"allowed_value_options"`
	Constraints         map[string]interface{}       `json:"constraints"`
	Description         string                       `json:"description"`
	SortOrder           uint32                       `json:"sort_order"`
	Alias               string                       `json:"alias"`
	DisplayType         string                       `json:"display_type"`
	IsDisplay           uint32                       `json:"is_display"`
}

type clientTemplateModelConfiguration struct {
	ModelID    uint64
	ModelCode  string
	ModelName  string
	ModelScore int64
	Parameters []ClientTemplateModelParameter
}

func (s *ClientTemplateService) loadTemplateModelConfigurations(ctx context.Context, templates []model.VideoTemplate) (map[uint64]clientTemplateModelConfiguration, error) {
	result := make(map[uint64]clientTemplateModelConfiguration, len(templates))
	if len(templates) == 0 {
		return result, nil
	}
	modelIDs := make([]int64, 0, len(templates))
	templateIDs := make([]uint64, 0, len(templates))
	modelIDByTemplate := make(map[uint64]int64, len(templates))
	seenModels := make(map[int64]struct{}, len(templates))
	for i := range templates {
		if templates[i].ModelID == 0 || templates[i].ModelID > math.MaxInt64 {
			return nil, fmt.Errorf("template %d has invalid model_id", templates[i].ID)
		}
		modelID := int64(templates[i].ModelID)
		if _, exists := seenModels[modelID]; !exists {
			seenModels[modelID] = struct{}{}
			modelIDs = append(modelIDs, modelID)
		}
		templateIDs = append(templateIDs, templates[i].ID)
		modelIDByTemplate[templates[i].ID] = modelID
	}
	models, err := s.modelRepo.ListEnabledByIDs(ctx, modelIDs)
	if err != nil {
		return nil, err
	}
	modelsByID := make(map[int64]model.VideoModel, len(models))
	for i := range models {
		modelsByID[models[i].ID] = models[i]
	}
	parameters, err := s.templateParameterRepo.ListByTemplates(ctx, templateIDs)
	if err != nil {
		return nil, err
	}
	definitions, err := s.modelParameterRepo.ListByModels(ctx, modelIDs)
	if err != nil {
		return nil, err
	}
	definitionsByKey := make(map[string]*model.VideoModelParameter, len(definitions))
	for i := range definitions {
		definitionsByKey[modelParameterDefinitionKey(definitions[i].ModelID, definitions[i].ParamKey)] = &definitions[i]
	}
	parametersByTemplate := make(map[uint64][]ClientTemplateModelParameter, len(templates))
	for i := range parameters {
		expectedModelID, exists := modelIDByTemplate[parameters[i].TemplateID]
		if !exists || parameters[i].ModelID != expectedModelID {
			return nil, fmt.Errorf("template %d parameter %s has mismatched model_id", parameters[i].TemplateID, parameters[i].ParamKey)
		}
		definition := definitionsByKey[modelParameterDefinitionKey(parameters[i].ModelID, parameters[i].ParamKey)]
		if definition == nil {
			return nil, fmt.Errorf("template %d parameter %s has no model definition", parameters[i].TemplateID, parameters[i].ParamKey)
		}
		view, err := clientTemplateModelParameterView(&parameters[i], definition)
		if err != nil {
			return nil, err
		}
		parametersByTemplate[parameters[i].TemplateID] = append(parametersByTemplate[parameters[i].TemplateID], view)
	}
	for i := range templates {
		item := templates[i]
		modelItem, exists := modelsByID[int64(item.ModelID)]
		if !exists {
			continue
		}
		if item.TemplateType != int64(modelItem.ModelType) {
			continue
		}
		items := parametersByTemplate[item.ID]
		if items == nil {
			items = []ClientTemplateModelParameter{}
		}
		result[item.ID] = clientTemplateModelConfiguration{
			ModelID: item.ModelID, ModelCode: modelItem.Code, ModelName: modelItem.Name,
			ModelScore: modelItem.Score, Parameters: items,
		}
	}
	return result, nil
}

func clientTemplateModelParameterView(item *model.VideoTemplateModelParameter, definition *model.VideoModelParameter) (ClientTemplateModelParameter, error) {
	var defaultValue interface{}
	if value := strings.TrimSpace(item.DefaultValue); value != "" {
		if err := json.Unmarshal([]byte(value), &defaultValue); err != nil {
			return ClientTemplateModelParameter{}, fmt.Errorf("template %d parameter %s has invalid default_value JSON: %w", item.TemplateID, item.ParamKey, err)
		}
	}
	allowedValues := make([]interface{}, 0)
	if value := strings.TrimSpace(item.AllowedValues); value != "" {
		if err := json.Unmarshal([]byte(value), &allowedValues); err != nil {
			return ClientTemplateModelParameter{}, fmt.Errorf("template %d parameter %s has invalid allowed_values JSON: %w", item.TemplateID, item.ParamKey, err)
		}
		if allowedValues == nil {
			allowedValues = []interface{}{}
		}
	}
	allowedValueOptions, err := templateAllowedValueOptions(definition, allowedValues)
	if err != nil {
		return ClientTemplateModelParameter{}, fmt.Errorf("template %d parameter %s has invalid allowed value options: %w", item.TemplateID, item.ParamKey, err)
	}
	constraints := make(map[string]interface{})
	if value := strings.TrimSpace(item.Constraints); value != "" {
		if err := json.Unmarshal([]byte(value), &constraints); err != nil {
			return ClientTemplateModelParameter{}, fmt.Errorf("template %d parameter %s has invalid constraints JSON: %w", item.TemplateID, item.ParamKey, err)
		}
		if constraints == nil {
			constraints = map[string]interface{}{}
		}
	}
	return ClientTemplateModelParameter{
		ParamKey: item.ParamKey, ValueType: item.ParamType, ParameterType: item.ParameterType,
		IsRequired: item.IsRequired, DefaultValue: defaultValue, AllowedValues: allowedValues, AllowedValueOptions: allowedValueOptions,
		Constraints: constraints, Description: item.Description, SortOrder: item.SortOrder,
		Alias: definition.Alias_, DisplayType: definition.DisplayType, IsDisplay: definition.IsDisplay,
	}, nil
}

func modelParameterDefinitionKey(modelID int64, paramKey string) string {
	return fmt.Sprintf("%d\x00%s", modelID, paramKey)
}

func templateAllowedValueOptions(definition *model.VideoModelParameter, allowedValues []interface{}) ([]modelparameter.ValueOption, error) {
	if definition == nil {
		return nil, fmt.Errorf("model parameter definition is required")
	}
	var modelAllowedValues []interface{}
	if value := strings.TrimSpace(definition.AllowedValues); value != "" {
		if err := json.Unmarshal([]byte(value), &modelAllowedValues); err != nil {
			return nil, fmt.Errorf("model allowed_values is invalid JSON: %w", err)
		}
	}
	modelOptions, err := modelparameter.ValueOptions(definition.AllowedValueOptions, modelAllowedValues)
	if err != nil {
		return nil, err
	}
	optionByValue := make(map[string]modelparameter.ValueOption, len(modelOptions))
	for i := range modelOptions {
		encoded, _ := json.Marshal(modelOptions[i].Value)
		optionByValue[string(encoded)] = modelOptions[i]
	}
	result := make([]modelparameter.ValueOption, len(allowedValues))
	for i := range allowedValues {
		encoded, _ := json.Marshal(allowedValues[i])
		option, exists := optionByValue[string(encoded)]
		if !exists {
			return nil, fmt.Errorf("value %s is not in the model allowed values", string(encoded))
		}
		result[i] = option
	}
	return result, nil
}

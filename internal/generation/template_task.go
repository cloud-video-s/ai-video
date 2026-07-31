package generation

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"ai-video/internal/gen/model"

	"gorm.io/gorm"
)

var ErrTemplateUnavailable = errors.New("template does not exist or is disabled")

// CreateTemplateTask resolves all server-owned generation settings from a
// template and then uses the normal durable task creation path. This keeps
// provider submission, polling, result storage, and idempotency identical to
// tasks created by explicitly selecting a model.
func (m *Manager) CreateTemplateTask(
	ctx context.Context,
	userID uint64,
	request *CreateTemplateTaskRequest,
) (*model.VideoUserGenerationTask, error) {
	if request == nil || request.TemplateID == 0 {
		return nil, errors.New("template_id is required")
	}
	if m.templateRepo == nil || m.templateParameterRepo == nil {
		return nil, errors.New("template generation is not configured")
	}
	template, err := m.templateRepo.GetEnabledByID(ctx, request.TemplateID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTemplateUnavailable
		}
		return nil, err
	}

	taskType, err := templateTaskType(template.TemplateType)
	if err != nil {
		return nil, fmt.Errorf("template %d: %w", template.ID, err)
	}
	if template.ModelID == 0 {
		return nil, fmt.Errorf("template %d has an invalid model_id", template.ID)
	}
	modelConfig, err := m.modelRepo.GetEnabledByID(ctx, int64(template.ModelID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("template %d model does not exist or is disabled", template.ID)
		}
		return nil, err
	}
	if modelConfig.ModelType != taskType {
		return nil, fmt.Errorf(
			"template %d type %d does not match model %s type %d",
			template.ID, taskType, modelConfig.Code, modelConfig.ModelType,
		)
	}

	configured, err := m.templateParameterRepo.ListByTemplate(ctx, template.ID)
	if err != nil {
		return nil, err
	}
	parameters, err := mergeTemplateParameters(template, configured, request.Parameters)
	if err != nil {
		return nil, err
	}

	prompt := strings.TrimSpace(template.Prompt)
	if prompt == "" {
		return nil, fmt.Errorf("template %d prompt is empty", template.ID)
	}
	input := cloneMap(request.Input)
	input["prompt"] = prompt
	fromMap, err := generationInputFromMap(taskType, input)
	if err != nil {
		return nil, err
	}
	if len(fromMap.Images) == 0 {
		return nil, fmt.Errorf("template %d has no images", template.ID)
	}

	if taskType == TaskTypeImage {
		fromMap.Images = append(fromMap.Images, template.OriginalURL)
		input["images"] = fromMap.Images
	} else {
		input["video"] = template.OriginalURL
	}

	return m.CreateTask(ctx, userID, &CreateTaskRequest{
		ModelCode:       modelConfig.Code,
		TaskType:        taskType,
		ClientRequestID: request.ClientRequestID,
		Input:           input,
		Parameters:      parameters,
		TemplateID:      template.ID,
	})
}

func templateTaskType(value int64) (uint32, error) {
	switch value {
	case int64(TaskTypeImage):
		return TaskTypeImage, nil
	case int64(TaskTypeVideo):
		return TaskTypeVideo, nil
	default:
		return 0, fmt.Errorf("unsupported template_type %d", value)
	}
}

func mergeTemplateParameters(
	template *model.VideoTemplate,
	configured []model.VideoTemplateModelParameter,
	request map[string]any,
) (map[string]any, error) {
	definitions := make([]model.VideoModelParameter, 0, len(configured))
	for i := range configured {
		item := configured[i]
		if item.ModelID != int64(template.ModelID) {
			return nil, fmt.Errorf(
				"template %d parameter %s belongs to model %d instead of %d",
				template.ID, item.ParamKey, item.ModelID, template.ModelID,
			)
		}
		definitions = append(definitions, model.VideoModelParameter{
			ModelID:       item.ModelID,
			ParamKey:      item.ParamKey,
			ParamType:     item.ParamType,
			IsRequired:    item.IsRequired,
			DefaultValue:  item.DefaultValue,
			AllowedValues: item.AllowedValues,
			Description:   item.Description,
			SortOrder:     item.SortOrder,
			ParameterType: item.ParameterType,
			Constraints:   item.Constraints,
		})
	}
	parameters, err := mergeConfiguredParameters(definitions, request)
	if err != nil {
		return nil, fmt.Errorf("template %d parameters: %w", template.ID, err)
	}
	return parameters, nil
}

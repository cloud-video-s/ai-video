package generation

import (
	"errors"
	"fmt"
	"strings"

	"ai-video/internal/domain"
	"ai-video/internal/gen/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var ErrToolUnavailable = errors.New("tool does not exist or is disabled")

// CreateToolTask resolves all server-owned generation settings from an online
// tool and queues the task through the normal durable generation path.
func (m *Manager) CreateToolTask(ctx *gin.Context, userID uint64, request *CreateToolTaskRequest) (*model.VideoUserGenerationTask, error) {
	if request == nil || request.ToolID == 0 {
		return nil, errors.New("tool_id is required")
	}
	if m.toolRepo == nil {
		return nil, errors.New("tool generation is not configured")
	}
	tool, err := m.toolRepo.GetEnabledByID(ctx, request.ToolID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrToolUnavailable
		}
		return nil, err
	}
	modelConfig, err := m.modelRepo.GetEnabledByID(ctx, tool.ModelID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("tool %d model does not exist or is disabled", tool.ID)
		}
		return nil, err
	}
	createRequest, err := buildToolCreateTaskRequest(tool, modelConfig, request)
	if err != nil {
		return nil, err
	}
	return m.CreateTask(ctx, userID, createRequest)
}

func buildToolCreateTaskRequest(tool *model.VideoToolConfig, modelConfig *model.VideoModel, request *CreateToolTaskRequest) (*CreateTaskRequest, error) {
	if tool == nil || tool.ID == 0 {
		return nil, ErrToolUnavailable
	}
	if request == nil {
		return nil, errors.New("tool request is required")
	}
	if request.ToolID != tool.ID {
		return nil, fmt.Errorf("requested tool %d does not match loaded tool %d", request.ToolID, tool.ID)
	}
	if request.ConfigType < uint64(domain.ToolConfigTypeNone) || request.ConfigType > uint64(domain.ToolConfigTypeRatio) {
		return nil, fmt.Errorf("unsupported config_type %d", request.ConfigType)
	}
	if request.ConfigType != uint64(tool.ConfigType) {
		return nil, fmt.Errorf("config_type %d does not match tool %d config_type %d", request.ConfigType, tool.ID, tool.ConfigType)
	}
	if modelConfig == nil || modelConfig.ID == 0 || strings.TrimSpace(modelConfig.Code) == "" {
		return nil, fmt.Errorf("tool %d has an invalid model", tool.ID)
	}
	if tool.ModelID != modelConfig.ID {
		return nil, fmt.Errorf("tool %d is configured for model %d instead of %d", tool.ID, tool.ModelID, modelConfig.ID)
	}
	taskType := uint32(tool.ToolType)
	if taskType != TaskTypeImage && taskType != TaskTypeVideo {
		return nil, fmt.Errorf("tool %d has unsupported tool_type %d", tool.ID, tool.ToolType)
	}
	if modelConfig.ModelType != taskType {
		return nil, fmt.Errorf(
			"tool %d type %d does not match model %s type %d",
			tool.ID, taskType, modelConfig.Code, modelConfig.ModelType,
		)
	}
	input := map[string]any{}
	prompt := strings.TrimSpace(tool.Prompt)
	if prompt == "" {
		return nil, fmt.Errorf("tool %d prompt is empty", tool.ID)
	}
	image := strings.TrimSpace(request.Image)
	video := strings.TrimSpace(request.Video)
	switch request.ConfigType {
	case uint64(domain.ToolConfigTypeNone):
		if image == "" {
			return nil, fmt.Errorf("tool %d has an empty image", tool.ID)
		}
		input["images"] = []string{image}
	case uint64(domain.ToolConfigTypeReferenceImage):
		if image == "" {
			return nil, fmt.Errorf("tool %d has an empty image", tool.ID)
		}
		image2 := strings.TrimSpace(request.Val)
		if image2 == "" {
			return nil, fmt.Errorf("tool %d reference image %s is required", tool.ID, request.Image)
		}
		input["images"] = []string{image, image2}
	case uint64(domain.ToolConfigTypeAge):
		if image == "" {
			return nil, fmt.Errorf("tool %d has an empty image", tool.ID)
		}
		if request.Val == "" {
			return nil, fmt.Errorf("tool %d config value is required", tool.ID)
		}
		prompt = BuildReplacer(map[string]string{"age": strings.TrimSpace(request.Val)}, prompt)
		input["images"] = []string{image}
	case uint64(domain.ToolConfigTypeRatio):
		if request.Val == "" {
			return nil, fmt.Errorf("tool %d config value is required", tool.ID)
		}
		prompt = BuildReplacer(map[string]string{"scale": strings.TrimSpace(request.Val)}, prompt)
	}
	input["prompt"] = prompt
	if video != "" {
		input["video"] = video
	}
	return &CreateTaskRequest{
		ModelCode:       strings.TrimSpace(modelConfig.Code),
		TaskType:        taskType,
		ClientRequestID: strings.TrimSpace(request.ClientRequestID),
		Input:           input,
		ToolConfigID:    tool.ID,
		SourceType:      TaskSourceTool,
	}, nil
}

// BuildReplacer 根据参数生成 strings.Replacer
func BuildReplacer(params map[string]string, str string) string {
	oldNew := make([]string, 0, len(params)*2)
	for k, v := range params {
		oldNew = append(oldNew, "{"+k+"}", v)
	}
	replacer := strings.NewReplacer(oldNew...)
	return replacer.Replace(str)
}

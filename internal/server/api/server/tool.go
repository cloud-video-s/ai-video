package service

import (
	"ai-video/internal/gen/model"
	"context"
	"encoding/json"

	"ai-video/internal/repository"
)

type ClientToolService struct {
	toolRepo *repository.ToolConfigRepo
}

func NewClientToolService() *ClientToolService {
	return &ClientToolService{toolRepo: repository.NewToolConfigRepo()}
}

// ClientTool contains the fields required to display and invoke a tool.
type ClientTool struct {
	ID              uint64          `json:"id"`
	Name            string          `json:"name"`
	Icon            string          `json:"icon"`
	BackgroundImage string          `json:"background_image"`
	ToolType        uint8           `json:"tool_type"`
	ToolsType       string          `json:"tools_type"`
	ModelID         int64           `json:"model_id"`
	ConfigType      uint8           `json:"config_type"`
	ConfigData      json.RawMessage `json:"config_data"`
	BadgeImage      string          `json:"badge_image"`
	Sort            int64           `json:"sort"`
	Prompt          string          `json:"prompt"`
	Status          int8            `json:"status"`
	ModelScore      int64           `json:"model_score"`
}

func (s *ClientToolService) List(ctx context.Context) ([]ClientTool, error) {
	items, err := s.toolRepo.ListForClient(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]ClientTool, 0, len(items))
	for i := range items {
		result = append(result, mapClientTool(items[i]))
	}
	return result, nil
}

func mapClientTool(item *model.VideoToolConfig) ClientTool {
	return ClientTool{
		ID: item.ID, Name: item.Name, Icon: item.Icon,
		BackgroundImage: item.BackgroundImage, ToolType: item.ToolType,
		ToolsType: item.ToolsType, ModelID: item.ModelID, ConfigType: item.ConfigType,
		ConfigData: append(json.RawMessage(nil), item.ConfigData...),
		BadgeImage: item.BadgeImage, Sort: item.Sort, Prompt: item.Prompt,
		Status: item.Status, ModelScore: item.Model.Score,
	}
}

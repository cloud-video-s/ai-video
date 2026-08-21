package domain

import (
	"encoding/json"
	"time"
)

const (
	ToolTypeImageGeneration uint8 = 1
	ToolTypeVideoGeneration uint8 = 2

	ToolConfigTypeNone           uint8 = 0
	ToolConfigTypeReferenceImage uint8 = 1
	ToolConfigTypeAge            uint8 = 2
	ToolConfigTypeRatio          uint8 = 3
)

// ToolConfig describes a client tool entry managed by the admin console.
// Persistence-specific metadata stays in the repository layer.
type ToolConfig struct {
	ID              uint64          `json:"id"`
	Name            string          `json:"name"`
	Icon            string          `json:"icon"`
	BackgroundImage string          `json:"background_image"`
	ToolType        uint8           `json:"tool_type"`
	ModelID         int64           `json:"model_id"`
	ModelName       string          `json:"model_name"`
	ConfigType      uint8           `json:"config_type"`
	ConfigData      json.RawMessage `json:"config_data"`
	BadgeImage      string          `json:"badge_image"`
	Sort            int64           `json:"sort"`
	Prompt          string          `json:"prompt"`
	Status          int8            `json:"status"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

type ToolModelOption struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	ModelType uint8  `json:"model_type"`
}

package model

// VideoTemplateDisplayConfig assigns one concrete template to a reusable
// client display position. Its sort and status are independent from the
// template itself so every position can be curated separately.
type VideoTemplateDisplayConfig struct {
	ID                 uint64 `json:"id" gorm:"primaryKey;autoIncrement"`
	TemplateID         uint64 `json:"template_id" gorm:"not null;index;uniqueIndex:uk_template_display_config,priority:1"`
	DisplayPositionKey string `json:"position_key" gorm:"column:position_key;size:64;not null;index;uniqueIndex:uk_template_display_config,priority:2"`
	Sort               int    `json:"sort" gorm:"not null;default:0;index;comment:position-specific sort order"`
	Status             int8   `json:"status" gorm:"not null;default:1;index;comment:status: 0 disabled, 1 enabled"`
	Remark             string `json:"remark" gorm:"size:500;comment:configuration remark"`
	BaseModel

	Template        VideoTemplate        `json:"template" gorm:"foreignKey:TemplateID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	DisplayPosition VideoDisplayPosition `json:"display_position" gorm:"foreignKey:DisplayPositionKey;references:PositionKey;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
}

func (VideoTemplateDisplayConfig) TableName() string {
	return "video_template_display_config"
}

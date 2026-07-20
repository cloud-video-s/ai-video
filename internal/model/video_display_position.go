package model

// VideoDisplayPosition defines a reusable placement where video templates can
// be surfaced in the client application.
type VideoDisplayPosition struct {
	ID           uint64 `json:"id" gorm:"primaryKey;autoIncrement"`
	PositionName string `json:"position_name" gorm:"size:128;not null;index;comment:display position name"`
	PositionKey  string `json:"position_key" gorm:"size:64;not null;uniqueIndex;comment:unique display position identifier"`
	Description  string `json:"description" gorm:"size:500;comment:display position description"`
	CoverImage   string `json:"cover_image" gorm:"size:1024;not null;comment:cover image URL"`
	Sort         int    `json:"sort" gorm:"not null;default:0;index;comment:sort order"`
	Status       int8   `json:"status" gorm:"not null;default:1;index;comment:status: 0 disabled, 1 enabled"`
	BaseModel
}

func (VideoDisplayPosition) TableName() string {
	return "video_display_position"
}

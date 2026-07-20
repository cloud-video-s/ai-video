package model

// VideoBannerDisplayPosition associates a banner with a stable display
// position key. A position key is used instead of the display-position ID so
// client placement lookups do not need to translate the key before filtering.
type VideoBannerDisplayPosition struct {
	BannerID    uint64 `json:"banner_id" gorm:"type:bigint unsigned;primaryKey;autoIncrement:false"`
	PositionKey string `json:"position_key" gorm:"size:64;primaryKey;autoIncrement:false;index"`
}

func (VideoBannerDisplayPosition) TableName() string {
	return "video_banner_display_position"
}

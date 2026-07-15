package model

type VideoDelayConfig struct {
	ID      uint   `json:"id" gorm:"primaryKey"`
	Group   string `json:"group" gorm:"size:64;index;not null;comment:config group"`
	Key     string `json:"key" gorm:"size:128;uniqueIndex;not null;comment:config key"`
	Value   string `json:"value" gorm:"size:64;not null;comment:config value"`
	Type    string `json:"type" gorm:"size:16;not null;default:string;comment:string/int/bool"`
	Options string `json:"options" gorm:"size:255;comment:allowed values"`
	Remark  string `json:"remark" gorm:"size:255;comment:description"`
	Sort    int    `json:"sort" gorm:"default:0"`
	BaseModel
}

func (VideoDelayConfig) TableName() string {
	return "video_delay_config"
}

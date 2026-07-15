package model

type VideoAPI struct {
	ID          uint   `json:"id" gorm:"primaryKey"`
	Path        string `json:"path" gorm:"size:255;not null"`
	Method      string `json:"method" gorm:"size:16;not null"`
	Group       string `json:"group" gorm:"size:64"`
	Description string `json:"description" gorm:"size:255"`
	BaseModel
}

func (VideoAPI) TableName() string {
	return "video_api"
}

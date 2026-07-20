package model

// VideoCountry stores the ISO 3166-1 alpha-2 country/region reference data.
type VideoCountry struct {
	ID     uint64 `json:"id" gorm:"primaryKey;autoIncrement"`
	Code   string `json:"code" gorm:"size:2;not null;uniqueIndex;comment:ISO 3166-1 alpha-2 code"`
	NameZh string `json:"name_zh" gorm:"size:100;not null;index;comment:Chinese name"`
	Status int8   `json:"status" gorm:"not null;default:1;index;comment:status: 0 disabled, 1 enabled"`
	BaseModel
}

func (VideoCountry) TableName() string {
	return "video_country"
}

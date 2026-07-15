package model

const SuperAdminRoleCode = "admin"

type VideoRole struct {
	ID     uint        `json:"id" gorm:"primaryKey"`
	Name   string      `json:"name" gorm:"size:64;not null"`
	Code   string      `json:"code" gorm:"uniqueIndex;size:64;not null"`
	Sort   int         `json:"sort" gorm:"default:0"`
	Status int8        `json:"status" gorm:"default:1;comment:1-正常 0-禁用"`
	Remark string      `json:"remark" gorm:"size:255"`
	Menus  []VideoMenu `json:"menus,omitempty" gorm:"many2many:video_role_menu;"`
	BaseModel
}

func (VideoRole) TableName() string {
	return "video_role"
}

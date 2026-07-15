package model

type VideoConfig struct {
	ID       uint   `json:"id" gorm:"primaryKey"`
	Group    string `json:"group" gorm:"size:64;index;comment:分组"`
	Key      string `json:"key" gorm:"size:128;uniqueIndex;not null;comment:配置键"`
	Name     string `json:"name" gorm:"size:128;comment:显示名"`
	Value    string `json:"value" gorm:"type:text;comment:值(统一字符串存储)"`
	Type     string `json:"type" gorm:"size:16;default:string;comment:string/int/bool/float/json/text/select"`
	Options  string `json:"options" gorm:"type:text;comment:select 选项或校验规则(JSON)"`
	IsPublic bool   `json:"is_public" gorm:"default:false;comment:是否免鉴权可读"`
	Editable bool   `json:"editable" gorm:"default:true;comment:是否允许后台编辑"`
	Builtin  bool   `json:"builtin" gorm:"default:false;comment:内置(不可删除)"`
	Sort     int    `json:"sort" gorm:"default:0"`
	Remark   string `json:"remark" gorm:"size:255"`
	BaseModel
}

func (VideoConfig) TableName() string {
	return "video_config"
}

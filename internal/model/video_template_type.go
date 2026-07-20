package model

// VideoTemplateType defines a category and its client delivery conditions.
type VideoTemplateType struct {
	ID                   uint64   `json:"id" gorm:"primaryKey;autoIncrement"`
	CategoryName         string   `json:"category_name" gorm:"size:128;not null;index;comment:category name"`
	Sort                 int64    `json:"sort" gorm:"not null;default:0;index;comment:sort order"`
	Status               int8     `json:"status" gorm:"not null;default:1;index;comment:status: 0 disabled, 1 enabled"`
	Description          string   `json:"description" gorm:"size:500;comment:description"`
	UserTypes            []int    `json:"user_types" gorm:"type:text;serializer:json;comment:target user types: 1 free, 2 paid"`
	SubscriptionStatuses []string `json:"subscription_statuses" gorm:"type:text;serializer:json;comment:subscribed and/or unsubscribed"`
	BaseModel

	DisplayPositions []VideoDisplayPosition `json:"display_positions" gorm:"many2many:video_template_type_display_position;foreignKey:ID;joinForeignKey:TemplateTypeID;references:PositionKey;joinReferences:PositionKey;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
	Countries        []VideoCountry         `json:"countries" gorm:"many2many:video_template_type_country;joinForeignKey:TemplateTypeID;joinReferences:CountryID"`
	Channels         []VideoChannel         `json:"channels" gorm:"many2many:video_template_type_channel;joinForeignKey:TemplateTypeID;joinReferences:ChannelID"`
	Packages         []VideoPackage         `json:"packages" gorm:"many2many:video_template_type_package;joinForeignKey:TemplateTypeID;joinReferences:PackageID"`

	LegacyCountry      string  `json:"-" gorm:"column:country;size:8;not null;default:'';index"`
	LegacyAppPackage   string  `json:"-" gorm:"column:app_package;size:255;not null;default:'';index"`
	LegacyChannelID    string  `json:"-" gorm:"column:channel_id;size:64;not null;default:'';index"`
	LegacyUserType     uint32  `json:"-" gorm:"column:user_type;type:tinyint unsigned;not null;default:0;index"`
	LegacyIsSubscribed bool    `json:"-" gorm:"column:is_subscribed;not null;default:false;index"`
	LegacyPackageID    *uint64 `json:"-" gorm:"column:package_id;index"`
}

func (VideoTemplateType) TableName() string {
	return "video_template_type"
}

type VideoTemplateTypeDisplayPosition struct {
	TemplateTypeID uint64 `json:"template_type_id" gorm:"primaryKey"`
	PositionKey    string `json:"position_key" gorm:"size:64;primaryKey"`
}

func (VideoTemplateTypeDisplayPosition) TableName() string {
	return "video_template_type_display_position"
}

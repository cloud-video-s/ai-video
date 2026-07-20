package model

const (
	VideoTemplateKindAction   = "action"
	VideoTemplateKindFaceSwap = "face_swap"
)

// VideoTemplate stores a video template and its media resources.
type VideoTemplate struct {
	ID                   uint64   `json:"id" gorm:"primaryKey;autoIncrement"`
	VideoTemplateTypeID  uint64   `json:"video_template_type_id" gorm:"not null;index;comment:video template type ID"`
	UserTypes            []int    `json:"user_types" gorm:"type:text;serializer:json;comment:target user types: 1 free, 2 paid"`
	SubscriptionStatuses []string `json:"subscription_statuses" gorm:"type:text;serializer:json;comment:subscribed and/or unsubscribed"`
	Name                 string   `json:"name" gorm:"size:128;not null;index;comment:template name"`
	TemplateType         string   `json:"template_type" gorm:"size:32;not null;index;comment:template kind, such as action or face_swap"`
	Sort                 int      `json:"sort" gorm:"not null;default:0;index;comment:sort order"`
	UsageCount           uint64   `json:"usage_count" gorm:"not null;default:0;index;comment:template usage count"`
	FavoriteCount        uint64   `json:"favorite_count" gorm:"not null;default:0;index;comment:template favorite count"`
	ViewCount            uint64   `json:"view_count" gorm:"not null;default:0;index;comment:template view count"`
	CoverImage           string   `json:"cover_image" gorm:"size:1024;not null;comment:cover image URL"`
	TemplateVideo        string   `json:"template_video" gorm:"size:1024;not null;comment:template video URL"`
	ThumbnailVideo       string   `json:"thumbnail_video" gorm:"size:1024;comment:thumbnail video URL"`
	Prompt               string   `json:"prompt" gorm:"type:text;comment:template prompt"`
	Status               int8     `json:"status" gorm:"not null;default:1;index;comment:status: 0 disabled, 1 enabled"`
	Description          string   `json:"description" gorm:"size:500;comment:description"`
	BaseModel

	VideoTemplateType VideoTemplateType `json:"video_template_type,omitempty" gorm:"foreignKey:VideoTemplateTypeID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
	Countries         []VideoCountry    `json:"countries" gorm:"many2many:video_template_country;joinForeignKey:TemplateID;joinReferences:CountryID"`
	Packages          []VideoPackage    `json:"packages" gorm:"many2many:video_template_package;joinForeignKey:TemplateID;joinReferences:PackageID"`
	Channels          []VideoChannel    `json:"channels" gorm:"many2many:video_template_channel;joinForeignKey:TemplateID;joinReferences:ChannelID"`
}

func (VideoTemplate) TableName() string {
	return "video_template"
}

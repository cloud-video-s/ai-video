package model

const (
	BannerJumpTypeLink        uint8 = 1
	BannerJumpTypeTemplate    uint8 = 2
	BannerJumpTypeTextToImage uint8 = 3
	BannerJumpTypeTextToVideo uint8 = 4
)

// VideoBanner stores a banner, its delivery targets and its navigation action.
type VideoBanner struct {
	ID         uint64  `json:"id" gorm:"primaryKey;autoIncrement"`
	Name       string  `json:"name" gorm:"size:128;not null;index;comment:banner name"`
	CoverImage string  `json:"cover_image" gorm:"size:1024;not null;comment:cover image URL"`
	Remark     string  `json:"remark" gorm:"size:500;comment:remark"`
	Sort       uint64  `json:"sort" gorm:"type:bigint unsigned;not null;default:0;index;comment:sort order"`
	JumpType   uint8   `json:"jump_type" gorm:"type:tinyint unsigned;not null;default:1;index;comment:跳转类型 1=链接 2=模板 3=文生图 4=文生视频"`
	JumpURL    string  `json:"jump_url" gorm:"size:1024;comment:link target URL"`
	TemplateID *uint64 `json:"template_id" gorm:"index;comment:target template ID"`
	Status     int8    `json:"status" gorm:"not null;default:1;index;comment:status: 0 disabled, 1 enabled"`
	BaseModel

	Template         *VideoTemplate         `json:"template,omitempty" gorm:"foreignKey:TemplateID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
	DisplayPositions []VideoDisplayPosition `json:"display_positions" gorm:"many2many:video_banner_display_position;foreignKey:ID;joinForeignKey:BannerID;references:PositionKey;joinReferences:PositionKey;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
	Countries        []VideoCountry         `json:"countries" gorm:"many2many:video_banner_country;joinForeignKey:BannerID;joinReferences:CountryID"`
	Channels         []VideoChannel         `json:"channels" gorm:"many2many:video_banner_channel;joinForeignKey:BannerID;joinReferences:ChannelID"`
	Packages         []VideoPackage         `json:"packages" gorm:"many2many:video_banner_package;joinForeignKey:BannerID;joinReferences:PackageCode"`
}

func (VideoBanner) TableName() string {
	return "video_banner"
}

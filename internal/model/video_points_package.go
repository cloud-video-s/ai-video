package model

// VideoPointsPackage defines a one-time points/credits product and the user,
// platform and channel scopes in which it can be offered.
type VideoPointsPackage struct {
	ID            uint64   `json:"id" gorm:"primaryKey;autoIncrement"`
	ProductID     string   `json:"product_id" gorm:"size:191;not null;uniqueIndex;comment:globally unique store product SKU"`
	Name          string   `json:"name" gorm:"size:128;not null;index;comment:points package name"`
	PackageID     uint64   `json:"package_id" gorm:"not null;index;comment:application package ID"`
	Systems       []string `json:"systems" gorm:"type:text;serializer:json;comment:android, ios, pc, harmony or other systems"`
	UserTypes     []int    `json:"user_types" gorm:"type:text;serializer:json;comment:app user types: 1 free, 2 paid"`
	ResourceType  string   `json:"resource_type" gorm:"size:32;not null;default:credits;index;comment:resource type"`
	Points        uint64   `json:"points" gorm:"not null;default:0;comment:granted points quantity"`
	Currency      string   `json:"currency" gorm:"size:8;not null;default:USD;comment:ISO currency code"`
	SalePrice     float64  `json:"sale_price" gorm:"type:decimal(12,2);not null;default:0;comment:sale price"`
	ActualRevenue float64  `json:"actual_revenue" gorm:"type:decimal(12,2);not null;default:0;comment:net revenue"`
	OriginalPrice float64  `json:"original_price" gorm:"type:decimal(12,2);not null;default:0;comment:strikethrough price"`
	BadgeText     string   `json:"badge_text" gorm:"size:64;comment:badge copy"`
	Description   string   `json:"description" gorm:"size:1000;comment:package description"`
	ButtonText    string   `json:"button_text" gorm:"size:128;comment:purchase button copy"`
	IsDefault     bool     `json:"is_default" gorm:"not null;default:false;index;comment:default package for app package and resource type"`
	Status        int8     `json:"status" gorm:"not null;default:1;index;comment:0 disabled, 1 enabled"`
	Sort          int      `json:"sort" gorm:"not null;default:0;index;comment:sort order"`
	BaseModel

	Package  VideoPackage   `json:"package,omitempty" gorm:"foreignKey:PackageID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
	Channels []VideoChannel `json:"channels,omitempty" gorm:"many2many:video_points_package_channel;joinForeignKey:PointsPackageID;joinReferences:ChannelID"`
}

func (VideoPointsPackage) TableName() string {
	return "video_points_package"
}

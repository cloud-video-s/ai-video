package model

const (
	VIPPlanTypeNormal  = "normal"
	VIPPlanTypeTrial   = "trial"
	VIPPlanTypePaywall = "paywall"

	VIPDisplayModeHidden = 0
	VIPDisplayModeNormal = 1
)

// VideoVIPSubscription stores a platform SKU and the presentation,
// targeting, pricing and entitlement rules used to sell a VIP plan.
type VideoVIPSubscription struct {
	ID                       uint64  `json:"id" gorm:"primaryKey;autoIncrement"`
	PackageID                uint64  `json:"package_id" gorm:"not null;uniqueIndex:uk_vip_product_scope,priority:1;index;comment:application package ID"`
	Platform                 string  `json:"platform" gorm:"size:32;not null;uniqueIndex:uk_vip_product_scope,priority:2;index;comment:android, ios, pc or web"`
	ProductID                string  `json:"product_id" gorm:"size:191;not null;uniqueIndex:uk_vip_product_scope,priority:3;index;comment:store product SKU"`
	Name                     string  `json:"name" gorm:"size:128;not null;index;comment:VIP plan name"`
	VIPLevel                 string  `json:"vip_level" gorm:"size:64;not null;index;comment:VIP level"`
	PlanType                 string  `json:"plan_type" gorm:"size:32;not null;index;comment:plan type"`
	AppVersion               string  `json:"app_version" gorm:"size:32;index;comment:minimum or targeted app version"`
	Currency                 string  `json:"currency" gorm:"size:8;not null;default:USD;comment:ISO currency code"`
	FirstSubscriptionPrice   float64 `json:"first_subscription_price" gorm:"type:decimal(12,2);not null;default:0;comment:first subscription price"`
	FirstSubscriptionRevenue float64 `json:"first_subscription_revenue" gorm:"type:decimal(12,2);not null;default:0;comment:first subscription net revenue"`
	FirstBonusPoints         uint64  `json:"first_bonus_points" gorm:"not null;default:0;comment:first subscription bonus points"`
	OriginalPrice            float64 `json:"original_price" gorm:"type:decimal(12,2);not null;default:0;comment:strikethrough price"`
	VIPDurationDays          uint32  `json:"vip_duration_days" gorm:"not null;default:0;comment:VIP entitlement duration in days"`
	TrialDays                uint32  `json:"trial_days" gorm:"not null;default:0;comment:free trial days"`
	RenewalText              string  `json:"renewal_text" gorm:"size:255;comment:renewal copy"`
	BadgeText                string  `json:"badge_text" gorm:"size:64;comment:badge copy"`
	AgreementDefaultChecked  bool    `json:"agreement_default_checked" gorm:"not null;default:false;comment:subscription agreement checked by default"`
	DisplayMode              int8    `json:"display_mode" gorm:"not null;default:1;index;comment:0 hidden, 1 normal"`
	Status                   int8    `json:"status" gorm:"not null;default:1;index;comment:0 disabled, 1 enabled"`
	FreeTrial                bool    `json:"free_trial" gorm:"not null;default:false;index;comment:free trial enabled"`
	IsSubscription           bool    `json:"is_subscription" gorm:"not null;default:true;index;comment:recurring subscription"`
	IsDefault                bool    `json:"is_default" gorm:"not null;default:false;index;comment:default plan for package and platform"`
	SubscriptionDescription  string  `json:"subscription_description" gorm:"size:500;comment:subscription description"`
	SubscriptionPrice        float64 `json:"subscription_price" gorm:"type:decimal(12,2);not null;default:0;comment:renewal subscription price"`
	SubscriptionRevenue      float64 `json:"subscription_revenue" gorm:"type:decimal(12,2);not null;default:0;comment:renewal net revenue"`
	SubscriptionPoints       uint64  `json:"subscription_points" gorm:"not null;default:0;comment:subscription points"`
	SubscriptionPeriod       string  `json:"subscription_period" gorm:"size:64;comment:subscription period, such as P1M or P1Y"`
	Sort                     int     `json:"sort" gorm:"not null;default:0;index;comment:sort order"`
	Description              string  `json:"description" gorm:"size:1000;comment:plan description"`
	Remark                   string  `json:"remark" gorm:"size:1000;comment:internal remark"`
	BaseModel

	Package          VideoPackage           `json:"package,omitempty" gorm:"foreignKey:PackageID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
	DisplayPositions []VideoDisplayPosition `json:"display_positions,omitempty" gorm:"many2many:video_vip_subscription_position;joinForeignKey:SubscriptionID;joinReferences:DisplayPositionID"`
	Channels         []VideoChannel         `json:"channels,omitempty" gorm:"many2many:video_vip_subscription_channel;joinForeignKey:SubscriptionID;joinReferences:ChannelID"`
	ExcludedChannels []VideoChannel         `json:"excluded_channels,omitempty" gorm:"many2many:video_vip_subscription_excluded_channel;joinForeignKey:SubscriptionID;joinReferences:ChannelID"`
}

func (VideoVIPSubscription) TableName() string {
	return "video_vip_subscription"
}

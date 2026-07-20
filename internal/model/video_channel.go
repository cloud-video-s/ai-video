package model

// VideoChannel stores advertising/distribution channel configuration and its
// commercial settlement values.
type VideoChannel struct {
	ChannelID       uint64  `json:"channel_id" gorm:"primaryKey;autoIncrement;comment:channel ID"`
	ChannelCode     string  `json:"channel_code" gorm:"size:64;not null;uniqueIndex;comment:unique channel identifier"`
	ChannelName     string  `json:"channel_name" gorm:"size:128;not null;index;comment:channel name"`
	AgencyCompany   string  `json:"agency_company" gorm:"size:128;index;comment:agency company"`
	AdPlatform      string  `json:"ad_platform" gorm:"size:64;not null;index;comment:advertising platform"`
	DeliveryPackage string  `json:"delivery_package" gorm:"size:255;not null;index;comment:delivery package"`
	TrackingURL     string  `json:"tracking_url" gorm:"size:1024;comment:tracking URL"`
	PortRebate      float64 `json:"port_rebate" gorm:"type:decimal(8,4);not null;default:0;comment:port rebate percentage"`
	ServiceOrderFee float64 `json:"service_order_fee" gorm:"type:decimal(12,2);not null;default:0;comment:service fee per order"`
	UploadMethod    string  `json:"upload_method" gorm:"size:32;not null;default:API;index;comment:data upload method"`
	Status          int8    `json:"status" gorm:"not null;default:1;index;comment:status: 0 disabled, 1 enabled"`
	BaseModel
}

func (VideoChannel) TableName() string {
	return "video_channel"
}

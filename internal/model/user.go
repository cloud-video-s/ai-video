package model

import "time"

const (
	AppUserLoginGuest  uint32 = 1
	AppUserLoginGoogle uint32 = 2
	AppUserLoginAppID  uint32 = 3

	AppUserTypeFree uint32 = 1
	AppUserTypePaid uint32 = 2

	AppUserSubscriptionNotSubscribed uint32 = 1
	AppUserSubscriptionSubscribed    uint32 = 2
	AppUserSubscriptionCancelled     uint32 = 3
)

// VideoUser stores client-side user profile, engagement and payment metrics.
// Its tags intentionally mirror the production video_user DDL.
type VideoUser struct {
	ID                       uint64     `gorm:"column:id;type:bigint unsigned;primaryKey;autoIncrement:true" json:"id"`
	IMEI                     string     `gorm:"column:imei;type:varchar(128);not null;uniqueIndex:uidx_video_user_phone_registration;index:idx_video_user_login_account;comment:设备编号" json:"imei"`
	Username                 string     `gorm:"column:username;type:varchar(128);not null;index:idx_video_user_username;comment:昵称" json:"username"`
	DeviceCountry            string     `gorm:"column:device_country;type:varchar(64);index:idx_video_user_device_country;comment:国家" json:"device_country"`
	ChannelID                string     `gorm:"column:channel_id;type:varchar(64);index:idx_video_user_channel_id;comment:渠道id" json:"channel_id"`
	AppVersion               string     `gorm:"column:app_version;type:varchar(32);index:idx_video_user_app_version;comment:激活版本号" json:"app_version"`
	FirstOpenedAt            *time.Time `gorm:"column:first_opened_at;type:datetime(3);index:idx_video_user_first_opened_at;comment:首次打开时间" json:"first_opened_at"`
	LastOpenedAt             *time.Time `gorm:"column:last_opened_at;type:datetime(3);index:idx_video_user_last_opened_at;comment:上次打开时间" json:"last_opened_at"`
	LoginType                uint32     `gorm:"column:login_type;type:tinyint unsigned;not null;index:idx_video_user_login_type;default:1;comment:登录方式 1=未登录 2=google 3=appid" json:"login_type"`
	UserType                 uint32     `gorm:"column:user_type;type:tinyint unsigned;not null;index:idx_video_user_user_type;default:1;comment:用户类型 1=免费 2=付费" json:"user_type"`
	ActiveDays               uint32     `gorm:"column:active_days;type:int unsigned;not null;comment:活跃天数" json:"active_days"`
	AvgDailyUsageSeconds     uint64     `gorm:"column:avg_daily_usage_seconds;type:bigint unsigned;not null;comment:平均日使用时长" json:"avg_daily_usage_seconds"`
	VipExpiresAt             *time.Time `gorm:"column:vip_expires_at;type:datetime(3);index:idx_video_user_vip_expires_at;comment:vip 到期时间" json:"vip_expires_at"`
	PointsBalance            uint64     `gorm:"column:points_balance;type:bigint unsigned;not null;comment:积分" json:"points_balance"`
	SubscriptionStatus       uint32     `gorm:"column:subscription_status;type:tinyint unsigned;not null;index:idx_video_user_subscription_status;default:1;comment:订阅状态 1未订阅 2订阅中 3=已取消" json:"subscription_status"`
	FirstOrderCreatedAt      *time.Time `gorm:"column:first_order_created_at;type:datetime(3);comment:首单创建时间" json:"first_order_created_at"`
	FirstPaidAt              *time.Time `gorm:"column:first_paid_at;type:datetime(3);comment:首单付费时间" json:"first_paid_at"`
	OrderCount               uint64     `gorm:"column:order_count;type:bigint unsigned;not null;comment:订单创建次数" json:"order_count"`
	PaymentCount             uint64     `gorm:"column:payment_count;type:bigint unsigned;not null;comment:付费次数（订阅和积分订单累计）" json:"payment_count"`
	SubscriptionPaymentCount uint64     `gorm:"column:subscription_payment_count;type:bigint unsigned;not null;comment:完成付费次数" json:"subscription_payment_count"`
	OneTimePaymentCount      uint64     `gorm:"column:one_time_payment_count;type:bigint unsigned;not null;comment:付费金额" json:"one_time_payment_count"`
	OrderAmountMoney         float64    `gorm:"column:order_amount_money;type:decimal(10,2) unsigned zerofill;not null;default:00000000.00;comment:累计付费金额" json:"order_amount_money"`
	ActualAmountMoney        float64    `gorm:"column:actual_amount_money;type:decimal(10,2) unsigned zerofill;not null;default:00000000.00;comment:累计税后金额" json:"actual_amount_money"`
	LastPaidAt               *time.Time `gorm:"column:last_paid_at;type:datetime(3);index:idx_video_user_last_paid_at;comment:最后付费时间" json:"last_paid_at"`
	RefundAmountMoney        float64    `gorm:"column:refund_amount_money;type:decimal(10,2);not null;default:0.00;comment:累计退款金额" json:"refund_amount_money"`
	PointsMoney              float64    `gorm:"column:points_money;type:decimal(10,0) unsigned;not null;default:0;comment:累计积分成本" json:"points_money"`
	AiCotsMoney              float64    `gorm:"column:ai_cots_money;type:decimal(10,2) unsigned zerofill;not null;index:idx_video_user_ai_version;comment:累计ai成本" json:"ai_cots_money"`
	Activated                uint32     `gorm:"column:activated;type:int unsigned;not null;index:idx_video_user_activated;comment:是否激活达标 1 是 0否" json:"activated"`
	KeyBehaviorMet           uint32     `gorm:"column:key_behavior_met;type:int unsigned;not null;index:idx_video_user_key_behavior_met;comment:关键行为是否达标 1 是 0否" json:"key_behavior_met"`
	PaymentMet               bool       `gorm:"column:payment_met;type:tinyint(1);not null;index:idx_video_user_payment_met;comment:付费是否达标 1 是 0否" json:"payment_met"`
	FirstPaymentMet          bool       `gorm:"column:first_payment_met;type:tinyint(1);not null;index:idx_video_user_first_payment_met;comment:首次付费是否达标 1 是 0否" json:"first_payment_met"`
	Registered               bool       `gorm:"column:registered;type:tinyint(1);not null;index:idx_video_user_registered;comment:注册达标 1 是 0否" json:"registered"`
	AttributionClickedAt     *time.Time `gorm:"column:attribution_clicked_at;type:datetime(3);index:idx_video_user_attribution_clicked_at;comment:归因点击时间" json:"attribution_clicked_at"`
	PhoneModel               string     `gorm:"column:phone_model;type:varchar(128);index:idx_video_user_phone_model;comment:手机品牌、型号" json:"phone_model"`
	ReRegisteredFromID       uint64     `gorm:"column:re_registered_from_id;type:bigint unsigned;index:idx_video_user_re_registered_from_id;comment:原用户ID" json:"re_registered_from_id,omitempty"`
	AppName                  string     `gorm:"column:app_name;type:varchar(255);not null;default:0;comment:APP应用名称" json:"app_name"`
	TokenVersion             int64      `gorm:"column:token_version;type:bigint;not null;comment:token版本 防止多端账号登录" json:"token_version"`
	Status                   int32      `gorm:"column:status;type:tinyint;not null;index:idx_video_user_status;default:1;comment:状态 1正常 0禁用" json:"status"`
	LastLoginAt              *time.Time `gorm:"column:last_login_at;type:datetime(3);comment:上次登录时间" json:"last_login_at"`
	LastLoginIP              string     `gorm:"column:last_login_ip;type:varchar(64);comment:上次登录IP" json:"last_login_ip"`
	LoginAccount             string     `gorm:"column:login_account;type:varchar(255)" json:"login_account"`
	AppIDEmail               string     `gorm:"column:appid_email;type:varchar(50);comment:苹果邮箱" json:"appid_email,omitempty"`
	AppIDThirdCode           string     `gorm:"column:appid_third_code;type:varchar(50);comment:苹果三方唯一码" json:"appid_third_code,omitempty"`
	GoogleEmail              string     `gorm:"column:google_email;type:varchar(50);uniqueIndex:idx_video_user_email;comment:谷歌邮箱" json:"google_email,omitempty"`
	GoogleThirdCode          string     `gorm:"column:google_third_code;type:varchar(50);comment:谷歌唯一编码" json:"google_third_code,omitempty"`
	PackageCode              string     `gorm:"column:package_code;type:varchar(128);not null;comment:package identifier" json:"package_code,omitempty"`
	BaseModel
}

func (VideoUser) TableName() string { return "video_user" }

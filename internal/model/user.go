package model

import "time"

const (
	AppUserLoginGuest  = "guest"
	AppUserLoginGoogle = "google"
	AppUserLoginApple  = "apple"

	AppUserTypeFree = "free"
	AppUserTypePaid = "paid"

	AppUserSubscriptionSubscribed    = "subscribed"
	AppUserSubscriptionNotSubscribed = "not_subscribed"
	AppUserSubscriptionCancelled     = "cancelled"
)

// VideoUser stores client-side user profile, engagement and payment metrics.
// Money values use the smallest currency unit to avoid floating-point errors.
type VideoUser struct {
	ID                       uint64     `json:"id" gorm:"primaryKey;autoIncrement"`
	PhoneCode                string     `json:"phone_code" gorm:"size:128;not null;uniqueIndex:uidx_video_user_phone_registration,priority:1"`
	RegistrationNo           uint32     `json:"registration_no" gorm:"not null;default:1;uniqueIndex:uidx_video_user_phone_registration,priority:2"`
	ReRegisteredFromID       *uint64    `json:"re_registered_from_id,omitempty" gorm:"index"`
	Email                    *string    `json:"email,omitempty" gorm:"size:255;uniqueIndex"`
	EmailVerified            bool       `json:"email_verified" gorm:"not null;default:false"`
	TokenVersion             int        `json:"-" gorm:"not null;default:0"`
	Status                   int8       `json:"status" gorm:"not null;default:1;index"`
	LastLoginAt              *time.Time `json:"last_login_at,omitempty"`
	LastLoginIP              string     `json:"-" gorm:"size:64"`
	Username                 string     `json:"username" gorm:"size:128;not null;index"`
	DeviceCountry            string     `json:"device_country" gorm:"size:64;index"`
	IPCountry                string     `json:"ip_country" gorm:"size:8;index"`
	ChannelID                string     `json:"channel_id" gorm:"size:64;index"`
	AppVersion               string     `json:"app_version" gorm:"size:32;index"`
	FirstOpenedAt            *time.Time `json:"first_opened_at" gorm:"index"`
	LastOpenedAt             *time.Time `json:"last_opened_at" gorm:"index"`
	LoginType                string     `json:"login_type" gorm:"size:16;not null;default:guest;index"`
	LoginAccount             string     `json:"login_account" gorm:"size:255;index"`
	UserType                 string     `json:"user_type" gorm:"size:16;not null;default:free;index"`
	ActiveDays               uint32     `json:"active_days" gorm:"not null;default:0"`
	AvgDailyUsageSeconds     uint64     `json:"avg_daily_usage_seconds" gorm:"not null;default:0"`
	VIPExpiresAt             *time.Time `json:"vip_expires_at" gorm:"column:vip_expires_at;index:idx_video_user_vip_expires_at"`
	PointsBalance            int64      `json:"points_balance" gorm:"not null;default:0"`
	SubscriptionStatus       string     `json:"subscription_status" gorm:"size:24;not null;default:not_subscribed;index"`
	FirstOrderCreatedAt      *time.Time `json:"first_order_created_at"`
	FirstPaidAt              *time.Time `json:"first_paid_at"`
	OrderCount               uint64     `json:"order_count" gorm:"not null;default:0"`
	PaymentCount             uint64     `json:"payment_count" gorm:"not null;default:0"`
	SubscriptionPaymentCount uint64     `json:"subscription_payment_count" gorm:"not null;default:0"`
	OneTimePaymentCount      uint64     `json:"one_time_payment_count" gorm:"not null;default:0"`
	OrderAmountCents         int64      `json:"order_amount_cents" gorm:"not null;default:0"`
	ActualAmountCents        int64      `json:"actual_amount_cents" gorm:"not null;default:0"`
	LastPaidAt               *time.Time `json:"last_paid_at" gorm:"index"`
	RefundAmountCents        int64      `json:"refund_amount_cents" gorm:"not null;default:0"`
	PointsCost               uint64     `json:"points_cost" gorm:"not null;default:0"`
	AIVersion                string     `json:"ai_version" gorm:"size:32;index"`
	Activated                bool       `json:"activated" gorm:"not null;default:false;index"`
	KeyBehaviorMet           bool       `json:"key_behavior_met" gorm:"not null;default:false;index"`
	PaymentMet               bool       `json:"payment_met" gorm:"not null;default:false;index"`
	FirstPaymentMet          bool       `json:"first_payment_met" gorm:"not null;default:false;index"`
	Registered               bool       `json:"registered" gorm:"not null;default:false;index"`
	AttributionClickedAt     *time.Time `json:"attribution_clicked_at" gorm:"index"`
	PhoneModel               string     `json:"phone_model" gorm:"size:128;index"`
	BaseModel
}

func (VideoUser) TableName() string {
	return "video_user"
}

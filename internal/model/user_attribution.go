package model

import (
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	AttributionEventActivation   = "activation"
	AttributionEventKeyBehavior  = "key_behavior"
	AttributionEventPayment      = "payment"
	AttributionEventFirstPayment = "first_payment"
	AttributionEventRegistration = "registration"

	AttributionActionCallback = "callback"
	AttributionActionDeduct   = "deduct"
)

// VideoUserAttribution stores acquisition identifiers and callback accounting
// for one client user. Behavior flags remain sourced from video_user.
type VideoUserAttribution struct {
	ID          uint64 `json:"id" gorm:"primaryKey;autoIncrement;comment:attribution ID"`
	UserID      uint64 `json:"user_id" gorm:"not null;uniqueIndex;comment:client user ID"`
	ChannelCode string `json:"channel_code" gorm:"size:64;not null;default:'';index;comment:channel code snapshot"`

	OAID      string `json:"oaid" gorm:"column:oaid;size:128;not null;default:'';index;comment:OAID"`
	IMEI      string `json:"imei" gorm:"size:128;not null;default:'';index;comment:IMEI"`
	AndroidID string `json:"android_id" gorm:"size:128;not null;default:'';index;comment:Android ID"`
	IP        string `json:"ip" gorm:"size:64;not null;default:'';index;comment:attribution IP"`
	UserAgent string `json:"user_agent" gorm:"size:1024;not null;default:'';comment:user agent"`

	ActivationCallbackCount   uint64 `json:"activation_callback_count" gorm:"not null;default:0"`
	ActivationDeductCount     uint64 `json:"activation_deduct_count" gorm:"not null;default:0"`
	KeyBehaviorCallbackCount  uint64 `json:"key_behavior_callback_count" gorm:"not null;default:0"`
	KeyBehaviorDeductCount    uint64 `json:"key_behavior_deduct_count" gorm:"not null;default:0"`
	PaymentCallbackCount      uint64 `json:"payment_callback_count" gorm:"not null;default:0"`
	PaymentDeductCount        uint64 `json:"payment_deduct_count" gorm:"not null;default:0"`
	FirstPaymentCallbackCount uint64 `json:"first_payment_callback_count" gorm:"not null;default:0"`
	FirstPaymentDeductCount   uint64 `json:"first_payment_deduct_count" gorm:"not null;default:0"`
	RegistrationCallbackCount uint64 `json:"registration_callback_count" gorm:"not null;default:0"`
	RegistrationDeductCount   uint64 `json:"registration_deduct_count" gorm:"not null;default:0"`

	AttributedAt   *time.Time `json:"attributed_at" gorm:"index;comment:attribution time"`
	LastOperatedAt *time.Time `json:"last_operated_at" gorm:"index;comment:last callback or deduct time"`
	Remark         string     `json:"remark" gorm:"size:255;not null;default:''"`

	User    VideoUser     `json:"user" gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Channel *VideoChannel `json:"channel,omitempty" gorm:"-"`
	BaseModel
}

func (VideoUserAttribution) TableName() string { return "video_user_attribution" }

// AfterCreate covers API automatic registration and Admin-created users.
func (u *VideoUser) AfterCreate(tx *gorm.DB) error {
	attributedAt := u.AttributionClickedAt
	if attributedAt == nil {
		attributedAt = u.FirstOpenedAt
	}
	row := VideoUserAttribution{
		UserID: u.ID, ChannelCode: u.ChannelID, IMEI: u.IMEI, IP: u.LastLoginIP, AttributedAt: attributedAt,
	}
	return tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		DoNothing: true,
	}).Create(&row).Error
}

package model

import "time"

const (
	IdentityProviderGoogle = "google"
	IdentityProviderApple  = "apple"
)

// VideoUserIdentity links a local client user to a stable subject issued by a
// trusted identity provider. ID/access tokens are intentionally never stored.
type VideoUserIdentity struct {
	ID                uint64     `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID            uint64     `json:"user_id" gorm:"not null;uniqueIndex:uk_user_identity_user_provider,priority:1;index"`
	Provider          string     `json:"provider" gorm:"size:32;not null;uniqueIndex:uk_user_identity_subject,priority:1;uniqueIndex:uk_user_identity_user_provider,priority:2;index"`
	ProviderSubject   string     `json:"provider_subject" gorm:"size:191;not null;uniqueIndex:uk_user_identity_subject,priority:2"`
	Issuer            string     `json:"issuer" gorm:"size:255;not null"`
	Audience          string     `json:"audience" gorm:"size:255;not null"`
	Email             string     `json:"email" gorm:"size:255;index"`
	EmailVerified     bool       `json:"email_verified" gorm:"not null;default:false"`
	IsPrivateEmail    bool       `json:"is_private_email" gorm:"not null;default:false"`
	DisplayName       string     `json:"display_name" gorm:"size:128"`
	GivenName         string     `json:"given_name" gorm:"size:128"`
	FamilyName        string     `json:"family_name" gorm:"size:128"`
	AvatarURL         string     `json:"avatar_url" gorm:"size:1024"`
	LastLoginAt       *time.Time `json:"last_login_at" gorm:"index"`
	LastTokenIssuedAt *time.Time `json:"last_token_issued_at"`
	CreatedAt         time.Time  `json:"created_at" gorm:"index"`
	UpdatedAt         time.Time  `json:"updated_at"`

	User VideoUser `json:"-" gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

func (VideoUserIdentity) TableName() string { return "video_user_identity" }

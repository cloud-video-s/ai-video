package app

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

var deprecatedVideoUserColumns = []string{
	"phone_code",
	"ip_country",
	"registration_no",
	"email_verified",
}

type videoUserCenterColumns struct {
	ID            uint64     `gorm:"column:id;primaryKey"`
	Email         string     `gorm:"column:email;type:varchar(255);not null;default:'';index:idx_video_user_email"`
	Phone         string     `gorm:"column:phone;type:varchar(32);not null;default:'';index:idx_video_user_phone"`
	VIPLevel      uint32     `gorm:"column:vip_level;not null;default:0;index:idx_video_user_vip_level"`
	VIPStartedAt  *time.Time `gorm:"column:vip_started_at;type:datetime(3)"`
	IsFrozen      bool       `gorm:"column:is_frozen;type:tinyint(1);not null;default:0;index:idx_video_user_is_frozen"`
	IsBlacklisted bool       `gorm:"column:is_blacklisted;type:tinyint(1);not null;default:0;index:idx_video_user_is_blacklisted"`
}

func (videoUserCenterColumns) TableName() string { return "video_user" }

// MigrateUserCenterColumns incrementally adds user-center fields without
// rebuilding the table or dropping existing user data.
func MigrateUserCenterColumns(db *gorm.DB) error {
	if db == nil || !db.Migrator().HasTable("video_user") {
		return nil
	}
	if err := db.AutoMigrate(&videoUserCenterColumns{}); err != nil {
		return fmt.Errorf("migrate video user center columns: %w", err)
	}
	if db.Migrator().HasColumn("video_user", "google_email") {
		if err := db.Exec(`UPDATE video_user SET email = google_email
			WHERE (email IS NULL OR email = '') AND google_email IS NOT NULL AND google_email <> ''`).Error; err != nil {
			return fmt.Errorf("backfill Google user emails: %w", err)
		}
	}
	if db.Migrator().HasColumn("video_user", "appid_email") {
		if err := db.Exec(`UPDATE video_user SET email = appid_email
			WHERE (email IS NULL OR email = '') AND appid_email IS NOT NULL AND appid_email <> ''`).Error; err != nil {
			return fmt.Errorf("backfill Apple user emails: %w", err)
		}
	}
	if db.Migrator().HasColumn("video_user", "status") {
		if err := db.Exec(`UPDATE video_user SET is_frozen = 1 WHERE status = 0 AND is_frozen = 0`).Error; err != nil {
			return fmt.Errorf("backfill frozen video users: %w", err)
		}
	}
	if db.Migrator().HasColumn("video_user", "vip_expires_at") &&
		db.Migrator().HasColumn("video_user", "user_type") &&
		db.Migrator().HasColumn("video_user", "subscription_status") {
		expiryCondition := "vip_expires_at > CURRENT_TIMESTAMP"
		if db.Dialector.Name() == "sqlite" {
			expiryCondition = "datetime(vip_expires_at) > CURRENT_TIMESTAMP"
		}
		if err := db.Exec(`UPDATE video_user SET vip_level = 1, user_type = 2, subscription_status = 2
			WHERE vip_expires_at IS NOT NULL AND ` + expiryCondition + ` AND vip_level = 0`).Error; err != nil {
			return fmt.Errorf("backfill legacy VIP users: %w", err)
		}
	}
	return nil
}

// PrepareVideoUserColumns preserves the legacy device identifier before the
// new model is auto-migrated. When both columns exist (an interrupted rollout),
// empty IMEI values are backfilled before the old column is removed.
func PrepareVideoUserColumns(db *gorm.DB) error {
	if db == nil || !db.Migrator().HasTable("video_user") {
		return nil
	}
	// An earlier AutoMigrate may have added package_code as nullable before a
	// later deployment made the field required. MySQL refuses ALTER ... NOT
	// NULL while any legacy row still contains NULL, so normalize those rows
	// before GORM reconciles the column definition.
	if db.Migrator().HasColumn("video_user", "package_code") {
		if err := db.Exec(`UPDATE video_user SET package_code = '' WHERE package_code IS NULL`).Error; err != nil {
			return fmt.Errorf("backfill video_user.package_code: %w", err)
		}
	}
	hasPhoneCode := db.Migrator().HasColumn("video_user", "phone_code")
	hasIMEI := db.Migrator().HasColumn("video_user", "imei")
	if hasPhoneCode && !hasIMEI {
		if err := db.Migrator().RenameColumn("video_user", "phone_code", "imei"); err != nil {
			return fmt.Errorf("rename video_user.phone_code to imei: %w", err)
		}
		return nil
	}
	if hasPhoneCode && hasIMEI {
		if err := db.Exec(`UPDATE video_user SET imei = phone_code
			WHERE (imei IS NULL OR imei = '') AND phone_code IS NOT NULL AND phone_code <> ''`).Error; err != nil {
			return fmt.Errorf("backfill video_user.imei: %w", err)
		}
	}
	return nil
}

// DropDeprecatedVideoUserColumns removes legacy video_user columns that are no
// longer represented by the application model. GORM AutoMigrate deliberately
// never drops columns, so this cleanup must be explicit and idempotent.
func DropDeprecatedVideoUserColumns(db *gorm.DB) error {
	if db == nil || !db.Migrator().HasTable("video_user") {
		return nil
	}
	if db.Migrator().HasColumn("video_user", "email") {
		lengthFunction := "CHAR_LENGTH"
		if db.Dialector.Name() == "sqlite" {
			lengthFunction = "LENGTH"
		}
		if db.Migrator().HasColumn("video_user", "google_email") {
			if err := db.Exec(`UPDATE video_user SET google_email = email
				WHERE login_type = 2 AND (google_email IS NULL OR google_email = '')
				AND email IS NOT NULL AND email <> '' AND ` + lengthFunction + `(email) <= 50`).Error; err != nil {
				return fmt.Errorf("backfill video_user.google_email: %w", err)
			}
		}
		if db.Migrator().HasColumn("video_user", "appid_email") {
			if err := db.Exec(`UPDATE video_user SET appid_email = email
				WHERE login_type = 3 AND (appid_email IS NULL OR appid_email = '')
				AND email IS NOT NULL AND email <> '' AND ` + lengthFunction + `(email) <= 50`).Error; err != nil {
				return fmt.Errorf("backfill video_user.appid_email: %w", err)
			}
		}
	}
	for _, column := range deprecatedVideoUserColumns {
		if !db.Migrator().HasColumn("video_user", column) {
			continue
		}
		if err := db.Migrator().DropColumn("video_user", column); err != nil {
			return fmt.Errorf("drop deprecated video_user.%s: %w", column, err)
		}
	}
	return nil
}

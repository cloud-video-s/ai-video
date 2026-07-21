package app

import (
	"fmt"

	"gorm.io/gorm"
)

var deprecatedVideoUserColumns = []string{
	"phone_code",
	"ip_country",
	"registration_no",
	"email",
	"email_verified",
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

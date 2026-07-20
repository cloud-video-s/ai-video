package app

import (
	"ai-video/internal/model"

	"gorm.io/gorm"
)

// NormalizeUserAttributionColumns keeps the API field "oaid" stable instead of
// GORM's default initialism split "oa_id".
func NormalizeUserAttributionColumns(db *gorm.DB) error {
	if !db.Migrator().HasTable(&model.VideoUserAttribution{}) {
		return nil
	}
	hasLegacy := db.Migrator().HasColumn(&model.VideoUserAttribution{}, "oa_id")
	hasCurrent := db.Migrator().HasColumn(&model.VideoUserAttribution{}, "oaid")
	if hasLegacy && !hasCurrent {
		return db.Migrator().RenameColumn(&model.VideoUserAttribution{}, "oa_id", "oaid")
	}
	if hasLegacy && hasCurrent {
		if err := db.Exec("UPDATE video_user_attribution SET oaid = oa_id WHERE oaid = '' AND oa_id <> ''").Error; err != nil {
			return err
		}
		return db.Migrator().DropColumn(&model.VideoUserAttribution{}, "oa_id")
	}
	return nil
}

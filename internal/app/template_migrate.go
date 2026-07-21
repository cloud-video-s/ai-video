package app

import (
	"ai-video/internal/gen/model"

	"gorm.io/gorm"
)

var legacyTemplateTypeColumns = []string{
	"app_name",
	"app_version",
	"account_type",
}

// RemoveLegacyTemplateTypeColumns removes dimensions that are no longer part
// of the basic template-category definition. It is idempotent for both MySQL
// and PostgreSQL and can safely run after every AutoMigrate.
func RemoveLegacyTemplateTypeColumns(db *gorm.DB) error {
	if !db.Migrator().HasTable(&model.VideoTemplateType{}) {
		return nil
	}
	for _, column := range legacyTemplateTypeColumns {
		if !db.Migrator().HasColumn(&model.VideoTemplateType{}, column) {
			continue
		}
		if err := db.Migrator().DropColumn(&model.VideoTemplateType{}, column); err != nil {
			return err
		}
	}
	return nil
}

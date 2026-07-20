package app

import "gorm.io/gorm"

// RemoveTemplateDisplayPositionTargets removes the obsolete direct Template
// placement relationship. Display positions are now assigned to Banners.
func RemoveTemplateDisplayPositionTargets(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	if db.Migrator().HasTable("video_template_display_position") {
		if err := db.Migrator().DropTable("video_template_display_position"); err != nil {
			return err
		}
	}
	if db.Migrator().HasTable("video_template") && db.Migrator().HasColumn("video_template", "display_position_id") {
		if err := db.Exec("ALTER TABLE video_template DROP COLUMN display_position_id").Error; err != nil {
			return err
		}
	}
	return nil
}

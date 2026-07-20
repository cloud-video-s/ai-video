package app

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestRemoveTemplateDisplayPositionTargets(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:remove-template-position?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("CREATE TABLE video_template (id INTEGER PRIMARY KEY, display_position_id INTEGER NOT NULL DEFAULT 0)").Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("CREATE TABLE video_template_display_position (template_id INTEGER, display_position_id INTEGER)").Error; err != nil {
		t.Fatal(err)
	}
	if err := RemoveTemplateDisplayPositionTargets(db); err != nil {
		t.Fatal(err)
	}
	if db.Migrator().HasTable("video_template_display_position") {
		t.Fatal("video_template_display_position still exists")
	}
	if db.Migrator().HasColumn("video_template", "display_position_id") {
		t.Fatal("video_template.display_position_id still exists")
	}
}

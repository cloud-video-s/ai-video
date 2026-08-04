package app

import (
	"testing"

	"ai-video/internal/gen/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestUpsertTemplateMenuPreservesNewDirectoryType(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:admin-menu-directory?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE video_menu (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		parent_id INTEGER NOT NULL DEFAULT 0,
		name TEXT NOT NULL,
		path TEXT NOT NULL DEFAULT '',
		component TEXT NOT NULL DEFAULT '',
		icon TEXT NOT NULL DEFAULT '',
		sort INTEGER NOT NULL DEFAULT 0,
		type INTEGER NOT NULL DEFAULT 1,
		permission TEXT NOT NULL DEFAULT '',
		visible INTEGER NOT NULL DEFAULT 1,
		status INTEGER NOT NULL DEFAULT 1,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME NULL
	)`).Error; err != nil {
		t.Fatal(err)
	}

	menu, err := upsertTemplateMenu(db, model.VideoMenu{
		Name: "运营管理", Path: "/operation", Type: 0, Visible: 1, Status: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if menu.Type != 0 {
		t.Fatalf("returned menu type = %d, want directory type 0", menu.Type)
	}
	var storedType uint8
	if err := db.Raw("SELECT type FROM video_menu WHERE id = ?", menu.ID).Scan(&storedType).Error; err != nil {
		t.Fatal(err)
	}
	if storedType != 0 {
		t.Fatalf("stored menu type = %d, want directory type 0", storedType)
	}
}

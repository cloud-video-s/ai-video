package app

import (
	"testing"

	"ai-video/internal/gen/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestUpsertSuperAdminRoleCreatesAndRepairsBuiltinRole(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:super-admin-role?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE video_role (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		code TEXT NOT NULL UNIQUE,
		sort INTEGER NOT NULL DEFAULT 0,
		status INTEGER NOT NULL DEFAULT 1,
		remark TEXT NOT NULL DEFAULT '',
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME NULL
	)`).Error; err != nil {
		t.Fatal(err)
	}

	role, err := upsertSuperAdminRole(db)
	if err != nil {
		t.Fatal(err)
	}
	if role.Code != "admin" || role.Status != 1 {
		t.Fatalf("created role=%+v", role)
	}
	if err := db.Model(&model.VideoRole{}).Where("id = ?", role.ID).Updates(map[string]interface{}{
		"name": "broken", "status": 0, "deleted_at": "2026-08-04 00:00:00",
	}).Error; err != nil {
		t.Fatal(err)
	}
	repaired, err := upsertSuperAdminRole(db)
	if err != nil {
		t.Fatal(err)
	}
	if repaired.ID != role.ID || repaired.Name != "超级管理员" || repaired.Status != 1 || repaired.DeletedAt.Valid {
		t.Fatalf("repaired role=%+v", repaired)
	}
}

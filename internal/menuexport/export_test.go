package menuexport

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestGenerateMenuSnapshot(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:menu-export?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open isolated database: %v", err)
	}
	if err := db.Exec(`CREATE TABLE video_menu (
		id INTEGER PRIMARY KEY, parent_id INTEGER NOT NULL, name TEXT NOT NULL,
		path TEXT NOT NULL, component TEXT NOT NULL, icon TEXT NOT NULL,
		sort INTEGER NOT NULL, type INTEGER NOT NULL, permission TEXT NOT NULL,
		visible INTEGER NOT NULL, status INTEGER NOT NULL,
		created_at DATETIME NOT NULL, updated_at DATETIME NOT NULL, deleted_at DATETIME NULL
	)`).Error; err != nil {
		t.Fatalf("create isolated video_menu: %v", err)
	}
	now := time.Now().Truncate(time.Second)
	if err := db.Exec(`INSERT INTO video_menu
		(id, parent_id, name, path, component, icon, sort, type, permission, visible, status, created_at, updated_at, deleted_at)
		VALUES (?, 0, '用户中心', '/user', '', 'User', 1, 0, '', 1, 1, ?, ?, NULL),
		       (?, 1, '旧菜单', '/old', '', '', 2, 1, 'old:list', 1, 0, ?, ?, ?)`,
		1, now, now, 2, now, now, now).Error; err != nil {
		t.Fatalf("insert isolated menu rows: %v", err)
	}

	var active bytes.Buffer
	count, err := Generate(context.Background(), db, &active, false)
	if err != nil {
		t.Fatalf("generate active snapshot: %v", err)
	}
	if count != 1 {
		t.Fatalf("active count = %d, want 1", count)
	}
	var snapshot Snapshot
	if err := json.Unmarshal(active.Bytes(), &snapshot); err != nil {
		t.Fatalf("decode snapshot: %v", err)
	}
	if snapshot.SourceTable != "video_menu" || len(snapshot.Menus) != 1 || snapshot.Menus[0].Path != "/user" {
		t.Fatalf("unexpected snapshot: %+v", snapshot)
	}

	var all bytes.Buffer
	count, err = Generate(context.Background(), db, &all, true)
	if err != nil {
		t.Fatalf("generate full snapshot: %v", err)
	}
	if count != 2 {
		t.Fatalf("full count = %d, want 2", count)
	}
}

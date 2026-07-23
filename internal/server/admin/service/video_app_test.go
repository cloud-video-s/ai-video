package service

import (
	"context"
	"testing"

	"ai-video/internal/config"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestVideoAppServiceCRUD(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:video-app-service?mode=memory&cache=shared"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	config.DB = db
	if err := db.Exec(`CREATE TABLE video_app (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		app_code TEXT NOT NULL,
		status INTEGER NOT NULL DEFAULT 1,
		sort INTEGER NOT NULL DEFAULT 0,
		description TEXT,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE video_package (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		package_name TEXT NOT NULL,
		package_code TEXT NOT NULL,
		app_code TEXT NOT NULL,
		description TEXT,
		sort INTEGER NOT NULL DEFAULT 0,
		status INTEGER NOT NULL DEFAULT 1,
		system_type INTEGER NOT NULL DEFAULT 1,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	)`).Error; err != nil {
		t.Fatal(err)
	}

	service := NewVideoAppService()
	ctx := context.Background()
	created, err := service.Create(ctx, &VideoAppPayload{
		Name: "AI Video", AppCode: "ai.video", Status: 1, Sort: 10, Description: "Video app",
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.ID == 0 || created.AppCode != "ai.video" || created.Sort != 10 || created.Description != "Video app" {
		t.Fatalf("unexpected create: %#v", created)
	}

	status := uint32(1)
	list, total, err := service.List(ctx, 1, 20, &ListVideoAppRequest{Keyword: "Video", AppCode: "ai.video", Status: &status})
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(list) != 1 || list[0].Name != "AI Video" || list[0].AppCode != "ai.video" {
		t.Fatalf("unexpected list: total=%d list=%#v", total, list)
	}

	updated, err := service.Update(ctx, created.ID, &VideoAppPayload{
		Name: "AI Studio", AppCode: "ai.studio", Status: 0, Sort: 20, Description: "Studio app",
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != "AI Studio" || updated.AppCode != "ai.studio" || updated.Status != 0 || updated.Sort != 20 || updated.Description != "Studio app" {
		t.Fatalf("unexpected update: %#v", updated)
	}
	if err := db.Exec(`INSERT INTO video_package
		(package_name, package_code, app_code, status, system_type)
		VALUES (?, ?, ?, ?, ?)`, "Android", "com.example.video", updated.AppCode, 1, 2).Error; err != nil {
		t.Fatal(err)
	}
	if err := service.Delete(ctx, created.ID); err == nil {
		t.Fatal("app with dependent packages must not be deleted")
	}
	if err := db.Exec("DELETE FROM video_package WHERE app_code = ?", updated.AppCode).Error; err != nil {
		t.Fatal(err)
	}
	if err := service.Delete(ctx, created.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := service.GetByID(ctx, created.ID); err == nil {
		t.Fatal("deleted app must not be returned")
	}
}

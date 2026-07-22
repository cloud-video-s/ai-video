package service

import (
	"context"
	"testing"
	"time"

	"ai-video/internal/app"
	"ai-video/internal/config"
	"ai-video/internal/gen/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestUserCenterVIPAndAccessOperations(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:user-center-operations?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	config.DB = db
	if err := db.Exec(`CREATE TABLE video_user (
		id INTEGER PRIMARY KEY, device_code TEXT, imei TEXT, username TEXT, app_name TEXT,
		status INTEGER NOT NULL DEFAULT 1, token_version INTEGER NOT NULL DEFAULT 0,
		vip_level INTEGER NOT NULL DEFAULT 0, vip_started_at DATETIME, vip_expires_at DATETIME,
		user_type INTEGER NOT NULL DEFAULT 1, subscription_status INTEGER NOT NULL DEFAULT 1,
		is_frozen BOOLEAN NOT NULL DEFAULT 0, is_blacklisted BOOLEAN NOT NULL DEFAULT 0,
		created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
	)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO video_user (id, device_code, imei, username, app_name) VALUES (1, 'device-1', 'imei-1', 'legacy', 'app')`).Error; err != nil {
		t.Fatal(err)
	}
	user := model.VideoUser{ID: 1}

	service := NewAppUserService()
	expiresAt := time.Now().Add(30 * 24 * time.Hour)
	if err := service.GrantVIP(context.Background(), user.ID, &GrantUserVIPRequest{Level: 3, ExpiresAt: expiresAt}); err != nil {
		t.Fatal(err)
	}
	if err := service.SetFrozen(context.Background(), user.ID, true); err != nil {
		t.Fatal(err)
	}

	updated, err := service.GetByID(context.Background(), user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.VIPLevel != 3 || updated.VipExpiresAt == nil || !updated.IsFrozen || updated.Status != 0 {
		t.Fatalf("unexpected updated user: level=%d expires=%v frozen=%v status=%d", updated.VIPLevel, updated.VipExpiresAt, updated.IsFrozen, updated.Status)
	}
	if updated.TokenVersion != 1 {
		t.Fatalf("token version=%d, want 1", updated.TokenVersion)
	}
	if _, _, err := service.repo.GetAuthState(context.Background(), user.ID); err == nil {
		t.Fatal("frozen user must not have a valid auth state")
	}
}

func TestMigrateUserCenterColumnsPreservesLegacyUsers(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:user-center-migration?mode=memory&cache=shared"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE video_user (
		id INTEGER PRIMARY KEY, status INTEGER, user_type INTEGER, subscription_status INTEGER,
		vip_expires_at DATETIME, google_email TEXT, appid_email TEXT
	)`).Error; err != nil {
		t.Fatal(err)
	}
	future := time.Now().Add(24 * time.Hour)
	if err := db.Exec(`INSERT INTO video_user (id, status, user_type, subscription_status, vip_expires_at, google_email)
		VALUES (7, 0, 1, 1, ?, 'legacy@example.com')`, future).Error; err != nil {
		t.Fatal(err)
	}
	if err := app.MigrateUserCenterColumns(db); err != nil {
		t.Fatal(err)
	}
	for _, column := range []string{"vip_expires_at", "user_type", "subscription_status", "vip_level"} {
		if !db.Migrator().HasColumn("video_user", column) {
			t.Errorf("missing column after migration: %s", column)
		}
	}
	var row struct {
		ID       uint64
		Email    string
		VIPLevel uint32 `gorm:"column:vip_level"`
		IsFrozen bool
	}
	if err := db.Table("video_user").First(&row, 7).Error; err != nil {
		t.Fatal(err)
	}
	if row.ID != 7 || row.Email != "legacy@example.com" || row.VIPLevel != 1 || !row.IsFrozen {
		t.Fatalf("legacy row was not preserved/backfilled: %#v", row)
	}
}

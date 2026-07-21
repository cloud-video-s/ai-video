package service

import (
	"ai-video/internal/config"
	"context"
	"testing"

	"ai-video/internal/pkg/cache"
	"ai-video/internal/pkg/setting"
	"ai-video/internal/repository"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestPrepareLoginSessionHonorsSingleDeviceConfig(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:single-device-login?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	config.DB = db
	previousStore := cache.GetStore()
	cache.InitStore(nil)
	t.Cleanup(func() { cache.InitStore(previousStore) })
	if err := db.Exec(`CREATE TABLE video_user (
		id INTEGER PRIMARY KEY, imei TEXT NOT NULL, username TEXT NOT NULL,
		token_version INTEGER NOT NULL DEFAULT 0, deleted_at DATETIME
	)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE video_config (
		id INTEGER PRIMARY KEY, key TEXT NOT NULL UNIQUE, value TEXT, deleted_at DATETIME
	)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO video_user (id, imei, username, token_version) VALUES (1, 'device-1', 'user-1', 3)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO video_config (id, key, value) VALUES (1, ?, 'true')`, setting.UserSingleDeviceLoginKey).Error; err != nil {
		t.Fatal(err)
	}

	svc := &AuthService{userRepo: repository.NewAppUserRepo()}
	user, err := svc.prepareLoginSession(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if user.TokenVersion != 4 {
		t.Fatalf("enabled token version=%d, want 4", user.TokenVersion)
	}

	if err := db.Exec(`UPDATE video_config SET value = 'false' WHERE key = ?`, setting.UserSingleDeviceLoginKey).Error; err != nil {
		t.Fatal(err)
	}
	user, err = svc.prepareLoginSession(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if user.TokenVersion != 4 {
		t.Fatalf("disabled token version=%d, want unchanged 4", user.TokenVersion)
	}
}

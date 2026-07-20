package repository

import (
	"context"
	"testing"

	"ai-video/internal/app"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestIncrementTokenVersion(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:increment-token-version?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	app.DB = db
	if err := db.Exec(`CREATE TABLE video_user (id INTEGER PRIMARY KEY, token_version INTEGER NOT NULL DEFAULT 0, deleted_at DATETIME)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO video_user (id, token_version) VALUES (1, 7)`).Error; err != nil {
		t.Fatal(err)
	}

	repo := NewAppUserRepo()
	if err := repo.IncrementTokenVersion(context.Background(), 1); err != nil {
		t.Fatal(err)
	}
	var version int64
	if err := db.Raw(`SELECT token_version FROM video_user WHERE id = 1`).Scan(&version).Error; err != nil {
		t.Fatal(err)
	}
	if version != 8 {
		t.Fatalf("token version=%d, want 8", version)
	}
	if err := repo.IncrementTokenVersion(context.Background(), 99); err != gorm.ErrRecordNotFound {
		t.Fatalf("missing user error=%v, want record not found", err)
	}
}

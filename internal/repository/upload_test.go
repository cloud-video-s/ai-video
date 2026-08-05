package repository

import (
	"context"
	"strings"
	"testing"

	"ai-video/internal/config"
	"ai-video/internal/domain"
	"ai-video/internal/pkg/upload"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestUploadRepoPreUploadConfirmationAndBackendRecording(t *testing.T) {
	db := newUploadRepoTestDB(t)
	repo := NewUploadRepo()
	owner := upload.UploadOwner{Type: upload.UploaderAPIUser, ID: 19}

	pending := upload.DirectPreUpload{
		Owner: owner,
		Request: upload.DirectUploadRequest{
			MediaType: upload.MediaImage, FileName: "avatar.png", Size: 123, ContentType: "image/png",
		},
		Credential: upload.DirectUploadCredential{
			UploadID: "1234567890abcdef1234567890abcdef", Provider: upload.StorageAliyunOSS,
			ObjectKey: "uploads/images/2026/08/05/1234567890abcdef1234567890abcdef.png",
			FileURL:   "https://cdn.example.com/uploads/images/2026/08/05/1234567890abcdef1234567890abcdef.png",
		},
	}
	if err := repo.RecordDirectPreUpload(context.Background(), pending); err != nil {
		t.Fatal(err)
	}

	var row struct {
		Status  int8
		FileURL string
	}
	if err := db.Table("video_upload").Select("status", "file_url").
		Where("upload_id = ?", pending.Credential.UploadID).Take(&row).Error; err != nil {
		t.Fatal(err)
	}
	if row.Status != domain.UploadStatusIncomplete || row.FileURL != "/uploads/images/2026/08/05/1234567890abcdef1234567890abcdef.png" {
		t.Fatalf("pre-upload row = %+v", row)
	}

	owned, err := repo.OwnedHalfURLs(context.Background(), owner, []string{pending.Credential.FileURL})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := owned[row.FileURL]; !ok {
		t.Fatalf("owned URLs = %#v", owned)
	}
	if err := repo.ConfirmUploadedByURLs(context.Background(), upload.UploadOwner{
		Type: upload.UploaderAPIUser, ID: 20,
	}, []string{row.FileURL}); err != nil {
		t.Fatal(err)
	}
	if err := db.Table("video_upload").Select("status").Where("upload_id = ?", pending.Credential.UploadID).Take(&row).Error; err != nil {
		t.Fatal(err)
	}
	if row.Status != domain.UploadStatusIncomplete {
		t.Fatalf("another user changed status to %d", row.Status)
	}
	if err := repo.ConfirmUploadedByURLs(context.Background(), owner, []string{pending.Credential.FileURL}); err != nil {
		t.Fatal(err)
	}
	if err := db.Table("video_upload").Select("status").Where("upload_id = ?", pending.Credential.UploadID).Take(&row).Error; err != nil {
		t.Fatal(err)
	}
	if row.Status != domain.UploadStatusCompleted {
		t.Fatalf("confirmed status = %d", row.Status)
	}

	stored := upload.StoredUpload{
		Owner: owner, Kind: upload.MediaVideo, OriginalName: "result.mp4", ContentType: "video/mp4",
		FileSize: 456, SHA256: strings.Repeat("a", 64),
		Stored: upload.StoredFile{
			Provider: upload.StorageAliyunOSS, Path: "uploads/generated/19/result.mp4",
			URL: "https://cdn.example.com/uploads/generated/19/result.mp4",
		},
	}
	if err := repo.RecordStored(context.Background(), stored); err != nil {
		t.Fatal(err)
	}
	if err := repo.RecordStored(context.Background(), stored); err != nil {
		t.Fatal(err)
	}
	var count int64
	if err := db.Table("video_upload").Where("file_path = ? AND status = ?", stored.Stored.Path, domain.UploadStatusCompleted).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("backend upload rows = %d, want 1", count)
	}
}

func newUploadRepoTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + strings.ReplaceAll(t.Name(), "/", "-") + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE video_upload (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		upload_id TEXT NOT NULL UNIQUE,
		user_type INTEGER NOT NULL,
		user_id INTEGER NOT NULL,
		media_type TEXT NOT NULL,
		file_type TEXT NOT NULL,
		mime_type TEXT NOT NULL,
		original_name TEXT NOT NULL,
		file_size INTEGER NOT NULL,
		storage_provider TEXT NOT NULL,
		file_path TEXT NOT NULL,
		file_url TEXT NOT NULL,
		sha256 TEXT NOT NULL,
		status INTEGER NOT NULL DEFAULT 1,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL,
		deleted_at DATETIME NULL
	)`).Error; err != nil {
		t.Fatal(err)
	}
	previousDB := config.DB
	config.DB = db
	t.Cleanup(func() {
		config.DB = previousDB
		if sqlDB, dbErr := db.DB(); dbErr == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

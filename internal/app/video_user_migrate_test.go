package app

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestPrepareVideoUserColumnsBackfillsNullPackageCode(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:video-user-package-code-migrate?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE video_user (
		id INTEGER PRIMARY KEY,
		package_code VARCHAR(128) NULL
	)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("INSERT INTO video_user (id, package_code) VALUES (1, NULL), (2, 'com.example.app')").Error; err != nil {
		t.Fatal(err)
	}

	if err := PrepareVideoUserColumns(db); err != nil {
		t.Fatal(err)
	}
	if err := PrepareVideoUserColumns(db); err != nil {
		t.Fatalf("second migration must be idempotent: %v", err)
	}

	var values []string
	if err := db.Table("video_user").Order("id").Pluck("package_code", &values).Error; err != nil {
		t.Fatal(err)
	}
	if len(values) != 2 || values[0] != "" || values[1] != "com.example.app" {
		t.Fatalf("package_code values = %#v, want empty legacy value and preserved non-empty value", values)
	}
}

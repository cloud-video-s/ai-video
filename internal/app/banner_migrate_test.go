package app

import (
	"testing"

	"ai-video/internal/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestMigrateLegacyBannerPositionsBackfillsAssociation(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:banner-position-association-migrate?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.VideoDisplayPosition{}, &model.VideoBanner{}); err != nil {
		t.Fatal(err)
	}
	if err := EnsureBannerPositionKeyColumn(db); err != nil {
		t.Fatal(err)
	}
	position := model.VideoDisplayPosition{
		PositionName: "Home", PositionKey: "home", CoverImage: "https://example.com/home.jpg", Status: 1,
	}
	if err := db.Create(&position).Error; err != nil {
		t.Fatal(err)
	}
	banner := model.VideoBanner{
		Name: "Legacy", CoverImage: "https://example.com/banner.jpg",
		JumpType: model.BannerJumpTypeLink, JumpURL: "/home", Status: 1,
	}
	if err := db.Create(&banner).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Table("video_banner").Where("id = ?", banner.ID).Update("position_key", "home").Error; err != nil {
		t.Fatal(err)
	}

	if err := MigrateLegacyBannerPositions(db); err != nil {
		t.Fatal(err)
	}
	if err := MigrateLegacyBannerPositions(db); err != nil {
		t.Fatal(err)
	}
	if count := db.Model(&banner).Association("DisplayPositions").Count(); count != 1 {
		t.Fatalf("display position association count = %d, want 1", count)
	}
}

func TestMigrateBannerDisplayPositionKeysPreservesOldAssociations(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:banner-position-key-migrate?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.VideoDisplayPosition{}); err != nil {
		t.Fatal(err)
	}
	position := model.VideoDisplayPosition{PositionName: "Home", PositionKey: "home", CoverImage: "/home.jpg", Status: 1}
	if err := db.Create(&position).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("CREATE TABLE video_banner_display_position (banner_id INTEGER NOT NULL, display_position_id INTEGER NOT NULL, PRIMARY KEY (banner_id, display_position_id))").Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("INSERT INTO video_banner_display_position (banner_id, display_position_id) VALUES (?, ?)", 10, position.ID).Error; err != nil {
		t.Fatal(err)
	}

	if err := MigrateBannerDisplayPositionKeys(db); err != nil {
		t.Fatal(err)
	}
	if !db.Migrator().HasColumn("video_banner_display_position", "position_key") || db.Migrator().HasColumn("video_banner_display_position", "display_position_id") {
		t.Fatal("association table was not converted to position_key")
	}
	var row model.VideoBannerDisplayPosition
	if err := db.First(&row, "banner_id = ?", 10).Error; err != nil {
		t.Fatal(err)
	}
	if row.PositionKey != "home" {
		t.Fatalf("position_key = %q, want home", row.PositionKey)
	}
	if err := MigrateBannerDisplayPositionKeys(db); err != nil {
		t.Fatalf("second migration must be idempotent: %v", err)
	}
}

func TestNormalizeBannerJumpTypesConvertsLegacyStrings(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:banner-jump-migrate?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("CREATE TABLE video_banner (id INTEGER PRIMARY KEY, jump_type VARCHAR(32))").Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("INSERT INTO video_banner (id, jump_type) VALUES (1, 'link'), (2, 'template'), (3, 'text_to_image'), (4, 'text_to_video')").Error; err != nil {
		t.Fatal(err)
	}

	if err := NormalizeBannerJumpTypes(db); err != nil {
		t.Fatal(err)
	}
	var rows []struct {
		ID       uint64
		JumpType string
	}
	if err := db.Table("video_banner").Order("id ASC").Find(&rows).Error; err != nil {
		t.Fatal(err)
	}
	for i := range rows {
		want := string(rune('1' + i))
		if rows[i].JumpType != want {
			t.Fatalf("row %d jump_type = %q, want %q", rows[i].ID, rows[i].JumpType, want)
		}
	}
}

func TestNormalizeBannerJumpTypesRejectsUnknownValue(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:banner-jump-invalid?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("CREATE TABLE video_banner (id INTEGER PRIMARY KEY, jump_type TEXT)").Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("INSERT INTO video_banner (id, jump_type) VALUES (1, 'unknown')").Error; err != nil {
		t.Fatal(err)
	}
	if err := NormalizeBannerJumpTypes(db); err == nil {
		t.Fatal("unsupported legacy jump type was accepted")
	}
}

func TestEnsureBannerPositionKeyColumnPreservesExistingRows(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:banner-position-migrate?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("CREATE TABLE video_banner (id INTEGER PRIMARY KEY, name TEXT NOT NULL)").Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("INSERT INTO video_banner (id, name) VALUES (1, 'legacy')").Error; err != nil {
		t.Fatal(err)
	}
	if err := EnsureBannerPositionKeyColumn(db); err != nil {
		t.Fatal(err)
	}
	var positionKey string
	if err := db.Table("video_banner").Select("position_key").Where("id = ?", 1).Scan(&positionKey).Error; err != nil {
		t.Fatal(err)
	}
	if positionKey != "" {
		t.Fatalf("legacy row position_key = %q, want empty migration placeholder", positionKey)
	}
}

package integration_test

import (
	"context"
	"testing"

	"ai-video/internal/app"
	"ai-video/internal/model"
	"ai-video/internal/repository"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestBannerDisplayPositionUsesPositionKeyEndToEnd(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:banner-position-key-integration?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	app.DB = db
	if err := db.AutoMigrate(
		&model.VideoTemplateType{}, &model.VideoCountry{}, &model.VideoPackage{},
		&model.VideoChannel{}, &model.VideoDisplayPosition{}, &model.VideoTemplate{},
		&model.VideoBanner{}, &model.VideoBannerDisplayPosition{},
	); err != nil {
		t.Fatal(err)
	}

	position := model.VideoDisplayPosition{
		PositionName: "Home", PositionKey: "home", CoverImage: "/home.jpg", Status: 1,
	}
	if err := db.Create(&position).Error; err != nil {
		t.Fatal(err)
	}
	banner := model.VideoBanner{
		Name: "Home Banner", CoverImage: "/banner.jpg",
		JumpType: model.BannerJumpTypeLink, JumpURL: "/home", Status: 1,
	}
	repo := repository.NewBannerRepo()
	if err := repo.Create(context.Background(), &banner); err != nil {
		t.Fatal(err)
	}
	if err := repo.ReplaceTargets(context.Background(), &banner, repository.BannerTargetIDs{
		DisplayPositionKeys: []string{"home"},
	}); err != nil {
		t.Fatal(err)
	}

	var relation model.VideoBannerDisplayPosition
	if err := db.First(&relation, "banner_id = ?", banner.ID).Error; err != nil {
		t.Fatal(err)
	}
	if relation.PositionKey != "home" {
		t.Fatalf("stored position_key = %q, want home", relation.PositionKey)
	}
	detail, err := repo.GetDetail(context.Background(), banner.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(detail.DisplayPositions) != 1 || detail.DisplayPositions[0].PositionKey != "home" {
		t.Fatalf("preloaded display positions = %#v", detail.DisplayPositions)
	}
	list, err := repo.ListForClient(context.Background(), repository.ClientBannerTargets{PositionKey: "home"})
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].ID != banner.ID {
		t.Fatalf("client banner list = %#v", list)
	}
}

func TestLegacyBannerDisplayPositionIDMigratesToPositionKey(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:banner-position-key-legacy-integration?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.VideoDisplayPosition{}); err != nil {
		t.Fatal(err)
	}
	position := model.VideoDisplayPosition{
		PositionName: "Home", PositionKey: "home", CoverImage: "/home.jpg", Status: 1,
	}
	if err := db.Create(&position).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("CREATE TABLE video_banner_display_position (banner_id INTEGER NOT NULL, display_position_id INTEGER NOT NULL, PRIMARY KEY (banner_id, display_position_id))").Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("INSERT INTO video_banner_display_position (banner_id, display_position_id) VALUES (?, ?)", 9, position.ID).Error; err != nil {
		t.Fatal(err)
	}

	if err := app.MigrateBannerDisplayPositionKeys(db); err != nil {
		t.Fatal(err)
	}
	if db.Migrator().HasColumn("video_banner_display_position", "display_position_id") {
		t.Fatal("legacy display_position_id column still exists")
	}
	var relation model.VideoBannerDisplayPosition
	if err := db.First(&relation, "banner_id = ?", 9).Error; err != nil {
		t.Fatal(err)
	}
	if relation.PositionKey != "home" {
		t.Fatalf("migrated position_key = %q, want home", relation.PositionKey)
	}
	if err := db.Exec("CREATE TABLE video_banner (id INTEGER PRIMARY KEY, position_key VARCHAR(100) NOT NULL DEFAULT '')").Error; err != nil {
		t.Fatal(err)
	}
	if err := app.RemoveLegacyBannerPositionKey(db); err != nil {
		t.Fatal(err)
	}
	if db.Migrator().HasColumn("video_banner", "position_key") {
		t.Fatal("legacy video_banner.position_key column still exists")
	}
}

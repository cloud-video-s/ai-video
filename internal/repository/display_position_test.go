package repository

import (
	"context"
	"testing"

	"ai-video/internal/app"
	"ai-video/internal/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestDisplayPositionOptionsOnlyReturnEnabledRows(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:display-position-options?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	app.DB = db
	if err := db.AutoMigrate(&model.VideoDisplayPosition{}); err != nil {
		t.Fatal(err)
	}
	rows := []model.VideoDisplayPosition{
		{PositionName: "Enabled", PositionKey: "enabled", CoverImage: "https://example.com/enabled.jpg", Status: 1},
		{PositionName: "Disabled", PositionKey: "disabled", CoverImage: "https://example.com/disabled.jpg", Status: 1},
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&rows[1]).Update("status", 0).Error; err != nil {
		t.Fatal(err)
	}

	options, err := NewDisplayPositionRepo().ListOptions(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(options) != 1 || options[0].PositionKey != "enabled" {
		t.Fatalf("display position options = %#v", options)
	}
}

func TestDisplayPositionBannerCountUsesAssociation(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:display-position-banner-rename?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	app.DB = db
	if err := db.AutoMigrate(&model.VideoDisplayPosition{}, &model.VideoBannerDisplayPosition{}); err != nil {
		t.Fatal(err)
	}
	positions := []model.VideoDisplayPosition{
		{ID: 8, PositionName: "Home", PositionKey: "home", CoverImage: "/home.jpg", Status: 1},
		{ID: 9, PositionName: "Profile", PositionKey: "profile", CoverImage: "/profile.jpg", Status: 1},
	}
	if err := db.Create(&positions).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&[]model.VideoBannerDisplayPosition{
		{BannerID: 1, PositionKey: "home"},
		{BannerID: 2, PositionKey: "home"},
		{BannerID: 3, PositionKey: "profile"},
	}).Error; err != nil {
		t.Fatal(err)
	}

	repo := NewDisplayPositionRepo()
	count, err := repo.BannerCount(context.Background(), "home")
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("associated Banner count = %d, want 2", count)
	}
}

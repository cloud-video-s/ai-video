package repository

import (
	"context"
	"testing"

	"ai-video/internal/config"
	"ai-video/internal/gen/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestPointsPackageTargetsFilterAndDefaultScope(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:points-package-targets?mode=memory&cache=shared&_time_format=sqlite"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	config.DB = db
	statements := []string{
		`CREATE TABLE video_package (
			id INTEGER PRIMARY KEY, package_name TEXT NOT NULL, package_code TEXT NOT NULL,
			app_code TEXT NOT NULL, description TEXT, sort INTEGER NOT NULL DEFAULT 0,
			status INTEGER NOT NULL DEFAULT 1, system_type INTEGER NOT NULL DEFAULT 1,
			created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
		)`,
		`CREATE TABLE video_channel (
			channel_id INTEGER PRIMARY KEY, channel_code TEXT NOT NULL, channel_name TEXT NOT NULL,
			agency_company TEXT, ad_platform TEXT, delivery_package TEXT, tracking_url TEXT,
			port_rebate NUMERIC NOT NULL DEFAULT 0, service_order_fee NUMERIC NOT NULL DEFAULT 0,
			upload_method TEXT, status INTEGER NOT NULL DEFAULT 1,
			description TEXT, sort INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
		)`,
		`CREATE TABLE video_points_package (
			id INTEGER PRIMARY KEY, product_code TEXT NOT NULL UNIQUE, name TEXT NOT NULL,
			systems TEXT, user_types TEXT, resource_type TEXT NOT NULL, points INTEGER NOT NULL,
			currency TEXT NOT NULL, sale_price NUMERIC NOT NULL DEFAULT 0, actual_revenue NUMERIC NOT NULL DEFAULT 0,
			original_price NUMERIC NOT NULL DEFAULT 0, badge_text TEXT, description TEXT, button_text TEXT,
			is_default INTEGER NOT NULL DEFAULT 0, status INTEGER NOT NULL DEFAULT 1, sort INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
		)`,
		`CREATE TABLE video_points_package_package (
			id INTEGER PRIMARY KEY AUTOINCREMENT, product_code TEXT NOT NULL, package_code TEXT NOT NULL,
			created_at DATETIME NOT NULL, updated_at DATETIME NOT NULL, deleted_at DATETIME
		)`,
		`CREATE TABLE video_points_package_channel (
			id INTEGER PRIMARY KEY AUTOINCREMENT, product_code TEXT NOT NULL, channel_code TEXT NOT NULL,
			created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
		)`,
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatal(err)
		}
	}

	packages := []model.VideoPackage{
		{ID: 1, PackageName: "App A", PackageCode: "app.a", AppCode: "video", SystemType: 2, Status: 1},
		{ID: 2, PackageName: "App B", PackageCode: "app.b", AppCode: "video", SystemType: 1, Status: 1},
	}
	channel := model.VideoChannel{ChannelID: 1, ChannelCode: "organic", ChannelName: "Organic", AdPlatform: "direct", DeliveryPackage: "app.a", UploadMethod: "API", Status: 1}
	if err := db.Create(&packages).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&channel).Error; err != nil {
		t.Fatal(err)
	}

	repo := NewPointsPackageRepo()
	items := []model.VideoPointsPackage{
		{ID: 11, ProductID: "credits_a_1", Name: "Credits A1", Systems: []string{"android"}, UserTypes: []int{1, 2}, ResourceType: "credits", Points: 100, Currency: "USD", Status: 1},
		{ID: 12, ProductID: "credits_a_2", Name: "Credits A2", Systems: []string{"android"}, UserTypes: []int{1, 2}, ResourceType: "credits", Points: 200, Currency: "USD", Status: 1},
		{ID: 13, ProductID: "credits_b_1", Name: "Credits B1", Systems: []string{"ios"}, UserTypes: []int{1, 2}, ResourceType: "credits", Points: 300, Currency: "USD", Status: 1},
	}
	ctx := context.Background()
	for i := range items {
		if err := repo.Create(ctx, &items[i]); err != nil {
			t.Fatal(err)
		}
		packageID := uint64(1)
		channelIDs := []uint64{channel.ChannelID}
		if i == 2 {
			packageID = 2
			channelIDs = nil
		}
		if err := repo.ReplaceTargets(ctx, &items[i], packageID, channelIDs); err != nil {
			t.Fatal(err)
		}
	}

	list, total, err := repo.PageList(ctx, 1, 20, &PointsPackageListFilter{PackageID: 1, ChannelID: 1})
	if err != nil {
		t.Fatal(err)
	}
	if total != 2 || len(list) != 2 || len(list[0].Packages) != 1 || len(list[0].Channels) != 1 {
		t.Fatalf("unexpected package targets: total=%d list=%#v", total, list)
	}

	first, err := repo.GetDetail(ctx, items[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	third, err := repo.GetDetail(ctx, items[2].ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.SetDefault(ctx, first); err != nil {
		t.Fatal(err)
	}
	if err := repo.SetDefault(ctx, third); err != nil {
		t.Fatal(err)
	}
	var defaultCount int64
	if err := db.Model(&model.VideoPointsPackage{}).Where("is_default = ?", true).Count(&defaultCount).Error; err != nil {
		t.Fatal(err)
	}
	if defaultCount != 2 {
		t.Fatalf("defaults across two app packages = %d, want 2", defaultCount)
	}

	second, err := repo.GetDetail(ctx, items[1].ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.SetDefault(ctx, second); err != nil {
		t.Fatal(err)
	}
	var firstReloaded model.VideoPointsPackage
	if err := db.First(&firstReloaded, items[0].ID).Error; err != nil {
		t.Fatal(err)
	}
	if firstReloaded.IsDefault {
		t.Fatal("previous default in the same app package was not cleared")
	}
}

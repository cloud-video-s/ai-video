package app

import (
	"testing"

	"ai-video/internal/gen/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestMigrateLegacyTemplateTypeTargets(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:template-type-target-migrate?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.VideoCountry{}, &model.VideoChannel{}, &model.VideoPackage{}, &model.VideoDisplayPosition{}, &model.VideoTemplateType{}); err != nil {
		t.Fatal(err)
	}
	for _, statement := range []string{
		"ALTER TABLE video_template_type ADD COLUMN country TEXT",
		"ALTER TABLE video_template_type ADD COLUMN channel_id TEXT",
		"ALTER TABLE video_template_type ADD COLUMN package_id INTEGER",
		"ALTER TABLE video_template_type ADD COLUMN app_package TEXT",
		"ALTER TABLE video_template_type ADD COLUMN user_type INTEGER",
		"ALTER TABLE video_template_type ADD COLUMN is_subscribed INTEGER",
	} {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatal(err)
		}
	}
	country := model.VideoCountry{Code: "CN", NameZh: "China", Status: 1}
	channel := model.VideoChannel{ChannelCode: "channel-a", ChannelName: "Channel A", AdPlatform: "direct", DeliveryPackage: "app.a", UploadMethod: "API", Status: 1}
	appPackage := model.VideoPackage{PackageName: "App A", PackageCode: "app.a", PackageVersion: "1.0.0", DownloadURL: "https://example.com/app", Status: 1}
	for _, value := range []interface{}{&country, &channel, &appPackage} {
		if err := db.Create(value).Error; err != nil {
			t.Fatal(err)
		}
	}
	item := model.VideoTemplateType{CategoryName: "Legacy", Status: 1}
	if err := db.Create(&item).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Table("video_template_type").Where("id = ?", item.ID).Updates(map[string]interface{}{
		"country": country.Code, "channel_id": channel.ChannelCode, "package_id": appPackage.ID,
		"user_type": 2, "is_subscribed": true,
	}).Error; err != nil {
		t.Fatal(err)
	}

	if err := MigrateLegacyTemplateTypeTargets(db); err != nil {
		t.Fatal(err)
	}
	if err := MigrateLegacyTemplateTypeTargets(db); err != nil {
		t.Fatalf("migration must be idempotent: %v", err)
	}
	var loaded model.VideoTemplateType
	if err := db.Preload("Countries").Preload("Channels").Preload("Packages").First(&loaded, item.ID).Error; err != nil {
		t.Fatal(err)
	}
	if len(loaded.Countries) != 1 || len(loaded.Channels) != 1 || len(loaded.Packages) != 1 {
		t.Fatalf("association counts: countries=%d channels=%d packages=%d", len(loaded.Countries), len(loaded.Channels), len(loaded.Packages))
	}
	if len(loaded.UserTypes) != 1 || loaded.UserTypes[0] != 2 || len(loaded.SubscriptionStatuses) != 1 || loaded.SubscriptionStatuses[0] != "subscribed" {
		t.Fatalf("migrated audience arrays: users=%v subscriptions=%v", loaded.UserTypes, loaded.SubscriptionStatuses)
	}
}

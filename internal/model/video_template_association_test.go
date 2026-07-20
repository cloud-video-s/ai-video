package model

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestVideoTemplateMultiTargetAssociations(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:template-targets?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	models := []interface{}{
		&VideoTemplateType{}, &VideoDisplayPosition{}, &VideoCountry{},
		&VideoPackage{}, &VideoChannel{}, &VideoTemplate{},
	}
	if err := db.AutoMigrate(models...); err != nil {
		t.Fatal(err)
	}

	typeItem := VideoTemplateType{CategoryName: "热门", UserTypes: []int{1, 2}, SubscriptionStatuses: []string{"subscribed", "unsubscribed"}, Status: 1}
	position := VideoDisplayPosition{PositionName: "首页", PositionKey: "home", CoverImage: "https://example.com/cover.jpg", Status: 1}
	country := VideoCountry{Code: "CN", NameZh: "中国", Status: 1}
	appPackage := VideoPackage{PackageName: "示例包", PackageCode: "com.example.app", PackageVersion: "1.0.0", DownloadURL: "https://example.com/app.apk", Status: 1}
	channel := VideoChannel{ChannelCode: "META_CN", ChannelName: "Meta 中国", AdPlatform: "Meta Ads", DeliveryPackage: "com.example.app", UploadMethod: "API", Status: 1}
	for _, value := range []interface{}{&typeItem, &position, &country, &appPackage, &channel} {
		if err := db.Create(value).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Model(&typeItem).Association("DisplayPositions").Append(&position); err != nil {
		t.Fatalf("append template type position: %v", err)
	}

	template := VideoTemplate{
		VideoTemplateTypeID: typeItem.ID, UserTypes: []int{1, 2},
		SubscriptionStatuses: []string{"subscribed", "unsubscribed"},
		Name:                 "测试模板", TemplateType: VideoTemplateKindAction,
		CoverImage: "https://example.com/template.jpg", TemplateVideo: "https://example.com/template.mp4", Status: 1,
	}
	if err := db.Create(&template).Error; err != nil {
		t.Fatal(err)
	}
	associations := []struct {
		name  string
		value interface{}
	}{
		{name: "Countries", value: &country},
		{name: "Packages", value: &appPackage},
		{name: "Channels", value: &channel},
	}
	for _, association := range associations {
		if err := db.Model(&template).Association(association.name).Append(association.value); err != nil {
			t.Fatalf("append %s: %v", association.name, err)
		}
	}

	for table, column := range map[string]string{
		"video_template_type_display_position": "position_key",
		"video_template_country":               "country_id",
		"video_template_package":               "package_id",
		"video_template_channel":               "channel_id",
	} {
		if !db.Migrator().HasColumn(table, column) {
			t.Fatalf("%s.%s was not created", table, column)
		}
	}

	var loaded VideoTemplate
	err = db.Preload("VideoTemplateType.DisplayPositions").Preload("Countries").Preload("Packages").Preload("Channels").First(&loaded, template.ID).Error
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.VideoTemplateType.DisplayPositions) != 1 || len(loaded.Countries) != 1 || len(loaded.Packages) != 1 || len(loaded.Channels) != 1 {
		t.Fatalf("unexpected association counts: type_positions=%d countries=%d packages=%d channels=%d",
			len(loaded.VideoTemplateType.DisplayPositions), len(loaded.Countries), len(loaded.Packages), len(loaded.Channels))
	}
}

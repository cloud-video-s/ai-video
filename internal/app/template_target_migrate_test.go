package app

import (
	"testing"

	"ai-video/internal/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestRepairTemplateUserTypesConvertsLegacyBase64(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:template-user-types?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.VideoTemplateType{}, &model.VideoDisplayPosition{}, &model.VideoCountry{}, &model.VideoPackage{}, &model.VideoChannel{}, &model.VideoTemplate{}); err != nil {
		t.Fatal(err)
	}
	typeItem := model.VideoTemplateType{CategoryName: "测试", Status: 1}
	if err := db.Create(&typeItem).Error; err != nil {
		t.Fatal(err)
	}
	item := model.VideoTemplate{
		VideoTemplateTypeID: typeItem.ID, UserTypes: []int{1, 2}, SubscriptionStatuses: []string{"subscribed", "unsubscribed"},
		Name: "模板", TemplateType: model.VideoTemplateKindAction, CoverImage: "https://example.com/a.jpg",
		TemplateVideo: "https://example.com/a.mp4", Status: 1,
	}
	if err := db.Create(&item).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Table("video_template").Where("id = ?", item.ID).Update("user_types", `"AQI="`).Error; err != nil {
		t.Fatal(err)
	}
	if err := repairTemplateUserTypes(db); err != nil {
		t.Fatal(err)
	}
	var raw string
	if err := db.Table("video_template").Select("user_types").Where("id = ?", item.ID).Scan(&raw).Error; err != nil {
		t.Fatal(err)
	}
	if raw != `[1,2]` {
		t.Fatalf("user_types = %q, want [1,2]", raw)
	}
}

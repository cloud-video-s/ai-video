package repository

import (
	"ai-video/internal/config"
	"context"
	"testing"

	"ai-video/internal/gen/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestTemplateDisplayConfigListForClientFiltersAndSorts(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:template-display-config-client?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	config.DB = db
	if err := db.AutoMigrate(
		&model.VideoCountry{}, &model.VideoChannel{}, &model.VideoPackage{},
		&model.VideoDisplayPosition{}, &model.VideoTemplateType{}, &model.VideoTemplate{},
		&model.VideoTemplateDisplayConfig{},
	); err != nil {
		t.Fatal(err)
	}

	position := model.VideoDisplayPosition{PositionName: "Home", PositionKey: "home", CoverImage: "/home.jpg", Status: 1}
	typeItem := model.VideoTemplateType{
		CategoryName: "热门", UserTypes: []int{1, 2},
		SubscriptionStatuses: []string{"subscribed", "unsubscribed"}, Status: 1,
	}
	if err := db.Create(&position).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&typeItem).Error; err != nil {
		t.Fatal(err)
	}
	templates := []model.VideoTemplate{
		{VideoTemplateTypeID: typeItem.ID, Name: "Second", TemplateType: "action", CoverImage: "/2.jpg", TemplateVideo: "/2.mp4", UserTypes: []int{1}, SubscriptionStatuses: []string{"unsubscribed"}, Status: 1},
		{VideoTemplateTypeID: typeItem.ID, Name: "First", TemplateType: "action", CoverImage: "/1.jpg", TemplateVideo: "/1.mp4", UserTypes: []int{1}, SubscriptionStatuses: []string{"unsubscribed"}, Status: 1},
		{VideoTemplateTypeID: typeItem.ID, Name: "Disabled template", TemplateType: "action", CoverImage: "/3.jpg", TemplateVideo: "/3.mp4", UserTypes: []int{1}, SubscriptionStatuses: []string{"unsubscribed"}, Status: 0},
	}
	if err := db.Create(&templates).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&templates[2]).Update("status", 0).Error; err != nil {
		t.Fatal(err)
	}
	configs := []model.VideoTemplateDisplayConfig{
		{TemplateID: templates[0].ID, DisplayPositionKey: "home", Sort: 10, Status: 1},
		{TemplateID: templates[1].ID, DisplayPositionKey: "home", Sort: 20, Status: 1},
		{TemplateID: templates[2].ID, DisplayPositionKey: "home", Sort: 30, Status: 1},
	}
	if err := db.Create(&configs).Error; err != nil {
		t.Fatal(err)
	}

	rows, err := NewTemplateDisplayConfigRepo().ListForClient(context.Background(), ClientTemplateDisplayTargets{
		PositionKey: "home", UserType: 1, SubscriptionState: "unsubscribed",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("configured template count = %d, want 2", len(rows))
	}
	if rows[0].Template.Name != "First" || rows[1].Template.Name != "Second" {
		t.Fatalf("configured template order = [%s %s], want [First Second]", rows[0].Template.Name, rows[1].Template.Name)
	}
}

func TestTemplateDisplayConfigPairExists(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:template-display-config-pair?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	config.DB = db
	if err := db.AutoMigrate(&model.VideoTemplateDisplayConfig{}); err != nil {
		t.Fatal(err)
	}
	row := model.VideoTemplateDisplayConfig{TemplateID: 7, DisplayPositionKey: "home", Status: 1}
	repo := NewTemplateDisplayConfigRepo()
	if err := repo.Create(context.Background(), &row); err != nil {
		t.Fatal(err)
	}
	exists, err := repo.PairExists(context.Background(), 7, "home", 0)
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Fatal("expected template-position pair to exist")
	}
}

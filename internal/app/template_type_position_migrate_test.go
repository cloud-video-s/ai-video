package app

import (
	"testing"

	"ai-video/internal/gen/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestMigrateTemplateTypeDisplayPositionKeys(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:template-type-position-key-migrate?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.VideoDisplayPosition{}, &model.VideoTemplateType{}); err != nil {
		t.Fatal(err)
	}
	position := model.VideoDisplayPosition{PositionName: "Home", PositionKey: "home", CoverImage: "https://example.com/home.jpg", Status: 1}
	templateType := model.VideoTemplateType{CategoryName: "Popular", UserTypes: []int{1, 2}, SubscriptionStatuses: []string{"subscribed", "unsubscribed"}, Status: 1}
	if err := db.Create(&position).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&templateType).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Migrator().DropTable("video_template_type_display_position"); err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("CREATE TABLE video_template_type_display_position (template_type_id INTEGER NOT NULL, display_position_id INTEGER NOT NULL, PRIMARY KEY (template_type_id, display_position_id))").Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("INSERT INTO video_template_type_display_position (template_type_id, display_position_id) VALUES (?, ?)", templateType.ID, position.ID).Error; err != nil {
		t.Fatal(err)
	}

	if err := MigrateTemplateTypeDisplayPositionKeys(db); err != nil {
		t.Fatal(err)
	}
	if !db.Migrator().HasColumn("video_template_type_display_position", "position_key") || db.Migrator().HasColumn("video_template_type_display_position", "display_position_id") {
		t.Fatal("join table was not converted to position_key")
	}
	var loaded model.VideoTemplateType
	if err := db.Preload("DisplayPositions").First(&loaded, templateType.ID).Error; err != nil {
		t.Fatal(err)
	}
	if len(loaded.DisplayPositions) != 1 || loaded.DisplayPositions[0].PositionKey != "home" {
		t.Fatalf("migrated positions = %#v", loaded.DisplayPositions)
	}
}

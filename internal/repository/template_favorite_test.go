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

func TestTemplateFavoriteIsIdempotentAndMaintainsCount(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:template-favorite?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent), TranslateError: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	config.DB = db
	if err := db.AutoMigrate(&model.VideoTemplate{}, &model.VideoUserTemplateFavorite{}); err != nil {
		t.Fatal(err)
	}
	template := model.VideoTemplate{ID: 9, VideoTemplateTypeID: 1, Name: "Favorite me", TemplateType: "action", CoverImage: "/cover.jpg", TemplateVideo: "/video.mp4", Status: 1}
	if err := db.Omit("CreatedAt", "UpdatedAt").Create(&template).Error; err != nil {
		t.Fatal(err)
	}

	repo := NewTemplateFavoriteRepo()
	ctx := context.Background()
	for i := 0; i < 2; i++ {
		state, err := repo.SetFavorite(ctx, 7, template.ID, true)
		if err != nil {
			t.Fatal(err)
		}
		if !state.Favorited || state.FavoriteCount != 1 {
			t.Fatalf("favorite state = %#v, want favorited with count 1", state)
		}
	}
	assertFavoriteRows(t, db, 1)

	for i := 0; i < 2; i++ {
		state, err := repo.SetFavorite(ctx, 7, template.ID, false)
		if err != nil {
			t.Fatal(err)
		}
		if state.Favorited || state.FavoriteCount != 0 {
			t.Fatalf("unfavorite state = %#v, want unfavorited with count 0", state)
		}
	}
	assertFavoriteRows(t, db, 0)
}

func assertFavoriteRows(t *testing.T, db *gorm.DB, want int64) {
	t.Helper()
	var count int64
	if err := db.Model(&model.VideoUserTemplateFavorite{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != want {
		t.Fatalf("favorite rows = %d, want %d", count, want)
	}
}

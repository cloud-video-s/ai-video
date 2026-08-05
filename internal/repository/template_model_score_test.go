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

func TestTemplateRepoPreloadsModelScoreForClientQueries(t *testing.T) {
	db := setupTemplateModelScoreDB(t)
	previousDB := config.DB
	config.DB = db
	t.Cleanup(func() { config.DB = previousDB })

	ctx := context.Background()
	repo := NewTemplateRepo()
	tests := []struct {
		name string
		load func() ([]model.VideoTemplate, error)
	}{
		{
			name: "category template page",
			load: func() ([]model.VideoTemplate, error) {
				rows, _, err := repo.GetPageList(ctx, 1, 10, &TemplateListRequest{TemplateTypeID: []uint64{5}})
				return templateValues(rows), err
			},
		},
		{
			name: "all client templates",
			load: func() ([]model.VideoTemplate, error) {
				return repo.ListForClient(ctx, ClientTemplateTargets{TemplateTypeIDs: []uint64{5}})
			},
		},
		{
			name: "category preview templates",
			load: func() ([]model.VideoTemplate, error) {
				return repo.GetListForClient(ctx, ClientTemplateTargets{TemplateTypeID: 5, Page: 1, PageSize: 10})
			},
		},
		{
			name: "template detail",
			load: func() ([]model.VideoTemplate, error) {
				item, err := repo.GetTemplateID(ctx, 10)
				if err != nil {
					return nil, err
				}
				return []model.VideoTemplate{*item}, nil
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rows, err := test.load()
			if err != nil {
				t.Fatal(err)
			}
			assertTemplateModelScore(t, rows)
		})
	}
}

func TestTemplateDisplayRecordsPreloadModelScore(t *testing.T) {
	db := setupTemplateModelScoreDB(t)
	previousDB := config.DB
	config.DB = db
	t.Cleanup(func() { config.DB = previousDB })

	records, err := NewTemplateDisplayConfigRepo().loadRecords(context.Background(), []model.VideoTemplatePlacementConfig{{
		ID: 1, TemplateID: 10, PlacementKey: "homeRecommend",
	}})
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 || records[0].Template == nil {
		t.Fatalf("display records = %#v", records)
	}
	assertTemplateModelScore(t, []model.VideoTemplate{records[0].Template.VideoTemplate})
}

func assertTemplateModelScore(t *testing.T, rows []model.VideoTemplate) {
	t.Helper()
	if len(rows) != 1 {
		t.Fatalf("template count = %d, want 1", len(rows))
	}
	if rows[0].AIModel.ID != 3 || rows[0].AIModel.Score != 42 {
		t.Fatalf("preloaded model = %#v, want id=3 score=42", rows[0].AIModel)
	}
}

func setupTemplateModelScoreDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	statements := []string{
		`CREATE TABLE video_model (
			id INTEGER PRIMARY KEY,
			score INTEGER NOT NULL,
			deleted_at DATETIME NULL
		)`,
		`CREATE TABLE video_template (
			id INTEGER PRIMARY KEY,
			template_type INTEGER NOT NULL,
			template_type_id INTEGER NOT NULL,
			model_id INTEGER NOT NULL,
			sort INTEGER NOT NULL DEFAULT 0,
			status INTEGER NOT NULL DEFAULT 1,
			usage_count INTEGER NOT NULL DEFAULT 0,
			like_count INTEGER NOT NULL DEFAULT 0,
			view_count INTEGER NOT NULL DEFAULT 0,
			deleted_at DATETIME NULL
		)`,
		`CREATE TABLE video_template_type (
			id INTEGER PRIMARY KEY,
			deleted_at DATETIME NULL
		)`,
		`CREATE TABLE video_template_type_display_position (
			id INTEGER PRIMARY KEY,
			template_type_id INTEGER NOT NULL,
			position_key TEXT NOT NULL,
			deleted_at DATETIME NULL
		)`,
		`CREATE TABLE video_template_type_country (
			id INTEGER PRIMARY KEY,
			template_type_id INTEGER NOT NULL,
			country_code TEXT NOT NULL,
			deleted_at DATETIME NULL
		)`,
		`CREATE TABLE video_template_type_app (
			id INTEGER PRIMARY KEY,
			template_type_id INTEGER NOT NULL,
			app_code TEXT NOT NULL,
			deleted_at DATETIME NULL
		)`,
		`CREATE TABLE video_template_type_package (
			id INTEGER PRIMARY KEY,
			template_type_id INTEGER NOT NULL,
			package_code TEXT NOT NULL,
			deleted_at DATETIME NULL
		)`,
		`CREATE TABLE video_template_type_version (
			id INTEGER PRIMARY KEY,
			template_type_id INTEGER NOT NULL,
			version_code TEXT NOT NULL,
			deleted_at DATETIME NULL
		)`,
		`CREATE TABLE video_display_position (
			id INTEGER PRIMARY KEY,
			position_key TEXT NOT NULL,
			deleted_at DATETIME NULL
		)`,
		`INSERT INTO video_model (id, score) VALUES (3, 42)`,
		`INSERT INTO video_template_type (id) VALUES (5)`,
		`INSERT INTO video_template (
			id, template_type, template_type_id, model_id, sort, status,
			usage_count, like_count, view_count
		) VALUES (10, 2, 5, 3, 9, 1, 8, 7, 6)`,
		`INSERT INTO video_display_position (id, position_key) VALUES (1, 'homeRecommend')`,
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatal(err)
		}
	}
	return db
}

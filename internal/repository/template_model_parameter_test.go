package repository

import (
	"context"
	"testing"

	"ai-video/internal/config"
	"ai-video/internal/gen/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestTemplateModelParameterRepoReplaceForTemplate(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:template-model-parameter?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE video_template_model_parameter (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		model_id INTEGER NOT NULL,
		template_id INTEGER NOT NULL,
		param_key TEXT NOT NULL,
		param_type TEXT NOT NULL,
		is_required INTEGER NOT NULL DEFAULT 0,
		default_value TEXT,
		allowed_values TEXT,
		description TEXT,
		sort_order INTEGER NOT NULL DEFAULT 0,
		parameter_type INTEGER NOT NULL DEFAULT 1,
		constraints TEXT,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		deleted_at DATETIME NULL
	)`).Error; err != nil {
		t.Fatal(err)
	}

	previousDB := config.DB
	config.DB = db
	t.Cleanup(func() { config.DB = previousDB })

	repo := NewTemplateModelParameterRepo()
	ctx := context.Background()
	first := []*model.VideoTemplateModelParameter{
		{ModelID: 4, ParamKey: "duration", ParamType: "integer", ParameterType: 1, DefaultValue: "5", AllowedValues: "[3,5]"},
		{ModelID: 4, ParamKey: "prompt", ParamType: "string", ParameterType: 2, Constraints: `{"max_length":100}`},
	}
	if err := repo.ReplaceForTemplate(ctx, 12, first); err != nil {
		t.Fatal(err)
	}
	items, err := repo.ListByTemplate(ctx, 12)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[0].ParamKey != "duration" || items[1].ParamKey != "prompt" {
		t.Fatalf("first replacement = %#v", items)
	}

	second := []*model.VideoTemplateModelParameter{
		{ModelID: 5, ParamKey: "aspect_ratio", ParamType: "string", ParameterType: 1, DefaultValue: `"16:9"`, AllowedValues: `["16:9"]`},
	}
	if err := repo.ReplaceForTemplate(ctx, 12, second); err != nil {
		t.Fatal(err)
	}
	items, err = repo.ListByTemplate(ctx, 12)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ModelID != 5 || items[0].ParamKey != "aspect_ratio" {
		t.Fatalf("second replacement = %#v", items)
	}
	var softDeleted int64
	if err := db.Table(model.TableNameVideoTemplateModelParameter).Where("template_id = ? AND deleted_at IS NOT NULL", 12).Count(&softDeleted).Error; err != nil {
		t.Fatal(err)
	}
	if softDeleted != 2 {
		t.Fatalf("soft-deleted rows = %d, want 2", softDeleted)
	}
}

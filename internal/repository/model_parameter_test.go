package repository

import (
	"context"
	"testing"

	"ai-video/internal/config"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestModelParameterRepoListByModelsIncludesAllParameterTypes(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:model-parameter-list?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE video_model_parameter (
		id INTEGER PRIMARY KEY,
		model_id INTEGER NOT NULL,
		param_key TEXT NOT NULL,
		parameter_type INTEGER NOT NULL,
		sort_order INTEGER NOT NULL,
		deleted_at DATETIME NULL
	)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO video_model_parameter
		(id, model_id, param_key, parameter_type, sort_order, deleted_at) VALUES
		(1, 10, 'prompt', 2, 1, NULL),
		(2, 10, 'duration', 1, 2, NULL),
		(3, 10, 'aspect_ratio', 1, 1, NULL),
		(4, 20, 'images', 2, 1, NULL),
		(5, 10, 'deleted', 1, 0, CURRENT_TIMESTAMP)`).Error; err != nil {
		t.Fatal(err)
	}

	previousDB := config.DB
	config.DB = db
	t.Cleanup(func() { config.DB = previousDB })

	items, err := NewModelParameterRepo().ListByModels(context.Background(), []int64{20, 10})
	if err != nil {
		t.Fatal(err)
	}
	wantKeys := []string{"aspect_ratio", "duration", "prompt", "images"}
	if len(items) != len(wantKeys) {
		t.Fatalf("ListByModels() returned %d items: %#v", len(items), items)
	}
	for i, wantKey := range wantKeys {
		if items[i].ParamKey != wantKey {
			t.Fatalf("ListByModels()[%d].ParamKey = %q, want %q", i, items[i].ParamKey, wantKey)
		}
	}
	if items[0].ParameterType != 1 || items[2].ParameterType != 2 {
		t.Fatalf("ListByModels() omitted or reordered parameter types: %#v", items)
	}
}

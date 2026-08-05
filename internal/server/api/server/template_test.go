package service

import (
	"testing"

	"ai-video/internal/gen/model"
)

func TestMapClientTemplateReturnsModelScore(t *testing.T) {
	item := model.VideoTemplate{
		ID: 10,
		AIModel: model.VideoModel{
			ID:    3,
			Score: 42,
		},
	}

	got := mapClientTemplate(&item)
	if got.ModelScore != 42 {
		t.Fatalf("model_score = %d, want 42", got.ModelScore)
	}
}

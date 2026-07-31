package generation

import (
	"testing"

	"ai-video/internal/gen/model"
)

func TestViewOfIncludesTaskCover(t *testing.T) {
	item := &model.VideoUserGenerationTask{
		ID: 1, TaskCode: "task-1", LocalUrls: `[]`,
		CoverImageURL: "https://cdn.example.com/generated/1/task-1-cover.jpg",
	}
	view := ViewOf(item)
	if view.CoverImageURL != item.CoverImageURL {
		t.Fatalf("cover image URL = %q, want %q", view.CoverImageURL, item.CoverImageURL)
	}
}

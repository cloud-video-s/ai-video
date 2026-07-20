package service

import (
	"testing"

	"ai-video/internal/model"
)

func TestBuildClientTemplateGroupsSortsAndDropsEmptyCategories(t *testing.T) {
	types := []model.VideoTemplateType{
		{ID: 1, CategoryName: "low", Sort: 1},
		{ID: 3, CategoryName: "empty", Sort: 99},
		{ID: 2, CategoryName: "high", Sort: 10},
	}
	rows := []model.VideoTemplate{
		{ID: 11, VideoTemplateTypeID: 1, Name: "views", Sort: 5, UsageCount: 1, FavoriteCount: 1, ViewCount: 9},
		{ID: 21, VideoTemplateTypeID: 2, Name: "primary-sort", Sort: 9},
		{ID: 12, VideoTemplateTypeID: 1, Name: "favorites", Sort: 5, UsageCount: 1, FavoriteCount: 2},
		{ID: 13, VideoTemplateTypeID: 1, Name: "usage", Sort: 5, UsageCount: 2},
	}

	got := buildClientTemplateGroups(types, rows)
	if len(got) != 2 {
		t.Fatalf("category count = %d, want 2", len(got))
	}
	if got[0].ID != 2 || got[1].ID != 1 {
		t.Fatalf("category order = [%d %d], want [2 1]", got[0].ID, got[1].ID)
	}
	lowTemplates := got[1].Templates
	if len(lowTemplates) != 3 {
		t.Fatalf("low category template count = %d, want 3", len(lowTemplates))
	}
	if lowTemplates[0].Name != "usage" || lowTemplates[1].Name != "favorites" || lowTemplates[2].Name != "views" {
		t.Fatalf("template order = [%s %s %s]", lowTemplates[0].Name, lowTemplates[1].Name, lowTemplates[2].Name)
	}
}

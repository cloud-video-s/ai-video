package service

import (
	"testing"

	"ai-video/internal/model"
)

func TestMapClientBannerIncludesNavigationTarget(t *testing.T) {
	templateID := uint64(42)
	item := model.VideoBanner{
		ID: 7, Name: "Featured", Status: 1, JumpType: model.BannerJumpTypeTemplate,
		CoverImage: "https://example.com/banner.jpg", TemplateID: &templateID,
		DisplayPositions: []model.VideoDisplayPosition{
			{ID: 1, PositionKey: "home"}, {ID: 2, PositionKey: "profile"},
		},
		Template: &model.VideoTemplate{
			ID: templateID, Name: "Target", TemplateType: model.VideoTemplateKindAction,
			CoverImage: "https://example.com/template.jpg", Status: 1,
		},
	}

	got := mapClientBanner(&item)
	if got.Name != item.Name || got.Route != "/templates/42" {
		t.Fatalf("mapped banner = %#v", got)
	}
	if got.PositionKey != "home" {
		t.Fatalf("position_key = %q, want home", got.PositionKey)
	}
	if got.JumpType != 2 {
		t.Fatalf("jump_type = %d, want 2", got.JumpType)
	}
	if got.TargetTemplate == nil || got.TargetTemplate.ID != templateID {
		t.Fatalf("mapped target template = %#v", got.TargetTemplate)
	}
}

func TestClientBannerRoutePrefersConfiguredRoute(t *testing.T) {
	item := model.VideoBanner{JumpType: model.BannerJumpTypeTextToVideo, JumpURL: "/campaign/summer"}
	if got := clientBannerRoute(&item); got != "/campaign/summer" {
		t.Fatalf("route = %q, want configured route", got)
	}
}

package service

import (
	"context"
	"testing"

	"ai-video/internal/model"
)

func TestValidBannerLink(t *testing.T) {
	tests := []struct {
		value string
		want  bool
	}{
		{value: "/feature/text-to-image", want: true},
		{value: "https://example.com/activity", want: true},
		{value: "myapp://feature/text-to-video", want: true},
		{value: "//example.com/activity", want: false},
		{value: "feature/text-to-image", want: false},
		{value: "javascript:alert(1)", want: false},
		{value: "", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			if got := validBannerLink(tt.value); got != tt.want {
				t.Fatalf("validBannerLink(%q) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestValidateBannerJumpNormalizesTargets(t *testing.T) {
	service := &BannerService{}
	templateID := uint64(9)

	link := &BannerPayload{
		JumpType: model.BannerJumpTypeLink, JumpURL: " https://example.com/activity ", TemplateID: &templateID,
	}
	if err := service.validateJump(context.Background(), link); err != nil {
		t.Fatal(err)
	}
	if link.JumpURL != "https://example.com/activity" || link.TemplateID != nil {
		t.Fatalf("unexpected normalized link payload: %+v", link)
	}

	feature := &BannerPayload{
		JumpType: model.BannerJumpTypeTextToVideo, JumpURL: "https://example.com", TemplateID: &templateID,
	}
	if err := service.validateJump(context.Background(), feature); err != nil {
		t.Fatal(err)
	}
	if feature.JumpURL != "" || feature.TemplateID != nil {
		t.Fatalf("feature jump retained an irrelevant target: %+v", feature)
	}

	unknown := &BannerPayload{JumpType: 9}
	if err := service.validateJump(context.Background(), unknown); err == nil {
		t.Fatal("unsupported jump type was accepted")
	}
}

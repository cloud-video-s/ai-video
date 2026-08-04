package service

import (
	"testing"

	"ai-video/internal/gen/model"
)

func TestModelIconRoundTrip(t *testing.T) {
	const icon = "/uploads/images/model-icon.png"
	payload := &ModelPayload{Icon: icon}
	item := &model.VideoModel{}

	applyModelPayload(item, payload)
	if item.Icon != icon {
		t.Fatalf("applyModelPayload() icon = %q, want %q", item.Icon, icon)
	}

	view := modelView(item, false)
	if view.Icon != icon {
		t.Fatalf("modelView() icon = %q, want %q", view.Icon, icon)
	}
}

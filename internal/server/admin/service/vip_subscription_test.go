package service

import (
	"encoding/json"
	"testing"

	"ai-video/internal/gen/model"
)

func TestVIPSubscriptionViewExposesManagedPackage(t *testing.T) {
	item := &model.VideoVipSubscription{
		ID: 7,
		Packages: []model.VideoPackage{{
			ID: 3, PackageName: "Android", PackageCode: "com.example.app", AppCode: "video", SystemType: 2,
		}},
	}
	data, err := json.Marshal(vipSubscriptionView(item))
	if err != nil {
		t.Fatalf("marshal view: %v", err)
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("unmarshal view: %v", err)
	}
	if payload["package_id"] != float64(3) {
		t.Fatalf("package_id=%v, want 3", payload["package_id"])
	}
	packagePayload, ok := payload["package"].(map[string]interface{})
	if !ok || packagePayload["package_code"] != "com.example.app" {
		t.Fatalf("package=%v, want com.example.app", payload["package"])
	}
}

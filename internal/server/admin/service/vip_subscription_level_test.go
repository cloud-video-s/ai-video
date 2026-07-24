package service

import (
	"testing"

	"ai-video/internal/gen/model"
)

func TestValidateVIPSubscriptionLevelPayload(t *testing.T) {
	if err := validateVIPSubscriptionLevelPayload(&VIPSubscriptionLevelPayload{Level: "黄金会员", Status: 1}); err != nil {
		t.Fatalf("valid payload rejected: %v", err)
	}
	if err := validateVIPSubscriptionLevelPayload(&VIPSubscriptionLevelPayload{Level: "   ", Status: 1}); err == nil {
		t.Fatal("empty level should be rejected")
	}
}

func TestApplyVIPSubscriptionLevelPayload(t *testing.T) {
	item := &model.VideoVipSubscriptionLevel{}
	applyVIPSubscriptionLevelPayload(item, &VIPSubscriptionLevelPayload{
		Level: "  黄金会员  ", Description: "  全部权益  ", Status: 1, Sort: 20,
	})
	if item.Level != "黄金会员" || item.Description != "全部权益" || item.Status != 1 || item.Sort != 20 {
		t.Fatalf("unexpected normalized level: %+v", item)
	}
}

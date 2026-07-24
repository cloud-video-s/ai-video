package service

import (
	"encoding/json"
	"reflect"
	"testing"

	"ai-video/internal/gen/model"
)

func TestVIPSubscriptionJSONUsesLatestAssociationFields(t *testing.T) {
	item := &model.VideoVipSubscription{
		ID:                7,
		SubscriptionLevel: model.VideoVipSubscriptionLevel{ID: 2, Level: "黄金会员"},
		Apps:              []*model.VideoApp{{ID: 1, AppCode: "video"}},
		Packages:          []*model.VideoPackage{{ID: 3, PackageName: "Android", PackageCode: "com.example.app", AppCode: "video"}},
		PackageVersion:    []*model.VideoPackageVersion{{ID: 4, VersionCode: "1.2.0", PackageCode: "com.example.app"}},
		Country:           []*model.VideoCountry{{ID: 5, Code: "US"}},
		Channels:          []*model.VideoChannel{{ChannelID: 6, ChannelCode: "google_ads"}},
	}
	data, err := json.Marshal(item)
	if err != nil {
		t.Fatalf("marshal model: %v", err)
	}
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("unmarshal model: %v", err)
	}
	for _, key := range []string{"subscription_level", "apps", "packages", "package_version", "country", "channels"} {
		if _, ok := payload[key]; !ok {
			t.Fatalf("latest association field %q is missing: %s", key, data)
		}
	}
	for _, legacyKey := range []string{"versions", "countries", "placement"} {
		if _, ok := payload[legacyKey]; ok {
			t.Fatalf("legacy association field %q must not be returned", legacyKey)
		}
	}
}

func TestVIPSubscriptionPayloadFromModelCopiesAllTargets(t *testing.T) {
	item := &model.VideoVipSubscription{
		VipType: 3, SukCode: "vip.monthly", Name: "月度会员", LevelID: 2,
		Currency: "USD", SubscriptionPeriod: 2,
		Apps:           []*model.VideoApp{{AppCode: "video"}},
		Packages:       []*model.VideoPackage{{PackageCode: "com.example.app"}},
		PackageVersion: []*model.VideoPackageVersion{{VersionCode: "1.2.0"}},
		Country:        []*model.VideoCountry{{Code: "US"}},
		Channels:       []*model.VideoChannel{{ChannelCode: "google_ads"}},
	}
	payload := vipSubscriptionPayloadFromModel(item)
	if payload.SukCode != item.SukCode || payload.LevelID != item.LevelID || payload.VipType != item.VipType {
		t.Fatalf("base fields not copied: %+v", payload)
	}
	checks := []struct {
		name string
		got  []string
		want []string
	}{
		{name: "apps", got: payload.AppCodes, want: []string{"video"}},
		{name: "packages", got: payload.PackageCodes, want: []string{"com.example.app"}},
		{name: "versions", got: payload.VersionCodes, want: []string{"1.2.0"}},
		{name: "countries", got: payload.CountryCodes, want: []string{"US"}},
		{name: "channels", got: payload.ChannelCodes, want: []string{"google_ads"}},
	}
	for _, check := range checks {
		if !reflect.DeepEqual(check.got, check.want) {
			t.Fatalf("%s = %v, want %v", check.name, check.got, check.want)
		}
	}
}

func TestNormalizeVIPTargetCodes(t *testing.T) {
	values, err := normalizeVIPTargetCodes([]string{" us ", "US", " cn "}, "国家", true)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if !reflect.DeepEqual(values, []string{"US", "CN"}) {
		t.Fatalf("values = %v", values)
	}
}

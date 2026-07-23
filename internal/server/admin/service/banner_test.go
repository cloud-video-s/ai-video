package service

import (
	"reflect"
	"testing"
)

func TestNormalizeBannerAppTargetsMergesDuplicatePackages(t *testing.T) {
	got, err := normalizeBannerAppTargets([]BannerAppTargetPayload{
		{AppCode: " ai-video ", PackageCode: " com.example.video ", VersionCodes: []string{"2.0.0", "1.0.0", "1.0.0"}},
		{AppCode: "ai-video", PackageCode: "com.example.video", VersionCodes: []string{"3.0.0"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].AppCode != "ai-video" || got[0].PackageCode != "com.example.video" ||
		!reflect.DeepEqual(got[0].VersionCodes, []string{"1.0.0", "2.0.0", "3.0.0"}) {
		t.Fatalf("normalized targets = %#v", got)
	}
}

func TestNormalizeBannerAppTargetsAllVersionsOverridesExplicitVersions(t *testing.T) {
	got, err := normalizeBannerAppTargets([]BannerAppTargetPayload{
		{AppCode: "ai-video", PackageCode: "com.example.video", VersionCodes: []string{"1.0.0", "2.0.0"}},
		{AppCode: "ai-video", PackageCode: "com.example.video", VersionCodes: nil},
		{AppCode: "ai-video", PackageCode: "com.example.video", VersionCodes: []string{"3.0.0"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || len(got[0].VersionCodes) != 0 {
		t.Fatalf("all-version target = %#v", got)
	}
}

func TestNormalizeBannerEmptySelectionsAreValidAllTargets(t *testing.T) {
	positions, err := normalizeBannerPositionKeys(nil)
	if err != nil {
		t.Fatal(err)
	}
	apps, err := normalizeBannerAppTargets(nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(positions) != 0 || len(apps) != 0 {
		t.Fatalf("empty selections must remain empty: positions=%v apps=%v", positions, apps)
	}
}

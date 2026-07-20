package upload

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestPolicyForExtensionsDerivesMIMETypes(t *testing.T) {
	policy, err := PolicyForExtensions(MediaImage, 8<<20, []string{"jpg", ".jpeg", ".png", ".jpg"})
	if err != nil {
		t.Fatal(err)
	}
	if policy.MaxFileSize != 8<<20 {
		t.Fatalf("MaxFileSize = %d", policy.MaxFileSize)
	}
	if len(policy.AllowedExts) != 3 {
		t.Fatalf("AllowedExts = %#v", policy.AllowedExts)
	}
	if len(policy.AllowedMIMETypes) != 2 {
		t.Fatalf("AllowedMIMETypes = %#v", policy.AllowedMIMETypes)
	}
	if _, err := PolicyForExtensions(MediaImage, 8<<20, []string{".mp4"}); err == nil {
		t.Fatal("video extension was accepted by image policy")
	}
	if _, err := PolicyForExtensions(MediaVideo, 8<<20, nil); err == nil {
		t.Fatal("empty extension policy was accepted")
	}
}

func TestManagerUsesPolicyResolverForNewUploads(t *testing.T) {
	config := Config{
		RootDir: t.TempDir(), ChunkSize: 8, MaxBatchFiles: 3, SessionTTL: time.Minute,
		Image: Policy{MaxFileSize: 100, AllowedExts: []string{".png"}, AllowedMIMETypes: []string{"image/png"}},
		Video: Policy{MaxFileSize: 100, AllowedExts: []string{".mp4"}, AllowedMIMETypes: []string{"video/mp4"}},
	}
	current := Policy{MaxFileSize: 10, AllowedExts: []string{".png"}, AllowedMIMETypes: []string{"image/png"}}
	config.PolicyResolver = func(kind MediaKind) (Policy, error) {
		if kind == MediaImage {
			return current, nil
		}
		return config.Video, nil
	}
	manager, err := NewManager(config)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.InitiateImages(context.Background(), []FileSpec{{FileName: "first.png", Size: 10, ContentType: "image/png"}}); err != nil {
		t.Fatal(err)
	}
	current = Policy{MaxFileSize: 5, AllowedExts: []string{".jpg"}, AllowedMIMETypes: []string{"image/jpeg"}}
	if _, err := manager.InitiateImages(context.Background(), []FileSpec{{FileName: "second.png", Size: 5, ContentType: "image/png"}}); err == nil {
		t.Fatal("removed extension was still accepted")
	}
	if _, err := manager.InitiateImages(context.Background(), []FileSpec{{FileName: "second.jpg", Size: 6, ContentType: "image/jpeg"}}); err == nil {
		t.Fatal("updated file size limit was not applied")
	}
}

func TestValidateSessionPolicyUsesCurrentLimits(t *testing.T) {
	policy, err := normalizePolicy(Policy{
		MaxFileSize: 5, AllowedExts: []string{".jpg"}, AllowedMIMETypes: []string{"image/jpeg"},
	})
	if err != nil {
		t.Fatal(err)
	}
	session := &Session{OriginalName: "old.png", Extension: ".png", TotalSize: 6}
	if err := validateSessionPolicy(session, policy); !errors.Is(err, ErrFileTooLarge) {
		t.Fatalf("size validation error = %v", err)
	}
	session.TotalSize = 5
	if err := validateSessionPolicy(session, policy); !errors.Is(err, ErrUnsupportedType) {
		t.Fatalf("extension validation error = %v", err)
	}
}

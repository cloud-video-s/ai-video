package uploadruntime

import (
	"testing"

	"ai-video/internal/config"
	"ai-video/internal/pkg/upload"
)

func TestTemplateMediaURLConversion(t *testing.T) {
	previousUpload := config.Cfg.Upload
	previousDB := config.DB
	config.DB = nil
	config.Cfg.Upload = config.UploadConfig{
		StorageProvider: upload.StorageAliyunOSS,
		OSSBaseURL:      "https://origin-bucket.example.com/",
		ProxyBaseURL:    "https://test-cdn.zdrawai.com/",
	}
	t.Cleanup(func() {
		config.Cfg.Upload = previousUpload
		config.DB = previousDB
	})

	half := "/uploads/videos/template.mp4"
	full := "https://test-cdn.zdrawai.com/uploads/videos/template.mp4"
	if got := PublicURL(half); got != full {
		t.Fatalf("PublicURL(%q) = %q, want %q", half, got, full)
	}
	halfWithoutLeadingSlash := "uploads/videos/template.mp4"
	if got := PublicURL(halfWithoutLeadingSlash); got != full {
		t.Fatalf("PublicURL(%q) = %q, want %q", halfWithoutLeadingSlash, got, full)
	}
	if got := PersistedURL(full); got != half {
		t.Fatalf("PersistedURL(%q) = %q, want %q", full, got, half)
	}
	origin := "https://origin-bucket.example.com/uploads/videos/template.mp4"
	if got := PublicURL(origin); got != full {
		t.Fatalf("PublicURL(%q) = %q, want proxy URL %q", origin, got, full)
	}
	if got := PersistedURL(origin); got != half {
		t.Fatalf("PersistedURL(%q) = %q, want %q", origin, got, half)
	}
	external := "https://media.example.com/template.mp4"
	if got := PublicURL(external); got != external {
		t.Fatalf("PublicURL changed external URL to %q", got)
	}
	if got := PersistedURL(external); got != external {
		t.Fatalf("PersistedURL changed external URL to %q", got)
	}
}

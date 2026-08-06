package service

import (
	"testing"

	"ai-video/internal/config"
	"ai-video/internal/gen/model"
	"ai-video/internal/pkg/upload"
)

func TestTemplateMediaURLsUseProxyInResponsesAndHalfURLsInPayloads(t *testing.T) {
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

	item := model.VideoTemplate{
		CoverImageURL: "/uploads/images/cover.png",
		OriginalURL:   "/uploads/videos/original.mp4",
		ThumbnailURL:  "/uploads/videos/thumbnail.mp4",
	}
	expandTemplateMediaURLs(&item)
	if item.OriginalURL != "https://test-cdn.zdrawai.com/uploads/videos/original.mp4" ||
		item.ThumbnailURL != "https://test-cdn.zdrawai.com/uploads/videos/thumbnail.mp4" {
		t.Fatalf("expanded template URLs = original %q, thumbnail %q", item.OriginalURL, item.ThumbnailURL)
	}

	payload := &TemplatePayload{
		CoverImageURL: item.CoverImageURL,
		OriginalURL:   item.OriginalURL,
		ThumbnailURL:  item.ThumbnailURL,
	}
	normalizeTemplatePayload(payload)
	if payload.CoverImageURL != "/uploads/images/cover.png" ||
		payload.OriginalURL != "/uploads/videos/original.mp4" ||
		payload.ThumbnailURL != "/uploads/videos/thumbnail.mp4" {
		t.Fatalf("normalized template URLs = %+v", payload)
	}
}

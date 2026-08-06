package service

import (
	"testing"

	"ai-video/internal/config"
	"ai-video/internal/domain"
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

func TestValidateTemplatePayloadAllowsImageOrVideoThumbnail(t *testing.T) {
	tests := []struct {
		name         string
		templateType int64
		originalURL  string
		thumbnailURL string
	}{
		{name: "image template with image thumbnail", templateType: domain.VideoTemplateKindImage, originalURL: "/uploads/images/original.png", thumbnailURL: "/uploads/images/thumbnail.webp"},
		{name: "image template with video thumbnail", templateType: domain.VideoTemplateKindImage, originalURL: "/uploads/images/original.png", thumbnailURL: "/uploads/videos/thumbnail.mp4"},
		{name: "video template with image thumbnail", templateType: domain.VideoTemplateKindVideo, originalURL: "/uploads/videos/original.mp4", thumbnailURL: "/uploads/images/thumbnail.jpg"},
		{name: "video template with video thumbnail", templateType: domain.VideoTemplateKindVideo, originalURL: "/uploads/videos/original.mp4", thumbnailURL: "/uploads/videos/thumbnail.webm"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := &TemplatePayload{
				TemplateTypeID: 1,
				ModelID:        1,
				Name:           "template",
				TemplateType:   tt.templateType,
				CoverImageURL:  "/uploads/images/cover.png",
				OriginalURL:    tt.originalURL,
				ThumbnailURL:   tt.thumbnailURL,
				Prompt:         "prompt",
			}
			if err := validateTemplatePayload(payload); err != nil {
				t.Fatalf("validateTemplatePayload() error = %v", err)
			}
		})
	}
}

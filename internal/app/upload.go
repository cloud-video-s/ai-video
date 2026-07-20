package app

import (
	"time"

	"ai-video/internal/pkg/upload"
)

func UploadManagerConfig() upload.Config {
	cfg := Cfg.Upload
	if cfg.RootDir == "" {
		return upload.DefaultConfig()
	}
	return upload.Config{
		RootDir: cfg.RootDir, ChunkSize: cfg.ChunkSize, MaxBatchFiles: cfg.MaxBatchFiles,
		SessionTTL: time.Duration(cfg.SessionTTLSeconds) * time.Second,
		Image: upload.Policy{
			MaxFileSize: cfg.ImageMaxFileSize, AllowedExts: cfg.ImageExtensions,
			AllowedMIMETypes: cfg.ImageMIMETypes,
		},
		Video: upload.Policy{
			MaxFileSize: cfg.VideoMaxFileSize, AllowedExts: cfg.VideoExtensions,
			AllowedMIMETypes: cfg.VideoMIMETypes,
		},
	}
}

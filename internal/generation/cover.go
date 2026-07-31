package generation

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"ai-video/internal/config"
	"ai-video/internal/gen/model"
	"ai-video/internal/pkg/upload"
)

func generateAndStoreTaskCover(
	ctx context.Context,
	storage upload.Storage,
	task *model.VideoUserGenerationTask,
	kind upload.MediaKind,
	source string,
) (string, error) {
	options := upload.DefaultMediaPreviewOptions()
	options.RemoteHTTPClient = secureDownloadClient()
	options.MaxRemoteBytes = config.Cfg.Upload.ImageMaxFileSize
	if options.MaxRemoteBytes <= 0 {
		options.MaxRemoteBytes = 20 << 20
	}
	return generateAndStoreTaskCoverWithOptions(ctx, storage, task, kind, source, options)
}

func generateAndStoreTaskCoverWithOptions(
	ctx context.Context,
	storage upload.Storage,
	task *model.VideoUserGenerationTask,
	kind upload.MediaKind,
	source string,
	options upload.MediaPreviewOptions,
) (string, error) {
	if storage == nil {
		return "", errors.New("generation task cover storage is not configured")
	}
	if task == nil || task.ID == 0 || task.UserID == 0 {
		return "", errors.New("generation task cover requires a persisted task")
	}
	if strings.TrimSpace(source) == "" {
		return "", errors.New("generation task cover source is empty")
	}

	previewPath, err := newGeneratedTemporaryFile()
	if err != nil {
		return "", err
	}
	defer os.Remove(previewPath)
	if err := upload.GenerateMediaPreview(ctx, kind, source, previewPath, options); err != nil {
		return "", fmt.Errorf("generate task cover preview: %w", err)
	}
	filename := fmt.Sprintf("task-%d-cover.jpg", task.ID)
	coverURL, err := storeGeneratedFile(
		ctx,
		storage,
		generatedObjectKey(task.UserID, filename),
		previewPath,
		"image/jpeg",
	)
	if err != nil {
		return "", fmt.Errorf("store task cover preview: %w", err)
	}
	return coverURL, nil
}

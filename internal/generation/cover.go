package generation

import (
	"context"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"os"
	"strings"

	"ai-video/internal/config"
	"ai-video/internal/gen/model"
	"ai-video/internal/pkg/upload"
)

const (
	defaultVideoCoverWidth  = 1280
	defaultVideoCoverHeight = 720
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

func generateAndStoreTaskCoverWithOptions(ctx context.Context, storage upload.Storage, task *model.VideoUserGenerationTask, kind upload.MediaKind, source string, options upload.MediaPreviewOptions) (string, error) {
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
	if err = upload.GenerateMediaPreview(ctx, kind, source, previewPath, options); err != nil {
		return "", fmt.Errorf("generate task cover preview: %w", err)
	}
	filename := fmt.Sprintf("task-%s-cover.jpg", task.TaskCode)
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

func generateImageTaskCoverOrOriginal(
	ctx context.Context,
	storage upload.Storage,
	task *model.VideoUserGenerationTask,
	source string,
	originalURL string,
) (string, error) {
	return generateImageTaskCoverOrOriginalWithRetry(ctx, storage, task, source, originalURL, 0)
}

func generateImageTaskCoverOrOriginalWithRetry(
	ctx context.Context,
	storage upload.Storage,
	task *model.VideoUserGenerationTask,
	source string,
	originalURL string,
	retryCount int,
) (string, error) {
	coverURL, err := retryGeneratedTaskCover(ctx, retryCount, func() (string, error) {
		return generateAndStoreTaskCover(ctx, storage, task, upload.MediaImage, source)
	})
	if err != nil {
		return originalURL, err
	}
	return coverURL, nil
}

func generateOrStoreDefaultVideoTaskCover(
	ctx context.Context,
	storage upload.Storage,
	task *model.VideoUserGenerationTask,
	sources ...string,
) (string, error) {
	return generateOrStoreDefaultVideoTaskCoverWithRetry(ctx, storage, task, 0, sources...)
}

func generateOrStoreDefaultVideoTaskCoverWithRetry(
	ctx context.Context,
	storage upload.Storage,
	task *model.VideoUserGenerationTask,
	retryCount int,
	sources ...string,
) (string, error) {
	return generateOrStoreDefaultVideoTaskCoverWithGenerator(
		ctx, storage, task, retryCount,
		func(source string) (string, error) {
			return generateAndStoreTaskCover(ctx, storage, task, upload.MediaVideo, source)
		},
		sources...,
	)
}

func generateOrStoreDefaultVideoTaskCoverWithGenerator(
	ctx context.Context,
	storage upload.Storage,
	task *model.VideoUserGenerationTask,
	retryCount int,
	generate func(source string) (string, error),
	sources ...string,
) (string, error) {
	if task == nil || task.ID == 0 || task.UserID == 0 {
		return "", errors.New("video task cover requires a persisted task")
	}
	if generate == nil {
		return "", errors.New("video task cover generator is not configured")
	}
	seen := make(map[string]struct{}, len(sources))
	var coverErr error
	for _, source := range sources {
		source = strings.TrimSpace(source)
		if source == "" {
			continue
		}
		if _, exists := seen[source]; exists {
			continue
		}
		seen[source] = struct{}{}

		coverURL, err := retryGeneratedTaskCover(ctx, retryCount, func() (string, error) {
			return generate(source)
		})
		if err == nil {
			return coverURL, nil
		}
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		coverErr = err
	}
	if coverErr != nil && config.Log != nil {
		config.Logger(ctx).Warnw("video task cover generation failed; using default cover",
			"task_id", task.ID,
			"task_code", task.TaskCode,
			"error", coverErr,
		)
	}
	return storeDefaultVideoTaskCover(ctx, storage, task)
}

func retryGeneratedTaskCover(
	ctx context.Context,
	retryCount int,
	generate func() (string, error),
) (string, error) {
	var coverURL string
	err := retryDownload(ctx, retryCount, func() error {
		var err error
		coverURL, err = generate()
		return err
	})
	return coverURL, err
}

func storeDefaultVideoTaskCover(
	ctx context.Context,
	storage upload.Storage,
	task *model.VideoUserGenerationTask,
) (string, error) {
	if task == nil || task.ID == 0 || task.UserID == 0 {
		return "", errors.New("default video task cover requires a persisted task")
	}
	temporary, err := newGeneratedTemporaryFile()
	if err != nil {
		return "", err
	}
	defer os.Remove(temporary)
	if err = writeDefaultVideoCover(temporary); err != nil {
		return "", fmt.Errorf("write default video task cover: %w", err)
	}
	filename := fmt.Sprintf("task-%s-cover.jpg", task.TaskCode)
	coverURL, err := storeGeneratedFile(
		ctx,
		storage,
		generatedObjectKey(task.UserID, filename),
		temporary,
		"image/jpeg",
	)
	if err != nil {
		return "", fmt.Errorf("store default video task cover: %w", err)
	}
	return coverURL, nil
}

func writeDefaultVideoCover(path string) error {
	cover := image.NewRGBA(image.Rect(0, 0, defaultVideoCoverWidth, defaultVideoCoverHeight))
	for y := 0; y < defaultVideoCoverHeight; y++ {
		red := uint8(18 + 12*y/defaultVideoCoverHeight)
		green := uint8(24 + 14*y/defaultVideoCoverHeight)
		blue := uint8(39 + 21*y/defaultVideoCoverHeight)
		row := cover.Pix[y*cover.Stride : y*cover.Stride+defaultVideoCoverWidth*4]
		for x := 0; x < defaultVideoCoverWidth; x++ {
			offset := x * 4
			row[offset] = red
			row[offset+1] = green
			row[offset+2] = blue
			row[offset+3] = 0xff
		}
	}

	centerX, centerY := defaultVideoCoverWidth/2, defaultVideoCoverHeight/2
	const buttonRadius = 92
	for y := centerY - buttonRadius; y <= centerY+buttonRadius; y++ {
		for x := centerX - buttonRadius; x <= centerX+buttonRadius; x++ {
			dx, dy := x-centerX, y-centerY
			if dx*dx+dy*dy > buttonRadius*buttonRadius {
				continue
			}
			offset := cover.PixOffset(x, y)
			cover.Pix[offset] = 57
			cover.Pix[offset+1] = 66
			cover.Pix[offset+2] = 88
			cover.Pix[offset+3] = 0xff
		}
	}
	const triangleHalfHeight = 52
	for dy := -triangleHalfHeight; dy <= triangleHalfHeight; dy++ {
		endX := centerX + 62 - absInt(dy)*84/triangleHalfHeight
		for x := centerX - 22; x <= endX; x++ {
			offset := cover.PixOffset(x, centerY+dy)
			cover.Pix[offset] = 0xff
			cover.Pix[offset+1] = 0xff
			cover.Pix[offset+2] = 0xff
			cover.Pix[offset+3] = 0xff
		}
	}

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	encodeErr := jpeg.Encode(file, cover, &jpeg.Options{Quality: 85})
	closeErr := file.Close()
	if encodeErr != nil {
		return encodeErr
	}
	return closeErr
}

func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

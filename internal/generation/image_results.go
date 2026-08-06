package generation

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"ai-video/internal/config"
	"ai-video/internal/gen/model"
	"ai-video/internal/pkg/ucloud"
	"ai-video/internal/pkg/upload"
	"ai-video/internal/pkg/uploadruntime"
)

func (m *Manager) finishImageTask(ctx context.Context, task *model.VideoUserGenerationTask, remoteURLs, base64Images []string) error {
	storage, err := uploadruntime.Storage()
	if err != nil {
		return err
	}
	storage = recordGeneratedUploads(storage, m.uploadRepo, task, upload.MediaImage)
	storedURLs, err := downloadImages(ctx, storage, task, remoteURLs)
	if err != nil {
		return err
	}
	encodedURLs, err := saveBase64Images(ctx, storage, task, base64Images, len(storedURLs))
	if err != nil {
		return err
	}
	storedURLs = append(storedURLs, encodedURLs...)
	if len(storedURLs) == 0 {
		return errors.New("image generation completed without an output")
	}
	coverSource, cleanupCoverSource, err := firstImageCoverSource(remoteURLs, base64Images)
	if err != nil {
		return err
	}
	defer cleanupCoverSource()
	coverURL, err := generateAndStoreTaskCover(ctx, storage, task, upload.MediaImage, coverSource)
	if err != nil {
		return fmt.Errorf("generate image task cover: %w", err)
	}
	rawURLs, _ := json.Marshal(storedURLs)
	now := time.Now()
	task.LocalUrls = string(rawURLs)
	task.CoverImageURL = coverURL
	task.Status = TaskStatusSuccess
	task.Progress = 100
	task.ErrorMessage = ""
	task.FinishedAt = now
	return m.completeTask(ctx, task,
		"LocalUrls", "CoverImageURL", "Status", "Progress", "ErrorMessage", "FinishedAt",
	)
}

func downloadImages(ctx context.Context, storage upload.Storage, task *model.VideoUserGenerationTask, remoteURLs []string) ([]string, error) {
	maxSize := config.Cfg.Upload.ImageMaxFileSize
	if maxSize <= 0 {
		maxSize = 20 << 20
	}
	client := secureDownloadClient()
	result := make([]string, 0, len(remoteURLs))
	for index, remoteURL := range remoteURLs {
		extension := imageURLSuffix(remoteURL)
		filename := fmt.Sprintf("task-%s-%d%s", task.TaskCode, index+1, extension)
		storedURL, err := downloadAndStoreGeneratedFile(
			ctx, storage, client, remoteURL, generatedObjectKey(task.UserID, filename), imageContentType(extension), maxSize,
		)
		if err != nil {
			return nil, err
		}
		result = append(result, storedURL)
	}
	return result, nil
}

func saveBase64Images(
	ctx context.Context,
	storage upload.Storage,
	task *model.VideoUserGenerationTask,
	values []string,
	startIndex int,
) ([]string, error) {
	maxSize := config.Cfg.Upload.ImageMaxFileSize
	if maxSize <= 0 {
		maxSize = 20 << 20
	}
	result := make([]string, 0, len(values))
	for index, encoded := range values {
		raw, extension, contentType, err := decodeGeneratedBase64Image(encoded, index, maxSize)
		if err != nil {
			return nil, err
		}
		filename := fmt.Sprintf("task-%s-%d%s", task.TaskCode, startIndex+index+1, extension)
		storedURL, err := storeGeneratedBytes(
			ctx, storage, generatedObjectKey(task.UserID, filename), contentType, raw,
		)
		if err != nil {
			return nil, err
		}
		result = append(result, storedURL)
	}
	return result, nil
}

func firstImageCoverSource(remoteURLs, base64Images []string) (string, func(), error) {
	if len(remoteURLs) > 0 && strings.TrimSpace(remoteURLs[0]) != "" {
		return strings.TrimSpace(remoteURLs[0]), func() {}, nil
	}
	if len(base64Images) == 0 {
		return "", func() {}, errors.New("image generation completed without a first image for its cover")
	}
	maxSize := config.Cfg.Upload.ImageMaxFileSize
	if maxSize <= 0 {
		maxSize = 20 << 20
	}
	raw, _, _, err := decodeGeneratedBase64Image(base64Images[0], 0, maxSize)
	if err != nil {
		return "", func() {}, err
	}
	temporary, err := newGeneratedTemporaryFile()
	if err != nil {
		return "", func() {}, err
	}
	cleanup := func() { _ = os.Remove(temporary) }
	if err := os.WriteFile(temporary, raw, 0o600); err != nil {
		cleanup()
		return "", func() {}, err
	}
	return temporary, cleanup, nil
}

func decodeGeneratedBase64Image(encoded string, index int, maxSize int64) ([]byte, string, string, error) {
	if comma := strings.IndexByte(encoded, ','); strings.HasPrefix(encoded, "data:") && comma >= 0 {
		encoded = encoded[comma+1:]
	}
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, "", "", fmt.Errorf("decode generated image %d: %w", index, err)
	}
	if len(raw) == 0 || int64(len(raw)) > maxSize {
		return nil, "", "", fmt.Errorf("generated image %d exceeds the configured size limit", index)
	}
	extension, contentType, err := detectedImageType(raw)
	if err != nil {
		return nil, "", "", err
	}
	return raw, extension, contentType, nil
}

func imageResultPayloads(raw string) ([]string, []string, error) {
	var response ucloud.TaskSubmitResponse
	if err := json.Unmarshal([]byte(raw), &response); err != nil {
		return nil, nil, fmt.Errorf("decode stored image response: %w", err)
	}
	urls := make([]string, 0, len(response.Images))
	encoded := make([]string, 0, len(response.Images))
	for i := range response.Images {
		if value := strings.TrimSpace(response.Images[i].URL); value != "" {
			urls = append(urls, value)
		}
		if value := strings.TrimSpace(response.Images[i].B64JSON); value != "" {
			encoded = append(encoded, value)
		}
	}
	if len(urls) == 0 && len(encoded) == 0 {
		return nil, nil, errors.New("stored image response does not contain generated images")
	}
	return urls, encoded, nil
}

func imageURLSuffix(raw string) string {
	parsed, err := url.Parse(raw)
	if err == nil {
		switch strings.ToLower(filepath.Ext(parsed.Path)) {
		case ".jpg", ".jpeg", ".png", ".webp":
			return strings.ToLower(filepath.Ext(parsed.Path))
		}
	}
	return ".png"
}

func detectedImageType(raw []byte) (string, string, error) {
	contentType := http.DetectContentType(raw)
	switch contentType {
	case "image/jpeg":
		return ".jpg", contentType, nil
	case "image/png":
		return ".png", contentType, nil
	case "image/webp":
		return ".webp", contentType, nil
	default:
		return "", "", errors.New("generated base64 payload is not a supported image")
	}
}

func imageContentType(extension string) string {
	switch strings.ToLower(extension) {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".webp":
		return "image/webp"
	default:
		return "image/png"
	}
}

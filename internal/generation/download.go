package generation

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"

	"ai-video/internal/config"
	"ai-video/internal/gen/model"
	"ai-video/internal/pkg/tracing"
	"ai-video/internal/pkg/upload"
)

func downloadVideosToStorage(
	ctx context.Context,
	storage upload.Storage,
	client *http.Client,
	task *model.VideoUserGenerationTask,
	remoteURLs []string,
	retryCount int,
) ([]string, error) {
	maxSize := config.Cfg.Upload.VideoMaxFileSize
	if maxSize <= 0 {
		maxSize = 2 << 30
	}
	result := make([]string, 0, len(remoteURLs))
	for index, remoteURL := range remoteURLs {
		filename := fmt.Sprintf("task-%s-%d.mp4", task.TaskCode, index+1)
		storedURL, err := downloadAndStoreGeneratedFile(
			ctx, storage, client, remoteURL, generatedObjectKey(task.UserID, filename), "video/mp4", maxSize, retryCount,
		)
		if err != nil {
			return nil, err
		}
		result = append(result, storedURL)
	}
	return result, nil
}

func downloadAndStoreGeneratedFile(
	ctx context.Context,
	storage upload.Storage,
	client *http.Client,
	remoteURL, objectKey, contentType string,
	maxSize int64,
	retryCount int,
) (string, error) {
	temporary, err := newGeneratedTemporaryFile()
	if err != nil {
		return "", err
	}
	defer os.Remove(temporary)
	if err := retryDownload(ctx, retryCount, func() error {
		return downloadOne(ctx, client, remoteURL, temporary, maxSize)
	}); err != nil {
		return "", err
	}
	return storeGeneratedFile(ctx, storage, objectKey, temporary, contentType)
}

func downloadOne(ctx context.Context, client *http.Client, remoteURL, destination string, maxSize int64) error {
	if err := validatePublicHTTPURL(remoteURL); err != nil {
		return err
	}
	request, err := tracing.NewRequestWithContext(ctx, http.MethodGet, remoteURL, nil)
	if err != nil {
		return err
	}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("下载生成结果返回 HTTP %d", response.StatusCode)
	}
	if response.ContentLength > maxSize {
		return errors.New("生成结果超过配置的文件大小限制")
	}
	file, err := os.OpenFile(destination, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	written, copyErr := io.Copy(file, io.LimitReader(response.Body, maxSize+1))
	closeErr := file.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}
	if written > maxSize {
		return errors.New("生成结果超过配置的文件大小限制")
	}
	return nil
}

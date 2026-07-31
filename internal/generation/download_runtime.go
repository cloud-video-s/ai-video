package generation

import (
	"context"

	"ai-video/internal/gen/model"
	"ai-video/internal/pkg/uploadruntime"
)

func downloadVideos(ctx context.Context, task *model.VideoUserGenerationTask, remoteURLs []string) ([]string, error) {
	storage, err := uploadruntime.Storage()
	if err != nil {
		return nil, err
	}
	return downloadVideosToStorage(ctx, storage, secureDownloadClient(), task, remoteURLs)
}

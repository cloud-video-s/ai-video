package generation

import (
	"context"

	"ai-video/internal/gen/model"
	"ai-video/internal/pkg/upload"
	"ai-video/internal/pkg/uploadruntime"
)

func downloadVideos(ctx context.Context, task *model.VideoUserGenerationTask, remoteURLs []string, recorder upload.StoredUploadRecorder) ([]string, error) {
	storage, err := uploadruntime.Storage()
	if err != nil {
		return nil, err
	}
	storage = recordGeneratedUploads(storage, recorder, task, upload.MediaVideo)
	return downloadVideosToStorage(ctx, storage, secureDownloadClient(), task, remoteURLs)
}

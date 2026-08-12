package generation

import (
	"context"

	"ai-video/internal/gen/model"
	"ai-video/internal/pkg/upload"
	"ai-video/internal/pkg/uploadruntime"
)

func (m *Manager) downloadVideos(ctx context.Context, task *model.VideoUserGenerationTask, remoteURLs []string, recorder upload.StoredUploadRecorder) ([]string, error) {
	storage, err := uploadruntime.Storage()
	if err != nil {
		return nil, err
	}
	storage = recordGeneratedUploads(storage, recorder, task, upload.MediaVideo)
	var result []string
	err = m.downloadController().run(ctx, func(retryCount int) error {
		result, err = downloadVideosToStorage(ctx, storage, secureDownloadClient(), task, remoteURLs, retryCount)
		return err
	})
	return result, err
}

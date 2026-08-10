package generation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"ai-video/internal/gen/model"
	"ai-video/internal/pkg/upload"
	"ai-video/internal/pkg/uploadruntime"
)

// processVideoTaskResult resumes the local result pipeline from its persisted
// checkpoint. An empty local_urls value means the video still needs to be
// downloaded; persisted local URLs mean the next pass only generates a cover.
func (m *Manager) processVideoTaskResult(ctx context.Context, task *model.VideoUserGenerationTask) error {
	remoteURLs, err := decodeVideoResultURLs(task.RemoteUrls, true)
	if err != nil {
		return m.failTask(ctx, task, "远程视频地址无效: "+err.Error())
	}
	localURLs, err := decodeVideoResultURLs(task.LocalUrls, false)
	if err != nil {
		return m.failTask(ctx, task, "本地视频地址无效: "+err.Error())
	}
	if len(localURLs) == 0 {
		return m.downloadVideoTaskResult(ctx, task, remoteURLs)
	}
	return m.generateVideoTaskCoverAndFinish(ctx, task, remoteURLs, localURLs)
}

// downloadVideoTaskResult performs only the download stage. local_urls is the
// durable checkpoint that lets the next worker pass skip downloading and move
// on to cover generation.
func (m *Manager) downloadVideoTaskResult(ctx context.Context, task *model.VideoUserGenerationTask, remoteURLs []string) error {
	localURLs, err := downloadVideos(ctx, task, remoteURLs, m.uploadRepo)
	if err != nil {
		return m.failTask(ctx, task, "保存生成视频失败: "+err.Error())
	}
	if err := m.saveDownloadedVideoURLs(ctx, task, localURLs); err != nil {
		return err
	}
	m.hub.Publish(task)
	return nil
}

func (m *Manager) saveDownloadedVideoURLs(ctx context.Context, task *model.VideoUserGenerationTask, localURLs []string) error {
	if len(localURLs) == 0 || strings.TrimSpace(localURLs[0]) == "" {
		return errors.New("downloaded video URLs are empty")
	}
	encoded, err := json.Marshal(localURLs)
	if err != nil {
		return fmt.Errorf("encode downloaded video URLs: %w", err)
	}
	task.LocalUrls = string(encoded)
	task.Progress = 95
	task.ErrorMessage = ""
	return m.taskRepo.UpdateFields(ctx, task, "LocalUrls", "Progress", "ErrorMessage", "LastPolledAt")
}

// generateVideoTaskCoverAndFinish performs only the cover stage. It runs only
// after local_urls has been persisted by downloadVideoTaskResult.
func (m *Manager) generateVideoTaskCoverAndFinish(
	ctx context.Context,
	task *model.VideoUserGenerationTask,
	remoteURLs, localURLs []string,
) error {
	if len(localURLs) == 0 || strings.TrimSpace(localURLs[0]) == "" {
		return m.failTask(ctx, task, "生成视频封面前缺少已下载的视频地址")
	}
	storage, err := uploadruntime.Storage()
	if err != nil {
		return m.failTask(ctx, task, "获取视频封面存储失败: "+err.Error())
	}
	storage = recordGeneratedUploads(storage, m.uploadRepo, task, upload.MediaImage)
	coverSource := uploadruntime.PublicURL(localURLs[0])
	remoteCoverSource := ""
	if len(remoteURLs) > 0 {
		remoteCoverSource = strings.TrimSpace(remoteURLs[0])
	}
	coverURL, err := generateOrStoreDefaultVideoTaskCover(ctx, storage, task, coverSource, remoteCoverSource)
	if err != nil {
		return m.failTask(ctx, task, "保存视频默认封面失败: "+err.Error())
	}
	task.CoverImageURL = coverURL
	task.Status = TaskStatusSuccess
	task.Progress = 100
	task.ErrorMessage = ""
	task.FinishedAt = time.Now()
	return m.completeTask(ctx, task,
		"CoverImageURL", "Status", "Progress", "ErrorMessage", "FinishedAt",
	)
}

func decodeVideoResultURLs(raw string, required bool) ([]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		if required {
			return nil, errors.New("URL list is empty")
		}
		return []string{}, nil
	}
	var urls []string
	if err := json.Unmarshal([]byte(raw), &urls); err != nil {
		return nil, err
	}
	if len(urls) == 0 {
		if required {
			return nil, errors.New("URL list is empty")
		}
		return []string{}, nil
	}
	for i := range urls {
		urls[i] = strings.TrimSpace(urls[i])
		if urls[i] == "" {
			return nil, fmt.Errorf("URL %d is empty", i+1)
		}
	}
	return urls, nil
}

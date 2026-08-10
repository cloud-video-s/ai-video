package generation

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"ai-video/internal/gen/model"
	"ai-video/internal/repository"
)

func TestProcessTaskRecordsRemoteVideoBeforeDownloading(t *testing.T) {
	db := newGenerationManagerTestDB(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/tasks/status" {
			t.Fatalf("status path = %q", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"output": map[string]any{
				"task_id": "third-task-success", "task_status": "Success",
				"urls": []string{"https://results.example/generated.mp4"}, "finish_time": 2,
			},
			"usage": map[string]any{"duration": 8}, "request_id": "poll-success",
		})
	}))
	defer server.Close()
	seedGenerationManagerVideoModel(t, db, server.URL)

	if err := db.Exec(`INSERT INTO video_user_generation_task
		(user_id, model_id, client_request_id, task_code, third_task_code, status, progress,
		 prompt, request_payload, remote_urls, local_urls, submitted_at, created_at, updated_at)
		VALUES (19, 7, 'poll-success-client', 'order-success', 'third-task-success', ?, 50,
		 'poll prompt', '{}', '[]', '[]', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`, TaskStatusRunning).Error; err != nil {
		t.Fatal(err)
	}
	var task model.VideoUserGenerationTask
	if err := db.Where("client_request_id = ?", "poll-success-client").First(&task).Error; err != nil {
		t.Fatal(err)
	}
	manager := &Manager{
		modelRepo: repository.NewModelRepo(), taskRepo: repository.NewUserGenerationTaskRepo(), hub: NewHub(),
	}

	if err := manager.processTask(context.Background(), &task); err != nil {
		t.Fatal(err)
	}
	if task.Status != TaskStatusDownloading || task.Progress != 90 {
		t.Fatalf("queried task status/progress = %d/%d", task.Status, task.Progress)
	}
	if !task.FinishedAt.IsZero() {
		t.Fatalf("query pass marked task finished at %v", task.FinishedAt)
	}
	if task.LocalUrls != "[]" || task.CoverImageURL != "" {
		t.Fatalf("query pass persisted local result too early: local_urls=%q cover=%q", task.LocalUrls, task.CoverImageURL)
	}

	var remoteURLs []string
	if err := json.Unmarshal([]byte(task.RemoteUrls), &remoteURLs); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(remoteURLs, []string{"https://results.example/generated.mp4"}) {
		t.Fatalf("remote URLs = %#v", remoteURLs)
	}
	var persisted model.VideoUserGenerationTask
	if err := db.First(&persisted, task.ID).Error; err != nil {
		t.Fatal(err)
	}
	if persisted.Status != TaskStatusDownloading || persisted.Progress != 90 || persisted.LocalUrls != "[]" || !persisted.FinishedAt.IsZero() {
		t.Fatalf("persisted query checkpoint = status %d progress %d local_urls %q finished_at %v",
			persisted.Status, persisted.Progress, persisted.LocalUrls, persisted.FinishedAt)
	}
}

func TestSaveDownloadedVideoURLsCreatesCoverStageCheckpoint(t *testing.T) {
	db := newGenerationManagerTestDB(t)
	if err := db.Exec(`INSERT INTO video_user_generation_task
		(user_id, model_id, client_request_id, task_code, third_task_code, status, progress,
		 prompt, request_payload, remote_urls, local_urls, submitted_at, last_polled_at, created_at, updated_at)
		VALUES (19, 7, 'downloaded-client', 'downloaded-task', 'third-downloaded', ?, 90,
		 '', '{}', '["https://results.example/generated.mp4"]', '[]', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP,
		 CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`, TaskStatusDownloading).Error; err != nil {
		t.Fatal(err)
	}
	var task model.VideoUserGenerationTask
	if err := db.Where("client_request_id = ?", "downloaded-client").First(&task).Error; err != nil {
		t.Fatal(err)
	}
	task.LastPolledAt = time.Now()
	manager := &Manager{taskRepo: repository.NewUserGenerationTaskRepo()}
	wantURLs := []string{"/uploads/generated/19/downloaded-task-1.mp4"}

	if err := manager.saveDownloadedVideoURLs(context.Background(), &task, wantURLs); err != nil {
		t.Fatal(err)
	}
	if task.Status != TaskStatusDownloading || task.Progress != 95 || task.CoverImageURL != "" {
		t.Fatalf("download checkpoint = status %d progress %d cover %q", task.Status, task.Progress, task.CoverImageURL)
	}

	var persisted model.VideoUserGenerationTask
	if err := db.First(&persisted, task.ID).Error; err != nil {
		t.Fatal(err)
	}
	var localURLs []string
	if err := json.Unmarshal([]byte(persisted.LocalUrls), &localURLs); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(localURLs, wantURLs) || persisted.Progress != 95 || persisted.Status != TaskStatusDownloading {
		t.Fatalf("persisted download checkpoint = URLs %#v status %d progress %d", localURLs, persisted.Status, persisted.Progress)
	}
}

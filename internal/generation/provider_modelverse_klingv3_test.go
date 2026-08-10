package generation

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ai-video/internal/gen/model"
	"ai-video/internal/pkg/ucloud"
)

func TestModelVerseSubmitKlingV3WithFirstAndEndFrames(t *testing.T) {
	var received ucloud.KlingV3SubmitRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/custom/video-submit" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Errorf("decode request: %v", err)
		}
		_, _ = w.Write([]byte(`{"output":{"task_id":"remote-v3"},"request_id":"request-v3"}`))
	}))
	defer server.Close()

	modelConfig := &model.VideoModel{
		Code: ucloud.ModelKlingV3, ModelType: TaskTypeVideo,
		HostURL: server.URL, SubmitEndpoint: "/custom/video-submit", StatusEndpoint: "/custom/video-status",
		APIKey: "test-key", AuthType: 1,
	}
	result, err := (&ModelVerseProvider{}).Submit(context.Background(), modelConfig, remoteSubmitRequest{
		Model: modelConfig.Code,
		Input: map[string]any{
			"prompt": "sunset", "first_frame": "https://cdn.example/first.png", "end_frame": "https://cdn.example/end.png",
		},
		Parameters: map[string]any{
			"kling_v3_type": "i2v", "mode": "pro", "duration": float64(5),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.TaskID != "remote-v3" || result.RequestID != "request-v3" || result.Completed {
		t.Fatalf("unexpected result: %#v", result)
	}
	if received.Model != ucloud.ModelKlingV3 || received.Input.Prompt != "sunset" ||
		received.Parameters.Image != "https://cdn.example/first.png" ||
		received.Parameters.ImageTail != "https://cdn.example/end.png" {
		t.Fatalf("unexpected request: %#v", received)
	}
}

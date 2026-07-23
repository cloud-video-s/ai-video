package generation

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ai-video/internal/gen/model"
)

func TestModelVerseSubmitUsesMergedParameters(t *testing.T) {
	var received remoteSubmitRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/tasks/submit" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("authorization = %q", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatal(err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"output":{"task_id":"remote-1"},"request_id":"request-1"}`))
	}))
	defer server.Close()

	parameters, err := mergeParameters(`{"mode":"pro","duration":5}`, map[string]interface{}{"duration": float64(8)})
	if err != nil {
		t.Fatal(err)
	}
	parameters["external_task_id"] = "client-1"
	modelConfig := &model.VideoAiModel{
		ModelName: "kling-v3-omni",
		BaseURL:   server.URL, SubmitPath: "/v1/tasks/submit", StatusPath: "/v1/tasks/status",
		APIKey: "test-key", HTTPTimeoutSeconds: 3,
	}
	result, err := (&ModelVerseProvider{}).Submit(context.Background(), modelConfig, remoteSubmitRequest{
		Model: modelConfig.ModelName, Input: map[string]interface{}{"prompt": "sunset"}, Parameters: parameters,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.TaskID != "remote-1" {
		t.Fatalf("unexpected result: %#v", result)
	}
	if received.Model != "kling-v3-omni" || received.Parameters["mode"] != "pro" || received.Parameters["duration"] != float64(8) {
		t.Fatalf("unexpected upstream request: %#v", received)
	}
	if received.Parameters["external_task_id"] != "client-1" {
		t.Fatalf("external_task_id = %#v", received.Parameters["external_task_id"])
	}
}

func TestModelVerseStatusMapping(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("task_id") != "remote-1" {
			t.Fatalf("task_id = %q", r.URL.Query().Get("task_id"))
		}
		_, _ = w.Write([]byte(`{"output":{"task_id":"remote-1","task_status":"Success","urls":["https://cdn.example/video.mp4"],"submit_time":1,"finish_time":2},"usage":{"duration":5},"request_id":"r1"}`))
	}))
	defer server.Close()
	status, err := (&ModelVerseProvider{}).Status(context.Background(), &model.VideoAiModel{
		BaseURL: server.URL, StatusPath: "/v1/tasks/status", APIKey: "key", HTTPTimeoutSeconds: 3,
	}, "remote-1")
	if err != nil {
		t.Fatal(err)
	}
	if status.Status != "Success" || len(status.URLs) != 1 || status.UsageDuration != 5 {
		t.Fatalf("unexpected status: %#v", status)
	}
}

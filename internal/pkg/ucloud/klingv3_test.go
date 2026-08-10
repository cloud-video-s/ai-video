package ucloud

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestSubmitKlingV3Task(t *testing.T) {
	var received KlingV3SubmitRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
		}
		if r.URL.Path != "/custom/submit" {
			t.Errorf("path = %q, want /custom/submit", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Errorf("authorization = %q", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Errorf("decode request: %v", err)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"output": map[string]any{"task_id": "task-1"}, "request_id": "request-1",
		})
	}))
	defer server.Close()

	client, err := NewClient(ClientConfig{
		APIKey: "test-key", BaseURL: server.URL, SubmitEndpoint: "/custom/submit",
	})
	if err != nil {
		t.Fatal(err)
	}
	response, err := client.SubmitKlingV3Task(context.Background(), KlingV3SubmitRequest{
		Input: KlingV3Input{Prompt: "A sunset over the ocean"},
		Parameters: KlingV3Parameters{
			KlingV3Type: KlingV3TypeT2V, Mode: "pro", AspectRatio: "16:9", Duration: 5, Sound: "on",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if response.Output.TaskID != "task-1" || response.RequestID != "request-1" {
		t.Fatalf("unexpected response: %#v", response)
	}
	if received.Model != ModelKlingV3 || received.Input.Prompt != "A sunset over the ocean" ||
		received.Parameters.KlingV3Type != KlingV3TypeT2V {
		t.Fatalf("unexpected request: %#v", received)
	}
}

func TestTaskSubmitRoutesKlingV3(t *testing.T) {
	var received KlingV3SubmitRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Errorf("decode request: %v", err)
		}
		_, _ = w.Write([]byte(`{"output":{"task_id":"task-2"},"request_id":"request-2"}`))
	}))
	defer server.Close()

	client, err := NewClient(ClientConfig{APIKey: "test-key", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	response, err := client.TaskSubmit(context.Background(), TaskSubmitRequest{
		Model: ModelKlingV3, GenerationType: GenerationTypeVideo, Prompt: "Animate this image",
		Images: []string{"https://cdn.example/first.png"},
		Parameters: map[string]any{
			"kling_v3_type": "i2v", "mode": "pro", "duration": 5,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if response.Model != ModelKlingV3 || response.TaskID != "task-2" {
		t.Fatalf("unexpected response: %#v", response)
	}
	if received.Model != ModelKlingV3 || !reflect.DeepEqual(received.Input.Images, []string{"https://cdn.example/first.png"}) {
		t.Fatalf("unexpected request: %#v", received)
	}
}

func TestValidateKlingV3RequestModes(t *testing.T) {
	tests := []struct {
		name    string
		request KlingV3SubmitRequest
		wantErr string
	}{
		{
			name: "text to video",
			request: KlingV3SubmitRequest{Input: KlingV3Input{Prompt: "A cat running"}, Parameters: KlingV3Parameters{
				KlingV3Type: KlingV3TypeT2V, Duration: 15,
			}},
		},
		{
			name: "image to video inferred",
			request: KlingV3SubmitRequest{Input: KlingV3Input{Prompt: "A cat running"}, Parameters: KlingV3Parameters{
				Image: "raw-base64-image", ImageTail: "raw-base64-tail", Duration: 3,
			}},
		},
		{
			name: "motion control",
			request: KlingV3SubmitRequest{Input: KlingV3Input{
				ImgURL: "raw-base64-image", VideoURL: "https://cdn.example/motion.mp4",
			}, Parameters: KlingV3Parameters{
				KlingV3Type: KlingV3TypeMotionControl, CharacterOrientation: "image", Duration: 10,
			}},
		},
		{
			name: "multi shot",
			request: KlingV3SubmitRequest{Parameters: KlingV3Parameters{
				KlingV3Type: KlingV3TypeT2V, Duration: 5, MultiShot: true, ShotType: "customize",
				MultiPrompt: []KlingV3MultiPromptItem{
					{Index: 1, Prompt: "Wide shot", Duration: "2"},
					{Index: 2, Prompt: "Close-up", Duration: "3"},
				},
			}},
		},
		{
			name: "tail without first frame",
			request: KlingV3SubmitRequest{Input: KlingV3Input{Prompt: "Animate"}, Parameters: KlingV3Parameters{
				ImageTail: "raw-base64-tail",
			}},
			wantErr: "image_tail requires image",
		},
		{
			name: "motion control missing image",
			request: KlingV3SubmitRequest{Input: KlingV3Input{VideoURL: "https://cdn.example/motion.mp4"}, Parameters: KlingV3Parameters{
				KlingV3Type: KlingV3TypeMotionControl, CharacterOrientation: "video", Duration: 5,
			}},
			wantErr: "requires input.video_url and input.img_url",
		},
		{
			name: "multi shot duration mismatch",
			request: KlingV3SubmitRequest{Parameters: KlingV3Parameters{
				KlingV3Type: KlingV3TypeT2V, Duration: 5, MultiShot: true, ShotType: "customize",
				MultiPrompt: []KlingV3MultiPromptItem{{Index: 1, Prompt: "Wide shot", Duration: "4"}},
			}},
			wantErr: "durations total 4, want 5",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateKlingV3Request(&test.request)
			if test.wantErr == "" && err != nil {
				t.Fatal(err)
			}
			if test.wantErr != "" && (err == nil || !strings.Contains(err.Error(), test.wantErr)) {
				t.Fatalf("error = %v, want substring %q", err, test.wantErr)
			}
		})
	}
}

func TestGetKlingV3TaskStatus(t *testing.T) {
	const taskID = "task id/1"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %q, want GET", r.Method)
		}
		if r.URL.Path != "/custom/status" {
			t.Errorf("path = %q, want /custom/status", r.URL.Path)
		}
		if got := r.URL.Query().Get("task_id"); got != taskID {
			t.Errorf("task_id = %q, want %q", got, taskID)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Errorf("authorization = %q", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"output": map[string]any{
				"task_id": taskID, "task_status": "Success", "urls": []string{"https://cdn.example/video.mp4"},
				"submit_time": int64(1756959000), "finish_time": int64(1756959050),
			},
			"usage": map[string]any{"duration": 5}, "request_id": "request-3",
		})
	}))
	defer server.Close()

	client, err := NewClient(ClientConfig{
		APIKey: "test-key", BaseURL: server.URL, StatusEndpoint: "/custom/status",
	})
	if err != nil {
		t.Fatal(err)
	}
	response, err := client.GetKlingV3TaskStatus(context.Background(), taskID)
	if err != nil {
		t.Fatal(err)
	}
	if response.Output.TaskStatus != KlingV3TaskStatusSuccess ||
		!reflect.DeepEqual(response.Output.URLs, []string{"https://cdn.example/video.mp4"}) ||
		response.Usage == nil || response.Usage.Duration != 5 || response.RequestID != "request-3" {
		t.Fatalf("unexpected response: %#v", response)
	}
}

func TestGetKlingV3TaskStatusReturnsAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"code":"invalid_task","message":"task not found"},"request_id":"request-4"}`))
	}))
	defer server.Close()

	client, err := NewClient(ClientConfig{APIKey: "test-key", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.GetKlingV3TaskStatus(context.Background(), "missing-task")
	var apiError *APIError
	if !errors.As(err, &apiError) || apiError.StatusCode != http.StatusBadRequest || apiError.Code != "invalid_task" {
		t.Fatalf("error = %#v", err)
	}
}

func TestNewClientValidatesStatusEndpoint(t *testing.T) {
	_, err := NewClient(ClientConfig{
		APIKey: "test-key", BaseURL: "https://api.modelverse.cn", StatusEndpoint: "https://example.com/status",
	})
	if err == nil || !strings.Contains(err.Error(), "status endpoint") {
		t.Fatalf("error = %v", err)
	}
}

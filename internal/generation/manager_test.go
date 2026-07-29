package generation

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"ai-video/internal/gen/model"
	"ai-video/internal/pkg/ucloud"
)

func TestModelVerseSubmitVideoWithFirstAndEndFrames(t *testing.T) {
	var received ucloud.KlingO3SubmitRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/custom/video-submit" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("authorization = %q", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte(`{"output":{"task_id":"remote-1"},"request_id":"request-1"}`))
	}))
	defer server.Close()

	modelConfig := &model.VideoModel{
		Code: ucloud.ModelKlingO3, ModelType: TaskTypeVideo,
		HostURL: server.URL, SubmitEndpoint: "/custom/video-submit", StatusEndpoint: "/v1/tasks/status",
		APIKey: "test-key", AuthType: 1,
	}
	result, err := (&ModelVerseProvider{}).Submit(context.Background(), modelConfig, remoteSubmitRequest{
		Model: modelConfig.Code,
		Input: map[string]interface{}{
			"prompt": "sunset", "first_frame": "https://cdn.example/first.png", "end_frame": "https://cdn.example/end.png",
		},
		Parameters: map[string]interface{}{"mode": "pro", "aspect_ratio": "16:9", "duration": float64(5)},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.TaskID != "remote-1" || result.Completed {
		t.Fatalf("unexpected result: %#v", result)
	}
	if received.Model != ucloud.ModelKlingO3 || received.Input.Prompt != "sunset" || len(received.Parameters.ImageList) != 2 {
		t.Fatalf("unexpected upstream request: %#v", received)
	}
	if received.Parameters.ImageList[0].Type != ucloud.KlingO3ImageTypeFirstFrame ||
		received.Parameters.ImageList[1].Type != ucloud.KlingO3ImageTypeEndFrame {
		t.Fatalf("unexpected frame types: %#v", received.Parameters.ImageList)
	}
}

func TestModelVerseSubmitImageWithReferences(t *testing.T) {
	var received ucloud.DoubaoSeedreamGenerationRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/custom/image-submit" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte(`{"model":"doubao-seedream-4.5","created":1,"data":[{"url":"https://cdn.example/result.png"}]}`))
	}))
	defer server.Close()

	result, err := (&ModelVerseProvider{}).Submit(context.Background(), &model.VideoModel{
		Code: ucloud.ModelDoubaoSeedream, ModelType: TaskTypeImage,
		HostURL: server.URL, SubmitEndpoint: "/custom/image-submit", APIKey: "key", AuthType: 1,
	}, remoteSubmitRequest{
		Input: map[string]interface{}{
			"prompt": "a small robot", "images": []interface{}{"https://cdn.example/a.png", "https://cdn.example/b.png"},
		},
		Parameters: map[string]interface{}{"size": "2K", "response_format": "url"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Completed || !reflect.DeepEqual(result.URLs, []string{"https://cdn.example/result.png"}) {
		t.Fatalf("unexpected result: %#v", result)
	}
	if received.Prompt != "a small robot" || len(received.Images) != 2 || received.Size != "2K" {
		t.Fatalf("unexpected upstream request: %#v", received)
	}
}

func TestGenerationInputModes(t *testing.T) {
	tests := []struct {
		name     string
		taskType uint32
		input    map[string]interface{}
		wantErr  string
	}{
		{name: "text to image", taskType: TaskTypeImage, input: map[string]interface{}{"prompt": "cat"}},
		{name: "text and multiple images to image", taskType: TaskTypeImage, input: map[string]interface{}{"prompt": "cat", "images": []interface{}{"a", "b"}}},
		{name: "text to video", taskType: TaskTypeVideo, input: map[string]interface{}{"prompt": "cat runs"}},
		{name: "text and video to video", taskType: TaskTypeVideo, input: map[string]interface{}{"prompt": "restyle", "video": "https://cdn.example/a.mp4"}},
		{name: "text and multiple images to video", taskType: TaskTypeVideo, input: map[string]interface{}{"prompt": "animate", "images": []interface{}{"a", "b"}}},
		{name: "first and end frames", taskType: TaskTypeVideo, input: map[string]interface{}{"prompt": "transition", "first_frame": "a", "end_frame": "b"}},
		{name: "end frame without first", taskType: TaskTypeVideo, input: map[string]interface{}{"prompt": "transition", "end_frame": "b"}, wantErr: "requires"},
		{name: "frames mixed with images", taskType: TaskTypeVideo, input: map[string]interface{}{"prompt": "transition", "images": []interface{}{"a"}, "first_frame": "b"}, wantErr: "cannot be combined"},
		{name: "image with video", taskType: TaskTypeImage, input: map[string]interface{}{"prompt": "cat", "video": "a"}, wantErr: "only supports"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := generationInputFromMap(test.taskType, test.input)
			if test.wantErr == "" && err != nil {
				t.Fatal(err)
			}
			if test.wantErr != "" && (err == nil || !strings.Contains(err.Error(), test.wantErr)) {
				t.Fatalf("error = %v, want substring %q", err, test.wantErr)
			}
		})
	}
}

func TestMergeModelParametersUsesConfiguredOptions(t *testing.T) {
	definitions := []model.VideoModelParameter{
		{ParamKey: "mode", ParamType: "string", DefaultValue: `"pro"`, AllowedValues: `["std","pro"]`, ParameterType: 1},
		{ParamKey: "duration", ParamType: "integer", DefaultValue: `5`, Constraints: `{"min":3,"max":15}`, ParameterType: 1},
		{ParamKey: "prompt", ParamType: "string", IsRequired: 1, ParameterType: 2},
	}
	parameters, err := mergeModelParameters(definitions, map[string]interface{}{"duration": float64(8)})
	if err != nil {
		t.Fatal(err)
	}
	if parameters["mode"] != "pro" || parameters["duration"] != float64(8) {
		t.Fatalf("unexpected parameters: %#v", parameters)
	}
	if _, exists := parameters["prompt"]; exists {
		t.Fatalf("request parameter leaked into model options: %#v", parameters)
	}
	if _, err := mergeModelParameters(definitions, map[string]interface{}{"duration": float64(20)}); err == nil {
		t.Fatal("duration above configured max must fail")
	}
	if _, err := mergeModelParameters(definitions, map[string]interface{}{"unknown": true}); err == nil {
		t.Fatal("unknown parameter must fail")
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
	status, err := (&ModelVerseProvider{}).Status(context.Background(), &model.VideoModel{
		HostURL: server.URL, StatusEndpoint: "/v1/tasks/status", APIKey: "key", AuthType: 1,
	}, "remote-1")
	if err != nil {
		t.Fatal(err)
	}
	if status.Status != "Success" || len(status.URLs) != 1 || status.UsageDuration != 5 {
		t.Fatalf("unexpected status: %#v", status)
	}
}

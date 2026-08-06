package generation

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	appconfig "ai-video/internal/config"
	"ai-video/internal/gen/model"
	"ai-video/internal/pkg/ucloud"
	"ai-video/internal/repository"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestCreateTaskQueuesBeforeProviderSubmission(t *testing.T) {
	db := newGenerationManagerTestDB(t)
	submitted := make(chan ucloud.KlingO3SubmitRequest, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request ucloud.KlingO3SubmitRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		submitted <- request
		_, _ = w.Write([]byte(`{"output":{"task_id":"remote-queued"},"request_id":"request-queued"}`))
	}))
	defer server.Close()

	seedGenerationManagerVideoModel(t, db, server.URL)

	manager := &Manager{
		modelRepo: repository.NewModelRepo(), parameterRepo: repository.NewModelParameterRepo(),
		taskRepo: repository.NewUserGenerationTaskRepo(), hub: NewHub(),
	}
	task, err := manager.CreateTask(context.Background(), 19, &CreateTaskRequest{
		ModelCode: ucloud.ModelKlingO3, TaskType: TaskTypeVideo,
		ClientRequestID: "client-async-1", Input: map[string]any{"prompt": "queued prompt"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if task.ID == 0 || task.TaskCode == "" || task.Status != TaskStatusSubmitting || task.Progress != 0 {
		t.Fatalf("queued task = %+v", task)
	}
	select {
	case request := <-submitted:
		t.Fatalf("provider was called synchronously: %+v", request)
	default:
	}

	var queuedRequest remoteSubmitRequest
	if err := json.Unmarshal([]byte(task.RequestPayload), &queuedRequest); err != nil {
		t.Fatal(err)
	}
	if queuedRequest.Model != ucloud.ModelKlingO3 || queuedRequest.Parameters["external_task_id"] != task.TaskCode {
		t.Fatalf("queued provider request = %#v, task code = %q", queuedRequest, task.TaskCode)
	}

	if err := manager.processTask(context.Background(), task); err != nil {
		t.Fatal(err)
	}
	var providerRequest ucloud.KlingO3SubmitRequest
	select {
	case providerRequest = <-submitted:
	default:
		t.Fatal("background processing did not submit the queued task")
	}
	if providerRequest.Parameters.ExternalTaskID != task.TaskCode {
		t.Fatalf("provider external_task_id = %q, want %q", providerRequest.Parameters.ExternalTaskID, task.TaskCode)
	}
	if task.Status != TaskStatusSubmitted || task.ThirdTaskCode != "remote-queued" || task.SubmittedAt.IsZero() {
		t.Fatalf("processed task = %+v", task)
	}
	persisted, err := manager.taskRepo.GetOwned(context.Background(), task.ID, task.UserID)
	if err != nil {
		t.Fatal(err)
	}
	if persisted.Status != TaskStatusSubmitted || persisted.ThirdTaskCode != "remote-queued" {
		t.Fatalf("persisted task = %+v", persisted)
	}
}

func TestCreateTemplateTaskDerivesSettingsAndPersistsTemplateID(t *testing.T) {
	db := newGenerationManagerTestDB(t)
	seedGenerationManagerVideoModel(t, db, "https://model.example.com")
	if err := db.Exec(`INSERT INTO video_template
		(id, name, template_type, template_type_id, model_id, prompt, status, created_at, updated_at)
		VALUES (9, 'Animate portrait', 2, 3, 7, 'use the server template prompt', 1,
		 CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO video_template_model_parameter
		(id, model_id, template_id, param_key, param_type, is_required, default_value,
		 allowed_values, sort_order, parameter_type, constraints, created_at, updated_at)
		VALUES (1, 7, 9, 'aspect_ratio', 'string', 0, '"1:1"', '["1:1"]',
		 1, 1, '{}', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`).Error; err != nil {
		t.Fatal(err)
	}

	manager := &Manager{
		modelRepo: repository.NewModelRepo(), parameterRepo: repository.NewModelParameterRepo(),
		taskRepo: repository.NewUserGenerationTaskRepo(), templateRepo: repository.NewTemplateRepo(),
		templateParameterRepo: repository.NewTemplateModelParameterRepo(), hub: NewHub(),
	}
	task, err := manager.CreateTemplateTask(context.Background(), 19, &CreateTemplateTaskRequest{
		TemplateID: 9, ClientRequestID: "template-client-1",
		Input: map[string]any{
			"prompt": "client prompt must not win",
			"images": []any{"https://cdn.example.com/reference.png"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if task.TemplateID != 9 || task.ModelID != 7 || task.TaskType != TaskTypeVideo {
		t.Fatalf("derived task settings = %+v", task)
	}
	var payload remoteSubmitRequest
	if err := json.Unmarshal([]byte(task.RequestPayload), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Input["prompt"] != "use the server template prompt" {
		t.Fatalf("payload prompt = %#v", payload.Input["prompt"])
	}
	if payload.Parameters["aspect_ratio"] != "1:1" {
		t.Fatalf("payload parameters = %#v", payload.Parameters)
	}
	var persistedTemplateID uint64
	if err := db.Table(model.TableNameVideoUserGenerationTask).
		Select("template_id").Where("id = ?", task.ID).Scan(&persistedTemplateID).Error; err != nil {
		t.Fatal(err)
	}
	if persistedTemplateID != 9 {
		t.Fatalf("persisted template_id = %d, want 9", persistedTemplateID)
	}
	if view := ViewOf(task); view.TemplateID != 9 {
		t.Fatalf("task view template_id = %d, want 9", view.TemplateID)
	}
}

func TestTemplateTaskType(t *testing.T) {
	tests := []struct {
		name    string
		value   int64
		want    uint32
		wantErr bool
	}{
		{name: "image", value: 1, want: TaskTypeImage},
		{name: "video", value: 2, want: TaskTypeVideo},
		{name: "unsupported", value: 3, wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := templateTaskType(test.value)
			if test.wantErr {
				if err == nil {
					t.Fatalf("templateTaskType(%d) succeeded", test.value)
				}
				return
			}
			if err != nil || got != test.want {
				t.Fatalf("templateTaskType(%d) = %d, %v; want %d", test.value, got, err, test.want)
			}
		})
	}
}

func TestProcessTaskPollsThirdTaskCodeUntilFailure(t *testing.T) {
	db := newGenerationManagerTestDB(t)
	statuses := []string{"Pending", "Running", "Failure"}
	var pollCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/tasks/status" {
			t.Errorf("status path = %q", r.URL.Path)
		}
		if got := r.URL.Query().Get("task_id"); got != "third-task-88" {
			t.Errorf("task_id = %q", got)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer model-key" {
			t.Errorf("authorization = %q", got)
		}
		index := int(pollCount.Add(1)) - 1
		if index >= len(statuses) {
			t.Errorf("unexpected poll %d", index+1)
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		output := map[string]any{
			"task_id": "third-task-88", "task_status": statuses[index],
			"submit_time": 1, "finish_time": 2,
		}
		if statuses[index] == "Failure" {
			output["error_message"] = "provider generation failed"
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"output": output, "usage": map[string]any{"duration": 5}, "request_id": "poll-request",
		})
	}))
	defer server.Close()
	seedGenerationManagerVideoModel(t, db, server.URL)

	if err := db.Exec(`INSERT INTO video_user_generation_task
		(user_id, model_id, client_request_id, task_code, third_task_code, status, progress,
		 prompt, request_payload, remote_urls, local_urls, submitted_at, created_at, updated_at)
		VALUES (19, 7, 'poll-client-1', 'order-88', 'third-task-88', ?, 5,
		 'poll prompt', '{}', '[]', '[]', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`, TaskStatusSubmitted).Error; err != nil {
		t.Fatal(err)
	}
	var task model.VideoUserGenerationTask
	if err := db.Where("client_request_id = ?", "poll-client-1").First(&task).Error; err != nil {
		t.Fatal(err)
	}
	manager := &Manager{
		modelRepo: repository.NewModelRepo(), parameterRepo: repository.NewModelParameterRepo(),
		taskRepo: repository.NewUserGenerationTaskRepo(), hub: NewHub(),
	}

	if err := manager.processTask(context.Background(), &task); err != nil {
		t.Fatal(err)
	}
	if task.Status != TaskStatusPending || task.Progress != 10 {
		t.Fatalf("pending task = %+v", task)
	}
	makeTaskPollable(t, db, &task)
	if err := manager.processTask(context.Background(), &task); err != nil {
		t.Fatal(err)
	}
	if task.Status != TaskStatusRunning || task.Progress != 50 || task.StartedAt.IsZero() {
		t.Fatalf("running task = %+v", task)
	}
	makeTaskPollable(t, db, &task)
	err := manager.processTask(context.Background(), &task)
	if err == nil || !strings.Contains(err.Error(), "provider generation failed") {
		t.Fatalf("failure error = %v", err)
	}
	if task.Status != TaskStatusFailure || task.Progress != 100 || task.UsageDuration != 5 {
		t.Fatalf("failed task = %+v", task)
	}
	persisted, err := manager.taskRepo.GetOwned(context.Background(), task.ID, task.UserID)
	if err != nil {
		t.Fatal(err)
	}
	if persisted.Status != TaskStatusFailure || persisted.UsageDuration != 5 ||
		!strings.Contains(persisted.ProviderResponse, `"task_status":"Failure"`) {
		t.Fatalf("persisted failed task = %+v", persisted)
	}
}

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
	tests := []struct {
		status       string
		urls         []string
		errorMessage string
	}{
		{status: "Pending"},
		{status: "Running"},
		{status: "Success", urls: []string{"https://cdn.example/video.mp4"}},
		{status: "Failure", errorMessage: "provider failed"},
	}
	for _, test := range tests {
		t.Run(test.status, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/v1/tasks/status" {
					t.Fatalf("path = %q", r.URL.Path)
				}
				if r.URL.Query().Get("task_id") != "remote-1" {
					t.Fatalf("task_id = %q", r.URL.Query().Get("task_id"))
				}
				if got := r.Header.Get("Authorization"); got != "Bearer key" {
					t.Fatalf("authorization = %q", got)
				}
				_ = json.NewEncoder(w).Encode(map[string]any{
					"output": map[string]any{
						"task_id": "remote-1", "task_status": test.status, "urls": test.urls,
						"submit_time": 1, "finish_time": 2, "error_message": test.errorMessage,
					},
					"usage": map[string]any{"duration": 5}, "request_id": "r1",
				})
			}))
			defer server.Close()

			status, err := (&ModelVerseProvider{}).Status(context.Background(), &model.VideoModel{
				HostURL: server.URL, APIKey: "key", AuthType: 1,
			}, "remote-1")
			if err != nil {
				t.Fatal(err)
			}
			if status.TaskID != "remote-1" || status.Status != test.status ||
				!reflect.DeepEqual(status.URLs, test.urls) || status.ErrorMessage != test.errorMessage || status.UsageDuration != 5 {
				t.Fatalf("unexpected status: %#v", status)
			}
		})
	}
}

func TestModelVerseStatusRejectsInvalidThirdTaskCode(t *testing.T) {
	provider := &ModelVerseProvider{}
	if _, err := provider.Status(context.Background(), &model.VideoModel{}, " "); err == nil ||
		!strings.Contains(err.Error(), "third_task_code") {
		t.Fatalf("empty third_task_code error = %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"output":{"task_id":"another-task","task_status":"Pending"},"request_id":"r1"}`))
	}))
	defer server.Close()
	_, err := provider.Status(context.Background(), &model.VideoModel{
		HostURL: server.URL, StatusEndpoint: "/v1/tasks/status", APIKey: "key", AuthType: 1,
	}, "expected-task")
	if err == nil || !strings.Contains(err.Error(), "不一致") {
		t.Fatalf("mismatched task_id error = %v", err)
	}
}

func newGenerationManagerTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + strings.ReplaceAll(t.Name(), "/", "-") + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	schemas := []string{
		`CREATE TABLE video_platform (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			code TEXT NOT NULL UNIQUE,
			base_url TEXT NOT NULL,
			description TEXT,
			status INTEGER NOT NULL DEFAULT 1,
			auth_type INTEGER NOT NULL DEFAULT 1,
			api_key TEXT,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME NULL
		)`,
		`CREATE TABLE video_model (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			platform_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			code TEXT NOT NULL UNIQUE,
			model_type INTEGER NOT NULL,
			version TEXT NOT NULL,
			submit_endpoint TEXT NOT NULL,
			status_endpoint TEXT NOT NULL,
			request_method TEXT NOT NULL,
			auth_type INTEGER NOT NULL,
			description TEXT,
			status INTEGER NOT NULL,
			host_url TEXT NOT NULL,
			score INTEGER NOT NULL,
			api_key_config TEXT,
			api_key TEXT,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME NULL
		)`,
		`CREATE TABLE video_model_parameter (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			model_id INTEGER NOT NULL,
			param_key TEXT NOT NULL,
			param_type TEXT NOT NULL,
			is_required INTEGER NOT NULL,
			default_value TEXT,
			allowed_values TEXT,
			description TEXT,
			sort_order INTEGER NOT NULL,
			parameter_type INTEGER NOT NULL,
			constraints TEXT,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME NULL
		)`,
		`CREATE TABLE video_template (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			template_type INTEGER NOT NULL,
			template_type_id INTEGER NOT NULL,
			model_id INTEGER NOT NULL,
			prompt TEXT NOT NULL,
			status INTEGER NOT NULL DEFAULT 1,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME NULL
		)`,
		`CREATE TABLE video_template_model_parameter (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			model_id INTEGER NOT NULL,
			template_id INTEGER NOT NULL,
			param_key TEXT NOT NULL,
			param_type TEXT NOT NULL,
			is_required INTEGER NOT NULL DEFAULT 0,
			default_value TEXT,
			allowed_values TEXT,
			description TEXT,
			sort_order INTEGER NOT NULL DEFAULT 0,
			parameter_type INTEGER NOT NULL DEFAULT 1,
			constraints TEXT,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME NULL
		)`,
		`CREATE TABLE video_user_generation_task (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			model_id INTEGER NOT NULL,
			client_request_id TEXT NOT NULL,
			task_code TEXT NOT NULL UNIQUE,
			third_task_code TEXT,
			status INTEGER NOT NULL,
			task_type INTEGER NOT NULL DEFAULT 1,
			progress INTEGER NOT NULL DEFAULT 0,
			prompt TEXT,
			request_payload TEXT NOT NULL,
			provider_response TEXT,
			remote_urls TEXT,
			local_urls TEXT,
			error_message TEXT,
			usage_duration INTEGER NOT NULL DEFAULT 0,
			submitted_at DATETIME NULL,
			started_at DATETIME NULL,
			finished_at DATETIME NULL,
			last_polled_at DATETIME NULL,
			template_id INTEGER NOT NULL DEFAULT 0,
			cover_image_url TEXT,
			score INTEGER NOT NULL DEFAULT 0,
			score_type INTEGER NOT NULL DEFAULT 0,
			vip_score INTEGER NOT NULL DEFAULT 0,
			points_score INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			deleted_at DATETIME NULL,
			UNIQUE (user_id, client_request_id),
			UNIQUE (third_task_code)
		)`,
		`CREATE TABLE video_user (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			vip_points INTEGER NOT NULL DEFAULT 0,
			points_balance INTEGER NOT NULL DEFAULT 0,
			frozen_points INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			deleted_at DATETIME NULL
		)`,
		`CREATE TABLE video_user_points_ledger (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			direction INTEGER NOT NULL,
			points_change INTEGER NOT NULL,
			balance_before INTEGER NOT NULL,
			balance_after INTEGER NOT NULL,
			description TEXT,
			source_type INTEGER NOT NULL,
			order_code TEXT,
			points_id INTEGER NOT NULL DEFAULT 0,
			vip_id INTEGER NOT NULL DEFAULT 0,
			occurred_at DATETIME NOT NULL,
			admin_id INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			deleted_at DATETIME NULL
		)`,
	}
	for _, schema := range schemas {
		if err := db.Exec(schema).Error; err != nil {
			t.Fatal(err)
		}
	}

	previousDB := appconfig.DB
	appconfig.DB = db
	t.Cleanup(func() {
		appconfig.DB = previousDB
		if sqlDB, dbErr := db.DB(); dbErr == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

func seedGenerationManagerVideoModel(t *testing.T, db *gorm.DB, hostURL string) {
	t.Helper()
	if err := db.Exec(`INSERT INTO video_user
		(id, vip_points, points_balance, frozen_points, created_at, updated_at)
		VALUES (19, 100, 100, 0, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO video_platform
		(id, name, code, base_url, status, auth_type, api_key, created_at, updated_at)
		VALUES (1, 'ModelVerse', 'modelverse', ?, 1, 1, 'platform-key', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`, hostURL).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO video_model
		(id, platform_id, name, code, model_type, version, submit_endpoint, status_endpoint,
		 request_method, auth_type, status, host_url, score, api_key, created_at, updated_at)
		VALUES (7, 1, 'Kling O3', ?, 2, '3.0', '/submit', '/v1/tasks/status',
		 'POST', 1, 1, ?, 1, 'model-key', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`, ucloud.ModelKlingO3, hostURL).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO video_model_parameter
		(id, model_id, param_key, param_type, is_required, default_value, allowed_values,
		 sort_order, parameter_type, constraints, created_at, updated_at)
		VALUES (1, 7, 'aspect_ratio', 'string', 1, '"16:9"', '["16:9","1:1"]',
		 1, 1, '{}', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`).Error; err != nil {
		t.Fatal(err)
	}
}

func makeTaskPollable(t *testing.T, db *gorm.DB, task *model.VideoUserGenerationTask) {
	t.Helper()
	lastPolledAt := time.Now().Add(-4 * time.Second)
	if err := db.Model(&model.VideoUserGenerationTask{}).
		Where("id = ?", task.ID).Update("last_polled_at", lastPolledAt).Error; err != nil {
		t.Fatal(err)
	}
	task.LastPolledAt = lastPolledAt
}

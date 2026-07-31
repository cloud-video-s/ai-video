package generation

import (
	"encoding/json"
	"time"

	"ai-video/internal/gen/model"
)

const (
	TaskTypeImage uint32 = 1
	TaskTypeVideo uint32 = 2

	TaskStatusSubmitting  = 1 // 提交中
	TaskStatusSubmitted   = 2 // 已提交
	TaskStatusPending     = 3 // 等待处理
	TaskStatusRunning     = 4 // 运行中
	TaskStatusDownloading = 5 // 下载中
	TaskStatusSuccess     = 6 // 成功
	TaskStatusFailure     = 7 // 失败
)

// CreateTaskRequest 是客户端通用生成请求。
// Input 和 Parameters 会与模型默认参数合并后发送给具体 provider。
type CreateTaskRequest struct {
	ModelCode       string         `json:"model_code" binding:"required,max=64"`
	TaskType        uint32         `json:"task_type" binding:"required,oneof=1 2"`
	ClientRequestID string         `json:"client_request_id" binding:"omitempty,max=64"`
	Input           map[string]any `json:"input" binding:"required"`
	Parameters      map[string]any `json:"parameters,omitempty"`
	TemplateID      uint64         `json:"-"`
}

// CreateTemplateTaskRequest selects the template-owned model, task type,
// prompt, and parameter defaults on the server. Input only carries optional
// user media references; a client-supplied prompt is replaced by the template
// prompt before the task is queued.
type CreateTemplateTaskRequest struct {
	TemplateID      uint64         `json:"template_id" binding:"required"`
	ClientRequestID string         `json:"client_request_id" binding:"omitempty,max=64"`
	Input           map[string]any `json:"input,omitempty"`
	Parameters      map[string]any `json:"parameters,omitempty"`
}

// GenerationInput provides the media combinations supported by the UCloud
// image and video models. The selected model's model_type determines which
// fields are legal.
type GenerationInput struct {
	Prompt     string   `json:"prompt"`
	Images     []string `json:"images,omitempty"`
	Video      string   `json:"video,omitempty"`
	FirstFrame string   `json:"first_frame,omitempty"`
	EndFrame   string   `json:"end_frame,omitempty"`
}

type remoteSubmitRequest struct {
	Model      string                 `json:"model"`
	Input      map[string]interface{} `json:"input"`
	Parameters map[string]interface{} `json:"parameters"`
}

type ProviderSubmitResult struct {
	TaskID       string
	RequestID    string
	RawResponse  string
	Completed    bool
	URLs         []string
	Base64Images []string
}

type ProviderTaskStatus struct {
	TaskID        string
	Status        string
	URLs          []string
	ErrorMessage  string
	UsageDuration uint32
	SubmitTime    int64
	FinishTime    int64
	RequestID     string
	RawResponse   string
}

// TaskView 是客户端可见的任务快照，不暴露第三方临时 URL 和原始响应。
type TaskView struct {
	ID            uint64         `json:"id"`
	TaskCode      string         `json:"task_code"`
	TemplateID    uint64         `json:"template_id,omitempty"`
	TaskType      uint32         `json:"task_type"`
	Status        int            `json:"status"`
	Progress      uint8          `json:"progress"`
	Input         map[string]any `json:"input,omitempty"`
	Parameters    map[string]any `json:"parameters,omitempty"`
	LocalURLs     []string       `json:"local_urls"`
	ErrorMessage  string         `json:"error_message,omitempty"`
	UsageDuration uint32         `json:"usage_duration"`
	SubmittedAt   *time.Time     `json:"submitted_at,omitempty"`
	StartedAt     *time.Time     `json:"started_at,omitempty"`
	FinishedAt    *time.Time     `json:"finished_at,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

func ViewOf(item *model.VideoUserGenerationTask) TaskView {
	view := TaskView{
		ID: item.ID, TaskCode: item.TaskCode, TemplateID: item.TemplateID, TaskType: item.TaskType,
		Status: item.Status, Progress: uint8(item.Progress),
		ErrorMessage: item.ErrorMessage, UsageDuration: item.UsageDuration,
		SubmittedAt: nullableTime(item.SubmittedAt), StartedAt: nullableTime(item.StartedAt), FinishedAt: nullableTime(item.FinishedAt),
		CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt, LocalURLs: []string{},
	}
	var request remoteSubmitRequest
	if json.Unmarshal([]byte(item.RequestPayload), &request) == nil {
		view.Input = request.Input
		view.Parameters = request.Parameters
	}
	_ = json.Unmarshal([]byte(item.LocalUrls), &view.LocalURLs)
	if view.LocalURLs == nil {
		view.LocalURLs = []string{}
	}
	return view
}

func nullableTime(value time.Time) *time.Time {
	if value.IsZero() {
		return nil
	}
	return &value
}

func IsTerminal(status int) bool {
	return status == TaskStatusSuccess || status == TaskStatusFailure
}

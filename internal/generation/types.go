package generation

import (
	"encoding/json"
	"time"

	"ai-video/internal/gen/model"
)

const (
	TaskStatusSubmitting  = "submitting"
	TaskStatusSubmitted   = "submitted"
	TaskStatusPending     = "pending"
	TaskStatusRunning     = "running"
	TaskStatusDownloading = "downloading"
	TaskStatusSuccess     = "success"
	TaskStatusFailure     = "failure"
)

// CreateTaskRequest 是客户端通用生成请求。
// Input 和 Parameters 会与模型默认参数合并后发送给具体 provider。
type CreateTaskRequest struct {
	ModelCode       string                 `json:"model_code" binding:"required,max=64"`
	ClientRequestID string                 `json:"client_request_id" binding:"omitempty,max=64"`
	Input           map[string]interface{} `json:"input" binding:"required"`
	Parameters      map[string]interface{} `json:"parameters"`
}

type remoteSubmitRequest struct {
	Model      string                 `json:"model"`
	Input      map[string]interface{} `json:"input"`
	Parameters map[string]interface{} `json:"parameters"`
}

type ProviderSubmitResult struct {
	TaskID      string
	RequestID   string
	RawResponse string
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
	ID              uint64                 `json:"id"`
	ClientRequestID string                 `json:"client_request_id"`
	ModelConfigID   uint64                 `json:"model_config_id"`
	ExternalTaskID  string                 `json:"external_task_id,omitempty"`
	Status          string                 `json:"status"`
	Progress        uint8                  `json:"progress"`
	Input           map[string]interface{} `json:"input,omitempty"`
	Parameters      map[string]interface{} `json:"parameters,omitempty"`
	LocalURLs       []string               `json:"local_urls"`
	ErrorMessage    string                 `json:"error_message,omitempty"`
	UsageDuration   uint32                 `json:"usage_duration"`
	SubmittedAt     *time.Time             `json:"submitted_at,omitempty"`
	StartedAt       *time.Time             `json:"started_at,omitempty"`
	FinishedAt      *time.Time             `json:"finished_at,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

func ViewOf(item *model.VideoGenerationTask) TaskView {
	view := TaskView{
		ID: item.ID, ClientRequestID: item.ClientRequestID, ModelConfigID: item.ModelConfigID,
		ExternalTaskID: item.ExternalTaskID, Status: item.Status, Progress: item.Progress,
		ErrorMessage: item.ErrorMessage, UsageDuration: item.UsageDuration,
		SubmittedAt: item.SubmittedAt, StartedAt: item.StartedAt, FinishedAt: item.FinishedAt,
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

func IsTerminal(status string) bool {
	return status == TaskStatusSuccess || status == TaskStatusFailure
}

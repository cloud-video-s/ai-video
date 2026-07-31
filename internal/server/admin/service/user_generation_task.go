package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"path"
	"strings"
	"time"

	"ai-video/internal/generation"
	"ai-video/internal/repository"
)

type UserGenerationTaskService struct {
	repo *repository.UserGenerationTaskRepo
}

func NewUserGenerationTaskService() *UserGenerationTaskService {
	return &UserGenerationTaskService{repo: repository.NewUserGenerationTaskRepo()}
}

type ListUserGenerationTaskRequest struct {
	UserID    uint64 `form:"user_id"`
	ModelID   uint64 `form:"model_id"`
	MediaType uint32 `form:"media_type" binding:"omitempty,oneof=1 2"`
	Status    int    `form:"status" binding:"omitempty,oneof=1 2 3 4 5 6 7"`
	TaskCode  string `form:"task_code" binding:"max=50"`
	Keyword   string `form:"keyword" binding:"max=255"`
	DateFrom  string `form:"date_from" binding:"omitempty,datetime=2006-01-02"`
	DateTo    string `form:"date_to" binding:"omitempty,datetime=2006-01-02"`
}

type GenerationTaskUserView struct {
	ID           uint64 `json:"id"`
	Username     string `json:"username"`
	Email        string `json:"email"`
	LoginAccount string `json:"login_account"`
	IMEI         string `json:"imei"`
	DeviceCode   string `json:"device_code"`
}

type GenerationTaskModelView struct {
	ID        uint64 `json:"id"`
	Name      string `json:"name"`
	Code      string `json:"code"`
	ModelType uint32 `json:"model_type"`
	Version   string `json:"version"`
}

type UserGenerationTaskView struct {
	ID               uint64                   `json:"id"`
	UserID           uint64                   `json:"user_id"`
	ModelID          uint64                   `json:"model_id"`
	TemplateID       uint64                   `json:"template_id,omitempty"`
	ClientRequestID  string                   `json:"client_request_id"`
	TaskCode         string                   `json:"task_code"`
	ThirdTaskCode    string                   `json:"third_task_code"`
	Status           int                      `json:"status"`
	StatusName       string                   `json:"status_name"`
	Progress         uint32                   `json:"progress"`
	MediaType        string                   `json:"media_type"`
	Prompt           string                   `json:"prompt"`
	RequestPayload   interface{}              `json:"request_payload,omitempty"`
	ProviderResponse interface{}              `json:"provider_response,omitempty"`
	RemoteURLs       []string                 `json:"remote_urls"`
	LocalURLs        []string                 `json:"local_urls"`
	PreviewURLs      []string                 `json:"preview_urls"`
	ResultCount      int                      `json:"result_count"`
	ErrorMessage     string                   `json:"error_message"`
	UsageDuration    uint32                   `json:"usage_duration"`
	SubmittedAt      *time.Time               `json:"submitted_at"`
	StartedAt        *time.Time               `json:"started_at"`
	FinishedAt       *time.Time               `json:"finished_at"`
	LastPolledAt     *time.Time               `json:"last_polled_at"`
	CreatedAt        time.Time                `json:"created_at"`
	UpdatedAt        time.Time                `json:"updated_at"`
	User             *GenerationTaskUserView  `json:"user"`
	Model            *GenerationTaskModelView `json:"model"`
}

func (s *UserGenerationTaskService) List(ctx context.Context, page, pageSize int, req *ListUserGenerationTaskRequest) ([]UserGenerationTaskView, int64, error) {
	from, to, err := parseGenerationTaskDateRange(req.DateFrom, req.DateTo)
	if err != nil {
		return nil, 0, err
	}
	records, total, err := s.repo.PageAdmin(ctx, page, pageSize, &repository.UserGenerationTaskAdminFilter{
		UserID: req.UserID, ModelID: req.ModelID, ModelType: req.MediaType, Status: req.Status,
		TaskCode: strings.TrimSpace(req.TaskCode), Keyword: strings.TrimSpace(req.Keyword),
		CreatedFrom: from, CreatedTo: to,
	})
	if err != nil {
		return nil, 0, err
	}
	result := make([]UserGenerationTaskView, 0, len(records))
	for i := range records {
		result = append(result, generationTaskView(&records[i], false))
	}
	return result, total, nil
}

func (s *UserGenerationTaskService) GetByID(ctx context.Context, id uint64) (*UserGenerationTaskView, error) {
	record, err := s.repo.GetAdminDetail(ctx, id)
	if err != nil {
		return nil, notFoundOr(err, "生成任务不存在")
	}
	view := generationTaskView(record, true)
	return &view, nil
}

func generationTaskView(record *repository.UserGenerationTaskAdminRecord, detail bool) UserGenerationTaskView {
	task := record.Task
	remoteURLs := parseGenerationTaskURLs(task.RemoteUrls)
	localURLs := parseGenerationTaskURLs(task.LocalUrls)
	previewURLs := remoteURLs
	if len(localURLs) > 0 {
		previewURLs = localURLs
	}
	if previewURLs == nil {
		previewURLs = []string{}
	}

	view := UserGenerationTaskView{
		ID: task.ID, UserID: task.UserID, ModelID: task.ModelID, TemplateID: task.TemplateID,
		ClientRequestID: task.ClientRequestID, TaskCode: task.TaskCode, ThirdTaskCode: task.ThirdTaskCode,
		Status: task.Status, StatusName: generationTaskStatusName(task.Status), Progress: task.Progress,
		MediaType: generationTaskMediaType(record, previewURLs), Prompt: task.Prompt,
		RemoteURLs: remoteURLs, LocalURLs: localURLs, PreviewURLs: previewURLs, ResultCount: len(previewURLs),
		ErrorMessage: task.ErrorMessage, UsageDuration: task.UsageDuration,
		SubmittedAt: generationTaskTimePtr(task.SubmittedAt), StartedAt: generationTaskTimePtr(task.StartedAt),
		FinishedAt: generationTaskTimePtr(task.FinishedAt), LastPolledAt: generationTaskTimePtr(task.LastPolledAt),
		CreatedAt: task.CreatedAt, UpdatedAt: task.UpdatedAt,
	}
	if detail {
		view.RequestPayload = parseGenerationTaskJSON(task.RequestPayload)
		view.ProviderResponse = parseGenerationTaskJSON(task.ProviderResponse)
	}
	if record.User != nil {
		view.User = &GenerationTaskUserView{
			ID: record.User.ID, Username: record.User.Username, Email: record.User.Email,
			LoginAccount: record.User.LoginAccount, IMEI: record.User.IMEI, DeviceCode: record.User.DeviceCode,
		}
	}
	if record.Model != nil && record.Model.ID > 0 {
		view.Model = &GenerationTaskModelView{
			ID: uint64(record.Model.ID), Name: record.Model.Name, Code: record.Model.Code,
			ModelType: record.Model.ModelType, Version: record.Model.Version,
		}
	}
	return view
}

func generationTaskStatusName(status int) string {
	switch status {
	case generation.TaskStatusSubmitting:
		return "submitting"
	case generation.TaskStatusSubmitted:
		return "submitted"
	case generation.TaskStatusPending:
		return "pending"
	case generation.TaskStatusRunning:
		return "running"
	case generation.TaskStatusDownloading:
		return "downloading"
	case generation.TaskStatusSuccess:
		return "success"
	case generation.TaskStatusFailure:
		return "failure"
	default:
		return "unknown"
	}
}

func generationTaskMediaType(record *repository.UserGenerationTaskAdminRecord, previewURLs []string) string {
	if record.Model != nil {
		switch record.Model.ModelType {
		case 1:
			return "image"
		case 2:
			return "video"
		}
	}
	for _, rawURL := range previewURLs {
		parsed, err := url.Parse(rawURL)
		if err != nil {
			continue
		}
		switch strings.ToLower(path.Ext(parsed.Path)) {
		case ".mp4", ".webm", ".mov", ".m4v", ".avi", ".mkv":
			return "video"
		case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp", ".avif":
			return "image"
		}
	}
	return "unknown"
}

func parseGenerationTaskURLs(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []string{}
	}
	var values []string
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		var decoded string
		if json.Unmarshal([]byte(raw), &decoded) == nil {
			raw = decoded
		}
		values = strings.FieldsFunc(raw, func(r rune) bool {
			return r == ',' || r == ';' || r == '|' || r == '\n' || r == '\r'
		})
	}
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.Trim(strings.TrimSpace(value), "\"'")
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func parseGenerationTaskJSON(raw string) interface{} {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var value interface{}
	if json.Unmarshal([]byte(raw), &value) == nil {
		return value
	}
	return raw
}

func generationTaskTimePtr(value time.Time) *time.Time {
	if value.IsZero() {
		return nil
	}
	return &value
}

func parseGenerationTaskDateRange(fromValue, toValue string) (*time.Time, *time.Time, error) {
	var from, to *time.Time
	if fromValue != "" {
		value, err := time.ParseInLocation("2006-01-02", fromValue, time.Local)
		if err != nil {
			return nil, nil, errors.New("开始日期格式错误")
		}
		from = &value
	}
	if toValue != "" {
		value, err := time.ParseInLocation("2006-01-02", toValue, time.Local)
		if err != nil {
			return nil, nil, errors.New("结束日期格式错误")
		}
		value = value.AddDate(0, 0, 1)
		to = &value
	}
	if from != nil && to != nil && !from.Before(*to) {
		return nil, nil, errors.New("开始日期不能晚于结束日期")
	}
	return from, to, nil
}

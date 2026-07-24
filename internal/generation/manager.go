package generation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"ai-video/internal/config"
	"ai-video/internal/gen/model"
	"ai-video/internal/repository"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var requestIDPattern = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

// Manager 负责创建任务、轮询第三方状态、保存本地文件并发布进度事件。
type Manager struct {
	modelRepo *repository.AIModelRepo
	taskRepo  *repository.GenerationTaskRepo
	hub       *Hub

	mu     sync.Mutex
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

var sharedManager = &Manager{
	modelRepo: repository.NewAIModelRepo(),
	taskRepo:  repository.NewGenerationTaskRepo(),
	hub:       NewHub(),
}

func Shared() *Manager { return sharedManager }

// Start 启动可恢复的任务轮询器，重复调用不会启动多个 worker。
func Start() { sharedManager.start() }

func Stop() { sharedManager.stop() }

func (m *Manager) start() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.cancel != nil {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel
	m.wg.Add(1)
	go m.worker(ctx)
}

func (m *Manager) stop() {
	m.mu.Lock()
	cancel := m.cancel
	m.cancel = nil
	m.mu.Unlock()
	if cancel != nil {
		cancel()
		m.wg.Wait()
	}
}

func (m *Manager) Subscribe(taskID uint64) (<-chan TaskView, func()) {
	return m.hub.Subscribe(taskID)
}

func (m *Manager) CreateTask(ctx context.Context, userID uint64, request *CreateTaskRequest) (*model.VideoGenerationTask, error) {
	if userID == 0 {
		return nil, errors.New("用户 ID 无效")
	}
	request.ModelCode = strings.TrimSpace(request.ModelCode)
	request.ClientRequestID = strings.TrimSpace(request.ClientRequestID)
	if request.ClientRequestID == "" {
		request.ClientRequestID = uuid.NewString()
	} else if !requestIDPattern.MatchString(request.ClientRequestID) {
		return nil, errors.New("client_request_id 只能包含字母、数字、点、下划线和中划线")
	}
	if len(request.Input) == 0 {
		return nil, errors.New("input 不能为空")
	}
	if existing, err := m.taskRepo.GetByClientRequestID(ctx, userID, request.ClientRequestID); err == nil {
		return existing, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	modelConfig, err := m.modelRepo.GetEnabledByCode(ctx, request.ModelCode)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("模型不存在或未启用")
		}
		return nil, err
	}
	if strings.TrimSpace(modelConfig.APIKey) == "" {
		return nil, errors.New("模型尚未配置 API Key")
	}
	parameters, err := mergeParameters(modelConfig.DefaultParameters, request.Parameters)
	if err != nil {
		return nil, err
	}
	if _, exists := parameters["external_task_id"]; !exists {
		parameters["external_task_id"] = request.ClientRequestID
	}
	remoteRequest := remoteSubmitRequest{
		Model: modelConfig.ModelName, Input: cloneMap(request.Input), Parameters: parameters,
	}
	payload, err := json.Marshal(remoteRequest)
	if err != nil {
		return nil, err
	}
	prompt, _ := request.Input["prompt"].(string)
	task := &model.VideoGenerationTask{
		UserID: userID, ModelConfigID: modelConfig.ID, ClientRequestID: request.ClientRequestID,
		Status: TaskStatusSubmitting, Progress: 0, Prompt: prompt, RequestPayload: string(payload),
		RemoteUrls: "[]", LocalUrls: "[]",
	}
	if err := m.taskRepo.Create(ctx, task); err != nil {
		if existing, lookupErr := m.taskRepo.GetByClientRequestID(ctx, userID, request.ClientRequestID); lookupErr == nil {
			return existing, nil
		}
		return nil, err
	}
	if task.ID == 0 {
		id, lookupErr := m.taskRepo.IDByClientRequestID(ctx, userID, request.ClientRequestID)
		if lookupErr != nil || id == 0 {
			return nil, errors.New("生成任务已创建但未能读取任务 ID")
		}
		task.ID = id
	}
	m.hub.Publish(task)

	provider, err := providerFor(modelConfig.Provider)
	if err != nil {
		_ = m.failTask(ctx, task, err.Error())
		return task, err
	}
	result, err := provider.Submit(ctx, modelConfig, remoteRequest)
	if err != nil {
		_ = m.failTask(ctx, task, err.Error())
		return task, err
	}
	now := time.Now()
	task.ExternalTaskID = result.TaskID
	task.ProviderResponse = result.RawResponse
	task.Status = TaskStatusSubmitted
	task.Progress = 5
	task.SubmittedAt = now
	task.ErrorMessage = ""
	if err := m.taskRepo.UpdateFields(ctx, task,
		"ExternalTaskID", "ProviderResponse", "Status", "Progress", "SubmittedAt", "ErrorMessage",
	); err != nil {
		return nil, err
	}
	m.hub.Publish(task)
	return task, nil
}

func (m *Manager) GetTask(ctx context.Context, userID, taskID uint64) (*model.VideoGenerationTask, error) {
	return m.taskRepo.GetOwned(ctx, taskID, userID)
}

func (m *Manager) ListTasks(ctx context.Context, userID uint64, page, pageSize int, status string) ([]model.VideoGenerationTask, int64, error) {
	status = strings.ToLower(strings.TrimSpace(status))
	if status != "" && !validTaskStatus(status) {
		return nil, 0, errors.New("任务状态无效")
	}
	return m.taskRepo.PageOwned(ctx, userID, page, pageSize, status)
}

func (m *Manager) ListModels(ctx context.Context) ([]model.VideoAiModel, error) {
	return m.modelRepo.ListEnabled(ctx)
}

func (m *Manager) DeleteTask(ctx context.Context, userID, taskID uint64) error {
	task, err := m.taskRepo.GetOwned(ctx, taskID, userID)
	if err != nil {
		return err
	}
	if !IsTerminal(task.Status) {
		return errors.New("进行中的任务不能删除")
	}
	return m.taskRepo.DeleteOwned(ctx, taskID, userID)
}

func (m *Manager) worker(ctx context.Context) {
	defer m.wg.Done()
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.pollBatch(ctx)
		}
	}
}

func (m *Manager) pollBatch(ctx context.Context) {
	tasks, err := m.taskRepo.ListActive(ctx, 100)
	if err != nil {
		config.Log.Warnf("list generation tasks: %v", err)
		return
	}
	for i := range tasks {
		if ctx.Err() != nil {
			return
		}
		if err := m.processTask(ctx, &tasks[i]); err != nil {
			config.Log.Warnf("process generation task %d: %v", tasks[i].ID, err)
		}
	}
}

func (m *Manager) processTask(ctx context.Context, task *model.VideoGenerationTask) error {
	modelConfig, err := m.modelRepo.GetByID(ctx, uint(task.ModelConfigID))
	if err != nil {
		return m.failTask(ctx, task, "模型配置不存在")
	}
	if task.Status == TaskStatusDownloading {
		var urls []string
		if err := json.Unmarshal([]byte(task.RemoteUrls), &urls); err != nil || len(urls) == 0 {
			return m.failTask(ctx, task, "远程结果 URL 无效")
		}
		return m.downloadAndFinish(ctx, task, urls)
	}
	pollInterval := time.Duration(modelConfig.PollIntervalSeconds) * time.Second
	if pollInterval <= 0 {
		pollInterval = 3 * time.Second
	}
	if task.LastPolledAt.IsZero() && time.Since(task.LastPolledAt) < pollInterval {
		return nil
	}
	timeout := time.Duration(modelConfig.TaskTimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Minute
	}
	started := task.CreatedAt
	if task.SubmittedAt.IsZero() {
		started = task.SubmittedAt
	}
	if time.Since(started) > timeout {
		return m.failTask(ctx, task, "生成任务超时")
	}
	now := time.Now()
	if err := m.taskRepo.MarkPolling(ctx, task.ID, now); err != nil {
		return err
	}
	task.LastPolledAt = now
	provider, err := providerFor(modelConfig.Provider)
	if err != nil {
		return m.failTask(ctx, task, err.Error())
	}
	status, err := provider.Status(ctx, modelConfig, task.ExternalTaskID)
	if err != nil {
		task.ErrorMessage = "轮询失败，将自动重试: " + err.Error()
		if updateErr := m.taskRepo.UpdateFields(ctx, task, "ErrorMessage", "LastPolledAt"); updateErr != nil {
			return updateErr
		}
		m.hub.Publish(task)
		return err
	}
	task.ProviderResponse = status.RawResponse
	task.UsageDuration = status.UsageDuration
	task.ErrorMessage = ""
	switch strings.ToLower(status.Status) {
	case "pending":
		task.Status, task.Progress = TaskStatusPending, 10
	case "running":
		task.Status, task.Progress = TaskStatusRunning, 50
		if task.StartedAt.IsZero() {
			task.StartedAt = now
		}
	case "success":
		if len(status.URLs) == 0 {
			return m.failTask(ctx, task, "上游任务成功但未返回视频 URL")
		}
		encodedURLs, _ := json.Marshal(status.URLs)
		task.RemoteUrls = string(encodedURLs)
		task.Status, task.Progress = TaskStatusDownloading, 90
		if status.FinishTime > 0 {
			finishedAt := time.Unix(status.FinishTime, 0)
			task.FinishedAt = finishedAt
		}
		if err := m.taskRepo.UpdateFields(ctx, task,
			"ProviderResponse", "UsageDuration", "ErrorMessage", "RemoteUrls", "Status", "Progress", "FinishedAt", "LastPolledAt",
		); err != nil {
			return err
		}
		m.hub.Publish(task)
		return m.downloadAndFinish(ctx, task, status.URLs)
	case "failure":
		message := strings.TrimSpace(status.ErrorMessage)
		if message == "" {
			message = "上游生成任务失败"
		}
		return m.failTask(ctx, task, message)
	default:
		return fmt.Errorf("未知上游任务状态: %s", status.Status)
	}
	if err := m.taskRepo.UpdateFields(ctx, task,
		"ProviderResponse", "UsageDuration", "ErrorMessage", "Status", "Progress", "StartedAt", "LastPolledAt",
	); err != nil {
		return err
	}
	m.hub.Publish(task)
	return nil
}

func (m *Manager) downloadAndFinish(ctx context.Context, task *model.VideoGenerationTask, remoteURLs []string) error {
	localURLs, err := downloadVideos(ctx, task, remoteURLs)
	if err != nil {
		return m.failTask(ctx, task, "保存生成视频失败: "+err.Error())
	}
	encoded, _ := json.Marshal(localURLs)
	now := time.Now()
	task.LocalUrls = string(encoded)
	task.Status = TaskStatusSuccess
	task.Progress = 100
	task.ErrorMessage = ""
	task.FinishedAt = now
	if err := m.taskRepo.UpdateFields(ctx, task, "LocalUrls", "Status", "Progress", "ErrorMessage", "FinishedAt"); err != nil {
		return err
	}
	m.hub.Publish(task)
	return nil
}

func (m *Manager) failTask(ctx context.Context, task *model.VideoGenerationTask, message string) error {
	now := time.Now()
	task.Status = TaskStatusFailure
	task.Progress = 100
	task.ErrorMessage = strings.TrimSpace(message)
	task.FinishedAt = now
	err := m.taskRepo.UpdateFields(ctx, task, "Status", "Progress", "ErrorMessage", "FinishedAt")
	m.hub.Publish(task)
	if err != nil {
		return err
	}
	return errors.New(task.ErrorMessage)
}

func providerFor(name string) (Provider, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "modelverse":
		return &ModelVerseProvider{}, nil
	default:
		return nil, fmt.Errorf("不支持的模型 provider: %s", name)
	}
}

func mergeParameters(defaultJSON string, request map[string]interface{}) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	if strings.TrimSpace(defaultJSON) != "" {
		if err := json.Unmarshal([]byte(defaultJSON), &result); err != nil {
			return nil, errors.New("模型默认参数不是有效 JSON 对象")
		}
	}
	for key, value := range request {
		result[key] = value
	}
	return result, nil
}

func cloneMap(source map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{}, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}

func validTaskStatus(status string) bool {
	switch status {
	case TaskStatusSubmitting, TaskStatusSubmitted, TaskStatusPending, TaskStatusRunning,
		TaskStatusDownloading, TaskStatusSuccess, TaskStatusFailure:
		return true
	default:
		return false
	}
}

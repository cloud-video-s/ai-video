package generation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"regexp"
	"strconv"
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

const submitClaimLease = 2 * time.Minute

// Manager 负责创建任务、轮询第三方状态、持久化生成结果并发布进度事件。
type Manager struct {
	modelRepo     *repository.ModelRepo
	parameterRepo *repository.ModelParameterRepo
	taskRepo      *repository.UserGenerationTaskRepo
	hub           *Hub

	mu     sync.Mutex
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

var sharedManager = &Manager{
	modelRepo:     repository.NewModelRepo(),
	parameterRepo: repository.NewModelParameterRepo(),
	taskRepo:      repository.NewUserGenerationTaskRepo(),
	hub:           NewHub(),
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

func (m *Manager) CreateTask(ctx context.Context, userID uint64, request *CreateTaskRequest) (*model.VideoUserGenerationTask, error) {
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
	if _, err := generationInputFromMap(request.TaskType, request.Input); err != nil {
		return nil, err
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
	if modelConfig.ModelType != request.TaskType {
		return nil, fmt.Errorf("task_type %d does not match model %s type %d", request.TaskType, modelConfig.Code, modelConfig.ModelType)
	}
	parameterDefinitions, err := m.parameterRepo.ListByModel(ctx, modelConfig.ID)
	if err != nil {
		return nil, err
	}
	parameters, err := mergeModelParameters(parameterDefinitions, request.Parameters)
	if err != nil {
		return nil, err
	}
	taskCode := uuid.NewString()
	if request.TaskType == TaskTypeVideo {
		// The upstream idempotency key is owned by us and must be persisted in
		// the queued payload before CreateTask returns. This makes submission
		// recoverable after a restart and prevents clients from overriding it.
		parameters["external_task_id"] = taskCode
	}
	remoteRequest := remoteSubmitRequest{
		Model: modelConfig.Code, Input: cloneMap(request.Input), Parameters: parameters,
	}
	payload, err := json.Marshal(remoteRequest)
	if err != nil {
		return nil, err
	}
	prompt, _ := request.Input["prompt"].(string)
	task := &model.VideoUserGenerationTask{
		UserID: userID, ModelID: uint64(modelConfig.ID), ClientRequestID: request.ClientRequestID, TaskCode: taskCode,
		TaskType: request.TaskType, Status: TaskStatusSubmitting, Progress: 0, Prompt: prompt, RequestPayload: string(payload),
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
	return task, nil
}

func (m *Manager) submitTask(ctx context.Context, task *model.VideoUserGenerationTask, modelConfig *model.VideoModel, request remoteSubmitRequest) error {
	provider := &ModelVerseProvider{}
	result, err := provider.Submit(ctx, modelConfig, request)
	if err != nil {
		return err
	}
	if result.Completed {
		encodedURLs, _ := json.Marshal(result.URLs)
		task.ThirdTaskCode = result.TaskID
		task.ProviderResponse = result.RawResponse
		task.RemoteUrls = string(encodedURLs)
		task.Status = TaskStatusDownloading
		task.Progress = 90
		task.SubmittedAt = time.Now()
		task.ErrorMessage = ""
		if task.ThirdTaskCode == "" {
			if err := m.taskRepo.UpdateFields(ctx, task,
				"ProviderResponse", "RemoteUrls", "Status", "Progress", "SubmittedAt", "ErrorMessage",
			); err != nil {
				return err
			}
		} else {
			if err := m.taskRepo.UpdateFields(ctx, task,
				"ThirdTaskCode", "ProviderResponse", "RemoteUrls", "Status", "Progress", "SubmittedAt", "ErrorMessage",
			); err != nil {
				return err
			}
		}

		m.hub.Publish(task)
		return m.finishImageTask(ctx, task, result.URLs, result.Base64Images)
	}
	task.ThirdTaskCode = result.TaskID
	task.ProviderResponse = result.RawResponse
	task.Status = TaskStatusSubmitted
	task.Progress = 5
	task.SubmittedAt = time.Now()
	task.ErrorMessage = ""
	if err := m.taskRepo.UpdateFields(ctx, task,
		"ThirdTaskCode", "ProviderResponse", "Status", "Progress", "SubmittedAt", "ErrorMessage",
	); err != nil {
		return err
	}
	m.hub.Publish(task)
	return nil
}

func (m *Manager) GetTask(ctx context.Context, userID, taskID uint64) (*model.VideoUserGenerationTask, error) {
	return m.taskRepo.GetOwned(ctx, taskID, userID)
}

func (m *Manager) ListTasks(ctx context.Context, userID uint64, page, pageSize int, status string) ([]model.VideoUserGenerationTask, int64, error) {
	status = strings.TrimSpace(status)
	if status == "" {
		return m.taskRepo.PageOwned(ctx, userID, page, pageSize, 0)
	}
	statusValue, err := strconv.Atoi(status)
	if err != nil || !validTaskStatus(statusValue) {
		return nil, 0, errors.New("任务状态无效")
	}
	return m.taskRepo.PageOwned(ctx, userID, page, pageSize, statusValue)
}

func (m *Manager) ListModels(ctx context.Context) ([]model.VideoModel, error) {
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
	tasks, err := m.taskRepo.ListActive(ctx, 100,
		TaskStatusSubmitting, TaskStatusSubmitted, TaskStatusPending, TaskStatusRunning, TaskStatusDownloading,
	)
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

func (m *Manager) processTask(ctx context.Context, task *model.VideoUserGenerationTask) error {
	modelConfig, err := m.modelRepo.GetByIDWithPlatform(ctx, int64(task.ModelID))
	if err != nil {
		return m.failTask(ctx, task, "模型配置不存在")
	}
	if task.Status == TaskStatusSubmitting {
		claimedAt := time.Now()
		claimed, err := m.taskRepo.TryClaimSubmitting(
			ctx, task.ID, TaskStatusSubmitting, claimedAt, claimedAt.Add(-submitClaimLease),
		)
		if err != nil {
			return err
		}
		if !claimed {
			return nil
		}
		task.LastPolledAt = claimedAt
		var request remoteSubmitRequest
		if err := json.Unmarshal([]byte(task.RequestPayload), &request); err != nil {
			return m.failTask(ctx, task, "生成任务请求数据无效")
		}
		if err := m.submitTask(ctx, task, modelConfig, request); err != nil {
			return m.failTask(ctx, task, err.Error())
		}
		return nil
	}
	if task.Status == TaskStatusDownloading && modelConfig.ModelType == TaskTypeImage {
		urls, encoded, err := imageResultPayloads(task.ProviderResponse)
		if err != nil {
			return m.failTask(ctx, task, err.Error())
		}
		return m.finishImageTask(ctx, task, urls, encoded)
		//if err := json.Unmarshal([]byte(task.RemoteUrls), &urls); err != nil || len(urls) == 0 {
		//	return m.failTask(ctx, task, "远程结果 URL 无效")
		//}
		//return m.downloadAndFinish(ctx, task, urls)
	}
	const pollInterval = 3 * time.Second
	if !task.LastPolledAt.IsZero() && time.Since(task.LastPolledAt) < pollInterval {
		return nil
	}
	thirdTaskCode := strings.TrimSpace(task.ThirdTaskCode)
	if thirdTaskCode == "" {
		return m.failTask(ctx, task, "已提交任务缺少 third_task_code")
	}
	now := time.Now()
	claimed, err := m.taskRepo.TryClaimPolling(
		ctx, task.ID, task.Status, now, now.Add(-pollInterval),
	)
	if err != nil {
		return err
	}
	if !claimed {
		return nil
	}
	task.LastPolledAt = now
	provider := &ModelVerseProvider{}
	status, err := provider.Status(ctx, modelConfig, thirdTaskCode)
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

func (m *Manager) downloadAndFinish(ctx context.Context, task *model.VideoUserGenerationTask, remoteURLs []string) error {
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

func (m *Manager) failTask(ctx context.Context, task *model.VideoUserGenerationTask, message string) error {
	now := time.Now()
	task.Status = TaskStatusFailure
	task.Progress = 100
	task.ErrorMessage = strings.TrimSpace(message)
	task.FinishedAt = now
	err := m.taskRepo.UpdateFields(
		ctx, task,
		"Status", "Progress", "ErrorMessage", "FinishedAt", "ProviderResponse", "UsageDuration", "LastPolledAt",
	)
	m.hub.Publish(task)
	if err != nil {
		return err
	}
	return errors.New(task.ErrorMessage)
}

func providerFor(name string) (Provider, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "modelverse", "ucloud":
		return &ModelVerseProvider{}, nil
	default:
		return nil, fmt.Errorf("不支持的模型 provider: %s", name)
	}
}

func mergeModelParameters(definitions []model.VideoModelParameter, request map[string]any) (map[string]any, error) {
	return mergeConfiguredParameters(definitions, request)
}

func mergeLegacyModelParameters(definitions []model.VideoModelParameter, request map[string]any) (map[string]any, error) {
	result := make(map[string]interface{}, len(definitions)+len(request))
	for i := range definitions {
		key := strings.TrimSpace(definitions[i].ParamKey)
		raw := strings.TrimSpace(definitions[i].DefaultValue)
		if key == "" || raw == "" {
			continue
		}
		var value interface{}
		if err := json.Unmarshal([]byte(raw), &value); err != nil {
			return nil, fmt.Errorf("模型参数 %s 的默认值不是有效 JSON: %w", key, err)
		}
		result[key] = value
	}
	for key, value := range request {
		result[key] = value
	}
	return result, nil
}

func cloneMap(source map[string]any) map[string]any {
	result := make(map[string]any, len(source))
	maps.Copy(result, source)
	return result
}

func validTaskStatus(status int) bool {
	switch status {
	case TaskStatusSubmitting, TaskStatusSubmitted, TaskStatusPending, TaskStatusRunning,
		TaskStatusDownloading, TaskStatusSuccess, TaskStatusFailure:
		return true
	default:
		return false
	}
}

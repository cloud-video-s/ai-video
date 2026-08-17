package generation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"math"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"ai-video/internal/config"
	"ai-video/internal/gen/model"
	"ai-video/internal/pkg/tracing"
	"ai-video/internal/pkg/ucloud"
	"ai-video/internal/pkg/upload"
	"ai-video/internal/pkg/uploadruntime"
	"ai-video/internal/repository"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var requestIDPattern = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

const submitClaimLease = 2 * time.Minute

// Manager 负责创建任务、轮询第三方状态、持久化生成结果并发布进度事件。
type Manager struct {
	modelRepo *repository.ModelRepo
	*repository.AppUserRepo
	parameterRepo         *repository.ModelParameterRepo
	taskRepo              *repository.UserGenerationTaskRepo
	uploadRepo            *repository.UploadRepo
	templateRepo          *repository.TemplateRepo
	templateParameterRepo *repository.TemplateModelParameterRepo
	hub                   *Hub

	downloadMu sync.Mutex
	downloads  *generatedDownloadController

	mu     sync.Mutex
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

var sharedManager = &Manager{
	AppUserRepo:           repository.NewAppUserRepo(),
	modelRepo:             repository.NewModelRepo(),
	parameterRepo:         repository.NewModelParameterRepo(),
	taskRepo:              repository.NewUserGenerationTaskRepo(),
	uploadRepo:            repository.NewUploadRepo(),
	templateRepo:          repository.NewTemplateRepo(),
	templateParameterRepo: repository.NewTemplateModelParameterRepo(),
	hub:                   NewHub(),
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
	generationInput, err := generationInputFromMap(request.TaskType, request.Input)
	if err != nil {
		return nil, err
	}
	if len(request.Input) == 0 {
		return nil, errors.New("input 不能为空")
	}
	if existing, err := m.taskRepo.GetByClientRequestID(ctx, userID, request.ClientRequestID); err == nil {
		if err := validateIdempotentTemplate(existing, request.TemplateID); err != nil {
			return nil, err
		}
		return existing, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	uploadOwner := upload.UploadOwner{Type: upload.UploaderAPIUser, ID: userID}
	inputFileURLs := generationInputFileURLs(generationInput)
	if m.uploadRepo != nil && len(inputFileURLs) > 0 {
		ownedURLs, err := m.uploadRepo.OwnedHalfURLs(ctx, uploadOwner, inputFileURLs)
		if err != nil {
			return nil, err
		}
		normalizeOwnedGenerationInput(request.Input, generationInput, ownedURLs)
		generationInput, err = generationInputFromMap(request.TaskType, request.Input)
		if err != nil {
			return nil, err
		}
		inputFileURLs = generationInputFileURLs(generationInput)
	}

	modelConfig, err := m.modelRepo.GetEnabledByCode(ctx, request.ModelCode)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("模型不存在或未启用")
		}
		return nil, err
	}
	if modelConfig.Score < 0 || modelConfig.Score > math.MaxUint32 {
		return nil, errors.New("model score is outside the supported range")
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
		if modelConfig.Code == ucloud.ModelKlingO3 {
			if video, ok := request.Input["video"].(string); ok && video != "" {
				parameters["sound"] = "off"
			}
		}

		if modelConfig.Code == ucloud.ModelKlingV3 {
			duration, durationOK := parameters["duration"].(int)
			klingV3Type, ok := parameters["kling_v3_type"].(string)
			if ok && klingV3Type == "motion_control" {
				if durationOK && (duration < 5 || duration > 10) {
					return nil, errors.New("the value of duration ranges from 5 to 10 seconds")
				}
			}
		}
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
		UserID: userID, ModelID: uint64(modelConfig.ID), ClientRequestID: request.ClientRequestID, TaskCode: taskCode, Score: uint32(modelConfig.Score),
		TemplateID: request.TemplateID, TaskType: request.TaskType, Status: TaskStatusSubmitting, Progress: 0, Prompt: prompt, RequestPayload: string(payload),
		RemoteUrls: "[]", LocalUrls: "[]",
	}
	if err = repository.Transaction(ctx, func(txCtx context.Context) error {
		user, err := m.GetByIDForUpdate(txCtx, userID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("user does not exist")
			}
			return err
		}
		allocation, err := freezeTaskPoints(user, task.Score)
		if err != nil {
			return err
		}
		scoreType := allocation.scoreType()
		task.ScoreType = scoreType
		task.VipScore = allocation.VIPScore
		task.PointsScore = allocation.PointsScore
		if err = m.taskRepo.Create(txCtx, task); err != nil {
			return err
		}
		if err = m.Update(txCtx, user.ID, map[string]any{
			"vip_points":     gorm.Expr("vip_points - ?", allocation.VIPScore),
			"points_balance": gorm.Expr("points_balance - ?", allocation.PointsScore),
			"frozen_points":  gorm.Expr("frozen_points + ?", task.Score),
		}); err != nil {
			return err
		}
		if m.uploadRepo != nil {
			return m.uploadRepo.ConfirmUploadedByURLs(txCtx, uploadOwner, inputFileURLs)
		}
		return nil
	}); err != nil {
		if existing, lookupErr := m.taskRepo.GetByClientRequestID(ctx, userID, request.ClientRequestID); lookupErr == nil {
			if err := validateIdempotentTemplate(existing, request.TemplateID); err != nil {
				return nil, err
			}
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

func generationInputFileURLs(input GenerationInput) []string {
	result := make([]string, 0, len(input.Images)+3)
	result = append(result, input.Images...)
	if input.Video != "" {
		result = append(result, input.Video)
	}
	if input.FirstFrame != "" {
		result = append(result, input.FirstFrame)
	}
	if input.EndFrame != "" {
		result = append(result, input.EndFrame)
	}
	return result
}

func normalizeOwnedGenerationInput(source map[string]any, input GenerationInput, owned map[string]struct{}) {
	normalize := func(value string) string {
		half := upload.HalfURL(value)
		if _, exists := owned[half]; exists {
			return half
		}
		return value
	}
	if len(input.Images) > 0 {
		images := make([]string, len(input.Images))
		for i := range input.Images {
			images[i] = normalize(input.Images[i])
		}
		source["images"] = images
	}
	if input.Video != "" {
		source["video"] = normalize(input.Video)
	}
	if input.FirstFrame != "" {
		source["first_frame"] = normalize(input.FirstFrame)
	}
	if input.EndFrame != "" {
		source["end_frame"] = normalize(input.EndFrame)
	}
}

func validateIdempotentTemplate(task *model.VideoUserGenerationTask, templateID uint64) error {
	if task != nil && task.TemplateID != templateID {
		return errors.New("client_request_id is already used by a task with a different template")
	}
	return nil
}

func (m *Manager) submitTask(ctx context.Context, task *model.VideoUserGenerationTask, modelConfig *model.VideoModel, request remoteSubmitRequest) error {
	provider := &ModelVerseProvider{}
	providerRequest := request
	providerRequest.Input = providerGenerationInput(request.Input)
	result, err := provider.Submit(ctx, modelConfig, providerRequest)
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
			if err = m.taskRepo.UpdateFields(ctx, task,
				"ProviderResponse", "RemoteUrls", "Status", "Progress", "SubmittedAt", "ErrorMessage",
			); err != nil {
				return err
			}
		} else {
			if err = m.taskRepo.UpdateFields(ctx, task,
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

func providerGenerationInput(source map[string]any) map[string]any {
	result := cloneMap(source)
	input, err := generationInputFromMap(TaskTypeVideo, source)
	if err != nil {
		input, err = generationInputFromMap(TaskTypeImage, source)
	}
	if err != nil {
		return result
	}
	expand := uploadruntime.PublicURL
	if len(input.Images) > 0 {
		images := make([]string, len(input.Images))
		for i := range input.Images {
			images[i] = expand(input.Images[i])
		}
		result["images"] = images
	}
	if input.Video != "" {
		result["video"] = expand(input.Video)
	}
	if input.FirstFrame != "" {
		result["first_frame"] = expand(input.FirstFrame)
	}
	if input.EndFrame != "" {
		result["end_frame"] = expand(input.EndFrame)
	}
	return result
}

func (m *Manager) GetTask(ctx context.Context, userID, taskID uint64) (*model.VideoUserGenerationTask, error) {
	return m.taskRepo.GetOwned(ctx, taskID, userID)
}

func (m *Manager) GetOngoingTask(ctx context.Context, userID uint64) ([]*model.VideoUserGenerationTask, error) {
	return m.taskRepo.OngoingTask(ctx, userID)
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
	batchCtx, _ := tracing.NewContext(ctx)
	tasks, err := m.taskRepo.ListActive(batchCtx, 100,
		TaskStatusSubmitting, TaskStatusSubmitted, TaskStatusPending, TaskStatusRunning, TaskStatusDownloading,
	)
	if err != nil {
		config.Logger(batchCtx).Warnf("list generation tasks: %v", err)
		return
	}
	for i := range tasks {
		if ctx.Err() != nil {
			return
		}
		taskCtx, _ := tracing.NewContext(ctx)
		if err = m.processTask(taskCtx, &tasks[i]); err != nil {
			config.Logger(taskCtx).Warnf("process generation task %d: %v", tasks[i].ID, err)
		}
	}
}

func (m *Manager) processTask(ctx context.Context, task *model.VideoUserGenerationTask) error {
	modelConfig, err := m.modelRepo.GetByIDWithPlatform(ctx, int64(task.ModelID))
	if err != nil {
		return m.failTask(ctx, task, "模型配置不存在")
	}

	switch task.Status {
	case TaskStatusSubmitting:
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
		if err = json.Unmarshal([]byte(task.RequestPayload), &request); err != nil {
			return m.failTask(ctx, task, "生成任务请求数据无效")
		}
		if err = m.submitTask(ctx, task, modelConfig, request); err != nil {
			return m.failTask(ctx, task, err.Error())
		}
		return nil

	case TaskStatusDownloading:
		//claimed, err := m.tryClaimTaskProcessing(ctx, task)
		//if err != nil || !claimed {
		//	return err
		//}
		switch modelConfig.ModelType {
		case TaskTypeImage:
			urls, encoded, err := imageResultPayloads(task.ProviderResponse)
			if err != nil {
				return m.failTask(ctx, task, err.Error())
			}
			return m.finishImageTaskOrFail(ctx, task, urls, encoded)
		case TaskTypeVideo:
			return m.processVideoTaskResult(ctx, task)
		default:
			return m.failTask(ctx, task, fmt.Sprintf("unsupported generation task type: %d", modelConfig.ModelType))
		}
	default:
		thirdTaskCode := strings.TrimSpace(task.ThirdTaskCode)
		if thirdTaskCode == "" {
			return m.failTask(ctx, task, "已提交任务缺少 third_task_code")
		}
		//claimed, err := m.tryClaimTaskProcessing(ctx, task)
		//if err != nil || !claimed {
		//	return err
		//}
		now := task.LastPolledAt
		provider := &ModelVerseProvider{}
		status, err := provider.Status(ctx, modelConfig, thirdTaskCode)
		if err != nil {
			task.ErrorMessage = "轮询失败，将自动重试: " + err.Error()
			if updateErr := m.taskRepo.UpdateFields(ctx, task, "ErrorMessage", "LastPolledAt"); updateErr != nil {
				return updateErr
			}
			return m.failTask(ctx, task, task.ErrorMessage)
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
			if err = m.taskRepo.UpdateFields(ctx, task,
				"ProviderResponse", "UsageDuration", "ErrorMessage", "RemoteUrls", "Status", "Progress", "LastPolledAt",
			); err != nil {
				return err
			}
			m.hub.Publish(task)
			// Downloading the video and generating its cover are recoverable local
			// stages. Leave them to later worker passes so the provider polling
			// request only records the remote result.
			if modelConfig.ModelType == TaskTypeVideo {
				return nil
			}
			return m.finishImageTaskOrFail(ctx, task, status.URLs, nil)
		case "failure":
			message := strings.TrimSpace(status.ErrorMessage)
			if message == "" {
				message = "上游生成任务失败"
			}
			return m.failTask(ctx, task, message)
		default:
			return fmt.Errorf("未知上游任务状态: %s", status.Status)
		}
		if err = m.taskRepo.UpdateFields(ctx, task,
			"ProviderResponse", "UsageDuration", "ErrorMessage", "Status", "Progress", "StartedAt", "LastPolledAt",
		); err != nil {
			return err
		}
		m.hub.Publish(task)
	}

	return nil
}

func (m *Manager) tryClaimTaskProcessing(ctx context.Context, task *model.VideoUserGenerationTask) (bool, error) {
	const pollInterval = 10 * time.Second
	if !task.LastPolledAt.IsZero() && time.Since(task.LastPolledAt) < pollInterval {
		return false, nil
	}
	//claimed, err := m.taskRepo.TryClaimPolling(
	//	ctx, task.ID, task.Status, now, now.Add(-pollInterval),
	//)
	//if err != nil || !claimed {
	//	return claimed, err
	//}
	task.LastPolledAt = time.Now()
	return true, nil
}

func (m *Manager) failTask(ctx context.Context, task *model.VideoUserGenerationTask, message string) error {
	now := time.Now()
	failed := *task
	failed.Status = TaskStatusFailure
	failed.Progress = 100
	failed.ErrorMessage = strings.TrimSpace(message)
	if config.Log != nil {
		config.Logger(ctx).Errorw("generation task failed",
			"task_id", failed.ID,
			"task_code", failed.TaskCode,
			"user_id", failed.UserID,
			"original_error", failed.ErrorMessage,
		)
	}
	failed.FinishedAt = now
	alreadyFailed := false
	err := repository.Transaction(ctx, func(txCtx context.Context) error {
		state, err := m.taskRepo.GetPointStateForUpdate(txCtx, failed.ID)
		if err != nil {
			return err
		}
		switch state.Status {
		case TaskStatusSuccess:
			return errors.New("successful generation task cannot be failed")
		case TaskStatusFailure:
			alreadyFailed = true
			return nil
		}
		if err = m.releaseFrozenTaskPoints(txCtx, state); err != nil {
			return err
		}
		return m.taskRepo.UpdateFields(
			txCtx, &failed,
			"Status", "Progress", "ErrorMessage", "FinishedAt", "ProviderResponse", "UsageDuration", "LastPolledAt",
		)
	})
	if err != nil {
		return err
	}
	if !alreadyFailed {
		*task = failed
	} else {
		task.Status = TaskStatusFailure
		task.Progress = 100
	}
	m.hub.Publish(task)
	return errors.New(failed.ErrorMessage)
}

func mergeModelParameters(definitions []model.VideoModelParameter, request map[string]any) (map[string]any, error) {
	return mergeConfiguredParameters(definitions, request)
}

func mergeLegacyModelParameters(definitions []model.VideoModelParameter, request map[string]any) (map[string]any, error) {
	result := make(map[string]any, len(definitions)+len(request))
	for i := range definitions {
		key := strings.TrimSpace(definitions[i].ParamKey)
		raw := strings.TrimSpace(definitions[i].DefaultValue)
		if key == "" || raw == "" {
			continue
		}
		var value any
		if err := json.Unmarshal([]byte(raw), &value); err != nil {
			return nil, fmt.Errorf("模型参数 %s 的默认值不是有效 JSON: %w", key, err)
		}
		result[key] = value
	}
	maps.Copy(result, request)
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

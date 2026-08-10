package generation

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"ai-video/internal/gen/model"
	"ai-video/internal/pkg/ucloud"
)

// Provider 定义第三方异步视频生成服务的统一接口。
type Provider interface {
	Submit(ctx context.Context, config *model.VideoModel, request remoteSubmitRequest) (*ProviderSubmitResult, error)
	Status(ctx context.Context, config *model.VideoModel, taskID string) (*ProviderTaskStatus, error)
}

// ModelVerseProvider 实现 ModelVerse 的任务提交和状态查询协议。
type ModelVerseProvider struct{}

func (p *ModelVerseProvider) Submit(ctx context.Context, config *model.VideoModel, request remoteSubmitRequest) (*ProviderSubmitResult, error) {
	return p.submitUCloud(ctx, config, request)
}

func (p *ModelVerseProvider) submitLegacy(ctx context.Context, config *model.VideoModel, request remoteSubmitRequest) (*ProviderSubmitResult, error) {
	body, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	endpoint, err := resolveEndpoint(modelBaseURL(config), config.SubmitEndpoint)
	if err != nil {
		return nil, err
	}
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	if err := applyModelAuthentication(httpRequest, config); err != nil {
		return nil, err
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	raw, statusCode, err := executeProviderRequest(config, httpRequest)
	if err != nil {
		return nil, err
	}
	if statusCode < http.StatusOK || statusCode >= http.StatusMultipleChoices {
		return nil, providerResponseError(statusCode, raw)
	}
	var response struct {
		Output struct {
			TaskID string `json:"task_id"`
		} `json:"output"`
		RequestID string `json:"request_id"`
	}
	if err := json.Unmarshal(raw, &response); err != nil {
		return nil, fmt.Errorf("解析 ModelVerse 提交响应失败: %w", err)
	}
	if strings.TrimSpace(response.Output.TaskID) == "" {
		return nil, errors.New("ModelVerse 未返回 task_id")
	}
	return &ProviderSubmitResult{
		TaskID: response.Output.TaskID, RequestID: response.RequestID, RawResponse: string(raw),
	}, nil
}

func (p *ModelVerseProvider) submitUCloud(ctx context.Context, config *model.VideoModel, request remoteSubmitRequest) (*ProviderSubmitResult, error) {
	input, err := generationInputFromMap(config.ModelType, request.Input)
	if err != nil {
		return nil, err
	}
	modelCode := strings.TrimSpace(config.Code)
	requestModel := strings.TrimSpace(request.Model)
	if requestModel != "" && modelCode != "" && requestModel != modelCode {
		return nil, fmt.Errorf("queued model %q does not match configured model %q", requestModel, modelCode)
	}
	if modelCode == "" {
		modelCode = requestModel
	}
	client, err := ucloud.NewClient(ucloud.ClientConfig{
		APIKey: modelAPIKey(config), BaseURL: modelBaseURL(config),
		SubmitEndpoint: config.SubmitEndpoint, StatusEndpoint: config.StatusEndpoint,
	})
	if err != nil {
		return nil, err
	}
	parameters := cloneMap(request.Parameters)
	if input.FirstFrame != "" || input.EndFrame != "" {
		switch modelCode {
		case ucloud.ModelKlingV3:
			if _, exists := parameters["image"]; exists {
				return nil, errors.New("parameters.image is managed by input.first_frame")
			}
			if _, exists := parameters["image_tail"]; exists {
				return nil, errors.New("parameters.image_tail is managed by input.end_frame")
			}
			parameters["image"] = input.FirstFrame
			if input.EndFrame != "" {
				parameters["image_tail"] = input.EndFrame
			}
		case ucloud.ModelKlingO3:
			if _, exists := parameters["image_list"]; exists {
				return nil, errors.New("parameters.image_list is managed by input.first_frame and input.end_frame")
			}
			images := []ucloud.KlingO3ImageReference{{
				ImageURL: input.FirstFrame, Type: ucloud.KlingO3ImageTypeFirstFrame,
			}}
			if input.EndFrame != "" {
				images = append(images, ucloud.KlingO3ImageReference{
					ImageURL: input.EndFrame, Type: ucloud.KlingO3ImageTypeEndFrame,
				})
			}
			parameters["image_list"] = images
		default:
			return nil, fmt.Errorf("model %q does not support first/end frame input", modelCode)
		}
	}
	generationType := ucloud.GenerationTypeImage
	if config.ModelType == TaskTypeVideo {
		generationType = ucloud.GenerationTypeVideo
	}
	response, err := client.TaskSubmit(ctx, ucloud.TaskSubmitRequest{
		Model: modelCode, GenerationType: generationType, Prompt: input.Prompt,
		Images: input.Images, Video: input.Video, Parameters: parameters,
	})
	if err != nil {
		return nil, err
	}
	raw, err := json.Marshal(response)
	if err != nil {
		return nil, err
	}
	result := &ProviderSubmitResult{
		TaskID: response.TaskID, RequestID: response.RequestID, RawResponse: string(raw),
		Completed: generationType == ucloud.GenerationTypeImage,
	}
	for i := range response.Images {
		image := response.Images[i]
		if image.Error != nil && strings.TrimSpace(image.Error.Message) != "" {
			return nil, fmt.Errorf("UCloud image %d failed: %s", i, image.Error.Message)
		}
		if value := strings.TrimSpace(image.URL); value != "" {
			result.URLs = append(result.URLs, value)
		}
		if value := strings.TrimSpace(image.B64JSON); value != "" {
			result.Base64Images = append(result.Base64Images, value)
		}
	}
	if result.Completed && len(result.URLs) == 0 && len(result.Base64Images) == 0 {
		return nil, errors.New("UCloud image response does not contain generated images")
	}
	return result, nil
}

func (p *ModelVerseProvider) Status(ctx context.Context, config *model.VideoModel, taskID string) (*ProviderTaskStatus, error) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return nil, errors.New("third_task_code 不能为空")
	}
	statusEndpoint := strings.TrimSpace(config.StatusEndpoint)
	if statusEndpoint == "" {
		statusEndpoint = "/v1/tasks/status"
	}
	endpoint, err := resolveEndpoint(modelBaseURL(config), statusEndpoint)
	if err != nil {
		return nil, err
	}
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}
	query := parsed.Query()
	query.Set("task_id", taskID)
	parsed.RawQuery = query.Encode()
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return nil, err
	}
	if err := applyModelAuthentication(httpRequest, config); err != nil {
		return nil, err
	}
	raw, statusCode, err := executeProviderRequest(config, httpRequest)
	if err != nil {
		return nil, err
	}
	if statusCode < http.StatusOK || statusCode >= http.StatusMultipleChoices {
		return nil, providerResponseError(statusCode, raw)
	}
	var response struct {
		Output struct {
			TaskID       string   `json:"task_id"`
			TaskStatus   string   `json:"task_status"`
			URLs         []string `json:"urls"`
			SubmitTime   int64    `json:"submit_time"`
			FinishTime   int64    `json:"finish_time"`
			ErrorMessage string   `json:"error_message"`
		} `json:"output"`
		Usage struct {
			Duration uint32 `json:"duration"`
		} `json:"usage"`
		RequestID string `json:"request_id"`
	}
	if err := json.Unmarshal(raw, &response); err != nil {
		return nil, fmt.Errorf("解析 ModelVerse 任务状态失败: %w", err)
	}
	response.Output.TaskID = strings.TrimSpace(response.Output.TaskID)
	response.Output.TaskStatus = strings.TrimSpace(response.Output.TaskStatus)
	if response.Output.TaskID == "" {
		return nil, errors.New("ModelVerse 未返回 task_id")
	}
	if response.Output.TaskID != taskID {
		return nil, fmt.Errorf("ModelVerse 返回的 task_id %q 与 third_task_code %q 不一致", response.Output.TaskID, taskID)
	}
	if response.Output.TaskStatus == "" {
		return nil, errors.New("ModelVerse 未返回 task_status")
	}
	return &ProviderTaskStatus{
		TaskID: response.Output.TaskID, Status: response.Output.TaskStatus, URLs: response.Output.URLs,
		ErrorMessage: response.Output.ErrorMessage, UsageDuration: response.Usage.Duration,
		SubmitTime: response.Output.SubmitTime, FinishTime: response.Output.FinishTime,
		RequestID: response.RequestID, RawResponse: string(raw),
	}, nil
}

func resolveEndpoint(baseURL, endpointPath string) (string, error) {
	base, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || base.Scheme == "" || base.Host == "" {
		return "", errors.New("模型 Base URL 无效")
	}
	if base.Scheme != "https" && base.Scheme != "http" {
		return "", errors.New("模型 Base URL 仅支持 HTTP(S)")
	}
	pathURL, err := url.Parse(strings.TrimSpace(endpointPath))
	if err != nil {
		return "", errors.New("模型 API 路径无效")
	}
	if pathURL.IsAbs() || pathURL.Host != "" {
		return "", errors.New("模型 API 路径不能覆盖 Base URL")
	}
	return base.ResolveReference(pathURL).String(), nil
}

func executeProviderRequest(_ *model.VideoModel, request *http.Request) ([]byte, int, error) {
	response, err := (&http.Client{Timeout: 30 * time.Second}).Do(request)
	if err != nil {
		return nil, 0, fmt.Errorf("请求 ModelVerse 失败: %w", err)
	}
	defer response.Body.Close()
	const maxResponseSize = 4 << 20
	raw, err := io.ReadAll(io.LimitReader(response.Body, maxResponseSize+1))
	if err != nil {
		return nil, response.StatusCode, err
	}
	if len(raw) > maxResponseSize {
		return nil, response.StatusCode, errors.New("ModelVerse 响应体过大")
	}
	return raw, response.StatusCode, nil
}

func modelBaseURL(config *model.VideoModel) string {
	if value := strings.TrimSpace(config.HostURL); value != "" {
		return value
	}
	return strings.TrimSpace(config.Platform.BaseURL)
}

func modelAPIKey(config *model.VideoModel) string {
	if value := strings.TrimSpace(config.APIKey); value != "" {
		return value
	}
	return strings.TrimSpace(config.Platform.APIKey)
}

func applyModelAuthentication(request *http.Request, config *model.VideoModel) error {
	apiKey := modelAPIKey(config)
	if apiKey == "" {
		return errors.New("模型尚未配置 API Key")
	}
	switch config.AuthType {
	case 1:
		request.Header.Set("Authorization", "Bearer "+apiKey)
	case 2:
		request.Header.Set("X-API-Key", apiKey)
	default:
		return fmt.Errorf("不支持的模型认证类型: %d", config.AuthType)
	}
	return nil
}

func providerResponseError(statusCode int, raw []byte) error {
	var response struct {
		Error struct {
			Message string      `json:"message"`
			Type    string      `json:"type"`
			Code    string      `json:"code"`
			Param   interface{} `json:"param"`
		} `json:"error"`
		RequestID string `json:"request_id"`
	}
	if json.Unmarshal(raw, &response) == nil && response.Error.Message != "" {
		return fmt.Errorf("ModelVerse HTTP %d (%s): %s", statusCode, response.Error.Code, response.Error.Message)
	}
	message := strings.TrimSpace(string(raw))
	if len(message) > 500 {
		message = message[:500]
	}
	return fmt.Errorf("ModelVerse HTTP %d: %s", statusCode, message)
}

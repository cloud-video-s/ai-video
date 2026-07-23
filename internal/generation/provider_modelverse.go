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
)

// Provider 定义第三方异步视频生成服务的统一接口。
type Provider interface {
	Submit(ctx context.Context, config *model.VideoAiModel, request remoteSubmitRequest) (*ProviderSubmitResult, error)
	Status(ctx context.Context, config *model.VideoAiModel, taskID string) (*ProviderTaskStatus, error)
}

// ModelVerseProvider 实现 ModelVerse 的任务提交和状态查询协议。
type ModelVerseProvider struct{}

func (p *ModelVerseProvider) Submit(ctx context.Context, config *model.VideoAiModel, request remoteSubmitRequest) (*ProviderSubmitResult, error) {
	body, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	endpoint, err := resolveEndpoint(config.BaseURL, config.SubmitPath)
	if err != nil {
		return nil, err
	}
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpRequest.Header.Set("Authorization", "Bearer "+config.APIKey)
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

func (p *ModelVerseProvider) Status(ctx context.Context, config *model.VideoAiModel, taskID string) (*ProviderTaskStatus, error) {
	endpoint, err := resolveEndpoint(config.BaseURL, config.StatusPath)
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
	httpRequest.Header.Set("Authorization", "Bearer "+config.APIKey)
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
	if strings.TrimSpace(response.Output.TaskStatus) == "" {
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

func executeProviderRequest(config *model.VideoAiModel, request *http.Request) ([]byte, int, error) {
	timeout := time.Duration(config.HTTPTimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	response, err := (&http.Client{Timeout: timeout}).Do(request)
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

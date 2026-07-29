package ucloud

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"
)

const (
	DefaultBaseURL        = "https://api.modelverse.cn"
	defaultRequestTimeout = 30 * time.Second
	defaultMaxResponse    = int64(64 << 20)
)

type GenerationType string

const (
	GenerationTypeImage GenerationType = "image"
	GenerationTypeVideo GenerationType = "video"
)

// HTTPDoer is implemented by *http.Client. It is exposed so callers can use
// their own transport and tests can provide an isolated HTTP server.
type HTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}

type ClientConfig struct {
	APIKey          string
	BaseURL         string
	SubmitEndpoint  string
	HTTPClient      HTTPDoer
	MaxResponseSize int64
}

type Client struct {
	apiKey          string
	baseURL         *url.URL
	submitEndpoint  string
	httpClient      HTTPDoer
	maxResponseSize int64
}

func NewClient(config ClientConfig) (*Client, error) {
	apiKey := strings.TrimSpace(config.APIKey)
	if apiKey == "" {
		return nil, errors.New("ucloud API key is required")
	}
	baseURL := strings.TrimSpace(config.BaseURL)
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, errors.New("ucloud base URL is invalid")
	}
	if parsed.Scheme != "https" && parsed.Scheme != "http" {
		return nil, errors.New("ucloud base URL must use HTTP or HTTPS")
	}
	if parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, errors.New("ucloud base URL must not contain credentials, query, or fragment")
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/") + "/"
	submitEndpoint := strings.TrimSpace(config.SubmitEndpoint)
	if submitEndpoint != "" {
		reference, err := url.Parse(submitEndpoint)
		if err != nil || reference.IsAbs() || reference.Host != "" || reference.RawQuery != "" || reference.Fragment != "" {
			return nil, errors.New("ucloud submit endpoint must be a relative URL path")
		}
		if !strings.HasPrefix(submitEndpoint, "/") {
			return nil, errors.New("ucloud submit endpoint must start with /")
		}
	}

	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultRequestTimeout}
	}
	maxResponseSize := config.MaxResponseSize
	if maxResponseSize <= 0 {
		maxResponseSize = defaultMaxResponse
	}
	return &Client{
		apiKey:          apiKey,
		baseURL:         parsed,
		submitEndpoint:  submitEndpoint,
		httpClient:      httpClient,
		maxResponseSize: maxResponseSize,
	}, nil
}

func (c *Client) generationEndpoint(defaultPath string) string {
	if c.submitEndpoint != "" {
		return c.submitEndpoint
	}
	return defaultPath
}

// TaskSubmitRequest is the application-facing generation request. Parameters
// are decoded strictly according to the selected model; unknown model options
// are rejected instead of being silently sent upstream.
type TaskSubmitRequest struct {
	GenerationType GenerationType         `json:"generation_type"`
	Prompt         string                 `json:"prompt"`
	Images         []string               `json:"images,omitempty"`
	Video          string                 `json:"video,omitempty"`
	Parameters     map[string]interface{} `json:"parameters,omitempty"`
}

type TaskSubmitResponse struct {
	Model          string                `json:"model"`
	GenerationType GenerationType        `json:"generation_type"`
	TaskID         string                `json:"task_id,omitempty"`
	RequestID      string                `json:"request_id,omitempty"`
	Created        int64                 `json:"created,omitempty"`
	Images         []DoubaoSeedreamImage `json:"images,omitempty"`
	Usage          *DoubaoSeedreamUsage  `json:"usage,omitempty"`
}

// TaskSubmit routes a unified request to the endpoint required by its model.
// Kling v3 and Kling O3 return an asynchronous task ID. Doubao Seedream
// returns generated image entries directly.
func (c *Client) TaskSubmit(ctx context.Context, request TaskSubmitRequest) (*TaskSubmitResponse, error) {
	request.Prompt = strings.TrimSpace(request.Prompt)
	request.Video = strings.TrimSpace(request.Video)
	for i := range request.Images {
		request.Images[i] = strings.TrimSpace(request.Images[i])
		if request.Images[i] == "" {
			return nil, fmt.Errorf("images[%d] must not be empty", i)
		}
	}

	switch request.GenerationType {
	case GenerationTypeVideo:
		//TaskModel, ok := request.Parameters["model"]
		//if ok {
		//	TaskModel = strings.TrimSpace(request.Parameters["model"].(string))
		//} else {
		//	TaskModel = ""
		//}
		//if TaskModel == "" {
		//	TaskModel = ModelKlingO3
		//}
		//switch TaskModel {
		//case ModelKlingO3:
		upstream, err := buildKlingO3Request(request)
		if err != nil {
			return nil, err
		}
		result, err := c.SubmitKlingO3Task(ctx, upstream)
		if err != nil {
			return nil, err
		}
		return &TaskSubmitResponse{
			Model: ModelKlingO3, GenerationType: GenerationTypeVideo,
			TaskID: result.Output.TaskID, RequestID: result.RequestID,
		}, nil
		//default:
		//	return nil, fmt.Errorf("model %q does not support video generation", TaskModel)
		//}
	case GenerationTypeImage:
		//if !isDoubaoSeedreamModel(request.Model) {
		//	return nil, fmt.Errorf("model %q does not support image generation", request.Model)
		//}
		if request.Video != "" {
			return nil, errors.New("video is only supported for video generation")
		}
		upstream, err := buildDoubaoSeedreamRequest(request)
		if err != nil {
			return nil, err
		}
		result, err := c.CreateDoubaoSeedreamImage(ctx, upstream)
		if err != nil {
			return nil, err
		}
		return &TaskSubmitResponse{
			Model: result.Model, GenerationType: GenerationTypeImage,
			Created: result.Created, Images: result.Data, Usage: result.Usage,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported generation_type %q", request.GenerationType)
	}
}

type APIError struct {
	StatusCode int    `json:"-"`
	Code       string `json:"code,omitempty"`
	Type       string `json:"type,omitempty"`
	Param      any    `json:"param,omitempty"`
	Message    string `json:"message"`
	RequestID  string `json:"request_id,omitempty"`
}

func (e *APIError) Error() string {
	detail := e.Message
	if detail == "" {
		detail = http.StatusText(e.StatusCode)
	}
	if e.Code != "" {
		return fmt.Sprintf("ucloud API HTTP %d (%s): %s", e.StatusCode, e.Code, detail)
	}
	return fmt.Sprintf("ucloud API HTTP %d: %s", e.StatusCode, detail)
}

func (c *Client) postJSON(ctx context.Context, endpointPath string, request, response any) error {
	body, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("marshal ucloud request: %w", err)
	}
	httpResponse, err := c.post(ctx, endpointPath, bytes.NewReader(body), "application/json")
	if err != nil {
		return err
	}
	defer httpResponse.Body.Close()
	raw, err := readLimited(httpResponse.Body, c.maxResponseSize)
	if err != nil {
		return err
	}
	if httpResponse.StatusCode < http.StatusOK || httpResponse.StatusCode >= http.StatusMultipleChoices {
		return parseAPIError(httpResponse.StatusCode, raw)
	}
	if err := json.Unmarshal(raw, response); err != nil {
		return fmt.Errorf("decode ucloud response: %w", err)
	}
	return nil
}

func (c *Client) post(ctx context.Context, endpointPath string, body io.Reader, accept string) (*http.Response, error) {
	reference, err := url.Parse(strings.TrimLeft(endpointPath, "/"))
	if err != nil {
		return nil, fmt.Errorf("invalid ucloud endpoint: %w", err)
	}
	endpoint := c.baseURL.ResolveReference(reference)
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), body)
	if err != nil {
		return nil, fmt.Errorf("create ucloud request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+c.apiKey)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", accept)
	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("request ucloud API: %w", err)
	}
	return response, nil
}

func decodeParameters(source map[string]any, target any) error {
	if len(source) == 0 {
		return nil
	}
	raw, err := json.Marshal(source)
	if err != nil {
		return fmt.Errorf("encode parameters: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("invalid model parameters: %w", err)
	}
	return nil
}

func readLimited(reader io.Reader, limit int64) ([]byte, error) {
	raw, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if err != nil {
		return nil, fmt.Errorf("read ucloud response: %w", err)
	}
	if int64(len(raw)) > limit {
		return nil, fmt.Errorf("ucloud response exceeds %d bytes", limit)
	}
	return raw, nil
}

func parseAPIError(statusCode int, raw []byte) error {
	var envelope struct {
		Error struct {
			Code    interface{} `json:"code"`
			Type    string      `json:"type"`
			Param   interface{} `json:"param"`
			Message string      `json:"message"`
		} `json:"error"`
		Code      interface{} `json:"code"`
		Message   string      `json:"message"`
		RequestID string      `json:"request_id"`
	}
	_ = json.Unmarshal(raw, &envelope)
	message := strings.TrimSpace(envelope.Error.Message)
	if message == "" {
		message = strings.TrimSpace(envelope.Message)
	}
	if message == "" {
		message = strings.TrimSpace(string(raw))
		if len(message) > 500 {
			message = message[:500]
		}
	}
	code := envelope.Error.Code
	if code == nil {
		code = envelope.Code
	}
	return &APIError{
		StatusCode: statusCode,
		Code:       stringifyCode(code),
		Type:       envelope.Error.Type,
		Param:      envelope.Error.Param,
		Message:    message,
		RequestID:  envelope.RequestID,
	}
}

func stringifyCode(value any) string {
	if value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return typed
	case float64:
		return fmt.Sprintf("%g", typed)
	default:
		return fmt.Sprint(typed)
	}
}

func oneOf(value string, allowed ...string) bool {
	return slices.Contains(allowed, value)
}

func isAbsoluteURL(value string) bool {
	parsed, err := url.ParseRequestURI(value)
	return err == nil && parsed.Scheme != "" && parsed.Host != ""
}

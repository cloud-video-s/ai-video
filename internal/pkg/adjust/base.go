// Package adjust provides a client for the Adjust Campaign API.
package adjust

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"ai-video/internal/pkg/tracing"
)

const (
	// DefaultBaseURL is the root URL of version 2 of the Adjust Campaign API.
	DefaultBaseURL = "https://api.adjust.com/public/v2"

	defaultRequestTimeout  = 30 * time.Second
	defaultMaxResponseSize = int64(4 << 20)
)

// HTTPDoer is implemented by *http.Client. It allows callers to configure
// their own transport and tests to use an isolated HTTP server.
type HTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}

// ClientConfig configures a Campaign API client.
type ClientConfig struct {
	APIToken        string
	BaseURL         string
	HTTPClient      HTTPDoer
	MaxResponseSize int64
}

// Client is an Adjust Campaign API client.
type Client struct {
	apiToken        string
	baseURL         *url.URL
	httpClient      HTTPDoer
	maxResponseSize int64
}

// NewClient creates an Adjust Campaign API client.
func NewClient(config ClientConfig) (*Client, error) {
	apiToken := strings.TrimSpace(config.APIToken)
	if apiToken == "" {
		return nil, errors.New("adjust API token is required")
	}

	baseURL := strings.TrimSpace(config.BaseURL)
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, errors.New("adjust base URL is invalid")
	}
	if parsed.Scheme != "https" && parsed.Scheme != "http" {
		return nil, errors.New("adjust base URL must use HTTP or HTTPS")
	}
	if parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, errors.New("adjust base URL must not contain credentials, query, or fragment")
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/") + "/"

	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultRequestTimeout}
	}
	maxResponseSize := config.MaxResponseSize
	if maxResponseSize <= 0 {
		maxResponseSize = defaultMaxResponseSize
	}

	return &Client{
		apiToken:        apiToken,
		baseURL:         parsed,
		httpClient:      httpClient,
		maxResponseSize: maxResponseSize,
	}, nil
}

// APIError describes a non-success response from the Adjust API.
type APIError struct {
	StatusCode int    `json:"-"`
	Code       string `json:"code,omitempty"`
	Message    string `json:"message"`
	RequestID  string `json:"request_id,omitempty"`
}

func (e *APIError) Error() string {
	message := strings.TrimSpace(e.Message)
	if message == "" {
		message = http.StatusText(e.StatusCode)
	}
	if e.Code != "" {
		return fmt.Sprintf("adjust API HTTP %d (%s): %s", e.StatusCode, e.Code, message)
	}
	return fmt.Sprintf("adjust API HTTP %d: %s", e.StatusCode, message)
}

func (c *Client) getJSON(ctx context.Context, endpointPath string, query url.Values, destination any) error {
	reference, err := url.Parse(strings.TrimLeft(endpointPath, "/"))
	if err != nil {
		return fmt.Errorf("create adjust endpoint: %w", err)
	}
	endpoint := c.baseURL.ResolveReference(reference)
	endpoint.RawQuery = query.Encode()

	request, err := tracing.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return fmt.Errorf("create adjust request: %w", err)
	}
	request.Header.Set("Authorization", "Token token="+c.apiToken)
	request.Header.Set("Accept", "application/json")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("request adjust API: %w", err)
	}
	defer response.Body.Close()

	body, err := readResponse(response.Body, c.maxResponseSize)
	if err != nil {
		return err
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return parseAPIError(response.StatusCode, body)
	}
	if err := json.Unmarshal(body, destination); err != nil {
		return fmt.Errorf("decode adjust response: %w", err)
	}
	return nil
}

func readResponse(reader io.Reader, limit int64) ([]byte, error) {
	body, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if err != nil {
		return nil, fmt.Errorf("read adjust response: %w", err)
	}
	if int64(len(body)) > limit {
		return nil, fmt.Errorf("adjust response exceeds %d bytes", limit)
	}
	return body, nil
}

func parseAPIError(statusCode int, body []byte) error {
	var envelope struct {
		Error            json.RawMessage `json:"error"`
		Code             any             `json:"code"`
		ErrorCode        any             `json:"error_code"`
		Message          string          `json:"message"`
		ErrorDescription string          `json:"error_description"`
		ErrorDesc        string          `json:"error_desc"`
		RequestID        string          `json:"request_id"`
	}
	_ = json.Unmarshal(body, &envelope)

	code := envelope.Code
	if code == nil {
		code = envelope.ErrorCode
	}
	message := firstNonEmpty(envelope.Message, envelope.ErrorDescription, envelope.ErrorDesc)
	if len(envelope.Error) > 0 && string(envelope.Error) != "null" {
		var detail struct {
			Code      any    `json:"code"`
			Message   string `json:"message"`
			RequestID string `json:"request_id"`
		}
		if err := json.Unmarshal(envelope.Error, &detail); err == nil {
			if code == nil {
				code = detail.Code
			}
			message = firstNonEmpty(detail.Message, message)
			envelope.RequestID = firstNonEmpty(envelope.RequestID, detail.RequestID)
		} else {
			var text string
			if json.Unmarshal(envelope.Error, &text) == nil {
				message = firstNonEmpty(text, message)
			}
		}
	}
	if message == "" {
		message = strings.TrimSpace(string(body))
		if len(message) > 500 {
			message = message[:500]
		}
	}

	return &APIError{
		StatusCode: statusCode,
		Code:       stringify(code),
		Message:    message,
		RequestID:  strings.TrimSpace(envelope.RequestID),
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}

func stringify(value any) string {
	if value == nil {
		return ""
	}
	if text, ok := value.(string); ok {
		return strings.TrimSpace(text)
	}
	return fmt.Sprint(value)
}

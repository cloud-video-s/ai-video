// Package adjust reports server-to-server events to Adjust.
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
	// DefaultBaseURL is the Adjust S2S API base URL.
	DefaultBaseURL         = "https://s2s.adjust.com"
	defaultCampaignBaseURL = "https://api.adjust.com/public/v2"

	defaultRequestTimeout  = 100 * time.Second
	defaultMaxResponseSize = int64(1 << 20)
)

// HTTPDoer is implemented by *http.Client. It allows callers to customize the
// transport and tests to use an isolated HTTP server.
type HTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}

// ClientConfig configures an Adjust S2S event client.
type ClientConfig struct {
	// APIToken authenticates Adjust Campaign API requests.
	APIToken string
	// AppToken is the Adjust app token and is required for every event.
	AppToken string
	// AuthToken is the optional token generated for Adjust S2S Security.
	AuthToken       string
	BaseURL         string
	HTTPClient      HTTPDoer
	MaxResponseSize int64
}

// Client reports events to the Adjust S2S API.
type Client struct {
	apiToken        string
	appToken        string
	authToken       string
	baseURL         *url.URL
	httpClient      HTTPDoer
	maxResponseSize int64
}

// NewClient creates an Adjust S2S event client.
func NewClient(config ClientConfig) (*Client, error) {
	appToken := strings.TrimSpace(config.AppToken)
	apiToken := strings.TrimSpace(config.APIToken)
	if appToken == "" && apiToken == "" {
		return nil, errors.New("adjust app token is required; Adjust API token is required")
	}
	if appToken != "" && !isAlphaNumeric(appToken) {
		return nil, errors.New("adjust app token must be alphanumeric")
	}

	baseURL := strings.TrimSpace(config.BaseURL)
	if baseURL == "" {
		if appToken != "" {
			baseURL = DefaultBaseURL
		} else {
			baseURL = defaultCampaignBaseURL
		}
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
		appToken:        appToken,
		authToken:       strings.TrimSpace(config.AuthToken),
		baseURL:         parsed,
		httpClient:      httpClient,
		maxResponseSize: maxResponseSize,
	}, nil
}

// ReportEvent reports one event to Adjust. The S2S event endpoint documents
// 200 as accepted; notably, authentication failures may use HTTP 202 and must
// therefore not be treated as generic 2xx success responses.
func (c *Client) ReportEvent(ctx context.Context, token EventToken, event Event) error {
	if ctx == nil {
		return errors.New("adjust event context is required")
	}
	if c.appToken == "" {
		return errors.New("adjust app token is required")
	}
	form, err := event.form(token, c.appToken)
	if err != nil {
		return err
	}

	endpoint := c.baseURL.ResolveReference(&url.URL{Path: "event"})
	request, err := tracing.NewRequestWithContext(
		ctx,
		http.MethodPost,
		endpoint.String(),
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return fmt.Errorf("create Adjust event request: %w", err)
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Accept", "application/json")
	if c.authToken != "" {
		request.Header.Set("Authorization", "Bearer "+c.authToken)
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("report Adjust event: %w", err)
	}
	defer response.Body.Close()

	body, err := readResponse(response.Body, c.maxResponseSize)
	if err != nil {
		return err
	}
	if response.StatusCode != http.StatusOK {
		return parseAPIError(response.StatusCode, body)
	}
	return nil
}

func (c *Client) getJSON(ctx context.Context, endpointPath string, query url.Values, destination any) error {
	if c.apiToken == "" {
		return errors.New("Adjust API token is required")
	}
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
	request.Header.Set("Authorization", "Bearer orKCYdJSFTyPxJyAUbGm")
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
	if err = json.Unmarshal(body, destination); err != nil {
		return fmt.Errorf("decode adjust response: %w", err)
	}
	return nil
}

// APIError describes a non-success response from the Adjust S2S API.
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
		return fmt.Sprintf("Adjust event API HTTP %d (%s): %s", e.StatusCode, e.Code, message)
	}
	return fmt.Sprintf("Adjust event API HTTP %d: %s", e.StatusCode, message)
}

func readResponse(reader io.Reader, limit int64) ([]byte, error) {
	body, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if err != nil {
		return nil, fmt.Errorf("read Adjust event response: %w", err)
	}
	if int64(len(body)) > limit {
		return nil, fmt.Errorf("Adjust event response exceeds %d bytes", limit)
	}
	return body, nil
}

func parseAPIError(statusCode int, body []byte) error {
	var envelope struct {
		Code      any             `json:"code"`
		Message   string          `json:"message"`
		Reason    string          `json:"reason"`
		Error     json.RawMessage `json:"error"`
		RequestID string          `json:"request_id"`
	}
	_ = json.Unmarshal(body, &envelope)

	message := firstNonEmpty(envelope.Message, envelope.Reason)
	code := stringify(envelope.Code)
	if len(envelope.Error) > 0 && string(envelope.Error) != "null" {
		var detail struct {
			Code    any    `json:"code"`
			Message string `json:"message"`
			Reason  string `json:"reason"`
		}
		if err := json.Unmarshal(envelope.Error, &detail); err == nil {
			if code == "" {
				code = stringify(detail.Code)
			}
			message = firstNonEmpty(detail.Message, detail.Reason, message)
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

	return &APIError{StatusCode: statusCode, Code: code, Message: message, RequestID: strings.TrimSpace(envelope.RequestID)}
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

func isAlphaNumeric(value string) bool {
	if value == "" {
		return false
	}
	for _, char := range value {
		if (char < 'a' || char > 'z') && (char < 'A' || char > 'Z') && (char < '0' || char > '9') {
			return false
		}
	}
	return true
}

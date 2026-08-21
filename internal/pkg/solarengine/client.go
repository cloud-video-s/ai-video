// Package solarengine provides a client for SolarEngine report export APIs.
package solarengine

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"ai-video/internal/pkg/tracing"
)

const (
	// DefaultBaseURL is the root URL of the SolarEngine open report API.
	DefaultBaseURL = "https://portal.solar-engine.com/portal-api/portal/openReport/"

	defaultRequestTimeout  = 30 * time.Second
	defaultMaxResponseSize = int64(8 << 20)
	defaultMaxConcurrency  = 2
)

// HTTPDoer is implemented by *http.Client. It lets callers customize the
// transport and lets tests use an isolated HTTP server.
type HTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}

// ClientConfig configures a SolarEngine report client.
type ClientConfig struct {
	// ReportKey is obtained from the SolarEngine portal and is used to sign
	// every report request.
	ReportKey string
	BaseURL   string

	HTTPClient      HTTPDoer
	MaxResponseSize int64
	// MaxConcurrency defaults to SolarEngine's documented per-interface limit
	// of two concurrent requests. Valid explicit values are one and two.
	MaxConcurrency int

	// Now is optional and mainly useful for deterministic tests.
	Now func() time.Time
}

// Client calls SolarEngine report export APIs.
type Client struct {
	reportKey       string
	baseURL         *url.URL
	httpClient      HTTPDoer
	maxResponseSize int64
	requestSlots    chan struct{}
	now             func() time.Time
}

// NewClient creates a SolarEngine report client.
func NewClient(config ClientConfig) (*Client, error) {
	reportKey := strings.TrimSpace(config.ReportKey)
	if reportKey == "" {
		return nil, errors.New("SolarEngine report key is required")
	}

	baseURL := strings.TrimSpace(config.BaseURL)
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, errors.New("SolarEngine base URL is invalid")
	}
	if parsed.Scheme != "https" && parsed.Scheme != "http" {
		return nil, errors.New("SolarEngine base URL must use HTTP or HTTPS")
	}
	if parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, errors.New("SolarEngine base URL must not contain credentials, query, or fragment")
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
	maxConcurrency := config.MaxConcurrency
	if maxConcurrency == 0 {
		maxConcurrency = defaultMaxConcurrency
	}
	if maxConcurrency < 1 || maxConcurrency > defaultMaxConcurrency {
		return nil, fmt.Errorf("SolarEngine max concurrency must be between 1 and %d", defaultMaxConcurrency)
	}
	now := config.Now
	if now == nil {
		now = time.Now
	}

	return &Client{
		reportKey:       reportKey,
		baseURL:         parsed,
		httpClient:      httpClient,
		maxResponseSize: maxResponseSize,
		requestSlots:    make(chan struct{}, maxConcurrency),
		now:             now,
	}, nil
}

func (c *Client) postReport(
	ctx context.Context,
	endpointPath string,
	startDate string,
	endDate string,
	payload any,
) ([]byte, int, error) {
	if ctx == nil {
		return nil, 0, errors.New("SolarEngine request context is required")
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, 0, fmt.Errorf("marshal SolarEngine request: %w", err)
	}
	reference, err := url.Parse(strings.TrimLeft(endpointPath, "/"))
	if err != nil {
		return nil, 0, fmt.Errorf("create SolarEngine endpoint: %w", err)
	}
	endpoint := c.baseURL.ResolveReference(reference)

	select {
	case c.requestSlots <- struct{}{}:
		defer func() { <-c.requestSlots }()
	case <-ctx.Done():
		return nil, 0, fmt.Errorf("wait for SolarEngine request slot: %w", ctx.Err())
	}

	timestamp := c.now().Unix()
	request, err := tracing.NewRequestWithContext(
		ctx,
		http.MethodPost,
		endpoint.String(),
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, 0, fmt.Errorf("create SolarEngine request: %w", err)
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("reportKey", c.reportKey)
	request.Header.Set("timestamp", strconv.FormatInt(timestamp, 10))
	request.Header.Set("sign", reportSignature(c.reportKey, timestamp, startDate, endDate))

	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, 0, fmt.Errorf("request SolarEngine API: %w", err)
	}
	defer response.Body.Close()

	raw, err := readResponse(response.Body, c.maxResponseSize)
	if err != nil {
		return nil, response.StatusCode, err
	}
	return raw, response.StatusCode, nil
}

func reportSignature(reportKey string, timestamp int64, startDate, endDate string) string {
	source := reportKey + strconv.FormatInt(timestamp, 10) + startDate + endDate
	digest := md5.Sum([]byte(source))
	return hex.EncodeToString(digest[:])
}

func readResponse(reader io.Reader, limit int64) ([]byte, error) {
	raw, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if err != nil {
		return nil, fmt.Errorf("read SolarEngine response: %w", err)
	}
	if int64(len(raw)) > limit {
		return nil, fmt.Errorf("SolarEngine response exceeds %d bytes", limit)
	}
	return raw, nil
}

// APIError describes an HTTP or application-level failure returned by
// SolarEngine.
type APIError struct {
	StatusCode int          `json:"-"`
	Code       ResponseCode `json:"code"`
	Message    string       `json:"message"`
	RequestID  string       `json:"request_id,omitempty"`
}

func (e *APIError) Error() string {
	message := strings.TrimSpace(e.Message)
	if message == "" {
		message = http.StatusText(e.StatusCode)
	}
	if e.Code != 0 {
		return fmt.Sprintf("SolarEngine API HTTP %d (code %d): %s", e.StatusCode, e.Code, message)
	}
	return fmt.Sprintf("SolarEngine API HTTP %d: %s", e.StatusCode, message)
}

func parseAPIError(statusCode int, raw []byte) error {
	var envelope struct {
		Code      ResponseCode `json:"code"`
		Message   string       `json:"message"`
		RequestID string       `json:"request_id"`
	}
	_ = json.Unmarshal(raw, &envelope)
	message := strings.TrimSpace(envelope.Message)
	if message == "" {
		message = strings.TrimSpace(string(raw))
		if len(message) > 500 {
			message = message[:500]
		}
	}
	return &APIError{
		StatusCode: statusCode,
		Code:       envelope.Code,
		Message:    message,
		RequestID:  strings.TrimSpace(envelope.RequestID),
	}
}

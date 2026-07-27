package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// HTTPClient 封装 http.Client，提供便捷的请求方法
type HTTPClient struct {
	client  *http.Client
	baseURL string // 可选基础 URL，便于复用
}

// NewHTTPClient 创建新的 HTTPClient 实例，支持自定义超时
func NewHTTPClient(timeout time.Duration) *HTTPClient {
	return &HTTPClient{
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

// SetBaseURL 设置基础 URL，后续请求可使用相对路径
func (c *HTTPClient) SetBaseURL(baseURL string) {
	c.baseURL = baseURL
}

// Request 核心请求方法，支持自定义请求头、请求体（io.Reader）
// 参数：
//
//	ctx: 上下文，可用于超时控制或取消
//	method: HTTP 方法 (GET, POST, PUT, DELETE, PATCH 等)
//	url: 请求地址（如果设置了 baseURL 则自动拼接）
//	headers: 自定义请求头，键值对
//	body: 请求体，若无需请求体则传 nil
//
// 返回：*http.Response 和 error
func (c *HTTPClient) Request(ctx context.Context, method, url string, headers map[string]string, body io.Reader) (*http.Response, error) {
	// 拼接完整 URL
	fullURL := url
	if c.baseURL != "" {
		// 简单拼接，实际可考虑使用 url.Parse 进行更规范拼接
		fullURL = c.baseURL + url
	}

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, method, fullURL, body)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置自定义请求头
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// 执行请求
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("执行请求失败: %w", err)
	}

	return resp, nil
}

// 便捷方法：GET 请求
func (c *HTTPClient) Get(ctx context.Context, url string, headers map[string]string) (*http.Response, error) {
	return c.Request(ctx, http.MethodGet, url, headers, nil)
}

// 便捷方法：POST 请求，支持任意 body（io.Reader）
func (c *HTTPClient) Post(ctx context.Context, url string, headers map[string]string, body io.Reader) (*http.Response, error) {
	return c.Request(ctx, http.MethodPost, url, headers, body)
}

// 便捷方法：POST JSON 请求，自动序列化 body 并设置 Content-Type: application/json
func (c *HTTPClient) PostJSON(ctx context.Context, url string, headers map[string]string, data interface{}) (*http.Response, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("JSON 序列化失败: %w", err)
	}
	// 合并自定义头，并增加 Content-Type
	finalHeaders := make(map[string]string)
	for k, v := range headers {
		finalHeaders[k] = v
	}
	finalHeaders["Content-Type"] = "application/json"
	return c.Post(ctx, url, finalHeaders, bytes.NewReader(jsonData))
}

// 便捷方法：PUT 请求
func (c *HTTPClient) Put(ctx context.Context, url string, headers map[string]string, body io.Reader) (*http.Response, error) {
	return c.Request(ctx, http.MethodPut, url, headers, body)
}

// 便捷方法：DELETE 请求
func (c *HTTPClient) Delete(ctx context.Context, url string, headers map[string]string) (*http.Response, error) {
	return c.Request(ctx, http.MethodDelete, url, headers, nil)
}

// 辅助函数：读取响应体为字符串（需手动关闭响应体）
func ReadResponseBody(resp *http.Response) (string, error) {
	defer resp.Body.Close()
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应体失败: %w", err)
	}
	return string(bodyBytes), nil
}

// 辅助函数：将响应体解析为 JSON（需手动关闭响应体）
func UnmarshalResponse(resp *http.Response, target interface{}) error {
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(target)
}

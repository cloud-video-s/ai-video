package service

import (
	"ai-video/internal/config"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func ResolveCountry(ctx context.Context, clientIP, countryHeader string) (string, error) {
	if country := normalizeCountry(countryHeader); country != "" {
		return country, nil
	}
	cfg := config.Cfg.GeoIP
	if cfg.LookupURL == "" {
		return "", nil
	}
	ip := net.ParseIP(strings.TrimSpace(clientIP))
	if ip == nil {
		return "", errors.New("客户端 IP 无效")
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() {
		return "", nil
	}
	lookupURL := strings.ReplaceAll(cfg.LookupURL, "{ip}", url.QueryEscape(ip.String()))
	parsedURL, err := url.Parse(lookupURL)
	if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		return "", errors.New("geoip.lookup_url 配置无效")
	}
	timeout := time.Duration(cfg.TimeoutMS) * time.Millisecond
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, lookupURL, nil)
	if err != nil {
		return "", err
	}
	resp, err := (&http.Client{Timeout: timeout}).Do(req)
	if err != nil {
		return "", fmt.Errorf("IP 国家查询失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("IP 国家查询返回状态码 %d", resp.StatusCode)
	}
	var payload map[string]interface{}
	decoder := json.NewDecoder(io.LimitReader(resp.Body, 64<<10))
	if err := decoder.Decode(&payload); err != nil {
		return "", fmt.Errorf("解析 IP 国家响应失败: %w", err)
	}
	value := lookupJSONField(payload, cfg.CountryField)
	country := normalizeCountry(fmt.Sprint(value))
	if country == "" {
		return "", errors.New("IP 国家响应中没有有效国家编码")
	}
	return country, nil
}

func lookupJSONField(payload map[string]interface{}, path string) interface{} {
	var current interface{} = payload
	for _, part := range strings.Split(path, ".") {
		object, ok := current.(map[string]interface{})
		if !ok {
			return nil
		}
		current = object[part]
	}
	return current
}

func normalizeCountry(country string) string {
	country = strings.ToUpper(strings.TrimSpace(country))
	if len(country) < 2 || len(country) > 8 {
		return ""
	}
	for _, char := range country {
		if char < 'A' || char > 'Z' {
			return ""
		}
	}
	return country
}

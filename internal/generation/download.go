package generation

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"ai-video/internal/config"
	"ai-video/internal/gen/model"
)

func downloadVideos(ctx context.Context, task *model.VideoGenerationTask, remoteURLs []string) ([]string, error) {
	root, err := filepath.Abs(config.Cfg.Upload.LocalRootDir)
	if err != nil {
		return nil, err
	}
	relativeDir := filepath.Join("generated", fmt.Sprintf("%d", task.UserID))
	directory := filepath.Join(root, relativeDir)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return nil, err
	}
	client := secureDownloadClient()
	maxSize := config.Cfg.Upload.VideoMaxFileSize
	if maxSize <= 0 {
		maxSize = 2 << 30
	}
	result := make([]string, 0, len(remoteURLs))
	for index, remoteURL := range remoteURLs {
		filename := fmt.Sprintf("task-%d-%d.mp4", task.ID, index+1)
		destination := filepath.Join(directory, filename)
		if info, statErr := os.Stat(destination); statErr == nil && info.Size() > 0 {
			result = append(result, localVideoURL(relativeDir, filename))
			continue
		}
		if err := downloadOne(ctx, client, remoteURL, destination, maxSize); err != nil {
			return nil, err
		}
		result = append(result, localVideoURL(relativeDir, filename))
	}
	return result, nil
}

func downloadOne(ctx context.Context, client *http.Client, remoteURL, destination string, maxSize int64) error {
	if err := validatePublicHTTPURL(remoteURL); err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, remoteURL, nil)
	if err != nil {
		return err
	}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("下载视频返回 HTTP %d", response.StatusCode)
	}
	if response.ContentLength > maxSize {
		return errors.New("生成视频超过本地文件大小限制")
	}
	temporary := destination + ".part"
	file, err := os.OpenFile(temporary, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	written, copyErr := io.Copy(file, io.LimitReader(response.Body, maxSize+1))
	closeErr := file.Close()
	if copyErr != nil {
		_ = os.Remove(temporary)
		return copyErr
	}
	if closeErr != nil {
		_ = os.Remove(temporary)
		return closeErr
	}
	if written > maxSize {
		_ = os.Remove(temporary)
		return errors.New("生成视频超过本地文件大小限制")
	}
	return os.Rename(temporary, destination)
}

func localVideoURL(relativeDir, filename string) string {
	base := strings.TrimRight(config.Cfg.Upload.LocalBaseURL, "/")
	path := filepath.ToSlash(filepath.Join(relativeDir, filename))
	return base + "/" + strings.TrimLeft(path, "/")
}

func secureDownloadClient() *http.Client {
	dialer := &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(address)
			if err != nil {
				return nil, err
			}
			ips, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
			if err != nil {
				return nil, err
			}
			for _, ip := range ips {
				if publicIP(ip) {
					return dialer.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
				}
			}
			return nil, errors.New("视频下载地址指向非公网 IP")
		},
		TLSHandshakeTimeout: 10 * time.Second,
	}
	client := &http.Client{Transport: transport, Timeout: 30 * time.Minute}
	client.CheckRedirect = func(request *http.Request, via []*http.Request) error {
		if len(via) >= 5 {
			return errors.New("视频下载重定向次数过多")
		}
		return validatePublicHTTPURL(request.URL.String())
	}
	return client
}

func validatePublicHTTPURL(raw string) error {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Hostname() == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return errors.New("视频结果 URL 无效")
	}
	if parsed.User != nil {
		return errors.New("视频结果 URL 不能包含用户凭据")
	}
	return nil
}

func publicIP(ip net.IP) bool {
	return ip != nil && !ip.IsLoopback() && !ip.IsPrivate() && !ip.IsUnspecified() &&
		!ip.IsLinkLocalUnicast() && !ip.IsLinkLocalMulticast() && !ip.IsMulticast()
}

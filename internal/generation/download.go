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
	"ai-video/internal/pkg/upload"
	"ai-video/internal/pkg/uploadruntime"
)

func downloadVideos(ctx context.Context, task *model.VideoUserGenerationTask, remoteURLs []string) ([]string, error) {
	storage, err := uploadruntime.Storage()
	if err != nil {
		return nil, err
	}
	return downloadVideosToStorage(ctx, storage, secureDownloadClient(), task, remoteURLs)
}

func downloadVideosToStorage(
	ctx context.Context,
	storage upload.Storage,
	client *http.Client,
	task *model.VideoUserGenerationTask,
	remoteURLs []string,
) ([]string, error) {
	maxSize := config.Cfg.Upload.VideoMaxFileSize
	if maxSize <= 0 {
		maxSize = 2 << 30
	}
	result := make([]string, 0, len(remoteURLs))
	for index, remoteURL := range remoteURLs {
		filename := fmt.Sprintf("task-%s-%d.mp4", task.TaskCode, index+1)
		storedURL, err := downloadAndStoreGeneratedFile(
			ctx, storage, client, remoteURL, generatedObjectKey(task.UserID, filename), "video/mp4", maxSize,
		)
		if err != nil {
			return nil, err
		}
		result = append(result, storedURL)
	}
	return result, nil
}

func downloadAndStoreGeneratedFile(
	ctx context.Context,
	storage upload.Storage,
	client *http.Client,
	remoteURL, objectKey, contentType string,
	maxSize int64,
) (string, error) {
	temporary, err := newGeneratedTemporaryFile()
	if err != nil {
		return "", err
	}
	defer os.Remove(temporary)
	if err := downloadOne(ctx, client, remoteURL, temporary, maxSize); err != nil {
		return "", err
	}
	return storeGeneratedFile(ctx, storage, objectKey, temporary, contentType)
}

func storeGeneratedBytes(
	ctx context.Context,
	storage upload.Storage,
	objectKey, contentType string,
	contents []byte,
) (string, error) {
	temporary, err := newGeneratedTemporaryFile()
	if err != nil {
		return "", err
	}
	defer os.Remove(temporary)
	if err := os.WriteFile(temporary, contents, 0o600); err != nil {
		return "", err
	}
	return storeGeneratedFile(ctx, storage, objectKey, temporary, contentType)
}

func storeGeneratedFile(ctx context.Context, storage upload.Storage, objectKey, sourcePath, contentType string) (string, error) {
	if storage == nil {
		return "", errors.New("生成结果存储未配置")
	}
	stored, err := storage.Store(ctx, objectKey, sourcePath, contentType)
	if err != nil {
		return "", err
	}
	if stored == nil || strings.TrimSpace(stored.URL) == "" {
		return "", errors.New("生成结果存储未返回访问地址")
	}
	return stored.URL, nil
}

func newGeneratedTemporaryFile() (string, error) {
	root := strings.TrimSpace(config.Cfg.Upload.RootDir)
	if root == "" {
		return "", errors.New("生成结果临时目录未配置")
	}
	directory := filepath.Join(root, ".generated")
	if err := os.MkdirAll(directory, 0o750); err != nil {
		return "", fmt.Errorf("创建生成结果临时目录: %w", err)
	}
	file, err := os.CreateTemp(directory, ".result-*")
	if err != nil {
		return "", fmt.Errorf("创建生成结果临时文件: %w", err)
	}
	name := file.Name()
	if err := file.Close(); err != nil {
		_ = os.Remove(name)
		return "", err
	}
	return name, nil
}

func generatedObjectKey(userID uint64, filename string) string {
	return filepath.ToSlash(filepath.Join("generated", fmt.Sprintf("%d", userID), filename))
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
		return fmt.Errorf("下载生成结果返回 HTTP %d", response.StatusCode)
	}
	if response.ContentLength > maxSize {
		return errors.New("生成结果超过配置的文件大小限制")
	}
	file, err := os.OpenFile(destination, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	written, copyErr := io.Copy(file, io.LimitReader(response.Body, maxSize+1))
	closeErr := file.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}
	if written > maxSize {
		return errors.New("生成结果超过配置的文件大小限制")
	}
	return nil
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
			return nil, errors.New("生成结果下载地址指向非公网 IP")
		},
		TLSHandshakeTimeout: 10 * time.Second,
	}
	client := &http.Client{Transport: transport, Timeout: 30 * time.Minute}
	client.CheckRedirect = func(request *http.Request, via []*http.Request) error {
		if len(via) >= 5 {
			return errors.New("生成结果下载重定向次数过多")
		}
		return validatePublicHTTPURL(request.URL.String())
	}
	return client
}

func validatePublicHTTPURL(raw string) error {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Hostname() == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return errors.New("生成结果 URL 无效")
	}
	if parsed.User != nil {
		return errors.New("生成结果 URL 不能包含用户凭据")
	}
	return nil
}

func publicIP(ip net.IP) bool {
	return ip != nil && !ip.IsLoopback() && !ip.IsPrivate() && !ip.IsUnspecified() &&
		!ip.IsLinkLocalUnicast() && !ip.IsLinkLocalMulticast() && !ip.IsMulticast()
}

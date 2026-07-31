package generation

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"ai-video/internal/config"
	"ai-video/internal/pkg/upload"
)

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

package generation

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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
)

type generatedRecordingStorage struct {
	upload.Storage
	recorder upload.StoredUploadRecorder
	task     *model.VideoUserGenerationTask
	kind     upload.MediaKind
}

func recordGeneratedUploads(
	storage upload.Storage,
	recorder upload.StoredUploadRecorder,
	task *model.VideoUserGenerationTask,
	kind upload.MediaKind,
) upload.Storage {
	if storage == nil || recorder == nil || task == nil {
		return storage
	}
	return &generatedRecordingStorage{Storage: storage, recorder: recorder, task: task, kind: kind}
}

func (s *generatedRecordingStorage) Store(ctx context.Context, objectKey, sourcePath, contentType string) (*upload.StoredFile, error) {
	stored, err := s.Storage.Store(ctx, objectKey, sourcePath, contentType)
	if err != nil {
		return nil, err
	}
	if stored == nil {
		return nil, errors.New("生成结果存储未返回文件信息")
	}
	info, err := os.Stat(sourcePath)
	if err != nil {
		return nil, err
	}
	file, err := os.Open(sourcePath)
	if err != nil {
		return nil, err
	}
	hasher := sha256.New()
	_, hashErr := io.Copy(hasher, file)
	closeErr := file.Close()
	if hashErr != nil {
		return nil, hashErr
	}
	if closeErr != nil {
		return nil, closeErr
	}
	if err := s.recorder.RecordStored(ctx, upload.StoredUpload{
		Owner: upload.UploadOwner{Type: upload.UploaderAPIUser, ID: s.task.UserID},
		Kind:  s.kind, OriginalName: filepath.Base(objectKey), ContentType: contentType,
		FileSize: info.Size(), SHA256: hex.EncodeToString(hasher.Sum(nil)), Stored: *stored,
	}); err != nil {
		return nil, err
	}
	return stored, nil
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
	root := strings.TrimSpace(config.Cfg.Upload.LocalRootDir)
	if root == "" {
		return "", errors.New("生成结果本地存储目录未配置")
	}
	directory := filepath.Join(root, "tmp")
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

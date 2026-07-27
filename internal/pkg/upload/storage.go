package upload

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss/credentials"
)

const (
	StorageLocal     = "local"
	StorageAliyunOSS = "aliyun_oss"
)

type StoredFile struct {
	Provider string
	Path     string
	URL      string
}

type Storage interface {
	Store(ctx context.Context, objectKey, sourcePath, contentType string) (*StoredFile, error)
}

type StorageResolver func() (Storage, error)

type DynamicStorage struct {
	resolver StorageResolver
}

func NewDynamicStorage(resolver StorageResolver) (*DynamicStorage, error) {
	if resolver == nil {
		return nil, uploadError(ErrInvalidRequest, "storage resolver is required")
	}
	return &DynamicStorage{resolver: resolver}, nil
}

func (s *DynamicStorage) Store(ctx context.Context, objectKey, sourcePath, contentType string) (*StoredFile, error) {
	storage, err := s.resolver()
	if err != nil {
		return nil, err
	}
	if storage == nil {
		return nil, uploadError(ErrInvalidRequest, "resolved storage is nil")
	}
	return storage.Store(ctx, objectKey, sourcePath, contentType)
}

type LocalStorage struct {
	rootDir string
	baseURL string
}

func NewLocalStorage(rootDir, baseURL string) (*LocalStorage, error) {
	if strings.TrimSpace(rootDir) == "" {
		return nil, uploadError(ErrInvalidRequest, "local storage root directory is required")
	}
	root, err := filepath.Abs(rootDir)
	if err != nil {
		return nil, fmt.Errorf("resolve local storage root: %w", err)
	}
	return &LocalStorage{rootDir: root, baseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/")}, nil
}

func (s *LocalStorage) Store(ctx context.Context, objectKey, sourcePath, _ string) (*StoredFile, error) {
	target, err := safeStoragePath(s.rootDir, objectKey)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
		return nil, err
	}
	source, err := os.Open(sourcePath)
	if err != nil {
		return nil, err
	}
	defer source.Close()
	temp, err := os.CreateTemp(filepath.Dir(target), ".upload-*")
	if err != nil {
		return nil, err
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)

	_, copyErr := copyWithContext(ctx, temp, source)
	if copyErr == nil {
		copyErr = temp.Sync()
	}
	if closeErr := temp.Close(); copyErr == nil {
		copyErr = closeErr
	}
	if copyErr != nil {
		return nil, copyErr
	}
	if err := os.Rename(tempPath, target); err != nil {
		return nil, err
	}
	return &StoredFile{Provider: StorageLocal, Path: filepath.ToSlash(objectKey), URL: joinPublicURL(s.baseURL, objectKey)}, nil
}

type OSSConfig struct {
	Region          string
	Endpoint        string
	AccessKeyID     string
	AccessKeySecret string
	Bucket          string
	ObjectPrefix    string
	BaseURL         string
}

type ossObjectUploader interface {
	PutObjectFromFile(context.Context, *oss.PutObjectRequest, string, ...func(*oss.Options)) (*oss.PutObjectResult, error)
}

type OSSStorage struct {
	client       ossObjectUploader
	bucket       string
	objectPrefix string
	baseURL      string
}

func NewOSSStorage(config OSSConfig) (*OSSStorage, error) {
	if strings.TrimSpace(config.Region) == "" || strings.TrimSpace(config.Endpoint) == "" || strings.TrimSpace(config.AccessKeyID) == "" ||
		strings.TrimSpace(config.AccessKeySecret) == "" || strings.TrimSpace(config.Bucket) == "" {
		return nil, uploadError(ErrInvalidRequest, "OSS region, endpoint, credentials and bucket are required")
	}
	region := strings.TrimSpace(config.Region)
	endpoint := strings.TrimSpace(config.Endpoint)
	bucket := strings.TrimSpace(config.Bucket)
	sdkConfig := oss.LoadDefaultConfig().
		WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			strings.TrimSpace(config.AccessKeyID),
			strings.TrimSpace(config.AccessKeySecret),
		)).
		WithRegion(region).
		WithEndpoint(endpoint)
	client := oss.NewClient(sdkConfig)

	baseURL := strings.TrimRight(strings.TrimSpace(config.BaseURL), "/")
	if baseURL == "" {
		endpoint = strings.TrimRight(endpoint, "/")
		if !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
			endpoint = "https://" + endpoint
		}
		parsed, parseErr := url.Parse(endpoint)
		if parseErr != nil || parsed.Host == "" {
			return nil, uploadError(ErrInvalidRequest, "invalid OSS endpoint %q", config.Endpoint)
		}
		parsed.Host = bucket + "." + parsed.Host
		baseURL = strings.TrimRight(parsed.String(), "/")
	}
	return &OSSStorage{
		client: client, bucket: bucket,
		objectPrefix: strings.Trim(strings.TrimSpace(config.ObjectPrefix), "/"), baseURL: baseURL,
	}, nil
}

func (s *OSSStorage) Store(ctx context.Context, objectKey, sourcePath, contentType string) (*StoredFile, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	key := strings.TrimLeft(filepath.ToSlash(objectKey), "/")
	if s.objectPrefix != "" {
		key = s.objectPrefix + "/" + key
	}
	if strings.Contains(key, "../") || key == "" {
		return nil, uploadError(ErrInvalidRequest, "invalid OSS object key")
	}
	request := &oss.PutObjectRequest{
		Bucket: oss.Ptr(s.bucket),
		Key:    oss.Ptr(key),
	}
	if contentType != "" {
		request.ContentType = oss.Ptr(contentType)
	}
	if _, err := s.client.PutObjectFromFile(ctx, request, sourcePath); err != nil {
		return nil, fmt.Errorf("upload to Aliyun OSS: %w", err)
	}
	return &StoredFile{Provider: StorageAliyunOSS, Path: key, URL: joinPublicURL(s.baseURL, key)}, nil
}

func safeStoragePath(root, objectKey string) (string, error) {
	cleanKey := filepath.Clean(filepath.FromSlash(strings.TrimLeft(objectKey, "/")))
	if cleanKey == "." || cleanKey == "" || strings.HasPrefix(cleanKey, "..") {
		return "", uploadError(ErrInvalidRequest, "invalid storage object key")
	}
	target := filepath.Join(root, cleanKey)
	relative, err := filepath.Rel(root, target)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", uploadError(ErrInvalidRequest, "storage path escapes root directory")
	}
	return target, nil
}

func joinPublicURL(baseURL, objectKey string) string {
	if baseURL == "" {
		return filepath.ToSlash(objectKey)
	}
	parts := strings.Split(filepath.ToSlash(objectKey), "/")
	for i := range parts {
		parts[i] = url.PathEscape(parts[i])
	}
	return strings.TrimRight(baseURL, "/") + "/" + strings.Join(parts, "/")
}

func copyWithContext(ctx context.Context, dst io.Writer, src io.Reader) (int64, error) {
	buffer := make([]byte, 128<<10)
	var total int64
	for {
		if err := ctx.Err(); err != nil {
			return total, err
		}
		read, readErr := src.Read(buffer)
		if read > 0 {
			written, writeErr := dst.Write(buffer[:read])
			total += int64(written)
			if writeErr != nil {
				return total, writeErr
			}
			if written != read {
				return total, io.ErrShortWrite
			}
		}
		if readErr != nil {
			if readErr == io.EOF {
				return total, nil
			}
			return total, readErr
		}
	}
}

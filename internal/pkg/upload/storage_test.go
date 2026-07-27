package upload

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
)

type recordingOSSUploader struct {
	request  *oss.PutObjectRequest
	filePath string
	err      error
}

func (u *recordingOSSUploader) PutObjectFromFile(
	_ context.Context,
	request *oss.PutObjectRequest,
	filePath string,
	_ ...func(*oss.Options),
) (*oss.PutObjectResult, error) {
	u.request = request
	u.filePath = filePath
	if u.err != nil {
		return nil, u.err
	}
	return &oss.PutObjectResult{}, nil
}

func TestNewOSSStorageRequiresV2Region(t *testing.T) {
	config := OSSConfig{
		Endpoint: "oss-cn-hangzhou.aliyuncs.com", AccessKeyID: "access-key-id",
		AccessKeySecret: "access-key-secret", Bucket: "example-bucket",
	}
	if _, err := NewOSSStorage(config); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("NewOSSStorage() without region error = %v, want invalid request", err)
	}

	config.Region = "cn-hangzhou"
	storage, err := NewOSSStorage(config)
	if err != nil {
		t.Fatal(err)
	}
	if storage.baseURL != "https://example-bucket.oss-cn-hangzhou.aliyuncs.com" {
		t.Fatalf("base URL = %q", storage.baseURL)
	}
}

func TestOSSStorageStoreUsesV2PutObjectRequest(t *testing.T) {
	sourcePath := filepath.Join(t.TempDir(), "source.png")
	if err := os.WriteFile(sourcePath, []byte("image"), 0o600); err != nil {
		t.Fatal(err)
	}
	uploader := &recordingOSSUploader{}
	storage := &OSSStorage{
		client: uploader, bucket: "example-bucket", objectPrefix: "uploads",
		baseURL: "https://cdn.example.com",
	}

	stored, err := storage.Store(context.Background(), "images/a b.png", sourcePath, "image/png")
	if err != nil {
		t.Fatal(err)
	}
	if uploader.request == nil {
		t.Fatal("PutObjectFromFile was not called")
	}
	if got := oss.ToString(uploader.request.Bucket); got != "example-bucket" {
		t.Fatalf("request bucket = %q", got)
	}
	if got := oss.ToString(uploader.request.Key); got != "uploads/images/a b.png" {
		t.Fatalf("request key = %q", got)
	}
	if got := oss.ToString(uploader.request.ContentType); got != "image/png" {
		t.Fatalf("request content type = %q", got)
	}
	if uploader.filePath != sourcePath {
		t.Fatalf("source path = %q", uploader.filePath)
	}
	if stored.Provider != StorageAliyunOSS || stored.Path != "uploads/images/a b.png" ||
		stored.URL != "https://cdn.example.com/uploads/images/a%20b.png" {
		t.Fatalf("stored file = %+v", stored)
	}
}

func TestOSSStorageStoreWrapsV2UploadError(t *testing.T) {
	wantErr := errors.New("put object failed")
	storage := &OSSStorage{
		client: &recordingOSSUploader{err: wantErr}, bucket: "example-bucket",
	}
	_, err := storage.Store(context.Background(), "images/a.png", "missing.png", "image/png")
	if !errors.Is(err, wantErr) {
		t.Fatalf("Store() error = %v, want wrapped upload error", err)
	}
}

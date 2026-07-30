package upload

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"testing"
	"time"
)

func directUploadTestConfig() DirectUploadConfig {
	defaults := DefaultConfig()
	return DirectUploadConfig{
		OSS: OSSConfig{
			Region: "cn-hangzhou", Endpoint: "https://oss-cn-hangzhou.aliyuncs.com",
			AccessKeyID: "test-access-key-id", AccessKeySecret: "test-access-key-secret",
			Bucket: "example-bucket", ObjectPrefix: "uploads", BaseURL: "https://cdn.example.com",
		},
		SignatureTTL: 10 * time.Minute,
		Image:        defaults.Image,
		Video:        defaults.Video,
	}
}

func TestOSSDirectUploadSignerSignsSizeBoundPut(t *testing.T) {
	signer, err := NewOSSDirectUploadSigner(directUploadTestConfig())
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().Truncate(time.Second)
	signer.now = func() time.Time { return now }
	signer.newObjectID = func() (string, error) { return "0123456789abcdef0123456789abcdef", nil }

	credential, err := signer.Sign(context.Background(), DirectUploadRequest{
		MediaType: MediaImage, FileName: "avatar.PNG", Size: 12345, ContentType: "image/png; charset=binary",
	})
	if err != nil {
		t.Fatal(err)
	}
	if credential.Provider != StorageAliyunOSS || credential.Method != "PUT" {
		t.Fatalf("credential provider/method = %q/%q", credential.Provider, credential.Method)
	}
	wantKey := "uploads/images/" + now.Format("2006/01/02") + "/0123456789abcdef0123456789abcdef.png"
	if credential.ObjectKey != wantKey {
		t.Fatalf("object key = %q, want %q", credential.ObjectKey, wantKey)
	}
	if credential.FileURL != "https://cdn.example.com/"+wantKey {
		t.Fatalf("file URL = %q", credential.FileURL)
	}
	if !credential.ExpiresAt.Equal(now.Add(10 * time.Minute)) {
		t.Fatalf("expiration = %s", credential.ExpiresAt)
	}
	for name, expected := range map[string]string{
		"Content-Length": "12345", "Content-Type": "image/png", "x-oss-forbid-overwrite": "true",
	} {
		if actual, ok := headerValue(credential.Headers, name); !ok || actual != expected {
			t.Fatalf("signed header %s = %q, %v", name, actual, ok)
		}
	}
	parsed, err := url.Parse(credential.UploadURL)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Host != "example-bucket.oss-cn-hangzhou.aliyuncs.com" || parsed.EscapedPath() != "/"+wantKey {
		t.Fatalf("upload URL target = %s%s", parsed.Host, parsed.EscapedPath())
	}
	query := parsed.Query()
	if query.Get("x-oss-signature-version") != "OSS4-HMAC-SHA256" || query.Get("x-oss-signature") == "" {
		t.Fatalf("upload URL is not V4-signed: %s", credential.UploadURL)
	}
	if strings.Contains(credential.UploadURL, "test-access-key-secret") {
		t.Fatal("access key secret leaked into upload URL")
	}
}

func TestOSSDirectUploadSignerValidatesPolicy(t *testing.T) {
	signer, err := NewOSSDirectUploadSigner(directUploadTestConfig())
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name    string
		request DirectUploadRequest
		wantErr error
	}{
		{name: "unknown media", request: DirectUploadRequest{MediaType: "audio", FileName: "a.mp3", Size: 1, ContentType: "audio/mpeg"}, wantErr: ErrUnsupportedType},
		{name: "wrong extension", request: DirectUploadRequest{MediaType: MediaImage, FileName: "a.exe", Size: 1, ContentType: "image/png"}, wantErr: ErrUnsupportedType},
		{name: "wrong mime", request: DirectUploadRequest{MediaType: MediaImage, FileName: "a.png", Size: 1, ContentType: "text/plain"}, wantErr: ErrUnsupportedType},
		{name: "too large", request: DirectUploadRequest{MediaType: MediaImage, FileName: "a.png", Size: (20 << 20) + 1, ContentType: "image/png"}, wantErr: ErrFileTooLarge},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := signer.Sign(context.Background(), test.request)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("Sign() error = %v, want %v", err, test.wantErr)
			}
		})
	}
}

func TestNewOSSDirectUploadSignerRejectsUnsafeTTL(t *testing.T) {
	config := directUploadTestConfig()
	config.SignatureTTL = 59 * time.Second
	if _, err := NewOSSDirectUploadSigner(config); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("short TTL error = %v", err)
	}
	config.SignatureTTL = time.Hour + time.Second
	if _, err := NewOSSDirectUploadSigner(config); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("long TTL error = %v", err)
	}
}

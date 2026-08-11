package service

import (
	"bytes"
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"ai-video/internal/pkg/upload"
)

type configFileTestStorage struct {
	objectKey   string
	contentType string
	content     []byte
	err         error
}

func (s *configFileTestStorage) Store(_ context.Context, objectKey, sourcePath, contentType string) (*upload.StoredFile, error) {
	if s.err != nil {
		return nil, s.err
	}
	content, err := os.ReadFile(sourcePath)
	if err != nil {
		return nil, err
	}
	s.objectKey = objectKey
	s.contentType = contentType
	s.content = content
	return &upload.StoredFile{Provider: upload.StorageAliyunOSS, Path: "uploads/" + objectKey, URL: "/uploads/" + objectKey}, nil
}

func TestConfigFileServiceStoresValidatedHTML(t *testing.T) {
	storage := &configFileTestStorage{}
	svc := NewConfigFileService(storage)
	svc.now = func() time.Time { return time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC) }
	svc.newID = func() string { return "file-id" }
	content := []byte("<!doctype html><html><body>Privacy</body></html>")

	result, err := svc.Store(context.Background(), "app.privacy_policy", `C:\fakepath\privacy.html`, bytes.NewReader(content))
	if err != nil {
		t.Fatal(err)
	}
	if storage.objectKey != "config-files/app/privacy_policy/2026/08/file-id.html" {
		t.Fatalf("object key = %q", storage.objectKey)
	}
	if storage.contentType != "text/html" || !bytes.Equal(storage.content, content) {
		t.Fatalf("stored type/content = %q/%q", storage.contentType, storage.content)
	}
	if result.OriginalName != "privacy.html" || result.FileURL != "/uploads/"+storage.objectKey {
		t.Fatalf("result = %+v", result)
	}
}

func TestConfigFileServiceRejectsInvalidFiles(t *testing.T) {
	svc := NewConfigFileService(&configFileTestStorage{})
	tests := []struct {
		name        string
		configKey   string
		fileName    string
		content     []byte
		targetError error
	}{
		{name: "invalid key", configKey: "../privacy", fileName: "privacy.txt", content: []byte("privacy"), targetError: ErrInvalidConfigFile},
		{name: "unsupported extension", configKey: "app.privacy_policy", fileName: "privacy.exe", content: []byte("MZ"), targetError: ErrUnsupportedConfigFile},
		{name: "binary renamed html", configKey: "app.privacy_policy", fileName: "privacy.html", content: []byte{0, 1, 2, 3}, targetError: ErrUnsupportedConfigFile},
		{name: "fake pdf", configKey: "app.terms", fileName: "terms.pdf", content: []byte("not a pdf"), targetError: ErrUnsupportedConfigFile},
		{name: "empty", configKey: "app.faq", fileName: "faq.txt", content: nil, targetError: ErrInvalidConfigFile},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.Store(context.Background(), tt.configKey, tt.fileName, bytes.NewReader(tt.content))
			if !errors.Is(err, tt.targetError) {
				t.Fatalf("error = %v, want %v", err, tt.targetError)
			}
		})
	}
}

func TestConfigFileServiceRejectsOversizedFile(t *testing.T) {
	svc := NewConfigFileService(&configFileTestStorage{})
	_, err := svc.Store(context.Background(), "app.faq", "faq.txt", strings.NewReader(strings.Repeat("a", int(ConfigFileMaxSize+1))))
	if !errors.Is(err, ErrConfigFileTooLarge) {
		t.Fatalf("error = %v, want ErrConfigFileTooLarge", err)
	}
}

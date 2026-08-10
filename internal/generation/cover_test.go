package generation

import (
	"bytes"
	"context"
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"

	"ai-video/internal/config"
	"ai-video/internal/gen/model"
	"ai-video/internal/pkg/upload"
)

type taskCoverCaptureStorage struct {
	objectKey   string
	contentType string
	contents    []byte
}

func (s *taskCoverCaptureStorage) Store(_ context.Context, objectKey, sourcePath, contentType string) (*upload.StoredFile, error) {
	contents, err := os.ReadFile(sourcePath)
	if err != nil {
		return nil, err
	}
	s.objectKey = objectKey
	s.contentType = contentType
	s.contents = contents
	return &upload.StoredFile{Provider: upload.StorageLocal, Path: objectKey, URL: "/uploads/" + objectKey}, nil
}

func TestGenerateImageTaskCoverOrOriginalFallsBackToOriginal(t *testing.T) {
	task := &model.VideoUserGenerationTask{ID: 7, UserID: 19, TaskCode: "image-task"}
	originalURL := "/uploads/generated/19/task-image-task-1.png"

	coverURL, err := generateImageTaskCoverOrOriginal(
		context.Background(), nil, task, "https://cdn.example.com/generated.png", originalURL,
	)
	if err == nil {
		t.Fatal("generateImageTaskCoverOrOriginal succeeded without cover storage")
	}
	if coverURL != originalURL {
		t.Fatalf("cover URL = %q, want original image %q", coverURL, originalURL)
	}
}

func TestGenerateOrStoreDefaultVideoTaskCoverFallsBackToDefaultImage(t *testing.T) {
	originalLocalRoot := config.Cfg.Upload.LocalRootDir
	config.Cfg.Upload.LocalRootDir = t.TempDir()
	t.Cleanup(func() { config.Cfg.Upload.LocalRootDir = originalLocalRoot })

	storage := &taskCoverCaptureStorage{}
	task := &model.VideoUserGenerationTask{ID: 8, UserID: 23, TaskCode: "video-task"}
	coverURL, err := generateOrStoreDefaultVideoTaskCover(
		context.Background(), storage, task, filepath.Join(t.TempDir(), "missing.mp4"),
	)
	if err != nil {
		t.Fatal(err)
	}
	if coverURL != "/uploads/generated/23/task-video-task-cover.jpg" {
		t.Fatalf("cover URL = %q", coverURL)
	}
	if storage.objectKey != "generated/23/task-video-task-cover.jpg" {
		t.Fatalf("stored object key = %q", storage.objectKey)
	}
	if storage.contentType != "image/jpeg" {
		t.Fatalf("stored content type = %q", storage.contentType)
	}

	decoded, err := jpeg.Decode(bytes.NewReader(storage.contents))
	if err != nil {
		t.Fatalf("decode stored default cover: %v", err)
	}
	if decoded.Bounds().Dx() != defaultVideoCoverWidth || decoded.Bounds().Dy() != defaultVideoCoverHeight {
		t.Fatalf("default cover dimensions = %dx%d", decoded.Bounds().Dx(), decoded.Bounds().Dy())
	}
	centerRed, _, _, _ := decoded.At(defaultVideoCoverWidth/2, defaultVideoCoverHeight/2).RGBA()
	cornerRed, _, _, _ := decoded.At(0, 0).RGBA()
	if centerRed <= cornerRed {
		t.Fatal("default cover does not contain the expected centered play symbol")
	}
}

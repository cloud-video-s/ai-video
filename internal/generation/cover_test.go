package generation

import (
	"bytes"
	"context"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"ai-video/internal/config"
	"ai-video/internal/gen/model"
	"ai-video/internal/pkg/upload"
)

type coverRoundTripFunc func(*http.Request) (*http.Response, error)

func (f coverRoundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

type coverStoredFile struct {
	objectKey   string
	contentType string
	contents    []byte
}

type coverRecordingStorage struct {
	files []coverStoredFile
}

func (s *coverRecordingStorage) Store(
	ctx context.Context,
	objectKey, sourcePath, contentType string,
) (*upload.StoredFile, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	contents, err := os.ReadFile(sourcePath)
	if err != nil {
		return nil, err
	}
	s.files = append(s.files, coverStoredFile{
		objectKey: objectKey, contentType: contentType, contents: contents,
	})
	return &upload.StoredFile{
		Provider: upload.StorageAliyunOSS,
		Path:     "uploads/" + objectKey,
		URL:      "https://cdn.example.com/uploads/" + objectKey,
	}, nil
}

func TestGenerateAndStoreImageTaskCover(t *testing.T) {
	stagingRoot := configureCoverTestStaging(t)
	var source bytes.Buffer
	if err := png.Encode(&source, image.NewRGBA(image.Rect(0, 0, 80, 40))); err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Transport: coverRoundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.URL.String() != "https://media.example.com/result.png" {
			t.Fatalf("image cover source URL = %q", request.URL.String())
		}
		return coverTestResponse(source.Bytes(), "image/png"), nil
	})}
	storage := &coverRecordingStorage{}
	options := upload.DefaultMediaPreviewOptions()
	options.MaxWidth, options.MaxHeight = 40, 40
	options.RemoteHTTPClient = client
	coverURL, err := generateAndStoreTaskCoverWithOptions(
		context.Background(), storage,
		&model.VideoUserGenerationTask{ID: 23, UserID: 17},
		upload.MediaImage, "https://media.example.com/result.png", options,
	)
	if err != nil {
		t.Fatal(err)
	}
	assertStoredTaskCover(t, storage, coverURL, "generated/17/task-23-cover.jpg", 40, 20)
	assertCoverStagingEmpty(t, stagingRoot)
}

func TestGenerateAndStoreVideoTaskCoverUsesOSSSnapshot(t *testing.T) {
	stagingRoot := configureCoverTestStaging(t)
	var snapshot bytes.Buffer
	if err := jpeg.Encode(&snapshot, image.NewRGBA(image.Rect(0, 0, 640, 360)), &jpeg.Options{Quality: 80}); err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Transport: coverRoundTripFunc(func(request *http.Request) (*http.Response, error) {
		if process := request.URL.Query().Get("x-oss-process"); process != "video/snapshot,t_0,f_jpg,m_fast" {
			t.Fatalf("OSS video snapshot process = %q", process)
		}
		return coverTestResponse(snapshot.Bytes(), "image/jpeg"), nil
	})}
	storage := &coverRecordingStorage{}
	options := upload.DefaultMediaPreviewOptions()
	options.MaxWidth, options.MaxHeight = 320, 180
	options.RemoteHTTPClient = client
	coverURL, err := generateAndStoreTaskCoverWithOptions(
		context.Background(), storage,
		&model.VideoUserGenerationTask{ID: 11, UserID: 9},
		upload.MediaVideo, "https://media.example.com/result.mp4", options,
	)
	if err != nil {
		t.Fatal(err)
	}
	assertStoredTaskCover(t, storage, coverURL, "generated/9/task-11-cover.jpg", 320, 180)
	assertCoverStagingEmpty(t, stagingRoot)
}

func configureCoverTestStaging(t *testing.T) string {
	t.Helper()
	originalUploadConfig := config.Cfg.Upload
	t.Cleanup(func() { config.Cfg.Upload = originalUploadConfig })
	stagingRoot := filepath.Join(t.TempDir(), "staging")
	config.Cfg.Upload.RootDir = stagingRoot
	return stagingRoot
}

func coverTestResponse(contents []byte, contentType string) *http.Response {
	return &http.Response{
		StatusCode:    http.StatusOK,
		Body:          io.NopCloser(bytes.NewReader(contents)),
		Header:        http.Header{"Content-Type": []string{contentType}},
		ContentLength: int64(len(contents)),
	}
}

func assertStoredTaskCover(
	t *testing.T,
	storage *coverRecordingStorage,
	coverURL, objectKey string,
	wantWidth, wantHeight int,
) {
	t.Helper()
	wantURL := "https://cdn.example.com/uploads/" + objectKey
	if coverURL != wantURL {
		t.Fatalf("cover URL = %q, want %q", coverURL, wantURL)
	}
	if len(storage.files) != 1 {
		t.Fatalf("stored cover count = %d", len(storage.files))
	}
	stored := storage.files[0]
	if stored.objectKey != objectKey || stored.contentType != "image/jpeg" {
		t.Fatalf("stored cover metadata = %+v", stored)
	}
	config, format, err := image.DecodeConfig(bytes.NewReader(stored.contents))
	if err != nil {
		t.Fatal(err)
	}
	if format != "jpeg" || config.Width != wantWidth || config.Height != wantHeight {
		t.Fatalf("stored cover = %s %dx%d, want jpeg %dx%d", format, config.Width, config.Height, wantWidth, wantHeight)
	}
}

func assertCoverStagingEmpty(t *testing.T, stagingRoot string) {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(stagingRoot, ".generated"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("cover staging entries = %v", entries)
	}
}

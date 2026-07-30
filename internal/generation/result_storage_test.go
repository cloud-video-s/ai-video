package generation

import (
	"bytes"
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"ai-video/internal/config"
	"ai-video/internal/gen/model"
	"ai-video/internal/pkg/upload"
)

type resultRoundTripFunc func(*http.Request) (*http.Response, error)

func (f resultRoundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

type recordedGeneratedFile struct {
	objectKey   string
	contentType string
	contents    []byte
}

type recordingGeneratedStorage struct {
	files []recordedGeneratedFile
}

func (s *recordingGeneratedStorage) Store(
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
	s.files = append(s.files, recordedGeneratedFile{
		objectKey: objectKey, contentType: contentType, contents: contents,
	})
	return &upload.StoredFile{
		Provider: upload.StorageAliyunOSS,
		Path:     "uploads/" + objectKey,
		URL:      "https://cdn.example.com/uploads/" + objectKey,
	}, nil
}

func TestDownloadVideosStoresResultInLocalUploadStorage(t *testing.T) {
	originalUploadConfig := config.Cfg.Upload
	t.Cleanup(func() { config.Cfg.Upload = originalUploadConfig })
	stagingRoot := filepath.Join(t.TempDir(), "staging")
	finalRoot := filepath.Join(t.TempDir(), "files")
	config.Cfg.Upload.RootDir = stagingRoot
	config.Cfg.Upload.VideoMaxFileSize = 1024

	storage, err := upload.NewLocalStorage(finalRoot, "/uploads")
	if err != nil {
		t.Fatal(err)
	}
	payload := []byte("generated-video")
	client := &http.Client{Transport: resultRoundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.URL.String() != "https://media.example.com/result.mp4" {
			t.Fatalf("download URL = %q", request.URL.String())
		}
		return &http.Response{
			StatusCode:    http.StatusOK,
			Body:          io.NopCloser(bytes.NewReader(payload)),
			Header:        make(http.Header),
			ContentLength: int64(len(payload)),
		}, nil
	})}

	urls, err := downloadVideosToStorage(context.Background(), storage, client, &model.VideoUserGenerationTask{
		ID: 11, UserID: 9,
	}, []string{"https://media.example.com/result.mp4"})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(urls, []string{"/uploads/generated/9/task-11-1.mp4"}) {
		t.Fatalf("stored URLs = %#v", urls)
	}
	stored, err := os.ReadFile(filepath.Join(finalRoot, "generated", "9", "task-11-1.mp4"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(stored, payload) {
		t.Fatalf("stored contents = %q", stored)
	}
	assertGeneratedStagingEmpty(t, stagingRoot)
}

func TestSaveBase64ImagesStoresResultThroughAliyunUploadStorage(t *testing.T) {
	originalUploadConfig := config.Cfg.Upload
	t.Cleanup(func() { config.Cfg.Upload = originalUploadConfig })
	stagingRoot := filepath.Join(t.TempDir(), "staging")
	config.Cfg.Upload.RootDir = stagingRoot
	config.Cfg.Upload.ImageMaxFileSize = 1024

	png := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 0, 0, 0, 0}
	storage := &recordingGeneratedStorage{}
	urls, err := saveBase64Images(
		context.Background(),
		storage,
		&model.VideoUserGenerationTask{ID: 23, UserID: 17},
		[]string{"data:image/png;base64," + base64.StdEncoding.EncodeToString(png)},
		2,
	)
	if err != nil {
		t.Fatal(err)
	}
	wantURL := "https://cdn.example.com/uploads/generated/17/task-23-3.png"
	if !reflect.DeepEqual(urls, []string{wantURL}) {
		t.Fatalf("stored URLs = %#v", urls)
	}
	if len(storage.files) != 1 {
		t.Fatalf("stored file count = %d", len(storage.files))
	}
	stored := storage.files[0]
	if stored.objectKey != "generated/17/task-23-3.png" || stored.contentType != "image/png" {
		t.Fatalf("stored metadata = %+v", stored)
	}
	if !bytes.Equal(stored.contents, png) {
		t.Fatalf("stored contents = %v", stored.contents)
	}
	assertGeneratedStagingEmpty(t, stagingRoot)
}

func assertGeneratedStagingEmpty(t *testing.T, stagingRoot string) {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(stagingRoot, ".generated"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("generated staging entries = %v", entries)
	}
}

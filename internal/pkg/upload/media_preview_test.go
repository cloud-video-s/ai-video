package upload

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

type previewRoundTripFunc func(*http.Request) (*http.Response, error)

func (f previewRoundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestDownloadPreviewSourceUsesRequestedTemporaryDirectory(t *testing.T) {
	temporaryDirectory := filepath.Join(t.TempDir(), "storage", "uploads", "files", "tmp")
	client := &http.Client{Transport: previewRoundTripFunc(func(*http.Request) (*http.Response, error) {
		contents := []byte("preview contents")
		return &http.Response{
			StatusCode:    http.StatusOK,
			Status:        "200 OK",
			Body:          io.NopCloser(bytes.NewReader(contents)),
			ContentLength: int64(len(contents)),
		}, nil
	})}
	remoteURL, err := url.Parse("https://media.example.com/result.jpeg")
	if err != nil {
		t.Fatal(err)
	}

	temporaryPath, cleanup, err := downloadPreviewSource(
		context.Background(), client, remoteURL, 1024, temporaryDirectory,
	)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()

	if filepath.Dir(temporaryPath) != temporaryDirectory {
		t.Fatalf("temporary directory = %q, want %q", filepath.Dir(temporaryPath), temporaryDirectory)
	}
	if _, err := os.Stat(temporaryPath); err != nil {
		t.Fatalf("temporary preview file: %v", err)
	}
	cleanup()
	if _, err := os.Stat(temporaryPath); !os.IsNotExist(err) {
		t.Fatalf("temporary preview file still exists after cleanup: %v", err)
	}
}

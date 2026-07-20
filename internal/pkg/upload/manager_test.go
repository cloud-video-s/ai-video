package upload

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestImageUploadSupportsChunksAndResume(t *testing.T) {
	manager := newTestManager(t)
	content := append([]byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a}, []byte("image-payload-data")...)
	digest := sha256.Sum256(content)
	sessions, err := manager.InitiateImages(context.Background(), []FileSpec{
		{FileName: "sample.png", Size: int64(len(content)), ContentType: "image/png", SHA256: hex.EncodeToString(digest[:])},
	})
	if err != nil {
		t.Fatal(err)
	}
	session := sessions[0]
	if session.TotalChunks != 4 {
		t.Fatalf("total chunks = %d, want 4", session.TotalChunks)
	}

	putTestChunk(t, manager, session, content, 3)
	putTestChunk(t, manager, session, content, 0)
	status, err := manager.Status(context.Background(), session.UploadID)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(status.UploadedChunks, []int{0, 3}) {
		t.Fatalf("uploaded chunks = %v", status.UploadedChunks)
	}
	if _, err := manager.Complete(context.Background(), session.UploadID); !errors.Is(err, ErrMissingChunks) {
		t.Fatalf("complete with missing chunks error = %v", err)
	}

	putTestChunk(t, manager, session, content, 1)
	putTestChunk(t, manager, session, content, 2)
	putTestChunk(t, manager, session, content, 2) // Retries are idempotent.
	completed, err := manager.Complete(context.Background(), session.UploadID)
	if err != nil {
		t.Fatal(err)
	}
	if !completed.Completed || !strings.HasPrefix(completed.FilePath, "images/") {
		t.Fatalf("unexpected completed session: %+v", completed)
	}
	stored, err := os.ReadFile(filepath.Join(manager.root, filepath.FromSlash(completed.FilePath)))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(stored, content) {
		t.Fatal("stored image differs from uploaded content")
	}

	restarted, err := NewManager(manager.config)
	if err != nil {
		t.Fatal(err)
	}
	restored, err := restarted.Status(context.Background(), session.UploadID)
	if err != nil || !restored.Completed {
		t.Fatalf("restored session = %+v, err = %v", restored, err)
	}
}

func TestVideoUploadUsesSeparatePolicyAndDirectory(t *testing.T) {
	manager := newTestManager(t)
	content := append([]byte{0, 0, 0, 20, 'f', 't', 'y', 'p', 'i', 's', 'o', 'm'}, []byte("video-data")...)
	sessions, err := manager.InitiateVideos(context.Background(), []FileSpec{
		{FileName: "clip.mp4", Size: int64(len(content)), ContentType: "video/mp4"},
	})
	if err != nil {
		t.Fatal(err)
	}
	session := sessions[0]
	for index := 0; index < session.TotalChunks; index++ {
		putTestChunk(t, manager, session, content, index)
	}
	completed, err := manager.Complete(context.Background(), session.UploadID)
	if err != nil {
		t.Fatal(err)
	}
	if completed.ContentType != "video/mp4" || !strings.HasPrefix(completed.FilePath, "videos/") {
		t.Fatalf("unexpected video result: %+v", completed)
	}
}

func TestUploadLimitsAndContentValidation(t *testing.T) {
	manager := newTestManager(t)
	_, err := manager.InitiateImages(context.Background(), []FileSpec{{FileName: "bad.exe", Size: 4}})
	if !errors.Is(err, ErrUnsupportedType) {
		t.Fatalf("extension error = %v", err)
	}
	_, err = manager.InitiateImages(context.Background(), []FileSpec{{FileName: "large.png", Size: 101}})
	if !errors.Is(err, ErrFileTooLarge) {
		t.Fatalf("size error = %v", err)
	}
	files := make([]FileSpec, 4)
	for i := range files {
		files[i] = FileSpec{FileName: "batch.png", Size: 8}
	}
	if _, err := manager.InitiateImages(context.Background(), files); !errors.Is(err, ErrBatchTooLarge) {
		t.Fatalf("batch error = %v", err)
	}

	sessions, err := manager.InitiateImages(context.Background(), []FileSpec{{FileName: "fake.png", Size: 8}})
	if err != nil {
		t.Fatal(err)
	}
	session := sessions[0]
	if _, err := manager.PutChunk(context.Background(), session.UploadID, 0, bytes.NewReader([]byte("notimage")), ""); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Complete(context.Background(), session.UploadID); !errors.Is(err, ErrUnsupportedType) {
		t.Fatalf("content validation error = %v", err)
	}
}

func TestChunkSizeAndChecksumAreValidated(t *testing.T) {
	manager := newTestManager(t)
	sessions, err := manager.InitiateImages(context.Background(), []FileSpec{{FileName: "sample.png", Size: 9}})
	if err != nil {
		t.Fatal(err)
	}
	session := sessions[0]
	if _, err := manager.PutChunk(context.Background(), session.UploadID, 0, bytes.NewReader([]byte("short")), ""); !errors.Is(err, ErrInvalidChunk) {
		t.Fatalf("short chunk error = %v", err)
	}
	if _, err := manager.PutChunk(context.Background(), session.UploadID, 0, bytes.NewReader(make([]byte, 8)), strings.Repeat("0", 64)); !errors.Is(err, ErrChecksumMismatch) {
		t.Fatalf("chunk checksum error = %v", err)
	}
}

func TestCleanupExpiredUpload(t *testing.T) {
	manager := newTestManager(t)
	started := time.Date(2026, time.July, 16, 10, 0, 0, 0, time.UTC)
	manager.now = func() time.Time { return started }
	sessions, err := manager.InitiateImages(context.Background(), []FileSpec{{FileName: "old.png", Size: 8}})
	if err != nil {
		t.Fatal(err)
	}
	manager.now = func() time.Time { return started.Add(2 * time.Hour) }
	removed, err := manager.CleanupExpired(context.Background())
	if err != nil || removed != 1 {
		t.Fatalf("cleanup removed = %d, err = %v", removed, err)
	}
	if _, err := manager.Status(context.Background(), sessions[0].UploadID); !errors.Is(err, ErrUploadNotFound) {
		t.Fatalf("status after cleanup error = %v", err)
	}
}

func TestHTTPHandlerRestrictsUploadSessionToOwner(t *testing.T) {
	manager := newTestManager(t)
	owner := UploadOwner{Type: UploaderAPIUser, ID: 42}
	sessions, err := manager.InitiateBatchForOwner(context.Background(), MediaImage, []FileSpec{
		{FileName: "owned.png", Size: 8},
	}, owner)
	if err != nil {
		t.Fatal(err)
	}

	handler := NewHTTPHandler(manager, WithCompletionRecording(nil, func(c *gin.Context) (UploadOwner, error) {
		return UploadOwner{Type: UploaderAPIUser, ID: c.GetUint64("test_owner_id")}, nil
	}))
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest("GET", "/", nil)
	ctx.Set("test_owner_id", uint64(7))
	if _, _, err := handler.requireAccess(ctx, sessions[0].UploadID, MediaImage); !errors.Is(err, ErrUploadNotFound) {
		t.Fatalf("other owner access error = %v, want upload not found", err)
	}
	ctx.Set("test_owner_id", owner.ID)
	if _, _, err := handler.requireAccess(ctx, sessions[0].UploadID, MediaImage); err != nil {
		t.Fatalf("session owner access failed: %v", err)
	}
}

func newTestManager(t *testing.T) *Manager {
	t.Helper()
	config := Config{
		RootDir: t.TempDir(), ChunkSize: 8, MaxBatchFiles: 3, SessionTTL: time.Hour,
		Image: Policy{MaxFileSize: 100, AllowedExts: []string{".png", ".jpg"}, AllowedMIMETypes: []string{"image/png", "image/jpeg"}},
		Video: Policy{MaxFileSize: 100, AllowedExts: []string{".mp4", ".webm"}, AllowedMIMETypes: []string{"video/mp4", "video/webm"}},
	}
	manager, err := NewManager(config)
	if err != nil {
		t.Fatal(err)
	}
	return manager
}

func putTestChunk(t *testing.T, manager *Manager, session Session, content []byte, index int) {
	t.Helper()
	start := int64(index) * session.ChunkSize
	end := start + expectedChunkSize(&session, index)
	if _, err := manager.PutChunk(context.Background(), session.UploadID, index, bytes.NewReader(content[start:end]), ""); err != nil {
		t.Fatalf("put chunk %d: %v", index, err)
	}
}

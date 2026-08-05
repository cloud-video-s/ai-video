package generation

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"ai-video/internal/gen/model"
	"ai-video/internal/pkg/upload"
)

type recordingTestStorage struct{}

func (recordingTestStorage) Store(_ context.Context, objectKey, _, _ string) (*upload.StoredFile, error) {
	return &upload.StoredFile{
		Provider: upload.StorageAliyunOSS,
		Path:     "uploads/" + filepath.ToSlash(objectKey),
		URL:      "/uploads/" + filepath.ToSlash(objectKey),
	}, nil
}

type recordingTestRecorder struct {
	records []upload.StoredUpload
}

func (r *recordingTestRecorder) RecordStored(_ context.Context, item upload.StoredUpload) error {
	r.records = append(r.records, item)
	return nil
}

func TestGeneratedRecordingStorageRecordsCompletedBackendUpload(t *testing.T) {
	source := filepath.Join(t.TempDir(), "result.mp4")
	if err := os.WriteFile(source, []byte("generated-video"), 0o600); err != nil {
		t.Fatal(err)
	}
	recorder := &recordingTestRecorder{}
	storage := recordGeneratedUploads(recordingTestStorage{}, recorder, &model.VideoUserGenerationTask{
		ID: 9, UserID: 19,
	}, upload.MediaVideo)
	stored, err := storage.Store(context.Background(), "generated/19/result.mp4", source, "video/mp4")
	if err != nil {
		t.Fatal(err)
	}
	if stored.URL != "/uploads/generated/19/result.mp4" || len(recorder.records) != 1 {
		t.Fatalf("stored = %+v, records = %+v", stored, recorder.records)
	}
	record := recorder.records[0]
	if record.Owner.Type != upload.UploaderAPIUser || record.Owner.ID != 19 ||
		record.Kind != upload.MediaVideo || record.FileSize != int64(len("generated-video")) || len(record.SHA256) != 64 {
		t.Fatalf("record = %+v", record)
	}
}

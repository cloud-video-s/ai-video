package upload

import (
	"context"
	"errors"
	"fmt"
	"time"
)

type MediaKind string

const (
	MediaImage MediaKind = "image"
	MediaVideo MediaKind = "video"
)

var (
	ErrInvalidRequest     = errors.New("invalid upload request")
	ErrUnsupportedType    = errors.New("unsupported file type")
	ErrFileTooLarge       = errors.New("file exceeds size limit")
	ErrBatchTooLarge      = errors.New("too many files in batch")
	ErrUploadNotFound     = errors.New("upload session not found")
	ErrUploadExpired      = errors.New("upload session expired")
	ErrInvalidChunk       = errors.New("invalid upload chunk")
	ErrMissingChunks      = errors.New("upload chunks are incomplete")
	ErrChecksumMismatch   = errors.New("checksum mismatch")
	ErrUploadKindMismatch = errors.New("upload media kind mismatch")
)

type Policy struct {
	MaxFileSize      int64
	AllowedExts      []string
	AllowedMIMETypes []string
}

type Config struct {
	RootDir        string
	ChunkSize      int64
	MaxBatchFiles  int
	SessionTTL     time.Duration
	Image          Policy
	Video          Policy
	PolicyResolver func(MediaKind) (Policy, error)
	Storage        Storage
}

func DefaultConfig() Config {
	return Config{
		RootDir:       "storage/uploads",
		ChunkSize:     5 << 20,
		MaxBatchFiles: 20,
		SessionTTL:    24 * time.Hour,
		Image: Policy{
			MaxFileSize:      20 << 20,
			AllowedExts:      []string{".jpg", ".jpeg", ".png", ".gif", ".webp"},
			AllowedMIMETypes: []string{"image/jpeg", "image/png", "image/gif", "image/webp"},
		},
		Video: Policy{
			MaxFileSize:      2 << 30,
			AllowedExts:      []string{".mp4", ".mov", ".webm", ".mkv"},
			AllowedMIMETypes: []string{"video/mp4", "video/quicktime", "video/webm", "video/x-matroska"},
		},
	}
}

type FileSpec struct {
	FileName    string `json:"file_name" binding:"required"`
	Size        int64  `json:"size" binding:"required,gt=0"`
	ContentType string `json:"content_type"`
	SHA256      string `json:"sha256"`
}

type Session struct {
	UploadID        string       `json:"upload_id"`
	Kind            MediaKind    `json:"kind"`
	OriginalName    string       `json:"original_name"`
	Extension       string       `json:"extension"`
	ContentType     string       `json:"content_type,omitempty"`
	TotalSize       int64        `json:"total_size"`
	ChunkSize       int64        `json:"chunk_size"`
	TotalChunks     int          `json:"total_chunks"`
	UploadedChunks  []int        `json:"uploaded_chunks"`
	ExpectedSHA256  string       `json:"expected_sha256,omitempty"`
	SHA256          string       `json:"sha256,omitempty"`
	UploaderType    UploaderType `json:"uploader_type,omitempty"`
	UploaderID      uint64       `json:"uploader_id,omitempty"`
	StorageProvider string       `json:"storage_provider,omitempty"`
	Completed       bool         `json:"completed"`
	FilePath        string       `json:"file_path,omitempty"`
	FileURL         string       `json:"file_url,omitempty"`
	CreatedAt       time.Time    `json:"created_at"`
	ExpiresAt       time.Time    `json:"expires_at"`
}

type UploaderType string

const (
	UploaderAdmin   UploaderType = "admin"
	UploaderAPIUser UploaderType = "api_user"
)

type UploadOwner struct {
	Type UploaderType
	ID   uint64
}

type CompletedUpload struct {
	Owner   UploadOwner
	Session Session
}

type CompletionRecorder interface {
	RecordCompleted(ctx context.Context, upload CompletedUpload) error
}

type UploadError struct {
	Err     error
	Message string
}

func (e *UploadError) Error() string {
	if e.Message == "" {
		return e.Err.Error()
	}
	return fmt.Sprintf("%s: %s", e.Err, e.Message)
}

func (e *UploadError) Unwrap() error { return e.Err }

func uploadError(err error, format string, args ...any) error {
	return &UploadError{Err: err, Message: fmt.Sprintf(format, args...)}
}

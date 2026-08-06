package repository

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"ai-video/internal/domain"
	"ai-video/internal/gen/model"
	"ai-video/internal/pkg/upload"

	"gorm.io/gorm"
)

type UploadRepo struct{}

// The production schema is expected to enforce a unique upload_id. Keep an
// application-side guard as well so retries remain idempotent while a lagging
// environment is waiting for the reviewed unique-index migration.
var uploadRecordMu sync.Mutex

func NewUploadRepo() *UploadRepo { return &UploadRepo{} }

type UploadListFilter struct {
	UserType        int8
	UserID          uint64
	Status          int8
	MediaType       string
	FileType        string
	StorageProvider string
	Keyword         string
}

func (r *UploadRepo) RecordCompleted(ctx context.Context, completed upload.CompletedUpload) error {
	session := completed.Session
	if session.UploaderType != completed.Owner.Type || session.UploaderID != completed.Owner.ID {
		return fmt.Errorf("upload session owner does not match completed owner")
	}
	return r.record(ctx, completed.Owner, uploadRecord{
		UploadID: session.UploadID, Kind: session.Kind, OriginalName: session.OriginalName,
		ContentType: session.ContentType, FileSize: session.TotalSize, SHA256: session.SHA256,
		StorageProvider: session.StorageProvider, FilePath: session.FilePath, FileURL: session.FileURL,
	}, domain.UploadStatusCompleted)
}

func (r *UploadRepo) RecordDirectPreUpload(ctx context.Context, pending upload.DirectPreUpload) error {
	credential := pending.Credential
	request := pending.Request
	return r.record(ctx, pending.Owner, uploadRecord{
		UploadID: credential.UploadID, Kind: request.MediaType, OriginalName: request.FileName,
		ContentType: request.ContentType, FileSize: request.Size,
		StorageProvider: credential.Provider, FilePath: credential.ObjectKey, FileURL: credential.FileURL,
	}, domain.UploadStatusIncomplete)
}

func (r *UploadRepo) RecordStored(ctx context.Context, completed upload.StoredUpload) error {
	uploadID := storedUploadID(completed.Stored.Provider, completed.Stored.Path)
	return r.record(ctx, completed.Owner, uploadRecord{
		UploadID: uploadID, Kind: completed.Kind, OriginalName: completed.OriginalName,
		ContentType: completed.ContentType, FileSize: completed.FileSize, SHA256: completed.SHA256,
		StorageProvider: completed.Stored.Provider, FilePath: completed.Stored.Path, FileURL: completed.Stored.URL,
	}, domain.UploadStatusCompleted)
}

// ConfirmUploadedByURLs marks only files owned by this uploader. Unknown URLs
// are intentionally ignored because generation requests may also reference
// enabled templates or other server-managed remote media.
func (r *UploadRepo) ConfirmUploadedByURLs(ctx context.Context, owner upload.UploadOwner, fileURLs []string) error {
	userType, err := uploadOwnerUserType(owner.Type)
	if err != nil || owner.ID == 0 {
		return fmt.Errorf("invalid upload owner")
	}
	urls := make([]string, 0, len(fileURLs))
	paths := make([]string, 0, len(fileURLs))
	seen := make(map[string]struct{}, len(fileURLs))
	for _, value := range fileURLs {
		half := upload.HalfURL(value)
		if half == "" {
			continue
		}
		if _, exists := seen[half]; exists {
			continue
		}
		seen[half] = struct{}{}
		urls = append(urls, half)
		paths = append(paths, strings.TrimPrefix(half, "/"))
	}
	if len(urls) == 0 {
		return nil
	}
	return dbFrom(ctx).Table(model.TableNameVideoUpload).
		Where("user_type = ? AND user_id = ? AND status = ?", userType, owner.ID, domain.UploadStatusIncomplete).
		Where("file_url IN ? OR file_path IN ?", urls, paths).
		Updates(map[string]any{"status": domain.UploadStatusCompleted, "updated_at": time.Now()}).Error
}

// OwnedHalfURLs returns the canonical domain-free addresses among the supplied
// values that belong to this uploader. Callers use it to avoid rewriting
// unrelated template or third-party URLs.
func (r *UploadRepo) OwnedHalfURLs(ctx context.Context, owner upload.UploadOwner, fileURLs []string) (map[string]struct{}, error) {
	userType, err := uploadOwnerUserType(owner.Type)
	if err != nil || owner.ID == 0 {
		return nil, fmt.Errorf("invalid upload owner")
	}
	urls := make([]string, 0, len(fileURLs))
	paths := make([]string, 0, len(fileURLs))
	for _, value := range fileURLs {
		if half := upload.HalfURL(value); half != "" {
			urls = append(urls, half)
			paths = append(paths, strings.TrimPrefix(half, "/"))
		}
	}
	result := make(map[string]struct{})
	if len(urls) == 0 {
		return result, nil
	}
	var rows []struct {
		FileURL string `gorm:"column:file_url"`
	}
	if err := dbFrom(ctx).Table(model.TableNameVideoUpload).Select("file_url").
		Where("user_type = ? AND user_id = ? AND deleted_at IS NULL", userType, owner.ID).
		Where("file_url IN ? OR file_path IN ?", urls, paths).Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		if half := upload.HalfURL(row.FileURL); half != "" {
			result[half] = struct{}{}
		}
	}
	return result, nil
}

func (r *UploadRepo) PageList(ctx context.Context, page, pageSize int, filter *UploadListFilter) ([]model.VideoUpload, int64, error) {
	table := model.TableNameVideoUpload
	dao := dbFrom(ctx).Table(table).Where("deleted_at IS NULL")
	if filter != nil {
		if filter.UserType != 0 {
			dao = dao.Where("user_type = ?", filter.UserType)
		}
		if filter.UserID != 0 {
			dao = dao.Where("user_id = ?", filter.UserID)
		}
		if filter.Status != 0 {
			dao = dao.Where("status = ?", filter.Status)
		}
		if filter.MediaType != "" {
			dao = dao.Where("media_type = ?", filter.MediaType)
		}
		if filter.FileType != "" {
			dao = dao.Where("file_type = ?", strings.TrimPrefix(strings.ToLower(filter.FileType), "."))
		}
		if filter.StorageProvider != "" {
			dao = dao.Where("storage_provider = ?", filter.StorageProvider)
		}
		if filter.Keyword != "" {
			keyword := "%" + filter.Keyword + "%"
			dao = dao.Where("original_name LIKE ? OR file_path LIKE ? OR upload_id LIKE ?", keyword, keyword, keyword)
		}
	}
	var total int64
	if err := dao.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	rows := make([]model.VideoUpload, 0)
	if err := dao.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

type uploadRecord struct {
	UploadID        string
	Kind            upload.MediaKind
	OriginalName    string
	ContentType     string
	FileSize        int64
	SHA256          string
	StorageProvider string
	FilePath        string
	FileURL         string
}

func (r *UploadRepo) record(ctx context.Context, owner upload.UploadOwner, record uploadRecord, status int8) error {
	userType, err := uploadOwnerUserType(owner.Type)
	if err != nil || owner.ID == 0 {
		return fmt.Errorf("invalid upload owner")
	}
	record.UploadID = strings.TrimSpace(record.UploadID)
	record.FilePath = strings.TrimLeft(filepath.ToSlash(strings.TrimSpace(record.FilePath)), "/")
	record.FileURL = upload.HalfURL(record.FileURL)
	if record.UploadID == "" || record.FilePath == "" || record.FileURL == "" || record.FileSize < 0 {
		return fmt.Errorf("invalid upload record")
	}
	if record.Kind != upload.MediaImage && record.Kind != upload.MediaVideo {
		return fmt.Errorf("invalid upload media type %q", record.Kind)
	}
	extension := strings.TrimPrefix(strings.ToLower(filepath.Ext(record.OriginalName)), ".")
	if extension == "" {
		extension = strings.TrimPrefix(strings.ToLower(filepath.Ext(record.FilePath)), ".")
	}
	now := time.Now()
	row := map[string]any{
		"upload_id": record.UploadID, "user_type": userType, "user_id": owner.ID,
		"media_type": string(record.Kind), "file_type": extension, "mime_type": strings.TrimSpace(record.ContentType),
		"original_name": strings.TrimSpace(record.OriginalName), "file_size": record.FileSize,
		"storage_provider": strings.TrimSpace(record.StorageProvider), "file_path": record.FilePath,
		"file_url": record.FileURL, "sha256": strings.ToLower(strings.TrimSpace(record.SHA256)),
		"status": status, "created_at": now, "updated_at": now,
	}
	uploadRecordMu.Lock()
	defer uploadRecordMu.Unlock()

	existing, err := findUploadOwner(ctx, record.UploadID)
	if err == nil {
		return updateExistingUpload(ctx, record.UploadID, userType, owner.ID, existing, row, status)
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	result := dbFrom(ctx).Table(model.TableNameVideoUpload).Create(row)
	if result.Error == nil {
		return nil
	}
	// MySQL's GORM conflict builder emits an invalid trailing
	// "ON DUPLICATE KEY UPDATE" for DoNothing with map inserts. Use a plain
	// insert and handle the translated duplicate-key error explicitly instead.
	if !errors.Is(result.Error, gorm.ErrDuplicatedKey) {
		return result.Error
	}
	existing, err = findUploadOwner(ctx, record.UploadID)
	if err != nil {
		return err
	}
	return updateExistingUpload(ctx, record.UploadID, userType, owner.ID, existing, row, status)
}

type uploadOwnerRow struct {
	UserType int8
	UserID   uint64
}

func findUploadOwner(ctx context.Context, uploadID string) (uploadOwnerRow, error) {
	var existing uploadOwnerRow
	err := dbFrom(ctx).Table(model.TableNameVideoUpload).
		Select("user_type", "user_id").Where("upload_id = ?", uploadID).Order("id ASC").Take(&existing).Error
	return existing, err
}

func updateExistingUpload(
	ctx context.Context,
	uploadID string,
	userType int8,
	userID uint64,
	existing uploadOwnerRow,
	row map[string]any,
	status int8,
) error {
	if existing.UserType != userType || existing.UserID != userID {
		return fmt.Errorf("upload %s is already owned by another user", uploadID)
	}
	if status != domain.UploadStatusCompleted {
		return nil
	}
	delete(row, "upload_id")
	delete(row, "user_type")
	delete(row, "user_id")
	delete(row, "created_at")
	return dbFrom(ctx).Table(model.TableNameVideoUpload).
		Where("upload_id = ? AND user_type = ? AND user_id = ?", uploadID, userType, userID).
		Updates(row).Error
}

func storedUploadID(provider, filePath string) string {
	digest := sha256.Sum256([]byte(strings.TrimSpace(provider) + "\x00" + filepath.ToSlash(strings.TrimSpace(filePath))))
	return hex.EncodeToString(digest[:16])
}

func uploadOwnerUserType(ownerType upload.UploaderType) (int8, error) {
	switch ownerType {
	case upload.UploaderAdmin:
		return domain.UploadUserAdmin, nil
	case upload.UploaderAPIUser:
		return domain.UploadUserClient, nil
	default:
		return 0, fmt.Errorf("unsupported upload owner type %q", ownerType)
	}
}

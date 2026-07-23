package repository

import (
	"context"
	"fmt"
	"strings"

	"ai-video/internal/domain"
	"ai-video/internal/gen/model"
	"ai-video/internal/pkg/upload"

	"gorm.io/gen/field"
	"gorm.io/gorm/clause"
)

type UploadRepo struct{}

func NewUploadRepo() *UploadRepo { return &UploadRepo{} }

type UploadListFilter struct {
	UserType        int8
	UserID          uint64
	MediaType       string
	FileType        string
	StorageProvider string
	Keyword         string
}

func (r *UploadRepo) RecordCompleted(ctx context.Context, completed upload.CompletedUpload) error {
	userType, err := uploadOwnerUserType(completed.Owner.Type)
	if err != nil || completed.Owner.ID == 0 {
		return fmt.Errorf("invalid upload owner")
	}
	session := completed.Session
	if session.UploaderType != completed.Owner.Type || session.UploaderID != completed.Owner.ID {
		return fmt.Errorf("upload session owner does not match completed owner")
	}
	row := model.VideoUpload{
		UploadID: session.UploadID, UserType: int8(userType), UserID: completed.Owner.ID,
		MediaType: string(session.Kind), FileType: strings.TrimPrefix(strings.ToLower(session.Extension), "."),
		//MIMEType: session.ContentType,
		OriginalName: session.OriginalName, FileSize: uint64(session.TotalSize),
		StorageProvider: session.StorageProvider, FilePath: session.FilePath, FileURL: session.FileURL,
		//SHA256: session.SHA256,
	}
	result := dbFrom(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "upload_id"}},
		DoNothing: true,
	}).Create(&row)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected > 0 {
		return nil
	}
	q := qFrom(ctx).VideoUpload
	existing, err := q.WithContext(ctx).Where(q.UploadID.Eq(session.UploadID)).First()
	if err != nil {
		return err
	}
	if existing.UserType != row.UserType || existing.UserID != row.UserID {
		return fmt.Errorf("upload %s is already owned by another user", session.UploadID)
	}
	return nil
}

func (r *UploadRepo) PageList(ctx context.Context, page, pageSize int, filter *UploadListFilter) ([]model.VideoUpload, int64, error) {
	q := qFrom(ctx).VideoUpload
	dao := q.WithContext(ctx)
	if filter != nil {
		if filter.UserType != 0 {
			dao = dao.Where(q.UserType.Eq(filter.UserType))
		}
		if filter.UserID != 0 {
			dao = dao.Where(q.UserID.Eq(filter.UserID))
		}
		if filter.MediaType != "" {
			dao = dao.Where(q.MediaType.Eq(filter.MediaType))
		}
		if filter.FileType != "" {
			dao = dao.Where(q.FileType.Eq(strings.TrimPrefix(strings.ToLower(filter.FileType), ".")))
		}
		if filter.StorageProvider != "" {
			dao = dao.Where(q.StorageProvider.Eq(filter.StorageProvider))
		}
		if filter.Keyword != "" {
			keyword := "%" + filter.Keyword + "%"
			dao = dao.Where(field.Or(q.OriginalName.Like(keyword), q.FilePath.Like(keyword), q.UploadID.Like(keyword)))
		}
	}
	total, err := dao.Count()
	if err != nil {
		return nil, 0, err
	}
	rows, err := dao.Order(q.ID.Desc()).Offset((page - 1) * pageSize).Limit(pageSize).Find()
	if err != nil {
		return nil, 0, err
	}
	return valuesOf(rows), total, nil
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

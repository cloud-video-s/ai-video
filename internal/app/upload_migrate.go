package app

import (
	"fmt"

	"ai-video/internal/model"

	"gorm.io/gorm"
)

type uploadUserTypeMigration struct {
	UserType int8 `gorm:"column:user_type;type:tinyint;not null;default:0"`
}

func (uploadUserTypeMigration) TableName() string { return "video_upload" }

// MigrateLegacyUploadOwnerColumns upgrades uploader_type/uploader_id to the
// numeric user_type/user_id columns without discarding existing upload rows.
func MigrateLegacyUploadOwnerColumns(db *gorm.DB) error {
	migrator := db.Migrator()
	if !migrator.HasTable(&model.VideoUpload{}) {
		return nil
	}
	hasLegacyType := migrator.HasColumn(&model.VideoUpload{}, "uploader_type")
	hasLegacyID := migrator.HasColumn(&model.VideoUpload{}, "uploader_id")
	hasUserType := migrator.HasColumn(&model.VideoUpload{}, "user_type")
	if !hasLegacyType && !hasLegacyID {
		return nil
	}
	if hasLegacyType {
		var invalid int64
		if err := db.Table("video_upload").Where("uploader_type NOT IN ?", []string{"", "admin", "api_user", "1", "2"}).Count(&invalid).Error; err != nil {
			return err
		}
		if invalid > 0 {
			return fmt.Errorf("video_upload contains %d unsupported uploader_type values", invalid)
		}
		if !hasUserType {
			var blank int64
			if err := db.Table("video_upload").Where("uploader_type = ''").Count(&blank).Error; err != nil {
				return err
			}
			if blank > 0 {
				return fmt.Errorf("video_upload contains %d empty uploader_type values without user_type", blank)
			}
		} else {
			var conflicts int64
			if err := db.Table("video_upload").Where(
				"(uploader_type = '' AND user_type NOT IN ?) OR (uploader_type IN ? AND user_type NOT IN ?) OR (uploader_type IN ? AND user_type NOT IN ?)",
				[]int8{model.UploadUserAdmin, model.UploadUserClient},
				[]string{"admin", "1"}, []int8{model.UploadUserUnknown, model.UploadUserAdmin},
				[]string{"api_user", "2"}, []int8{model.UploadUserUnknown, model.UploadUserClient},
			).Count(&conflicts).Error; err != nil {
				return err
			}
			if conflicts > 0 {
				return fmt.Errorf("video_upload contains %d conflicting user_type values", conflicts)
			}
		}
	}
	if hasLegacyID && migrator.HasColumn(&model.VideoUpload{}, "user_id") {
		var mismatched int64
		if err := db.Table("video_upload").Where(
			"uploader_id <> 0 AND user_id <> 0 AND user_id <> uploader_id",
		).Count(&mismatched).Error; err != nil {
			return err
		}
		if mismatched > 0 {
			return fmt.Errorf("video_upload contains %d conflicting user_id values", mismatched)
		}
	}

	return db.Transaction(func(tx *gorm.DB) error {
		m := tx.Migrator()
		if m.HasIndex(&model.VideoUpload{}, "idx_video_upload_owner") {
			if err := m.DropIndex(&model.VideoUpload{}, "idx_video_upload_owner"); err != nil {
				return err
			}
		}

		if hasLegacyType {
			if !hasUserType {
				if err := m.AddColumn(&uploadUserTypeMigration{}, "UserType"); err != nil {
					return err
				}
			}
			if err := tx.Table("video_upload").Where("user_type = 0").Update("user_type",
				gorm.Expr("CASE WHEN uploader_type IN ? THEN ? WHEN uploader_type IN ? THEN ? ELSE user_type END",
					[]string{"admin", "1"}, model.UploadUserAdmin,
					[]string{"api_user", "2"}, model.UploadUserClient),
			).Error; err != nil {
				return err
			}
			if err := m.DropColumn(&model.VideoUpload{}, "uploader_type"); err != nil {
				return err
			}
		}

		if hasLegacyID {
			if m.HasColumn(&model.VideoUpload{}, "user_id") {
				if err := tx.Table("video_upload").Where("user_id = 0").Update("user_id", gorm.Expr("uploader_id")).Error; err != nil {
					return err
				}
				if err := m.DropColumn(&model.VideoUpload{}, "uploader_id"); err != nil {
					return err
				}
			} else if err := m.RenameColumn(&model.VideoUpload{}, "uploader_id", "user_id"); err != nil {
				return err
			}
		}
		return nil
	})
}

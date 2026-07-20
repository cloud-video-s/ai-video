package app

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"

	"ai-video/internal/model"

	"gorm.io/gorm"
)

// MigrateLegacyTemplateTargets copies the former single-value targeting fields
// into the new many-to-many relations and multi-value JSON fields. It is
// idempotent and intentionally keeps the legacy columns for rollback safety.
func MigrateLegacyTemplateTargets() error {
	return DB.Transaction(func(tx *gorm.DB) error {
		if err := migrateLegacyTemplatePositionsToTypes(tx); err != nil {
			return err
		}
		if err := repairTemplateUserTypes(tx); err != nil {
			return err
		}
		var templates []model.VideoTemplate
		if err := tx.Find(&templates).Error; err != nil {
			return err
		}
		return nil
	})
}

func migrateLegacyTemplatePositionsToTypes(tx *gorm.DB) error {
	if !tx.Migrator().HasTable(&model.VideoTemplate{}) || !tx.Migrator().HasColumn(&model.VideoTemplate{}, "display_position_id") {
		return nil
	}
	type legacyTemplatePosition struct {
		TemplateTypeID    uint64 `gorm:"column:video_template_type_id"`
		DisplayPositionID uint64 `gorm:"column:display_position_id"`
	}
	var rows []legacyTemplatePosition
	if err := tx.Table("video_template").
		Select("video_template_type_id, display_position_id").
		Where("display_position_id > 0").Scan(&rows).Error; err != nil {
		return err
	}
	for _, row := range rows {
		if row.TemplateTypeID == 0 || row.DisplayPositionID == 0 {
			continue
		}
		var position model.VideoDisplayPosition
		if err := tx.First(&position, row.DisplayPositionID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			}
			return err
		}
		item := model.VideoTemplateType{ID: row.TemplateTypeID}
		if err := tx.Model(&item).Association("DisplayPositions").Append(&position); err != nil {
			return err
		}
	}
	return nil
}

func repairTemplateUserTypes(tx *gorm.DB) error {
	type rawTemplateUserTypes struct {
		ID             uint64
		UserTypes      string
		LegacyUserType uint8 `gorm:"column:user_type"`
	}
	var rows []rawTemplateUserTypes
	columns := "id, user_types"
	if tx.Migrator().HasColumn(&model.VideoTemplate{}, "user_type") {
		columns += ", user_type"
	}
	if err := tx.Table("video_template").Select(columns).Scan(&rows).Error; err != nil {
		return err
	}
	for _, row := range rows {
		raw := strings.TrimSpace(row.UserTypes)
		var values []int
		if raw != "" && raw != "null" {
			if err := json.Unmarshal([]byte(raw), &values); err == nil && validTemplateUserTypes(values) {
				continue
			}
			var encoded string
			if err := json.Unmarshal([]byte(raw), &encoded); err == nil {
				if decoded, err := base64.StdEncoding.DecodeString(encoded); err == nil {
					values = values[:0]
					for _, value := range decoded {
						if value == 1 || value == 2 {
							values = append(values, int(value))
						}
					}
				}
			}
		}
		if !validTemplateUserTypes(values) {
			if row.LegacyUserType == 1 || row.LegacyUserType == 2 {
				values = []int{int(row.LegacyUserType)}
			} else {
				values = []int{1, 2}
			}
		}
		encoded, err := json.Marshal(values)
		if err != nil {
			return err
		}
		if err := tx.Table("video_template").Where("id = ?", row.ID).Update("user_types", string(encoded)).Error; err != nil {
			return err
		}
	}
	return nil
}

func validTemplateUserTypes(values []int) bool {
	if len(values) == 0 {
		return false
	}
	for _, value := range values {
		if value != 1 && value != 2 {
			return false
		}
	}
	return true
}

func appendAssociationWhenEmpty(tx *gorm.DB, template *model.VideoTemplate, name string, value interface{}) error {
	association := tx.Model(template).Association(name)
	if association.Error != nil {
		return association.Error
	}
	if association.Count() > 0 {
		return association.Error
	}
	return association.Append(value)
}

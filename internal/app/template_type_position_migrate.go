package app

import (
	"errors"
	"strings"

	"ai-video/internal/gen/model"

	"gorm.io/gorm"
)

// MigrateLegacyTemplateTypePositions copies the former single position_key
// value into the many-to-many relation. The legacy column is retained for
// rollback compatibility and the migration is safe to run on every startup.
func MigrateLegacyTemplateTypePositions(db *gorm.DB) error {
	if db == nil || !db.Migrator().HasTable(&model.VideoTemplateType{}) ||
		!db.Migrator().HasColumn(&model.VideoTemplateType{}, "position_key") {
		return nil
	}
	type legacyRow struct {
		ID          uint64
		PositionKey string
	}
	var rows []legacyRow
	if err := db.Model(&model.VideoTemplateType{}).Select("id, position_key").Scan(&rows).Error; err != nil {
		return err
	}
	return db.Transaction(func(tx *gorm.DB) error {
		for _, row := range rows {
			key := strings.TrimSpace(row.PositionKey)
			if key == "" {
				continue
			}
			item := model.VideoTemplateType{ID: row.ID}
			association := tx.Model(&item).Association("DisplayPositions")
			if association.Error != nil {
				return association.Error
			}
			if association.Count() > 0 {
				continue
			}
			var position model.VideoDisplayPosition
			if err := tx.Where("position_key = ?", key).First(&position).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					continue
				}
				return err
			}
			if err := association.Append(&position); err != nil {
				return err
			}
		}
		return nil
	})
}

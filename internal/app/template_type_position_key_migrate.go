package app

import (
	"ai-video/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// MigrateTemplateTypeDisplayPositionKeys converts the join table from a
// numeric display-position ID to its stable position key without losing rows.
func MigrateTemplateTypeDisplayPositionKeys(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	table := model.VideoTemplateTypeDisplayPosition{}.TableName()
	if !db.Migrator().HasTable(table) {
		return db.AutoMigrate(&model.VideoTemplateTypeDisplayPosition{})
	}
	if db.Migrator().HasColumn(table, "position_key") && !db.Migrator().HasColumn(table, "display_position_id") {
		return nil
	}
	if !db.Migrator().HasColumn(table, "display_position_id") {
		return nil
	}

	var rows []model.VideoTemplateTypeDisplayPosition
	if err := db.Table(table + " AS relation").
		Select("relation.template_type_id, position.position_key").
		Joins("JOIN video_display_position AS position ON position.id = relation.display_position_id").
		Scan(&rows).Error; err != nil {
		return err
	}
	if err := db.Migrator().DropTable(table); err != nil {
		return err
	}
	if err := db.AutoMigrate(&model.VideoTemplateTypeDisplayPosition{}); err != nil {
		return err
	}
	if len(rows) == 0 {
		return nil
	}
	return db.Clauses(clause.OnConflict{DoNothing: true}).Create(&rows).Error
}

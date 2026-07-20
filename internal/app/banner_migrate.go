package app

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
)

// EnsureBannerPositionKeyColumn adds the new required placement column without
// breaking databases that already contain Banner rows. Existing rows receive
// an empty key and stay hidden from placement queries until assigned in admin.
func EnsureBannerPositionKeyColumn(db *gorm.DB) error {
	if db == nil || !db.Migrator().HasTable("video_banner") || db.Migrator().HasColumn("video_banner", "position_key") {
		return nil
	}
	statement := "ALTER TABLE video_banner ADD COLUMN position_key varchar(100) NOT NULL DEFAULT ''"
	if db.Dialector.Name() == "mysql" {
		statement += " COMMENT '位置编号' AFTER cover_image"
	}
	return db.Exec(statement).Error
}

// NormalizeBannerJumpTypes converts the legacy string enum before AutoMigrate
// changes jump_type to TINYINT UNSIGNED. Databases already using a numeric
// column are left untouched.
func NormalizeBannerJumpTypes(db *gorm.DB) error {
	if db == nil || !db.Migrator().HasTable("video_banner") || !db.Migrator().HasColumn("video_banner", "jump_type") {
		return nil
	}
	columns, err := db.Migrator().ColumnTypes("video_banner")
	if err != nil {
		return err
	}
	legacyStringColumn := false
	for _, column := range columns {
		if !strings.EqualFold(column.Name(), "jump_type") {
			continue
		}
		databaseType := strings.ToUpper(column.DatabaseTypeName())
		legacyStringColumn = strings.Contains(databaseType, "CHAR") || strings.Contains(databaseType, "TEXT")
		break
	}
	if !legacyStringColumn {
		return nil
	}

	var invalidCount int64
	if err := db.Table("video_banner").Where(
		"jump_type IS NULL OR LOWER(TRIM(jump_type)) NOT IN ?",
		[]string{"1", "2", "3", "4", "link", "template", "text_to_image", "text_to_video"},
	).Count(&invalidCount).Error; err != nil {
		return err
	}
	if invalidCount > 0 {
		return fmt.Errorf("video_banner contains %d unsupported legacy jump_type values", invalidCount)
	}

	return db.Exec(`UPDATE video_banner SET jump_type = CASE LOWER(TRIM(jump_type))
		WHEN 'link' THEN '1'
		WHEN 'template' THEN '2'
		WHEN 'text_to_image' THEN '3'
		WHEN 'text_to_video' THEN '4'
		ELSE TRIM(jump_type)
	END`).Error
}

package app

import (
	"errors"
	"strings"

	"ai-video/internal/gen/model"

	"gorm.io/gorm"
)

const bannerPositionKeyMigrationTable = "video_banner_display_position_key_migration"

type bannerPositionKeyMigrationRow struct {
	BannerID    uint64 `gorm:"column:banner_id;type:bigint unsigned;primaryKey;autoIncrement:false"`
	PositionKey string `gorm:"column:position_key;size:64;primaryKey;autoIncrement:false;index"`
}

func (bannerPositionKeyMigrationRow) TableName() string {
	return bannerPositionKeyMigrationTable
}

// MigrateBannerDisplayPositionKeys upgrades the former banner_id +
// display_position_id association to banner_id + position_key while preserving
// every association whose display position still exists.
func MigrateBannerDisplayPositionKeys(db *gorm.DB) error {
	const table = "video_banner_display_position"
	if db == nil || !db.Migrator().HasTable(table) || db.Migrator().HasColumn(table, "position_key") {
		return nil
	}
	if !db.Migrator().HasColumn(table, "display_position_id") || !db.Migrator().HasTable("video_display_position") {
		return errors.New("video_banner_display_position has an unsupported schema")
	}

	return db.Transaction(func(tx *gorm.DB) error {
		if tx.Migrator().HasTable(bannerPositionKeyMigrationTable) {
			if err := tx.Migrator().DropTable(bannerPositionKeyMigrationTable); err != nil {
				return err
			}
		}
		if err := tx.AutoMigrate(&bannerPositionKeyMigrationRow{}); err != nil {
			return err
		}
		if err := tx.Exec(`INSERT INTO video_banner_display_position_key_migration (banner_id, position_key)
			SELECT DISTINCT relation.banner_id, position.position_key
			FROM video_banner_display_position relation
			JOIN video_display_position position ON position.id = relation.display_position_id
			WHERE position.position_key <> ''`).Error; err != nil {
			return err
		}
		if err := tx.Migrator().DropTable(table); err != nil {
			return err
		}
		return tx.Migrator().RenameTable(bannerPositionKeyMigrationTable, table)
	})
}

// MigrateLegacyBannerPositions copies the former single position key into the
// Banner display-position relation. Existing relations are never overwritten.
func MigrateLegacyBannerPositions(db *gorm.DB) error {
	if db == nil || !db.Migrator().HasTable("video_banner") ||
		!db.Migrator().HasTable("video_display_position") ||
		!db.Migrator().HasTable("video_banner_display_position") ||
		!db.Migrator().HasColumn("video_banner", "position_key") {
		return nil
	}

	type legacyBannerPosition struct {
		ID          uint64 `gorm:"column:id"`
		PositionKey string `gorm:"column:position_key"`
	}
	return db.Transaction(func(tx *gorm.DB) error {
		var banners []legacyBannerPosition
		if err := tx.Table("video_banner").Select("id, position_key").Where("position_key <> ''").Scan(&banners).Error; err != nil {
			return err
		}
		for i := range banners {
			var count int64
			if err := tx.Model(&model.VideoBannerDisplayPosition{}).
				Where("banner_id = ?", banners[i].ID).Count(&count).Error; err != nil {
				return err
			}
			if count > 0 {
				continue
			}

			key := strings.TrimSpace(banners[i].PositionKey)
			var position model.VideoDisplayPosition
			if err := tx.Where("position_key = ?", key).First(&position).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					continue
				}
				return err
			}
			if err := tx.Create(&model.VideoBannerDisplayPosition{
				BannerID: banners[i].ID, PositionKey: position.PositionKey,
			}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// RemoveLegacyBannerPositionKey removes the former single-position column
// after its data has been copied into video_banner_display_position.
func RemoveLegacyBannerPositionKey(db *gorm.DB) error {
	if db == nil || !db.Migrator().HasTable("video_banner") || !db.Migrator().HasColumn("video_banner", "position_key") {
		return nil
	}
	return db.Exec("ALTER TABLE video_banner DROP COLUMN position_key").Error
}

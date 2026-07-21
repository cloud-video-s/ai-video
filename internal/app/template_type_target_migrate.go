package app

import (
	"errors"
	"strconv"
	"strings"

	"ai-video/internal/gen/model"

	"gorm.io/gorm"
)

type legacyTemplateTypeTargets struct {
	ID           uint64  `gorm:"column:id"`
	Country      string  `gorm:"column:country"`
	ChannelID    string  `gorm:"column:channel_id"`
	PackageID    *uint64 `gorm:"column:package_id"`
	AppPackage   string  `gorm:"column:app_package"`
	UserType     uint8   `gorm:"column:user_type"`
	IsSubscribed bool    `gorm:"column:is_subscribed"`
}

// MigrateLegacyTemplateTypeTargets backfills former single-value delivery
// columns into generated-model associations and JSON arrays. Legacy columns
// are read through a migration-only projection and never added to the model.
func MigrateLegacyTemplateTypeTargets(db *gorm.DB) error {
	if db == nil || !db.Migrator().HasTable(&model.VideoTemplateType{}) {
		return nil
	}
	legacyColumns := []string{"country", "channel_id", "package_id", "app_package", "user_type", "is_subscribed"}
	selectedColumns := []string{"id"}
	for _, column := range legacyColumns {
		if db.Migrator().HasColumn("video_template_type", column) {
			selectedColumns = append(selectedColumns, column)
		}
	}

	return db.Transaction(func(tx *gorm.DB) error {
		var items []model.VideoTemplateType
		if err := tx.Find(&items).Error; err != nil {
			return err
		}
		for i := range items {
			item := &items[i]
			var legacy legacyTemplateTypeTargets
			if err := tx.Table("video_template_type").Select(selectedColumns).Where("id = ?", item.ID).Scan(&legacy).Error; err != nil {
				return err
			}

			changed := false
			if len(item.UserTypes) == 0 {
				if legacy.UserType == 1 || legacy.UserType == 2 {
					item.UserTypes = []int{int(legacy.UserType)}
				} else {
					item.UserTypes = []int{1, 2}
				}
				changed = true
			}
			if len(item.SubscriptionStatuses) == 0 {
				if legacy.IsSubscribed {
					item.SubscriptionStatuses = []string{"subscribed"}
				} else {
					item.SubscriptionStatuses = []string{"unsubscribed"}
				}
				changed = true
			}
			if changed {
				if err := tx.Model(item).Select("UserTypes", "SubscriptionStatuses").Updates(item).Error; err != nil {
					return err
				}
			}

			if code := strings.ToUpper(strings.TrimSpace(legacy.Country)); code != "" {
				var country model.VideoCountry
				if err := tx.Where("code = ?", code).First(&country).Error; err == nil {
					if err := appendTemplateTypeAssociationWhenEmpty(tx, item, "Countries", &country); err != nil {
						return err
					}
				} else if !errors.Is(err, gorm.ErrRecordNotFound) {
					return err
				}
			}
			if value := strings.TrimSpace(legacy.ChannelID); value != "" {
				var channel model.VideoChannel
				query := tx.Where("channel_code = ?", value)
				if id, err := strconv.ParseUint(value, 10, 64); err == nil {
					query = tx.Where("channel_code = ? OR channel_id = ?", value, id)
				}
				if err := query.First(&channel).Error; err == nil {
					if err := appendTemplateTypeAssociationWhenEmpty(tx, item, "Channels", &channel); err != nil {
						return err
					}
				} else if !errors.Is(err, gorm.ErrRecordNotFound) {
					return err
				}
			}
			var packageQuery *gorm.DB
			if legacy.PackageID != nil && *legacy.PackageID != 0 {
				packageQuery = tx.Where("id = ?", *legacy.PackageID)
			} else if code := strings.TrimSpace(legacy.AppPackage); code != "" {
				packageQuery = tx.Where("package_code = ?", code).Order("id DESC")
			}
			if packageQuery == nil {
				continue
			}
			var appPackage model.VideoPackage
			if err := packageQuery.First(&appPackage).Error; err == nil {
				if err := appendTemplateTypeAssociationWhenEmpty(tx, item, "Packages", &appPackage); err != nil {
					return err
				}
			} else if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
		}
		return nil
	})
}

func appendTemplateTypeAssociationWhenEmpty(tx *gorm.DB, item *model.VideoTemplateType, name string, value interface{}) error {
	association := tx.Model(item).Association(name)
	if association.Error != nil {
		return association.Error
	}
	if association.Count() > 0 {
		return association.Error
	}
	return association.Append(value)
}

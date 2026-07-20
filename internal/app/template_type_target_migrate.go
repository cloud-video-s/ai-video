package app

import (
	"errors"
	"strconv"
	"strings"

	"ai-video/internal/model"

	"gorm.io/gorm"
)

// MigrateLegacyTemplateTypeTargets backfills the former single-value delivery
// columns into the new multi-select associations and JSON arrays.
func MigrateLegacyTemplateTypeTargets(db *gorm.DB) error {
	if db == nil || !db.Migrator().HasTable(&model.VideoTemplateType{}) {
		return nil
	}
	return db.Transaction(func(tx *gorm.DB) error {
		var items []model.VideoTemplateType
		if err := tx.Find(&items).Error; err != nil {
			return err
		}
		for i := range items {
			item := &items[i]
			changed := false
			if len(item.UserTypes) == 0 {
				if item.LegacyUserType == 1 || item.LegacyUserType == 2 {
					item.UserTypes = []int{int(item.LegacyUserType)}
				} else {
					item.UserTypes = []int{1, 2}
				}
				changed = true
			}
			if len(item.SubscriptionStatuses) == 0 {
				if item.LegacyIsSubscribed {
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

			if code := strings.ToUpper(strings.TrimSpace(item.LegacyCountry)); code != "" {
				var country model.VideoCountry
				if err := tx.Where("code = ?", code).First(&country).Error; err == nil {
					if err := appendTemplateTypeAssociationWhenEmpty(tx, item, "Countries", &country); err != nil {
						return err
					}
				} else if !errors.Is(err, gorm.ErrRecordNotFound) {
					return err
				}
			}
			if value := strings.TrimSpace(item.LegacyChannelID); value != "" {
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
			if item.LegacyPackageID != nil && *item.LegacyPackageID != 0 {
				packageQuery = tx.Where("id = ?", *item.LegacyPackageID)
			} else if code := strings.TrimSpace(item.LegacyAppPackage); code != "" {
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

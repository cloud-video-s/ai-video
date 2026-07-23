package repository

import (
	"context"
	"fmt"
	"time"

	"ai-video/internal/gen/model"

	"gorm.io/gorm"
)

type PointsPackageRepo struct {
	BaseRepo[model.VideoPointsPackage]
}

func NewPointsPackageRepo() *PointsPackageRepo { return &PointsPackageRepo{} }

type PointsPackageListFilter struct {
	PackageID    uint64
	ChannelID    uint64
	System       string
	UserType     int
	ResourceType string
	Status       *int8
	Keyword      string
}

func (r *PointsPackageRepo) PageList(ctx context.Context, page, pageSize int, filter *PointsPackageListFilter) ([]model.VideoPointsPackage, int64, error) {
	buildQuery := func() *gorm.DB {
		db := dbFrom(ctx).Model(&model.VideoPointsPackage{})
		if filter == nil {
			return db
		}
		if filter.PackageID != 0 {
			db = db.Where(`EXISTS (
				SELECT 1 FROM video_points_package_package vppp
				JOIN video_package vp ON vp.package_code = vppp.package_code
				WHERE vppp.product_code = video_points_package.product_code
					AND vp.id = ? AND vppp.deleted_at IS NULL
			)`, filter.PackageID)
		}
		if filter.ChannelID != 0 {
			db = db.Where(`EXISTS (
				SELECT 1 FROM video_points_package_channel vpc
				JOIN video_channel vc ON vc.channel_code = vpc.channel_code
				WHERE vpc.product_code = video_points_package.product_code
					AND vc.channel_id = ? AND vpc.deleted_at IS NULL
			)`, filter.ChannelID)
		}
		if filter.System != "" {
			db = db.Where("systems LIKE ?", "%\""+filter.System+"\"%")
		}
		if filter.UserType != 0 {
			db = db.Where("user_types LIKE ?", "%"+fmt.Sprint(filter.UserType)+"%")
		}
		if filter.ResourceType != "" {
			db = db.Where("resource_type = ?", filter.ResourceType)
		}
		if filter.Status != nil {
			db = db.Where("status = ?", *filter.Status)
		}
		if filter.Keyword != "" {
			keyword := "%" + filter.Keyword + "%"
			db = db.Where("product_code LIKE ? OR name LIKE ? OR badge_text LIKE ? OR description LIKE ?", keyword, keyword, keyword, keyword)
		}
		return db
	}

	var total int64
	if err := buildQuery().Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.VideoPointsPackage
	err := preloadPointsPackageTargets(buildQuery()).Order("sort ASC, id DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}

func (r *PointsPackageRepo) GetDetail(ctx context.Context, id uint64) (*model.VideoPointsPackage, error) {
	var item model.VideoPointsPackage
	if err := preloadPointsPackageTargets(dbFrom(ctx)).First(&item, id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *PointsPackageRepo) GetByProductID(ctx context.Context, productID string) (*model.VideoPointsPackage, error) {
	return r.BaseRepo.GetOne(ctx, &QueryOptions{Where: map[string]interface{}{"product_code": productID}})
}

func (r *PointsPackageRepo) ListOptions(ctx context.Context) ([]model.VideoPointsPackage, error) {
	var list []model.VideoPointsPackage
	err := preloadPointsPackageTargets(dbFrom(ctx)).Where("status = ?", int8(1)).Order("sort ASC, id ASC").Find(&list).Error
	return list, err
}

func preloadPointsPackageTargets(db *gorm.DB) *gorm.DB {
	return db.Preload("Packages").Preload("Channels")
}

func (r *PointsPackageRepo) UpdateFields(ctx context.Context, item *model.VideoPointsPackage) error {
	return r.BaseRepo.Update(ctx, item,
		"ProductID", "Name", "Systems", "UserTypes", "ResourceType", "Points",
		"Currency", "SalePrice", "ActualRevenue", "OriginalPrice", "BadgeText", "Description",
		"ButtonText", "IsDefault", "Status", "Sort",
	)
}

func (r *PointsPackageRepo) ReplaceTargets(ctx context.Context, item *model.VideoPointsPackage, packageID uint64, channelIDs []uint64) error {
	db := dbFrom(ctx)
	var appPackage model.VideoPackage
	if err := db.Select("id", "package_code").First(&appPackage, packageID).Error; err != nil {
		return err
	}
	channels := make([]model.VideoChannel, 0, len(channelIDs))
	if len(channelIDs) > 0 {
		if err := db.Select("channel_id", "channel_code").Where("channel_id IN ?", channelIDs).Find(&channels).Error; err != nil {
			return err
		}
		if len(channels) != len(channelIDs) {
			return fmt.Errorf("one or more channels do not exist")
		}
	}
	if err := db.Unscoped().Where("product_code = ?", item.ProductID).Delete(&model.VideoPointsPackagePackage{}).Error; err != nil {
		return err
	}
	now := time.Now()
	if err := db.Create(&model.VideoPointsPackagePackage{
		ProductCode: item.ProductID,
		PackageCode: appPackage.PackageCode,
		CreatedAt:   now,
		UpdatedAt:   now,
	}).Error; err != nil {
		return err
	}
	if err := db.Unscoped().Where("product_code = ?", item.ProductID).Delete(&model.VideoPointsPackageChannel{}).Error; err != nil {
		return err
	}
	if len(channels) == 0 {
		return nil
	}
	rows := make([]model.VideoPointsPackageChannel, 0, len(channels))
	for _, channel := range channels {
		rows = append(rows, model.VideoPointsPackageChannel{
			ProductCode: item.ProductID,
			ChannelCode: channel.ChannelCode,
			CreatedAt:   now,
			UpdatedAt:   now,
		})
	}
	return db.Create(&rows).Error
}

func (r *PointsPackageRepo) ClearDefaults(ctx context.Context, packageID uint64, resourceType string, exceptID uint64) error {
	db := dbFrom(ctx).Model(&model.VideoPointsPackage{}).
		Where("resource_type = ? AND is_default = ?", resourceType, true).
		Where(`EXISTS (
			SELECT 1 FROM video_points_package_package vppp
			JOIN video_package vp ON vp.package_code = vppp.package_code
			WHERE vppp.product_code = video_points_package.product_code
				AND vp.id = ? AND vppp.deleted_at IS NULL
		)`, packageID)
	if exceptID != 0 {
		db = db.Where("id <> ?", exceptID)
	}
	return db.Update("is_default", false).Error
}

func (r *PointsPackageRepo) SetDefault(ctx context.Context, item *model.VideoPointsPackage) error {
	return Transaction(ctx, func(ctx context.Context) error {
		if len(item.Packages) == 0 {
			return fmt.Errorf("points package is not associated with an app package")
		}
		if err := r.ClearDefaults(ctx, item.Packages[0].ID, item.ResourceType, item.ID); err != nil {
			return err
		}
		item.IsDefault = true
		return dbFrom(ctx).Model(&model.VideoPointsPackage{}).
			Where("id = ?", item.ID).Update("is_default", true).Error
	})
}

func (r *PointsPackageRepo) UpdateStatus(ctx context.Context, id uint64, status int8) error {
	return dbFrom(ctx).Model(&model.VideoPointsPackage{}).Where("id = ?", id).Update("status", status).Error
}

func (r *PointsPackageRepo) DeleteWithTargets(ctx context.Context, id uint64) error {
	db := dbFrom(ctx)
	return db.Transaction(func(tx *gorm.DB) error {
		var item model.VideoPointsPackage
		if err := tx.Select("id", "product_code").First(&item, id).Error; err != nil {
			return err
		}
		if err := tx.Unscoped().Where("product_code = ?", item.ProductID).Delete(&model.VideoPointsPackagePackage{}).Error; err != nil {
			return err
		}
		if err := tx.Unscoped().Where("product_code = ?", item.ProductID).Delete(&model.VideoPointsPackageChannel{}).Error; err != nil {
			return err
		}
		return tx.Delete(&item).Error
	})
}

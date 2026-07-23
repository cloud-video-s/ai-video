package repository

import (
	"context"
	"fmt"

	"ai-video/internal/gen/model"

	"gorm.io/gorm"
)

type VIPSubscriptionRepo struct {
	BaseRepo[model.VideoVipSubscription]
}

func NewVIPSubscriptionRepo() *VIPSubscriptionRepo { return &VIPSubscriptionRepo{} }

type VIPSubscriptionListFilter struct {
	PackageID         uint64
	DisplayPositionID uint64
	ChannelID         uint64
	ExcludedChannelID uint64
	PlanType          string
	Platform          string
	DisplayMode       *int8
	Status            *int8
	IsSubscription    *bool
	Keyword           string
}

func (r *VIPSubscriptionRepo) PageList(ctx context.Context, page, pageSize int, filter *VIPSubscriptionListFilter) ([]model.VideoVipSubscription, int64, error) {
	db := dbFrom(ctx).Model(&model.VideoVipSubscription{})
	if filter != nil {
		if filter.PackageID != 0 {
			db = db.Where(`EXISTS (
				SELECT 1
				FROM video_vip_subscription_package vsp
				JOIN video_package vp ON vp.package_code = vsp.package_code
				WHERE vsp.subscription_id = video_vip_subscription.id
					AND vsp.deleted_at IS NULL AND vp.deleted_at IS NULL AND vp.id = ?
			)`, filter.PackageID)
		}
		if filter.DisplayPositionID != 0 {
			db = db.Where(`EXISTS (
				SELECT 1
				FROM video_vip_subscription_position vsp
				JOIN video_display_position vdp ON vdp.position_key = vsp.product_code
				WHERE vsp.subscription_id = video_vip_subscription.id
					AND vsp.deleted_at IS NULL AND vdp.deleted_at IS NULL AND vdp.id = ?
			)`, filter.DisplayPositionID)
		}
		if filter.ChannelID != 0 {
			db = db.Where(`EXISTS (
				SELECT 1
				FROM video_vip_subscription_channel vsc
				JOIN video_channel vc ON vc.channel_code = vsc.channel_code
				WHERE vsc.subscription_id = video_vip_subscription.id
					AND vsc.deleted_at IS NULL AND vc.deleted_at IS NULL AND vc.channel_id = ?
			)`, filter.ChannelID)
		}
		if filter.ExcludedChannelID != 0 {
			db = db.Where(`EXISTS (
				SELECT 1
				FROM video_vip_subscription_excluded_channel vsec
				JOIN video_channel vc ON vc.channel_id = vsec.channel_id
				WHERE vsec.subscription_id = video_vip_subscription.id
					AND vc.deleted_at IS NULL AND vc.channel_id = ?
			)`, filter.ExcludedChannelID)
		}
		if filter.PlanType != "" {
			db = db.Where("plan_type = ?", filter.PlanType)
		}
		if filter.Platform != "" {
			db = db.Where("platform = ?", filter.Platform)
		}
		if filter.DisplayMode != nil {
			db = db.Where("display_mode = ?", *filter.DisplayMode)
		}
		if filter.Status != nil {
			db = db.Where("status = ?", *filter.Status)
		}
		if filter.IsSubscription != nil {
			db = db.Where("is_subscription = ?", *filter.IsSubscription)
		}
		if filter.Keyword != "" {
			keyword := "%" + filter.Keyword + "%"
			db = db.Where("product_id LIKE ? OR name LIKE ? OR v_ip_level LIKE ? OR description LIKE ?", keyword, keyword, keyword, keyword)
		}
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.VideoVipSubscription
	err := preloadVIPSubscription(db).Order("sort ASC, id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}

func (r *VIPSubscriptionRepo) GetDetail(ctx context.Context, id uint64) (*model.VideoVipSubscription, error) {
	var item model.VideoVipSubscription
	if err := preloadVIPSubscription(dbFrom(ctx)).First(&item, id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func preloadVIPSubscription(db *gorm.DB) *gorm.DB {
	return db.Preload("Packages").Preload("DisplayPositions").Preload("Channels").Preload("ExcludedChannels")
}

func (r *VIPSubscriptionRepo) UpdateFields(ctx context.Context, item *model.VideoVipSubscription) error {
	return r.BaseRepo.Update(ctx, item,
		"Platform", "ProductID", "Name", "VIPLevel", "PlanType", "AppVersion", "Currency",
		"FirstSubscriptionPrice", "FirstSubscriptionRevenue", "FirstBonusPoints", "OriginalPrice",
		"VIPDurationDays", "TrialDays", "RenewalText", "BadgeText", "AgreementDefaultChecked",
		"DisplayMode", "Status", "FreeTrial", "IsSubscription", "IsDefault", "SubscriptionDescription",
		"SubscriptionPrice", "SubscriptionRevenue", "SubscriptionPoints", "SubscriptionPeriod",
		"Sort", "Description", "Remark",
	)
}

type VIPSubscriptionTargets struct {
	PackageIDs         []uint64
	DisplayPositionIDs []uint64
	ChannelIDs         []uint64
	ExcludedChannelIDs []uint64
}

func (r *VIPSubscriptionRepo) ReplaceTargets(ctx context.Context, item *model.VideoVipSubscription, targets VIPSubscriptionTargets) error {
	db := dbFrom(ctx)
	packages, err := loadVIPPackages(db, targets.PackageIDs)
	if err != nil {
		return err
	}
	positions, err := loadVIPDisplayPositions(db, targets.DisplayPositionIDs)
	if err != nil {
		return err
	}
	channels, err := loadVIPChannels(db, targets.ChannelIDs)
	if err != nil {
		return err
	}
	excludedChannels, err := loadVIPChannels(db, targets.ExcludedChannelIDs)
	if err != nil {
		return err
	}
	associations := []struct {
		name   string
		values interface{}
	}{
		{name: "Packages", values: packages},
		{name: "DisplayPositions", values: positions},
		{name: "Channels", values: channels},
		{name: "ExcludedChannels", values: excludedChannels},
	}
	for _, association := range associations {
		if err := db.Model(item).Association(association.name).Replace(association.values); err != nil {
			return err
		}
	}
	return nil
}

func loadVIPPackages(db *gorm.DB, ids []uint64) ([]model.VideoPackage, error) {
	items := make([]model.VideoPackage, 0, len(ids))
	if len(ids) == 0 {
		return items, nil
	}
	if err := db.Where("id IN ?", ids).Find(&items).Error; err != nil {
		return nil, err
	}
	if len(items) != len(ids) {
		return nil, fmt.Errorf("one or more packages do not exist")
	}
	return items, nil
}

func loadVIPDisplayPositions(db *gorm.DB, ids []uint64) ([]model.VideoDisplayPosition, error) {
	items := make([]model.VideoDisplayPosition, 0, len(ids))
	if len(ids) == 0 {
		return items, nil
	}
	if err := db.Where("id IN ?", ids).Find(&items).Error; err != nil {
		return nil, err
	}
	if len(items) != len(ids) {
		return nil, fmt.Errorf("one or more display positions do not exist")
	}
	return items, nil
}

func loadVIPChannels(db *gorm.DB, ids []uint64) ([]model.VideoChannel, error) {
	items := make([]model.VideoChannel, 0, len(ids))
	if len(ids) == 0 {
		return items, nil
	}
	if err := db.Where("channel_id IN ?", ids).Find(&items).Error; err != nil {
		return nil, err
	}
	if len(items) != len(ids) {
		return nil, fmt.Errorf("one or more channels do not exist")
	}
	return items, nil
}

func (r *VIPSubscriptionRepo) ClearDefaults(ctx context.Context, packageID uint64, platform string, exceptID uint64) error {
	db := dbFrom(ctx).Model(&model.VideoVipSubscription{}).
		Where("platform = ? AND is_default = ?", platform, true).
		Where(`EXISTS (
			SELECT 1
			FROM video_vip_subscription_package vsp
			JOIN video_package vp ON vp.package_code = vsp.package_code
			WHERE vsp.subscription_id = video_vip_subscription.id
				AND vsp.deleted_at IS NULL AND vp.deleted_at IS NULL AND vp.id = ?
		)`, packageID)
	if exceptID != 0 {
		db = db.Where("id <> ?", exceptID)
	}
	return db.Update("is_default", false).Error
}

func (r *VIPSubscriptionRepo) UpdateStatus(ctx context.Context, id uint64, status int8) error {
	return dbFrom(ctx).Model(&model.VideoVipSubscription{}).Where("id = ?", id).Update("status", status).Error
}

func (r *VIPSubscriptionRepo) UpdateDisplayMode(ctx context.Context, id uint64, mode int8) error {
	return dbFrom(ctx).Model(&model.VideoVipSubscription{}).Where("id = ?", id).Update("display_mode", mode).Error
}

func (r *VIPSubscriptionRepo) SetDefault(ctx context.Context, item *model.VideoVipSubscription) error {
	if len(item.Packages) != 1 {
		return fmt.Errorf("VIP 订阅套餐必须关联且只能关联一个应用包")
	}
	return repositorySetDefault(ctx, r, item, item.Packages[0].ID)
}

func repositorySetDefault(ctx context.Context, r *VIPSubscriptionRepo, item *model.VideoVipSubscription, packageID uint64) error {
	return Transaction(ctx, func(ctx context.Context) error {
		if err := r.ClearDefaults(ctx, packageID, item.Platform, item.ID); err != nil {
			return err
		}
		item.IsDefault = true
		return dbFrom(ctx).Model(item).Update("is_default", true).Error
	})
}

func (r *VIPSubscriptionRepo) DeleteWithTargets(ctx context.Context, id uint64) error {
	return dbFrom(ctx).Select("Packages", "DisplayPositions", "Channels", "ExcludedChannels").Delete(&model.VideoVipSubscription{ID: id}).Error
}

func (r *VIPSubscriptionRepo) PackageCount(ctx context.Context, packageID uint64) (int64, error) {
	var count int64
	err := dbFrom(ctx).Model(&model.VideoVipSubscription{}).
		Where(`EXISTS (
			SELECT 1
			FROM video_vip_subscription_package vsp
			JOIN video_package vp ON vp.package_code = vsp.package_code
			WHERE vsp.subscription_id = video_vip_subscription.id
				AND vsp.deleted_at IS NULL AND vp.deleted_at IS NULL AND vp.id = ?
		)`, packageID).
		Count(&count).Error
	return count, err
}

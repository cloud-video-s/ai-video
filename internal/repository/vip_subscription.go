package repository

import (
	"context"

	"ai-video/internal/model"

	"gorm.io/gorm"
)

type VIPSubscriptionRepo struct {
	BaseRepo[model.VideoVIPSubscription]
}

func NewVIPSubscriptionRepo() *VIPSubscriptionRepo { return &VIPSubscriptionRepo{} }

type VIPSubscriptionListFilter struct {
	PackageID         uint64
	DisplayPositionID uint64
	ChannelID         uint64
	PlanType          string
	Platform          string
	DisplayMode       *int8
	Status            *int8
	IsSubscription    *bool
	Keyword           string
}

func (r *VIPSubscriptionRepo) PageList(ctx context.Context, page, pageSize int, filter *VIPSubscriptionListFilter) ([]model.VideoVIPSubscription, int64, error) {
	db := dbFrom(ctx).Model(&model.VideoVIPSubscription{})
	if filter != nil {
		if filter.PackageID != 0 {
			db = db.Where("package_id = ?", filter.PackageID)
		}
		if filter.DisplayPositionID != 0 {
			db = db.Where("EXISTS (SELECT 1 FROM video_vip_subscription_position vsp WHERE vsp.subscription_id = video_vip_subscription.id AND vsp.display_position_id = ?)", filter.DisplayPositionID)
		}
		if filter.ChannelID != 0 {
			db = db.Where("EXISTS (SELECT 1 FROM video_vip_subscription_channel vsc WHERE vsc.subscription_id = video_vip_subscription.id AND vsc.channel_id = ?)", filter.ChannelID)
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
			db = db.Where("product_id LIKE ? OR name LIKE ? OR vip_level LIKE ? OR description LIKE ?", keyword, keyword, keyword, keyword)
		}
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.VideoVIPSubscription
	err := preloadVIPSubscription(db).Order("sort ASC, id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}

func (r *VIPSubscriptionRepo) GetDetail(ctx context.Context, id uint64) (*model.VideoVIPSubscription, error) {
	var item model.VideoVIPSubscription
	if err := preloadVIPSubscription(dbFrom(ctx)).First(&item, id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func preloadVIPSubscription(db *gorm.DB) *gorm.DB {
	return db.Preload("Package").Preload("DisplayPositions").Preload("Channels").Preload("ExcludedChannels")
}

func (r *VIPSubscriptionRepo) UpdateFields(ctx context.Context, item *model.VideoVIPSubscription) error {
	return r.BaseRepo.Update(ctx, item,
		"PackageID", "Platform", "ProductID", "Name", "VIPLevel", "PlanType", "AppVersion", "Currency",
		"FirstSubscriptionPrice", "FirstSubscriptionRevenue", "FirstBonusPoints", "OriginalPrice",
		"VIPDurationDays", "TrialDays", "RenewalText", "BadgeText", "AgreementDefaultChecked",
		"DisplayMode", "Status", "FreeTrial", "IsSubscription", "IsDefault", "SubscriptionDescription",
		"SubscriptionPrice", "SubscriptionRevenue", "SubscriptionPoints", "SubscriptionPeriod",
		"Sort", "Description", "Remark",
	)
}

type VIPSubscriptionTargets struct {
	DisplayPositionIDs []uint64
	ChannelIDs         []uint64
	ExcludedChannelIDs []uint64
}

func (r *VIPSubscriptionRepo) ReplaceTargets(ctx context.Context, item *model.VideoVIPSubscription, targets VIPSubscriptionTargets) error {
	db := dbFrom(ctx)
	associations := []struct {
		name   string
		values interface{}
	}{
		{name: "DisplayPositions", values: displayPositionsFromIDs(targets.DisplayPositionIDs)},
		{name: "Channels", values: channelsFromIDs(targets.ChannelIDs)},
		{name: "ExcludedChannels", values: channelsFromIDs(targets.ExcludedChannelIDs)},
	}
	for _, association := range associations {
		if err := db.Model(item).Association(association.name).Replace(association.values); err != nil {
			return err
		}
	}
	return nil
}

func (r *VIPSubscriptionRepo) ClearDefaults(ctx context.Context, packageID uint64, platform string, exceptID uint64) error {
	db := dbFrom(ctx).Model(&model.VideoVIPSubscription{}).Where("package_id = ? AND platform = ? AND is_default = ?", packageID, platform, true)
	if exceptID != 0 {
		db = db.Where("id <> ?", exceptID)
	}
	return db.Update("is_default", false).Error
}

func (r *VIPSubscriptionRepo) UpdateStatus(ctx context.Context, id uint64, status int8) error {
	return dbFrom(ctx).Model(&model.VideoVIPSubscription{}).Where("id = ?", id).Update("status", status).Error
}

func (r *VIPSubscriptionRepo) UpdateDisplayMode(ctx context.Context, id uint64, mode int8) error {
	return dbFrom(ctx).Model(&model.VideoVIPSubscription{}).Where("id = ?", id).Update("display_mode", mode).Error
}

func (r *VIPSubscriptionRepo) SetDefault(ctx context.Context, item *model.VideoVIPSubscription) error {
	return repositorySetDefault(ctx, r, item)
}

func repositorySetDefault(ctx context.Context, r *VIPSubscriptionRepo, item *model.VideoVIPSubscription) error {
	return Transaction(ctx, func(ctx context.Context) error {
		if err := r.ClearDefaults(ctx, item.PackageID, item.Platform, item.ID); err != nil {
			return err
		}
		item.IsDefault = true
		return dbFrom(ctx).Model(item).Update("is_default", true).Error
	})
}

func (r *VIPSubscriptionRepo) DeleteWithTargets(ctx context.Context, id uint64) error {
	return dbFrom(ctx).Select("DisplayPositions", "Channels", "ExcludedChannels").Delete(&model.VideoVIPSubscription{ID: id}).Error
}

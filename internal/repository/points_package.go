package repository

import (
	"context"
	"fmt"

	"ai-video/internal/gen/model"

	"gorm.io/gorm"
)

type PointsPackageRepo struct {
	BaseRepo[model.VideoPointsPackage]
}

func NewPointsPackageRepo() *PointsPackageRepo { return &PointsPackageRepo{} }

type PointsPackageListFilter struct {
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
		if filter.ChannelID != 0 {
			db = db.Where("EXISTS (SELECT 1 FROM video_points_package_channel vpc JOIN video_channel vc ON vc.channel_code = vpc.channel_code WHERE vpc.product_code = video_points_package.product_code AND vc.channel_id = ?)", filter.ChannelID)
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
	err := buildQuery().Preload("Channels").Order("sort ASC, id DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}

func (r *PointsPackageRepo) GetDetail(ctx context.Context, id uint64) (*model.VideoPointsPackage, error) {
	var item model.VideoPointsPackage
	if err := dbFrom(ctx).Preload("Channels").First(&item, id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *PointsPackageRepo) GetByProductID(ctx context.Context, productID string) (*model.VideoPointsPackage, error) {
	return r.BaseRepo.GetOne(ctx, &QueryOptions{Where: map[string]interface{}{"product_code": productID}})
}

func (r *PointsPackageRepo) ListOptions(ctx context.Context) ([]model.VideoPointsPackage, error) {
	return r.BaseRepo.List(ctx, &QueryOptions{Where: map[string]interface{}{"status": int8(1)}, Order: []string{"sort ASC", "id ASC"}})
}

func (r *PointsPackageRepo) UpdateFields(ctx context.Context, item *model.VideoPointsPackage) error {
	return r.BaseRepo.Update(ctx, item,
		"ProductID", "Name", "Systems", "UserTypes", "ResourceType", "Points",
		"Currency", "SalePrice", "ActualRevenue", "OriginalPrice", "BadgeText", "Description",
		"ButtonText", "IsDefault", "Status", "Sort",
	)
}

func (r *PointsPackageRepo) ReplaceChannels(ctx context.Context, item *model.VideoPointsPackage, channelIDs []uint64) error {
	channels := make([]model.VideoChannel, 0, len(channelIDs))
	if len(channelIDs) > 0 {
		if err := dbFrom(ctx).Where("channel_id IN ?", channelIDs).Find(&channels).Error; err != nil {
			return err
		}
		if len(channels) != len(channelIDs) {
			return fmt.Errorf("one or more channels do not exist")
		}
	}
	return dbFrom(ctx).Model(item).Association("Channels").Replace(channels)
}

func (r *PointsPackageRepo) ClearDefaults(ctx context.Context, resourceType string, exceptID uint64) error {
	db := dbFrom(ctx).Model(&model.VideoPointsPackage{}).
		Where("resource_type = ? AND is_default = ?", resourceType, true)
	if exceptID != 0 {
		db = db.Where("id <> ?", exceptID)
	}
	return db.Update("is_default", false).Error
}

func (r *PointsPackageRepo) SetDefault(ctx context.Context, item *model.VideoPointsPackage) error {
	return Transaction(ctx, func(ctx context.Context) error {
		if err := r.ClearDefaults(ctx, item.ResourceType, item.ID); err != nil {
			return err
		}
		item.IsDefault = true
		return dbFrom(ctx).Model(item).Update("is_default", true).Error
	})
}

func (r *PointsPackageRepo) UpdateStatus(ctx context.Context, id uint64, status int8) error {
	return dbFrom(ctx).Model(&model.VideoPointsPackage{}).Where("id = ?", id).Update("status", status).Error
}

func (r *PointsPackageRepo) DeleteWithChannels(ctx context.Context, id uint64) error {
	return dbFrom(ctx).Select("Channels").Delete(&model.VideoPointsPackage{ID: id}).Error
}

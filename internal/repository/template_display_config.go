package repository

import (
	"context"
	"fmt"

	"ai-video/internal/gen/model"

	"gorm.io/gorm"
)

type TemplateDisplayConfigRepo struct {
	BaseRepo[model.VideoTemplateDisplayConfig]
}

func NewTemplateDisplayConfigRepo() *TemplateDisplayConfigRepo {
	return &TemplateDisplayConfigRepo{}
}

// Create selects Status explicitly so a deliberately disabled configuration
// is not replaced by GORM's database default for the zero value.
func (r *TemplateDisplayConfigRepo) Create(ctx context.Context, item *model.VideoTemplateDisplayConfig) error {
	return dbFrom(ctx).Select("TemplateID", "DisplayPositionKey", "Sort", "Status", "Remark", "CreatedAt", "UpdatedAt").Create(item).Error
}

type TemplateDisplayConfigListFilter struct {
	TemplateID          uint64
	VideoTemplateTypeID uint64
	PositionKey         string
	Status              *int8
	Keyword             string
}

func (r *TemplateDisplayConfigRepo) PageList(ctx context.Context, page, pageSize int, filter *TemplateDisplayConfigListFilter) ([]model.VideoTemplateDisplayConfig, int64, error) {
	buildQuery := func() *gorm.DB {
		db := dbFrom(ctx).Model(&model.VideoTemplateDisplayConfig{})
		if filter == nil {
			return db
		}
		if filter.TemplateID != 0 {
			db = db.Where("template_id = ?", filter.TemplateID)
		}
		if filter.VideoTemplateTypeID != 0 {
			db = db.Where("EXISTS (SELECT 1 FROM video_template vt WHERE vt.id = video_template_display_config.template_id AND vt.video_template_type_id = ? AND vt.deleted_at IS NULL)", filter.VideoTemplateTypeID)
		}
		if filter.PositionKey != "" {
			db = db.Where("position_key = ?", filter.PositionKey)
		}
		if filter.Status != nil {
			db = db.Where("status = ?", *filter.Status)
		}
		if filter.Keyword != "" {
			keyword := "%" + filter.Keyword + "%"
			db = db.Where(`remark LIKE ? OR EXISTS (
				SELECT 1 FROM video_template vt
				WHERE vt.id = video_template_display_config.template_id
					AND vt.deleted_at IS NULL AND vt.name LIKE ?
			) OR EXISTS (
				SELECT 1 FROM video_display_position vdp
				WHERE vdp.position_key = video_template_display_config.position_key
					AND vdp.deleted_at IS NULL AND (vdp.position_name LIKE ? OR vdp.position_key LIKE ?)
			)`, keyword, keyword, keyword, keyword)
		}
		return db
	}

	var total int64
	if err := buildQuery().Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.VideoTemplateDisplayConfig
	err := preloadTemplateDisplayConfig(buildQuery()).
		Order("sort DESC, id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}

func (r *TemplateDisplayConfigRepo) GetDetail(ctx context.Context, id uint64) (*model.VideoTemplateDisplayConfig, error) {
	var item model.VideoTemplateDisplayConfig
	if err := preloadTemplateDisplayConfig(dbFrom(ctx)).First(&item, id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func preloadTemplateDisplayConfig(db *gorm.DB) *gorm.DB {
	return db.Preload("Template.VideoTemplateType").Preload("DisplayPosition")
}

func (r *TemplateDisplayConfigRepo) PairExists(ctx context.Context, templateID uint64, positionKey string, excludeID uint64) (bool, error) {
	db := dbFrom(ctx).Model(&model.VideoTemplateDisplayConfig{}).
		Where("template_id = ? AND position_key = ?", templateID, positionKey)
	if excludeID != 0 {
		db = db.Where("id <> ?", excludeID)
	}
	var count int64
	if err := db.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *TemplateDisplayConfigRepo) UpdateFields(ctx context.Context, item *model.VideoTemplateDisplayConfig) error {
	return r.BaseRepo.Update(ctx, item, "TemplateID", "DisplayPositionKey", "Sort", "Status", "Remark")
}

type ClientTemplateDisplayTargets struct {
	PositionKey       string
	CountryID         uint64
	ChannelIDs        []uint64
	PackageIDs        []uint64
	UserType          uint32
	SubscriptionState string
}

func (r *TemplateDisplayConfigRepo) ListForClient(ctx context.Context, targets ClientTemplateDisplayTargets) ([]model.VideoTemplateDisplayConfig, error) {
	db := dbFrom(ctx).Model(&model.VideoTemplateDisplayConfig{}).
		Joins("JOIN video_template vt ON vt.id = video_template_display_config.template_id AND vt.deleted_at IS NULL").
		Joins("JOIN video_template_type vtt ON vtt.id = vt.video_template_type_id AND vtt.deleted_at IS NULL").
		Joins("JOIN video_display_position vdp ON vdp.position_key = video_template_display_config.position_key AND vdp.deleted_at IS NULL").
		Where("video_template_display_config.position_key = ?", targets.PositionKey).
		Where("video_template_display_config.status = ? AND vt.status = ? AND vtt.status = ? AND vdp.status = ?", 1, 1, 1, 1)

	if targets.CountryID != 0 {
		db = db.Where("(NOT EXISTS (SELECT 1 FROM video_template_country vtc WHERE vtc.template_id = vt.id) OR EXISTS (SELECT 1 FROM video_template_country vtc WHERE vtc.template_id = vt.id AND vtc.country_id = ?))", targets.CountryID).
			Where("(NOT EXISTS (SELECT 1 FROM video_template_type_country vttc WHERE vttc.template_type_id = vtt.id) OR EXISTS (SELECT 1 FROM video_template_type_country vttc WHERE vttc.template_type_id = vtt.id AND vttc.country_id = ?))", targets.CountryID)
	} else {
		db = db.Where("NOT EXISTS (SELECT 1 FROM video_template_country vtc WHERE vtc.template_id = vt.id)").
			Where("NOT EXISTS (SELECT 1 FROM video_template_type_country vttc WHERE vttc.template_type_id = vtt.id)")
	}
	if len(targets.ChannelIDs) > 0 {
		db = db.Where("(NOT EXISTS (SELECT 1 FROM video_template_channel vtc WHERE vtc.template_id = vt.id) OR EXISTS (SELECT 1 FROM video_template_channel vtc WHERE vtc.template_id = vt.id AND vtc.channel_id IN ?))", targets.ChannelIDs).
			Where("(NOT EXISTS (SELECT 1 FROM video_template_type_channel vttc WHERE vttc.template_type_id = vtt.id) OR EXISTS (SELECT 1 FROM video_template_type_channel vttc WHERE vttc.template_type_id = vtt.id AND vttc.channel_id IN ?))", targets.ChannelIDs)
	} else {
		db = db.Where("NOT EXISTS (SELECT 1 FROM video_template_channel vtc WHERE vtc.template_id = vt.id)").
			Where("NOT EXISTS (SELECT 1 FROM video_template_type_channel vttc WHERE vttc.template_type_id = vtt.id)")
	}
	if len(targets.PackageIDs) > 0 {
		db = db.Where("(NOT EXISTS (SELECT 1 FROM video_template_package vtp WHERE vtp.template_id = vt.id) OR EXISTS (SELECT 1 FROM video_template_package vtp WHERE vtp.template_id = vt.id AND vtp.package_id IN ?))", targets.PackageIDs).
			Where("(NOT EXISTS (SELECT 1 FROM video_template_type_package vttp WHERE vttp.template_type_id = vtt.id) OR EXISTS (SELECT 1 FROM video_template_type_package vttp WHERE vttp.template_type_id = vtt.id AND vttp.package_id IN ?))", targets.PackageIDs)
	} else {
		db = db.Where("NOT EXISTS (SELECT 1 FROM video_template_package vtp WHERE vtp.template_id = vt.id)").
			Where("NOT EXISTS (SELECT 1 FROM video_template_type_package vttp WHERE vttp.template_type_id = vtt.id)")
	}
	if targets.UserType != 0 {
		pattern := "%" + fmt.Sprint(targets.UserType) + "%"
		db = db.Where("(COALESCE(vt.user_types, '') IN ('', 'null') OR vt.user_types LIKE ?)", pattern).
			Where("(COALESCE(vtt.user_types, '') IN ('', 'null') OR vtt.user_types LIKE ?)", pattern)
	}
	if targets.SubscriptionState != "" {
		pattern := "%\"" + targets.SubscriptionState + "\"%"
		db = db.Where("(COALESCE(vt.subscription_statuses, '') IN ('', 'null') OR vt.subscription_statuses LIKE ?)", pattern).
			Where("(COALESCE(vtt.subscription_statuses, '') IN ('', 'null') OR vtt.subscription_statuses LIKE ?)", pattern)
	}

	var list []model.VideoTemplateDisplayConfig
	err := db.Preload("Template").
		Order("video_template_display_config.sort DESC, vt.sort DESC, vt.usage_count DESC, vt.favorite_count DESC, vt.view_count DESC, vt.id DESC").
		Find(&list).Error
	return list, err
}

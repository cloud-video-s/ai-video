package repository

import (
	"context"

	"ai-video/internal/domain"
	"ai-video/internal/gen/model"

	"gorm.io/gorm"
)

type BannerRepo struct {
	BaseRepo[model.VideoBanner]
}

func NewBannerRepo() *BannerRepo {
	return &BannerRepo{}
}

type BannerListFilter struct {
	CountryID   uint64
	ChannelID   uint64
	PackageID   uint64
	PositionKey string
	JumpType    uint8
	Status      *int8
	Keyword     string
}

func (r *BannerRepo) PageList(ctx context.Context, page, pageSize int, filter *BannerListFilter) ([]model.VideoBanner, int64, error) {
	dao := dbFrom(ctx).Model(&model.VideoBanner{})
	if filter != nil {
		if filter.PositionKey != "" {
			dao = dao.Where("EXISTS (SELECT 1 FROM video_banner_display_position vbdp JOIN video_display_position vdp ON vdp.position_key = vbdp.position_key WHERE vbdp.banner_id = video_banner.id AND vbdp.position_key = ? AND vdp.deleted_at IS NULL)", filter.PositionKey)
		}
		if filter.CountryID != 0 {
			dao = dao.Where(`EXISTS (
				SELECT 1 FROM video_banner_country vbc
				JOIN video_country vc ON vc.code = vbc.country_code
				WHERE vbc.banner_id = video_banner.id AND vc.id = ? AND vbc.deleted_at IS NULL
			)`, filter.CountryID)
		}
		if filter.ChannelID != 0 {
			dao = dao.Where("EXISTS (SELECT 1 FROM video_banner_channel vbc WHERE vbc.banner_id = video_banner.id AND vbc.channel_code = ? AND vbc.deleted_at IS NULL)", filter.ChannelID)
		}
		if filter.PackageID != 0 {
			dao = dao.Where(`EXISTS (
				SELECT 1 FROM video_banner_package vbp
				JOIN video_package vp ON vp.package_code = vbp.package_code
				WHERE vbp.banner_id = video_banner.id AND vp.id = ? AND vbp.deleted_at IS NULL
			)`, filter.PackageID)
		}
		if filter.JumpType != 0 {
			dao = dao.Where("jump_type = ?", filter.JumpType)
		}
		if filter.Status != nil {
			dao = dao.Where("status = ?", *filter.Status)
		}
		if filter.Keyword != "" {
			keyword := "%" + filter.Keyword + "%"
			dao = dao.Where("name LIKE ? OR remark LIKE ?", keyword, keyword)
		}
	}
	var total int64
	if err := dao.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.VideoBanner
	err := preloadBannerTargets(dao).Order("sort ASC, id DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}

func (r *BannerRepo) GetDetail(ctx context.Context, id uint64) (*model.VideoBanner, error) {
	var item model.VideoBanner
	if err := preloadBannerTargets(dbFrom(ctx)).First(&item, id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func preloadBannerTargets(db *gorm.DB) *gorm.DB {
	return db.Preload("Template").Preload("DisplayPositions").Preload("Countries").Preload("Channels").Preload("Packages")
}

func (r *BannerRepo) UpdateFields(ctx context.Context, item *model.VideoBanner) error {
	return r.BaseRepo.Update(ctx, item,
		"Name", "CoverImage", "Remark", "Sort", "JumpType", "JumpURL", "TemplateID", "Status",
	)
}

type BannerTargetIDs struct {
	DisplayPositionKeys []string
	CountryIDs          []uint64
	ChannelIDs          []uint64
	PackageIDs          []uint64
}

type ClientBannerTargets struct {
	PositionKey string
	CountryID   uint64
	ChannelIDs  []uint64
	PackageIDs  []uint64
}

// ListForClient applies delivery targeting. An empty association means the
// banner is global for that dimension; otherwise the client must match.
func (r *BannerRepo) ListForClient(ctx context.Context, targets ClientBannerTargets) ([]model.VideoBanner, error) {
	db := dbFrom(ctx).Model(&model.VideoBanner{}).
		Where("video_banner.status = ?", 1).
		Where("EXISTS (SELECT 1 FROM video_banner_display_position vbdp JOIN video_display_position vdp ON vdp.position_key = vbdp.position_key WHERE vbdp.banner_id = video_banner.id AND vbdp.position_key = ? AND vdp.status = ? AND vdp.deleted_at IS NULL)", targets.PositionKey, 1)
	if targets.CountryID != 0 {
		db = db.Where(`(NOT EXISTS (
			SELECT 1 FROM video_banner_country vbc WHERE vbc.banner_id = video_banner.id AND vbc.deleted_at IS NULL
		) OR EXISTS (
			SELECT 1 FROM video_banner_country vbc JOIN video_country vc ON vc.code = vbc.country_code
			WHERE vbc.banner_id = video_banner.id AND vc.id = ? AND vbc.deleted_at IS NULL
		))`, targets.CountryID)
	} else {
		db = db.Where("NOT EXISTS (SELECT 1 FROM video_banner_country vbc WHERE vbc.banner_id = video_banner.id AND vbc.deleted_at IS NULL)")
	}
	if len(targets.ChannelIDs) > 0 {
		db = db.Where(`(NOT EXISTS (
			SELECT 1 FROM video_banner_channel vbc WHERE vbc.banner_id = video_banner.id AND vbc.deleted_at IS NULL
		) OR EXISTS (
			SELECT 1 FROM video_banner_channel vbc
			WHERE vbc.banner_id = video_banner.id AND vbc.channel_code IN ? AND vbc.deleted_at IS NULL
		))`, targets.ChannelIDs)
	} else {
		db = db.Where("NOT EXISTS (SELECT 1 FROM video_banner_channel vbc WHERE vbc.banner_id = video_banner.id AND vbc.deleted_at IS NULL)")
	}
	if len(targets.PackageIDs) > 0 {
		db = db.Where(`(NOT EXISTS (
			SELECT 1 FROM video_banner_package vbp WHERE vbp.banner_id = video_banner.id AND vbp.deleted_at IS NULL
		) OR EXISTS (
			SELECT 1 FROM video_banner_package vbp JOIN video_package vp ON vp.package_code = vbp.package_code
			WHERE vbp.banner_id = video_banner.id AND vp.id IN ? AND vbp.deleted_at IS NULL
		))`, targets.PackageIDs)
	} else {
		db = db.Where("NOT EXISTS (SELECT 1 FROM video_banner_package vbp WHERE vbp.banner_id = video_banner.id AND vbp.deleted_at IS NULL)")
	}
	db = db.Where("(video_banner.jump_type <> ? OR EXISTS (SELECT 1 FROM video_template vt WHERE vt.id = video_banner.template_id AND vt.status = ? AND vt.deleted_at IS NULL))", domain.BannerJumpTypeTemplate, 1)
	var list []model.VideoBanner
	err := db.Preload("Template").Preload("DisplayPositions", "status = ?", 1).
		Order("video_banner.sort ASC, video_banner.id DESC").Find(&list).Error
	return list, err
}

func (r *BannerRepo) ReplaceTargets(ctx context.Context, item *model.VideoBanner, targets BannerTargetIDs) error {
	db := dbFrom(ctx)
	countries, err := loadCountriesByIDs(db, targets.CountryIDs)
	if err != nil {
		return err
	}
	channels, err := loadChannelsByIDs(db, targets.ChannelIDs)
	if err != nil {
		return err
	}
	packages, err := loadPackagesByIDs(db, targets.PackageIDs)
	if err != nil {
		return err
	}
	positionKeys := targets.DisplayPositionKeys
	if err := db.Where("banner_id = ?", item.ID).Delete(&model.VideoBannerDisplayPosition{}).Error; err != nil {
		return err
	}
	if len(positionKeys) > 0 {
		rows := make([]model.VideoBannerDisplayPosition, 0, len(positionKeys))
		for _, key := range positionKeys {
			rows = append(rows, model.VideoBannerDisplayPosition{BannerID: item.ID, PositionKey: key})
		}
		if err := db.Create(&rows).Error; err != nil {
			return err
		}
	}
	associations := []struct {
		name   string
		values interface{}
	}{
		{name: "Countries", values: countries},
		{name: "Channels", values: channels},
		{name: "Packages", values: packages},
	}
	for _, association := range associations {
		if err := db.Model(item).Association(association.name).Replace(association.values); err != nil {
			return err
		}
	}
	return nil
}

func (r *BannerRepo) DeleteWithTargets(ctx context.Context, id uint64) error {
	db := dbFrom(ctx)
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("banner_id = ?", id).Delete(&model.VideoBannerDisplayPosition{}).Error; err != nil {
			return err
		}
		return tx.Select("Countries", "Channels", "Packages").Delete(&model.VideoBanner{ID: id}).Error
	})
}

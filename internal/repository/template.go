package repository

import (
	"context"
	"fmt"

	"ai-video/internal/model"

	"gorm.io/gorm"
)

type TemplateTypeRepo struct {
	BaseRepo[model.VideoTemplateType]
}

func NewTemplateTypeRepo() *TemplateTypeRepo {
	return &TemplateTypeRepo{}
}

type TemplateTypeListFilter struct {
	Status      *int8
	PositionKey string
	CountryID   uint64
	ChannelID   uint64
	PackageID   uint64
	Keyword     string
}

func (r *TemplateTypeRepo) PageList(ctx context.Context, page, pageSize int, filter *TemplateTypeListFilter) ([]model.VideoTemplateType, int64, error) {
	buildQuery := func() *gorm.DB {
		db := dbFrom(ctx).Model(&model.VideoTemplateType{})
		if filter == nil {
			return db
		}
		if filter.Status != nil {
			db = db.Where("status = ?", *filter.Status)
		}
		if filter.PositionKey != "" {
			db = db.Where(`EXISTS (
				SELECT 1 FROM video_template_type_display_position vttdp
				WHERE vttdp.template_type_id = video_template_type.id AND vttdp.position_key = ?
			)`, filter.PositionKey)
		}
		if filter.CountryID != 0 {
			db = db.Where("EXISTS (SELECT 1 FROM video_template_type_country vttc WHERE vttc.template_type_id = video_template_type.id AND vttc.country_id = ?)", filter.CountryID)
		}
		if filter.ChannelID != 0 {
			db = db.Where("EXISTS (SELECT 1 FROM video_template_type_channel vttc WHERE vttc.template_type_id = video_template_type.id AND vttc.channel_id = ?)", filter.ChannelID)
		}
		if filter.PackageID != 0 {
			db = db.Where("EXISTS (SELECT 1 FROM video_template_type_package vttp WHERE vttp.template_type_id = video_template_type.id AND vttp.package_id = ?)", filter.PackageID)
		}
		if filter.Keyword != "" {
			keyword := "%" + filter.Keyword + "%"
			db = db.Where(`category_name LIKE ? OR description LIKE ? OR EXISTS (
				SELECT 1 FROM video_template_type_display_position vttdp
				JOIN video_display_position vdp ON vdp.position_key = vttdp.position_key
				WHERE vttdp.template_type_id = video_template_type.id
				AND (vdp.position_name LIKE ? OR vdp.position_key LIKE ?)
			) OR EXISTS (
				SELECT 1 FROM video_template_type_country vttc JOIN video_country vc ON vc.id = vttc.country_id
				WHERE vttc.template_type_id = video_template_type.id AND (vc.code LIKE ? OR vc.name_zh LIKE ?)
			) OR EXISTS (
				SELECT 1 FROM video_template_type_channel vttc JOIN video_channel vc ON vc.channel_id = vttc.channel_id
				WHERE vttc.template_type_id = video_template_type.id AND (vc.channel_code LIKE ? OR vc.channel_name LIKE ?)
			) OR EXISTS (
				SELECT 1 FROM video_template_type_package vttp JOIN video_package vp ON vp.id = vttp.package_id
				WHERE vttp.template_type_id = video_template_type.id AND (vp.package_code LIKE ? OR vp.package_name LIKE ?)
			)`, keyword, keyword, keyword, keyword, keyword, keyword, keyword, keyword, keyword, keyword)
		}
		return db
	}

	var total int64
	if err := buildQuery().Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.VideoTemplateType
	err := preloadTemplateTypeTargets(buildQuery()).Order("sort ASC, id DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}

func (r *TemplateTypeRepo) ListOptions(ctx context.Context) ([]model.VideoTemplateType, error) {
	var list []model.VideoTemplateType
	err := preloadTemplateTypeTargets(dbFrom(ctx)).Order("sort ASC, id ASC").Find(&list).Error
	return list, err
}

func (r *TemplateTypeRepo) GetDetail(ctx context.Context, id uint64) (*model.VideoTemplateType, error) {
	var item model.VideoTemplateType
	if err := preloadTemplateTypeTargets(dbFrom(ctx)).First(&item, id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *TemplateTypeRepo) UpdateFields(ctx context.Context, item *model.VideoTemplateType) error {
	return r.BaseRepo.Update(ctx, item,
		"CategoryName", "Sort", "Status", "Description", "LegacyCountry", "LegacyAppPackage", "LegacyChannelID",
		"LegacyUserType", "LegacyIsSubscribed", "LegacyPackageID", "UserTypes", "SubscriptionStatuses",
	)
}

func preloadTemplateTypeTargets(db *gorm.DB) *gorm.DB {
	return db.Preload("DisplayPositions").Preload("Countries").Preload("Channels").Preload("Packages")
}

func (r *TemplateTypeRepo) ReplaceDisplayPositions(ctx context.Context, item *model.VideoTemplateType, keys []string) error {
	return dbFrom(ctx).Model(item).Association("DisplayPositions").Replace(displayPositionsFromKeys(keys))
}

type TemplateTypeTargetIDs struct {
	DisplayPositionKeys []string
	CountryIDs          []uint64
	ChannelIDs          []uint64
	PackageIDs          []uint64
}

func (r *TemplateTypeRepo) ReplaceTargets(ctx context.Context, item *model.VideoTemplateType, targets TemplateTypeTargetIDs) error {
	associations := []struct {
		name   string
		values interface{}
	}{
		{name: "DisplayPositions", values: displayPositionsFromKeys(targets.DisplayPositionKeys)},
		{name: "Countries", values: countriesFromIDs(targets.CountryIDs)},
		{name: "Channels", values: channelsFromIDs(targets.ChannelIDs)},
		{name: "Packages", values: packagesFromIDs(targets.PackageIDs)},
	}
	for _, association := range associations {
		if err := dbFrom(ctx).Model(item).Association(association.name).Replace(association.values); err != nil {
			return err
		}
	}
	return nil
}

func (r *TemplateTypeRepo) DeleteWithDisplayPositions(ctx context.Context, id uint64) error {
	return dbFrom(ctx).Select("DisplayPositions", "Countries", "Channels", "Packages").Delete(&model.VideoTemplateType{ID: id}).Error
}

func (r *TemplateTypeRepo) TemplateCount(ctx context.Context, typeID uint64) (int64, error) {
	var count int64
	err := dbFrom(ctx).Model(&model.VideoTemplate{}).
		Where("video_template_type_id = ?", typeID).
		Count(&count).Error
	return count, err
}

type ClientTemplateTypeTargets struct {
	PositionKey       string
	CountryID         uint64
	ChannelIDs        []uint64
	PackageIDs        []uint64
	UserType          uint32
	SubscriptionState string
}

func (r *TemplateTypeRepo) ListForClient(ctx context.Context, targets ClientTemplateTypeTargets) ([]model.VideoTemplateType, error) {
	db := dbFrom(ctx).Model(&model.VideoTemplateType{}).
		Where("video_template_type.status = ?", 1).
		Where(`EXISTS (
			SELECT 1 FROM video_template_type_display_position vttdp
			JOIN video_display_position vdp ON vdp.position_key = vttdp.position_key
			WHERE vttdp.template_type_id = video_template_type.id
				AND vttdp.position_key = ? AND vdp.status = ? AND vdp.deleted_at IS NULL
		)`, targets.PositionKey, 1)
	if targets.CountryID != 0 {
		db = db.Where("(NOT EXISTS (SELECT 1 FROM video_template_type_country vttc WHERE vttc.template_type_id = video_template_type.id) OR EXISTS (SELECT 1 FROM video_template_type_country vttc WHERE vttc.template_type_id = video_template_type.id AND vttc.country_id = ?))", targets.CountryID)
	} else {
		db = db.Where("NOT EXISTS (SELECT 1 FROM video_template_type_country vttc WHERE vttc.template_type_id = video_template_type.id)")
	}
	if len(targets.ChannelIDs) > 0 {
		db = db.Where("(NOT EXISTS (SELECT 1 FROM video_template_type_channel vttc WHERE vttc.template_type_id = video_template_type.id) OR EXISTS (SELECT 1 FROM video_template_type_channel vttc WHERE vttc.template_type_id = video_template_type.id AND vttc.channel_id IN ?))", targets.ChannelIDs)
	} else {
		db = db.Where("NOT EXISTS (SELECT 1 FROM video_template_type_channel vttc WHERE vttc.template_type_id = video_template_type.id)")
	}
	if len(targets.PackageIDs) > 0 {
		db = db.Where("(NOT EXISTS (SELECT 1 FROM video_template_type_package vttp WHERE vttp.template_type_id = video_template_type.id) OR EXISTS (SELECT 1 FROM video_template_type_package vttp WHERE vttp.template_type_id = video_template_type.id AND vttp.package_id IN ?))", targets.PackageIDs)
	} else {
		db = db.Where("NOT EXISTS (SELECT 1 FROM video_template_type_package vttp WHERE vttp.template_type_id = video_template_type.id)")
	}
	if targets.UserType != 0 {
		db = db.Where("(COALESCE(user_types, '') IN ('', 'null') OR user_types LIKE ?)", "%"+fmt.Sprint(targets.UserType)+"%")
	}
	if targets.SubscriptionState != "" {
		db = db.Where("(COALESCE(subscription_statuses, '') IN ('', 'null') OR subscription_statuses LIKE ?)", "%\""+targets.SubscriptionState+"\"%")
	}
	var list []model.VideoTemplateType
	err := db.Preload("DisplayPositions", "status = ?", 1).Preload("Countries").Preload("Channels").Preload("Packages").
		Order("sort DESC, id DESC").Find(&list).Error
	return list, err
}

type TemplateRepo struct {
	BaseRepo[model.VideoTemplate]
}

func NewTemplateRepo() *TemplateRepo {
	return &TemplateRepo{}
}

type TemplateListFilter struct {
	VideoTemplateTypeID uint64
	PositionKey         string
	CountryID           uint64
	PackageID           uint64
	ChannelID           uint64
	UserType            uint8
	SubscriptionStatus  string
	TemplateType        string
	Status              *int8
	Keyword             string
}

func (r *TemplateRepo) PageList(ctx context.Context, page, pageSize int, filter *TemplateListFilter) ([]model.VideoTemplate, int64, error) {
	buildQuery := func() *gorm.DB {
		dao := dbFrom(ctx).Model(&model.VideoTemplate{})
		if filter == nil {
			return dao
		}
		if filter.VideoTemplateTypeID != 0 {
			dao = dao.Where("video_template_type_id = ?", filter.VideoTemplateTypeID)
		}
		if filter.PositionKey != "" {
			dao = dao.Where(`EXISTS (
				SELECT 1 FROM video_template_type_display_position vttdp
				WHERE vttdp.template_type_id = video_template.video_template_type_id
					AND vttdp.position_key = ?
			)`, filter.PositionKey)
		}
		if filter.CountryID != 0 {
			dao = dao.Where("EXISTS (SELECT 1 FROM video_template_country vtc WHERE vtc.template_id = video_template.id AND vtc.country_id = ?)", filter.CountryID)
		}
		if filter.PackageID != 0 {
			dao = dao.Where("EXISTS (SELECT 1 FROM video_template_package vtp WHERE vtp.template_id = video_template.id AND vtp.package_id = ?)", filter.PackageID)
		}
		if filter.ChannelID != 0 {
			dao = dao.Where("EXISTS (SELECT 1 FROM video_template_channel vtc WHERE vtc.template_id = video_template.id AND vtc.channel_id = ?)", filter.ChannelID)
		}
		if filter.UserType != 0 {
			dao = dao.Where("user_types LIKE ?", "%"+fmt.Sprint(filter.UserType)+"%")
		}
		if filter.SubscriptionStatus != "" {
			dao = dao.Where("subscription_statuses LIKE ?", "%\""+filter.SubscriptionStatus+"\"%")
		}
		if filter.TemplateType != "" {
			dao = dao.Where("template_type = ?", filter.TemplateType)
		}
		if filter.Status != nil {
			dao = dao.Where("status = ?", *filter.Status)
		}
		if filter.Keyword != "" {
			keyword := "%" + filter.Keyword + "%"
			dao = dao.Where("name LIKE ? OR prompt LIKE ? OR description LIKE ?", keyword, keyword, keyword)
		}
		return dao
	}

	var total int64
	if err := buildQuery().Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.VideoTemplate
	err := preloadTemplateCategoryTargets(buildQuery()).Preload("Countries").
		Preload("Packages").Preload("Channels").Order("sort ASC, id DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}

func (r *TemplateRepo) GetWithType(ctx context.Context, id uint64) (*model.VideoTemplate, error) {
	var item model.VideoTemplate
	err := preloadTemplateCategoryTargets(dbFrom(ctx)).Preload("Countries").
		Preload("Packages").Preload("Channels").First(&item, id).Error
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *TemplateRepo) ListOptions(ctx context.Context) ([]model.VideoTemplate, error) {
	var list []model.VideoTemplate
	err := dbFrom(ctx).Preload("VideoTemplateType").Order("status DESC, sort DESC, id DESC").Find(&list).Error
	return list, err
}

type ClientTemplateTargets struct {
	TemplateTypeIDs    []uint64
	CountryID          uint64
	ChannelIDs         []uint64
	PackageIDs         []uint64
	UserType           uint32
	SubscriptionStatus string
}

func (r *TemplateRepo) ListForClient(ctx context.Context, targets ClientTemplateTargets) ([]model.VideoTemplate, error) {
	if len(targets.TemplateTypeIDs) == 0 {
		return []model.VideoTemplate{}, nil
	}
	db := dbFrom(ctx).Model(&model.VideoTemplate{}).
		Where("video_template.status = ?", 1).
		Where("video_template.video_template_type_id IN ?", targets.TemplateTypeIDs)
	if targets.CountryID != 0 {
		db = db.Where("(NOT EXISTS (SELECT 1 FROM video_template_country vtc WHERE vtc.template_id = video_template.id) OR EXISTS (SELECT 1 FROM video_template_country vtc WHERE vtc.template_id = video_template.id AND vtc.country_id = ?))", targets.CountryID)
	} else {
		db = db.Where("NOT EXISTS (SELECT 1 FROM video_template_country vtc WHERE vtc.template_id = video_template.id)")
	}
	if len(targets.ChannelIDs) > 0 {
		db = db.Where("(NOT EXISTS (SELECT 1 FROM video_template_channel vtc WHERE vtc.template_id = video_template.id) OR EXISTS (SELECT 1 FROM video_template_channel vtc WHERE vtc.template_id = video_template.id AND vtc.channel_id IN ?))", targets.ChannelIDs)
	} else {
		db = db.Where("NOT EXISTS (SELECT 1 FROM video_template_channel vtc WHERE vtc.template_id = video_template.id)")
	}
	if len(targets.PackageIDs) > 0 {
		db = db.Where("(NOT EXISTS (SELECT 1 FROM video_template_package vtp WHERE vtp.template_id = video_template.id) OR EXISTS (SELECT 1 FROM video_template_package vtp WHERE vtp.template_id = video_template.id AND vtp.package_id IN ?))", targets.PackageIDs)
	} else {
		db = db.Where("NOT EXISTS (SELECT 1 FROM video_template_package vtp WHERE vtp.template_id = video_template.id)")
	}
	if targets.UserType != 0 {
		db = db.Where("(COALESCE(user_types, '') IN ('', 'null') OR user_types LIKE ?)", "%"+fmt.Sprint(targets.UserType)+"%")
	}
	if targets.SubscriptionStatus != "" {
		db = db.Where("(COALESCE(subscription_statuses, '') IN ('', 'null') OR subscription_statuses LIKE ?)", "%\""+targets.SubscriptionStatus+"\"%")
	}
	var list []model.VideoTemplate
	err := preloadTemplateCategoryTargets(db).
		Order("video_template.sort DESC, video_template.usage_count DESC, video_template.favorite_count DESC, video_template.view_count DESC, video_template.id DESC").
		Find(&list).Error
	return list, err
}

func preloadTemplateCategoryTargets(db *gorm.DB) *gorm.DB {
	return db.Preload("VideoTemplateType.DisplayPositions").Preload("VideoTemplateType.Countries").
		Preload("VideoTemplateType.Channels").Preload("VideoTemplateType.Packages")
}

func (r *TemplateRepo) UpdateFields(ctx context.Context, item *model.VideoTemplate) error {
	return r.BaseRepo.Update(ctx, item,
		"VideoTemplateTypeID", "UserTypes", "SubscriptionStatuses", "Name", "TemplateType", "Sort",
		"CoverImage", "TemplateVideo", "ThumbnailVideo", "Prompt", "Status", "Description",
	)
}

func (r *TemplateRepo) ReplaceTargets(ctx context.Context, item *model.VideoTemplate, req TemplateTargetIDs) error {
	db := dbFrom(ctx)
	associations := []struct {
		name   string
		values interface{}
	}{
		{name: "Countries", values: countriesFromIDs(req.CountryIDs)},
		{name: "Packages", values: packagesFromIDs(req.PackageIDs)},
		{name: "Channels", values: channelsFromIDs(req.ChannelIDs)},
	}
	for _, association := range associations {
		if err := db.Model(item).Association(association.name).Replace(association.values); err != nil {
			return err
		}
	}
	return nil
}

type TemplateTargetIDs struct {
	CountryIDs []uint64
	PackageIDs []uint64
	ChannelIDs []uint64
}

func displayPositionsFromIDs(ids []uint64) []model.VideoDisplayPosition {
	items := make([]model.VideoDisplayPosition, 0, len(ids))
	for _, id := range ids {
		items = append(items, model.VideoDisplayPosition{ID: id})
	}
	return items
}

func displayPositionsFromKeys(keys []string) []model.VideoDisplayPosition {
	items := make([]model.VideoDisplayPosition, 0, len(keys))
	for _, key := range keys {
		items = append(items, model.VideoDisplayPosition{PositionKey: key})
	}
	return items
}

func countriesFromIDs(ids []uint64) []model.VideoCountry {
	items := make([]model.VideoCountry, 0, len(ids))
	for _, id := range ids {
		items = append(items, model.VideoCountry{ID: id})
	}
	return items
}

func packagesFromIDs(ids []uint64) []model.VideoPackage {
	items := make([]model.VideoPackage, 0, len(ids))
	for _, id := range ids {
		items = append(items, model.VideoPackage{ID: id})
	}
	return items
}

func channelsFromIDs(ids []uint64) []model.VideoChannel {
	items := make([]model.VideoChannel, 0, len(ids))
	for _, id := range ids {
		items = append(items, model.VideoChannel{ChannelID: id})
	}
	return items
}

func (r *TemplateRepo) DeleteWithTargets(ctx context.Context, id uint64) error {
	db := dbFrom(ctx)
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Unscoped().Where("template_id = ?", id).Delete(&model.VideoTemplateDisplayConfig{}).Error; err != nil {
			return err
		}
		return tx.Select("Countries", "Packages", "Channels").Delete(&model.VideoTemplate{ID: id}).Error
	})
}

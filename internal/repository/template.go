package repository

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"ai-video/internal/gen/model"

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
	AppCode     string
	PackageCode string
	VersionCode string
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
			db = db.Where("EXISTS (SELECT 1 FROM video_template_type_country vttc JOIN video_country vc ON vc.code = vttc.country_code WHERE vttc.template_type_id = video_template_type.id AND vc.id = ? AND vttc.deleted_at IS NULL)", filter.CountryID)
		}
		if filter.AppCode != "" {
			db = db.Where("EXISTS (SELECT 1 FROM video_template_type_app vtta WHERE vtta.template_type_id = video_template_type.id AND vtta.app_code = ? AND vtta.deleted_at IS NULL)", filter.AppCode)
		}
		if filter.PackageCode != "" {
			db = db.Where("EXISTS (SELECT 1 FROM video_template_type_app vtta WHERE vtta.template_type_id = video_template_type.id AND vtta.package_code = ? AND vtta.deleted_at IS NULL)", filter.PackageCode)
		}
		if filter.VersionCode != "" {
			db = db.Where("EXISTS (SELECT 1 FROM video_template_type_app vtta WHERE vtta.template_type_id = video_template_type.id AND vtta.version_code = ? AND vtta.deleted_at IS NULL)", filter.VersionCode)
		}
		if filter.Keyword != "" {
			keyword := "%" + filter.Keyword + "%"
			db = db.Where(`category_name LIKE ? OR description LIKE ? OR EXISTS (
				SELECT 1 FROM video_template_type_display_position vttdp
				JOIN video_display_position vdp ON vdp.position_key = vttdp.position_key
				WHERE vttdp.template_type_id = video_template_type.id
				AND (vdp.position_name LIKE ? OR vdp.position_key LIKE ?)
			) OR EXISTS (
				SELECT 1 FROM video_template_type_country vttc JOIN video_country vc ON vc.code = vttc.country_code
				WHERE vttc.template_type_id = video_template_type.id AND (vc.code LIKE ? OR vc.name_zh LIKE ?)
			) OR EXISTS (
				SELECT 1 FROM video_template_type_app vtta
				WHERE vtta.template_type_id = video_template_type.id AND vtta.deleted_at IS NULL
					AND (vtta.app_code LIKE ? OR vtta.package_code LIKE ? OR vtta.version_code LIKE ?)
			)`, keyword, keyword, keyword, keyword, keyword, keyword, keyword, keyword, keyword)
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
		"CategoryName", "Sort", "Status", "Description", "IsSubscribed", "UserTypes", "SubscriptionStatuses",
	)
}

func preloadTemplateTypeTargets(db *gorm.DB) *gorm.DB {
	return db.Preload("DisplayPositions").Preload("Countries").Preload("AppRules")
}

func (r *TemplateTypeRepo) ReplaceDisplayPositions(ctx context.Context, item *model.VideoTemplateType, keys []string) error {
	db := dbFrom(ctx)
	positions, err := loadDisplayPositionsByKeys(db, keys)
	if err != nil {
		return err
	}
	return db.Model(item).Association("DisplayPositions").Replace(positions)
}

type TemplateTypeTargetIDs struct {
	DisplayPositionKeys []string
	CountryIDs          []uint64
	AppRules            []TemplateTypeAppRule
}

// TemplateTypeAppRule is one exact APP/package/version delivery rule.
// An empty rule list means the template type is available to every app.
type TemplateTypeAppRule struct {
	AppCode     string
	PackageCode string
	VersionCode string
}

func (r *TemplateTypeRepo) ReplaceTargets(ctx context.Context, item *model.VideoTemplateType, targets TemplateTypeTargetIDs) error {
	db := dbFrom(ctx)
	positions, err := loadDisplayPositionsByKeys(db, targets.DisplayPositionKeys)
	if err != nil {
		return err
	}
	countries, err := loadCountriesByIDs(db, targets.CountryIDs)
	if err != nil {
		return err
	}
	associations := []struct {
		name   string
		values interface{}
	}{
		{name: "DisplayPositions", values: positions},
		{name: "Countries", values: countries},
	}
	for _, association := range associations {
		if err := db.Model(item).Association(association.name).Replace(association.values); err != nil {
			return err
		}
	}
	if err := db.Unscoped().Where("template_type_id = ?", item.ID).Delete(&model.VideoTemplateTypeApp{}).Error; err != nil {
		return err
	}
	if len(targets.AppRules) == 0 {
		return nil
	}
	rules := make([]model.VideoTemplateTypeApp, 0, len(targets.AppRules))
	for _, rule := range targets.AppRules {
		rules = append(rules, model.VideoTemplateTypeApp{
			ID: nextTemplateTypeAppID(), TemplateTypeID: item.ID,
			AppCode: rule.AppCode, PackageCode: rule.PackageCode, VersionCode: rule.VersionCode,
		})
	}
	return db.Create(&rules).Error

}

func (r *TemplateTypeRepo) DeleteWithDisplayPositions(ctx context.Context, id uint64) error {
	db := dbFrom(ctx)
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Unscoped().Where("template_type_id = ?", id).Delete(&model.VideoTemplateTypeApp{}).Error; err != nil {
			return err
		}
		return tx.Select("DisplayPositions", "Countries").Delete(&model.VideoTemplateType{ID: id}).Error
	})
}

var templateTypeAppID atomic.Uint64

func nextTemplateTypeAppID() uint64 {
	now := uint64(time.Now().UnixNano())
	for {
		last := templateTypeAppID.Load()
		next := now
		if next <= last {
			next = last + 1
		}
		if templateTypeAppID.CompareAndSwap(last, next) {
			return next
		}
	}
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
	AppCode           string
	PackageCode       string
	VersionCode       string
	UserType          uint8
	SubscriptionState string
}

func (r *TemplateTypeRepo) ListForClient(ctx context.Context, targets ClientTemplateTypeTargets) ([]model.VideoTemplateType, error) {
	db := dbFrom(ctx).Model(&model.VideoTemplateType{}).
		Where("video_template_type.status = ?", 1).
		Where(`(NOT EXISTS (
			SELECT 1 FROM video_template_type_display_position all_positions
			WHERE all_positions.template_type_id = video_template_type.id AND all_positions.deleted_at IS NULL
		) OR EXISTS (
			SELECT 1 FROM video_template_type_display_position vttdp
			JOIN video_display_position vdp ON vdp.position_key = vttdp.position_key
			WHERE vttdp.template_type_id = video_template_type.id
				AND vttdp.position_key = ? AND vttdp.deleted_at IS NULL
				AND vdp.status = ? AND vdp.deleted_at IS NULL
		))`, targets.PositionKey, 1)
	if targets.CountryID != 0 {
		db = db.Where(`(NOT EXISTS (
			SELECT 1 FROM video_template_type_country vttc WHERE vttc.template_type_id = video_template_type.id AND vttc.deleted_at IS NULL
		) OR EXISTS (
			SELECT 1 FROM video_template_type_country vttc JOIN video_country vc ON vc.code = vttc.country_code
			WHERE vttc.template_type_id = video_template_type.id AND vc.id = ? AND vttc.deleted_at IS NULL
		))`, targets.CountryID)
	} else {
		db = db.Where("NOT EXISTS (SELECT 1 FROM video_template_type_country vttc WHERE vttc.template_type_id = video_template_type.id AND vttc.deleted_at IS NULL)")
	}
	db = db.Where(`(NOT EXISTS (
		SELECT 1 FROM video_template_type_app vtta
		WHERE vtta.template_type_id = video_template_type.id AND vtta.deleted_at IS NULL
	) OR EXISTS (
		SELECT 1 FROM video_template_type_app vtta
		WHERE vtta.template_type_id = video_template_type.id AND vtta.deleted_at IS NULL
			AND vtta.app_code = ? AND vtta.package_code = ? AND vtta.version_code = ?
	))`, targets.AppCode, targets.PackageCode, targets.VersionCode)
	if targets.UserType != 0 {
		db = db.Where("(COALESCE(user_types, '') IN ('', 'null') OR user_types LIKE ?)", "%"+fmt.Sprint(targets.UserType)+"%")
	}
	if targets.SubscriptionState != "" {
		db = db.Where("(COALESCE(subscription_statuses, '') IN ('', 'null') OR subscription_statuses LIKE ?)", "%\""+targets.SubscriptionState+"\"%")
	}
	var list []model.VideoTemplateType
	err := db.Preload("DisplayPositions", "status = ?", 1).Preload("Countries").Preload("AppRules").
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
			dao = dao.Where("video_template.video_template_type_id = ?", filter.VideoTemplateTypeID)
		}
		if filter.PositionKey != "" {
			dao = dao.Where(`EXISTS (
				SELECT 1 FROM video_template_display_config vtdc
				WHERE vtdc.template_id = video_template.id
					AND vtdc.position_key = ? AND vtdc.deleted_at IS NULL
			)`, filter.PositionKey)
		}
		if filter.UserType != 0 {
			dao = dao.Where("video_template.user_types LIKE ?", "%"+fmt.Sprint(filter.UserType)+"%")
		}
		if filter.SubscriptionStatus != "" {
			dao = dao.Where("video_template.subscription_statuses LIKE ?", "%\""+filter.SubscriptionStatus+"\"%")
		}
		if filter.TemplateType != "" {
			dao = dao.Where("video_template.template_type = ?", filter.TemplateType)
		}
		if filter.Status != nil {
			dao = dao.Where("video_template.status = ?", *filter.Status)
		}
		if filter.Keyword != "" {
			keyword := "%" + filter.Keyword + "%"
			dao = dao.Where("video_template.name LIKE ? OR video_template.prompt LIKE ? OR video_template.description LIKE ?", keyword, keyword, keyword)
		}
		return dao
	}

	var total int64
	if err := buildQuery().Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.VideoTemplate
	err := preloadTemplateCategoryTargets(buildQuery()).Preload("DisplayConfigs.DisplayPosition").
		Order("video_template.sort ASC, video_template.id DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}

func (r *TemplateRepo) GetWithType(ctx context.Context, id uint64) (*model.VideoTemplate, error) {
	var item model.VideoTemplate
	err := preloadTemplateCategoryTargets(dbFrom(ctx)).Preload("DisplayConfigs.DisplayPosition").First(&item, id).Error
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
	UserType           uint8
	SubscriptionStatus string
}

func (r *TemplateRepo) ListForClient(ctx context.Context, targets ClientTemplateTargets) ([]model.VideoTemplate, error) {
	if len(targets.TemplateTypeIDs) == 0 {
		return []model.VideoTemplate{}, nil
	}
	db := dbFrom(ctx).Model(&model.VideoTemplate{}).
		Where("video_template.status = ?", 1).
		Where("video_template.video_template_type_id IN ?", targets.TemplateTypeIDs)
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
		Preload("VideoTemplateType.AppRules")
}

func (r *TemplateRepo) UpdateFields(ctx context.Context, item *model.VideoTemplate) error {
	return r.BaseRepo.Update(ctx, item,
		"VideoTemplateTypeID", "UserTypes", "SubscriptionStatuses", "Name", "TemplateType", "Sort",
		"CoverImage", "TemplateVideo", "ThumbnailVideo", "Prompt", "Status", "Description",
	)
}

func loadDisplayPositionsByKeys(db *gorm.DB, keys []string) ([]model.VideoDisplayPosition, error) {
	items := make([]model.VideoDisplayPosition, 0, len(keys))
	if len(keys) == 0 {
		return items, nil
	}
	if err := db.Where("position_key IN ?", keys).Find(&items).Error; err != nil {
		return nil, err
	}
	if len(items) != len(keys) {
		return nil, fmt.Errorf("one or more display positions do not exist")
	}
	return items, nil
}

func loadCountriesByIDs(db *gorm.DB, ids []uint64) ([]model.VideoCountry, error) {
	items := make([]model.VideoCountry, 0, len(ids))
	if len(ids) == 0 {
		return items, nil
	}
	if err := db.Where("id IN ?", ids).Find(&items).Error; err != nil {
		return nil, err
	}
	if len(items) != len(ids) {
		return nil, fmt.Errorf("one or more countries do not exist")
	}
	return items, nil
}

func (r *TemplateRepo) DeleteWithTargets(ctx context.Context, id uint64) error {
	db := dbFrom(ctx)
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Unscoped().Where("template_id = ?", id).Delete(&model.VideoTemplateDisplayConfig{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.VideoTemplate{ID: id}).Error
	})
}

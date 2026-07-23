package repository

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"ai-video/internal/gen/model"
	genquery "ai-video/internal/gen/query"

	"gorm.io/gen"
	"gorm.io/gen/field"
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
	q := qFrom(ctx)
	templateType := q.VideoTemplateType
	dao := templateType.WithContext(ctx)
	if filter != nil {
		if filter.Status != nil {
			dao = dao.Where(templateType.Status.Eq(*filter.Status))
		}
		if filter.PositionKey != "" {
			dao = dao.Where(templateSQLCondition(`EXISTS (
				SELECT 1 FROM video_template_type_display_position vttdp
				WHERE vttdp.template_type_id = video_template_type.id
					AND vttdp.position_key = ? AND vttdp.deleted_at IS NULL
			)`, filter.PositionKey)...)
		}
		if filter.CountryID != 0 {
			dao = dao.Where(templateSQLCondition(`EXISTS (
				SELECT 1 FROM video_template_type_country vttc
				JOIN video_country country_item ON country_item.code = vttc.country_code AND country_item.deleted_at IS NULL
				WHERE vttc.template_type_id = video_template_type.id
					AND country_item.id = ? AND vttc.deleted_at IS NULL
			)`, filter.CountryID)...)
		}
		if filter.AppCode != "" {
			dao = dao.Where(templateSQLCondition(`EXISTS (
				SELECT 1 FROM video_template_type_app vtta
				JOIN video_app app_item ON app_item.id = vtta.app_id AND app_item.deleted_at IS NULL
				WHERE vtta.template_type_id = video_template_type.id
					AND app_item.app_code = ? AND vtta.deleted_at IS NULL
			)`, filter.AppCode)...)
		}
		if filter.PackageCode != "" {
			dao = dao.Where(templateSQLCondition(`EXISTS (
				SELECT 1 FROM video_template_type_package vttp
				JOIN video_package package_item ON package_item.id = vttp.package_id AND package_item.deleted_at IS NULL
				WHERE vttp.template_type_id = video_template_type.id
					AND package_item.package_code = ? AND vttp.deleted_at IS NULL
			)`, filter.PackageCode)...)
		}
		if filter.VersionCode != "" {
			dao = dao.Where(templateSQLCondition(`EXISTS (
				SELECT 1 FROM video_template_type_version vttv
				JOIN video_package_version version_item ON version_item.id = vttv.version_id AND version_item.deleted_at IS NULL
				WHERE vttv.template_type_id = video_template_type.id
					AND version_item.version_code = ? AND vttv.deleted_at IS NULL
			)`, filter.VersionCode)...)
		}
		if filter.Keyword != "" {
			keyword := "%" + filter.Keyword + "%"
			dao = dao.Where(templateSQLCondition(`(
				video_template_type.category_name LIKE ? OR video_template_type.description LIKE ? OR EXISTS (
				SELECT 1 FROM video_template_type_display_position vttdp
				JOIN video_display_position vdp ON vdp.position_key = vttdp.position_key
				WHERE vttdp.template_type_id = video_template_type.id
					AND vttdp.deleted_at IS NULL
					AND (vdp.position_name LIKE ? OR vdp.position_key LIKE ?)
			) OR EXISTS (
				SELECT 1 FROM video_template_type_country vttc
				JOIN video_country country_item ON country_item.code = vttc.country_code
				WHERE vttc.template_type_id = video_template_type.id AND vttc.deleted_at IS NULL
					AND (country_item.code LIKE ? OR country_item.name_zh LIKE ?)
			) OR EXISTS (
				SELECT 1 FROM video_template_type_app vtta
				JOIN video_app app_item ON app_item.id = vtta.app_id
				WHERE vtta.template_type_id = video_template_type.id AND vtta.deleted_at IS NULL
					AND (app_item.app_code LIKE ? OR app_item.name LIKE ?)
			) OR EXISTS (
				SELECT 1 FROM video_template_type_package vttp
				JOIN video_package package_item ON package_item.id = vttp.package_id
				WHERE vttp.template_type_id = video_template_type.id AND vttp.deleted_at IS NULL
					AND (package_item.package_code LIKE ? OR package_item.package_name LIKE ?)
			) OR EXISTS (
				SELECT 1 FROM video_template_type_version vttv
				JOIN video_package_version version_item ON version_item.id = vttv.version_id
				WHERE vttv.template_type_id = video_template_type.id AND vttv.deleted_at IS NULL
					AND version_item.version_code LIKE ?
			))`, keyword, keyword, keyword, keyword, keyword, keyword, keyword, keyword, keyword, keyword, keyword)...)
		}
	}

	total, err := dao.Count()
	if err != nil {
		return nil, 0, err
	}
	rows, err := dao.Preload(
		templateType.DisplayPositions, templateType.Countries, templateType.Apps,
		templateType.Packages, templateType.Versions,
	).Order(templateType.Sort.Asc(), templateType.ID.Desc()).
		Offset((page - 1) * pageSize).Limit(pageSize).Find()
	if err != nil {
		return nil, 0, err
	}
	return templateTypeValues(rows), total, nil
}

func (r *TemplateTypeRepo) ListOptions(ctx context.Context) ([]model.VideoTemplateType, error) {
	q := qFrom(ctx).VideoTemplateType
	rows, err := q.WithContext(ctx).
		Preload(q.DisplayPositions, q.Countries, q.Apps, q.Packages, q.Versions).
		Order(q.Sort.Asc(), q.ID.Asc()).Find()
	if err != nil {
		return nil, err
	}
	return templateTypeValues(rows), nil
}

func (r *TemplateTypeRepo) GetDetail(ctx context.Context, id uint64) (*model.VideoTemplateType, error) {
	q := qFrom(ctx).VideoTemplateType
	return q.WithContext(ctx).
		Preload(q.DisplayPositions, q.Countries, q.Apps, q.Packages, q.Versions).
		Where(q.ID.Eq(id)).First()
}

func (r *TemplateTypeRepo) UpdateFields(ctx context.Context, item *model.VideoTemplateType) error {
	return r.BaseRepo.Update(ctx, item,
		"CategoryName", "Sort", "Status", "Description", "IsSubscribed", "UserTypes", "SubscriptionStatuses",
	)
}

func templateSQLCondition(sql string, args ...interface{}) []gen.Condition {
	return []gen.Condition{field.NewUnsafeFieldRaw(sql, args...)}
}

func templateTypeValues(rows []*model.VideoTemplateType) []model.VideoTemplateType {
	result := make([]model.VideoTemplateType, 0, len(rows))
	for _, row := range rows {
		if row != nil {
			result = append(result, *row)
		}
	}
	return result
}

func (r *TemplateTypeRepo) ReplaceDisplayPositions(ctx context.Context, item *model.VideoTemplateType, keys []string) error {
	db := dbFrom(ctx)
	positions, err := loadDisplayPositionsByKeys(ctx, keys)
	if err != nil {
		return err
	}
	return db.Model(item).Association("DisplayPositions").Replace(positions)
}

type TemplateTypeTargetIDs struct {
	DisplayPositionKeys []string
	CountryCodes        []string
	AppRules            []TemplateTypeAppRule
}

// TemplateTypeAppRule selects one application delivery target.
// An empty rule list means the template type is available to every app.
type TemplateTypeAppRule struct {
	AppCode string
}

func (r *TemplateTypeRepo) ReplaceTargets(ctx context.Context, item *model.VideoTemplateType, targets TemplateTypeTargetIDs) error {
	db := dbFrom(ctx)
	positions, err := loadDisplayPositionsByKeys(ctx, targets.DisplayPositionKeys)
	if err != nil {
		return err
	}
	countries, err := loadCountriesByIDs(ctx, targets.CountryCodes)
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
			AppCode: rule.AppCode,
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
		return tx.Select("DisplayPositions", "Countries", "Apps", "Packages", "Versions").Delete(&model.VideoTemplateType{ID: id}).Error
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
	q := qFrom(ctx).VideoTemplate
	return q.WithContext(ctx).Where(q.VideoTemplateTypeID.Eq(typeID)).Count()
}

type ClientTemplateTypeTargets struct {
	PositionKey string
	CountryCode string
	AppCode     string
	PackageCode string
	VersionCode string
}

func (r *TemplateTypeRepo) ListForClient(ctx context.Context, targets ClientTemplateTypeTargets) ([]model.VideoTemplateType, error) {
	q := qFrom(ctx)
	templateType := q.VideoTemplateType
	dao := templateType.WithContext(ctx).Where(templateType.Status.Eq(1))
	if targets.PositionKey == "" {
		dao = dao.Where(templateSQLCondition(`NOT EXISTS (
			SELECT 1 FROM video_template_type_display_position relation
			WHERE relation.template_type_id = video_template_type.id AND relation.deleted_at IS NULL
		)`)...)
	} else {
		dao = dao.Where(templateSQLCondition(`(
			NOT EXISTS (
				SELECT 1 FROM video_template_type_display_position relation
				WHERE relation.template_type_id = video_template_type.id AND relation.deleted_at IS NULL
			)
			OR EXISTS (
				SELECT 1 FROM video_template_type_display_position relation
				JOIN video_display_position position_item
					ON position_item.position_key = relation.position_key AND position_item.deleted_at IS NULL
				WHERE relation.template_type_id = video_template_type.id
					AND relation.position_key = ? AND relation.deleted_at IS NULL
					AND position_item.status = ?
			)
		)`, targets.PositionKey, 1)...)
	}
	if targets.CountryCode != "" {
		dao = dao.Where(templateSQLCondition(`(
			NOT EXISTS (
				SELECT 1 FROM video_template_type_country relation
				WHERE relation.template_type_id = video_template_type.id AND relation.deleted_at IS NULL
			)
			OR EXISTS (
				SELECT 1 FROM video_template_type_country relation
				JOIN video_country country_item ON country_item.code = relation.country_code AND country_item.deleted_at IS NULL
				WHERE relation.template_type_id = video_template_type.id
					AND relation.country_code = ? AND country_item.status = ? AND relation.deleted_at IS NULL
			)
		)`, targets.CountryCode, 1)...)
	} else {
		dao = dao.Where(templateSQLCondition(`NOT EXISTS (
			SELECT 1 FROM video_template_type_country relation
			WHERE relation.template_type_id = video_template_type.id AND relation.deleted_at IS NULL
		)`)...)
	}
	dao = applyTemplateTypeCodeTarget(dao, "video_template_type_app", "video_app", "app_id", "app_code", targets.AppCode)
	dao = applyTemplateTypeCodeTarget(dao, "video_template_type_package", "video_package", "package_id", "package_code", targets.PackageCode)
	dao = applyTemplateTypeCodeTarget(dao, "video_template_type_version", "video_package_version", "version_id", "version_code", targets.VersionCode)
	rows, err := dao.Preload(
		templateType.DisplayPositions, templateType.Countries, templateType.Apps,
		templateType.Packages, templateType.Versions,
	).Order(templateType.Sort.Desc(), templateType.ID.Desc()).Find()
	if err != nil {
		return nil, err
	}
	return templateTypeValues(rows), nil
}

func applyTemplateTypeCodeTarget(
	dao genquery.IVideoTemplateTypeDo,
	relationTable, targetTable, relationTargetColumn, targetCodeColumn, code string,
) genquery.IVideoTemplateTypeDo {
	if code == "" {
		return dao.Where(templateSQLCondition(fmt.Sprintf(`NOT EXISTS (
			SELECT 1 FROM %s relation
			WHERE relation.template_type_id = video_template_type.id AND relation.deleted_at IS NULL
		)`, relationTable))...)
	}
	return dao.Where(templateSQLCondition(fmt.Sprintf(`(
		NOT EXISTS (
			SELECT 1 FROM %s relation
			WHERE relation.template_type_id = video_template_type.id AND relation.deleted_at IS NULL
		)
		OR EXISTS (
			SELECT 1 FROM %s relation
			JOIN %s target_item ON target_item.id = relation.%s AND target_item.deleted_at IS NULL
			WHERE relation.template_type_id = video_template_type.id
				AND target_item.%s = ? AND target_item.status = 1 AND relation.deleted_at IS NULL
		)
	)`, relationTable, relationTable, targetTable, relationTargetColumn, targetCodeColumn), code)...)
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
	TemplateType        string
	Status              *int8
	Keyword             string
}

func (r *TemplateRepo) PageList(ctx context.Context, page, pageSize int, filter *TemplateListFilter) ([]model.VideoTemplate, int64, error) {
	q := qFrom(ctx).VideoTemplate
	dao := q.WithContext(ctx)
	if filter != nil {
		if filter.VideoTemplateTypeID != 0 {
			dao = dao.Where(q.VideoTemplateTypeID.Eq(filter.VideoTemplateTypeID))
		}
		if filter.PositionKey != "" {
			dao = dao.Where(templateSQLCondition(`EXISTS (
				SELECT 1 FROM video_template_placement_config placement_config
				WHERE placement_config.template_id = video_template.id
					AND placement_config.placement_key = ? AND placement_config.deleted_at IS NULL
			)`, filter.PositionKey)...)
		}
		if filter.TemplateType != "" {
			dao = dao.Where(q.TemplateType.Eq(filter.TemplateType))
		}
		if filter.Status != nil {
			dao = dao.Where(q.Status.Eq(*filter.Status))
		}
		if filter.Keyword != "" {
			keyword := "%" + filter.Keyword + "%"
			dao = dao.Where(field.Or(q.Name.Like(keyword), q.Prompt.Like(keyword), q.Description.Like(keyword)))
		}
	}

	total, err := dao.Count()
	if err != nil {
		return nil, 0, err
	}
	rows, err := dao.Preload(q.VideoTemplateType).
		Order(q.Sort.Asc(), q.ID.Desc()).
		Offset((page - 1) * pageSize).Limit(pageSize).Find()
	if err != nil {
		return nil, 0, err
	}
	return templateValues(rows), total, nil
}

func (r *TemplateRepo) GetTemplateID(ctx context.Context, id uint64) (*model.VideoTemplate, error) {
	q := qFrom(ctx).VideoTemplate
	return q.WithContext(ctx).Preload(q.VideoTemplateType).Where(q.ID.Eq(id)).First()
}

func (r *TemplateRepo) GetWithType(ctx context.Context, id uint64) (*model.VideoTemplate, error) {
	q := qFrom(ctx).VideoTemplate
	return q.WithContext(ctx).Preload(q.VideoTemplateType).Where(q.ID.Eq(id)).First()
}

func (r *TemplateRepo) ListOptions(ctx context.Context) ([]model.VideoTemplate, error) {
	q := qFrom(ctx).VideoTemplate
	rows, err := q.WithContext(ctx).Preload(q.VideoTemplateType).
		Order(q.Status.Desc(), q.Sort.Desc(), q.ID.Desc()).Find()
	if err != nil {
		return nil, err
	}
	return templateValues(rows), nil
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
	q := qFrom(ctx).VideoTemplate
	rows, err := q.WithContext(ctx).
		Where(q.Status.Eq(1), q.VideoTemplateTypeID.In(targets.TemplateTypeIDs...)).
		Preload(q.VideoTemplateType).
		Order(q.Sort.Desc(), q.UsageCount.Desc(), q.LikeCount.Desc(), q.ViewCount.Desc(), q.ID.Desc()).Find()
	if err != nil {
		return nil, err
	}
	return templateValues(rows), nil
}

func templateValues(rows []*model.VideoTemplate) []model.VideoTemplate {
	result := make([]model.VideoTemplate, 0, len(rows))
	for _, row := range rows {
		if row != nil {
			result = append(result, *row)
		}
	}
	return result
}

func (r *TemplateRepo) UpdateFields(ctx context.Context, item *model.VideoTemplate) error {
	return r.BaseRepo.Update(ctx, item,
		"VideoTemplateTypeID", "Name", "TemplateType", "Sort",
		"CoverImage", "TemplateVideo", "ThumbnailVideo", "Prompt", "Status", "Description",
	)
}

func loadDisplayPositionsByKeys(ctx context.Context, keys []string) ([]model.VideoDisplayPosition, error) {
	items := make([]model.VideoDisplayPosition, 0, len(keys))
	if len(keys) == 0 {
		return items, nil
	}
	q := qFrom(ctx).VideoDisplayPosition
	rows, err := q.WithContext(ctx).Where(q.PositionKey.In(keys...)).Find()
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		if row != nil {
			items = append(items, *row)
		}
	}
	if len(items) != len(keys) {
		return nil, fmt.Errorf("one or more display positions do not exist")
	}
	return items, nil
}

func loadCountriesByIDs(ctx context.Context, ids []string) ([]model.VideoCountry, error) {
	items := make([]model.VideoCountry, 0, len(ids))
	if len(ids) == 0 {
		return items, nil
	}
	q := qFrom(ctx).VideoCountry
	rows, err := q.WithContext(ctx).Where(q.Code.In(ids...)).Find()
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		if row != nil {
			items = append(items, *row)
		}
	}
	if len(items) != len(ids) {
		return nil, fmt.Errorf("one or more countries do not exist")
	}
	return items, nil
}

func (r *TemplateRepo) DeleteWithTargets(ctx context.Context, id uint64) error {
	db := dbFrom(ctx)
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Unscoped().Where("template_id = ?", id).Delete(&model.VideoTemplatePlacementConfig{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.VideoTemplate{ID: id}).Error
	})
}

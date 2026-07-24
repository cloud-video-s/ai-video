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

type TemplateTypeRecord struct {
	model.VideoTemplateType
	DisplayPositions []model.VideoDisplayPosition `json:"display_positions"`
	Countries        []model.VideoCountry         `json:"countries"`
	Apps             []model.VideoApp             `json:"apps"`
	Packages         []model.VideoPackage         `json:"packages"`
	Versions         []model.VideoPackageVersion  `json:"versions"`
}

func (r *TemplateTypeRepo) PageList(ctx context.Context, page, pageSize int, filter *TemplateTypeListFilter) ([]TemplateTypeRecord, int64, error) {
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
				WHERE vtta.template_type_id = video_template_type.id
					AND vtta.app_code = ? AND vtta.deleted_at IS NULL
			)`, filter.AppCode)...)
		}
		if filter.PackageCode != "" {
			dao = dao.Where(templateSQLCondition(`EXISTS (
				SELECT 1 FROM video_template_type_package vttp
				WHERE vttp.template_type_id = video_template_type.id
					AND vttp.package_code = ? AND vttp.deleted_at IS NULL
			)`, filter.PackageCode)...)
		}
		if filter.VersionCode != "" {
			dao = dao.Where(templateSQLCondition(`EXISTS (
				SELECT 1 FROM video_template_type_version vttv
				WHERE vttv.template_type_id = video_template_type.id
					AND vttv.version_code = ? AND vttv.deleted_at IS NULL
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
				JOIN video_app app_item ON app_item.app_code = vtta.app_code AND app_item.deleted_at IS NULL
				WHERE vtta.template_type_id = video_template_type.id AND vtta.deleted_at IS NULL
					AND (app_item.app_code LIKE ? OR app_item.name LIKE ?)
			) OR EXISTS (
				SELECT 1 FROM video_template_type_package vttp
				JOIN video_package package_item ON package_item.package_code = vttp.package_code AND package_item.deleted_at IS NULL
				WHERE vttp.template_type_id = video_template_type.id AND vttp.deleted_at IS NULL
					AND (package_item.package_code LIKE ? OR package_item.package_name LIKE ?)
			) OR EXISTS (
				SELECT 1 FROM video_template_type_version vttv
				JOIN video_package_version version_item ON version_item.version_code = vttv.version_code AND version_item.deleted_at IS NULL
				WHERE vttv.template_type_id = video_template_type.id AND vttv.deleted_at IS NULL
					AND version_item.version_code LIKE ?
			))`, keyword, keyword, keyword, keyword, keyword, keyword, keyword, keyword, keyword, keyword, keyword)...)
		}
	}

	total, err := dao.Count()
	if err != nil {
		return nil, 0, err
	}
	rows, err := dao.Order(templateType.Sort.Asc(), templateType.ID.Desc()).
		Offset((page - 1) * pageSize).Limit(pageSize).Find()
	if err != nil {
		return nil, 0, err
	}
	records, err := r.loadRecords(ctx, templateTypeValues(rows))
	return records, total, err
}

func (r *TemplateTypeRepo) ListOptions(ctx context.Context) ([]TemplateTypeRecord, error) {
	q := qFrom(ctx).VideoTemplateType
	rows, err := q.WithContext(ctx).Order(q.Sort.Asc(), q.ID.Asc()).Find()
	if err != nil {
		return nil, err
	}
	return r.loadRecords(ctx, templateTypeValues(rows))
}

func (r *TemplateTypeRepo) GetDetail(ctx context.Context, id uint64) (*TemplateTypeRecord, error) {
	q := qFrom(ctx).VideoTemplateType
	item, err := q.WithContext(ctx).Where(q.ID.Eq(id)).First()
	if err != nil {
		return nil, err
	}
	records, err := r.loadRecords(ctx, []model.VideoTemplateType{*item})
	if err != nil {
		return nil, err
	}
	return &records[0], nil
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

func (r *TemplateTypeRepo) loadRecords(ctx context.Context, items []model.VideoTemplateType) ([]TemplateTypeRecord, error) {
	result := make([]TemplateTypeRecord, 0, len(items))
	q := qFrom(ctx)
	for i := range items {
		record := TemplateTypeRecord{VideoTemplateType: items[i]}
		typeID := items[i].ID

		positionRelation := q.VideoTemplateTypeDisplayPosition
		var positionKeys []string
		if err := positionRelation.WithContext(ctx).Where(positionRelation.TemplateTypeID.Eq(typeID)).Pluck(positionRelation.PositionKey, &positionKeys); err != nil {
			return nil, err
		}
		if len(positionKeys) > 0 {
			rows, err := q.VideoDisplayPosition.WithContext(ctx).Where(q.VideoDisplayPosition.PositionKey.In(positionKeys...)).Find()
			if err != nil {
				return nil, err
			}
			record.DisplayPositions = valuesOf(rows)
		}

		countryRelation := q.VideoTemplateTypeCountry
		var countryCodes []string
		if err := countryRelation.WithContext(ctx).Where(countryRelation.TemplateTypeID.Eq(typeID)).Pluck(countryRelation.CountryCode, &countryCodes); err != nil {
			return nil, err
		}
		if len(countryCodes) > 0 {
			rows, err := q.VideoCountry.WithContext(ctx).Where(q.VideoCountry.Code.In(countryCodes...)).Find()
			if err != nil {
				return nil, err
			}
			record.Countries = valuesOf(rows)
		}

		appRelation := q.VideoTemplateTypeApp
		var appCodes []string
		if err := appRelation.WithContext(ctx).Where(appRelation.TemplateTypeID.Eq(typeID)).Pluck(appRelation.AppCode, &appCodes); err != nil {
			return nil, err
		}
		if len(appCodes) > 0 {
			rows, err := q.VideoApp.WithContext(ctx).Where(q.VideoApp.AppCode.In(appCodes...)).Find()
			if err != nil {
				return nil, err
			}
			record.Apps = valuesOf(rows)
		}

		packageRelation := q.VideoTemplateTypePackage
		var packageCodes []string
		if err := packageRelation.WithContext(ctx).Where(packageRelation.TemplateTypeID.Eq(typeID)).Pluck(packageRelation.PackageCode, &packageCodes); err != nil {
			return nil, err
		}
		if len(packageCodes) > 0 {
			rows, err := q.VideoPackage.WithContext(ctx).Where(q.VideoPackage.PackageCode.In(packageCodes...)).Find()
			if err != nil {
				return nil, err
			}
			record.Packages = valuesOf(rows)
		}

		versionRelation := q.VideoTemplateTypeVersion
		var versionCodes []string
		if err := versionRelation.WithContext(ctx).Where(versionRelation.TemplateTypeID.Eq(typeID)).Pluck(versionRelation.VersionCode, &versionCodes); err != nil {
			return nil, err
		}
		if len(versionCodes) > 0 {
			rows, err := q.VideoPackageVersion.WithContext(ctx).Where(q.VideoPackageVersion.VersionCode.In(versionCodes...)).Find()
			if err != nil {
				return nil, err
			}
			record.Versions = valuesOf(rows)
		}
		result = append(result, record)
	}
	return result, nil
}

func (r *TemplateTypeRepo) ReplaceDisplayPositions(ctx context.Context, item *model.VideoTemplateType, keys []string) error {
	if _, err := loadDisplayPositionsByKeys(ctx, keys); err != nil {
		return err
	}
	relation := qFrom(ctx).VideoTemplateTypeDisplayPosition
	if _, err := relation.WithContext(ctx).Unscoped().Where(relation.TemplateTypeID.Eq(item.ID)).Delete(); err != nil {
		return err
	}
	rows := make([]*model.VideoTemplateTypeDisplayPosition, 0, len(keys))
	for _, key := range keys {
		rows = append(rows, &model.VideoTemplateTypeDisplayPosition{TemplateTypeID: item.ID, PositionKey: key})
	}
	if len(rows) == 0 {
		return nil
	}
	return relation.WithContext(ctx).Create(rows...)
}

type TemplateTypeTargetIDs struct {
	DisplayPositionKeys []string
	CountryCodes        []string
	AppRules            []TemplateTypeAppRule
	PackageCodes        []string
	VersionCodes        []string
}

// TemplateTypeAppRule selects one application delivery target.
// An empty rule list means the template type is available to every app.
type TemplateTypeAppRule struct {
	AppCode string
}

func (r *TemplateTypeRepo) ReplaceTargets(ctx context.Context, item *model.VideoTemplateType, targets TemplateTypeTargetIDs) error {
	if _, err := loadDisplayPositionsByKeys(ctx, targets.DisplayPositionKeys); err != nil {
		return err
	}
	if _, err := loadCountriesByIDs(ctx, targets.CountryCodes); err != nil {
		return err
	}

	q := qFrom(ctx)
	positionRelation := q.VideoTemplateTypeDisplayPosition
	if _, err := positionRelation.WithContext(ctx).Unscoped().Where(positionRelation.TemplateTypeID.Eq(item.ID)).Delete(); err != nil {
		return err
	}
	positionRows := make([]*model.VideoTemplateTypeDisplayPosition, 0, len(targets.DisplayPositionKeys))
	for _, key := range targets.DisplayPositionKeys {
		positionRows = append(positionRows, &model.VideoTemplateTypeDisplayPosition{TemplateTypeID: item.ID, PositionKey: key})
	}
	if len(positionRows) > 0 {
		if err := positionRelation.WithContext(ctx).Create(positionRows...); err != nil {
			return err
		}
	}

	countryRelation := q.VideoTemplateTypeCountry
	if _, err := countryRelation.WithContext(ctx).Unscoped().Where(countryRelation.TemplateTypeID.Eq(item.ID)).Delete(); err != nil {
		return err
	}
	countryRows := make([]*model.VideoTemplateTypeCountry, 0, len(targets.CountryCodes))
	for _, code := range targets.CountryCodes {
		countryRows = append(countryRows, &model.VideoTemplateTypeCountry{TemplateTypeID: item.ID, CountryCode: code})
	}
	if len(countryRows) > 0 {
		if err := countryRelation.WithContext(ctx).Create(countryRows...); err != nil {
			return err
		}
	}

	appRelation := q.VideoTemplateTypeApp
	if _, err := appRelation.WithContext(ctx).Unscoped().Where(appRelation.TemplateTypeID.Eq(item.ID)).Delete(); err != nil {
		return err
	}
	appRows := make([]*model.VideoTemplateTypeApp, 0, len(targets.AppRules))
	for _, rule := range targets.AppRules {
		appRows = append(appRows, &model.VideoTemplateTypeApp{
			ID: nextTemplateTypeAppID(), TemplateTypeID: item.ID,
			AppCode: rule.AppCode,
		})
	}
	if len(appRows) > 0 {
		if err := appRelation.WithContext(ctx).Create(appRows...); err != nil {
			return err
		}
	}

	packageRelation := q.VideoTemplateTypePackage
	if _, err := packageRelation.WithContext(ctx).Unscoped().Where(packageRelation.TemplateTypeID.Eq(item.ID)).Delete(); err != nil {
		return err
	}
	packageRows := make([]*model.VideoTemplateTypePackage, 0, len(targets.PackageCodes))
	for _, code := range targets.PackageCodes {
		packageRows = append(packageRows, &model.VideoTemplateTypePackage{TemplateTypeID: item.ID, PackageCode: code})
	}
	if len(packageRows) > 0 {
		if err := packageRelation.WithContext(ctx).Create(packageRows...); err != nil {
			return err
		}
	}

	versionRelation := q.VideoTemplateTypeVersion
	if _, err := versionRelation.WithContext(ctx).Unscoped().Where(versionRelation.TemplateTypeID.Eq(item.ID)).Delete(); err != nil {
		return err
	}
	versionRows := make([]*model.VideoTemplateTypeVersion, 0, len(targets.VersionCodes))
	for _, code := range targets.VersionCodes {
		versionRows = append(versionRows, &model.VideoTemplateTypeVersion{TemplateTypeID: item.ID, VersionCode: code})
	}
	if len(versionRows) > 0 {
		if err := versionRelation.WithContext(ctx).Create(versionRows...); err != nil {
			return err
		}
	}
	return nil

}

func (r *TemplateTypeRepo) DeleteWithDisplayPositions(ctx context.Context, id uint64) error {
	return Transaction(ctx, func(txCtx context.Context) error {
		q := qFrom(txCtx)
		if _, err := q.VideoTemplateTypeDisplayPosition.WithContext(txCtx).Unscoped().Where(q.VideoTemplateTypeDisplayPosition.TemplateTypeID.Eq(id)).Delete(); err != nil {
			return err
		}
		if _, err := q.VideoTemplateTypeCountry.WithContext(txCtx).Unscoped().Where(q.VideoTemplateTypeCountry.TemplateTypeID.Eq(id)).Delete(); err != nil {
			return err
		}
		if _, err := q.VideoTemplateTypeApp.WithContext(txCtx).Unscoped().Where(q.VideoTemplateTypeApp.TemplateTypeID.Eq(id)).Delete(); err != nil {
			return err
		}
		if _, err := q.VideoTemplateTypePackage.WithContext(txCtx).Unscoped().Where(q.VideoTemplateTypePackage.TemplateTypeID.Eq(id)).Delete(); err != nil {
			return err
		}
		if _, err := q.VideoTemplateTypeVersion.WithContext(txCtx).Unscoped().Where(q.VideoTemplateTypeVersion.TemplateTypeID.Eq(id)).Delete(); err != nil {
			return err
		}
		_, err := q.VideoTemplateType.WithContext(txCtx).Where(q.VideoTemplateType.ID.Eq(id)).Delete()
		return err
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
	dao = applyTemplateTypeCodeTarget(dao, "video_template_type_app", "video_app", "app_code", "app_code", targets.AppCode)
	dao = applyTemplateTypeCodeTarget(dao, "video_template_type_package", "video_package", "package_code", "package_code", targets.PackageCode)
	dao = applyTemplateTypeCodeTarget(dao, "video_template_type_version", "video_package_version", "version_code", "version_code", targets.VersionCode)
	rows, err := dao.Order(templateType.Sort.Desc(), templateType.ID.Desc()).Find()
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
			JOIN %s target_item ON target_item.%s = relation.%s AND target_item.deleted_at IS NULL
			WHERE relation.template_type_id = video_template_type.id
				AND target_item.%s = ? AND target_item.status = 1 AND relation.deleted_at IS NULL
		)
	)`, relationTable, relationTable, targetTable, targetCodeColumn, relationTargetColumn, targetCodeColumn), code)...)
}

type TemplateRepo struct {
	BaseRepo[model.VideoTemplate]
}

type TemplateRecord struct {
	model.VideoTemplate
	VideoTemplateType *TemplateTypeRecord         `json:"video_template_type,omitempty"`
	Countries         []model.VideoCountry        `json:"countries"`
	Apps              []model.VideoApp            `json:"apps"`
	Packages          []model.VideoPackage        `json:"packages"`
	Versions          []model.VideoPackageVersion `json:"versions"`
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

func (r *TemplateRepo) PageList(ctx context.Context, page, pageSize int, filter *TemplateListFilter) ([]TemplateRecord, int64, error) {
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
	rows, err := dao.Order(q.Sort.Asc(), q.ID.Desc()).
		Offset((page - 1) * pageSize).Limit(pageSize).Find()
	if err != nil {
		return nil, 0, err
	}
	records, err := r.loadRecords(ctx, templateValues(rows))
	return records, total, err
}

func (r *TemplateRepo) GetTemplateID(ctx context.Context, id uint64) (*model.VideoTemplate, error) {
	q := qFrom(ctx).VideoTemplate
	return q.WithContext(ctx).Where(q.ID.Eq(id)).First()
}

func (r *TemplateRepo) GetWithType(ctx context.Context, id uint64) (*TemplateRecord, error) {
	q := qFrom(ctx).VideoTemplate
	item, err := q.WithContext(ctx).Where(q.ID.Eq(id)).First()
	if err != nil {
		return nil, err
	}
	records, err := r.loadRecords(ctx, []model.VideoTemplate{*item})
	if err != nil {
		return nil, err
	}
	return &records[0], nil
}

func (r *TemplateRepo) ListOptions(ctx context.Context) ([]TemplateRecord, error) {
	q := qFrom(ctx).VideoTemplate
	rows, err := q.WithContext(ctx).Order(q.Status.Desc(), q.Sort.Desc(), q.ID.Desc()).Find()
	if err != nil {
		return nil, err
	}
	return r.loadRecords(ctx, templateValues(rows))
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
		Order(q.Sort.Desc(), q.UsageCount.Desc(), q.LikeCount.Desc(), q.ViewCount.Desc(), q.ID.Desc()).Find()
	if err != nil {
		return nil, err
	}
	return templateValues(rows), nil
}

func (r *TemplateRepo) loadRecords(ctx context.Context, items []model.VideoTemplate) ([]TemplateRecord, error) {
	result := make([]TemplateRecord, 0, len(items))
	if len(items) == 0 {
		return result, nil
	}
	typeIDs := make([]uint64, 0, len(items))
	seen := make(map[uint64]struct{}, len(items))
	for i := range items {
		if _, ok := seen[items[i].VideoTemplateTypeID]; !ok {
			seen[items[i].VideoTemplateTypeID] = struct{}{}
			typeIDs = append(typeIDs, items[i].VideoTemplateTypeID)
		}
	}
	q := qFrom(ctx).VideoTemplateType
	typeRows, err := q.WithContext(ctx).Where(q.ID.In(typeIDs...)).Find()
	if err != nil {
		return nil, err
	}
	typeRecords, err := NewTemplateTypeRepo().loadRecords(ctx, templateTypeValues(typeRows))
	if err != nil {
		return nil, err
	}
	typeByID := make(map[uint64]*TemplateTypeRecord, len(typeRecords))
	for i := range typeRecords {
		typeByID[typeRecords[i].ID] = &typeRecords[i]
	}
	for i := range items {
		record := TemplateRecord{VideoTemplate: items[i]}
		if typeRecord := typeByID[items[i].VideoTemplateTypeID]; typeRecord != nil {
			record.VideoTemplateType = typeRecord
			record.Countries = append([]model.VideoCountry(nil), typeRecord.Countries...)
			record.Apps = append([]model.VideoApp(nil), typeRecord.Apps...)
			record.Packages = append([]model.VideoPackage(nil), typeRecord.Packages...)
			record.Versions = append([]model.VideoPackageVersion(nil), typeRecord.Versions...)
		}
		result = append(result, record)
	}
	return result, nil
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
	return Transaction(ctx, func(txCtx context.Context) error {
		q := qFrom(txCtx)
		config := q.VideoTemplatePlacementConfig
		if _, err := config.WithContext(txCtx).Unscoped().Where(config.TemplateID.Eq(id)).Delete(); err != nil {
			return err
		}
		template := q.VideoTemplate
		_, err := template.WithContext(txCtx).Where(template.ID.Eq(id)).Delete()
		return err
	})
}

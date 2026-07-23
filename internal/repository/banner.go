package repository

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"ai-video/internal/gen/model"

	"gorm.io/gen"
	"gorm.io/gen/field"
)

type BannerRepo struct {
	BaseRepo[model.VideoBanner]
}

func NewBannerRepo() *BannerRepo {
	return &BannerRepo{}
}

type BannerListFilter struct {
	CountryCode string
	AppCode     string
	PackageCode string
	VersionCode string
	PositionKey string
	JumpType    uint8
	Status      *int8
	Keyword     string
}

func (r *BannerRepo) PageList(ctx context.Context, page, pageSize int, filter *BannerListFilter) ([]model.VideoBanner, int64, error) {
	q := qFrom(ctx).VideoBanner
	dao := q.WithContext(ctx)
	if filter != nil {
		if filter.PositionKey != "" {
			dao = dao.Where(bannerSQLCondition(`EXISTS (
				SELECT 1 FROM video_banner_placement_association relation
				WHERE relation.banner_id = video_banner.id
					AND relation.placement_key = ? AND relation.deleted_at IS NULL
			)`, filter.PositionKey)...)
		}
		if filter.CountryCode != "" {
			dao = dao.Where(bannerSQLCondition(`EXISTS (
				SELECT 1 FROM video_banner_country vbc
				WHERE vbc.banner_id = video_banner.id AND vbc.country_code = ? AND vbc.deleted_at IS NULL
			)`, filter.CountryCode)...)
		}
		if filter.AppCode != "" {
			dao = dao.Where(bannerSQLCondition(`EXISTS (
				SELECT 1 FROM video_banner_app vba
				JOIN video_app app ON app.id = vba.app_id AND app.deleted_at IS NULL
				WHERE vba.banner_id = video_banner.id AND app.app_code = ? AND vba.deleted_at IS NULL
			)`, filter.AppCode)...)
		}
		if filter.PackageCode != "" {
			dao = dao.Where(bannerSQLCondition(`EXISTS (
				SELECT 1 FROM video_banner_package vbp
				JOIN video_package package_item ON package_item.id = vbp.package_id AND package_item.deleted_at IS NULL
				WHERE vbp.banner_id = video_banner.id AND package_item.package_code = ? AND vbp.deleted_at IS NULL
			)`, filter.PackageCode)...)
		}
		if filter.VersionCode != "" {
			dao = dao.Where(bannerSQLCondition(`(
				NOT EXISTS (SELECT 1 FROM video_banner_version vbv WHERE vbv.banner_id = video_banner.id AND vbv.deleted_at IS NULL)
				OR EXISTS (
					SELECT 1 FROM video_banner_version vbv
					JOIN video_package_version version_item ON version_item.id = vbv.version_id AND version_item.deleted_at IS NULL
					WHERE vbv.banner_id = video_banner.id AND version_item.version_code = ? AND vbv.deleted_at IS NULL
				)
			)`, filter.VersionCode)...)
		}
		if filter.JumpType != 0 {
			dao = dao.Where(q.JumpType.Eq(filter.JumpType))
		}
		if filter.Status != nil {
			dao = dao.Where(q.Status.Eq(*filter.Status))
		}
		if filter.Keyword != "" {
			keyword := "%" + filter.Keyword + "%"
			dao = dao.Where(bannerSQLCondition(
				"(video_banner.name LIKE ? OR video_banner.remark LIKE ?)", keyword, keyword,
			)...)
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
	return bannerValues(rows), total, nil
}

func (r *BannerRepo) GetDetail(ctx context.Context, id uint64) (*model.VideoBanner, error) {
	q := qFrom(ctx).VideoBanner
	return q.WithContext(ctx).Where(q.ID.Eq(id)).First()
}

func (r *BannerRepo) Create(ctx context.Context, item *model.VideoBanner) error {
	return qFrom(ctx).VideoBanner.WithContext(ctx).Create(item)
}

func (r *BannerRepo) UpdateFields(ctx context.Context, item *model.VideoBanner) error {
	q := qFrom(ctx).VideoBanner
	_, err := q.WithContext(ctx).Where(q.ID.Eq(item.ID)).
		Select(q.Name, q.CoverImage, q.Remark, q.Sort, q.JumpType, q.JumpURL, q.TemplateID, q.Status, q.SubscriptionStatus).
		Updates(item)
	return err
}

func bannerSQLCondition(sql string, args ...interface{}) []gen.Condition {
	return []gen.Condition{field.NewUnsafeFieldRaw(sql, args...)}
}

func bannerValues(rows []*model.VideoBanner) []model.VideoBanner {
	result := make([]model.VideoBanner, 0, len(rows))
	for _, row := range rows {
		if row != nil {
			result = append(result, *row)
		}
	}
	return result
}

type BannerAppTargetInput struct {
	AppCode      string
	PackageCode  string
	VersionCodes []string
}

type BannerAppTarget struct {
	AppCode      string   `json:"app_code"`
	AppName      string   `json:"app_name"`
	PackageCode  string   `json:"package_code"`
	PackageName  string   `json:"package_name"`
	VersionCodes []string `json:"version_codes"`
}

type BannerDeliveryVersion struct {
	VersionCode string `json:"version_code"`
}

type BannerDeliveryPackage struct {
	PackageCode string                  `json:"package_code"`
	PackageName string                  `json:"package_name"`
	Versions    []BannerDeliveryVersion `json:"versions"`
}

type BannerDeliveryApp struct {
	AppCode  string                  `json:"app_code"`
	AppName  string                  `json:"app_name"`
	Packages []BannerDeliveryPackage `json:"packages"`
}

type BannerTargetIDs struct {
	DisplayPositionKeys []string
	CountryIDs          []uint64
	AppTargets          []BannerAppTargetInput
}

type ClientBannerTargets struct {
	PositionKey        string
	CountryCode        string
	AppCode            string
	PackageCode        string
	VersionCode        string
	SubscriptionStatus uint8
}

func (r *BannerRepo) ListForClient(ctx context.Context, targets ClientBannerTargets) ([]model.VideoBanner, error) {
	q := qFrom(ctx).VideoBanner
	dao := q.WithContext(ctx).Where(q.Status.Eq(1))
	if targets.SubscriptionStatus != 0 {
		dao = dao.Where(q.SubscriptionStatus.In(3, targets.SubscriptionStatus))
	}
	if targets.PositionKey == "" {
		dao = dao.Where(bannerSQLCondition(`NOT EXISTS (
        SELECT 1 FROM video_banner_placement_association relation
        WHERE relation.banner_id = video_banner.id AND relation.deleted_at IS NULL
    )`)...)
	} else {
		dao = dao.Where(bannerSQLCondition(`(
        NOT EXISTS (
            SELECT 1 FROM video_banner_placement_association relation
            WHERE relation.banner_id = video_banner.id AND relation.deleted_at IS NULL
        )
        OR EXISTS (
            SELECT 1 FROM video_banner_placement_association relation
            WHERE relation.banner_id = video_banner.id
                AND relation.placement_key = ? AND relation.deleted_at IS NULL
        )
    )`, targets.PositionKey)...)
	}
	if targets.CountryCode == "" {
		dao = dao.Where(bannerSQLCondition(`NOT EXISTS (
			SELECT 1 FROM video_banner_country relation
			WHERE relation.banner_id = video_banner.id AND relation.deleted_at IS NULL
		)`)...)
	} else {
		dao = dao.Where(bannerSQLCondition(`(
			NOT EXISTS (
				SELECT 1 FROM video_banner_country relation
				WHERE relation.banner_id = video_banner.id AND relation.deleted_at IS NULL
			)
			OR EXISTS (
				SELECT 1 FROM video_banner_country relation
				WHERE relation.banner_id = video_banner.id
					AND relation.country_code = ? AND relation.deleted_at IS NULL
			)
		)`, targets.CountryCode)...)
	}
	if targets.AppCode == "" {
		dao = dao.Where(bannerSQLCondition(`NOT EXISTS (
			SELECT 1 FROM video_banner_app relation
			WHERE relation.banner_id = video_banner.id AND relation.deleted_at IS NULL
		)`)...)
	} else {
		dao = dao.Where(bannerSQLCondition(`(
			NOT EXISTS (
				SELECT 1 FROM video_banner_app relation
				WHERE relation.banner_id = video_banner.id AND relation.deleted_at IS NULL
			)
			OR EXISTS (
				SELECT 1 FROM video_banner_app relation
				JOIN video_app app ON app.id = relation.app_id AND app.deleted_at IS NULL
				WHERE relation.banner_id = video_banner.id
					AND app.app_code = ? AND relation.deleted_at IS NULL
			)
		)`, targets.AppCode)...)
	}
	if targets.PackageCode == "" {
		dao = dao.Where(bannerSQLCondition(`NOT EXISTS (
			SELECT 1 FROM video_banner_package relation
			WHERE relation.banner_id = video_banner.id AND relation.deleted_at IS NULL
		)`)...)
	} else {
		dao = dao.Where(bannerSQLCondition(`(
			NOT EXISTS (
				SELECT 1 FROM video_banner_package relation
				WHERE relation.banner_id = video_banner.id AND relation.deleted_at IS NULL
			)
			OR EXISTS (
				SELECT 1 FROM video_banner_package relation
				JOIN video_package package_item ON package_item.id = relation.package_id AND package_item.deleted_at IS NULL
				WHERE relation.banner_id = video_banner.id
					AND package_item.package_code = ? AND relation.deleted_at IS NULL
			)
		)`, targets.PackageCode)...)
	}
	if targets.VersionCode == "" {
		dao = dao.Where(bannerSQLCondition(`NOT EXISTS (
			SELECT 1 FROM video_banner_version relation
			WHERE relation.banner_id = video_banner.id AND relation.deleted_at IS NULL
		)`)...)
	} else {
		dao = dao.Where(bannerSQLCondition(`(
			NOT EXISTS (
				SELECT 1 FROM video_banner_version relation
				WHERE relation.banner_id = video_banner.id AND relation.deleted_at IS NULL
			)
			OR EXISTS (
				SELECT 1 FROM video_banner_version relation
				JOIN video_package_version version_item ON version_item.id = relation.version_id AND version_item.deleted_at IS NULL
				WHERE relation.banner_id = video_banner.id
					AND version_item.version_code = ? AND relation.deleted_at IS NULL
			)
		)`, targets.VersionCode)...)
	}
	rows, err := dao.Preload(q.Template).Order(q.Sort.Asc(), q.ID.Desc()).Find()
	if err != nil {
		return nil, err
	}
	return bannerValues(rows), nil
}

func (r *BannerRepo) ReplaceTargets(ctx context.Context, item *model.VideoBanner, targets BannerTargetIDs) error {
	if err := deleteBannerTargetRows(ctx, item.ID); err != nil {
		return err
	}
	q := qFrom(ctx)
	placementRows := make([]*model.VideoBannerPlacementAssociation, 0, len(targets.DisplayPositionKeys))
	for _, key := range sortedUniqueStrings(targets.DisplayPositionKeys) {
		placementRows = append(placementRows, &model.VideoBannerPlacementAssociation{BannerID: item.ID, PlacementKey: key})
	}
	if len(placementRows) > 0 {
		if err := q.VideoBannerPlacementAssociation.WithContext(ctx).Create(placementRows...); err != nil {
			return err
		}
	}
	countryIDs := uniqueUint64s(targets.CountryIDs)
	if len(countryIDs) > 0 {
		countryDAO := q.VideoCountry
		countries, err := countryDAO.WithContext(ctx).Where(countryDAO.ID.In(countryIDs...)).Find()
		if err != nil {
			return err
		}
		if len(countries) != len(countryIDs) {
			return fmt.Errorf("one or more countries do not exist")
		}
		rows := make([]*model.VideoBannerCountry, 0, len(countries))
		for _, country := range countries {
			rows = append(rows, &model.VideoBannerCountry{BannerID: item.ID, CountryCode: country.Code})
		}
		if err := q.VideoBannerCountry.WithContext(ctx).Create(rows...); err != nil {
			return err
		}
	}
	return createBannerAppTargets(ctx, item.ID, targets.AppTargets)
}

func (r *BannerRepo) DeleteWithTargets(ctx context.Context, id uint64) error {
	return Transaction(ctx, func(ctx context.Context) error {
		if err := deleteBannerTargetRows(ctx, id); err != nil {
			return err
		}
		q := qFrom(ctx).VideoBanner
		_, err := q.WithContext(ctx).Where(q.ID.Eq(id)).Delete()
		return err
	})
}

func createBannerAppTargets(ctx context.Context, bannerID uint64, targets []BannerAppTargetInput) error {
	q := qFrom(ctx)
	appRows := make([]*model.VideoBannerApp, 0, len(targets))
	packageRows := make([]*model.VideoBannerPackage, 0, len(targets))
	versionRows := make([]*model.VideoBannerVersion, 0)
	seenApps := make(map[uint64]struct{}, len(targets))
	seenPackages := make(map[uint64]struct{}, len(targets))
	seenVersions := make(map[uint64]struct{})
	for _, target := range targets {
		appDAO := q.VideoApp
		app, err := appDAO.WithContext(ctx).
			Where(appDAO.AppCode.Eq(strings.TrimSpace(target.AppCode)), appDAO.Status.Eq(1)).First()
		if err != nil {
			return err
		}
		if _, exists := seenApps[app.ID]; !exists {
			seenApps[app.ID] = struct{}{}
			appRows = append(appRows, &model.VideoBannerApp{BannerID: bannerID, AppID: app.ID})
		}
		packageDAO := q.VideoPackage
		packageItem, err := packageDAO.WithContext(ctx).Where(
			packageDAO.PackageCode.Eq(strings.TrimSpace(target.PackageCode)),
			packageDAO.AppCode.Eq(app.AppCode), packageDAO.Status.Eq(1),
		).First()
		if err != nil {
			return err
		}
		if _, exists := seenPackages[packageItem.ID]; !exists {
			seenPackages[packageItem.ID] = struct{}{}
			packageRows = append(packageRows, &model.VideoBannerPackage{BannerID: bannerID, PackageID: packageItem.ID})
		}
		versionCodes := sortedUniqueStrings(target.VersionCodes)
		if len(versionCodes) == 0 {
			continue
		}
		versionDAO := q.VideoPackageVersion
		versions, err := versionDAO.WithContext(ctx).Where(
			versionDAO.PackageCode.Eq(packageItem.PackageCode),
			versionDAO.VersionCode.In(versionCodes...), versionDAO.Status.Eq(1),
		).Find()
		if err != nil {
			return err
		}
		if len(versions) != len(versionCodes) {
			return fmt.Errorf("one or more package versions do not exist")
		}
		for _, version := range versions {
			if _, exists := seenVersions[version.ID]; exists {
				continue
			}
			seenVersions[version.ID] = struct{}{}
			versionRows = append(versionRows, &model.VideoBannerVersion{BannerID: bannerID, VersionID: version.ID})
		}
	}
	if len(appRows) > 0 {
		if err := q.VideoBannerApp.WithContext(ctx).Create(appRows...); err != nil {
			return err
		}
	}
	if len(packageRows) > 0 {
		if err := q.VideoBannerPackage.WithContext(ctx).Create(packageRows...); err != nil {
			return err
		}
	}
	if len(versionRows) > 0 {
		if err := q.VideoBannerVersion.WithContext(ctx).Create(versionRows...); err != nil {
			return err
		}
	}
	return nil
}

func deleteBannerTargetRows(ctx context.Context, bannerID uint64) error {
	q := qFrom(ctx)
	placementDAO := q.VideoBannerPlacementAssociation
	if _, err := placementDAO.WithContext(ctx).Where(placementDAO.BannerID.Eq(bannerID)).Delete(); err != nil {
		return err
	}
	countryDAO := q.VideoBannerCountry
	if _, err := countryDAO.WithContext(ctx).Where(countryDAO.BannerID.Eq(bannerID)).Delete(); err != nil {
		return err
	}
	appDAO := q.VideoBannerApp
	if _, err := appDAO.WithContext(ctx).Where(appDAO.BannerID.Eq(bannerID)).Delete(); err != nil {
		return err
	}
	packageDAO := q.VideoBannerPackage
	if _, err := packageDAO.WithContext(ctx).Where(packageDAO.BannerID.Eq(bannerID)).Delete(); err != nil {
		return err
	}
	versionDAO := q.VideoBannerVersion
	_, err := versionDAO.WithContext(ctx).Where(versionDAO.BannerID.Eq(bannerID)).Delete()
	return err
}

func (r *BannerRepo) LoadAppTargets(ctx context.Context, bannerIDs []uint64) (map[uint64][]BannerAppTarget, error) {
	result := make(map[uint64][]BannerAppTarget, len(bannerIDs))
	bannerIDs = uniqueUint64s(bannerIDs)
	if len(bannerIDs) == 0 {
		return result, nil
	}
	q := qFrom(ctx)
	appRelationDAO := q.VideoBannerApp
	appRelations, err := appRelationDAO.WithContext(ctx).
		Where(appRelationDAO.BannerID.In(bannerIDs...)).
		Order(appRelationDAO.BannerID.Asc(), appRelationDAO.AppID.Asc()).Find()
	if err != nil {
		return nil, err
	}
	packageRelationDAO := q.VideoBannerPackage
	packageRelations, err := packageRelationDAO.WithContext(ctx).
		Where(packageRelationDAO.BannerID.In(bannerIDs...)).
		Order(packageRelationDAO.BannerID.Asc(), packageRelationDAO.PackageID.Asc()).Find()
	if err != nil {
		return nil, err
	}
	versionRelationDAO := q.VideoBannerVersion
	versionRelations, err := versionRelationDAO.WithContext(ctx).
		Where(versionRelationDAO.BannerID.In(bannerIDs...)).
		Order(versionRelationDAO.BannerID.Asc(), versionRelationDAO.VersionID.Asc()).Find()
	if err != nil {
		return nil, err
	}
	appIDs := make([]uint64, 0, len(appRelations))
	for _, relation := range appRelations {
		appIDs = append(appIDs, relation.AppID)
	}
	packageIDs := make([]uint64, 0, len(packageRelations))
	for _, relation := range packageRelations {
		packageIDs = append(packageIDs, relation.PackageID)
	}
	versionIDs := make([]uint64, 0, len(versionRelations))
	for _, relation := range versionRelations {
		if relation.VersionID > 0 {
			versionIDs = append(versionIDs, uint64(relation.VersionID))
		}
	}
	appsByID := make(map[uint64]*model.VideoApp)
	if appIDs = uniqueUint64s(appIDs); len(appIDs) > 0 {
		appDAO := q.VideoApp
		apps, err := appDAO.WithContext(ctx).Where(appDAO.ID.In(appIDs...)).Find()
		if err != nil {
			return nil, err
		}
		for _, app := range apps {
			appsByID[app.ID] = app
		}
	}
	packagesByID := make(map[uint64]*model.VideoPackage)
	if packageIDs = uniqueUint64s(packageIDs); len(packageIDs) > 0 {
		packageDAO := q.VideoPackage
		packages, err := packageDAO.WithContext(ctx).Where(packageDAO.ID.In(packageIDs...)).Find()
		if err != nil {
			return nil, err
		}
		for _, packageItem := range packages {
			packagesByID[packageItem.ID] = packageItem
		}
	}
	versionsByID := make(map[uint64]*model.VideoPackageVersion)
	if versionIDs = uniqueUint64s(versionIDs); len(versionIDs) > 0 {
		versionDAO := q.VideoPackageVersion
		versions, err := versionDAO.WithContext(ctx).Where(versionDAO.ID.In(versionIDs...)).Find()
		if err != nil {
			return nil, err
		}
		for _, version := range versions {
			versionsByID[version.ID] = version
		}
	}
	appsByBanner := make(map[uint64][]uint64)
	for _, relation := range appRelations {
		appsByBanner[relation.BannerID] = append(appsByBanner[relation.BannerID], relation.AppID)
	}
	packagesByBanner := make(map[uint64][]uint64)
	for _, relation := range packageRelations {
		packagesByBanner[relation.BannerID] = append(packagesByBanner[relation.BannerID], relation.PackageID)
	}
	versionsByBanner := make(map[uint64][]uint64)
	for _, relation := range versionRelations {
		if relation.VersionID > 0 {
			versionsByBanner[relation.BannerID] = append(versionsByBanner[relation.BannerID], uint64(relation.VersionID))
		}
	}
	for _, bannerID := range bannerIDs {
		for _, appID := range uniqueUint64s(appsByBanner[bannerID]) {
			app := appsByID[appID]
			if app == nil {
				continue
			}
			for _, packageID := range uniqueUint64s(packagesByBanner[bannerID]) {
				packageItem := packagesByID[packageID]
				if packageItem == nil || packageItem.AppCode != app.AppCode {
					continue
				}
				versionCodes := make([]string, 0)
				for _, versionID := range uniqueUint64s(versionsByBanner[bannerID]) {
					version := versionsByID[versionID]
					if version != nil && version.PackageCode == packageItem.PackageCode {
						versionCodes = append(versionCodes, version.VersionCode)
					}
				}
				result[bannerID] = append(result[bannerID], BannerAppTarget{
					AppCode: app.AppCode, AppName: app.Name,
					PackageCode: packageItem.PackageCode, PackageName: packageItem.PackageName,
					VersionCodes: sortedUniqueStrings(versionCodes),
				})
			}
		}
	}
	return result, nil
}

func (r *BannerRepo) ListDeliveryOptions(ctx context.Context) ([]BannerDeliveryApp, error) {
	q := qFrom(ctx)
	appDAO := q.VideoApp
	apps, err := appDAO.WithContext(ctx).Where(appDAO.Status.Eq(1)).
		Order(appDAO.Sort.Asc(), appDAO.ID.Asc()).Find()
	if err != nil {
		return nil, err
	}
	packageDAO := q.VideoPackage
	packages, err := packageDAO.WithContext(ctx).Where(packageDAO.Status.Eq(1)).
		Order(packageDAO.Sort.Asc(), packageDAO.ID.Asc()).Find()
	if err != nil {
		return nil, err
	}
	versionDAO := q.VideoPackageVersion
	versions, err := versionDAO.WithContext(ctx).Where(versionDAO.Status.Eq(1)).
		Order(versionDAO.VersionCode.Asc(), versionDAO.ID.Asc()).Find()
	if err != nil {
		return nil, err
	}
	result := make([]BannerDeliveryApp, 0, len(apps))
	for _, app := range apps {
		appOption := BannerDeliveryApp{AppCode: app.AppCode, AppName: app.Name, Packages: []BannerDeliveryPackage{}}
		for _, packageItem := range packages {
			if packageItem.AppCode != "" && packageItem.AppCode != app.AppCode {
				continue
			}
			packageOption := BannerDeliveryPackage{
				PackageCode: packageItem.PackageCode, PackageName: packageItem.PackageName,
				Versions: []BannerDeliveryVersion{},
			}
			for _, version := range versions {
				if version.PackageCode == "" || version.PackageCode == packageItem.PackageCode {
					packageOption.Versions = append(packageOption.Versions, BannerDeliveryVersion{VersionCode: version.VersionCode})
				}
			}
			appOption.Packages = append(appOption.Packages, packageOption)
		}
		result = append(result, appOption)
	}
	return result, nil
}

func (r *BannerRepo) ValidateAppTarget(ctx context.Context, target BannerAppTargetInput) error {
	target.AppCode = strings.TrimSpace(target.AppCode)
	target.PackageCode = strings.TrimSpace(target.PackageCode)
	q := qFrom(ctx)
	appDAO := q.VideoApp
	appCount, err := appDAO.WithContext(ctx).
		Where(appDAO.AppCode.Eq(target.AppCode), appDAO.Status.Eq(1)).Count()
	if err != nil {
		return err
	}
	if appCount == 0 {
		return fmt.Errorf("应用 %s 不存在或已禁用", target.AppCode)
	}
	packageDAO := q.VideoPackage
	packageCount, err := packageDAO.WithContext(ctx).Where(
		packageDAO.PackageCode.Eq(target.PackageCode), packageDAO.Status.Eq(1), packageDAO.AppCode.Eq(target.AppCode),
	).Count()
	if err != nil {
		return err
	}
	if packageCount == 0 {
		return fmt.Errorf("包 %s 不属于所选应用或已禁用", target.PackageCode)
	}
	target.VersionCodes = sortedUniqueStrings(target.VersionCodes)
	if len(target.VersionCodes) == 0 {
		return nil
	}
	versionDAO := q.VideoPackageVersion
	count, err := versionDAO.WithContext(ctx).Where(
		versionDAO.VersionCode.In(target.VersionCodes...), versionDAO.Status.Eq(1),
		versionDAO.PackageCode.Eq(target.PackageCode),
	).Distinct(versionDAO.VersionCode).Count()
	if err != nil {
		return err
	}
	if count != int64(len(target.VersionCodes)) {
		return fmt.Errorf("包 %s 包含不存在、已禁用或不匹配的版本", target.PackageCode)
	}
	return nil
}

func sortedUniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func uniqueUint64s(values []uint64) []uint64 {
	seen := make(map[uint64]struct{}, len(values))
	result := make([]uint64, 0, len(values))
	for _, value := range values {
		if value == 0 {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}

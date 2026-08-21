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

type PointsRepo struct {
	BaseRepo[model.VideoPoint]
}

func NewPointsRepo() *PointsRepo { return &PointsRepo{} }

type PointsListFilter struct {
	ListSort     ListSort
	AppCode      string
	PackageCode  string
	VersionCode  string
	CountryCode  string
	ChannelCode  string
	System       string
	UserType     int
	ResourceType string
	Status       *int8
	Keyword      string
}

// ClientPointsTargets contains the authenticated client dimensions used to
// select points products that are available for the current request.
type ClientPointsTargets struct {
	ProductID   uint64
	AppCode     string
	PackageCode string
	VersionCode string
	CountryCode string
	ChannelCode string
	System      string
	UserType    int
}

// ListForClient returns enabled points products matching every client
// dimension. Empty optional target relations mean "all", while a product must
// always be assigned to the caller's package before it can be purchased.
func (r *PointsRepo) ListForClient(ctx context.Context, targets ClientPointsTargets) ([]*model.VideoPoint, error) {
	points := qFrom(ctx).VideoPoint
	dao := points.WithContext(ctx).Where(points.Status.Eq(1))
	if targets.ProductID != 0 {
		dao = dao.Where(points.ID.Eq(targets.ProductID))
	}
	if targets.System != "" {
		dao = dao.Where(points.Systems.Like("%" + fmt.Sprint(targets.System) + "%"))
	}
	if targets.UserType > 0 {
		dao = dao.Where(points.UserTypes.Like("%" + fmt.Sprint(targets.UserType) + "%"))
	}
	dao = applyClientPointsTarget(dao, "video_points_app", "app_code", targets.AppCode, true)
	dao = applyClientPointsTarget(dao, "video_points_package", "package_code", targets.PackageCode, false)
	dao = applyClientPointsTarget(dao, "video_points_version", "version_code", targets.VersionCode, true)
	dao = applyClientPointsTarget(dao, "video_points_country", "country_code", targets.CountryCode, true)
	dao = applyClientPointsTarget(dao, "video_points_channel", "channel_code", targets.ChannelCode, true)

	rows, err := dao.Order(points.IsDefault.Desc(), points.Sort.Asc(), points.ID.Desc()).Find()
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func applyClientPointsTarget(
	dao genquery.IVideoPointDo,
	relationTable, codeColumn, code string,
	emptyIsWildcard bool,
) genquery.IVideoPointDo {
	withoutTargets := fmt.Sprintf(`NOT EXISTS (
		SELECT 1 FROM %s relation
		WHERE relation.points_id = video_points.id AND relation.deleted_at IS NULL
	)`, relationTable)
	if code == "" {
		if !emptyIsWildcard {
			return dao.Where(pointsSQLCondition("1 = 0")...)
		}
		return dao.Where(pointsSQLCondition(withoutTargets)...)
	}

	matchingTarget := fmt.Sprintf(`EXISTS (
		SELECT 1 FROM %s relation
		WHERE relation.points_id = video_points.id
			AND relation.%s = ? AND relation.deleted_at IS NULL
	)`, relationTable, codeColumn)
	if !emptyIsWildcard {
		return dao.Where(pointsSQLCondition(matchingTarget, code)...)
	}
	return dao.Where(pointsSQLCondition("("+withoutTargets+" OR "+matchingTarget+")", code)...)
}

func pointsSQLCondition(sql string, args ...any) []gen.Condition {
	return []gen.Condition{field.NewUnsafeFieldRaw(sql, args...)}
}

func (r *PointsRepo) PageList(ctx context.Context, page, pageSize int, filter *PointsListFilter) ([]model.VideoPoint, int64, error) {
	q := qFrom(ctx)
	points := q.VideoPoint
	dao := points.WithContext(ctx)
	if filter != nil {
		idSets, err := r.targetFilterIDs(ctx, filter)
		if err != nil {
			return nil, 0, err
		}
		for _, ids := range idSets {
			if len(ids) == 0 {
				return []model.VideoPoint{}, 0, nil
			}
			dao = dao.Where(points.ID.In(ids...))
		}
		if filter.System != "" {
			dao = dao.Where(points.Systems.Like("%\"" + filter.System + "\"%"))
		}
		if filter.UserType != 0 {
			dao = dao.Where(points.UserTypes.Like("%" + fmt.Sprint(filter.UserType) + "%"))
		}
		if filter.ResourceType != "" {
			dao = dao.Where(points.ResourceType.Eq(filter.ResourceType))
		}
		if filter.Status != nil {
			dao = dao.Where(points.Status.Eq(*filter.Status))
		}
		if filter.Keyword != "" {
			keyword := "%" + filter.Keyword + "%"
			dao = dao.Where(field.Or(
				points.ProductCode.Like(keyword), points.Name.Like(keyword),
				points.Description.Like(keyword),
			))
		}
	}
	total, err := dao.Count()
	if err != nil {
		return nil, 0, err
	}
	listSort := ListSort{}
	if filter != nil {
		listSort = filter.ListSort
	}
	order := orderForList(listSort, map[string]field.OrderExpr{"id": points.ID, "sort": points.Sort}, points.ID, points.Sort.Asc(), points.ID.Desc())
	rows, err := dao.Order(order...).Offset((page - 1) * pageSize).Limit(pageSize).Find()
	if err != nil {
		return nil, 0, err
	}
	items := valuesOf(rows)
	if err := r.loadAssociations(ctx, items); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *PointsRepo) targetFilterIDs(ctx context.Context, filter *PointsListFilter) ([][]uint64, error) {
	q := qFrom(ctx)
	sets := make([][]uint64, 0, 5)
	appendUint64 := func(enabled bool, pluck func(*[]uint64) error) error {
		if !enabled {
			return nil
		}
		var ids []uint64
		if err := pluck(&ids); err != nil {
			return err
		}
		sets = append(sets, ids)
		return nil
	}
	appendInt64 := func(enabled bool, pluck func(*[]int64) error) error {
		if !enabled {
			return nil
		}
		var raw []int64
		if err := pluck(&raw); err != nil {
			return err
		}
		ids := make([]uint64, 0, len(raw))
		for _, id := range raw {
			if id > 0 {
				ids = append(ids, uint64(id))
			}
		}
		sets = append(sets, ids)
		return nil
	}
	if err := appendUint64(filter.AppCode != "", func(ids *[]uint64) error {
		relation := q.VideoPointsApp
		return relation.WithContext(ctx).Where(relation.AppCode.Eq(filter.AppCode)).Pluck(relation.PointsID, ids)
	}); err != nil {
		return nil, err
	}
	if err := appendInt64(filter.PackageCode != "", func(ids *[]int64) error {
		relation := q.VideoPointsPackage
		return relation.WithContext(ctx).Where(relation.PackageCode.Eq(filter.PackageCode)).Pluck(relation.PointsID, ids)
	}); err != nil {
		return nil, err
	}
	if err := appendUint64(filter.VersionCode != "", func(ids *[]uint64) error {
		relation := q.VideoPointsVersion
		return relation.WithContext(ctx).Where(relation.VersionCode.Eq(filter.VersionCode)).Pluck(relation.PointsID, ids)
	}); err != nil {
		return nil, err
	}
	if err := appendInt64(filter.CountryCode != "", func(ids *[]int64) error {
		relation := q.VideoPointsCountry
		return relation.WithContext(ctx).Where(relation.CountryCode.Eq(filter.CountryCode)).Pluck(relation.PointsID, ids)
	}); err != nil {
		return nil, err
	}
	if err := appendUint64(filter.ChannelCode != "", func(ids *[]uint64) error {
		relation := q.VideoPointsChannel
		return relation.WithContext(ctx).Where(relation.ChannelCode.Eq(filter.ChannelCode)).Pluck(relation.PointsID, ids)
	}); err != nil {
		return nil, err
	}
	return sets, nil
}

func (r *PointsRepo) GetDetail(ctx context.Context, id uint64) (*model.VideoPoint, error) {
	q := qFrom(ctx).VideoPoint
	item, err := q.WithContext(ctx).Where(q.ID.Eq(id)).First()
	if err != nil {
		return nil, err
	}
	items := []model.VideoPoint{*item}
	if err := r.loadAssociations(ctx, items); err != nil {
		return nil, err
	}
	return &items[0], nil
}

func (r *PointsRepo) GetByProductID(ctx context.Context, productID string) (*model.VideoPoint, error) {
	q := qFrom(ctx).VideoPoint
	return q.WithContext(ctx).Where(q.ProductCode.Eq(productID)).First()
}

// GetAppleProduct resolves an enabled points product by store SKU and caller package.
func (r *PointsRepo) GetAppleProduct(ctx context.Context, productID, packageCode string) (*model.VideoPoint, error) {
	q := qFrom(ctx)
	points := q.VideoPoint
	item, err := points.WithContext(ctx).Where(points.ProductCode.Eq(productID), points.Status.Eq(1)).First()
	if err != nil {
		return nil, err
	}
	relation := q.VideoPointsPackage
	if _, err := relation.WithContext(ctx).Where(
		relation.PointsID.Eq(int64(item.ID)), relation.PackageCode.Eq(packageCode),
	).First(); err != nil {
		return nil, err
	}
	return item, nil
}

// GetEnabledForPackage loads an enabled points product only when it is
// assigned to the authenticated application package.
func (r *PointsRepo) GetEnabledForPackage(ctx context.Context, id uint64, packageCode string) (*model.VideoPoint, error) {
	q := qFrom(ctx)
	relation := q.VideoPointsPackage
	if _, err := relation.WithContext(ctx).Where(
		relation.PointsID.Eq(int64(id)), relation.PackageCode.Eq(packageCode),
	).First(); err != nil {
		return nil, err
	}
	points := q.VideoPoint
	return points.WithContext(ctx).Where(points.ID.Eq(id), points.Status.Eq(1)).First()
}

func (r *PointsRepo) ListOptions(ctx context.Context) ([]model.VideoPoint, error) {
	q := qFrom(ctx).VideoPoint
	rows, err := q.WithContext(ctx).Where(q.Status.Eq(1)).Order(q.Sort.Asc(), q.ID.Asc()).Find()
	return valuesOf(rows), err
}

func (r *PointsRepo) UpdateFields(ctx context.Context, item *model.VideoPoint) error {
	q := qFrom(ctx).VideoPoint
	_, err := q.WithContext(ctx).Where(q.ID.Eq(item.ID)).Select(
		q.ProductCode, q.Name, q.Systems, q.UserTypes, q.ResourceType, q.Points,
		q.Currency, q.SalePrice, q.ActualRevenue, q.OriginalPrice, q.Icon,
		q.Description, q.ButtonText, q.IsDefault, q.Status, q.Sort,
	).Updates(item)
	return err
}

type PointsTargets struct {
	AppCodes     []string
	PackageCodes []string
	VersionCodes []string
	CountryCodes []string
	ChannelCodes []string
}

func (r *PointsRepo) ReplaceTargets(ctx context.Context, item *model.VideoPoint, targets PointsTargets) error {
	q := qFrom(ctx)
	if err := validatePointsTargetCodes(targets.AppCodes, func(codes []string) (int, error) {
		rows, err := q.VideoApp.WithContext(ctx).Select(q.VideoApp.AppCode).Where(q.VideoApp.AppCode.In(codes...)).Find()
		return uniquePointsTargetCount(rows, func(item *model.VideoApp) string { return item.AppCode }), err
	}); err != nil {
		return fmt.Errorf("应用: %w", err)
	}
	appCodeSet := pointsStringSet(targets.AppCodes)
	if err := validatePointsTargetCodes(targets.PackageCodes, func(codes []string) (int, error) {
		rows, err := q.VideoPackage.WithContext(ctx).Select(q.VideoPackage.PackageCode, q.VideoPackage.AppCode).
			Where(q.VideoPackage.PackageCode.In(codes...)).Find()
		if err != nil {
			return 0, err
		}
		found := make(map[string]struct{}, len(rows))
		for _, target := range rows {
			if target == nil {
				continue
			}
			if len(appCodeSet) > 0 {
				if _, ok := appCodeSet[target.AppCode]; !ok {
					continue
				}
			}
			found[target.PackageCode] = struct{}{}
		}
		return len(found), nil
	}); err != nil {
		return fmt.Errorf("安装包: %w", err)
	}
	packageCodeSet := pointsStringSet(targets.PackageCodes)
	if err := validatePointsTargetCodes(targets.VersionCodes, func(codes []string) (int, error) {
		rows, err := q.VideoPackageVersion.WithContext(ctx).
			Select(q.VideoPackageVersion.VersionCode, q.VideoPackageVersion.PackageCode).
			Where(q.VideoPackageVersion.VersionCode.In(codes...)).Find()
		if err != nil {
			return 0, err
		}
		found := make(map[string]struct{}, len(rows))
		for _, target := range rows {
			if target != nil {
				if _, ok := packageCodeSet[target.PackageCode]; ok {
					found[target.VersionCode] = struct{}{}
				}
			}
		}
		return len(found), nil
	}); err != nil {
		return fmt.Errorf("版本: %w", err)
	}
	if err := validatePointsTargetCodes(targets.CountryCodes, func(codes []string) (int, error) {
		rows, err := q.VideoCountry.WithContext(ctx).Select(q.VideoCountry.Code).Where(q.VideoCountry.Code.In(codes...)).Find()
		return uniquePointsTargetCount(rows, func(item *model.VideoCountry) string { return item.Code }), err
	}); err != nil {
		return fmt.Errorf("国家: %w", err)
	}
	if err := validatePointsTargetCodes(targets.ChannelCodes, func(codes []string) (int, error) {
		rows, err := q.VideoChannel.WithContext(ctx).Select(q.VideoChannel.ChannelCode).Where(q.VideoChannel.ChannelCode.In(codes...)).Find()
		return uniquePointsTargetCount(rows, func(item *model.VideoChannel) string { return item.ChannelCode }), err
	}); err != nil {
		return fmt.Errorf("渠道: %w", err)
	}

	if err := r.deleteTargets(ctx, item.ID); err != nil {
		return err
	}
	now := time.Now()
	apps := make([]*model.VideoPointsApp, 0, len(targets.AppCodes))
	for _, code := range targets.AppCodes {
		apps = append(apps, &model.VideoPointsApp{ID: nextPointsAppID(), PointsID: item.ID, AppCode: code, CreatedAt: now, UpdatedAt: now})
	}
	if len(apps) > 0 {
		if err := q.VideoPointsApp.WithContext(ctx).Create(apps...); err != nil {
			return err
		}
	}
	packages := make([]*model.VideoPointsPackage, 0, len(targets.PackageCodes))
	for _, code := range targets.PackageCodes {
		packages = append(packages, &model.VideoPointsPackage{PointsID: int64(item.ID), PackageCode: code, CreatedAt: now, UpdatedAt: now})
	}
	if len(packages) > 0 {
		if err := q.VideoPointsPackage.WithContext(ctx).Create(packages...); err != nil {
			return err
		}
	}
	versions := make([]*model.VideoPointsVersion, 0, len(targets.VersionCodes))
	for _, code := range targets.VersionCodes {
		versions = append(versions, &model.VideoPointsVersion{PointsID: item.ID, VersionCode: code, CreatedAt: now, UpdatedAt: now})
	}
	if len(versions) > 0 {
		if err := q.VideoPointsVersion.WithContext(ctx).Create(versions...); err != nil {
			return err
		}
	}
	countries := make([]*model.VideoPointsCountry, 0, len(targets.CountryCodes))
	for _, code := range targets.CountryCodes {
		countries = append(countries, &model.VideoPointsCountry{PointsID: int64(item.ID), CountryCode: code, CreatedAt: now, UpdatedAt: now})
	}
	if len(countries) > 0 {
		if err := q.VideoPointsCountry.WithContext(ctx).Create(countries...); err != nil {
			return err
		}
	}
	channels := make([]*model.VideoPointsChannel, 0, len(targets.ChannelCodes))
	for _, code := range targets.ChannelCodes {
		channels = append(channels, &model.VideoPointsChannel{PointsID: item.ID, ChannelCode: code, CreatedAt: now, UpdatedAt: now})
	}
	if len(channels) > 0 {
		if err := q.VideoPointsChannel.WithContext(ctx).Create(channels...); err != nil {
			return err
		}
	}
	return nil
}

func (r *PointsRepo) ClearDefaults(ctx context.Context, packageCode, resourceType string, exceptID uint64) error {
	q := qFrom(ctx)
	relation := q.VideoPointsPackage
	var rawIDs []int64
	if err := relation.WithContext(ctx).Where(relation.PackageCode.Eq(packageCode)).Pluck(relation.PointsID, &rawIDs); err != nil {
		return err
	}
	ids := make([]uint64, 0, len(rawIDs))
	for _, id := range rawIDs {
		if id > 0 {
			ids = append(ids, uint64(id))
		}
	}
	if len(ids) == 0 {
		return nil
	}
	points := q.VideoPoint
	dao := points.WithContext(ctx).Where(points.ID.In(ids...), points.ResourceType.Eq(resourceType), points.IsDefault.Eq(1))
	if exceptID != 0 {
		dao = dao.Where(points.ID.Neq(exceptID))
	}
	_, err := dao.Update(points.IsDefault, int8(0))
	return err
}

func (r *PointsRepo) SetDefault(ctx context.Context, item *model.VideoPoint) error {
	if len(item.Packages) == 0 {
		return fmt.Errorf("积分套餐未关联安装包")
	}
	return Transaction(ctx, func(txCtx context.Context) error {
		seen := make(map[string]struct{}, len(item.Packages))
		for _, target := range item.Packages {
			if target == nil {
				continue
			}
			if _, ok := seen[target.PackageCode]; ok {
				continue
			}
			seen[target.PackageCode] = struct{}{}
			if err := r.ClearDefaults(txCtx, target.PackageCode, item.ResourceType, item.ID); err != nil {
				return err
			}
		}
		points := qFrom(txCtx).VideoPoint
		_, err := points.WithContext(txCtx).Where(points.ID.Eq(item.ID)).Update(points.IsDefault, int8(1))
		return err
	})
}

func (r *PointsRepo) UpdateStatus(ctx context.Context, id uint64, status int8) error {
	q := qFrom(ctx).VideoPoint
	_, err := q.WithContext(ctx).Where(q.ID.Eq(id)).Update(q.Status, status)
	return err
}

func (r *PointsRepo) DeleteWithTargets(ctx context.Context, id uint64) error {
	return Transaction(ctx, func(txCtx context.Context) error {
		if err := r.deleteTargets(txCtx, id); err != nil {
			return err
		}
		points := qFrom(txCtx).VideoPoint
		_, err := points.WithContext(txCtx).Where(points.ID.Eq(id)).Delete()
		return err
	})
}

func (r *PointsRepo) deleteTargets(ctx context.Context, id uint64) error {
	q := qFrom(ctx)
	if _, err := q.VideoPointsApp.WithContext(ctx).Unscoped().Where(q.VideoPointsApp.PointsID.Eq(id)).Delete(); err != nil {
		return err
	}
	if _, err := q.VideoPointsPackage.WithContext(ctx).Unscoped().Where(q.VideoPointsPackage.PointsID.Eq(int64(id))).Delete(); err != nil {
		return err
	}
	if _, err := q.VideoPointsVersion.WithContext(ctx).Unscoped().Where(q.VideoPointsVersion.PointsID.Eq(id)).Delete(); err != nil {
		return err
	}
	if _, err := q.VideoPointsCountry.WithContext(ctx).Unscoped().Where(q.VideoPointsCountry.PointsID.Eq(int64(id))).Delete(); err != nil {
		return err
	}
	if _, err := q.VideoPointsChannel.WithContext(ctx).Unscoped().Where(q.VideoPointsChannel.PointsID.Eq(id)).Delete(); err != nil {
		return err
	}
	return nil
}

func (r *PointsRepo) loadAssociations(ctx context.Context, items []model.VideoPoint) error {
	if len(items) == 0 {
		return nil
	}
	ids := make([]uint64, 0, len(items))
	indexByID := make(map[uint64]int, len(items))
	for i := range items {
		ids = append(ids, items[i].ID)
		indexByID[items[i].ID] = i
		items[i].Apps = []*model.VideoApp{}
		items[i].Packages = []*model.VideoPackage{}
		items[i].PackageVersion = []*model.VideoPackageVersion{}
		items[i].Country = []*model.VideoCountry{}
		items[i].Channels = []*model.VideoChannel{}
	}
	q := qFrom(ctx)

	appRelations, err := q.VideoPointsApp.WithContext(ctx).Where(q.VideoPointsApp.PointsID.In(ids...)).Find()
	if err != nil {
		return err
	}
	appCodes := make([]string, 0, len(appRelations))
	for _, relation := range appRelations {
		appCodes = append(appCodes, relation.AppCode)
	}
	appByCode := make(map[string]*model.VideoApp, len(appCodes))
	if len(appCodes) > 0 {
		apps, err := q.VideoApp.WithContext(ctx).Where(q.VideoApp.AppCode.In(appCodes...)).Find()
		if err != nil {
			return err
		}
		for _, target := range apps {
			if target != nil {
				appByCode[target.AppCode] = target
			}
		}
	}
	for _, relation := range appRelations {
		if index, ok := indexByID[relation.PointsID]; ok && appByCode[relation.AppCode] != nil {
			items[index].Apps = append(items[index].Apps, appByCode[relation.AppCode])
		}
	}

	packageRelations, err := q.VideoPointsPackage.WithContext(ctx).Where(q.VideoPointsPackage.PointsID.In(pointsUint64sToInt64s(ids)...)).Find()
	if err != nil {
		return err
	}
	packageCodes := make([]string, 0, len(packageRelations))
	for _, relation := range packageRelations {
		packageCodes = append(packageCodes, relation.PackageCode)
	}
	packageByCode := make(map[string]*model.VideoPackage, len(packageCodes))
	if len(packageCodes) > 0 {
		packages, err := q.VideoPackage.WithContext(ctx).Where(q.VideoPackage.PackageCode.In(packageCodes...)).Find()
		if err != nil {
			return err
		}
		for _, target := range packages {
			if target != nil {
				packageByCode[target.PackageCode] = target
			}
		}
	}
	for _, relation := range packageRelations {
		if relation.PointsID > 0 {
			if index, ok := indexByID[uint64(relation.PointsID)]; ok && packageByCode[relation.PackageCode] != nil {
				items[index].Packages = append(items[index].Packages, packageByCode[relation.PackageCode])
			}
		}
	}

	versionRelations, err := q.VideoPointsVersion.WithContext(ctx).Where(q.VideoPointsVersion.PointsID.In(ids...)).Find()
	if err != nil {
		return err
	}
	versionCodes := make([]string, 0, len(versionRelations))
	for _, relation := range versionRelations {
		versionCodes = append(versionCodes, relation.VersionCode)
	}
	versionByCode := make(map[string]*model.VideoPackageVersion, len(versionCodes))
	if len(versionCodes) > 0 {
		versions, err := q.VideoPackageVersion.WithContext(ctx).Where(q.VideoPackageVersion.VersionCode.In(versionCodes...)).Find()
		if err != nil {
			return err
		}
		for _, target := range versions {
			if target != nil {
				versionByCode[target.VersionCode] = target
			}
		}
	}
	for _, relation := range versionRelations {
		if index, ok := indexByID[relation.PointsID]; ok && versionByCode[relation.VersionCode] != nil {
			items[index].PackageVersion = append(items[index].PackageVersion, versionByCode[relation.VersionCode])
		}
	}

	countryRelations, err := q.VideoPointsCountry.WithContext(ctx).Where(q.VideoPointsCountry.PointsID.In(pointsUint64sToInt64s(ids)...)).Find()
	if err != nil {
		return err
	}
	countryCodes := make([]string, 0, len(countryRelations))
	for _, relation := range countryRelations {
		countryCodes = append(countryCodes, relation.CountryCode)
	}
	countryByCode := make(map[string]*model.VideoCountry, len(countryCodes))
	if len(countryCodes) > 0 {
		countries, err := q.VideoCountry.WithContext(ctx).Where(q.VideoCountry.Code.In(countryCodes...)).Find()
		if err != nil {
			return err
		}
		for _, target := range countries {
			if target != nil {
				countryByCode[target.Code] = target
			}
		}
	}
	for _, relation := range countryRelations {
		if relation.PointsID > 0 {
			if index, ok := indexByID[uint64(relation.PointsID)]; ok && countryByCode[relation.CountryCode] != nil {
				items[index].Country = append(items[index].Country, countryByCode[relation.CountryCode])
			}
		}
	}

	channelRelations, err := q.VideoPointsChannel.WithContext(ctx).Where(q.VideoPointsChannel.PointsID.In(ids...)).Find()
	if err != nil {
		return err
	}
	channelCodes := make([]string, 0, len(channelRelations))
	for _, relation := range channelRelations {
		channelCodes = append(channelCodes, relation.ChannelCode)
	}
	channelByCode := make(map[string]*model.VideoChannel, len(channelCodes))
	if len(channelCodes) > 0 {
		channels, err := q.VideoChannel.WithContext(ctx).Where(q.VideoChannel.ChannelCode.In(channelCodes...)).Find()
		if err != nil {
			return err
		}
		for _, target := range channels {
			if target != nil {
				channelByCode[target.ChannelCode] = target
			}
		}
	}
	for _, relation := range channelRelations {
		if index, ok := indexByID[relation.PointsID]; ok && channelByCode[relation.ChannelCode] != nil {
			items[index].Channels = append(items[index].Channels, channelByCode[relation.ChannelCode])
		}
	}
	return nil
}

func validatePointsTargetCodes(codes []string, count func([]string) (int, error)) error {
	if len(codes) == 0 {
		return nil
	}
	actual, err := count(codes)
	if err != nil {
		return err
	}
	if actual != len(codes) {
		return fmt.Errorf("一个或多个关联对象不存在或不属于所选上级")
	}
	return nil
}

func pointsStringSet(values []string) map[string]struct{} {
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		result[value] = struct{}{}
	}
	return result
}

func uniquePointsTargetCount[T any](items []*T, codeOf func(*T) string) int {
	found := make(map[string]struct{}, len(items))
	for _, item := range items {
		if item != nil {
			found[codeOf(item)] = struct{}{}
		}
	}
	return len(found)
}

func pointsUint64sToInt64s(values []uint64) []int64 {
	result := make([]int64, 0, len(values))
	for _, value := range values {
		if value <= uint64(^uint64(0)>>1) {
			result = append(result, int64(value))
		}
	}
	return result
}

var pointsAppID atomic.Uint64

func nextPointsAppID() uint64 {
	now := uint64(time.Now().UnixNano())
	for {
		last := pointsAppID.Load()
		next := now
		if next <= last {
			next = last + 1
		}
		if pointsAppID.CompareAndSwap(last, next) {
			return next
		}
	}
}

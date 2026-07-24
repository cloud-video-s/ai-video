package repository

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"ai-video/internal/gen/model"

	"gorm.io/gen/field"
)

type VIPSubscriptionRepo struct {
	BaseRepo[model.VideoVipSubscription]
}

func NewVIPSubscriptionRepo() *VIPSubscriptionRepo { return &VIPSubscriptionRepo{} }

type VIPSubscriptionListFilter struct {
	AppCode            string
	PackageCode        string
	VersionCode        string
	CountryCode        string
	ChannelCode        string
	LevelID            uint64
	VipType            uint64
	VipTypes           []uint64
	UserType           uint32
	SubscriptionStatus uint32
	DisplayMode        *int8
	Status             *int8
	IsSubscription     *int8
	Keyword            string
}

func (r *VIPSubscriptionRepo) PageList(ctx context.Context, page, pageSize int, filter *VIPSubscriptionListFilter) ([]model.VideoVipSubscription, int64, error) {
	q := qFrom(ctx)
	vip := q.VideoVipSubscription
	dao := vip.WithContext(ctx)
	if filter != nil {
		idSets, err := r.targetFilterIDs(ctx, filter)
		if err != nil {
			return nil, 0, err
		}
		for _, ids := range idSets {
			if len(ids) == 0 {
				return []model.VideoVipSubscription{}, 0, nil
			}
			dao = dao.Where(vip.ID.In(ids...))
		}
		if filter.LevelID != 0 {
			dao = dao.Where(vip.LevelID.Eq(filter.LevelID))
		}
		if filter.VipType > 0 {
			dao = dao.Where(vip.VipType.Eq(filter.VipType))
		}
		if filter.DisplayMode != nil {
			dao = dao.Where(vip.DisplayMode.Eq(*filter.DisplayMode))
		}
		if filter.Status != nil {
			dao = dao.Where(vip.Status.Eq(*filter.Status))
		}
		if filter.IsSubscription != nil {
			dao = dao.Where(vip.IsSubscription.Eq(*filter.IsSubscription))
		}
		if filter.Keyword != "" {
			keyword := "%" + filter.Keyword + "%"
			dao = dao.Where(field.Or(
				vip.SukCode.Like(keyword), vip.Name.Like(keyword),
				vip.Description.Like(keyword), vip.Remark.Like(keyword),
			))
		}
	}
	total, err := dao.Count()
	if err != nil {
		return nil, 0, err
	}
	rows, err := dao.Order(vip.Sort.Asc(), vip.ID.Desc()).Offset((page - 1) * pageSize).Limit(pageSize).Find()
	if err != nil {
		return nil, 0, err
	}
	items := valuesOf(rows)
	if err := r.loadAssociations(ctx, items); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *VIPSubscriptionRepo) Recommend(ctx context.Context, req *VIPSubscriptionListFilter) (*model.VideoVipSubscription, error) {
	q := qFrom(ctx).VideoVipSubscription
	dao := q.WithContext(ctx).Where(q.VipType.Eq(req.VipType), q.Status.Eq(1), q.DisplayMode.Eq(1))
	if req.LevelID > 0 {
		dao = dao.Where(q.LevelID.Eq(req.LevelID))
	}
	if req.IsSubscription != nil {
		dao = dao.Where(q.IsSubscription.Eq(*req.IsSubscription))
	}
	if req.Keyword != "" {
		keyword := "%" + req.Keyword + "%"
		dao = dao.Where(field.Or(
			q.SukCode.Like(keyword), q.Name.Like(keyword),
			q.Description.Like(keyword), q.Remark.Like(keyword),
		))
	}
	rows, err := dao.Order(q.IsDefault.Desc(), q.Sort.Asc(), q.ID.Desc()).First()
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *VIPSubscriptionRepo) VipList(ctx context.Context, req *VIPSubscriptionListFilter) ([]model.VideoVipSubscription, error) {
	q := qFrom(ctx).VideoVipSubscription
	dao := q.WithContext(ctx).Where(q.Status.Eq(1), q.DisplayMode.Eq(1))
	if req.LevelID > 0 {
		dao = dao.Where(q.LevelID.Eq(req.LevelID))
	}
	if req.IsSubscription != nil {
		dao = dao.Where(q.IsSubscription.Eq(*req.IsSubscription))
	}
	if len(req.VipTypes) > 0 {
		dao = dao.Where(q.VipType.In(req.VipTypes...))
	}
	if req.Keyword != "" {
		keyword := "%" + req.Keyword + "%"
		dao = dao.Where(field.Or(
			q.SukCode.Like(keyword), q.Name.Like(keyword),
			q.Description.Like(keyword), q.Remark.Like(keyword),
		))
	}
	rows, err := dao.Order(q.IsDefault.Desc(), q.Sort.Desc(), q.ID.Desc()).Find()
	if err != nil {
		return nil, err
	}
	items := valuesOf(rows)
	if err := r.loadAssociations(ctx, items); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *VIPSubscriptionRepo) GetDetail(ctx context.Context, id uint64) (*model.VideoVipSubscription, error) {
	q := qFrom(ctx).VideoVipSubscription
	item, err := q.WithContext(ctx).Where(q.ID.Eq(id)).First()
	if err != nil {
		return nil, err
	}
	items := []model.VideoVipSubscription{*item}
	if err := r.loadAssociations(ctx, items); err != nil {
		return nil, err
	}
	return &items[0], nil
}

func (r *VIPSubscriptionRepo) targetFilterIDs(ctx context.Context, filter *VIPSubscriptionListFilter) ([][]uint64, error) {
	q := qFrom(ctx)
	sets := make([][]uint64, 0, 5)
	pluckUint64 := func(enabled bool, pluck func(*[]uint64) error) error {
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
	if err := pluckUint64(filter.AppCode != "", func(ids *[]uint64) error {
		relation := q.VideoVipSubscriptionApp
		return relation.WithContext(ctx).Where(relation.AppCode.Eq(filter.AppCode)).Pluck(relation.SubscriptionID, ids)
	}); err != nil {
		return nil, err
	}
	if err := pluckUint64(filter.PackageCode != "", func(ids *[]uint64) error {
		relation := q.VideoVipSubscriptionPackage
		return relation.WithContext(ctx).Where(relation.PackageCode.Eq(filter.PackageCode)).Pluck(relation.SubscriptionID, ids)
	}); err != nil {
		return nil, err
	}
	if err := pluckUint64(filter.VersionCode != "", func(ids *[]uint64) error {
		relation := q.VideoVipSubscriptionVersion
		return relation.WithContext(ctx).Where(relation.VersionCode.Eq(filter.VersionCode)).Pluck(relation.SubscriptionID, ids)
	}); err != nil {
		return nil, err
	}
	pluckInt64 := func(enabled bool, pluck func(*[]int64) error) error {
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
	if err := pluckInt64(filter.CountryCode != "", func(ids *[]int64) error {
		relation := q.VideoVipSubscriptionCountry
		return relation.WithContext(ctx).Where(relation.CountryCode.Eq(filter.CountryCode)).Pluck(relation.SubscriptionID, ids)
	}); err != nil {
		return nil, err
	}
	if err := pluckInt64(filter.ChannelCode != "", func(ids *[]int64) error {
		relation := q.VideoVipSubscriptionChannel
		return relation.WithContext(ctx).Where(relation.ChannelCode.Eq(filter.ChannelCode)).Pluck(relation.SubscriptionID, ids)
	}); err != nil {
		return nil, err
	}
	return sets, nil
}

func matchesVIPRecommendation(item *model.VideoVipSubscription, filter *VIPSubscriptionListFilter) bool {
	return targetMatches(item.Apps, filter.AppCode, true, func(value *model.VideoApp) string { return value.AppCode }) &&
		targetMatches(item.Packages, filter.PackageCode, false, func(value *model.VideoPackage) string { return value.PackageCode }) &&
		targetMatches(item.PackageVersion, filter.VersionCode, true, func(value *model.VideoPackageVersion) string { return value.VersionCode }) &&
		targetMatches(item.Country, filter.CountryCode, true, func(value *model.VideoCountry) string { return value.Code }) &&
		targetMatches(item.Channels, filter.ChannelCode, true, func(value *model.VideoChannel) string { return value.ChannelCode })
}

func targetMatches[T any](items []*T, code string, emptyIsWildcard bool, codeOf func(*T) string) bool {
	if code == "" {
		return true
	}
	if len(items) == 0 {
		return emptyIsWildcard
	}
	for _, item := range items {
		if item != nil && codeOf(item) == code {
			return true
		}
	}
	return false
}

func (r *VIPSubscriptionRepo) loadAssociations(ctx context.Context, items []model.VideoVipSubscription) error {
	if len(items) == 0 {
		return nil
	}
	subscriptionIDs := make([]uint64, 0, len(items))
	levelIDs := make([]uint64, 0, len(items))
	indexByID := make(map[uint64]int, len(items))
	for i := range items {
		subscriptionIDs = append(subscriptionIDs, items[i].ID)
		indexByID[items[i].ID] = i
		items[i].Apps = make([]*model.VideoApp, 0)
		items[i].Packages = make([]*model.VideoPackage, 0)
		items[i].PackageVersion = make([]*model.VideoPackageVersion, 0)
		items[i].Country = make([]*model.VideoCountry, 0)
		items[i].Channels = make([]*model.VideoChannel, 0)
		if items[i].LevelID > 0 {
			levelIDs = append(levelIDs, items[i].LevelID)
		}
	}

	q := qFrom(ctx)
	if len(levelIDs) > 0 {
		levels, err := q.VideoVipSubscriptionLevel.WithContext(ctx).Where(q.VideoVipSubscriptionLevel.ID.In(levelIDs...)).Find()
		if err != nil {
			return err
		}
		levelByID := make(map[uint64]*model.VideoVipSubscriptionLevel, len(levels))
		for _, item := range levels {
			if item != nil {
				levelByID[item.ID] = item
			}
		}
		for i := range items {
			if level := levelByID[items[i].LevelID]; level != nil {
				items[i].SubscriptionLevel = *level
			}
		}
	}

	appRelations, err := q.VideoVipSubscriptionApp.WithContext(ctx).
		Where(q.VideoVipSubscriptionApp.SubscriptionID.In(subscriptionIDs...)).Find()
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
		for _, item := range apps {
			if item != nil {
				appByCode[item.AppCode] = item
			}
		}
	}
	for _, relation := range appRelations {
		if index, ok := indexByID[relation.SubscriptionID]; ok {
			if item := appByCode[relation.AppCode]; item != nil {
				items[index].Apps = append(items[index].Apps, item)
			}
		}
	}

	packageRelations, err := q.VideoVipSubscriptionPackage.WithContext(ctx).
		Where(q.VideoVipSubscriptionPackage.SubscriptionID.In(subscriptionIDs...)).Find()
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
		for _, item := range packages {
			if item != nil {
				packageByCode[item.PackageCode] = item
			}
		}
	}
	for _, relation := range packageRelations {
		if index, ok := indexByID[relation.SubscriptionID]; ok {
			if item := packageByCode[relation.PackageCode]; item != nil {
				items[index].Packages = append(items[index].Packages, item)
			}
		}
	}

	versionRelations, err := q.VideoVipSubscriptionVersion.WithContext(ctx).
		Where(q.VideoVipSubscriptionVersion.SubscriptionID.In(subscriptionIDs...)).Find()
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
		for _, item := range versions {
			if item != nil {
				versionByCode[item.VersionCode] = item
			}
		}
	}
	for _, relation := range versionRelations {
		if index, ok := indexByID[relation.SubscriptionID]; ok {
			if item := versionByCode[relation.VersionCode]; item != nil {
				items[index].PackageVersion = append(items[index].PackageVersion, item)
			}
		}
	}

	countryRelations, err := q.VideoVipSubscriptionCountry.WithContext(ctx).
		Where(q.VideoVipSubscriptionCountry.SubscriptionID.In(uint64sToInt64s(subscriptionIDs)...)).Find()
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
		for _, item := range countries {
			if item != nil {
				countryByCode[item.Code] = item
			}
		}
	}
	for _, relation := range countryRelations {
		if relation.SubscriptionID > 0 {
			if index, ok := indexByID[uint64(relation.SubscriptionID)]; ok {
				if item := countryByCode[relation.CountryCode]; item != nil {
					items[index].Country = append(items[index].Country, item)
				}
			}
		}
	}

	channelRelations, err := q.VideoVipSubscriptionChannel.WithContext(ctx).
		Where(q.VideoVipSubscriptionChannel.SubscriptionID.In(uint64sToInt64s(subscriptionIDs)...)).Find()
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
		for _, item := range channels {
			if item != nil {
				channelByCode[item.ChannelCode] = item
			}
		}
	}
	for _, relation := range channelRelations {
		if relation.SubscriptionID > 0 {
			if index, ok := indexByID[uint64(relation.SubscriptionID)]; ok {
				if item := channelByCode[relation.ChannelCode]; item != nil {
					items[index].Channels = append(items[index].Channels, item)
				}
			}
		}
	}
	return nil
}

func uint64sToInt64s(values []uint64) []int64 {
	result := make([]int64, 0, len(values))
	for _, value := range values {
		if value <= uint64(^uint64(0)>>1) {
			result = append(result, int64(value))
		}
	}
	return result
}

// GetAppleProduct 按 iOS SKU 和调用方包名解析启用的 VIP 商品。
func (r *VIPSubscriptionRepo) GetAppleProduct(ctx context.Context, productID, packageCode string) (*model.VideoVipSubscription, error) {
	q := qFrom(ctx)
	appPackage, err := q.VideoPackage.WithContext(ctx).Select(q.VideoPackage.PackageCode).
		Where(q.VideoPackage.PackageCode.Eq(packageCode)).First()
	if err != nil {
		return nil, err
	}
	relation := q.VideoVipSubscriptionPackage
	var subscriptionIDs []uint64
	if err := relation.WithContext(ctx).Where(relation.PackageCode.Eq(appPackage.PackageCode)).
		Pluck(relation.SubscriptionID, &subscriptionIDs); err != nil {
		return nil, err
	}
	if len(subscriptionIDs) == 0 {
		return nil, fmt.Errorf("VIP product is not associated with package %s", packageCode)
	}
	vip := q.VideoVipSubscription
	return vip.WithContext(ctx).Where(
		vip.ID.In(subscriptionIDs...), vip.SukCode.Eq(productID), vip.Status.Eq(1),
	).First()
}

func (r *VIPSubscriptionRepo) UpdateFields(ctx context.Context, item *model.VideoVipSubscription) error {
	q := qFrom(ctx).VideoVipSubscription
	_, err := q.WithContext(ctx).Where(q.ID.Eq(item.ID)).Select(
		q.VipType, q.SukCode, q.Name, q.LevelID, q.Currency,
		q.FirstSubscriptionPrice, q.FirstSubscriptionRevenue, q.FirstBonusPoints, q.OriginalPrice,
		q.VIPDurationDays, q.TrialDays, q.RenewalText, q.BadgeText, q.AgreementDefaultChecked,
		q.DisplayMode, q.Status, q.FreeTrial, q.IsSubscription, q.IsDefault,
		q.SubscriptionDescription, q.SubscriptionPrice, q.SubscriptionRevenue, q.SubscriptionPoints,
		q.SubscriptionPeriod, q.Sort, q.Description, q.Remark,
	).Updates(item)
	return err
}

type VIPSubscriptionTargets struct {
	AppCodes     []string
	PackageCodes []string
	VersionCodes []string
	CountryCodes []string
	ChannelCodes []string
}

func (r *VIPSubscriptionRepo) ReplaceTargets(ctx context.Context, item *model.VideoVipSubscription, targets VIPSubscriptionTargets) error {
	q := qFrom(ctx)
	if err := validateVIPTargetIDs(targets.AppCodes, func(codes []string) (int, error) {
		rows, err := q.VideoApp.WithContext(ctx).Select(q.VideoApp.AppCode).Where(q.VideoApp.AppCode.In(codes...)).Find()
		return uniqueVIPTargetCount(rows, func(item *model.VideoApp) string { return item.AppCode }), err
	}); err != nil {
		return fmt.Errorf("apps: %w", err)
	}
	appCodeSet := stringSet(targets.AppCodes)
	if err := validateVIPTargetIDs(targets.PackageCodes, func(codes []string) (int, error) {
		rows, err := q.VideoPackage.WithContext(ctx).Select(q.VideoPackage.PackageCode, q.VideoPackage.AppCode).Where(q.VideoPackage.PackageCode.In(codes...)).Find()
		if err != nil {
			return 0, err
		}
		found := make(map[string]struct{}, len(rows))
		for _, item := range rows {
			if item == nil {
				continue
			}
			if len(appCodeSet) > 0 {
				if _, ok := appCodeSet[item.AppCode]; !ok {
					continue
				}
			}
			found[item.PackageCode] = struct{}{}
		}
		return len(found), nil
	}); err != nil {
		return fmt.Errorf("packages: %w", err)
	}
	packageCodeSet := stringSet(targets.PackageCodes)
	if err := validateVIPTargetIDs(targets.VersionCodes, func(codes []string) (int, error) {
		rows, err := q.VideoPackageVersion.WithContext(ctx).
			Select(q.VideoPackageVersion.VersionCode, q.VideoPackageVersion.PackageCode).
			Where(q.VideoPackageVersion.VersionCode.In(codes...)).Find()
		if err != nil {
			return 0, err
		}
		found := make(map[string]struct{}, len(rows))
		for _, item := range rows {
			if item == nil {
				continue
			}
			if _, ok := packageCodeSet[item.PackageCode]; ok {
				found[item.VersionCode] = struct{}{}
			}
		}
		return len(found), nil
	}); err != nil {
		return fmt.Errorf("versions: %w", err)
	}
	if err := validateVIPTargetIDs(targets.CountryCodes, func(codes []string) (int, error) {
		rows, err := q.VideoCountry.WithContext(ctx).Select(q.VideoCountry.Code).Where(q.VideoCountry.Code.In(codes...)).Find()
		return uniqueVIPTargetCount(rows, func(item *model.VideoCountry) string { return item.Code }), err
	}); err != nil {
		return fmt.Errorf("countries: %w", err)
	}
	if err := validateVIPTargetIDs(targets.ChannelCodes, func(codes []string) (int, error) {
		rows, err := q.VideoChannel.WithContext(ctx).Select(q.VideoChannel.ChannelCode).Where(q.VideoChannel.ChannelCode.In(codes...)).Find()
		return uniqueVIPTargetCount(rows, func(item *model.VideoChannel) string { return item.ChannelCode }), err
	}); err != nil {
		return fmt.Errorf("channels: %w", err)
	}

	now := time.Now()
	appRelation := q.VideoVipSubscriptionApp
	if _, err := appRelation.WithContext(ctx).Unscoped().Where(appRelation.SubscriptionID.Eq(item.ID)).Delete(); err != nil {
		return err
	}
	apps := make([]*model.VideoVipSubscriptionApp, 0, len(targets.AppCodes))
	for _, id := range targets.AppCodes {
		apps = append(apps, &model.VideoVipSubscriptionApp{
			ID: nextVIPSubscriptionAppID(), SubscriptionID: item.ID, AppCode: id, CreatedAt: now, UpdatedAt: now,
		})
	}
	if len(apps) > 0 {
		if err := appRelation.WithContext(ctx).Create(apps...); err != nil {
			return err
		}
	}

	packageRelation := q.VideoVipSubscriptionPackage
	if _, err := packageRelation.WithContext(ctx).Unscoped().Where(packageRelation.SubscriptionID.Eq(item.ID)).Delete(); err != nil {
		return err
	}
	packages := make([]*model.VideoVipSubscriptionPackage, 0, len(targets.PackageCodes))
	for _, id := range targets.PackageCodes {
		packages = append(packages, &model.VideoVipSubscriptionPackage{SubscriptionID: item.ID, PackageCode: id, CreatedAt: now, UpdatedAt: now})
	}
	if len(packages) > 0 {
		if err := packageRelation.WithContext(ctx).Create(packages...); err != nil {
			return err
		}
	}

	versionRelation := q.VideoVipSubscriptionVersion
	if _, err := versionRelation.WithContext(ctx).Unscoped().Where(versionRelation.SubscriptionID.Eq(item.ID)).Delete(); err != nil {
		return err
	}
	versions := make([]*model.VideoVipSubscriptionVersion, 0, len(targets.VersionCodes))
	for _, id := range targets.VersionCodes {
		versions = append(versions, &model.VideoVipSubscriptionVersion{SubscriptionID: item.ID, VersionCode: id, CreatedAt: now, UpdatedAt: now})
	}
	if len(versions) > 0 {
		if err := versionRelation.WithContext(ctx).Create(versions...); err != nil {
			return err
		}
	}

	countryRelation := q.VideoVipSubscriptionCountry
	if _, err := countryRelation.WithContext(ctx).Unscoped().Where(countryRelation.SubscriptionID.Eq(int64(item.ID))).Delete(); err != nil {
		return err
	}
	countries := make([]*model.VideoVipSubscriptionCountry, 0, len(targets.CountryCodes))
	for _, code := range targets.CountryCodes {
		countries = append(countries, &model.VideoVipSubscriptionCountry{
			SubscriptionID: int64(item.ID), CountryCode: code, CreatedAt: now, UpdatedAt: now,
		})
	}
	if len(countries) > 0 {
		if err := countryRelation.WithContext(ctx).Create(countries...); err != nil {
			return err
		}
	}

	channelRelation := q.VideoVipSubscriptionChannel
	if _, err := channelRelation.WithContext(ctx).Unscoped().Where(channelRelation.SubscriptionID.Eq(int64(item.ID))).Delete(); err != nil {
		return err
	}
	channels := make([]*model.VideoVipSubscriptionChannel, 0, len(targets.ChannelCodes))
	for _, code := range targets.ChannelCodes {
		channels = append(channels, &model.VideoVipSubscriptionChannel{
			SubscriptionID: int64(item.ID), ChannelCode: code, CreatedAt: now, UpdatedAt: now,
		})
	}
	if len(channels) > 0 {
		if err := channelRelation.WithContext(ctx).Create(channels...); err != nil {
			return err
		}
	}
	return nil
}

func validateVIPTargetIDs(ids []string, count func([]string) (int, error)) error {
	if len(ids) == 0 {
		return nil
	}
	actual, err := count(ids)
	if err != nil {
		return err
	}
	if actual != len(ids) {
		return fmt.Errorf("one or more targets do not exist")
	}
	return nil
}

func stringSet(values []string) map[string]struct{} {
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		result[value] = struct{}{}
	}
	return result
}

func uniqueVIPTargetCount[T any](items []*T, codeOf func(*T) string) int {
	found := make(map[string]struct{}, len(items))
	for _, item := range items {
		if item != nil {
			found[codeOf(item)] = struct{}{}
		}
	}
	return len(found)
}

var vipSubscriptionAppID atomic.Uint64

func nextVIPSubscriptionAppID() uint64 {
	now := uint64(time.Now().UnixNano())
	for {
		last := vipSubscriptionAppID.Load()
		next := now
		if next <= last {
			next = last + 1
		}
		if vipSubscriptionAppID.CompareAndSwap(last, next) {
			return next
		}
	}
}

func (r *VIPSubscriptionRepo) ClearDefaults(ctx context.Context, packageCode string, vipType uint64, exceptID uint64) error {
	q := qFrom(ctx)
	relation := q.VideoVipSubscriptionPackage
	var ids []uint64
	if err := relation.WithContext(ctx).Where(relation.PackageCode.Eq(packageCode)).Pluck(relation.SubscriptionID, &ids); err != nil {
		return err
	}
	if len(ids) == 0 {
		return nil
	}
	vip := q.VideoVipSubscription
	dao := vip.WithContext(ctx).Where(vip.ID.In(ids...), vip.VipType.Eq(vipType), vip.IsDefault.Eq(1))
	if exceptID != 0 {
		dao = dao.Where(vip.ID.Neq(exceptID))
	}
	_, err := dao.Update(vip.IsDefault, int8(0))
	return err
}

func (r *VIPSubscriptionRepo) UpdateStatus(ctx context.Context, id uint64, status int8) error {
	q := qFrom(ctx).VideoVipSubscription
	_, err := q.WithContext(ctx).Where(q.ID.Eq(id)).Update(q.Status, status)
	return err
}

func (r *VIPSubscriptionRepo) UpdateDisplayMode(ctx context.Context, id uint64, mode int8) error {
	q := qFrom(ctx).VideoVipSubscription
	_, err := q.WithContext(ctx).Where(q.ID.Eq(id)).Update(q.DisplayMode, mode)
	return err
}

func (r *VIPSubscriptionRepo) SetDefault(ctx context.Context, item *model.VideoVipSubscription) error {
	if len(item.Packages) == 0 {
		return fmt.Errorf("VIP subscription must be associated with at least one package")
	}
	return Transaction(ctx, func(txCtx context.Context) error {
		for _, appPackage := range item.Packages {
			if appPackage == nil {
				continue
			}
			if err := r.ClearDefaults(txCtx, appPackage.PackageCode, item.VipType, item.ID); err != nil {
				return err
			}
		}
		q := qFrom(txCtx).VideoVipSubscription
		_, err := q.WithContext(txCtx).Where(q.ID.Eq(item.ID)).Update(q.IsDefault, int8(1))
		return err
	})
}

func (r *VIPSubscriptionRepo) DeleteWithTargets(ctx context.Context, id uint64) error {
	return Transaction(ctx, func(txCtx context.Context) error {
		q := qFrom(txCtx)
		if _, err := q.VideoVipSubscriptionApp.WithContext(txCtx).Unscoped().Where(q.VideoVipSubscriptionApp.SubscriptionID.Eq(id)).Delete(); err != nil {
			return err
		}
		if _, err := q.VideoVipSubscriptionPackage.WithContext(txCtx).Unscoped().Where(q.VideoVipSubscriptionPackage.SubscriptionID.Eq(id)).Delete(); err != nil {
			return err
		}
		if _, err := q.VideoVipSubscriptionVersion.WithContext(txCtx).Unscoped().Where(q.VideoVipSubscriptionVersion.SubscriptionID.Eq(id)).Delete(); err != nil {
			return err
		}
		if _, err := q.VideoVipSubscriptionCountry.WithContext(txCtx).Unscoped().Where(q.VideoVipSubscriptionCountry.SubscriptionID.Eq(int64(id))).Delete(); err != nil {
			return err
		}
		if _, err := q.VideoVipSubscriptionChannel.WithContext(txCtx).Unscoped().Where(q.VideoVipSubscriptionChannel.SubscriptionID.Eq(int64(id))).Delete(); err != nil {
			return err
		}
		vip := q.VideoVipSubscription
		_, err := vip.WithContext(txCtx).Where(vip.ID.Eq(id)).Delete()
		return err
	})
}

func (r *VIPSubscriptionRepo) PackageCount(ctx context.Context, packageCode string) (int64, error) {
	q := qFrom(ctx).VideoVipSubscriptionPackage
	return q.WithContext(ctx).Where(q.PackageCode.Eq(packageCode)).Count()
}

func (r *VIPSubscriptionRepo) GetLevelByID(ctx context.Context, id uint64) (*model.VideoVipSubscriptionLevel, error) {
	q := qFrom(ctx).VideoVipSubscriptionLevel
	return q.WithContext(ctx).Where(q.ID.Eq(id)).First()
}

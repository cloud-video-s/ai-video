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
	AppCode        string
	PackageCode    string
	VersionCode    string
	CountryCode    string
	PlacementKey   string
	LevelID        int64
	PlanType       string
	VipType        string
	DisplayMode    *int8
	Status         *int8
	IsSubscription *bool
	Keyword        string
}

type VIPSubscriptionRecord struct {
	model.VideoVipSubscription
	Placement         *model.VideoVipPlacement         `json:"placement,omitempty"`
	SubscriptionLevel *model.VideoVipSubscriptionLevel `json:"subscription_level,omitempty"`
	Apps              []model.VideoApp                 `json:"apps"`
	Packages          []model.VideoPackage             `json:"packages"`
	Versions          []model.VideoPackageVersion      `json:"versions"`
	Countries         []model.VideoCountry             `json:"countries"`
	Channels          []model.VideoChannel             `json:"channels"`
}

func (r *VIPSubscriptionRepo) PageList(ctx context.Context, page, pageSize int, filter *VIPSubscriptionListFilter) ([]VIPSubscriptionRecord, int64, error) {
	q := qFrom(ctx)
	vip := q.VideoVipSubscription
	dao := vip.WithContext(ctx)
	if filter != nil {
		var ids []uint64
		var err error
		switch {
		case filter.AppCode != "":
			relation := q.VideoVipSubscriptionApp
			err = relation.WithContext(ctx).Where(relation.AppCode.Eq(filter.AppCode)).Pluck(relation.SubscriptionID, &ids)
		case filter.PackageCode != "":
			relation := q.VideoVipSubscriptionPackage
			err = relation.WithContext(ctx).Where(relation.PackageCode.Eq(filter.PackageCode)).Pluck(relation.SubscriptionID, &ids)
		case filter.VersionCode != "":
			relation := q.VideoVipSubscriptionVersion
			err = relation.WithContext(ctx).Where(relation.VersionCode.Eq(filter.VersionCode)).Pluck(relation.SubscriptionID, &ids)
		case filter.CountryCode != "":
			relation := q.VideoVipSubscriptionCountry
			err = relation.WithContext(ctx).Where(relation.CountryCode.Eq(filter.CountryCode)).Pluck(relation.SubscriptionID, &ids)
		}
		if err != nil {
			return nil, 0, err
		}
		if filter.AppCode != "" || filter.PackageCode != "" || filter.VersionCode != "" || filter.CountryCode != "" {
			if len(ids) == 0 {
				return []VIPSubscriptionRecord{}, 0, nil
			}
			dao = dao.Where(vip.ID.In(ids...))
		}
		if filter.PlacementKey != "" {
			dao = dao.Where(vip.PlacementKey.Eq(filter.PlacementKey))
		}
		if filter.LevelID != 0 {
			dao = dao.Where(vip.LevelID.Eq(filter.LevelID))
		}
		if filter.PlanType != "" {
			dao = dao.Where(vip.PlanType.Eq(filter.PlanType))
		}
		if filter.VipType != "" {
			dao = dao.Where(vip.VipType.Eq(filter.VipType))
		}
		if filter.DisplayMode != nil {
			dao = dao.Where(vip.DisplayMode.Eq(*filter.DisplayMode))
		}
		if filter.Status != nil {
			dao = dao.Where(vip.Status.Eq(*filter.Status))
		}
		if filter.IsSubscription != nil {
			value := int8(0)
			if *filter.IsSubscription {
				value = 1
			}
			dao = dao.Where(vip.IsSubscription.Eq(value))
		}
		if filter.Keyword != "" {
			keyword := "%" + filter.Keyword + "%"
			dao = dao.Where(field.Or(
				vip.ProductCode.Like(keyword), vip.Name.Like(keyword),
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
	records, err := r.loadRecords(ctx, valuesOf(rows))
	return records, total, err
}

func (r *VIPSubscriptionRepo) Recommend(ctx context.Context, req *VIPSubscriptionListFilter) (*model.VideoVipSubscription, error) {
	sql := qFrom(ctx)
	q := sql.VideoVipSubscription

	dao := q.WithContext(ctx).Where(q.VipType.Eq(req.VipType))
	var err error
	var ids []uint64
	switch {
	case req.AppCode != "":
		relation := sql.VideoVipSubscriptionApp
		err = relation.WithContext(ctx).Where(relation.AppCode.Eq(req.AppCode)).Pluck(relation.SubscriptionID, &ids)
	case req.PackageCode != "":
		relation := sql.VideoVipSubscriptionPackage
		err = relation.WithContext(ctx).Where(relation.PackageCode.Eq(req.PackageCode)).Pluck(relation.SubscriptionID, &ids)
	case req.VersionCode != "":
		relation := sql.VideoVipSubscriptionVersion
		err = relation.WithContext(ctx).Where(relation.VersionCode.Eq(req.VersionCode)).Pluck(relation.SubscriptionID, &ids)
	case req.CountryCode != "":
		relation := sql.VideoVipSubscriptionCountry
		err = relation.WithContext(ctx).Where(relation.CountryCode.Eq(req.CountryCode)).Pluck(relation.SubscriptionID, &ids)
	}
	if err != nil {
		return nil, err
	}
	if req.AppCode != "" || req.PackageCode != "" || req.VersionCode != "" || req.CountryCode != "" {
		if len(ids) == 0 {
			return nil, nil
		}
		dao = dao.Where(q.ID.In(ids...))
	}
	//if req.PlacementKey != "" {
	//	dao = dao.Where(q.PlacementKey.Eq(req.PlacementKey))
	//}
	if req.LevelID != 0 {
		dao = dao.Where(q.LevelID.Eq(req.LevelID))
	}
	if req.PlanType != "" {
		dao = dao.Where(q.PlanType.Eq(req.PlanType))
	}
	if req.VipType != "" {
		dao = dao.Where(q.VipType.Eq(req.VipType))
	}
	if req.DisplayMode != nil {
		dao = dao.Where(q.DisplayMode.Eq(*req.DisplayMode))
	}
	if req.Status != nil {
		dao = dao.Where(q.Status.Eq(*req.Status))
	}
	if req.IsSubscription != nil {
		value := int8(0)
		if *req.IsSubscription {
			value = 1
		}
		dao = dao.Where(q.IsSubscription.Eq(value))
	}
	if req.Keyword != "" {
		keyword := "%" + req.Keyword + "%"
		dao = dao.Where(field.Or(
			q.ProductCode.Like(keyword), q.Name.Like(keyword),
			q.Description.Like(keyword), q.Remark.Like(keyword),
		))
	}
	return dao.Order(q.Sort.Desc()).First()
}

func (r *VIPSubscriptionRepo) GetDetail(ctx context.Context, id uint64) (*VIPSubscriptionRecord, error) {
	q := qFrom(ctx).VideoVipSubscription
	item, err := q.WithContext(ctx).Where(q.ID.Eq(id)).First()
	if err != nil {
		return nil, err
	}
	records, err := r.loadRecords(ctx, []model.VideoVipSubscription{*item})
	if err != nil {
		return nil, err
	}
	return &records[0], nil
}

func (r *VIPSubscriptionRepo) loadRecords(ctx context.Context, items []model.VideoVipSubscription) ([]VIPSubscriptionRecord, error) {
	result := make([]VIPSubscriptionRecord, len(items))
	if len(items) == 0 {
		return result, nil
	}
	subscriptionIDs := make([]uint64, 0, len(items))
	placementKeys := make([]string, 0, len(items))
	levelIDs := make([]uint64, 0, len(items))
	for i := range items {
		result[i].VideoVipSubscription = items[i]
		subscriptionIDs = append(subscriptionIDs, items[i].ID)
		placementKeys = append(placementKeys, items[i].PlacementKey)
		if items[i].LevelID > 0 {
			levelIDs = append(levelIDs, uint64(items[i].LevelID))
		}
	}
	indexByID := make(map[uint64]int, len(items))
	for i := range items {
		indexByID[items[i].ID] = i
	}

	q := qFrom(ctx)
	placements, err := q.VideoVipPlacement.WithContext(ctx).Where(q.VideoVipPlacement.PlacementKey.In(placementKeys...)).Find()
	if err != nil {
		return nil, err
	}
	placementByKey := make(map[string]*model.VideoVipPlacement, len(placements))
	for _, item := range placements {
		if item != nil {
			placementByKey[item.PlacementKey] = item
		}
	}
	levels, err := q.VideoVipSubscriptionLevel.WithContext(ctx).Where(q.VideoVipSubscriptionLevel.ID.In(levelIDs...)).Find()
	if err != nil {
		return nil, err
	}
	levelByID := make(map[uint64]*model.VideoVipSubscriptionLevel, len(levels))
	for _, item := range levels {
		if item != nil {
			levelByID[item.ID] = item
		}
	}
	for i := range items {
		result[i].Placement = placementByKey[items[i].PlacementKey]
		if items[i].LevelID > 0 {
			result[i].SubscriptionLevel = levelByID[uint64(items[i].LevelID)]
		}
	}

	appRelations, err := q.VideoVipSubscriptionApp.WithContext(ctx).
		Where(q.VideoVipSubscriptionApp.SubscriptionID.In(subscriptionIDs...)).Find()
	if err != nil {
		return nil, err
	}
	appCodes := make([]string, 0, len(appRelations))
	for _, relation := range appRelations {
		appCodes = append(appCodes, relation.AppCode)
	}
	appByCode := make(map[string]model.VideoApp, len(appCodes))
	if len(appCodes) > 0 {
		apps, err := q.VideoApp.WithContext(ctx).Where(q.VideoApp.AppCode.In(appCodes...)).Find()
		if err != nil {
			return nil, err
		}
		for _, item := range apps {
			if item != nil {
				appByCode[item.AppCode] = *item
			}
		}
	}
	for _, relation := range appRelations {
		if index, ok := indexByID[relation.SubscriptionID]; ok {
			if item, found := appByCode[relation.AppCode]; found {
				result[index].Apps = append(result[index].Apps, item)
			}
		}
	}

	packageRelations, err := q.VideoVipSubscriptionPackage.WithContext(ctx).
		Where(q.VideoVipSubscriptionPackage.SubscriptionID.In(subscriptionIDs...)).Find()
	if err != nil {
		return nil, err
	}
	packageCodes := make([]string, 0, len(packageRelations))
	for _, relation := range packageRelations {
		packageCodes = append(packageCodes, relation.PackageCode)
	}
	packageByCode := make(map[string]model.VideoPackage, len(packageCodes))
	if len(packageCodes) > 0 {
		packages, err := q.VideoPackage.WithContext(ctx).Where(q.VideoPackage.PackageCode.In(packageCodes...)).Find()
		if err != nil {
			return nil, err
		}
		for _, item := range packages {
			if item != nil {
				packageByCode[item.PackageCode] = *item
			}
		}
	}
	for _, relation := range packageRelations {
		if index, ok := indexByID[relation.SubscriptionID]; ok {
			if item, found := packageByCode[relation.PackageCode]; found {
				result[index].Packages = append(result[index].Packages, item)
			}
		}
	}

	versionRelations, err := q.VideoVipSubscriptionVersion.WithContext(ctx).
		Where(q.VideoVipSubscriptionVersion.SubscriptionID.In(subscriptionIDs...)).Find()
	if err != nil {
		return nil, err
	}
	versionCodes := make([]string, 0, len(versionRelations))
	for _, relation := range versionRelations {
		versionCodes = append(versionCodes, relation.VersionCode)
	}
	versionByCode := make(map[string]model.VideoPackageVersion, len(versionCodes))
	if len(versionCodes) > 0 {
		versions, err := q.VideoPackageVersion.WithContext(ctx).Where(q.VideoPackageVersion.VersionCode.In(versionCodes...)).Find()
		if err != nil {
			return nil, err
		}
		for _, item := range versions {
			if item != nil {
				versionByCode[item.VersionCode] = *item
			}
		}
	}
	for _, relation := range versionRelations {
		if index, ok := indexByID[relation.SubscriptionID]; ok {
			if item, found := versionByCode[relation.VersionCode]; found {
				result[index].Versions = append(result[index].Versions, item)
			}
		}
	}
	return result, nil
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
		vip.ID.In(subscriptionIDs...), vip.VipType.Eq("ios"), vip.ProductCode.Eq(productID), vip.Status.Eq(1),
	).First()
}

func (r *VIPSubscriptionRepo) UpdateFields(ctx context.Context, item *model.VideoVipSubscription) error {
	q := qFrom(ctx).VideoVipSubscription
	_, err := q.WithContext(ctx).Where(q.ID.Eq(item.ID)).Select(
		q.VipType, q.ProductCode, q.Name, q.LevelID, q.PlanType, q.AppVersion, q.Currency,
		q.FirstSubscriptionPrice, q.FirstSubscriptionRevenue, q.FirstBonusPoints, q.OriginalPrice,
		q.VIPDurationDays, q.TrialDays, q.RenewalText, q.BadgeText, q.AgreementDefaultChecked,
		q.DisplayMode, q.Status, q.FreeTrial, q.IsSubscription, q.IsDefault,
		q.SubscriptionDescription, q.SubscriptionPrice, q.SubscriptionRevenue, q.SubscriptionPoints,
		q.SubscriptionPeriod, q.Sort, q.Description, q.Remark, q.PlacementKey,
	).Updates(item)
	return err
}

type VIPSubscriptionTargets struct {
	AppCodes     []string
	PackageCodes []string
	VersionCdes  []string
	CountryCode  []string
	ChannelCodes []string
}

func (r *VIPSubscriptionRepo) ReplaceTargets(ctx context.Context, item *model.VideoVipSubscription, targets VIPSubscriptionTargets) error {
	q := qFrom(ctx)
	if err := validateVIPTargetIDs(targets.AppCodes, func(codes []string) (int, error) {
		rows, err := q.VideoApp.WithContext(ctx).Select(q.VideoApp.AppCode).Where(q.VideoApp.AppCode.In(codes...)).Find()
		return len(rows), err
	}); err != nil {
		return fmt.Errorf("apps: %w", err)
	}
	if err := validateVIPTargetIDs(targets.PackageCodes, func(codes []string) (int, error) {
		rows, err := q.VideoPackage.WithContext(ctx).Select(q.VideoPackage.PackageCode).Where(q.VideoPackage.PackageCode.In(codes...)).Find()
		return len(rows), err
	}); err != nil {
		return fmt.Errorf("packages: %w", err)
	}
	if err := validateVIPTargetIDs(targets.VersionCdes, func(codes []string) (int, error) {
		rows, err := q.VideoPackageVersion.WithContext(ctx).Select(q.VideoPackageVersion.VersionCode).Where(q.VideoPackageVersion.VersionCode.In(codes...)).Find()
		return len(rows), err
	}); err != nil {
		return fmt.Errorf("versions: %w", err)
	}
	if err := validateVIPTargetIDs(targets.CountryCode, func(codes []string) (int, error) {
		rows, err := q.VideoCountry.WithContext(ctx).Select(q.VideoCountry.Code).Where(q.VideoCountry.Code.In(codes...)).Find()
		return len(rows), err
	}); err != nil {
		return fmt.Errorf("countries: %w", err)
	}
	if err := validateVIPTargetIDs(targets.ChannelCodes, func(codes []string) (int, error) {
		rows, err := q.VideoChannel.WithContext(ctx).Select(q.VideoChannel.ChannelCode).Where(q.VideoChannel.ChannelCode.In(codes...)).Find()
		return len(rows), err
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
	versions := make([]*model.VideoVipSubscriptionVersion, 0, len(targets.VersionCdes))
	for _, id := range targets.VersionCdes {
		versions = append(versions, &model.VideoVipSubscriptionVersion{SubscriptionID: item.ID, VersionCode: id, CreatedAt: now, UpdatedAt: now})
	}
	if len(versions) > 0 {
		if err := versionRelation.WithContext(ctx).Create(versions...); err != nil {
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

func (r *VIPSubscriptionRepo) ClearDefaults(ctx context.Context, packageCode string, vipType string, exceptID uint64) error {
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

func (r *VIPSubscriptionRepo) SetDefault(ctx context.Context, item *VIPSubscriptionRecord) error {
	if len(item.Packages) == 0 {
		return fmt.Errorf("VIP subscription must be associated with at least one package")
	}
	return Transaction(ctx, func(txCtx context.Context) error {
		for _, appPackage := range item.Packages {
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
		vip := q.VideoVipSubscription
		_, err := vip.WithContext(txCtx).Where(vip.ID.Eq(id)).Delete()
		return err
	})
}

func (r *VIPSubscriptionRepo) PackageCount(ctx context.Context, packageCode string) (int64, error) {
	q := qFrom(ctx).VideoVipSubscriptionPackage
	return q.WithContext(ctx).Where(q.PackageCode.Eq(packageCode)).Count()
}

func (r *VIPSubscriptionRepo) GetPlacementByKey(ctx context.Context, key string) (*model.VideoVipPlacement, error) {
	q := qFrom(ctx).VideoVipPlacement
	return q.WithContext(ctx).Where(q.PlacementKey.Eq(key)).First()
}

func (r *VIPSubscriptionRepo) GetLevelByID(ctx context.Context, id int64) (*model.VideoVipSubscriptionLevel, error) {
	q := qFrom(ctx).VideoVipSubscriptionLevel
	return q.WithContext(ctx).Where(q.ID.Eq(uint64(id))).First()
}

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
	AppID          uint64
	PackageID      uint64
	VersionID      uint64
	CountryID      uint64
	PlacementKey   string
	LevelID        int64
	PlanType       string
	VipType        string
	DisplayMode    *int8
	Status         *int8
	IsSubscription *bool
	Keyword        string
}

func (r *VIPSubscriptionRepo) PageList(ctx context.Context, page, pageSize int, filter *VIPSubscriptionListFilter) ([]model.VideoVipSubscription, int64, error) {
	q := qFrom(ctx)
	vip := q.VideoVipSubscription
	dao := vip.WithContext(ctx)
	if filter != nil {
		var ids []uint64
		var err error
		switch {
		case filter.AppID != 0:
			relation := q.VideoVipSubscriptionApp
			err = relation.WithContext(ctx).Where(relation.AppID.Eq(filter.AppID)).Pluck(relation.SubscriptionID, &ids)
		case filter.PackageID != 0:
			relation := q.VideoVipSubscriptionPackage
			err = relation.WithContext(ctx).Where(relation.PackageID.Eq(filter.PackageID)).Pluck(relation.SubscriptionID, &ids)
		case filter.VersionID != 0:
			relation := q.VideoVipSubscriptionVersion
			var rawIDs []int64
			err = relation.WithContext(ctx).Where(relation.VersionID.Eq(int64(filter.VersionID))).Pluck(relation.SubscriptionID, &rawIDs)
			ids = int64IDsToUint64(rawIDs)
		case filter.CountryID != 0:
			relation := q.VideoVipSubscriptionCountry
			var rawIDs []int64
			err = relation.WithContext(ctx).Where(relation.CountryID.Eq(filter.CountryID)).Pluck(relation.SubscriptionID, &rawIDs)
			ids = int64IDsToUint64(rawIDs)
		}
		if err != nil {
			return nil, 0, err
		}
		if filter.AppID != 0 || filter.PackageID != 0 || filter.VersionID != 0 || filter.CountryID != 0 {
			if len(ids) == 0 {
				return []model.VideoVipSubscription{}, 0, nil
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
	rows, err := dao.Preload(vip.Placement, vip.SubscriptionLevel, vip.Apps, vip.Packages, vip.Versions, vip.Countries).
		Order(vip.Sort.Asc(), vip.ID.Desc()).Offset((page - 1) * pageSize).Limit(pageSize).Find()
	return valuesOf(rows), total, err
}

func int64IDsToUint64(values []int64) []uint64 {
	result := make([]uint64, 0, len(values))
	for _, value := range values {
		if value > 0 {
			result = append(result, uint64(value))
		}
	}
	return result
}

func (r *VIPSubscriptionRepo) GetDetail(ctx context.Context, id uint64) (*model.VideoVipSubscription, error) {
	q := qFrom(ctx).VideoVipSubscription
	return q.WithContext(ctx).Preload(q.Placement, q.SubscriptionLevel, q.Apps, q.Packages, q.Versions, q.Countries).
		Where(q.ID.Eq(id)).First()
}

// GetAppleProduct 按 iOS SKU 和调用方包名解析启用的 VIP 商品。
func (r *VIPSubscriptionRepo) GetAppleProduct(ctx context.Context, productID, packageCode string) (*model.VideoVipSubscription, error) {
	q := qFrom(ctx)
	appPackage, err := q.VideoPackage.WithContext(ctx).Select(q.VideoPackage.ID).
		Where(q.VideoPackage.PackageCode.Eq(packageCode)).First()
	if err != nil {
		return nil, err
	}
	relation := q.VideoVipSubscriptionPackage
	var subscriptionIDs []uint64
	if err := relation.WithContext(ctx).Where(relation.PackageID.Eq(appPackage.ID)).
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
	AppIDs     []uint64
	PackageIDs []uint64
	VersionIDs []uint64
	CountryIDs []uint64
}

func (r *VIPSubscriptionRepo) ReplaceTargets(ctx context.Context, item *model.VideoVipSubscription, targets VIPSubscriptionTargets) error {
	q := qFrom(ctx)
	if err := validateVIPTargetIDs(targets.AppIDs, func(ids []uint64) (int, error) {
		rows, err := q.VideoApp.WithContext(ctx).Select(q.VideoApp.ID).Where(q.VideoApp.ID.In(ids...)).Find()
		return len(rows), err
	}); err != nil {
		return fmt.Errorf("apps: %w", err)
	}
	if err := validateVIPTargetIDs(targets.PackageIDs, func(ids []uint64) (int, error) {
		rows, err := q.VideoPackage.WithContext(ctx).Select(q.VideoPackage.ID).Where(q.VideoPackage.ID.In(ids...)).Find()
		return len(rows), err
	}); err != nil {
		return fmt.Errorf("packages: %w", err)
	}
	if err := validateVIPTargetIDs(targets.VersionIDs, func(ids []uint64) (int, error) {
		rows, err := q.VideoPackageVersion.WithContext(ctx).Select(q.VideoPackageVersion.ID).Where(q.VideoPackageVersion.ID.In(ids...)).Find()
		return len(rows), err
	}); err != nil {
		return fmt.Errorf("versions: %w", err)
	}
	if err := validateVIPTargetIDs(targets.CountryIDs, func(ids []uint64) (int, error) {
		rows, err := q.VideoCountry.WithContext(ctx).Select(q.VideoCountry.ID).Where(q.VideoCountry.ID.In(ids...)).Find()
		return len(rows), err
	}); err != nil {
		return fmt.Errorf("countries: %w", err)
	}

	now := time.Now()
	appRelation := q.VideoVipSubscriptionApp
	if _, err := appRelation.WithContext(ctx).Unscoped().Where(appRelation.SubscriptionID.Eq(item.ID)).Delete(); err != nil {
		return err
	}
	apps := make([]*model.VideoVipSubscriptionApp, 0, len(targets.AppIDs))
	for _, id := range targets.AppIDs {
		apps = append(apps, &model.VideoVipSubscriptionApp{
			ID: nextVIPSubscriptionAppID(), SubscriptionID: item.ID, AppID: id, CreatedAt: now, UpdatedAt: now,
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
	packages := make([]*model.VideoVipSubscriptionPackage, 0, len(targets.PackageIDs))
	for _, id := range targets.PackageIDs {
		packages = append(packages, &model.VideoVipSubscriptionPackage{SubscriptionID: item.ID, PackageID: id, CreatedAt: now, UpdatedAt: now})
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
	versions := make([]*model.VideoVipSubscriptionVersion, 0, len(targets.VersionIDs))
	for _, id := range targets.VersionIDs {
		versions = append(versions, &model.VideoVipSubscriptionVersion{SubscriptionID: item.ID, VersionID: int64(id), CreatedAt: now, UpdatedAt: now})
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
	countries := make([]*model.VideoVipSubscriptionCountry, 0, len(targets.CountryIDs))
	for _, id := range targets.CountryIDs {
		countries = append(countries, &model.VideoVipSubscriptionCountry{SubscriptionID: int64(item.ID), CountryID: id, CreatedAt: now, UpdatedAt: now})
	}
	if len(countries) > 0 {
		return countryRelation.WithContext(ctx).Create(countries...)
	}
	return nil
}

func validateVIPTargetIDs(ids []uint64, count func([]uint64) (int, error)) error {
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

func (r *VIPSubscriptionRepo) ClearDefaults(ctx context.Context, packageID uint64, vipType string, exceptID uint64) error {
	q := qFrom(ctx)
	relation := q.VideoVipSubscriptionPackage
	var ids []uint64
	if err := relation.WithContext(ctx).Where(relation.PackageID.Eq(packageID)).Pluck(relation.SubscriptionID, &ids); err != nil {
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
			if err := r.ClearDefaults(txCtx, appPackage.ID, item.VipType, item.ID); err != nil {
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

func (r *VIPSubscriptionRepo) PackageCount(ctx context.Context, packageID uint64) (int64, error) {
	q := qFrom(ctx).VideoVipSubscriptionPackage
	return q.WithContext(ctx).Where(q.PackageID.Eq(packageID)).Count()
}

func (r *VIPSubscriptionRepo) GetPlacementByKey(ctx context.Context, key string) (*model.VideoVipPlacement, error) {
	q := qFrom(ctx).VideoVipPlacement
	return q.WithContext(ctx).Where(q.PlacementKey.Eq(key)).First()
}

func (r *VIPSubscriptionRepo) GetLevelByID(ctx context.Context, id int64) (*model.VideoVipSubscriptionLevel, error) {
	q := qFrom(ctx).VideoVipSubscriptionLevel
	return q.WithContext(ctx).Where(q.ID.Eq(uint64(id))).First()
}

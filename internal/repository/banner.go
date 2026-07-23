package repository

import (
	"context"
	"fmt"
	"sort"
	"strings"

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
	dao := dbFrom(ctx).Model(&model.VideoBanner{})
	if filter != nil {
		if filter.PositionKey != "" {
			dao = dao.Where("EXISTS (SELECT 1 FROM video_banner_display_position vbdp JOIN video_display_position vdp ON vdp.position_key = vbdp.position_key WHERE vbdp.banner_id = video_banner.id AND vbdp.position_key = ? AND vdp.deleted_at IS NULL)", filter.PositionKey)
		}
		if filter.CountryCode != "" {
			dao = dao.Where(`EXISTS (
				SELECT 1 FROM video_banner_country vbc
				WHERE vbc.banner_id = video_banner.id AND vbc.country_code = ? AND vbc.deleted_at IS NULL
			)`, filter.CountryCode)
		}
		if filter.AppCode != "" {
			dao = dao.Where(`EXISTS (
				SELECT 1 FROM video_banner_app vba
				WHERE vba.banner_id = video_banner.id AND vba.app_code = ?
					AND vba.deleted_at IS NULL
			)`, filter.AppCode)
		}
		if filter.PackageCode != "" {
			dao = dao.Where("EXISTS (SELECT 1 FROM video_banner_app vba WHERE vba.banner_id = video_banner.id AND vba.package_code = ? AND vba.deleted_at IS NULL)", filter.PackageCode)
		}
		if filter.VersionCode != "" {
			dao = dao.Where("EXISTS (SELECT 1 FROM video_banner_app vba WHERE vba.banner_id = video_banner.id AND (vba.version_code = '' OR vba.version_code = ?) AND vba.deleted_at IS NULL)", filter.VersionCode)
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
	return db.Preload("Template").Preload("DisplayPositions").Preload("Countries")
}

func (r *BannerRepo) UpdateFields(ctx context.Context, item *model.VideoBanner) error {
	return r.BaseRepo.Update(ctx, item,
		"Name", "CoverImage", "Remark", "Sort", "JumpType", "JumpURL", "TemplateID", "Status", "SubscriptionStatus",
	)
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

// ListForClient applies delivery targeting. An empty association means the
// banner is global for that dimension; otherwise the client must match.
func (r *BannerRepo) ListForClient(ctx context.Context, targets ClientBannerTargets) ([]model.VideoBanner, error) {
	db := dbFrom(ctx).Model(&model.VideoBanner{}).
		Where("video_banner.status = ?", 1).
		Where(`(NOT EXISTS (
			SELECT 1 FROM video_banner_display_position vbdp
			WHERE vbdp.banner_id = video_banner.id AND vbdp.deleted_at IS NULL
		) OR EXISTS (
			SELECT 1 FROM video_banner_display_position vbdp
			JOIN video_display_position vdp ON vdp.position_key = vbdp.position_key
			WHERE vbdp.banner_id = video_banner.id AND vbdp.position_key = ?
				AND vdp.status = ? AND vbdp.deleted_at IS NULL AND vdp.deleted_at IS NULL
		))`, targets.PositionKey, 1).
		Where("video_banner.subscription_status IN (?, ?)", 3, targets.SubscriptionStatus)
	if targets.CountryCode != "" {
		db = db.Where(`(NOT EXISTS (
			SELECT 1 FROM video_banner_country vbc WHERE vbc.banner_id = video_banner.id AND vbc.deleted_at IS NULL
		) OR EXISTS (
			SELECT 1 FROM video_banner_country vbc JOIN video_country vc ON vc.code = vbc.country_code
			WHERE vbc.banner_id = video_banner.id AND vc.code = ? AND vbc.deleted_at IS NULL
		))`, targets.CountryCode)
	} else {
		db = db.Where("NOT EXISTS (SELECT 1 FROM video_banner_country vbc WHERE vbc.banner_id = video_banner.id AND vbc.deleted_at IS NULL)")
	}
	if targets.AppCode != "" {
		db = db.Where(`(NOT EXISTS (
			SELECT 1 FROM video_banner_app vba
			WHERE vba.banner_id = video_banner.id AND vba.app_code <> '' AND vba.deleted_at IS NULL
		) OR EXISTS (
			SELECT 1 FROM video_banner_app vba
			WHERE vba.banner_id = video_banner.id AND vba.app_code = ?
				AND (vba.package_code IS NULL OR vba.package_code = '' OR vba.package_code = ?)
				AND (vba.version_code = '' OR vba.version_code = ?)
				AND vba.deleted_at IS NULL
		))`, targets.AppCode, targets.PackageCode, targets.VersionCode)
	} else {
		db = db.Where("NOT EXISTS (SELECT 1 FROM video_banner_app vba WHERE vba.banner_id = video_banner.id AND vba.app_code <> '' AND vba.deleted_at IS NULL)")
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
	if err := db.Model(item).Association("Countries").Replace(countries); err != nil {
		return err
	}
	return replaceBannerAppTargets(db, item.ID, targets.AppTargets)
}

func (r *BannerRepo) DeleteWithTargets(ctx context.Context, id uint64) error {
	db := dbFrom(ctx)
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("banner_id = ?", id).Delete(&model.VideoBannerDisplayPosition{}).Error; err != nil {
			return err
		}
		if err := tx.Where("banner_id = ?", id).Delete(&model.VideoBannerApp{}).Error; err != nil {
			return err
		}
		return tx.Select("Countries").Delete(&model.VideoBanner{ID: id}).Error
	})
}

func replaceBannerAppTargets(db *gorm.DB, bannerID uint64, targets []BannerAppTargetInput) error {
	if err := db.Where("banner_id = ?", bannerID).Delete(&model.VideoBannerApp{}).Error; err != nil {
		return err
	}
	rows := make([]model.VideoBannerApp, 0)
	for _, target := range targets {
		if len(target.VersionCodes) == 0 {
			rows = append(rows, model.VideoBannerApp{
				BannerID: bannerID, AppCode: target.AppCode, PackageCode: target.PackageCode, VersionCode: "",
			})
			continue
		}
		for _, versionCode := range target.VersionCodes {
			rows = append(rows, model.VideoBannerApp{
				BannerID: bannerID, AppCode: target.AppCode, PackageCode: target.PackageCode, VersionCode: versionCode,
			})
		}
	}
	if len(rows) == 0 {
		return nil
	}
	return db.Create(&rows).Error
}

type bannerAppTargetRow struct {
	BannerID    uint64
	AppCode     string
	AppName     string
	PackageCode string
	PackageName string
	VersionCode string
}

func (r *BannerRepo) LoadAppTargets(ctx context.Context, bannerIDs []uint64) (map[uint64][]BannerAppTarget, error) {
	result := make(map[uint64][]BannerAppTarget, len(bannerIDs))
	if len(bannerIDs) == 0 {
		return result, nil
	}
	var rows []bannerAppTargetRow
	err := dbFrom(ctx).Table("video_banner_app AS vba").
		Select(`vba.banner_id, vba.app_code, COALESCE(va.name, '') AS app_name,
			COALESCE(vba.package_code, '') AS package_code, COALESCE(vp.package_name, '') AS package_name, vba.version_code`).
		Joins("LEFT JOIN video_app va ON va.app_code = vba.app_code AND va.deleted_at IS NULL").
		Joins("LEFT JOIN video_package vp ON vp.app_code = vba.app_code AND vp.package_code = vba.package_code AND vp.deleted_at IS NULL").
		Where("vba.banner_id IN ? AND vba.app_code <> '' AND vba.deleted_at IS NULL", bannerIDs).
		Order("vba.banner_id ASC, va.sort ASC, vp.sort ASC, vba.version_code ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	type targetKey struct {
		bannerID    uint64
		packageCode string
	}
	indexes := make(map[targetKey]int)
	allVersions := make(map[targetKey]bool)
	for _, row := range rows {
		key := targetKey{bannerID: row.BannerID, packageCode: row.PackageCode}
		index, exists := indexes[key]
		if !exists {
			index = len(result[row.BannerID])
			indexes[key] = index
			result[row.BannerID] = append(result[row.BannerID], BannerAppTarget{
				AppCode: row.AppCode, AppName: row.AppName, PackageCode: row.PackageCode,
				PackageName: row.PackageName, VersionCodes: []string{},
			})
		}
		if row.VersionCode == "" {
			allVersions[key] = true
			result[row.BannerID][index].VersionCodes = []string{}
			continue
		}
		if !allVersions[key] {
			result[row.BannerID][index].VersionCodes = append(result[row.BannerID][index].VersionCodes, row.VersionCode)
		}
	}
	return result, nil
}

func (r *BannerRepo) ListDeliveryOptions(ctx context.Context) ([]BannerDeliveryApp, error) {
	var apps []model.VideoApp
	if err := dbFrom(ctx).Where("status = ?", 1).Order("sort ASC, id ASC").Find(&apps).Error; err != nil {
		return nil, err
	}
	var packages []model.VideoPackage
	if err := dbFrom(ctx).Where("status = ?", 1).Order("sort ASC, id ASC").Find(&packages).Error; err != nil {
		return nil, err
	}
	var versions []model.VideoPackageVersion
	if err := dbFrom(ctx).Where("status = ?", 1).Order("version_code ASC, id ASC").Find(&versions).Error; err != nil {
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
	var appCount int64
	if err := dbFrom(ctx).Model(&model.VideoApp{}).
		Where("app_code = ? AND status = ?", target.AppCode, 1).Count(&appCount).Error; err != nil {
		return err
	}
	if appCount == 0 {
		return fmt.Errorf("应用 %s 不存在或已禁用", target.AppCode)
	}
	var packageCount int64
	if err := dbFrom(ctx).Model(&model.VideoPackage{}).
		Where("package_code = ? AND status = ? AND (app_code = ? OR app_code = '')", target.PackageCode, 1, target.AppCode).
		Count(&packageCount).Error; err != nil {
		return err
	}
	if packageCount == 0 {
		return fmt.Errorf("包 %s 不属于所选应用或已禁用", target.PackageCode)
	}
	if len(target.VersionCodes) == 0 {
		return nil
	}
	var count int64
	if err := dbFrom(ctx).Model(&model.VideoPackageVersion{}).
		Where("version_code IN ? AND status = ? AND (package_code = ? OR package_code = '')", target.VersionCodes, 1, target.PackageCode).
		Distinct("version_code").Count(&count).Error; err != nil {
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

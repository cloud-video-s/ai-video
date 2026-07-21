package repository

import (
	"context"
	"errors"
	"strings"

	"ai-video/internal/gen/model"

	"gorm.io/gorm"
)

type PackageRepo struct {
	BaseRepo[model.VideoPackage]
}

func NewPackageRepo() *PackageRepo {
	return &PackageRepo{}
}

type PackageListFilter struct {
	PackageCode    string
	PackageVersion string
	SystemType     string
	Status         *int8
	Keyword        string
}

func (r *PackageRepo) PageList(ctx context.Context, page, pageSize int, filter *PackageListFilter) ([]model.VideoPackage, int64, error) {
	q := &QueryOptions{Where: map[string]interface{}{}, Order: []string{"sort ASC", "id DESC"}}
	if filter != nil {
		if filter.PackageCode != "" {
			q.Where["package_code"] = filter.PackageCode
		}
		if filter.PackageVersion != "" {
			q.Where["package_version"] = filter.PackageVersion
		}
		if filter.SystemType != "" {
			q.Conds = append(q.Conds, Cond{Query: "system_types LIKE ?", Args: []interface{}{"%\"" + filter.SystemType + "\"%"}})
		}
		if filter.Status != nil {
			q.Where["status"] = *filter.Status
		}
		if filter.Keyword != "" {
			keyword := "%" + filter.Keyword + "%"
			q.Conds = append(q.Conds, Cond{
				Query: "package_name LIKE ? OR package_code LIKE ? OR package_version LIKE ? OR description LIKE ?",
				Args:  []interface{}{keyword, keyword, keyword, keyword},
			})
		}
	}
	return r.BaseRepo.PageList(ctx, page, pageSize, q)
}

func (r *PackageRepo) GetByCodeVersion(ctx context.Context, code, version string) (*model.VideoPackage, error) {
	return r.BaseRepo.GetOne(ctx, &QueryOptions{Where: map[string]interface{}{
		"package_code": code, "package_version": version,
	}})
}

func (r *PackageRepo) ListOptions(ctx context.Context) ([]model.VideoPackage, error) {
	return r.BaseRepo.List(ctx, &QueryOptions{Order: []string{"sort ASC", "id ASC"}})
}

func (r *PackageRepo) ResolveEnabledTargets(ctx context.Context, code, version string) ([]model.VideoPackage, error) {
	code = strings.TrimSpace(code)
	version = strings.TrimSpace(version)
	db := dbFrom(ctx).Model(&model.VideoPackage{}).Where("status = ?", 1)
	if code != "" {
		db = db.Where("package_code = ?", code)
	}
	if version != "" {
		db = db.Where("package_version = ?", version)
	}
	if code == "" && version == "" {
		return []model.VideoPackage{}, nil
	}
	var list []model.VideoPackage
	if err := db.Order("sort ASC, id DESC").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (r *PackageRepo) ResolveLanguage(ctx context.Context, code, version string) (string, error) {
	code = strings.TrimSpace(code)
	version = strings.TrimSpace(version)
	if code == "" {
		return "", gorm.ErrRecordNotFound
	}
	db := dbFrom(ctx).Model(&model.VideoPackage{}).Where("package_code = ? AND status = ?", code, 1)
	if version != "" {
		db = db.Where("package_version = ?", version)
	}
	var item model.VideoPackage
	err := db.Order("sort ASC, id DESC").Select("language").First(&item).Error
	if errors.Is(err, gorm.ErrRecordNotFound) && version != "" {
		err = dbFrom(ctx).Model(&model.VideoPackage{}).
			Where("package_code = ? AND status = ?", code, 1).
			Order("sort ASC, id DESC").Select("language").First(&item).Error
	}
	return item.Language, err
}

func (r *PackageRepo) UpdateFields(ctx context.Context, item *model.VideoPackage) error {
	return r.BaseRepo.Update(ctx, item,
		"PackageName", "PackageCode", "PackageVersion", "Language", "SystemTypes", "DownloadURL",
		"InstallCount", "DownloadCount", "DeviceCount", "Description", "Sort", "Status",
	)
}

func (r *PackageRepo) TemplateCount(ctx context.Context, packageID uint64) (int64, error) {
	var templateCount, typeCount int64
	if err := dbFrom(ctx).Table("video_template_package").Where("package_id = ?", packageID).Count(&templateCount).Error; err != nil {
		return 0, err
	}
	if err := dbFrom(ctx).Table("video_template_type_package").Where("package_id = ?", packageID).Count(&typeCount).Error; err != nil {
		return 0, err
	}
	return templateCount + typeCount, nil
}

func (r *PackageRepo) PointsPackageCount(ctx context.Context, packageID uint64) (int64, error) {
	var count int64
	err := dbFrom(ctx).Model(&model.VideoPointsPackage{}).Where("package_id = ?", packageID).Count(&count).Error
	return count, err
}

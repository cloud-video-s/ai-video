package repository

import (
	"context"
	"strings"

	"ai-video/internal/gen/model"
)

type PackageRepo struct {
	BaseRepo[model.VideoPackage]
}

func NewPackageRepo() *PackageRepo { return &PackageRepo{} }

type PackageListFilter struct {
	AppCode     string
	PackageCode string
	SystemType  *uint32
	Status      *int8
	Keyword     string
}

func (r *PackageRepo) PageList(ctx context.Context, page, pageSize int, filter *PackageListFilter) ([]model.VideoPackage, int64, error) {
	q := &QueryOptions{Where: map[string]interface{}{}, Order: []string{"sort ASC", "id DESC"}}
	if filter != nil {
		if filter.AppCode != "" {
			q.Where["app_code"] = filter.AppCode
		}
		if filter.PackageCode != "" {
			q.Where["package_code"] = filter.PackageCode
		}
		if filter.SystemType != nil {
			q.Where["system_type"] = *filter.SystemType
		}
		if filter.Status != nil {
			q.Where["status"] = *filter.Status
		}
		if filter.Keyword != "" {
			keyword := "%" + filter.Keyword + "%"
			q.Conds = append(q.Conds, Cond{
				Query: "package_name LIKE ? OR package_code LIKE ? OR app_code LIKE ? OR description LIKE ?",
				Args:  []interface{}{keyword, keyword, keyword, keyword},
			})
		}
	}
	return r.BaseRepo.PageList(ctx, page, pageSize, q)
}

func (r *PackageRepo) GetByCode(ctx context.Context, code string) (*model.VideoPackage, error) {
	return r.BaseRepo.GetOne(ctx, &QueryOptions{Where: map[string]interface{}{"package_code": code}})
}

func (r *PackageRepo) ListOptions(ctx context.Context) ([]model.VideoPackage, error) {
	return r.BaseRepo.List(ctx, &QueryOptions{Order: []string{"sort ASC", "id ASC"}})
}

// ResolveEnabledTargets resolves an enabled package and, when supplied, an
// enabled version belonging to that package. Version data is stored separately
// in video_package_version and therefore must not be filtered on video_package.
func (r *PackageRepo) ResolveEnabledTargets(ctx context.Context, code, version string) ([]model.VideoPackage, error) {
	code = strings.TrimSpace(code)
	version = strings.TrimSpace(version)
	if code == "" && version == "" {
		return []model.VideoPackage{}, nil
	}
	db := dbFrom(ctx).Model(&model.VideoPackage{}).Where("video_package.status = ?", 1)
	if code != "" {
		db = db.Where("video_package.package_code = ?", code)
	}
	if version != "" {
		db = db.Where(`EXISTS (
			SELECT 1 FROM video_package_version vpv
			WHERE vpv.package_code = video_package.package_code
				AND vpv.version_code = ? AND vpv.status = ? AND vpv.deleted_at IS NULL
		)`, version, 1)
	}
	var list []model.VideoPackage
	if err := db.Order("video_package.sort ASC, video_package.id DESC").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (r *PackageRepo) UpdateFields(ctx context.Context, item *model.VideoPackage) error {
	return r.BaseRepo.Update(ctx, item,
		"PackageName", "PackageCode", "AppCode", "Description", "Sort", "Status", "SystemType",
	)
}

func (r *PackageRepo) VersionCount(ctx context.Context, packageID uint64) (int64, error) {
	var count int64
	err := dbFrom(ctx).Model(&model.VideoPackageVersion{}).
		Where(`package_code = (SELECT package_code FROM video_package WHERE id = ?)`, packageID).
		Count(&count).Error
	return count, err
}

func (r *PackageRepo) TemplateCount(ctx context.Context, packageID uint64) (int64, error) {
	var templateCount, typeCount int64
	if err := dbFrom(ctx).Table("video_template_package AS relation").
		Joins("JOIN video_package vp ON vp.package_code = relation.package_code").
		Where("vp.id = ? AND relation.deleted_at IS NULL", packageID).Count(&templateCount).Error; err != nil {
		return 0, err
	}
	if err := dbFrom(ctx).Table("video_template_type_package AS relation").
		Joins("JOIN video_package vp ON vp.package_code = relation.package_code").
		Where("vp.id = ? AND relation.deleted_at IS NULL", packageID).Count(&typeCount).Error; err != nil {
		return 0, err
	}
	return templateCount + typeCount, nil
}

func (r *PackageRepo) PointsPackageCount(ctx context.Context, packageID uint64) (int64, error) {
	var count int64
	err := dbFrom(ctx).Model(&model.VideoPointsPackage{}).Where(`EXISTS (
		SELECT 1 FROM video_points_package_package vppp
		JOIN video_package vp ON vp.package_code = vppp.package_code
		WHERE vppp.product_code = video_points_package.product_code
			AND vp.id = ? AND vppp.deleted_at IS NULL
	)`, packageID).Count(&count).Error
	return count, err
}

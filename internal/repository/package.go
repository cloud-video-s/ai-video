package repository

import (
	"context"
	"strings"

	"ai-video/internal/gen/model"

	"gorm.io/gen/field"
)

type PackageRepo struct{ BaseRepo[model.VideoPackage] }

func NewPackageRepo() *PackageRepo { return &PackageRepo{} }

type PackageListFilter struct {
	AppCode     string
	PackageCode string
	SystemType  *uint32
	Status      *int8
	Keyword     string
}

func (r *PackageRepo) PageList(ctx context.Context, page, pageSize int, filter *PackageListFilter) ([]model.VideoPackage, int64, error) {
	q := qFrom(ctx).VideoPackage
	dao := q.WithContext(ctx)
	if filter != nil {
		if filter.AppCode != "" {
			dao = dao.Where(q.AppCode.Eq(filter.AppCode))
		}
		if filter.PackageCode != "" {
			dao = dao.Where(q.PackageCode.Eq(filter.PackageCode))
		}
		if filter.SystemType != nil {
			dao = dao.Where(q.SystemType.Eq(uint8(*filter.SystemType)))
		}
		if filter.Status != nil {
			dao = dao.Where(q.Status.Eq(*filter.Status))
		}
		if filter.Keyword != "" {
			keyword := "%" + filter.Keyword + "%"
			dao = dao.Where(field.Or(
				q.PackageName.Like(keyword), q.PackageCode.Like(keyword),
				q.AppCode.Like(keyword), q.Description.Like(keyword),
			))
		}
	}
	total, err := dao.Count()
	if err != nil {
		return nil, 0, err
	}
	rows, err := dao.Order(q.Sort.Asc(), q.ID.Desc()).Offset((page - 1) * pageSize).Limit(pageSize).Find()
	return valuesOf(rows), total, err
}

func (r *PackageRepo) GetByCode(ctx context.Context, code string) (*model.VideoPackage, error) {
	q := qFrom(ctx).VideoPackage
	return q.WithContext(ctx).Where(q.PackageCode.Eq(code)).First()
}

func (r *PackageRepo) ListOptions(ctx context.Context) ([]model.VideoPackage, error) {
	q := qFrom(ctx).VideoPackage
	rows, err := q.WithContext(ctx).Order(q.Sort.Asc(), q.ID.Asc()).Find()
	return valuesOf(rows), err
}

// ResolveEnabledTargets 校验启用的安装包及其可选版本。
func (r *PackageRepo) ResolveEnabledTargets(ctx context.Context, code, version string) ([]model.VideoPackage, error) {
	code = strings.TrimSpace(code)
	version = strings.TrimSpace(version)
	if code == "" && version == "" {
		return []model.VideoPackage{}, nil
	}
	q := qFrom(ctx)
	packageQuery := q.VideoPackage
	dao := packageQuery.WithContext(ctx).Where(packageQuery.Status.Eq(1))
	if code != "" {
		dao = dao.Where(packageQuery.PackageCode.Eq(code))
	}
	if version != "" {
		versionQuery := q.VideoPackageVersion
		versionDAO := versionQuery.WithContext(ctx).Select(versionQuery.PackageCode).
			Where(versionQuery.VersionCode.Eq(version), versionQuery.Status.Eq(1))
		if code != "" {
			versionDAO = versionDAO.Where(versionQuery.PackageCode.Eq(code))
		}
		var packageCodes []string
		if err := versionDAO.Pluck(versionQuery.PackageCode, &packageCodes); err != nil {
			return nil, err
		}
		if len(packageCodes) == 0 {
			return []model.VideoPackage{}, nil
		}
		dao = dao.Where(packageQuery.PackageCode.In(packageCodes...))
	}
	rows, err := dao.Order(packageQuery.Sort.Asc(), packageQuery.ID.Desc()).Find()
	return valuesOf(rows), err
}

func (r *PackageRepo) UpdateFields(ctx context.Context, item *model.VideoPackage) error {
	q := qFrom(ctx).VideoPackage
	_, err := q.WithContext(ctx).Where(q.ID.Eq(item.ID)).Select(
		q.PackageName, q.PackageCode, q.AppCode, q.Description, q.Sort, q.Status, q.SystemType,
	).Updates(item)
	return err
}

func (r *PackageRepo) VersionCount(ctx context.Context, packageID uint64) (int64, error) {
	q := qFrom(ctx)
	item, err := q.VideoPackage.WithContext(ctx).Select(q.VideoPackage.PackageCode).
		Where(q.VideoPackage.ID.Eq(packageID)).First()
	if err != nil {
		return 0, err
	}
	version := q.VideoPackageVersion
	return version.WithContext(ctx).Where(version.PackageCode.Eq(item.PackageCode)).Count()
}

func (r *PackageRepo) TemplateCount(ctx context.Context, packageCode string) (int64, error) {
	relation := qFrom(ctx).VideoTemplateTypePackage
	return relation.WithContext(ctx).Where(relation.PackageCode.Eq(packageCode)).Count()
}

func (r *PackageRepo) PointsPackageCount(ctx context.Context, packageID uint64) (int64, error) {
	q := qFrom(ctx)
	item, err := q.VideoPackage.WithContext(ctx).Select(q.VideoPackage.PackageCode).
		Where(q.VideoPackage.ID.Eq(packageID)).First()
	if err != nil {
		return 0, err
	}
	relation := q.VideoPointsPackagePackage
	return relation.WithContext(ctx).Where(relation.PackageCode.Eq(item.PackageCode)).Count()
}

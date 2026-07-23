package repository

import (
	"context"

	"ai-video/internal/gen/model"

	"gorm.io/gen/field"
)

type PackageVersionRepo struct {
	BaseRepo[model.VideoPackageVersion]
}

func NewPackageVersionRepo() *PackageVersionRepo { return &PackageVersionRepo{} }

type PackageVersionListFilter struct {
	PackageCode string
	VersionCode string
	Status      *uint32
	Keyword     string
}

func (r *PackageVersionRepo) PageList(ctx context.Context, page, pageSize int, filter *PackageVersionListFilter) ([]model.VideoPackageVersion, int64, error) {
	q := qFrom(ctx).VideoPackageVersion
	dao := q.WithContext(ctx)
	if filter != nil {
		if filter.PackageCode != "" {
			dao = dao.Where(q.PackageCode.Eq(filter.PackageCode))
		}
		if filter.VersionCode != "" {
			dao = dao.Where(q.VersionCode.Eq(filter.VersionCode))
		}
		if filter.Status != nil {
			dao = dao.Where(q.Status.Eq(uint8(*filter.Status)))
		}
		if filter.Keyword != "" {
			keyword := "%" + filter.Keyword + "%"
			dao = dao.Where(field.Or(q.PackageCode.Like(keyword), q.VersionCode.Like(keyword), q.Description.Like(keyword)))
		}
	}
	total, err := dao.Count()
	if err != nil {
		return nil, 0, err
	}
	rows, err := dao.Order(q.ID.Desc()).Offset((page - 1) * pageSize).Limit(pageSize).Find()
	return valuesOf(rows), total, err
}

func (r *PackageVersionRepo) GetByPackageVersion(ctx context.Context, packageCode, versionCode string) (*model.VideoPackageVersion, error) {
	q := qFrom(ctx).VideoPackageVersion
	return q.WithContext(ctx).Where(q.PackageCode.Eq(packageCode), q.VersionCode.Eq(versionCode)).First()
}

func (r *PackageVersionRepo) UpdateFields(ctx context.Context, item *model.VideoPackageVersion) error {
	q := qFrom(ctx).VideoPackageVersion
	_, err := q.WithContext(ctx).Where(q.ID.Eq(item.ID)).Select(
		q.PackageCode, q.VersionCode, q.DownloadURL, q.InstallCount, q.DownloadCount,
		q.DeviceCount, q.Description, q.Status,
	).Updates(item)
	return err
}

package repository

import (
	"context"

	"ai-video/internal/gen/model"
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
	q := &QueryOptions{Where: map[string]interface{}{}, Order: []string{"id DESC"}}
	if filter != nil {
		if filter.PackageCode != "" {
			q.Where["package_code"] = filter.PackageCode
		}
		if filter.VersionCode != "" {
			q.Where["version_code"] = filter.VersionCode
		}
		if filter.Status != nil {
			q.Where["status"] = *filter.Status
		}
		if filter.Keyword != "" {
			keyword := "%" + filter.Keyword + "%"
			q.Conds = append(q.Conds, Cond{
				Query: "package_code LIKE ? OR version_code LIKE ? OR description LIKE ?",
				Args:  []interface{}{keyword, keyword, keyword},
			})
		}
	}
	return r.BaseRepo.PageList(ctx, page, pageSize, q)
}

func (r *PackageVersionRepo) GetByPackageVersion(ctx context.Context, packageCode, versionCode string) (*model.VideoPackageVersion, error) {
	return r.BaseRepo.GetOne(ctx, &QueryOptions{Where: map[string]interface{}{
		"package_code": packageCode,
		"version_code": versionCode,
	}})
}

func (r *PackageVersionRepo) UpdateFields(ctx context.Context, item *model.VideoPackageVersion) error {
	return r.BaseRepo.Update(ctx, item,
		"PackageCode", "VersionCode", "DownloadURL", "InstallCount", "DownloadCount",
		"DeviceCount", "Description", "Status",
	)
}

package repository

import (
	"context"

	"ai-video/internal/gen/model"
)

type CountryRepo struct {
	BaseRepo[model.VideoCountry]
}

func NewCountryRepo() *CountryRepo {
	return &CountryRepo{}
}

type CountryListFilter struct {
	Keyword string
	Status  *int8
}

func (r *CountryRepo) PageList(ctx context.Context, page, pageSize int, filter *CountryListFilter) ([]model.VideoCountry, int64, error) {
	q := &QueryOptions{Where: map[string]interface{}{}, Order: []string{"code ASC"}}
	if filter != nil {
		if filter.Status != nil {
			q.Where["status"] = *filter.Status
		}
		if filter.Keyword != "" {
			keyword := "%" + filter.Keyword + "%"
			q.Conds = append(q.Conds, Cond{
				Query: "code LIKE ? OR name_zh LIKE ?",
				Args:  []interface{}{keyword, keyword},
			})
		}
	}
	return r.BaseRepo.PageList(ctx, page, pageSize, q)
}

func (r *CountryRepo) ListEnabled(ctx context.Context) ([]model.VideoCountry, error) {
	return r.BaseRepo.List(ctx, &QueryOptions{
		Where: map[string]interface{}{"status": int8(1)},
		Order: []string{"code ASC"},
	})
}

func (r *CountryRepo) GetEnabledByCode(ctx context.Context, code string) (*model.VideoCountry, error) {
	return r.BaseRepo.GetOne(ctx, &QueryOptions{Where: map[string]interface{}{"code": code, "status": int8(1)}})
}

func (r *CountryRepo) UpdateFields(ctx context.Context, item *model.VideoCountry) error {
	return r.BaseRepo.Update(ctx, item, "Code", "NameZh", "Status")
}

func (r *CountryRepo) TemplateCount(ctx context.Context, countryID uint64) (int64, error) {
	var templateCount, typeCount int64
	if err := dbFrom(ctx).Table("video_template_country").Where("country_id = ?", countryID).Count(&templateCount).Error; err != nil {
		return 0, err
	}
	if err := dbFrom(ctx).Table("video_template_type_country").Where("country_id = ?", countryID).Count(&typeCount).Error; err != nil {
		return 0, err
	}
	return templateCount + typeCount, nil
}

package repository

import (
	"context"
	"strings"

	"ai-video/internal/gen/model"
	"ai-video/internal/pkg/i18n"

	"gorm.io/gen/field"
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
	q := qFrom(ctx).VideoCountry
	dao := q.WithContext(ctx)
	if filter != nil {
		if filter.Status != nil {
			dao = dao.Where(q.Status.Eq(*filter.Status))
		}
		if filter.Keyword != "" {
			keyword := "%" + filter.Keyword + "%"
			dao = dao.Where(field.Or(q.Code.Like(keyword), q.NameZh.Like(keyword)))
		}
	}
	total, err := dao.Count()
	if err != nil {
		return nil, 0, err
	}
	rows, err := dao.Order(q.Code.Asc()).Offset((page - 1) * pageSize).Limit(pageSize).Find()
	return valuesOf(rows), total, err
}

func (r *CountryRepo) ListEnabled(ctx context.Context) ([]model.VideoCountry, error) {
	q := qFrom(ctx).VideoCountry
	rows, err := q.WithContext(ctx).Where(q.Status.Eq(1)).Order(q.Code.Asc()).Find()
	return valuesOf(rows), err
}

func (r *CountryRepo) GetEnabledByCode(ctx context.Context, code string) (*model.VideoCountry, error) {
	q := qFrom(ctx).VideoCountry
	return q.WithContext(ctx).Where(q.Code.Eq(code), q.Status.Eq(1)).First()
}

func (r *CountryRepo) ResolveLanguage(ctx context.Context, countryCode string) (string, error) {
	countryCode = strings.ToUpper(strings.TrimSpace(countryCode))
	item, err := r.GetEnabledByCode(ctx, countryCode)
	if err != nil {
		return "", err
	}
	if language := strings.TrimSpace(item.Language); language != "" {
		return i18n.NormalizeLocale(language), nil
	}
	return i18n.LocaleForCountry(countryCode), nil
}

func (r *CountryRepo) UpdateFields(ctx context.Context, item *model.VideoCountry) error {
	q := qFrom(ctx).VideoCountry
	_, err := q.WithContext(ctx).Where(q.ID.Eq(item.ID)).Select(q.Code, q.NameZh, q.Language, q.Status).Updates(item)
	return err
}

func (r *CountryRepo) TemplateCount(ctx context.Context, countryID uint64) (int64, error) {
	q := qFrom(ctx)
	country, err := q.VideoCountry.WithContext(ctx).Select(q.VideoCountry.Code).Where(q.VideoCountry.ID.Eq(countryID)).First()
	if err != nil {
		return 0, err
	}
	relation := q.VideoTemplateTypeCountry
	return relation.WithContext(ctx).Where(relation.CountryCode.Eq(country.Code)).Count()
}

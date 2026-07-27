package repository

import (
	"context"

	"ai-video/internal/gen/model"

	"gorm.io/gen/field"
)

type ModelRepo struct {
	BaseRepo[model.VideoModel]
}

func NewModelRepo() *ModelRepo { return &ModelRepo{} }

type ModelListFilter struct {
	Keyword    string
	PlatformID *int64
	ModelType  uint32
	Status     *uint32
}

func (r *ModelRepo) PageList(ctx context.Context, page, pageSize int, filter *ModelListFilter) ([]model.VideoModel, int64, error) {
	q := qFrom(ctx).VideoModel
	dao := q.WithContext(ctx)
	if filter != nil {
		if filter.PlatformID != nil {
			dao = dao.Where(q.PlatformID.Eq(*filter.PlatformID))
		}
		if filter.ModelType > 0 {
			dao = dao.Where(q.ModelType.Eq(filter.ModelType))
		}
		if filter.Status != nil {
			dao = dao.Where(q.Status.Eq(*filter.Status))
		}
		if filter.Keyword != "" {
			keyword := "%" + filter.Keyword + "%"
			dao = dao.Where(field.Or(q.Name.Like(keyword), q.Code.Like(keyword), q.Version.Like(keyword)))
		}
	}
	total, err := dao.Count()
	if err != nil {
		return nil, 0, err
	}
	rows, err := dao.Preload(q.Platform).Order(q.ID.Desc()).Offset((page - 1) * pageSize).Limit(pageSize).Find()
	return valuesOf(rows), total, err
}

func (r *ModelRepo) GetByIDWithPlatform(ctx context.Context, id int64) (*model.VideoModel, error) {
	q := qFrom(ctx).VideoModel
	return q.WithContext(ctx).Preload(q.Platform).Where(q.ID.Eq(id)).First()
}

func (r *ModelRepo) GetByCode(ctx context.Context, code string) (*model.VideoModel, error) {
	q := qFrom(ctx).VideoModel
	return q.WithContext(ctx).Where(q.Code.Eq(code)).First()
}

func (r *ModelRepo) UpdateFields(ctx context.Context, item *model.VideoModel) error {
	q := qFrom(ctx).VideoModel
	_, err := q.WithContext(ctx).Where(q.ID.Eq(item.ID)).Select(
		q.PlatformID, q.Name, q.Code, q.ModelType, q.Version,
		q.SubmitEndpoint, q.StatusEndpoint, q.RequestMethod, q.AuthType,
		q.Description, q.Status, q.HostURL, q.Score,
	).Updates(item)
	return err
}

// UpdateAPIKey writes the schema-managed api_key column without exposing the
// credential through generated model JSON. The column is added by the reviewed
// script in scripts/schema and is never returned in plaintext by the service.
func (r *ModelRepo) UpdateAPIKey(ctx context.Context, id int64, apiKey string) error {
	return dbFrom(ctx).Table(model.TableNameVideoModel).Where("id = ?", id).Update("api_key", apiKey).Error
}

func (r *ModelRepo) APIKeyConfigured(ctx context.Context, ids []int64) (map[int64]bool, error) {
	configured := make(map[int64]bool, len(ids))
	if len(ids) == 0 {
		return configured, nil
	}
	var rows []struct {
		ID     int64
		APIKey string
	}
	if err := dbFrom(ctx).Table(model.TableNameVideoModel).
		Select("id, api_key").Where("id IN ?", ids).Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		configured[row.ID] = row.APIKey != ""
	}
	return configured, nil
}

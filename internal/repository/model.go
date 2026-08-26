package repository

import (
	"context"

	"ai-video/internal/gen/model"

	"gorm.io/gen/field"
	"gorm.io/gorm"
)

type ModelRepo struct {
	BaseRepo[model.VideoModel]
}

func NewModelRepo() *ModelRepo { return &ModelRepo{} }

type ModelListFilter struct {
	ListSort      ListSort
	Keyword       string
	PlatformID    *int64
	ModelType     uint32
	ModelFeatures []uint32
	Status        *uint32
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
		if len(filter.ModelFeatures) > 0 {
			dao = dao.Where(q.ModelFeatures.In(filter.ModelFeatures...))
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
	listSort := ListSort{}
	if filter != nil {
		listSort = filter.ListSort
	}
	order := orderForList(listSort, map[string]field.OrderExpr{"id": q.ID}, q.ID, q.ID.Desc())
	rows, err := dao.Preload(q.Platform).Order(order...).Offset((page - 1) * pageSize).Limit(pageSize).Find()
	return valuesOf(rows), total, err
}

func (r *ModelRepo) GetByIDWithPlatform(ctx context.Context, id int64) (*model.VideoModel, error) {
	q := qFrom(ctx).VideoModel
	return q.WithContext(ctx).Preload(q.Platform).Where(q.ID.Eq(id)).First()
}

func (r *ModelRepo) ListByIDs(ctx context.Context, ids []int64) ([]model.VideoModel, error) {
	if len(ids) == 0 {
		return []model.VideoModel{}, nil
	}
	q := qFrom(ctx).VideoModel
	rows, err := q.WithContext(ctx).Where(q.ID.In(ids...)).Order(q.ID.Asc()).Find()
	return valuesOf(rows), err
}

func (r *ModelRepo) GetByCode(ctx context.Context, code string) (*model.VideoModel, error) {
	q := qFrom(ctx).VideoModel
	return q.WithContext(ctx).Where(q.Code.Eq(code)).First()
}

// GetEnabledByCode loads the platform together with an enabled model and also
// requires the platform itself to be enabled.
func (r *ModelRepo) GetEnabledByCode(ctx context.Context, code string) (*model.VideoModel, error) {
	q := qFrom(ctx).VideoModel
	item, err := q.WithContext(ctx).Preload(q.Platform).
		Where(q.Code.Eq(code), q.Status.Eq(1)).First()
	if err != nil {
		return nil, err
	}
	if item.Platform.ID == 0 || item.Platform.Status != 1 || item.Platform.DeletedAt.Valid {
		return nil, gorm.ErrRecordNotFound
	}
	return item, nil
}

// GetEnabledByID loads an enabled model and requires its platform to be
// enabled as well. Template generation uses the model ID owned by the
// template, rather than accepting a model code from the client.
func (r *ModelRepo) GetEnabledByID(ctx context.Context, id int64) (*model.VideoModel, error) {
	q := qFrom(ctx).VideoModel
	item, err := q.WithContext(ctx).Preload(q.Platform).
		Where(q.ID.Eq(id), q.Status.Eq(1)).First()
	if err != nil {
		return nil, err
	}
	if item.Platform.ID == 0 || item.Platform.Status != 1 || item.Platform.DeletedAt.Valid {
		return nil, gorm.ErrRecordNotFound
	}
	return item, nil
}

func (r *ModelRepo) ListEnabled(ctx context.Context) ([]model.VideoModel, error) {
	return r.listEnabled(ctx, 0)
}

// ListEnabledByType returns enabled models whose owning platform is also
// enabled. The explicit platform soft-delete predicate is needed because GORM
// only applies the model table's soft-delete scope automatically in this join.
func (r *ModelRepo) ListEnabledByType(ctx context.Context, modelType uint32) ([]model.VideoModel, error) {
	return r.listEnabled(ctx, modelType)
}

func (r *ModelRepo) ListEnabledByIDs(ctx context.Context, ids []int64) ([]model.VideoModel, error) {
	if len(ids) == 0 {
		return []model.VideoModel{}, nil
	}
	var rows []model.VideoModel
	err := qFrom(ctx).UnderlyingDB().Model(&model.VideoModel{}).
		Select(model.TableNameVideoModel+".*").
		Joins("JOIN "+model.TableNameVideoPlatform+" ON "+model.TableNameVideoPlatform+".id = "+model.TableNameVideoModel+".platform_id").
		Where(model.TableNameVideoModel+".id IN ?", ids).
		Where(model.TableNameVideoModel+".status = ?", 1).
		Where(model.TableNameVideoPlatform+".status = ?", 1).
		Where(model.TableNameVideoPlatform + ".deleted_at IS NULL").
		Order(model.TableNameVideoModel + ".id ASC").
		Find(&rows).Error
	return rows, err
}

func (r *ModelRepo) listEnabled(ctx context.Context, modelType uint32) ([]model.VideoModel, error) {
	var rows []model.VideoModel
	db := qFrom(ctx).UnderlyingDB().Model(&model.VideoModel{}).
		Select(model.TableNameVideoModel+".*").
		Joins("JOIN "+model.TableNameVideoPlatform+" ON "+model.TableNameVideoPlatform+".id = "+model.TableNameVideoModel+".platform_id").
		Where(model.TableNameVideoModel+".status = ?", 1).
		Where(model.TableNameVideoPlatform+".status = ?", 1).
		Where(model.TableNameVideoModel + ".model_features IN (1,3)").
		Where(model.TableNameVideoModel + ".deleted_at IS NULL").
		Where(model.TableNameVideoPlatform + ".deleted_at IS NULL")
	if modelType != 0 {
		db = db.Where(model.TableNameVideoModel+".model_type = ?", modelType)
	}
	err := db.Preload("Platform").
		Order(model.TableNameVideoModel + ".id ASC").
		Find(&rows).Error
	return rows, err
}

func (r *ModelRepo) UpdateFields(ctx context.Context, item *model.VideoModel) error {
	q := qFrom(ctx).VideoModel
	_, err := q.WithContext(ctx).Where(q.ID.Eq(item.ID)).Select(
		q.PlatformID, q.Name, q.Code, q.ModelType, q.ModelFeatures, q.Version,
		q.SubmitEndpoint, q.StatusEndpoint, q.RequestMethod, q.AuthType,
		q.Description, q.Status, q.HostURL, q.Score, q.Icon,
	).Updates(item)
	return err
}

// UpdateAPIKey writes the schema-managed api_key column without exposing the
// credential through generated model JSON. The column is added by the reviewed
// script in scripts/schema and is never returned in plaintext by the service.
func (r *ModelRepo) UpdateAPIKey(ctx context.Context, id int64, apiKey string) error {
	q := qFrom(ctx).VideoModel
	_, err := q.WithContext(ctx).Where(q.ID.Eq(id)).Update(q.APIKey, apiKey)
	return err
}

func (r *ModelRepo) APIKeyConfigured(ctx context.Context, ids []int64) (map[int64]bool, error) {
	configured := make(map[int64]bool, len(ids))
	if len(ids) == 0 {
		return configured, nil
	}
	q := qFrom(ctx).VideoModel
	rows, err := q.WithContext(ctx).Select(q.ID, q.APIKey).Where(q.ID.In(ids...)).Find()
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		configured[row.ID] = row.APIKey != ""
	}
	return configured, nil
}

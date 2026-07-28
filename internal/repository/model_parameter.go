package repository

import (
	"context"

	"ai-video/internal/gen/model"
)

type ModelParameterRepo struct {
	BaseRepo[model.VideoModelParameter]
}

func NewModelParameterRepo() *ModelParameterRepo { return &ModelParameterRepo{} }

func (r *ModelParameterRepo) ListByModel(ctx context.Context, modelID int64) ([]model.VideoModelParameter, error) {
	q := qFrom(ctx).VideoModelParameter
	rows, err := q.WithContext(ctx).Where(q.ModelID.Eq(modelID)).
		Order(q.ParameterType.Asc(), q.SortOrder.Asc(), q.ID.Asc()).Find()
	return valuesOf(rows), err
}

// ListOptionsByModels returns option parameters for all requested models in a
// single query. Rows are grouped deterministically by model and display order.
func (r *ModelParameterRepo) ListOptionsByModels(ctx context.Context, modelIDs []int64) ([]model.VideoModelParameter, error) {
	if len(modelIDs) == 0 {
		return []model.VideoModelParameter{}, nil
	}
	q := qFrom(ctx).VideoModelParameter
	rows, err := q.WithContext(ctx).
		Where(q.ModelID.In(modelIDs...), q.ParameterType.Eq(1)).
		Order(q.ModelID.Asc(), q.SortOrder.Asc(), q.ID.Asc()).
		Find()
	return valuesOf(rows), err
}

func (r *ModelParameterRepo) GetByModelAndID(ctx context.Context, modelID, id int64) (*model.VideoModelParameter, error) {
	q := qFrom(ctx).VideoModelParameter
	return q.WithContext(ctx).Where(q.ModelID.Eq(modelID), q.ID.Eq(id)).First()
}

func (r *ModelParameterRepo) GetByKey(ctx context.Context, modelID int64, key string) (*model.VideoModelParameter, error) {
	q := qFrom(ctx).VideoModelParameter
	return q.WithContext(ctx).Where(q.ModelID.Eq(modelID), q.ParamKey.Eq(key)).First()
}

func (r *ModelParameterRepo) UpdateFields(ctx context.Context, item *model.VideoModelParameter) error {
	q := qFrom(ctx).VideoModelParameter
	_, err := q.WithContext(ctx).Where(q.ID.Eq(item.ID), q.ModelID.Eq(item.ModelID)).Select(
		q.ParamKey, q.ParamType, q.IsRequired, q.DefaultValue, q.AllowedValues,
		q.Description, q.SortOrder, q.ParameterType,
	).Updates(item)
	return err
}

func (r *ModelParameterRepo) UpdateConstraints(ctx context.Context, id int64, constraints string) error {
	return dbFrom(ctx).Table(model.TableNameVideoModelParameter).
		Where("id = ?", id).Update("constraints", constraints).Error
}

func (r *ModelParameterRepo) ConstraintsByIDs(ctx context.Context, ids []int64) (map[int64]string, error) {
	result := make(map[int64]string, len(ids))
	if len(ids) == 0 {
		return result, nil
	}
	var rows []struct {
		ID          int64
		Constraints string
	}
	if err := dbFrom(ctx).Table(model.TableNameVideoModelParameter).
		Select("id, constraints").Where("id IN ?", ids).Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		result[row.ID] = row.Constraints
	}
	return result, nil
}

func (r *ModelParameterRepo) SoftDeleteByModel(ctx context.Context, modelID int64) error {
	q := qFrom(ctx).VideoModelParameter
	items, err := q.WithContext(ctx).Where(q.ModelID.Eq(modelID)).Find()
	if err != nil {
		return err
	}
	if len(items) == 0 {
		return nil
	}
	_, err = q.WithContext(ctx).Delete(items...)
	return err
}

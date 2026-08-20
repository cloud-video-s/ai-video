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

// ListByModels returns all parameters for the requested models in a single
// query. Rows are grouped deterministically by model, parameter type, and
// display order.
func (r *ModelParameterRepo) ListByModels(ctx context.Context, modelIDs []int64) ([]model.VideoModelParameter, error) {
	if len(modelIDs) == 0 {
		return []model.VideoModelParameter{}, nil
	}
	q := qFrom(ctx).VideoModelParameter
	rows, err := q.WithContext(ctx).
		Where(q.ModelID.In(modelIDs...)).
		Order(q.ModelID.Asc(), q.ParameterType.Asc(), q.SortOrder.Asc(), q.ID.Asc()).
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

func (r *ModelParameterRepo) GetApiPageTask(ctx context.Context, userID uint64, page, pageSize int, taskType, status uint32) ([]*model.VideoUserGenerationTask, int64, error) {
	q := qFrom(ctx).VideoUserGenerationTask
	dao := q.WithContext(ctx).Where(q.UserID.Eq(userID))
	if taskType > 0 && taskType < 3 {
		dao = dao.Where(q.TaskType.Eq(taskType))
	}
	if status > 0 {
		dao = dao.Where(q.Status.Eq(int(status)))
	}
	total, err := dao.Count()
	if err != nil {
		return nil, 0, err
	}
	rows, err := dao.Order(q.CreatedAt.Desc(), q.ID.Desc()).Offset((page - 1) * pageSize).Limit(pageSize).Find()
	return rows, total, err
}

func (r *ModelParameterRepo) UpdateFields(ctx context.Context, item *model.VideoModelParameter) error {
	return dbFrom(ctx).Model(&model.VideoModelParameter{}).
		Where("id = ? AND model_id = ?", item.ID, item.ModelID).
		Updates(map[string]interface{}{
			"param_key":             item.ParamKey,
			"param_type":            item.ParamType,
			"is_required":           item.IsRequired,
			"default_value":         item.DefaultValue,
			"allowed_values":        item.AllowedValues,
			"allowed_value_aliases": item.AllowedValueOptions,
			"description":           item.Description,
			"sort_order":            item.SortOrder,
			"parameter_type":        item.ParameterType,
			"constraints":           item.Constraints,
			"alias":                 item.Alias_,
			"display_type":          item.DisplayType,
			"is_display":            item.IsDisplay,
		}).Error
}

func (r *ModelParameterRepo) UpdateConstraints(ctx context.Context, id int64, constraints string) error {
	q := qFrom(ctx).VideoModelParameter
	_, err := q.WithContext(ctx).Where(q.ID.Eq(id)).Update(q.Constraints, constraints)
	return err
}

func (r *ModelParameterRepo) ConstraintsByIDs(ctx context.Context, ids []int64) (map[int64]string, error) {
	result := make(map[int64]string, len(ids))
	if len(ids) == 0 {
		return result, nil
	}
	q := qFrom(ctx).VideoModelParameter
	rows, err := q.WithContext(ctx).Select(q.ID, q.Constraints).Where(q.ID.In(ids...)).Find()
	if err != nil {
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

package repository

import (
	"context"

	"ai-video/internal/gen/model"
)

// TemplateModelParameterRepo stores the model parameter configuration that is
// specific to a template. All rows for one template must use the same model_id
// as video_template.model_id; the service layer validates that invariant before
// calling ReplaceForTemplate.
type TemplateModelParameterRepo struct {
	BaseRepo[model.VideoTemplateModelParameter]
}

func NewTemplateModelParameterRepo() *TemplateModelParameterRepo {
	return &TemplateModelParameterRepo{}
}

func (r *TemplateModelParameterRepo) ListByTemplate(ctx context.Context, templateID uint64) ([]model.VideoTemplateModelParameter, error) {
	q := qFrom(ctx).VideoTemplateModelParameter
	rows, err := q.WithContext(ctx).
		Where(q.TemplateID.Eq(templateID)).
		Order(q.ParameterType.Asc(), q.SortOrder.Asc(), q.ID.Asc()).
		Find()
	return valuesOf(rows), err
}

func (r *TemplateModelParameterRepo) ListByTemplates(ctx context.Context, templateIDs []uint64) ([]model.VideoTemplateModelParameter, error) {
	if len(templateIDs) == 0 {
		return []model.VideoTemplateModelParameter{}, nil
	}
	q := qFrom(ctx).VideoTemplateModelParameter
	rows, err := q.WithContext(ctx).
		Where(q.TemplateID.In(templateIDs...)).
		Order(q.TemplateID.Asc(), q.ParameterType.Asc(), q.SortOrder.Asc(), q.ID.Asc()).
		Find()
	return valuesOf(rows), err
}

func (r *TemplateModelParameterRepo) ListByModelAndKey(ctx context.Context, modelID int64, paramKey string) ([]model.VideoTemplateModelParameter, error) {
	q := qFrom(ctx).VideoTemplateModelParameter
	rows, err := q.WithContext(ctx).
		Where(q.ModelID.Eq(modelID), q.ParamKey.Eq(paramKey)).
		Order(q.TemplateID.Asc(), q.ID.Asc()).
		Find()
	return valuesOf(rows), err
}

// ReplaceForTemplate soft-deletes the previous configuration and inserts the
// submitted snapshot. Call it inside repository.Transaction together with the
// template create/update so callers never observe mismatched model IDs.
func (r *TemplateModelParameterRepo) ReplaceForTemplate(ctx context.Context, templateID uint64, items []*model.VideoTemplateModelParameter) error {
	q := qFrom(ctx).VideoTemplateModelParameter
	if _, err := q.WithContext(ctx).Where(q.TemplateID.Eq(templateID)).Delete(); err != nil {
		return err
	}
	if len(items) == 0 {
		return nil
	}
	for i := range items {
		items[i].ID = 0
		items[i].TemplateID = templateID
	}
	return q.WithContext(ctx).CreateInBatches(items, 100)
}

func (r *TemplateModelParameterRepo) SoftDeleteByTemplate(ctx context.Context, templateID uint64) error {
	q := qFrom(ctx).VideoTemplateModelParameter
	_, err := q.WithContext(ctx).Where(q.TemplateID.Eq(templateID)).Delete()
	return err
}

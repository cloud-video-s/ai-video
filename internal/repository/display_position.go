package repository

import (
	"context"

	"ai-video/internal/model"
)

type DisplayPositionRepo struct {
	BaseRepo[model.VideoDisplayPosition]
}

func NewDisplayPositionRepo() *DisplayPositionRepo {
	return &DisplayPositionRepo{}
}

type DisplayPositionListFilter struct {
	Status  *int8
	Keyword string
}

func (r *DisplayPositionRepo) PageList(ctx context.Context, page, pageSize int, filter *DisplayPositionListFilter) ([]model.VideoDisplayPosition, int64, error) {
	q := &QueryOptions{Where: map[string]interface{}{}, Order: []string{"sort ASC", "id DESC"}}
	if filter != nil {
		if filter.Status != nil {
			q.Where["status"] = *filter.Status
		}
		if filter.Keyword != "" {
			keyword := "%" + filter.Keyword + "%"
			q.Conds = append(q.Conds, Cond{
				Query: "position_name LIKE ? OR position_key LIKE ? OR description LIKE ?",
				Args:  []interface{}{keyword, keyword, keyword},
			})
		}
	}
	return r.BaseRepo.PageList(ctx, page, pageSize, q)
}

func (r *DisplayPositionRepo) ListOptions(ctx context.Context) ([]model.VideoDisplayPosition, error) {
	return r.BaseRepo.List(ctx, &QueryOptions{
		Where: map[string]interface{}{"status": int8(1)},
		Order: []string{"sort ASC", "id ASC"},
	})
}

func (r *DisplayPositionRepo) GetByKey(ctx context.Context, key string) (*model.VideoDisplayPosition, error) {
	return r.BaseRepo.GetOne(ctx, &QueryOptions{Where: map[string]interface{}{"position_key": key}})
}

func (r *DisplayPositionRepo) UpdateFields(ctx context.Context, item *model.VideoDisplayPosition) error {
	return r.BaseRepo.Update(ctx, item,
		"PositionName", "PositionKey", "Description", "CoverImage", "Sort", "Status",
	)
}

func (r *DisplayPositionRepo) BannerCount(ctx context.Context, positionKey string) (int64, error) {
	var count int64
	err := dbFrom(ctx).Table("video_banner_display_position").
		Where("position_key = ?", positionKey).Count(&count).Error
	return count, err
}

func (r *DisplayPositionRepo) TemplateTypeCount(ctx context.Context, positionKey string) (int64, error) {
	var count int64
	err := dbFrom(ctx).Table("video_template_type_display_position").
		Where("position_key = ?", positionKey).Count(&count).Error
	return count, err
}

func (r *DisplayPositionRepo) RenameTemplateTypePositionKey(ctx context.Context, oldKey, newKey string) error {
	if oldKey == newKey {
		return nil
	}
	db := dbFrom(ctx)
	if err := db.Model(&model.VideoTemplateTypeDisplayPosition{}).
		Where("position_key = ?", oldKey).Update("position_key", newKey).Error; err != nil {
		return err
	}
	return db.Model(&model.VideoBannerDisplayPosition{}).
		Where("position_key = ?", oldKey).Update("position_key", newKey).Error
}

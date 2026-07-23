package repository

import (
	"context"
	"strings"

	"ai-video/internal/gen/model"
)

type VideoAppRepo struct{ BaseRepo[model.VideoApp] }

func NewVideoAppRepo() *VideoAppRepo { return &VideoAppRepo{} }

type VideoAppListFilter struct {
	Keyword string
	AppCode string
	Status  *uint32
}

func (r *VideoAppRepo) PageList(ctx context.Context, page, pageSize int, filter *VideoAppListFilter) ([]model.VideoApp, int64, error) {
	options := &QueryOptions{Order: []string{"sort ASC", "id DESC"}}
	if filter != nil {
		options.Where = make(map[string]interface{})
		if filter.AppCode != "" {
			options.Where["app_code"] = filter.AppCode
		}
		if filter.Status != nil {
			options.Where["status"] = *filter.Status
		}
		if keyword := strings.TrimSpace(filter.Keyword); keyword != "" {
			value := "%" + keyword + "%"
			options.Conds = append(options.Conds, Cond{
				Query: "name LIKE ? OR app_code LIKE ? OR description LIKE ?",
				Args:  []interface{}{value, value, value},
			})
		}
	}
	return r.BaseRepo.PageList(ctx, page, pageSize, options)
}

func (r *VideoAppRepo) GetByAppCode(ctx context.Context, appCode string) (*model.VideoApp, error) {
	return r.BaseRepo.GetOne(ctx, &QueryOptions{Where: map[string]interface{}{"app_code": appCode}})
}

func (r *VideoAppRepo) ListOptions(ctx context.Context) ([]model.VideoApp, error) {
	return r.BaseRepo.List(ctx, &QueryOptions{Order: []string{"sort ASC", "id ASC"}})
}

func (r *VideoAppRepo) PackageCount(ctx context.Context, appCode string) (int64, error) {
	var count int64
	err := dbFrom(ctx).Model(&model.VideoPackage{}).Where("app_code = ?", appCode).Count(&count).Error
	return count, err
}

func (r *VideoAppRepo) UpdateFields(ctx context.Context, app *model.VideoApp) error {
	return r.BaseRepo.Update(ctx, app, "Name", "AppCode", "Status", "Sort", "Description")
}

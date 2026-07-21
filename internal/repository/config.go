package repository

import (
	"context"

	"ai-video/internal/gen/model"
)

type ConfigRepo struct{}

func NewConfigRepo() *ConfigRepo {
	return &ConfigRepo{}
}

func (d *ConfigRepo) Create(ctx context.Context, c *model.VideoConfig) error {
	return qFrom(ctx).VideoConfig.WithContext(ctx).UnderlyingDB().Create(c).Error
}

func (d *ConfigRepo) GetByID(ctx context.Context, id uint) (*model.VideoConfig, error) {
	var c model.VideoConfig
	q := qFrom(ctx).VideoConfig
	err := q.WithContext(ctx).Where(q.ID.Eq(uint64(id))).UnderlyingDB().First(&c).Error
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// GetByKey fetches a config by its unique key.
func (d *ConfigRepo) GetByKey(ctx context.Context, key string) (*model.VideoConfig, error) {
	var c model.VideoConfig
	q := qFrom(ctx).VideoConfig
	err := q.WithContext(ctx).Where(q.Key.Eq(key)).UnderlyingDB().First(&c).Error
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (d *ConfigRepo) Exists(ctx context.Context, opts *QueryOptions) (bool, error) {
	q := qFrom(ctx).VideoConfig
	dao := q.WithContext(ctx)
	if opts != nil {
		if key, ok := opts.Where["key"].(string); ok {
			dao = dao.Where(q.Key.Eq(key))
		}
		if group, ok := opts.Where["group"].(string); ok {
			dao = dao.Where(q.Group.Eq(group))
		}
	}
	total, err := dao.Count()
	return total > 0, err
}

func (d *ConfigRepo) List(ctx context.Context, opts *QueryOptions) ([]model.VideoConfig, error) {
	q := qFrom(ctx).VideoConfig
	dao := q.WithContext(ctx)
	if opts != nil {
		if group, ok := opts.Where["group"].(string); ok {
			dao = dao.Where(q.Group.Eq(group))
		}
		if isPublic, ok := opts.Where["is_public"].(bool); ok {
			dao = dao.Where(q.IsPublic.Is(isPublic))
		}
	}
	var list []model.VideoConfig
	if err := dao.Order(q.Sort.Asc(), q.ID.Asc()).Scan(&list); err != nil {
		return nil, err
	}
	return list, nil
}

// ListAll returns every config ordered for stable display.
func (d *ConfigRepo) ListAll(ctx context.Context) ([]model.VideoConfig, error) {
	return d.List(ctx, &QueryOptions{Order: []string{"sort ASC", "id ASC"}})
}

// ListPublic returns only configs flagged is_public.
func (d *ConfigRepo) ListPublic(ctx context.Context) ([]model.VideoConfig, error) {
	return d.List(ctx, &QueryOptions{
		Where: map[string]interface{}{"is_public": true},
		Order: []string{"sort ASC", "id ASC"},
	})
}

func (d *ConfigRepo) Update(ctx context.Context, c *model.VideoConfig, fields ...string) error {
	q := qFrom(ctx).VideoConfig
	db := q.WithContext(ctx).Where(q.ID.Eq(uint64(c.ID))).UnderlyingDB().Model(&model.VideoConfig{})
	if len(fields) > 0 {
		db = db.Select(fields)
	}
	return db.Updates(c).Error
}

func (d *ConfigRepo) HardDelete(ctx context.Context, id uint) error {
	q := qFrom(ctx).VideoConfig
	return q.WithContext(ctx).Where(q.ID.Eq(uint64(id))).UnderlyingDB().Unscoped().Delete(&model.VideoConfig{}).Error
}

// UpdateValue updates just the value column for key.
func (d *ConfigRepo) UpdateValue(ctx context.Context, key, value string) error {
	q := qFrom(ctx).VideoConfig
	_, err := q.WithContext(ctx).Where(q.Key.Eq(key)).Update(q.Value, value)
	return err
}

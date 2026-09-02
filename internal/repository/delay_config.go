package repository

import (
	"ai-video/internal/gen/model"
	"context"
	"fmt"
	"strconv"
	"strings"
)

type DelayConfigRepo struct{}

func NewDelayConfigRepo() *DelayConfigRepo {
	return &DelayConfigRepo{}
}

type DelayConfigListFilter struct {
	ListSort ListSort
	Group    string
	Keyword  string
}

func (d *DelayConfigRepo) Create(ctx context.Context, config *model.VideoDelayConfig) error {
	return qFrom(ctx).VideoDelayConfig.WithContext(ctx).UnderlyingDB().Create(config).Error
}

func (d *DelayConfigRepo) GetByID(ctx context.Context, id uint) (*model.VideoDelayConfig, error) {
	var config model.VideoDelayConfig
	q := qFrom(ctx).VideoDelayConfig
	if err := q.WithContext(ctx).Where(q.ID.Eq(uint64(id))).UnderlyingDB().First(&config).Error; err != nil {
		return nil, err
	}
	return &config, nil
}

func (d *DelayConfigRepo) GetByKey(ctx context.Context, key string) (*model.VideoDelayConfig, error) {
	var config model.VideoDelayConfig
	q := qFrom(ctx).VideoDelayConfig
	if err := q.WithContext(ctx).Where(q.Key.Eq(key)).UnderlyingDB().First(&config).Error; err != nil {
		return nil, err
	}
	return &config, nil
}

func (d *DelayConfigRepo) ExistsByKey(ctx context.Context, key string) (bool, error) {
	q := qFrom(ctx).VideoDelayConfig
	count, err := q.WithContext(ctx).Where(q.Key.Eq(key)).Count()
	return count > 0, err
}

func (d *DelayConfigRepo) Update(ctx context.Context, config *model.VideoDelayConfig) error {
	q := qFrom(ctx).VideoDelayConfig
	return q.WithContext(ctx).Where(q.ID.Eq(config.ID)).UnderlyingDB().
		Model(&model.VideoDelayConfig{}).
		Select("Group", "Value", "Type", "Options", "Remark", "Sort").
		Updates(config).Error
}

func (d *DelayConfigRepo) UpdateValue(ctx context.Context, key, value string) error {
	q := qFrom(ctx).VideoDelayConfig
	_, err := q.WithContext(ctx).Where(q.Key.Eq(key)).Update(q.Value, value)
	return err
}

func (d *DelayConfigRepo) HardDelete(ctx context.Context, id uint) error {
	q := qFrom(ctx).VideoDelayConfig
	return q.WithContext(ctx).Where(q.ID.Eq(uint64(id))).UnderlyingDB().
		Unscoped().Delete(&model.VideoDelayConfig{}).Error
}

func (d *DelayConfigRepo) PageList(ctx context.Context, page, pageSize int, filter *DelayConfigListFilter) ([]model.VideoDelayConfig, int64, error) {
	q := qFrom(ctx).VideoDelayConfig
	dao := q.WithContext(ctx)
	if filter != nil && filter.Group != "" {
		dao = dao.Where(q.Group.Eq(filter.Group))
	}
	db := dao.UnderlyingDB().Model(&model.VideoDelayConfig{})
	if filter != nil && filter.Keyword != "" {
		keyword := "%" + filter.Keyword + "%"
		db = db.Where("`key` LIKE ? OR remark LIKE ?", keyword, keyword)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.VideoDelayConfig
	listSort := ListSort{}
	if filter != nil {
		listSort = filter.ListSort
	}
	order := orderSQLForList(listSort, map[string]string{"sort": "sort"}, "id", "sort ASC, id ASC")
	if err := db.Order(order).Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (d *DelayConfigRepo) ListAll(ctx context.Context) ([]model.VideoDelayConfig, error) {
	q := qFrom(ctx).VideoDelayConfig
	var list []model.VideoDelayConfig
	err := q.WithContext(ctx).
		Order(q.Sort.Asc(), q.ID.Asc()).
		UnderlyingDB().
		Find(&list).Error
	return list, err
}

// ListValues returns the client-facing configuration as a key-value object.
// Numeric and boolean database values are normalized to JSON numbers.
func (d *DelayConfigRepo) ListValues(ctx context.Context) (map[string]interface{}, error) {
	q := qFrom(ctx).VideoDelayConfig
	type configValue struct {
		Key   string
		Value string
		Type  string
	}
	var list []configValue
	if err := q.WithContext(ctx).
		Select(q.Key, q.Value, q.Type).
		Order(q.Sort.Asc(), q.ID.Asc()).
		Scan(&list); err != nil {
		return nil, err
	}
	result := make(map[string]interface{}, len(list))
	for _, item := range list {
		value, err := parseDelayConfigNumber(item.Type, item.Value)
		if err != nil {
			return nil, fmt.Errorf("delay config %s: %w", item.Key, err)
		}
		result[item.Key] = value
	}
	return result, nil
}

func parseDelayConfigNumber(strType, value string) (any, error) {
	switch strType {
	case "string":
		return value, nil
	case "int":
		v, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
		if err != nil {
			return nil, err
		}
		return v, nil
	case "float":
		v, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return nil, err
		}
		return v, nil
	default:
		parsed, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
		if err != nil {
			return 0, fmt.Errorf("value %q is not numeric", value)
		}
		return parsed, nil
	}
}

func (d *DelayConfigRepo) ListGroups(ctx context.Context) ([]string, error) {
	q := qFrom(ctx).VideoDelayConfig
	var groups []string
	err := q.WithContext(ctx).Distinct(q.Group).Order(q.Group.Asc()).Pluck(q.Group, &groups)
	return groups, err
}

package repository

import (
	"context"
	"time"

	"ai-video/internal/gen/model"
)

type OperationLogRepo struct{}

func NewOperationLogRepo() *OperationLogRepo {
	return &OperationLogRepo{}
}

func (d *OperationLogRepo) Create(ctx context.Context, log *model.VideoOperationLog) error {
	return qFrom(ctx).VideoOperationLog.WithContext(ctx).UnderlyingDB().Create(log).Error
}

func (d *OperationLogRepo) GetByID(ctx context.Context, id uint) (*model.VideoOperationLog, error) {
	var log model.VideoOperationLog
	q := qFrom(ctx).VideoOperationLog
	err := q.WithContext(ctx).Where(q.ID.Eq(uint64(id))).UnderlyingDB().First(&log).Error
	if err != nil {
		return nil, err
	}
	return &log, nil
}

func (d *OperationLogRepo) Delete(ctx context.Context, id uint) error {
	q := qFrom(ctx).VideoOperationLog
	return q.WithContext(ctx).Where(q.ID.Eq(uint64(id))).UnderlyingDB().Delete(&model.VideoOperationLog{}).Error
}

func (d *OperationLogRepo) PageList(ctx context.Context, page, pageSize int, opts *QueryOptions) ([]model.VideoOperationLog, int64, error) {
	q := qFrom(ctx).VideoOperationLog
	db := q.WithContext(ctx).UnderlyingDB().Model(&model.VideoOperationLog{})
	if opts != nil {
		db = opts.applyFilter(db)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.VideoOperationLog
	if err := db.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// DeleteBefore hard-deletes logs created before t. Returns rows removed.
func (d *OperationLogRepo) DeleteBefore(ctx context.Context, t time.Time) (int64, error) {
	q := qFrom(ctx).VideoOperationLog
	res := q.WithContext(ctx).Where(q.CreatedAt.Lt(t)).UnderlyingDB().Unscoped().Delete(&model.VideoOperationLog{})
	return res.RowsAffected, res.Error
}

// Clear hard-deletes all operation logs.
func (d *OperationLogRepo) Clear(ctx context.Context) error {
	q := qFrom(ctx).VideoOperationLog
	return q.WithContext(ctx).UnderlyingDB().Unscoped().Where("1 = 1").Delete(&model.VideoOperationLog{}).Error
}

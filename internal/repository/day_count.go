package repository

import (
	"ai-video/internal/gen/model"
	"context"
	"time"
)

type DayCountRepo struct{}

func NewDayCountRepo() *DayCountRepo {
	return &DayCountRepo{}
}

func (d *DayCountRepo) Create(ctx context.Context, day *model.VideoDayCount) error {
	return qFrom(ctx).VideoDayCount.WithContext(ctx).UnderlyingDB().Create(day).Error
}

func (d *DayCountRepo) GetByDayTime(ctx context.Context, dayTime time.Time) (*model.VideoDayCount, error) {
	q := qFrom(ctx).VideoDayCount
	info, err := q.WithContext(ctx).Where(q.DayTime.Eq(dayTime)).First()
	if err != nil {
		return nil, err
	}
	return info, nil
}

func (d *DayCountRepo) GetByID(ctx context.Context, id uint) (*model.VideoDayCount, error) {
	q := qFrom(ctx).VideoDayCount
	info, err := q.WithContext(ctx).Where(q.ID.Eq(uint64(id))).First()
	if err != nil {
		return nil, err
	}
	return info, nil
}

func (d *DayCountRepo) Update(ctx context.Context, config *model.VideoDayCount) error {
	q := qFrom(ctx).VideoDayCount
	_, err := q.WithContext(ctx).Where(q.ID.Eq(config.ID)).Updates(config)
	if err != nil {
		return err
	}
	return nil
}

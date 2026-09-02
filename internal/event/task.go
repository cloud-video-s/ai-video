package event

import (
	"ai-video/internal/config"
	"ai-video/internal/gen/model"
	"ai-video/internal/repository"
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
)

// DayTaskPoints 更新算力成本
func DayTaskPoints(c context.Context, task *model.VideoUserGenerationTask) {
	if task.Status == 6 {
		now := time.Now()
		unlock, locked := lockDayCount(c, "task", now)
		if !locked {
			return
		}
		defer unlock()
		if task.Score > 0 {
			dayResp := repository.NewDayCountRepo()
			dayCount, err := dayResp.GetByDayTime(c, now)
			if errors.Is(err, gorm.ErrRecordNotFound) {
				dayCount = &model.VideoDayCount{}
			}
			//todo 算力成本系数待完善
			dayCount.ComputeCost = dayCount.ComputeCost + float64(task.Score)*0.01
			dayCount.EstimatedTotalCost = dayCount.EstimatedTotalCost + float64(task.Score)*0.01
			if dayCount.ID > 0 {
				err = dayResp.Create(c, dayCount)
				if err != nil {
					config.Logger(c).Error("day_count.Create", "err", err)
				}
			} else {
				err = dayResp.Update(c, dayCount)
				if err != nil {
					config.Logger(c).Error("day_count.Update", "err", err)
				}
			}
		}
	}
}

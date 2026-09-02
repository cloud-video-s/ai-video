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

// ActiveCountUser 日活和激活更新  logType 1=日活 2=激活
func ActiveCountUser(c context.Context, logType int) {
	now := time.Now()
	unlock, locked := lockDayCount(c, "user", now)
	if !locked {
		return
	}
	defer unlock()

	dayResp := repository.NewDayCountRepo()
	dayCount, err := dayResp.GetByDayTime(c, now)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		dayCount = &model.VideoDayCount{}
	}
	if logType == 1 {
		dayCount.DailyActiveUsers = dayCount.DailyActiveUsers + 1
	} else {
		dayCount.ActivationCount = dayCount.ActivationCount + 1
	}
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

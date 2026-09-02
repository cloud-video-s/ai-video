package event

import (
	"ai-video/internal/config"
	"ai-video/internal/gen/model"
	"ai-video/internal/repository"
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

var ActiveDayKey = "active_day_key_user_id_"

func ActiveLog(c context.Context, userID uint64) {
	if config.Redis.Get(c, fmt.Sprintf("%s%d", ActiveDayKey, userID)).Val() == "" {
		q := repository.QFrom(c).VideoUserActive
		err := q.WithContext(c).Create(&model.VideoUserActive{
			UserID:  userID,
			DayTime: time.Now(),
		})
		if err != nil {
			config.Logger(c).Error("create video_user_active failed", zap.Error(err))
		} else {
			config.Redis.Set(c, fmt.Sprintf("%s%d", ActiveDayKey, userID), 0, time.Hour*24)
			u := repository.QFrom(c).VideoUser
			_, err = u.WithContext(c).Where(u.ID.Eq(userID)).UpdateColumn(u.ActiveDays, gorm.Expr("active_days + 1"))
			if err != nil {
				config.Logger(c).Error("update video_user active_day failed", zap.Error(err))
			}
		}
		ActiveCountUser(c, 1)
	}
}

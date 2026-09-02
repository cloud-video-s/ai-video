package event

import (
	"ai-video/internal/config"
	"ai-video/internal/gen/model"
	"ai-video/internal/repository"
	"context"
	"time"

	"go.uber.org/zap"
)

func LoginLog(ctx context.Context, userID uint64, loginType int32, ip string) {
	q := repository.QFrom(ctx).VideoUserLoginLog
	err := q.WithContext(ctx).Create(&model.VideoUserLoginLog{
		UserID:    userID,
		LoginTime: time.Now(),
		LoginType: loginType,
		LoginIP:   ip,
	})
	if err != nil {
		config.Logger(ctx).Error("create video_user_active failed", zap.Error(err))
	}
}

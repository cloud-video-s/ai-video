package event

import (
	"ai-video/internal/config"
	"ai-video/internal/gen/model"
	"ai-video/internal/pkg/cache"
	"context"
	"fmt"
	"time"
)

const (
	eventCountLockPrefix        = "lock:video-event-count:"
	eventCountLockTTL           = time.Minute
	eventCountLockRetryInterval = 50 * time.Millisecond
)

func lockDayCount(c context.Context, key string, day time.Time) (func(), bool) {
	lockKey := fmt.Sprintf("%s_%s_%s", eventCountLockPrefix, key, day.Format("2006-01-02"))
	token, err := cache.AcquireLockWithRetry(c, lockKey, eventCountLockTTL, eventCountLockRetryInterval)
	if err != nil {
		config.Logger(c).Errorw("acquire day count lock", "error", err, "lock_key", lockKey)
		return nil, false
	}

	return func() {
		if err := cache.ReleaseLock(lockKey, token); err != nil {
			config.Logger(c).Errorw("release day count lock", "error", err, "lock_key", lockKey)
		}
	}, true
}

// EventActivate 激活事件（APP 新注册用户触发）
func EventActivate(ctx context.Context, user *model.VideoUser) {
	ActiveLog(ctx, user.ID)
	ActiveCountUser(ctx, 2)
}

// EventActive 活跃事件（APP登录或登录后客户端交互触发）
func EventActive(ctx context.Context, user *model.VideoUser) {
	ActiveLog(ctx, user.ID)
}

// EventLogin 登录事件（登录就触发）
func EventLogin(ctx context.Context, user *model.VideoUser) {
	LoginLog(ctx, user.ID, int32(user.LoginType), user.LastLoginIP)
}

// EventOrder 订单事件（订单完成和退款触发）
func EventOrder(ctx context.Context, order *model.VideoOrder) {
	OrderDayCount(ctx, order)
}

// EventTask 任务事件（完成任务后触发）
func EventTask(ctx context.Context, task *model.VideoUserGenerationTask) {
	DayTaskPoints(ctx, task)
}

// EventAttribution 用户归因事件（归因成功触发）
func EventAttribution(ctx context.Context, attribution *model.VideoUserAttribution) {

}

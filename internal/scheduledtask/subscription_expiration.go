package scheduledtask

import (
	"context"
	"fmt"
	"time"

	"ai-video/internal/pkg/task"
	"ai-video/internal/repository"
)

const TypeExpireUserSubscriptions = "user:subscription:expire"

type subscriptionExpirationRepository interface {
	ExpireDueSubscriptions(ctx context.Context, now time.Time) (int64, error)
}

type subscriptionExpirationHandler struct {
	users       subscriptionExpirationRepository
	now         func() time.Time
	onCompleted func(expiredUsers int64)
}

func (h *subscriptionExpirationHandler) Handle(ctx context.Context, _ []byte) error {
	expiredUsers, err := h.users.ExpireDueSubscriptions(ctx, h.now())
	if err != nil {
		return fmt.Errorf("expire due user subscriptions: %w", err)
	}
	if h.onCompleted != nil {
		h.onCompleted(expiredUsers)
	}
	return nil
}

// RegisterSubscriptionExpiration wires the handler and its recurring schedule
// into the shared task manager. The update itself is idempotent; Unique also
// suppresses duplicate enqueue attempts when multiple app instances overlap.
func RegisterSubscriptionExpiration(manager *task.Manager, cronExpr string, onCompleted func(expiredUsers int64)) error {
	handler := &subscriptionExpirationHandler{
		users:       repository.NewAppUserRepo(),
		now:         time.Now,
		onCompleted: onCompleted,
	}
	manager.Worker.Handle(TypeExpireUserSubscriptions, handler.Handle)

	_, err := manager.Scheduler.Register(task.CronTask{
		Cron:     cronExpr,
		TypeName: TypeExpireUserSubscriptions,
		Payload:  struct{}{},
		Queue:    "default",
		Unique:   50 * time.Second,
	})
	return err
}

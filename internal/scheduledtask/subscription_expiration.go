package scheduledtask

import (
	"context"
	"errors"
	"fmt"
	"time"

	"ai-video/internal/pkg/task"
	"ai-video/internal/repository"

	"github.com/hibiken/asynq"
)

const TypeExpireUserSubscriptions = "user:subscription:expire"

type subscriptionExpirationRepository interface {
	ExpireDueSubscriptions(ctx context.Context, now time.Time) (int64, error)
}

type subscriptionExpirationHandler struct {
	users       subscriptionExpirationRepository
	now         func() time.Time
	onCompleted func(ctx context.Context, expiredUsers int64)
}

func (h *subscriptionExpirationHandler) Handle(ctx context.Context, _ []byte) error {
	expiredUsers, err := h.users.ExpireDueSubscriptions(ctx, h.now())
	if err != nil {
		return fmt.Errorf("expire due user subscriptions: %w", err)
	}
	if h.onCompleted != nil {
		h.onCompleted(ctx, expiredUsers)
	}
	return nil
}

// RegisterSubscriptionExpiration wires the handler into the shared task
// manager and enqueues one execution at executeAt. A deterministic task ID
// prevents duplicate enqueues when multiple app instances start before the
// execution time. Past and zero times are intentionally not enqueued.
func RegisterSubscriptionExpiration(manager *task.Manager, executeAt time.Time, onCompleted func(context.Context, int64)) error {
	handler := &subscriptionExpirationHandler{
		users:       repository.NewAppUserRepo(),
		now:         time.Now,
		onCompleted: onCompleted,
	}
	manager.Worker.Handle(TypeExpireUserSubscriptions, handler.Handle)

	if executeAt.IsZero() || !executeAt.After(time.Now()) {
		return nil
	}
	_, err := manager.Client.EnqueueAt(
		TypeExpireUserSubscriptions,
		struct{}{},
		executeAt,
		asynq.Queue("default"),
		asynq.TaskID(subscriptionExpirationTaskID(executeAt)),
	)
	if errors.Is(err, asynq.ErrTaskIDConflict) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("enqueue subscription expiration task at %s: %w", executeAt.Format(time.RFC3339), err)
	}
	return nil
}

func subscriptionExpirationTaskID(executeAt time.Time) string {
	return fmt.Sprintf("subscription-expiration-%d", executeAt.UTC().UnixNano())
}

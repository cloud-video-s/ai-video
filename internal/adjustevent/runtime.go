package adjustevent

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"ai-video/internal/config"
	"ai-video/internal/pkg/adjust"
	"ai-video/internal/pkg/task"

	"github.com/hibiken/asynq"
	"gorm.io/gorm"
)

const (
	taskTypeAdjustEvent = "adjust:event"
	adjustEventQueue    = "default"
	maxRetries          = 5
	completedRetention  = 24 * time.Hour
)

type RuntimeConfig struct {
	Client      *task.Client
	Worker      *task.Worker
	AuthToken   string
	BaseURL     string
	Environment adjust.Environment
}

// Runtime publishes business triggers to Redis and asynchronously processes
// both rule evaluation and the final Adjust S2S report through the shared task
// worker.
type Runtime struct {
	client  taskEnqueuer
	store   eventDataStore
	pending runtimePendingRepository
	process *processor
	errors  chan error
}

type taskEnqueuer interface {
	EnqueueContext(context.Context, string, interface{}, ...asynq.Option) (*asynq.TaskInfo, error)
}

type runtimePendingRepository interface {
	Save(context.Context, Message) error
	ListByUser(context.Context, uint64, int) ([]Message, error)
	ListUnqueued(context.Context, int) ([]Message, error)
	MarkRequeued(context.Context, string, time.Time) error
}

func NewRuntime(runtimeConfig RuntimeConfig) (*Runtime, error) {
	if runtimeConfig.Client == nil {
		return nil, errors.New("Adjust event Redis queue client is unavailable")
	}
	if runtimeConfig.Worker == nil {
		return nil, errors.New("Adjust event Redis queue worker is unavailable")
	}
	if runtimeConfig.Environment != "" &&
		runtimeConfig.Environment != adjust.EnvironmentProduction &&
		runtimeConfig.Environment != adjust.EnvironmentSandbox {
		return nil, fmt.Errorf("unsupported Adjust event environment %q", runtimeConfig.Environment)
	}
	store := newGORMEventStore()
	runtime := &Runtime{
		client: runtimeConfig.Client, store: store, pending: pendingRepository{},
		errors: make(chan error, 1),
	}
	runtime.process = &processor{
		store: store, publisher: runtime,
		reporter: adjustReporter{
			authToken: strings.TrimSpace(runtimeConfig.AuthToken), baseURL: strings.TrimSpace(runtimeConfig.BaseURL),
			environment: runtimeConfig.Environment,
		},
	}
	task.HandleJSON(runtimeConfig.Worker, taskTypeAdjustEvent, runtime.processMessage)
	return runtime, nil
}

// Start launches pending-event recovery. Message consumption is handled by
// the shared Redis task worker after all handlers have been registered.
func (runtime *Runtime) Start(ctx context.Context) error {
	if ctx == nil {
		return errors.New("Adjust event runtime context is required")
	}
	go runtime.recoverPending(ctx)
	return nil
}

// Errors reports pending-event recovery incidents for monitoring. Redis
// consumer and handler errors are reported by the shared task worker.
func (runtime *Runtime) Errors() <-chan error { return runtime.errors }

func (runtime *Runtime) Enqueue(ctx context.Context, userID uint64, action adjust.EventToken, options EnqueueOptions) error {
	if userID == 0 {
		return errors.New("Adjust event user ID is required")
	}
	if _, ok := triggerName(action); !ok {
		return fmt.Errorf("unsupported Adjust trigger action %q", action)
	}
	user, err := runtime.store.GetUser(ctx, userID)
	if err != nil {
		return err
	}
	channelID, err := runtime.store.ResolveChannel(ctx, user.ChannelID)
	if err != nil {
		return err
	}
	message := Message{
		Kind: messageKindTrigger, EventID: triggerEventID(userID, action, options.OrderNo),
		UserID: userID, Action: action, ChannelID: channelID, OrderNo: strings.TrimSpace(options.OrderNo),
		OrderCount: user.OrderCount, OccurredAt: options.OccurredAt,
	}
	if channelID == 0 {
		if err := runtime.pending.Save(ctx, message); err != nil {
			return fmt.Errorf("persist Adjust event awaiting attributed channel: %w", err)
		}
		config.Logger(ctx).Infow("Adjust event is waiting for attributed channel", "event_id", message.EventID,
			"user_id", message.UserID, "action", message.Action)
		return nil
	}

	message.normalize()
	if err = runtime.PublishMessage(ctx, message); err == nil {
		return nil
	} else if pendingErr := runtime.pending.Save(ctx, message); pendingErr != nil {
		return errors.Join(err, fmt.Errorf("persist Adjust event after Redis enqueue failure: %w", pendingErr))
	}
	return nil
}

func (runtime *Runtime) PublishMessage(ctx context.Context, message Message) error {
	if ctx == nil {
		return errors.New("Adjust event publish context is required")
	}
	if runtime.client == nil {
		return errors.New("Adjust event Redis queue client is unavailable")
	}
	message.normalize()
	if err := message.validate(); err != nil {
		return err
	}
	options := []asynq.Option{
		asynq.Queue(adjustEventQueue),
		asynq.MaxRetry(maxRetries),
		asynq.TaskID(message.EventID),
		asynq.Retention(completedRetention),
	}
	if message.NotBefore.After(time.Now()) {
		options = append(options, asynq.ProcessAt(message.NotBefore))
	}
	if _, err := runtime.client.EnqueueContext(ctx, taskTypeAdjustEvent, message, options...); err != nil {
		// A pending-event replay can race with the periodic recovery loop. The
		// stable event ID makes an already queued task equivalent to success.
		if errors.Is(err, asynq.ErrTaskIDConflict) {
			return nil
		}
		return fmt.Errorf("enqueue Adjust event %s in Redis: %w", message.EventID, err)
	}
	return nil
}

func (runtime *Runtime) processMessage(ctx context.Context, message Message) error {
	if err := runtime.process.Process(ctx, message); err != nil {
		return fmt.Errorf(
			"process Adjust event event_id=%s user_id=%d action=%s: %w",
			message.EventID,
			message.UserID,
			message.Action,
			err,
		)
	}
	return nil
}

func (runtime *Runtime) ReplayPending(ctx context.Context, userID uint64) error {
	attribution, err := runtime.store.GetUserAttribution(ctx, userID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if attribution.ChannelID == 0 {
		return nil
	}
	messages, err := runtime.pending.ListByUser(ctx, userID, 500)
	if err != nil {
		return err
	}
	return runtime.replayMessages(ctx, messages, attribution.ChannelID)
}

func (runtime *Runtime) replayMessages(ctx context.Context, messages []Message, attributedChannelID uint64) error {
	for _, message := range messages {
		if message.ChannelID == 0 {
			message.ChannelID = attributedChannelID
		}
		if err := runtime.PublishMessage(ctx, message); err != nil {
			return err
		}
		if err := runtime.pending.MarkRequeued(ctx, message.EventID, time.Now()); err != nil {
			return err
		}
	}
	return nil
}

func (runtime *Runtime) recoverPending(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			messages, err := runtime.pending.ListUnqueued(ctx, 200)
			if err != nil {
				runtime.reportRuntimeError(ctx, fmt.Errorf("scan pending Adjust events: %w", err))
				continue
			}
			for _, message := range messages {
				attribution, attributionErr := runtime.store.GetUserAttribution(ctx, message.UserID)
				if errors.Is(attributionErr, gorm.ErrRecordNotFound) {
					continue
				} else if attributionErr != nil {
					runtime.reportRuntimeError(ctx, fmt.Errorf(
						"check pending Adjust event %s attribution: %w",
						message.EventID,
						attributionErr,
					))
					continue
				}
				if attribution.ChannelID == 0 {
					continue
				}
				if err = runtime.replayMessages(ctx, []Message{message}, attribution.ChannelID); err != nil {
					runtime.reportRuntimeError(ctx, fmt.Errorf("replay pending Adjust event %s: %w", message.EventID, err))
				}
			}
		}
	}
}

func (runtime *Runtime) reportRuntimeError(ctx context.Context, err error) {
	select {
	case runtime.errors <- err:
	default:
		config.Logger(ctx).Errorw("Adjust event runtime error", "error", err)
	}
}

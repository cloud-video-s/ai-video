package adjustevent

import (
	"context"
	"testing"
	"time"

	"ai-video/internal/gen/model"
	"ai-video/internal/pkg/adjust"

	"github.com/hibiken/asynq"
	"gorm.io/gorm"
)

type runtimeTestStore struct {
	user              *model.VideoUser
	attribution       *model.VideoUserAttribution
	attributionErr    error
	resolvedChannelID uint64
	saved             []Message
	deleted           []string
}

func (store *runtimeTestStore) GetUser(context.Context, uint64) (*model.VideoUser, error) {
	return store.user, nil
}

func (store *runtimeTestStore) GetUserAttribution(context.Context, uint64) (*model.VideoUserAttribution, error) {
	if store.attributionErr != nil {
		return nil, store.attributionErr
	}
	if store.attribution == nil {
		return nil, gorm.ErrRecordNotFound
	}
	return store.attribution, nil
}

func (*runtimeTestStore) GetAdjustAttribution(context.Context, string) (*model.VideoAdjustAttribution, error) {
	return nil, gorm.ErrRecordNotFound
}

func (*runtimeTestStore) GetChannel(context.Context, uint64) (*model.VideoChannel, error) {
	return nil, gorm.ErrRecordNotFound
}

func (*runtimeTestStore) GetOrder(context.Context, string) (*model.VideoOrder, error) {
	return nil, gorm.ErrRecordNotFound
}

func (store *runtimeTestStore) ResolveChannel(context.Context, string) (uint64, error) {
	return store.resolvedChannelID, nil
}

func (store *runtimeTestStore) SavePending(_ context.Context, message Message) error {
	store.saved = append(store.saved, message)
	return nil
}

func (store *runtimeTestStore) DeletePending(_ context.Context, eventID string) error {
	store.deleted = append(store.deleted, eventID)
	return nil
}

type runtimeTestPending struct {
	saved           []Message
	messages        []Message
	listByUserCalls int
}

func (pending *runtimeTestPending) Save(_ context.Context, message Message) error {
	pending.saved = append(pending.saved, message)
	return nil
}

func (pending *runtimeTestPending) ListByUser(context.Context, uint64, int) ([]Message, error) {
	pending.listByUserCalls++
	return pending.messages, nil
}

func (pending *runtimeTestPending) ListUnqueued(context.Context, int) ([]Message, error) {
	return pending.messages, nil
}

func (*runtimeTestPending) MarkRequeued(context.Context, string, time.Time) error { return nil }

type runtimeTestPublisher struct {
	messages []Message
}

func (publisher *runtimeTestPublisher) PublishMessage(_ context.Context, message Message) error {
	publisher.messages = append(publisher.messages, message)
	return nil
}

type runtimeTestTaskEnqueuer struct {
	ctx      context.Context
	typeName string
	payload  interface{}
	options  []asynq.Option
	err      error
}

func (enqueuer *runtimeTestTaskEnqueuer) EnqueueContext(
	ctx context.Context,
	typeName string,
	payload interface{},
	options ...asynq.Option,
) (*asynq.TaskInfo, error) {
	enqueuer.ctx = ctx
	enqueuer.typeName = typeName
	enqueuer.payload = payload
	enqueuer.options = options
	return &asynq.TaskInfo{ID: "redis-task"}, enqueuer.err
}

func TestEnqueueWithoutChannelPersistsPendingInsteadOfPublishing(t *testing.T) {
	store := &runtimeTestStore{user: &model.VideoUser{ID: 42}}
	pending := &runtimeTestPending{}
	runtime := &Runtime{store: store, pending: pending}

	err := runtime.Enqueue(context.Background(), 42, adjust.EventTokenActivation, EnqueueOptions{
		OccurredAt: time.Date(2026, 8, 21, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(pending.saved) != 1 {
		t.Fatalf("pending events = %d, want 1", len(pending.saved))
	}
	if pending.saved[0].ChannelID != 0 || pending.saved[0].UserID != 42 {
		t.Fatalf("pending event = %#v", pending.saved[0])
	}
}

func TestReplayPendingWaitsUntilAttributionHasChannel(t *testing.T) {
	store := &runtimeTestStore{attribution: &model.VideoUserAttribution{UserID: 42, ChannelID: 0}}
	pending := &runtimeTestPending{messages: []Message{{
		Kind: messageKindTrigger, EventID: "pending-event", UserID: 42,
		Action: adjust.EventTokenActivation, OccurredAt: time.Now(),
	}}}
	runtime := &Runtime{store: store, pending: pending}

	if err := runtime.ReplayPending(context.Background(), 42); err != nil {
		t.Fatal(err)
	}
	if pending.listByUserCalls != 0 {
		t.Fatalf("pending events were read before attribution had a channel")
	}
}

func TestProcessorKeepsZeroChannelEventPending(t *testing.T) {
	store := &runtimeTestStore{attribution: &model.VideoUserAttribution{UserID: 42, ChannelID: 0}}
	publisher := &runtimeTestPublisher{}
	processor := &processor{store: store, publisher: publisher}
	message := Message{
		Kind: messageKindTrigger, EventID: "pending-event", UserID: 42,
		Action: adjust.EventTokenActivation, OccurredAt: time.Now(),
	}

	if err := processor.Process(context.Background(), message); err != nil {
		t.Fatal(err)
	}
	if len(store.saved) != 1 {
		t.Fatalf("saved pending events = %d, want 1", len(store.saved))
	}
	if len(store.deleted) != 0 || len(publisher.messages) != 0 {
		t.Fatalf("zero-channel event was deleted or published: deleted=%v published=%v", store.deleted, publisher.messages)
	}
}

func TestPublishMessageEnqueuesDelayedRedisTask(t *testing.T) {
	enqueuer := &runtimeTestTaskEnqueuer{}
	runtime := &Runtime{client: enqueuer}
	ctx := context.Background()
	processAt := time.Now().Add(time.Hour).Truncate(time.Second)
	message := Message{
		Kind: messageKindTrigger, EventID: "event-id", UserID: 42,
		Action: adjust.EventTokenActivation, OccurredAt: time.Now(), NotBefore: processAt,
	}

	if err := runtime.PublishMessage(ctx, message); err != nil {
		t.Fatal(err)
	}
	if enqueuer.ctx != ctx || enqueuer.typeName != taskTypeAdjustEvent {
		t.Fatalf("enqueued context/type = %v/%q", enqueuer.ctx, enqueuer.typeName)
	}
	gotMessage, ok := enqueuer.payload.(Message)
	if !ok || gotMessage.EventID != message.EventID {
		t.Fatalf("enqueued payload = %#v", enqueuer.payload)
	}
	wantOptions := map[asynq.OptionType]interface{}{
		asynq.QueueOpt:     adjustEventQueue,
		asynq.MaxRetryOpt:  maxRetries,
		asynq.TaskIDOpt:    message.EventID,
		asynq.RetentionOpt: completedRetention,
		asynq.ProcessAtOpt: processAt,
	}
	for _, option := range enqueuer.options {
		if want, exists := wantOptions[option.Type()]; exists {
			if got := option.Value(); got != want {
				t.Fatalf("option %v = %#v, want %#v", option.Type(), got, want)
			}
			delete(wantOptions, option.Type())
		}
	}
	if len(wantOptions) != 0 {
		t.Fatalf("missing Redis task options: %#v", wantOptions)
	}
}

func TestPublishMessageTreatsTaskIDConflictAsAlreadyQueued(t *testing.T) {
	runtime := &Runtime{client: &runtimeTestTaskEnqueuer{err: asynq.ErrTaskIDConflict}}
	message := Message{
		Kind: messageKindTrigger, EventID: "event-id", UserID: 42,
		Action: adjust.EventTokenActivation, OccurredAt: time.Now(),
	}

	if err := runtime.PublishMessage(context.Background(), message); err != nil {
		t.Fatalf("duplicate task should be treated as queued: %v", err)
	}
}

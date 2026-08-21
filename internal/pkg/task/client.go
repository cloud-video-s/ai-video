package task

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
)

// Client wraps asynq.Client as the task producer.
type Client struct {
	client taskEnqueuer
}

type taskEnqueuer interface {
	EnqueueContext(context.Context, *asynq.Task, ...asynq.Option) (*asynq.TaskInfo, error)
	Close() error
}

func NewClient(redisAddr, username, password string, db int) *Client {
	client := asynq.NewClient(asynq.RedisClientOpt{
		Addr:     redisAddr,
		Username: username,
		Password: password,
		DB:       db,
	})
	return &Client{client: client}
}

func (c *Client) Close() error {
	return c.client.Close()
}

// Enqueue sends a task to the default queue for immediate processing.
func (c *Client) Enqueue(typeName string, payload interface{}, opts ...asynq.Option) (*asynq.TaskInfo, error) {
	return c.EnqueueContext(context.Background(), typeName, payload, opts...)
}

// EnqueueContext sends a task immediately and lets the caller cancel the
// Redis operation through ctx.
func (c *Client) EnqueueContext(ctx context.Context, typeName string, payload interface{}, opts ...asynq.Option) (*asynq.TaskInfo, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal task payload: %w", err)
	}
	task := asynq.NewTask(typeName, data)
	info, err := c.client.EnqueueContext(ctx, task, opts...)
	if err != nil {
		return nil, fmt.Errorf("enqueue task %q: %w", typeName, err)
	}
	return info, nil
}

// EnqueueDelay sends a task to be processed after the given delay.
func (c *Client) EnqueueDelay(typeName string, payload interface{}, delay time.Duration, opts ...asynq.Option) (*asynq.TaskInfo, error) {
	return c.EnqueueDelayContext(context.Background(), typeName, payload, delay, opts...)
}

func (c *Client) EnqueueDelayContext(ctx context.Context, typeName string, payload interface{}, delay time.Duration, opts ...asynq.Option) (*asynq.TaskInfo, error) {
	opts = appendTaskOption(opts, asynq.ProcessIn(delay))
	return c.EnqueueContext(ctx, typeName, payload, opts...)
}

// EnqueueAt sends a task to be processed at the given time.
func (c *Client) EnqueueAt(typeName string, payload interface{}, at time.Time, opts ...asynq.Option) (*asynq.TaskInfo, error) {
	return c.EnqueueAtContext(context.Background(), typeName, payload, at, opts...)
}

func (c *Client) EnqueueAtContext(ctx context.Context, typeName string, payload interface{}, at time.Time, opts ...asynq.Option) (*asynq.TaskInfo, error) {
	opts = appendTaskOption(opts, asynq.ProcessAt(at))
	return c.EnqueueContext(ctx, typeName, payload, opts...)
}

// EnqueueUnique sends a task with deduplication within the given TTL.
func (c *Client) EnqueueUnique(typeName string, payload interface{}, uniqueTTL time.Duration, opts ...asynq.Option) (*asynq.TaskInfo, error) {
	return c.EnqueueUniqueContext(context.Background(), typeName, payload, uniqueTTL, opts...)
}

func (c *Client) EnqueueUniqueContext(ctx context.Context, typeName string, payload interface{}, uniqueTTL time.Duration, opts ...asynq.Option) (*asynq.TaskInfo, error) {
	opts = appendTaskOption(opts, asynq.Unique(uniqueTTL))
	return c.EnqueueContext(ctx, typeName, payload, opts...)
}

// EnqueueToQueue sends a task to a specific named queue.
func (c *Client) EnqueueToQueue(typeName string, payload interface{}, queue string, opts ...asynq.Option) (*asynq.TaskInfo, error) {
	return c.EnqueueToQueueContext(context.Background(), typeName, payload, queue, opts...)
}

func (c *Client) EnqueueToQueueContext(ctx context.Context, typeName string, payload interface{}, queue string, opts ...asynq.Option) (*asynq.TaskInfo, error) {
	opts = appendTaskOption(opts, asynq.Queue(queue))
	return c.EnqueueContext(ctx, typeName, payload, opts...)
}

func appendTaskOption(options []asynq.Option, option asynq.Option) []asynq.Option {
	result := make([]asynq.Option, 0, len(options)+1)
	result = append(result, options...)
	return append(result, option)
}

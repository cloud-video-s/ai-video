package task

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"ai-video/internal/pkg/tracing"

	"github.com/hibiken/asynq"
)

// HandlerFunc is the function signature for task handlers.
type HandlerFunc func(ctx context.Context, payload []byte) error

// Worker wraps asynq.Server as the task consumer.
type Worker struct {
	server   workerServer
	mux      *asynq.ServeMux
	handlers map[string]HandlerFunc
	done     chan struct{}
	stopOnce sync.Once
}

type workerServer interface {
	Start(handler asynq.Handler) error
	Stop()
	Shutdown()
}

func NewWorker(redisAddr, password string, db int, concurrency int, queues map[string]int, errHandler func(context.Context, string, error)) *Worker {
	if concurrency <= 0 {
		concurrency = 10
	}
	if len(queues) == 0 {
		queues = map[string]int{"default": 1}
	}
	if errHandler == nil {
		errHandler = func(_ context.Context, taskType string, err error) {
			fmt.Printf("[task error] type=%s err=%v\n", taskType, err)
		}
	}

	srv := asynq.NewServer(
		asynq.RedisClientOpt{
			Addr:     redisAddr,
			Password: password,
			DB:       db,
		},
		asynq.Config{
			Concurrency: concurrency,
			Queues:      queues,
			ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, task *asynq.Task, err error) {
				var tracedErr *tracedTaskError
				if errors.As(err, &tracedErr) {
					ctx = tracing.ContextWithSpan(ctx, tracedErr.span)
					err = tracedErr.err
				}
				errHandler(ctx, task.Type(), err)
			}),
		},
	)

	return &Worker{
		server:   srv,
		mux:      asynq.NewServeMux(),
		handlers: make(map[string]HandlerFunc),
		done:     make(chan struct{}),
	}
}

// Handle registers a handler for the given task type.
func (w *Worker) Handle(typeName string, handler HandlerFunc) {
	w.handlers[typeName] = handler
	w.mux.HandleFunc(typeName, func(ctx context.Context, t *asynq.Task) error {
		ctx, span := tracing.NewContext(ctx)
		if err := handler(ctx, t.Payload()); err != nil {
			return &tracedTaskError{err: err, span: span}
		}
		return nil
	})
}

type tracedTaskError struct {
	err  error
	span tracing.SpanContext
}

func (e *tracedTaskError) Error() string { return e.err.Error() }
func (e *tracedTaskError) Unwrap() error { return e.err }

// Start starts the worker (blocking).
func (w *Worker) Start() error {
	// asynq.Server.Start is intentionally non-blocking. Keep this application
	// wrapper alive until Stop finishes; Server.Run is not used because it
	// installs its own OS signal handler while admin-server owns shutdown.
	if err := w.server.Start(w.mux); err != nil {
		return err
	}
	<-w.done
	return nil
}

// Stop gracefully shuts down the worker.
func (w *Worker) Stop() {
	w.stopOnce.Do(func() {
		w.server.Stop()
		w.server.Shutdown()
		close(w.done)
	})
}

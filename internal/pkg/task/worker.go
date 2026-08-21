package task

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"ai-video/internal/pkg/tracing"

	"github.com/hibiken/asynq"
)

const (
	defaultRestartDelay    = time.Second
	defaultRestartMaxDelay = 30 * time.Second
)

var errWorkerStopped = errors.New("task worker stopped")

// HandlerFunc processes one raw task payload. Returning an error asks Asynq
// to retry the message according to its retry options.
type HandlerFunc func(ctx context.Context, payload []byte) error

// JSONHandlerFunc processes a payload decoded from JSON.
type JSONHandlerFunc[T any] func(ctx context.Context, payload T) error

// HandlerPanicError is returned to Asynq when a consumer handler panics. Asynq
// treats it like any other handler failure, so the message is retried and the
// consumer process remains alive.
type HandlerPanicError struct {
	Value any
	Stack []byte
}

func (e *HandlerPanicError) Error() string {
	return fmt.Sprintf("task handler panic: %v", e.Value)
}

// RestartHandler observes a failed worker start before the supervisor retries.
type RestartHandler func(err error, nextDelay time.Duration)

type workerOptions struct {
	restartDelay    time.Duration
	restartMaxDelay time.Duration
	restartHandler  RestartHandler
}

// WorkerOption customizes the worker supervisor.
type WorkerOption func(*workerOptions)

// WithRestartBackoff configures the initial and maximum delay between worker
// reconstruction attempts. Non-positive values use safe defaults.
func WithRestartBackoff(initial, maximum time.Duration) WorkerOption {
	return func(options *workerOptions) {
		options.restartDelay = initial
		options.restartMaxDelay = maximum
	}
}

// WithRestartHandler configures worker restart reporting.
func WithRestartHandler(handler RestartHandler) WorkerOption {
	return func(options *workerOptions) {
		options.restartHandler = handler
	}
}

// Worker wraps an Asynq Redis consumer and supervises its startup lifecycle.
// A failed or panicking server instance is discarded and rebuilt indefinitely
// with bounded exponential backoff until Stop is called.
type Worker struct {
	serverFactory func() workerServer
	mux           *asynq.ServeMux
	options       workerOptions
	done          chan struct{}
	stopOnce      sync.Once
	startMu       sync.Mutex
	serverMu      sync.Mutex
	server        workerServer
	stopped       atomic.Bool
}

type workerServer interface {
	Start(handler asynq.Handler) error
	Stop()
	Shutdown()
}

func NewWorker(
	redisAddr, username, password string,
	db, concurrency int,
	queues map[string]int,
	errHandler func(context.Context, string, error),
	workerOpts ...WorkerOption,
) *Worker {
	if concurrency <= 0 {
		concurrency = 10
	}
	if len(queues) == 0 {
		queues = map[string]int{"default": 1}
	} else {
		queues = cloneQueues(queues)
	}
	if errHandler == nil {
		errHandler = func(_ context.Context, taskType string, err error) {
			fmt.Printf("[task error] type=%s err=%v\n", taskType, err)
		}
	}

	options := workerOptions{
		restartDelay:    defaultRestartDelay,
		restartMaxDelay: defaultRestartMaxDelay,
		restartHandler: func(err error, nextDelay time.Duration) {
			fmt.Printf("[task worker restart] err=%v retry_in=%s\n", err, nextDelay)
		},
	}
	for _, apply := range workerOpts {
		if apply != nil {
			apply(&options)
		}
	}
	normalizeWorkerOptions(&options)

	redisOptions := asynq.RedisClientOpt{
		Addr:     redisAddr,
		Username: username,
		Password: password,
		DB:       db,
	}
	serverConfig := asynq.Config{
		Concurrency: concurrency,
		Queues:      queues,
		ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, message *asynq.Task, err error) {
			var tracedErr *tracedTaskError
			if errors.As(err, &tracedErr) {
				ctx = tracing.ContextWithSpan(ctx, tracedErr.span)
				err = tracedErr.err
			}
			errHandler(ctx, message.Type(), err)
		}),
	}

	return &Worker{
		serverFactory: func() workerServer {
			return asynq.NewServer(redisOptions, serverConfig)
		},
		mux:     asynq.NewServeMux(),
		options: options,
		done:    make(chan struct{}),
	}
}

// Handle registers a raw payload handler for a task type. Register all
// handlers before Start; duplicate task types are rejected by Asynq.
func (w *Worker) Handle(typeName string, handler HandlerFunc) {
	if handler == nil {
		panic("task: nil handler")
	}
	w.mux.HandleFunc(typeName, func(ctx context.Context, message *asynq.Task) (result error) {
		ctx, span := tracing.NewContext(ctx)
		defer func() {
			if recovered := recover(); recovered != nil {
				result = &tracedTaskError{
					err:  &HandlerPanicError{Value: recovered, Stack: debug.Stack()},
					span: span,
				}
			}
		}()
		if err := handler(ctx, message.Payload()); err != nil {
			return &tracedTaskError{err: err, span: span}
		}
		return nil
	})
}

// HandleJSON registers a typed JSON consumer while keeping serialization
// details out of business handlers.
func HandleJSON[T any](worker *Worker, typeName string, handler JSONHandlerFunc[T]) {
	if worker == nil {
		panic("task: nil worker")
	}
	if handler == nil {
		panic("task: nil JSON handler")
	}
	worker.Handle(typeName, func(ctx context.Context, payload []byte) error {
		var decoded T
		if err := json.Unmarshal(payload, &decoded); err != nil {
			return fmt.Errorf("decode task %q payload: %w", typeName, err)
		}
		return handler(ctx, decoded)
	})
}

type tracedTaskError struct {
	err  error
	span tracing.SpanContext
}

func (e *tracedTaskError) Error() string { return e.err.Error() }
func (e *tracedTaskError) Unwrap() error { return e.err }

// Start supervises the Redis consumer and blocks until Stop is called. Worker
// bootstrap errors and panics are reported and restarted automatically.
func (w *Worker) Start() error {
	w.startMu.Lock()
	defer w.startMu.Unlock()

	delay := w.options.restartDelay
	for {
		if w.stopped.Load() {
			return nil
		}
		err := w.startServer()
		if err == nil {
			<-w.done
			return nil
		}
		if errors.Is(err, errWorkerStopped) || w.stopped.Load() {
			return nil
		}

		w.reportRestart(err, delay)
		if !w.waitForRestart(delay) {
			return nil
		}
		delay = nextRestartDelay(delay, w.options.restartMaxDelay)
	}
}

func (w *Worker) startServer() (err error) {
	var server workerServer
	serverMuLocked := false
	defer func() {
		if serverMuLocked {
			w.serverMu.Unlock()
		}
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("task worker start panic: %v\n%s", recovered, debug.Stack())
		}
		if err != nil && server != nil && w.clearServer(server) {
			server.Shutdown()
		}
	}()

	w.serverMu.Lock()
	serverMuLocked = true
	if w.stopped.Load() {
		w.serverMu.Unlock()
		serverMuLocked = false
		return errWorkerStopped
	}

	// Construct and start while holding serverMu so Stop cannot observe and
	// shut down a not-yet-started Asynq server between these two operations.
	server = w.serverFactory()
	if server == nil {
		w.serverMu.Unlock()
		serverMuLocked = false
		return errors.New("task worker server factory returned nil")
	}
	w.server = server
	if err := server.Start(w.mux); err != nil {
		w.serverMu.Unlock()
		serverMuLocked = false
		return fmt.Errorf("start task worker: %w", err)
	}
	w.serverMu.Unlock()
	serverMuLocked = false
	return nil
}

func (w *Worker) clearServer(server workerServer) bool {
	w.serverMu.Lock()
	defer w.serverMu.Unlock()
	if w.server != server {
		return false
	}
	w.server = nil
	return true
}

func (w *Worker) reportRestart(err error, delay time.Duration) {
	if w.options.restartHandler == nil {
		return
	}
	defer func() {
		_ = recover()
	}()
	w.options.restartHandler(err, delay)
}

func (w *Worker) waitForRestart(delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-timer.C:
		return true
	case <-w.done:
		return false
	}
}

// Stop gracefully shuts down the active consumer and cancels pending restart
// attempts. It is safe to call more than once.
func (w *Worker) Stop() {
	w.stopOnce.Do(func() {
		w.stopped.Store(true)
		close(w.done)

		w.serverMu.Lock()
		server := w.server
		w.server = nil
		w.serverMu.Unlock()
		if server != nil {
			server.Stop()
			server.Shutdown()
		}
	})
}

func normalizeWorkerOptions(options *workerOptions) {
	if options.restartDelay <= 0 {
		options.restartDelay = defaultRestartDelay
	}
	if options.restartMaxDelay <= 0 {
		options.restartMaxDelay = defaultRestartMaxDelay
	}
	if options.restartMaxDelay < options.restartDelay {
		options.restartMaxDelay = options.restartDelay
	}
}

func nextRestartDelay(current, maximum time.Duration) time.Duration {
	if current >= maximum || current > maximum/2 {
		return maximum
	}
	return current * 2
}

func cloneQueues(queues map[string]int) map[string]int {
	result := make(map[string]int, len(queues))
	maps.Copy(result, queues)
	return result
}

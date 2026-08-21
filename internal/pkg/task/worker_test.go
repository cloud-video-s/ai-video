package task

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hibiken/asynq"
)

type fakeWorkerServer struct {
	startErr     error
	startPanic   any
	startBlock   <-chan struct{}
	started      chan struct{}
	stopped      chan struct{}
	shutdownDone chan struct{}
	startOnce    sync.Once
	stopOnce     sync.Once
	shutdownOnce sync.Once
}

func newFakeWorkerServer() *fakeWorkerServer {
	return &fakeWorkerServer{
		started:      make(chan struct{}),
		stopped:      make(chan struct{}),
		shutdownDone: make(chan struct{}),
	}
}

func (s *fakeWorkerServer) Start(asynq.Handler) error {
	s.startOnce.Do(func() { close(s.started) })
	if s.startPanic != nil {
		panic(s.startPanic)
	}
	if s.startBlock != nil {
		<-s.startBlock
	}
	return s.startErr
}

func (s *fakeWorkerServer) Stop() {
	s.stopOnce.Do(func() { close(s.stopped) })
}

func (s *fakeWorkerServer) Shutdown() {
	s.shutdownOnce.Do(func() { close(s.shutdownDone) })
}

func newTestWorker(factory func() workerServer, restartHandler RestartHandler) *Worker {
	return &Worker{
		serverFactory: factory,
		mux:           asynq.NewServeMux(),
		options: workerOptions{
			restartDelay:    time.Millisecond,
			restartMaxDelay: 5 * time.Millisecond,
			restartHandler:  restartHandler,
		},
		done: make(chan struct{}),
	}
}

func TestWorkerStartBlocksUntilStop(t *testing.T) {
	server := newFakeWorkerServer()
	worker := newTestWorker(func() workerServer { return server }, nil)
	result := make(chan error, 1)
	go func() {
		result <- worker.Start()
	}()

	waitClosed(t, server.started, "worker start")
	select {
	case err := <-result:
		t.Fatalf("Start returned before Stop: %v", err)
	default:
	}

	worker.Stop()
	select {
	case err := <-result:
		if err != nil {
			t.Fatalf("Start returned error after Stop: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Start did not return after Stop")
	}
	waitClosed(t, server.stopped, "worker stop")
	waitClosed(t, server.shutdownDone, "worker shutdown")

	// Stop is safe for repeated shutdown paths.
	worker.Stop()
}

func TestWorkerRestartsAfterStartupError(t *testing.T) {
	want := errors.New("worker bootstrap failed")
	failed := newFakeWorkerServer()
	failed.startErr = want
	recovered := newFakeWorkerServer()
	servers := []workerServer{failed, recovered}
	next := 0
	restarts := make(chan error, 1)
	worker := newTestWorker(func() workerServer {
		server := servers[next]
		next++
		return server
	}, func(err error, _ time.Duration) {
		restarts <- err
	})

	result := make(chan error, 1)
	go func() { result <- worker.Start() }()

	select {
	case err := <-restarts:
		if !errors.Is(err, want) {
			t.Fatalf("restart error = %v, want wrapped %v", err, want)
		}
	case <-time.After(time.Second):
		t.Fatal("startup error was not reported")
	}
	waitClosed(t, failed.shutdownDone, "failed worker cleanup")
	waitClosed(t, recovered.started, "replacement worker start")

	worker.Stop()
	if err := <-result; err != nil {
		t.Fatalf("Start returned error after recovery: %v", err)
	}
}

func TestWorkerStopDuringStartupShutsDownStartedServer(t *testing.T) {
	allowStart := make(chan struct{})
	server := newFakeWorkerServer()
	server.startBlock = allowStart
	worker := newTestWorker(func() workerServer { return server }, nil)
	startResult := make(chan error, 1)
	go func() { startResult <- worker.Start() }()
	waitClosed(t, server.started, "worker startup")

	stopDone := make(chan struct{})
	go func() {
		worker.Stop()
		close(stopDone)
	}()
	select {
	case <-stopDone:
		t.Fatal("Stop returned before the starting server became active")
	case <-time.After(10 * time.Millisecond):
	}

	close(allowStart)
	waitClosed(t, stopDone, "concurrent worker stop")
	waitClosed(t, server.stopped, "concurrently started worker stop")
	waitClosed(t, server.shutdownDone, "concurrently started worker shutdown")
	select {
	case err := <-startResult:
		if err != nil {
			t.Fatalf("Start returned error: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Start did not return after concurrent Stop")
	}
}

func TestWorkerRestartsAfterStartupPanic(t *testing.T) {
	failed := newFakeWorkerServer()
	failed.startPanic = "boom"
	recovered := newFakeWorkerServer()
	servers := []workerServer{failed, recovered}
	next := 0
	restarts := make(chan error, 1)
	worker := newTestWorker(func() workerServer {
		server := servers[next]
		next++
		return server
	}, func(err error, _ time.Duration) {
		restarts <- err
	})

	result := make(chan error, 1)
	go func() { result <- worker.Start() }()

	select {
	case err := <-restarts:
		if !strings.Contains(err.Error(), "task worker start panic: boom") {
			t.Fatalf("restart error = %v, want recovered panic", err)
		}
	case <-time.After(time.Second):
		t.Fatal("startup panic was not reported")
	}
	waitClosed(t, recovered.started, "replacement worker start")

	worker.Stop()
	if err := <-result; err != nil {
		t.Fatalf("Start returned error after panic recovery: %v", err)
	}
}

func TestWorkerStopCancelsRestartBackoff(t *testing.T) {
	failed := newFakeWorkerServer()
	failed.startErr = errors.New("keep failing")
	reported := make(chan struct{}, 1)
	worker := newTestWorker(func() workerServer { return failed }, func(error, time.Duration) {
		reported <- struct{}{}
	})
	worker.options.restartDelay = time.Minute
	worker.options.restartMaxDelay = time.Minute

	result := make(chan error, 1)
	go func() { result <- worker.Start() }()
	select {
	case <-reported:
	case <-time.After(time.Second):
		t.Fatal("startup error was not reported")
	}

	worker.Stop()
	select {
	case err := <-result:
		if err != nil {
			t.Fatalf("Start returned error after Stop: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Stop did not cancel restart backoff")
	}
}

func TestWorkerConvertsHandlerPanicToRetryableError(t *testing.T) {
	worker := newTestWorker(nil, nil)
	worker.Handle("test:panic", func(context.Context, []byte) error {
		panic("handler exploded")
	})

	err := worker.mux.ProcessTask(context.Background(), asynq.NewTask("test:panic", nil))
	var panicErr *HandlerPanicError
	if !errors.As(err, &panicErr) {
		t.Fatalf("handler error = %T %v, want HandlerPanicError", err, err)
	}
	if panicErr.Value != "handler exploded" || len(panicErr.Stack) == 0 {
		t.Fatalf("panic error = %#v, want value and stack", panicErr)
	}
}

func TestHandleJSONDecodesPayload(t *testing.T) {
	type payload struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	worker := newTestWorker(nil, nil)
	received := payload{}
	HandleJSON(worker, "test:json", func(_ context.Context, message payload) error {
		received = message
		return nil
	})

	err := worker.mux.ProcessTask(
		context.Background(),
		asynq.NewTask("test:json", []byte(`{"id":7,"name":"video"}`)),
	)
	if err != nil {
		t.Fatalf("process JSON task: %v", err)
	}
	if received.ID != 7 || received.Name != "video" {
		t.Fatalf("decoded payload = %#v", received)
	}
}

func TestNextRestartDelayIsBounded(t *testing.T) {
	if got := nextRestartDelay(time.Second, 30*time.Second); got != 2*time.Second {
		t.Fatalf("next delay = %s, want 2s", got)
	}
	if got := nextRestartDelay(20*time.Second, 30*time.Second); got != 30*time.Second {
		t.Fatalf("bounded delay = %s, want 30s", got)
	}
}

func waitClosed(t *testing.T, channel <-chan struct{}, operation string) {
	t.Helper()
	select {
	case <-channel:
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for %s", operation)
	}
}

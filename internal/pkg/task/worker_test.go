package task

import (
	"errors"
	"testing"
	"time"

	"github.com/hibiken/asynq"
)

type fakeWorkerServer struct {
	startErr     error
	started      chan struct{}
	stopped      chan struct{}
	shutdownDone chan struct{}
}

func newFakeWorkerServer() *fakeWorkerServer {
	return &fakeWorkerServer{
		started:      make(chan struct{}),
		stopped:      make(chan struct{}),
		shutdownDone: make(chan struct{}),
	}
}

func (s *fakeWorkerServer) Start(asynq.Handler) error {
	close(s.started)
	return s.startErr
}

func (s *fakeWorkerServer) Stop() {
	close(s.stopped)
}

func (s *fakeWorkerServer) Shutdown() {
	close(s.shutdownDone)
}

func TestWorkerStartBlocksUntilStop(t *testing.T) {
	server := newFakeWorkerServer()
	worker := &Worker{
		server:   server,
		mux:      asynq.NewServeMux(),
		handlers: make(map[string]HandlerFunc),
		done:     make(chan struct{}),
	}
	result := make(chan error, 1)
	go func() {
		result <- worker.Start()
	}()

	<-server.started
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

	// Stop is safe for repeated shutdown paths.
	worker.Stop()
}

func TestWorkerStartReturnsStartupError(t *testing.T) {
	want := errors.New("redis unavailable")
	server := newFakeWorkerServer()
	server.startErr = want
	worker := &Worker{
		server:   server,
		mux:      asynq.NewServeMux(),
		handlers: make(map[string]HandlerFunc),
		done:     make(chan struct{}),
	}

	if err := worker.Start(); !errors.Is(err, want) {
		t.Fatalf("Start error = %v, want %v", err, want)
	}
}

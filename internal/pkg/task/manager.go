package task

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"
)

// ManagerConfig holds the configuration needed to create a Manager.
type ManagerConfig struct {
	RedisAddr       string
	RedisUsername   string
	RedisPassword   string
	RedisDB         int
	Concurrency     int
	Queues          []string
	RestartDelay    time.Duration
	RestartMaxDelay time.Duration
	// ErrorHandler is invoked whenever a handler returns an error or panics.
	// If nil, errors are printed to stdout.
	ErrorHandler func(ctx context.Context, taskType string, err error)
	// RestartHandler observes worker bootstrap errors and panics before the
	// worker automatically creates a new Redis consumer instance.
	RestartHandler RestartHandler
}

// Manager owns the producer, worker and recurring scheduler lifecycle.
type Manager struct {
	Client *Client
	Worker *Worker

	scheduler     *scheduler
	periodicTypes map[string]struct{}
	closeOnce     sync.Once
}

func NewManager(cfg ManagerConfig) *Manager {
	queues := make(map[string]int)
	if len(cfg.Queues) == 0 {
		cfg.Queues = []string{"default"}
	}
	for i, q := range cfg.Queues {
		queues[q] = len(cfg.Queues) - i
	}
	workerOptions := []WorkerOption{
		WithRestartBackoff(cfg.RestartDelay, cfg.RestartMaxDelay),
	}
	if cfg.RestartHandler != nil {
		workerOptions = append(workerOptions, WithRestartHandler(cfg.RestartHandler))
	}

	return &Manager{
		Client: NewClient(cfg.RedisAddr, cfg.RedisUsername, cfg.RedisPassword, cfg.RedisDB),
		Worker: NewWorker(
			cfg.RedisAddr,
			cfg.RedisUsername,
			cfg.RedisPassword,
			cfg.RedisDB,
			cfg.Concurrency,
			queues,
			cfg.ErrorHandler,
			workerOptions...,
		),
		scheduler:     newScheduler(cfg.RedisAddr, cfg.RedisUsername, cfg.RedisPassword, cfg.RedisDB),
		periodicTypes: make(map[string]struct{}),
	}
}

// RegisterPeriodic maps stable task type names to their execution method and
// interval. It must be called before Start. Queueing, cron conversion and
// cross-instance enqueue deduplication are handled internally.
func (m *Manager) RegisterPeriodic(tasks PeriodicTasks) error {
	if m == nil || m.Worker == nil || m.scheduler == nil {
		return errors.New("task manager is not initialized")
	}

	typeNames := make([]string, 0, len(tasks))
	for typeName, periodicTask := range tasks {
		if strings.TrimSpace(typeName) == "" {
			return errors.New("periodic task type is required")
		}
		if periodicTask.Every <= 0 {
			return fmt.Errorf("periodic task %q interval must be positive", typeName)
		}
		if periodicTask.Run == nil {
			return fmt.Errorf("periodic task %q handler is required", typeName)
		}
		if _, exists := m.periodicTypes[typeName]; exists {
			return fmt.Errorf("periodic task %q is already registered", typeName)
		}
		typeNames = append(typeNames, typeName)
	}
	slices.Sort(typeNames)

	for _, typeName := range typeNames {
		periodicTask := tasks[typeName]
		if err := m.scheduler.register(typeName, periodicTask.Every); err != nil {
			return err
		}
		m.Worker.Handle(typeName, func(ctx context.Context, _ []byte) error {
			return periodicTask.Run(ctx)
		})
		m.periodicTypes[typeName] = struct{}{}
	}
	return nil
}

// Start starts recurring enqueues and then blocks while the worker runs.
func (m *Manager) Start() error {
	if m == nil || m.Worker == nil || m.scheduler == nil {
		return errors.New("task manager is not initialized")
	}
	if err := m.scheduler.Start(); err != nil {
		return fmt.Errorf("start task scheduler: %w", err)
	}
	defer m.scheduler.Stop()
	return m.Worker.Start()
}

func (m *Manager) Close() {
	if m == nil {
		return
	}
	m.closeOnce.Do(func() {
		m.scheduler.Stop()
		m.Worker.Stop()
		_ = m.Client.Close()
	})
}

package task

import (
	"context"
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

// Manager holds the task producer and worker instances.
type Manager struct {
	Client *Client
	Worker *Worker
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
	}
}

func (m *Manager) Close() {
	if m.Worker != nil {
		m.Worker.Stop()
	}
	if m.Client != nil {
		_ = m.Client.Close()
	}
}

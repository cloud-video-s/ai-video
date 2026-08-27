package task

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/hibiken/asynq"
)

const defaultPeriodicUniqueTTL = 2 * time.Hour

// PeriodicTask is the complete public definition of a recurring task.
// Infrastructure details such as cron syntax, queue selection, payloads and
// deduplication are intentionally kept inside Manager.
type PeriodicTask struct {
	Every time.Duration
	Run   func(context.Context) error
}

type PeriodicTasks map[string]PeriodicTask

type scheduler struct {
	scheduler *asynq.Scheduler

	mu      sync.Mutex
	entries int
	started bool
	active  bool
	stopped bool
}

func newScheduler(redisAddr, username, password string, db int) *scheduler {
	s := asynq.NewScheduler(
		asynq.RedisClientOpt{
			Addr:     redisAddr,
			Username: username,
			Password: password,
			DB:       db,
		},
		&asynq.SchedulerOpts{Location: time.Local},
	)
	return &scheduler{scheduler: s}
}

func (s *scheduler) register(typeName string, every time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.started || s.stopped {
		return fmt.Errorf("register periodic task %q after task manager start", typeName)
	}

	uniqueTTL := max(every, defaultPeriodicUniqueTTL)
	_, err := s.scheduler.Register(
		periodicSpec(every),
		asynq.NewTask(typeName, nil),
		asynq.Queue("default"),
		asynq.Unique(uniqueTTL),
	)
	if err != nil {
		return fmt.Errorf("register periodic task %q: %w", typeName, err)
	}
	s.entries++
	return nil
}

// periodicSpec keeps common minute/hour/day intervals aligned to wall-clock
// boundaries. This lets parallel app instances enqueue at the same instant so
// Asynq's uniqueness lock can collapse duplicate schedules.
func periodicSpec(every time.Duration) string {
	if every%time.Minute != 0 {
		return "@every " + every.String()
	}
	minutes := int(every / time.Minute)
	if minutes < 60 && 60%minutes == 0 {
		return fmt.Sprintf("*/%d * * * *", minutes)
	}
	if minutes%60 == 0 {
		hours := minutes / 60
		if hours < 24 && 24%hours == 0 {
			return fmt.Sprintf("0 */%d * * *", hours)
		}
		if hours == 24 {
			return "0 0 * * *"
		}
	}
	return "@every " + every.String()
}

// Start starts the Asynq scheduler. Asynq starts its own goroutines, so this
// call is intentionally non-blocking. An empty scheduler is not started.
func (s *scheduler) Start() error {
	if s == nil || s.scheduler == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stopped {
		return nil
	}
	s.started = true
	if s.entries == 0 {
		return nil
	}
	if err := s.scheduler.Start(); err != nil {
		return err
	}
	s.active = true
	return nil
}

// Stop shuts down the scheduler.
func (s *scheduler) Stop() {
	if s == nil || s.scheduler == nil {
		return
	}
	s.mu.Lock()
	if s.stopped {
		s.mu.Unlock()
		return
	}
	s.stopped = true
	active := s.active
	s.mu.Unlock()
	if active {
		s.scheduler.Shutdown()
	}
}

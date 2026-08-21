package adjustevent

import (
	"context"
	"errors"
	"sync"

	"ai-video/internal/pkg/adjust"
)

var ErrRuntimeUnavailable = errors.New("Adjust event runtime is unavailable")

var defaultRuntime struct {
	sync.RWMutex
	runtime *Runtime
}

// SetDefault installs the process-wide Adjust event runtime used by business
// services. Passing nil disables event publication, which keeps isolated unit
// tests independent from Redis. Enqueue returns ErrRuntimeUnavailable while
// no runtime is installed so production misconfiguration is never silent.
func SetDefault(runtime *Runtime) {
	defaultRuntime.Lock()
	defaultRuntime.runtime = runtime
	defaultRuntime.Unlock()
}

func currentRuntime() *Runtime {
	defaultRuntime.RLock()
	runtime := defaultRuntime.runtime
	defaultRuntime.RUnlock()
	return runtime
}

// Enqueue snapshots the user's current channel and order count and publishes a
// durable trigger message. A queue failure is persisted in the pending-event
// table by Runtime so analytics availability never rolls back a user action.
func Enqueue(ctx context.Context, userID uint64, action adjust.EventToken, options EnqueueOptions) error {
	runtime := currentRuntime()
	if runtime == nil {
		return ErrRuntimeUnavailable
	}
	return runtime.Enqueue(ctx, userID, action, options)
}

// ReplayPending republishes unqueued events after user attribution has become
// available. It is called after Adjust fusion commits successfully.
func ReplayPending(ctx context.Context, userID uint64) error {
	runtime := currentRuntime()
	if runtime == nil {
		return nil
	}
	return runtime.ReplayPending(ctx, userID)
}

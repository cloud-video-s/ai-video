package config

import (
	"testing"
	"time"
)

func TestTaskWorkerRestartBackoff(t *testing.T) {
	tests := []struct {
		name        string
		config      TaskConfig
		wantInitial time.Duration
		wantMax     time.Duration
	}{
		{
			name:        "defaults",
			wantInitial: time.Second,
			wantMax:     30 * time.Second,
		},
		{
			name:        "configured",
			config:      TaskConfig{WorkerRestartDelaySeconds: 2, WorkerRestartMaxDelaySeconds: 20},
			wantInitial: 2 * time.Second,
			wantMax:     20 * time.Second,
		},
		{
			name:        "default maximum",
			config:      TaskConfig{WorkerRestartDelaySeconds: 10},
			wantInitial: 10 * time.Second,
			wantMax:     30 * time.Second,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			initial, maximum := test.config.WorkerRestartBackoff()
			if initial != test.wantInitial || maximum != test.wantMax {
				t.Fatalf("WorkerRestartBackoff() = %s, %s; want %s, %s", initial, maximum, test.wantInitial, test.wantMax)
			}
		})
	}
}

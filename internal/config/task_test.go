package config

import (
	"testing"
	"time"
)

func TestSubscriptionExpirationTime(t *testing.T) {
	shanghai, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatalf("load location: %v", err)
	}

	tests := []struct {
		name    string
		value   string
		enabled bool
		want    time.Time
		wantErr bool
	}{
		{name: "disabled", value: "", enabled: false},
		{
			name:    "local time",
			value:   "2026-08-14 09:30:00",
			enabled: true,
			want:    time.Date(2026, 8, 14, 9, 30, 0, 0, shanghai),
		},
		{
			name:    "RFC3339",
			value:   "2026-08-14T01:30:00Z",
			enabled: true,
			want:    time.Date(2026, 8, 14, 1, 30, 0, 0, time.UTC),
		},
		{name: "invalid", value: "tomorrow", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, enabled, err := (TaskConfig{SubscriptionExpirationAt: tt.value}).SubscriptionExpirationTime(shanghai)
			if (err != nil) != tt.wantErr {
				t.Fatalf("SubscriptionExpirationTime() error = %v, wantErr %v", err, tt.wantErr)
			}
			if enabled != tt.enabled {
				t.Fatalf("SubscriptionExpirationTime() enabled = %v, want %v", enabled, tt.enabled)
			}
			if !got.Equal(tt.want) {
				t.Fatalf("SubscriptionExpirationTime() = %v, want %v", got, tt.want)
			}
		})
	}
}

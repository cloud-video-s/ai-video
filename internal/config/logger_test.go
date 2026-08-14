package config

import (
	"context"
	"testing"

	"ai-video/internal/pkg/tracing"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestLoggerAddsTraceFields(t *testing.T) {
	core, observed := observer.New(zap.InfoLevel)
	previous := Log
	Log = zap.New(core).Sugar()
	t.Cleanup(func() { Log = previous })

	ctx := tracing.ContextWithSpan(context.Background(), tracing.SpanContext{
		TraceID:    "4bf92f3577b34da6a3ce929d0e0e4736",
		SpanID:     "00f067aa0ba902b7",
		TraceFlags: "01",
	})
	Logger(ctx).Info("test message")

	entries := observed.All()
	if len(entries) != 1 {
		t.Fatalf("log entries = %d, want 1", len(entries))
	}
	fields := entries[0].ContextMap()
	if fields["trace_id"] != "4bf92f3577b34da6a3ce929d0e0e4736" {
		t.Fatalf("trace_id = %#v", fields["trace_id"])
	}
	if fields["span_id"] != "00f067aa0ba902b7" {
		t.Fatalf("span_id = %#v", fields["span_id"])
	}
}

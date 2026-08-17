package monitor

import (
	"errors"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestReportWritesStableEventFields(t *testing.T) {
	core, observed := observer.New(zap.ErrorLevel)
	logger := zap.New(core).Sugar()

	Report(logger, KindTaskFailure, "task_worker", errors.New("redis unavailable"), "task_type", "download")

	entries := observed.All()
	if len(entries) != 1 {
		t.Fatalf("log entries = %d, want 1", len(entries))
	}
	fields := entries[0].ContextMap()
	if fields["monitor_event"] != true || fields["event_kind"] != string(KindTaskFailure) {
		t.Fatalf("monitor fields = %#v", fields)
	}
	if fields["source"] != "task_worker" || fields["error"] != "redis unavailable" || fields["task_type"] != "download" {
		t.Fatalf("event fields = %#v", fields)
	}
}

func TestReportPanicIncludesStack(t *testing.T) {
	core, observed := observer.New(zap.ErrorLevel)
	logger := zap.New(core).Sugar()

	ReportPanic(logger, "http", "boom", []byte("test stack"))

	fields := observed.All()[0].ContextMap()
	if fields["event_kind"] != string(KindPanic) || fields["stack"] != "test stack" {
		t.Fatalf("panic fields = %#v", fields)
	}
}

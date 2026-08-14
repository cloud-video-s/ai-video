package tracing

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

func TestContextFromHeadersInheritsTraceParent(t *testing.T) {
	const traceID = "4bf92f3577b34da6a3ce929d0e0e4736"
	headers := make(http.Header)
	headers.Set(HeaderTraceParent, "00-"+traceID+"-00f067aa0ba902b7-01")

	ctx, span := ContextFromHeaders(context.Background(), headers)
	if span.TraceID != traceID {
		t.Fatalf("TraceID = %q, want %q", span.TraceID, traceID)
	}
	if !validHexID(span.SpanID, 16) || span.SpanID == "00f067aa0ba902b7" {
		t.Fatalf("expected a new valid server span ID, got %q", span.SpanID)
	}
	if got, ok := SpanFromContext(ctx); !ok || got != span {
		t.Fatalf("SpanFromContext() = %#v, %v; want %#v, true", got, ok, span)
	}
}

func TestContextFromHeadersFallsBackToXTraceID(t *testing.T) {
	const traceID = "70f5e8423bd84b1d84d5ab1107487618"
	headers := make(http.Header)
	headers.Set(HeaderTraceParent, "invalid")
	headers.Set(HeaderTraceID, traceID)

	_, span := ContextFromHeaders(context.Background(), headers)
	if span.TraceID != traceID {
		t.Fatalf("TraceID = %q, want %q", span.TraceID, traceID)
	}
}

func TestContextFromHeadersRejectsInvalidClientIDs(t *testing.T) {
	headers := make(http.Header)
	headers.Set(HeaderTraceParent, "00-00000000000000000000000000000000-0000000000000000-01")
	headers.Set(HeaderTraceID, strings.Repeat("z", 32))

	_, span := ContextFromHeaders(context.Background(), headers)
	if !span.Valid() {
		t.Fatalf("generated span is invalid: %#v", span)
	}
	if span.TraceID == strings.Repeat("0", 32) || span.TraceID == strings.Repeat("z", 32) {
		t.Fatalf("invalid inbound trace ID was accepted: %q", span.TraceID)
	}
}

func TestNewRequestWithContextInjectsChildTrace(t *testing.T) {
	ctx := ContextWithSpan(context.Background(), SpanContext{
		TraceID:    "4bf92f3577b34da6a3ce929d0e0e4736",
		SpanID:     "00f067aa0ba902b7",
		TraceFlags: "01",
	})
	req, err := NewRequestWithContext(ctx, http.MethodGet, "https://example.com", nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := req.Header.Get(HeaderTraceID); got != "4bf92f3577b34da6a3ce929d0e0e4736" {
		t.Fatalf("%s = %q", HeaderTraceID, got)
	}
	parts := strings.Split(req.Header.Get(HeaderTraceParent), "-")
	if len(parts) != 4 || parts[1] != "4bf92f3577b34da6a3ce929d0e0e4736" {
		t.Fatalf("traceparent = %q", req.Header.Get(HeaderTraceParent))
	}
	if parts[2] == "00f067aa0ba902b7" || !validHexID(parts[2], 16) {
		t.Fatalf("expected a new child span ID, got %q", parts[2])
	}
}

// Package tracing provides lightweight trace correlation for application logs.
// It understands the W3C traceparent header but does not require a tracing
// backend: trace and span IDs live in context.Context and can be attached to
// logs, SQL calls, and outbound HTTP requests.
package tracing

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
)

const (
	HeaderTraceParent = "traceparent"
	HeaderTraceID     = "X-Trace-ID"

	defaultTraceFlags = "01"
)

type spanContextKey struct{}

// SpanContext is the trace identity of the current unit of work.
type SpanContext struct {
	TraceID    string
	SpanID     string
	TraceFlags string
}

// Valid reports whether all SpanContext fields can be emitted as a W3C
// traceparent value.
func (s SpanContext) Valid() bool {
	return validHexID(s.TraceID, 32) && validHexID(s.SpanID, 16) && validHex(s.TraceFlags, 2)
}

// TraceParent returns the W3C traceparent representation of the span.
func (s SpanContext) TraceParent() string {
	if !s.Valid() {
		return ""
	}
	return "00-" + strings.ToLower(s.TraceID) + "-" + strings.ToLower(s.SpanID) + "-" + strings.ToLower(s.TraceFlags)
}

// NewContext starts a new trace. It is useful for background jobs that have no
// inbound HTTP request from which to inherit a trace ID.
func NewContext(ctx context.Context) (context.Context, SpanContext) {
	if ctx == nil {
		ctx = context.Background()
	}
	span := SpanContext{
		TraceID:    randomHex(16),
		SpanID:     randomHex(8),
		TraceFlags: defaultTraceFlags,
	}
	return context.WithValue(ctx, spanContextKey{}, span), span
}

// ContextFromHeaders starts a server span, inheriting a valid W3C trace ID or
// X-Trace-ID from the inbound request. Invalid client values are ignored.
func ContextFromHeaders(ctx context.Context, headers http.Header) (context.Context, SpanContext) {
	if ctx == nil {
		ctx = context.Background()
	}
	span, ok := parseTraceParent(headers.Get(HeaderTraceParent))
	if !ok {
		traceID := strings.ToLower(strings.TrimSpace(headers.Get(HeaderTraceID)))
		if validHexID(traceID, 32) {
			span = SpanContext{TraceID: traceID, TraceFlags: defaultTraceFlags}
			ok = true
		}
	}
	if !ok {
		span.TraceID = randomHex(16)
		span.TraceFlags = defaultTraceFlags
	}
	span.SpanID = randomHex(8)
	return context.WithValue(ctx, spanContextKey{}, span), span
}

// ContextWithSpan installs a validated span in ctx. Invalid spans are ignored.
func ContextWithSpan(ctx context.Context, span SpanContext) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	span.TraceID = strings.ToLower(span.TraceID)
	span.SpanID = strings.ToLower(span.SpanID)
	span.TraceFlags = strings.ToLower(span.TraceFlags)
	if !span.Valid() {
		return ctx
	}
	return context.WithValue(ctx, spanContextKey{}, span)
}

// SpanFromContext returns the current trace identity.
func SpanFromContext(ctx context.Context) (SpanContext, bool) {
	if ctx == nil {
		return SpanContext{}, false
	}
	span, ok := ctx.Value(spanContextKey{}).(SpanContext)
	return span, ok && span.Valid()
}

// TraceIDFromContext returns the current trace ID, or an empty string when the
// context is not traced.
func TraceIDFromContext(ctx context.Context) string {
	span, _ := SpanFromContext(ctx)
	return span.TraceID
}

// NewRequestWithContext behaves like http.NewRequestWithContext and also
// injects trace correlation headers when ctx contains a valid span.
func NewRequestWithContext(ctx context.Context, method, url string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}
	InjectHTTP(req)
	return req, nil
}

// InjectHTTP propagates the current trace to an outbound HTTP request. A fresh
// child span ID is generated for each request while the trace ID is preserved.
func InjectHTTP(req *http.Request) {
	if req == nil {
		return
	}
	parent, ok := SpanFromContext(req.Context())
	if !ok {
		return
	}
	child := parent
	child.SpanID = randomHex(8)
	req.Header.Set(HeaderTraceParent, child.TraceParent())
	req.Header.Set(HeaderTraceID, child.TraceID)
}

func parseTraceParent(value string) (SpanContext, bool) {
	parts := strings.Split(strings.TrimSpace(value), "-")
	if len(parts) != 4 || strings.EqualFold(parts[0], "ff") || !validHex(parts[0], 2) {
		return SpanContext{}, false
	}
	// Version 00 must contain exactly four fields. Later versions are rejected
	// here until their extension format is explicitly supported.
	if !strings.EqualFold(parts[0], "00") {
		return SpanContext{}, false
	}
	span := SpanContext{
		TraceID:    strings.ToLower(parts[1]),
		SpanID:     strings.ToLower(parts[2]),
		TraceFlags: strings.ToLower(parts[3]),
	}
	return span, span.Valid()
}

func validHexID(value string, length int) bool {
	if !validHex(value, length) {
		return false
	}
	for _, char := range value {
		if char != '0' {
			return true
		}
	}
	return false
}

func validHex(value string, length int) bool {
	if len(value) != length {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

var fallbackIDCounter atomic.Uint64

func randomHex(byteCount int) string {
	value := make([]byte, byteCount)
	if _, err := rand.Read(value); err == nil {
		return hex.EncodeToString(value)
	}
	// Entropy failures are exceptionally rare. Keep requests observable rather
	// than failing them, while mixing time and a process-local counter to avoid
	// collisions in the fallback path.
	fallback := fmt.Sprintf("%d-%d", time.Now().UnixNano(), fallbackIDCounter.Add(1))
	sum := sha256.Sum256([]byte(fallback))
	return hex.EncodeToString(sum[:byteCount])
}

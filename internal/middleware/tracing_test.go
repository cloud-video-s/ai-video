package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"ai-video/internal/pkg/tracing"

	"github.com/gin-gonic/gin"
)

func TestTracingMiddlewarePropagatesInboundTrace(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const traceID = "4bf92f3577b34da6a3ce929d0e0e4736"
	engine := gin.New()
	engine.Use(Tracing())
	engine.GET("/test", func(c *gin.Context) {
		span, ok := tracing.SpanFromContext(c.Request.Context())
		if !ok {
			t.Error("request context does not contain a span")
		}
		if GetTraceID(c) != traceID || span.TraceID != traceID {
			t.Errorf("trace IDs = %q, %q; want %q", GetTraceID(c), span.TraceID, traceID)
		}
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set(tracing.HeaderTraceParent, "00-"+traceID+"-00f067aa0ba902b7-01")
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, req)

	if got := response.Header().Get(tracing.HeaderTraceID); got != traceID {
		t.Fatalf("response %s = %q, want %q", tracing.HeaderTraceID, got, traceID)
	}
	parts := strings.Split(response.Header().Get(tracing.HeaderTraceParent), "-")
	if len(parts) != 4 || parts[1] != traceID || parts[2] == "00f067aa0ba902b7" {
		t.Fatalf("response traceparent = %q", response.Header().Get(tracing.HeaderTraceParent))
	}
}

func TestTracingMiddlewareGeneratesTrace(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(Tracing())
	engine.GET("/test", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	response := httptest.NewRecorder()
	engine.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/test", nil))

	if got := response.Header().Get(tracing.HeaderTraceID); len(got) != 32 {
		t.Fatalf("generated trace ID = %q", got)
	}
}

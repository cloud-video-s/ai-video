package middleware

import (
	"ai-video/internal/pkg/tracing"

	"github.com/gin-gonic/gin"
)

const (
	traceIDGinKey = "trace_id"
	spanIDGinKey  = "span_id"
)

// Tracing installs a trace/span identity at the HTTP boundary. It accepts a
// valid W3C traceparent (preferred) or X-Trace-ID, generates a new server span,
// and exposes the trace ID in the response for support and log correlation.
func Tracing() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, span := tracing.ContextFromHeaders(c.Request.Context(), c.Request.Header)
		c.Request = c.Request.WithContext(ctx)
		c.Set(traceIDGinKey, span.TraceID)
		c.Set(spanIDGinKey, span.SpanID)
		c.Header(tracing.HeaderTraceID, span.TraceID)
		c.Header(tracing.HeaderTraceParent, span.TraceParent())
		c.Next()
	}
}

func GetTraceID(c *gin.Context) string {
	if c == nil {
		return ""
	}
	if traceID := c.GetString(traceIDGinKey); traceID != "" {
		return traceID
	}
	if c.Request == nil {
		return ""
	}
	return tracing.TraceIDFromContext(c.Request.Context())
}

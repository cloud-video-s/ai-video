package monitor

import (
	"errors"
	"fmt"
	"os"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Kind is a stable event category intended for log-based alert rules.
type Kind string

const (
	KindError         Kind = "error"
	KindHTTPError     Kind = "http_error"
	KindPanic         Kind = "panic"
	KindTaskFailure   Kind = "task_failure"
	KindComponentExit Kind = "component_exit"
	KindProcessExit   Kind = "process_exit"
)

const httpErrorContextKey = "monitor_http_error"

// Report writes one structured exception event. monitor_event and event_kind
// are deliberately stable so log collectors can alert without parsing the
// human-readable message.
func Report(logger *zap.SugaredLogger, kind Kind, source string, err error, fields ...any) {
	if err == nil {
		err = errors.New("unknown error")
	}
	baseFields := []any{
		"monitor_event", true,
		"event_kind", string(kind),
		"source", source,
		"error", err.Error(),
	}
	baseFields = append(baseFields, fields...)

	if logger == nil {
		_, _ = fmt.Fprintf(os.Stderr, "monitor_event=true event_kind=%s source=%s error=%q\n", kind, source, err)
		return
	}
	logger.Errorw("exception monitored", baseFields...)
}

// ReportPanic records a recovered panic together with its stack.
func ReportPanic(logger *zap.SugaredLogger, source string, recovered any, stack []byte) {
	Report(logger, KindPanic, source, fmt.Errorf("panic: %v", recovered), "stack", string(stack))
	if logger == nil && len(stack) > 0 {
		_, _ = fmt.Fprintf(os.Stderr, "%s\n", stack)
	}
}

// MarkHTTPError attaches a server-side error to the current request. The
// request logger consumes it after all handlers have completed, so the event
// also contains the final status, route, latency and trace identity.
func MarkHTTPError(c *gin.Context, err error) {
	if c == nil || err == nil {
		return
	}
	c.Set(httpErrorContextKey, err)
}

// HTTPError returns the server-side error marked on a request.
func HTTPError(c *gin.Context) (error, bool) {
	if c == nil {
		return nil, false
	}
	value, ok := c.Get(httpErrorContextKey)
	if !ok {
		return nil, false
	}
	err, ok := value.(error)
	return err, ok && err != nil
}

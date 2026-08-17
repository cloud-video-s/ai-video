package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"ai-video/internal/config"
	"ai-video/internal/pkg/errcode"
	"ai-video/internal/pkg/monitor"
	"ai-video/internal/pkg/response"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestLoggerMarksServerFailureAsMonitorEvent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	core, observed := observer.New(zap.InfoLevel)
	previous := config.Log
	config.Log = zap.New(core).Sugar()
	t.Cleanup(func() { config.Log = previous })

	engine := gin.New()
	engine.Use(Logger())
	engine.GET("/failure", func(c *gin.Context) {
		response.Fail(c, errcode.ErrServer, "database unavailable")
	})
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/failure", nil))

	entries := observed.All()
	if len(entries) != 1 {
		t.Fatalf("log entries = %d, want 1", len(entries))
	}
	fields := entries[0].ContextMap()
	if fields["monitor_event"] != true || fields["event_kind"] != string(monitor.KindHTTPError) {
		t.Fatalf("monitor fields = %#v", fields)
	}
	if fields["error"] != "database unavailable" || fields["path"] != "/failure" {
		t.Fatalf("request fields = %#v", fields)
	}
}

func TestLoggerMarksUnhandledHTTP500AsMonitorEvent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	core, observed := observer.New(zap.InfoLevel)
	previous := config.Log
	config.Log = zap.New(core).Sugar()
	t.Cleanup(func() { config.Log = previous })

	engine := gin.New()
	engine.Use(Logger())
	engine.GET("/failure", func(c *gin.Context) {
		c.Status(http.StatusInternalServerError)
	})
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/failure", nil))

	fields := observed.All()[0].ContextMap()
	if fields["monitor_event"] != true || fields["event_kind"] != string(monitor.KindHTTPError) {
		t.Fatalf("monitor fields = %#v", fields)
	}
}

package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"ai-video/internal/config"
	"ai-video/internal/pkg/monitor"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestRecoveryReportsPanicWithStack(t *testing.T) {
	gin.SetMode(gin.TestMode)
	core, observed := observer.New(zap.ErrorLevel)
	previous := config.Log
	config.Log = zap.New(core).Sugar()
	t.Cleanup(func() { config.Log = previous })

	engine := gin.New()
	engine.Use(Recovery())
	engine.GET("/panic", func(*gin.Context) {
		panic("boom")
	})
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/panic", nil))

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}
	entries := observed.All()
	if len(entries) != 1 {
		t.Fatalf("log entries = %d, want 1", len(entries))
	}
	fields := entries[0].ContextMap()
	if fields["monitor_event"] != true || fields["event_kind"] != string(monitor.KindPanic) {
		t.Fatalf("monitor fields = %#v", fields)
	}
	if fields["source"] != "http" || !strings.Contains(fields["stack"].(string), "TestRecoveryReportsPanicWithStack") {
		t.Fatalf("panic fields = %#v", fields)
	}
}

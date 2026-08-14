package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"ai-video/internal/config"
	"ai-video/internal/pkg/tracing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestRecoveryLogsPanicWithTrace(t *testing.T) {
	gin.SetMode(gin.TestMode)
	core, observed := observer.New(zap.ErrorLevel)
	previous := config.Log
	config.Log = zap.New(core).Sugar()
	t.Cleanup(func() { config.Log = previous })

	engine := gin.New()
	engine.Use(Tracing(), Recovery())
	engine.GET("/panic", func(_ *gin.Context) { panic("boom") })

	response := httptest.NewRecorder()
	engine.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/panic", nil))
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusInternalServerError)
	}

	entries := observed.FilterMessage("panic recovered").All()
	if len(entries) != 1 {
		t.Fatalf("panic log entries = %d, want 1", len(entries))
	}
	if got := entries[0].ContextMap()["trace_id"]; got != response.Header().Get(tracing.HeaderTraceID) {
		t.Fatalf("logged trace_id = %#v, response trace ID = %q", got, response.Header().Get(tracing.HeaderTraceID))
	}
	if entries[0].ContextMap()["stack"] == "" {
		t.Fatal("panic stack was not logged")
	}
}

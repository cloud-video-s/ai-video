package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"ai-video/internal/config"
	"ai-video/internal/pkg/errcode"
	"ai-video/internal/pkg/i18n"
	"ai-video/internal/pkg/response"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestLoggerPrintsOriginalAPIErrorAtErrorLevel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	core, logs := observer.New(zap.ErrorLevel)
	previousLogger := config.Log
	config.Log = zap.New(core).Sugar()
	t.Cleanup(func() { config.Log = previousLogger })

	router := gin.New()
	router.Use(Logger())
	router.Use(APIErrorSanitizer())
	router.GET("/api/test", func(c *gin.Context) {
		response.Fail(c, errcode.ErrServer, "database connection refused")
	})
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/test", nil))
	if got := recorder.Header().Get("Content-Language"); got != i18n.LocaleEnUS {
		t.Fatalf("Content-Language = %q", got)
	}

	entries := logs.All()
	if len(entries) != 1 {
		t.Fatalf("error log entries = %d, want 1", len(entries))
	}
	loggedErrors, _ := entries[0].ContextMap()["errors"].(string)
	if !strings.Contains(loggedErrors, "database connection refused") {
		t.Fatalf("logged errors = %q", loggedErrors)
	}
}

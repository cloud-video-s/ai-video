package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestDelayConfigsRequireAuthentication(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	New().RegisterRoutes(router.Group("/api"))

	req := httptest.NewRequest(http.MethodGet, "/api/delay-configs", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("GET /api/delay-configs status = %d, want %d", resp.Code, http.StatusUnauthorized)
	}
}

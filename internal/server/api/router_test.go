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

func TestOBDelayConfigsRequireAuthentication(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	New().RegisterRoutes(router.Group("/api"))

	req := httptest.NewRequest(http.MethodGet, "/api/ob-delay-configs", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("GET /api/ob-delay-configs status = %d, want %d", resp.Code, http.StatusUnauthorized)
	}
}

func TestUploadsRequireAPIAuthentication(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	New().RegisterRoutes(router.Group("/api"))

	req := httptest.NewRequest(http.MethodPost, "/api/uploads/images/batches", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("POST /api/uploads/images/batches status = %d, want %d", resp.Code, http.StatusUnauthorized)
	}
}

func TestBannersRequireAPIAuthentication(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	New().RegisterRoutes(router.Group("/api"))

	req := httptest.NewRequest(http.MethodGet, "/api/banners/list", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("GET /api/banners/list status = %d, want %d", resp.Code, http.StatusUnauthorized)
	}
}

func TestTemplatesRequireAPIAuthentication(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	New().RegisterRoutes(router.Group("/api"))

	req := httptest.NewRequest(http.MethodGet, "/api/templates/list?position_key=home", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("GET /api/templates/list status = %d, want %d", resp.Code, http.StatusUnauthorized)
	}
}

func TestTemplateCategoriesRequireAPIAuthentication(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	New().RegisterRoutes(router.Group("/api"))

	req := httptest.NewRequest(http.MethodGet, "/api/templates/categories?position_key=home", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("GET /api/templates/categories status = %d, want %d", resp.Code, http.StatusUnauthorized)
	}
}

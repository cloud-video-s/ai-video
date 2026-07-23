package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"ai-video/internal/pkg/i18n"

	"github.com/gin-gonic/gin"
)

type countryLanguageStub struct {
	language string
	err      error
	country  string
}

func (s *countryLanguageStub) ResolveLanguage(_ context.Context, countryCode string) (string, error) {
	s.country = countryCode
	return s.language, s.err
}

func TestAPILocalizationUsesCountryLanguage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &countryLanguageStub{language: i18n.LocaleJaJP}
	router := gin.New()
	router.Use(APILocalization(repo))
	router.GET("/", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(HeaderDeviceCountry, "JP")
	req.Header.Set("Accept-Language", i18n.LocaleEnUS)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, req)

	if repo.country != "JP" {
		t.Fatalf("resolved country = %q, want JP", repo.country)
	}
	if got := response.Header().Get("Content-Language"); got != i18n.LocaleJaJP {
		t.Fatalf("Content-Language = %q, want %q", got, i18n.LocaleJaJP)
	}
}

func TestAPILocalizationFallsBackToCountryMapping(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(APILocalization(&countryLanguageStub{err: errors.New("not found")}))
	router.GET("/", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(HeaderDeviceCountry, "US")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, req)

	if got := response.Header().Get("Content-Language"); got != i18n.LocaleEnUS {
		t.Fatalf("Content-Language = %q, want %q", got, i18n.LocaleEnUS)
	}
}

package middleware

import (
	"context"
	"strings"

	"ai-video/internal/pkg/i18n"

	"github.com/gin-gonic/gin"
)

type CountryLanguageRepo interface {
	ResolveLanguage(ctx context.Context, countryCode string) (string, error)
}

// APILocalization marks client API requests for generic localized error
// responses. Country configuration takes precedence over Accept-Language.
func APILocalization(countryRepo CountryLanguageRepo) gin.HandlerFunc {
	return func(c *gin.Context) {
		locale := c.GetHeader("Accept-Language")
		countryCode := strings.TrimSpace(c.GetHeader(HeaderDeviceCountry))
		if countryRepo != nil && countryCode != "" {
			if configured, err := countryRepo.ResolveLanguage(c.Request.Context(), countryCode); err == nil && strings.TrimSpace(configured) != "" {
				locale = configured
			}
			if strings.TrimSpace(locale) == "" {
				locale = i18n.LocaleForCountry(countryCode)
			}
		}
		i18n.MarkAPI(c, locale)
		c.Next()
	}
}

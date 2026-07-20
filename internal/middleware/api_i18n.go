package middleware

import (
	"context"
	"strings"

	"ai-video/internal/pkg/i18n"

	"github.com/gin-gonic/gin"
)

type PackageLanguageRepo interface {
	ResolveLanguage(ctx context.Context, packageCode, packageVersion string) (string, error)
}

// APILocalization marks client API requests for generic localized error
// responses. Package configuration takes precedence over Accept-Language.
func APILocalization(packageRepo PackageLanguageRepo) gin.HandlerFunc {
	return func(c *gin.Context) {
		locale := c.GetHeader("Accept-Language")
		packageCode := strings.TrimSpace(c.GetHeader(HeaderAppPackage))
		packageVersion := strings.TrimSpace(c.GetHeader(HeaderAppVersion))
		if packageRepo != nil && packageCode != "" {
			if configured, err := packageRepo.ResolveLanguage(c.Request.Context(), packageCode, packageVersion); err == nil && strings.TrimSpace(configured) != "" {
				locale = configured
			}
		}
		i18n.MarkAPI(c, locale)
		c.Next()
	}
}

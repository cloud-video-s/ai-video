package middleware

import (
	"ai-video/internal/pkg/i18n"

	"github.com/gin-gonic/gin"
)

// APIErrorSanitizer marks client API requests for sanitized English errors.
func APIErrorSanitizer() gin.HandlerFunc {
	return func(c *gin.Context) {
		i18n.MarkAPI(c, i18n.LocaleEnUS)
		c.Next()
	}
}

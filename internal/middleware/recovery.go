package middleware

import (
	"net/http"
	"runtime/debug"

	"ai-video/internal/config"

	"github.com/gin-gonic/gin"
)

// Recovery records recovered panics through the context-aware application
// logger so the stack trace can be located with the same trace ID as the
// request log.
func Recovery() gin.HandlerFunc {
	return gin.CustomRecoveryWithWriter(nil, func(c *gin.Context, recovered any) {
		config.Logger(c.Request.Context()).Errorw(
			"panic recovered",
			"panic", recovered,
			"stack", string(debug.Stack()),
		)
		c.AbortWithStatus(http.StatusInternalServerError)
	})
}

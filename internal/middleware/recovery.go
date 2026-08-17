package middleware

import (
	"net/http"
	"runtime/debug"

	"ai-video/internal/config"
	"ai-video/internal/pkg/monitor"

	"github.com/gin-gonic/gin"
)

// Recovery records recovered panics through the context-aware application
// logger so the stack trace can be located with the same trace ID as the
// request log.
func Recovery() gin.HandlerFunc {
	return gin.CustomRecoveryWithWriter(nil, func(c *gin.Context, recovered any) {
		monitor.ReportPanic(config.Logger(c.Request.Context()), "http", recovered, debug.Stack())
		c.AbortWithStatus(http.StatusInternalServerError)
	})
}

package middleware

import (
	"ai-video/internal/config"
	"time"

	"github.com/gin-gonic/gin"
)

func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		clientIP := c.ClientIP()
		method := c.Request.Method
		privateErrors := c.Errors.ByType(gin.ErrorTypePrivate)

		logger := config.Logger(c.Request.Context())
		logRequest := logger.Infow
		if len(privateErrors) > 0 {
			logRequest = logger.Errorw
		}
		logRequest("request",
			"status", status,
			"method", method,
			"path", path,
			"query", query,
			"body", c.Request.Body,
			"ip", clientIP,
			"latency", latency.String(),
			"errors", privateErrors.String(),
		)
	}
}

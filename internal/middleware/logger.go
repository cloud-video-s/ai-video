package middleware

import (
	"ai-video/internal/config"
	"ai-video/internal/pkg/monitor"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := redactQuery(c.Request.URL.RawQuery)

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		clientIP := c.ClientIP()
		method := c.Request.Method
		privateErrors := c.Errors.ByType(gin.ErrorTypePrivate)
		monitoredErr, monitored := monitor.HTTPError(c)

		logger := config.Logger(c.Request.Context())
		logRequest := logger.Infow
		if len(privateErrors) > 0 || status >= http.StatusInternalServerError {
			logRequest = logger.Errorw
		}
		fields := []any{
			"status", status,
			"method", method,
			"path", path,
			"query", query,
			"body", c.Request.Body,
			"ip", clientIP,
			"latency", latency.String(),
			"errors", privateErrors.String(),
		}
		if monitored {
			logRequest = logger.Errorw
			fields = append(fields,
				"monitor_event", true,
				"event_kind", string(monitor.KindHTTPError),
				"source", "http",
				"error", monitoredErr.Error(),
			)
		} else if status >= http.StatusInternalServerError {
			fields = append(fields,
				"monitor_event", true,
				"event_kind", string(monitor.KindHTTPError),
				"source", "http",
				"error", http.StatusText(status),
			)
		}
		logRequest("request", fields...)
	}
}

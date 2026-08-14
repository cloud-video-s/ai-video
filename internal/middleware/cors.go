package middleware

import (
	"time"

	"ai-video/internal/config"
	"ai-video/internal/pkg/tracing"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func Cors() gin.HandlerFunc {

	// in production; an empty list falls back to "*".
	origins := config.Cfg.Server.AllowOrigins
	if len(origins) == 0 {
		origins = []string{"*"}
	}
	return cors.New(cors.Config{
		AllowOrigins: origins,
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders: []string{
			"Origin", "Content-Type", "Authorization", "Accept", "X-Chunk-SHA256",
			tracing.HeaderTraceParent, tracing.HeaderTraceID,
		},
		ExposeHeaders: []string{
			"Content-Length", HeaderRefreshedToken,
			tracing.HeaderTraceParent, tracing.HeaderTraceID,
		},
		MaxAge: 12 * time.Hour,
	})
}

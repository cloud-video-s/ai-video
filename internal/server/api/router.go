package api

import (
	"ai-video/internal/middleware"
	"ai-video/internal/server/api/handler"

	"github.com/gin-gonic/gin"
)

type Module struct{}

func New() *Module {
	return &Module{}
}

func (m *Module) Name() string {
	return "api"
}

func (m *Module) RegisterRoutes(rg *gin.RouterGroup) {
	healthHandler := handler.NewHealthHandler()
	configHandler := handler.NewConfigHandler()
	delayConfigHandler := handler.NewDelayConfigHandler()
	authHandler := handler.NewAuthHandler()

	rg.GET("/health", healthHandler.Health)
	rg.GET("/configs/public", configHandler.Public)

	rg.POST("/auth/device-register", authHandler.DeviceRegister)
	rg.POST("/auth/re-register", authHandler.ReRegister)

	authenticated := rg.Group("", middleware.ApiAuth())
	{
		authenticated.POST("/auth/logout", authHandler.Logout)
		authenticated.GET("/delay-configs", delayConfigHandler.All)
		authenticated.GET("/users/me", authHandler.Profile)
		authenticated.PUT("/users/me/country", authHandler.UpdateCountry)
	}
}

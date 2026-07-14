package api

import (
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

	rg.GET("/health", healthHandler.Health)
	rg.GET("/configs/public", configHandler.Public)
}

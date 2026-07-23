package api

import (
	"ai-video/internal/middleware"
	"ai-video/internal/pkg/upload"
	"ai-video/internal/pkg/uploadruntime"
	"ai-video/internal/repository"
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
	rg.Use(middleware.APILocalization(repository.NewCountryRepo()))
	healthHandler := handler.NewHealthHandler()
	configHandler := handler.NewConfigHandler()
	delayConfigHandler := handler.NewDelayConfigHandler()
	authHandler := handler.NewAuthHandler()
	bannerHandler := handler.NewBannerHandler()
	templateHandler := handler.NewTemplateHandler()
	generationHandler := handler.NewGenerationHandler()
	vipHandler := handler.NewVipHandler()
	paymentHandler := handler.NewPaymentHandler()
	uploadConfig, err := uploadruntime.ManagerConfig()
	if err != nil {
		panic(err)
	}
	uploadManager, err := upload.SharedManager(uploadConfig)
	if err != nil {
		panic(err)
	}
	uploadHandler := upload.NewHTTPHandler(uploadManager, upload.WithCompletionRecording(
		repository.NewUploadRepo(),
		func(c *gin.Context) (upload.UploadOwner, error) {
			return upload.UploadOwner{Type: upload.UploaderAPIUser, ID: middleware.GetAPIUserID(c)}, nil
		},
	))

	rg.GET("/health", healthHandler.Health)
	rg.POST("/auth/login", authHandler.Login)
	authenticated := rg.Group("", middleware.ApiAuth(repository.NewAppUserRepo()))
	{
		authenticated.GET("/ob_delay", delayConfigHandler.All)
		authenticated.POST("/third_binding", authHandler.ThirdBinding)
		uploadHandler.RegisterRoutes(authenticated.Group("/uploads"))
		auth := authenticated.Group("/auth")
		{
			auth.POST("/logout", authHandler.Logout)
		}

		users := authenticated.Group("/users")
		{
			users.GET("/me", authHandler.Profile)
			users.PUT("/me/country", authHandler.UpdateCountry)
			users.GET("/me/identities", authHandler.ListIdentities)
			users.DELETE("/me/identities/:provider", authHandler.UnbindIdentity)
		}

		banners := authenticated.Group("/banners")
		{
			banners.GET("/list", bannerHandler.List)
		}

		templates := authenticated.Group("/templates")
		{
			templates.GET("/categories", templateHandler.Categories)
			templates.GET("/recommend", templateHandler.Recommend)

			templates.GET("/list", templateHandler.List)
			templates.GET("/template_list", templateHandler.TemplateList)
			templates.GET("/template_info", templateHandler.TemplateInfo)

			templates.POST("/:id/favorite", templateHandler.Favorite)
			templates.DELETE("/:id/favorite", templateHandler.Unfavorite)
		}

		generationTasks := authenticated.Group("/generation")
		{
			generationTasks.GET("/models", generationHandler.Models)
			generationTasks.POST("/tasks", generationHandler.Create)
			generationTasks.GET("/tasks", generationHandler.List)
			generationTasks.GET("/tasks/:id", generationHandler.Get)
			generationTasks.GET("/tasks/:id/events", generationHandler.Events)
			generationTasks.DELETE("/tasks/:id", generationHandler.Delete)
		}

		vip := authenticated.Group("/vip")
		{
			vip.GET("/")
			vip.GET("/recommend", vipHandler.Recommend)
		}

		payments := authenticated.Group("/payments")
		{
			payments.POST("/apple/confirm", paymentHandler.ConfirmApple)
		}

		conf := authenticated.Group("/configs")
		{
			conf.GET("/list", configHandler.Public)
		}

	}
}

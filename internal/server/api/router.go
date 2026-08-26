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
	rg.Use(middleware.APIErrorSanitizer())
	healthHandler := handler.NewHealthHandler()
	configHandler := handler.NewConfigHandler()
	delayConfigHandler := handler.NewDelayConfigHandler()
	authHandler := handler.NewAuthHandler()
	bannerHandler := handler.NewBannerHandler()
	toolHandler := handler.NewToolHandler()
	templateHandler := handler.NewTemplateHandler()
	generationHandler := handler.NewGenerationHandler()
	vipHandler := handler.NewVipHandler()
	pointProductHandler := handler.NewPointProductHandler()
	paymentHandler := handler.NewPaymentHandler()
	adjustAttributionHandler := handler.NewAdjustAttributionHandler()
	trackingEventHandler := handler.NewTrackingEventHandler()
	uploadRepo := repository.NewUploadRepo()
	uploadRecordHandler := handler.NewUploadHandler()
	directUploadHandler := upload.NewHTTPHandler(
		nil,
		upload.WithDirectUploadSigner(uploadruntime.DirectSigner()),
		upload.WithDirectPreUploadRecording(uploadRepo),
		upload.WithPublicURLResolver(uploadruntime.PublicURL),
		upload.WithUploadOwnerResolver(func(c *gin.Context) (upload.UploadOwner, error) {
			return upload.UploadOwner{Type: upload.UploaderAPIUser, ID: middleware.GetAPIUserID(c)}, nil
		}),
	)

	rg.GET("/health", healthHandler.Health)

	rg.Group("", middleware.ApiHeader()).
		POST("/auth/login", authHandler.Login)

	// App Store Server Notifications V2 Webhook（公开端点，Apple 服务器调用，无需鉴权）
	// 该 URL 必须在 App Store Connect → App → 服务中配置为版本 2 的通知地址。
	rg.POST("/payments/apple/notification", paymentHandler.AppleServerNotification)

	// Public Adjust server callback. It is authenticated with the dedicated
	// callback token rather than an app user's Bearer token.
	rg.GET("/attributions/adjust/callback", adjustAttributionHandler.Callback)
	rg.POST("/attributions/adjust/callback", adjustAttributionHandler.Callback)

	authenticated := rg.Group("", middleware.ApiAuth(repository.NewAppUserRepo()))
	{
		authenticated.GET("/ob_delay", delayConfigHandler.All)
		authenticated.POST("/third_binding", authHandler.ThirdBinding)
		directUploadHandler.RegisterDirectRoute(authenticated.Group("/uploads"))
		authenticated.GET("/uploads", uploadRecordHandler.ListMine)
		auth := authenticated.Group("/auth")
		{
			auth.POST("/apple_order_login", authHandler.AppleOrderLogin)
			auth.POST("/refresh", authHandler.Refresh)
			auth.POST("/logout", authHandler.Logout)
		}

		users := authenticated.Group("/users")
		{
			users.GET("/me", authHandler.Profile)
			users.PUT("/me/country", authHandler.UpdateCountry)
			users.GET("/points", authHandler.GetPointsList)
			users.POST("/active_reporting", authHandler.ActiveReporting)
		}

		banners := authenticated.Group("/banners")
		{
			banners.GET("/list", bannerHandler.List)
		}

		tools := authenticated.Group("/tools")
		{
			tools.GET("/list", toolHandler.List)
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

			templates.POST("/complaint", templateHandler.Complaint)
		}

		generationTasks := authenticated.Group("/generation")
		{
			generationTasks.GET("/models", generationHandler.Models)
			generationTasks.POST("/tasks", generationHandler.Create)
			generationTasks.POST("/template-tasks", generationHandler.CreateFromTemplate)
			generationTasks.GET("/tasks", generationHandler.List)
			generationTasks.GET("/tasks/:id", generationHandler.Get)
			generationTasks.DELETE("/tasks/:id", generationHandler.Delete)
		}

		vip := authenticated.Group("/vip")
		{
			vip.GET("/recommend", vipHandler.Recommend)
			vip.GET("list", vipHandler.VipList)
		}

		points := authenticated.Group("/points")
		{
			points.GET("/list", pointProductHandler.List)
		}

		authenticated.POST("/orders", paymentHandler.CreateOrder)

		payments := authenticated.Group("/payments")
		{
			payments.POST("/apple/pay", paymentHandler.ConfirmApple)
		}

		conf := authenticated.Group("/configs")
		{
			conf.GET("/list", configHandler.Public)
		}

		attributions := authenticated.Group("/attributions")
		{
			attributions.POST("/adjust/report", adjustAttributionHandler.ReportApp)
		}

		tracking := authenticated.Group("/tracking")
		{
			tracking.POST("/events", trackingEventHandler.Report)
		}

	}
}

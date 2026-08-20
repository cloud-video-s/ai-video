package admin

import (
	"ai-video/internal/middleware"
	"ai-video/internal/pkg/upload"
	"ai-video/internal/pkg/uploadruntime"
	"ai-video/internal/repository"
	"ai-video/internal/server/admin/handler"

	"github.com/gin-gonic/gin"
)

type Module struct{}

func New() *Module {
	return &Module{}
}

func (m *Module) Name() string {
	return "admin"
}

func (m *Module) RegisterRoutes(rg *gin.RouterGroup) {
	authHandler := handler.NewAuthHandler()
	dashboardHandler := handler.NewDashboardHandler()
	userHandler := handler.NewUserHandler()
	appUserHandler := handler.NewAppUserHandler()
	attributionHandler := handler.NewUserAttributionHandler()
	roleHandler := handler.NewRoleHandler()
	menuHandler := handler.NewMenuHandler()
	apiHandler := handler.NewAPIHandler()
	operationLogHandler := handler.NewOperationLogHandler()
	configHandler := handler.NewConfigHandler()
	delayConfigHandler := handler.NewDelayConfigHandler()
	countryHandler := handler.NewCountryHandler()
	channelHandler := handler.NewChannelHandler()
	templateTypeHandler := handler.NewTemplateTypeHandler()
	templateHandler := handler.NewTemplateHandler()
	templateDisplayConfigHandler := handler.NewTemplateDisplayConfigHandler()
	displayPositionHandler := handler.NewDisplayPositionHandler()
	packageHandler := handler.NewPackageHandler()
	packageVersionHandler := handler.NewPackageVersionHandler()
	videoAppHandler := handler.NewVideoAppHandler()
	vipSubscriptionHandler := handler.NewVIPSubscriptionHandler()
	vipSubscriptionLevelHandler := handler.NewVIPSubscriptionLevelHandler()
	pointsHandler := handler.NewPointsHandler()
	orderAdminHandler := handler.NewOrderAdminHandler()
	userPointsLedgerHandler := handler.NewUserPointsLedgerHandler()
	userGenerationTaskHandler := handler.NewUserGenerationTaskHandler()
	bannerHandler := handler.NewBannerHandler()
	bannerPlacementHandler := handler.NewBannerPlacementHandler()
	platformHandler := handler.NewPlatformHandler()
	modelHandler := handler.NewModelHandler()
	modelParameterHandler := handler.NewModelParameterHandler()
	uploadConfig, err := uploadruntime.ManagerConfig()
	if err != nil {
		panic(err)
	}
	uploadManager, err := upload.SharedManager(uploadConfig)
	if err != nil {
		panic(err)
	}
	uploadRecordHandler := handler.NewUploadHandler()
	configFileHandler := handler.NewConfigFileHandler(uploadConfig.Storage)
	uploadHandler := upload.NewHTTPHandler(
		uploadManager,
		upload.WithCompletionRecording(
			repository.NewUploadRepo(),
			func(c *gin.Context) (upload.UploadOwner, error) {
				return upload.UploadOwner{Type: upload.UploaderAdmin, ID: middleware.GetAdminID(c)}, nil
			},
		),
		upload.WithPublicURLResolver(uploadruntime.PublicURL),
	)

	// Public routes (no auth required)
	rg.POST("/login", authHandler.Login)

	// Authenticated routes (JWT only, no RBAC)
	// For: user profile, option/dropdown data queries
	authenticated := rg.Group("", middleware.AdminAuth())
	{
		authenticated.POST("/logout", authHandler.Logout)
		authenticated.GET("/dashboard", dashboardHandler.Get)
		authenticated.GET("/profile", authHandler.GetProfile)
		authenticated.GET("/permissions", authHandler.GetPermissions)
		authenticated.GET("/menus/user", menuHandler.GetUserMenuTree)
		authenticated.GET("/roles/all", roleHandler.ListAll)
		authenticated.GET("/users/options", userHandler.ListOptions)
		authenticated.GET("/menus/tree", menuHandler.GetTree)
		authenticated.GET("/apis/all", apiHandler.ListAll)
		authenticated.GET("/countries/options", countryHandler.ListOptions)
		authenticated.GET("/channels/options", channelHandler.ListOptions)
		authenticated.GET("/channels/media-options", channelHandler.ListMediaOptions)
		authenticated.GET("/template-types/options", templateTypeHandler.ListOptions)
		authenticated.GET("/templates/options", templateHandler.ListOptions)
		authenticated.GET("/display-positions/options", displayPositionHandler.ListOptions)
		authenticated.GET("/packages/options", packageHandler.ListOptions)
		authenticated.GET("/apps/options", videoAppHandler.ListOptions)
		authenticated.GET("/points/options", pointsHandler.ListOptions)
		authenticated.GET("/vip-subscription-levels/options", vipSubscriptionLevelHandler.ListOptions)
		authenticated.GET("/banners/delivery-options", bannerHandler.DeliveryOptions)
		authenticated.GET("/banner-placements/options", bannerPlacementHandler.ListOptions)
		authenticated.GET("/platforms/options", platformHandler.ListOptions)
	}

	// Protected routes (JWT + Casbin RBAC)
	// For: CRUD management operations
	// OperationLog sits between auth and RBAC so denied (403) attempts are still audited.
	auth := rg.Group("", middleware.AdminAuth(), middleware.OperationLog(), middleware.CasbinRBAC())
	{
		// Users
		auth.GET("/users", userHandler.List)
		auth.POST("/users", userHandler.Create)
		auth.GET("/users/:id", userHandler.GetByID)
		auth.PUT("/users/:id", userHandler.Update)
		auth.DELETE("/users/:id", userHandler.Delete)

		// Client app users
		auth.GET("/app-users", appUserHandler.List)
		auth.POST("/app-users", appUserHandler.Create)
		auth.GET("/app-users/lookup", appUserHandler.Lookup)
		auth.GET("/app-users/:id", appUserHandler.GetByID)
		auth.GET("/app-users/:id/center", appUserHandler.GetCenter)
		auth.PUT("/app-users/:id", appUserHandler.Update)
		auth.DELETE("/app-users/:id", appUserHandler.Delete)
		auth.PATCH("/app-users/:id/frozen", appUserHandler.SetFrozen)
		auth.PATCH("/app-users/:id/blacklisted", appUserHandler.SetBlacklisted)
		auth.PUT("/app-users/:id/phone", appUserHandler.BindPhone)
		auth.POST("/app-users/:id/vip", appUserHandler.GrantVIP)
		auth.POST("/app-users/:id/vip/extend", appUserHandler.ExtendVIP)
		auth.POST("/app-users/:id/vip/transfer", appUserHandler.TransferVIP)
		auth.DELETE("/app-users/:id/vip", appUserHandler.TerminateVIP)
		auth.DELETE("/app-users/:id/device", appUserHandler.ClearDevice)

		// User attribution
		auth.GET("/user-attributions", attributionHandler.List)
		auth.POST("/user-attributions/sync", attributionHandler.SyncUsers)
		auth.GET("/user-attributions/:id", attributionHandler.GetByID)
		auth.PUT("/user-attributions/:id", attributionHandler.Update)
		auth.POST("/user-attributions/:id/events", attributionHandler.RecordEvent)

		// Roles
		auth.GET("/roles", roleHandler.List)
		auth.POST("/roles", roleHandler.Create)
		auth.GET("/roles/:id", roleHandler.GetByID)
		auth.PUT("/roles/:id", roleHandler.Update)
		auth.DELETE("/roles/:id", roleHandler.Delete)
		auth.PUT("/roles/:id/menus", roleHandler.SetMenus)
		auth.PUT("/roles/:id/apis", roleHandler.SetAPIs)
		auth.GET("/roles/:id/apis", roleHandler.GetAPIs)

		// Menus
		auth.POST("/menus", menuHandler.Create)
		auth.GET("/menus/:id", menuHandler.GetByID)
		auth.PUT("/menus/:id", menuHandler.Update)
		auth.DELETE("/menus/:id", menuHandler.Delete)

		// APIs
		auth.GET("/apis", apiHandler.List)
		auth.POST("/apis", apiHandler.Create)
		auth.GET("/apis/:id", apiHandler.GetByID)
		auth.PUT("/apis/:id", apiHandler.Update)
		auth.DELETE("/apis/:id", apiHandler.Delete)

		// Operation logs (audit)
		auth.GET("/operation-logs", operationLogHandler.List)
		auth.GET("/operation-logs/:id", operationLogHandler.GetByID)
		auth.DELETE("/operation-logs/:id", operationLogHandler.Delete)
		auth.DELETE("/operation-logs", operationLogHandler.Clear)

		// System configs
		auth.GET("/configs", configHandler.List)
		auth.POST("/configs", configHandler.Create)
		auth.PUT("/configs", configHandler.BatchUpdate)
		auth.PUT("/configs/:id", configHandler.Update)
		auth.DELETE("/configs/:id", configHandler.Delete)
		auth.POST("/configs/refresh", configHandler.Refresh)

		// Video generation platforms and models
		auth.GET("/platforms", platformHandler.List)
		auth.POST("/platforms", platformHandler.Create)
		auth.GET("/platforms/:id", platformHandler.Get)
		auth.PUT("/platforms/:id", platformHandler.Update)
		auth.DELETE("/platforms/:id", platformHandler.Delete)

		auth.GET("/models", modelHandler.List)
		auth.POST("/models", modelHandler.Create)
		auth.GET("/models/:id", modelHandler.Get)
		auth.PUT("/models/:id", modelHandler.Update)
		auth.DELETE("/models/:id", modelHandler.Delete)
		auth.GET("/models/:id/parameters", modelParameterHandler.List)
		auth.POST("/models/:id/parameters", modelParameterHandler.Create)
		auth.PUT("/models/:id/parameters/:parameter_id", modelParameterHandler.Update)
		auth.DELETE("/models/:id/parameters/:parameter_id", modelParameterHandler.Delete)

		// OB delay configs
		auth.GET("/delay-configs", delayConfigHandler.List)
		auth.GET("/delay-configs/groups", delayConfigHandler.ListGroups)
		auth.POST("/delay-configs", delayConfigHandler.Create)
		auth.PUT("/delay-configs/values", delayConfigHandler.BatchUpdateValues)
		auth.POST("/delay-configs/sync", delayConfigHandler.Sync)
		auth.GET("/delay-configs/:id", delayConfigHandler.GetByID)
		auth.PUT("/delay-configs/:id", delayConfigHandler.Update)
		auth.DELETE("/delay-configs/:id", delayConfigHandler.Delete)

		// Country reference data
		auth.GET("/countries", countryHandler.List)
		auth.POST("/countries", countryHandler.Create)
		auth.GET("/countries/:id", countryHandler.GetByID)
		auth.PUT("/countries/:id", countryHandler.Update)
		auth.PATCH("/countries/:id/status", countryHandler.UpdateStatus)
		auth.DELETE("/countries/:id", countryHandler.Delete)

		// Advertising and distribution channels
		auth.GET("/channels", channelHandler.List)
		auth.POST("/channels", channelHandler.Create)
		auth.GET("/channels/:id", channelHandler.GetByID)
		auth.PUT("/channels/:id", channelHandler.Update)
		auth.PATCH("/channels/:id/status", channelHandler.UpdateStatus)
		auth.DELETE("/channels/:id", channelHandler.Delete)

		// Video template categories
		auth.GET("/template-types", templateTypeHandler.List)
		auth.POST("/template-types", templateTypeHandler.Create)
		auth.GET("/template-types/:id", templateTypeHandler.GetByID)
		auth.PUT("/template-types/:id", templateTypeHandler.Update)
		auth.DELETE("/template-types/:id", templateTypeHandler.Delete)

		// Video templates
		auth.GET("/templates", templateHandler.List)
		auth.POST("/templates", templateHandler.Create)
		auth.GET("/templates/:id", templateHandler.GetByID)
		auth.PUT("/templates/:id", templateHandler.Update)
		auth.DELETE("/templates/:id", templateHandler.Delete)
		auth.GET("/templates/:id/model-parameters", templateHandler.ModelParameters)
		auth.PUT("/templates/:id/model-parameters", templateHandler.ReplaceModelParameters)

		// Concrete template display-position configurations
		auth.GET("/template-display-configs", templateDisplayConfigHandler.List)
		auth.POST("/template-display-configs", templateDisplayConfigHandler.Create)
		auth.GET("/template-display-configs/:id", templateDisplayConfigHandler.GetByID)
		auth.PUT("/template-display-configs/:id", templateDisplayConfigHandler.Update)
		auth.DELETE("/template-display-configs/:id", templateDisplayConfigHandler.Delete)

		// Display positions
		auth.GET("/display-positions", displayPositionHandler.List)
		auth.POST("/display-positions", displayPositionHandler.Create)
		auth.GET("/display-positions/:id", displayPositionHandler.GetByID)
		auth.PUT("/display-positions/:id", displayPositionHandler.Update)
		auth.DELETE("/display-positions/:id", displayPositionHandler.Delete)

		// Applications
		auth.GET("/apps", videoAppHandler.List)
		auth.POST("/apps", videoAppHandler.Create)
		auth.GET("/apps/:id", videoAppHandler.GetByID)
		auth.PUT("/apps/:id", videoAppHandler.Update)
		auth.DELETE("/apps/:id", videoAppHandler.Delete)

		// Downloadable application packages
		auth.GET("/packages", packageHandler.List)
		auth.POST("/packages", packageHandler.Create)
		auth.GET("/packages/:id", packageHandler.GetByID)
		auth.PUT("/packages/:id", packageHandler.Update)
		auth.DELETE("/packages/:id", packageHandler.Delete)

		// Downloadable application package versions
		auth.GET("/package-versions", packageVersionHandler.List)
		auth.POST("/package-versions", packageVersionHandler.Create)
		auth.GET("/package-versions/:id", packageVersionHandler.GetByID)
		auth.PUT("/package-versions/:id", packageVersionHandler.Update)
		auth.DELETE("/package-versions/:id", packageVersionHandler.Delete)

		// VIP subscription plans
		auth.GET("/vip-subscriptions", vipSubscriptionHandler.List)
		auth.POST("/vip-subscriptions", vipSubscriptionHandler.Create)
		auth.GET("/vip-subscriptions/:id", vipSubscriptionHandler.GetByID)
		auth.PUT("/vip-subscriptions/:id", vipSubscriptionHandler.Update)
		auth.DELETE("/vip-subscriptions/:id", vipSubscriptionHandler.Delete)
		auth.PATCH("/vip-subscriptions/:id/status", vipSubscriptionHandler.UpdateStatus)
		auth.PATCH("/vip-subscriptions/:id/display", vipSubscriptionHandler.UpdateDisplayMode)
		auth.PATCH("/vip-subscriptions/:id/default", vipSubscriptionHandler.SetDefault)
		auth.POST("/vip-subscriptions/:id/clone", vipSubscriptionHandler.Clone)

		// VIP subscription levels
		auth.GET("/vip-subscription-levels", vipSubscriptionLevelHandler.List)
		auth.POST("/vip-subscription-levels", vipSubscriptionLevelHandler.Create)
		auth.GET("/vip-subscription-levels/:id", vipSubscriptionLevelHandler.GetByID)
		auth.PUT("/vip-subscription-levels/:id", vipSubscriptionLevelHandler.Update)
		auth.PATCH("/vip-subscription-levels/:id/status", vipSubscriptionLevelHandler.UpdateStatus)
		auth.DELETE("/vip-subscription-levels/:id", vipSubscriptionLevelHandler.Delete)

		// One-time points packages
		auth.GET("/points", pointsHandler.List)
		auth.POST("/points", pointsHandler.Create)
		auth.GET("/points/:id", pointsHandler.GetByID)
		auth.PUT("/points/:id", pointsHandler.Update)
		auth.DELETE("/points/:id", pointsHandler.Delete)
		auth.PATCH("/points/:id/status", pointsHandler.UpdateStatus)
		auth.PATCH("/points/:id/default", pointsHandler.SetDefault)

		// Read-only commerce orders
		auth.GET("/orders", orderAdminHandler.List)
		auth.GET("/orders/:id", orderAdminHandler.GetByID)

		// Read-only user points ledger
		auth.GET("/user-points-ledgers", userPointsLedgerHandler.List)
		auth.GET("/user-points-ledgers/:id", userPointsLedgerHandler.GetByID)

		// Read-only client user generation tasks
		auth.GET("/user-generation-tasks", userGenerationTaskHandler.List)
		auth.GET("/user-generation-tasks/:id", userGenerationTaskHandler.GetByID)

		// Banners
		auth.GET("/banners", bannerHandler.List)
		auth.POST("/banners", bannerHandler.Create)
		auth.GET("/banners/:id", bannerHandler.GetByID)
		auth.PUT("/banners/:id", bannerHandler.Update)
		auth.DELETE("/banners/:id", bannerHandler.Delete)

		// Banner placements
		auth.GET("/banner-placements", bannerPlacementHandler.List)
		auth.POST("/banner-placements", bannerPlacementHandler.Create)
		auth.GET("/banner-placements/:id", bannerPlacementHandler.GetByID)
		auth.PUT("/banner-placements/:id", bannerPlacementHandler.Update)
		auth.DELETE("/banner-placements/:id", bannerPlacementHandler.Delete)

		// Chunked media uploads and small APP/config documents
		auth.GET("/uploads", uploadRecordHandler.List)
		auth.POST("/uploads/config-files", configFileHandler.Upload)
		uploadHandler.RegisterRoutes(auth.Group("/uploads"))
	}
}

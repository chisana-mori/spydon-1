package api

import (
	"net/http"
	"net/url"
	"strings"

	"robusta-web/backend/internal/config"
	"robusta-web/backend/internal/constants"
	"robusta-web/backend/internal/db"
	"robusta-web/backend/internal/middleware"
	"robusta-web/backend/internal/services"

	"github.com/gin-gonic/gin"
	cas "gopkg.in/cas.v2"
)

type handlerSet struct {
	cfg           *config.Config
	apiKeyService *services.APIKeyService
	ingest        *IngestHandler
	query         *QueryHandler
	rca           *RCAHandler
	holmesProxy   *HolmesProxyHandler
	auth          *AuthHandler
	user          *UserHandler
	apiKey        *APIKeyHandler
	health        *HealthHandler
	knowledge     *KnowledgeHandler
	systemSetting *SystemSettingHandler
}

func buildHandlerSet(database *db.Database, cfg *config.Config) (*handlerSet, error) {
	clusterService := services.NewClusterService(database)
	auditService := services.NewAuditService(database)

	objectStorage, err := services.NewObjectStorageService(cfg)
	if err != nil {
		return nil, err
	}

	knowledgeService := services.NewKnowledgeService(database, objectStorage)
	systemSettingService := services.NewSystemSettingService(database)
	holmesService := services.NewHolmesService(database, cfg, objectStorage, knowledgeService)
	rcaService := services.NewRCAService(database, auditService, objectStorage, systemSettingService, holmesService, knowledgeService)
	alertService := services.NewAlertService(database, clusterService, auditService, rcaService, objectStorage)
	authService := services.NewAuthService(database, cfg)
	userService := services.NewUserService(database)
	apiKeyService := services.NewAPIKeyService(database)

	casClient, err := buildCASClient(cfg)
	if err != nil {
		return nil, err
	}

	handlers := &handlerSet{
		cfg:           cfg,
		apiKeyService: apiKeyService,
		ingest: NewIngestHandler(&IngestHandlerConfig{
			AlertService:   alertService,
			RCAService:     rcaService,
			ClusterService: clusterService,
			AuditService:   auditService,
			StorageService: objectStorage,
		}),
		query:         NewQueryHandler(alertService, rcaService, clusterService, objectStorage),
		rca:           NewRCAHandler(holmesService, rcaService),
		holmesProxy:   NewHolmesProxyHandler(cfg, holmesService, alertService),
		auth:          NewAuthHandler(authService, casClient, cfg),
		user:          NewUserHandler(userService),
		apiKey:        NewAPIKeyHandler(apiKeyService),
		health:        NewHealthHandler(database),
		knowledge:     NewKnowledgeHandler(knowledgeService),
		systemSetting: NewSystemSettingHandler(systemSettingService, rcaService),
	}

	return handlers, nil
}

func buildCASClient(cfg *config.Config) (*cas.Client, error) {
	if !cfg.CAS.Enabled || cfg.CAS.ServerURL == "" {
		return nil, nil
	}

	casURL, err := url.Parse(cfg.CAS.ServerURL)
	if err != nil {
		return nil, err
	}

	cookieSecure := strings.HasPrefix(strings.ToLower(cfg.CAS.RedirectURL), "https")
	casOptions := &cas.Options{
		URL:          casURL,
		SessionStore: cas.NewMemorySessionStore(),
		SendService:  true,
		Cookie: &http.Cookie{
			Path:     "/",
			HttpOnly: true,
			Secure:   cookieSecure,
			SameSite: http.SameSiteLaxMode,
		},
	}

	return cas.NewClient(casOptions), nil
}

type routeRegistrar struct {
	router   *gin.Engine
	cfg      *config.Config
	handlers *handlerSet
}

func newRouteRegistrar(router *gin.Engine, cfg *config.Config, handlers *handlerSet) *routeRegistrar {
	return &routeRegistrar{router: router, cfg: cfg, handlers: handlers}
}

func (r *routeRegistrar) register() {
	r.applyGlobalMiddleware()
	r.registerHealthRoutes()
	r.registerAuthRoutes()
	r.registerCASRoutes()

	v1 := r.router.Group(constants.APIVersionV1)
	r.registerWebhookRoute(v1)
	r.registerIngestRoutes(v1)
	r.registerProfileRoutes(v1)
	r.registerQueryRoutes(v1)
	r.registerHolmesRoutes(v1)
	r.registerAPIKeyRoutes(v1)
	r.registerAdminRoutes(v1)
	r.registerKnowledgeRoutes(v1)
}

func (r *routeRegistrar) applyGlobalMiddleware() {
	r.router.Use(middleware.CORSMiddleware())
	r.router.Use(middleware.SecurityHeadersMiddleware())
	r.router.Use(middleware.RequestIDMiddleware())
	r.router.Use(middleware.ErrorHandler())
	r.router.Use(middleware.RequestResponseLogger())
}

func (r *routeRegistrar) registerHealthRoutes() {
	r.router.GET(constants.HealthPath, r.handlers.health.HealthCheck)
	r.router.GET(constants.ReadyPath, r.handlers.health.ReadinessCheck)
}

func (r *routeRegistrar) registerAuthRoutes() {
	authGroup := r.router.Group(constants.AuthPathBase)
	authGroup.GET("/url", r.handlers.auth.GetAuthURL)
	authGroup.POST("/callback", r.handlers.auth.HandleCallback)
	authGroup.POST("/refresh", r.handlers.auth.RefreshToken)
	authGroup.POST("/logout", r.handlers.auth.Logout)
}

func (r *routeRegistrar) registerCASRoutes() {
	// 无论 CAS 是否启用，都注册登录与登出路由，由处理器自行判断
	r.router.GET(constants.CASLoginPath, r.handlers.auth.CASLogin)

	callbackPath := r.cfg.CAS.CallbackPath
	if callbackPath == "" {
		callbackPath = "/auth/cas/callback"
	}

	r.router.GET(callbackPath, r.handlers.auth.CASCallback)
	r.router.GET(constants.CASLogoutPath, r.handlers.auth.CASLogout)
}

func (r *routeRegistrar) registerWebhookRoute(v1 *gin.RouterGroup) {
	group := v1.Group("")
	group.Use(middleware.APIKeyMiddleware(r.cfg, r.handlers.apiKeyService))
	group.POST("/ingest/robusta-webhook", r.handlers.ingest.IngestRobustaFinding)
	group.POST("/ingest/alertmanager", r.handlers.ingest.IngestAlertmanagerWebhook)
}

func (r *routeRegistrar) registerIngestRoutes(v1 *gin.RouterGroup) {
	ingestGroup := v1.Group("/ingest")
	ingestGroup.Use(middleware.EnhancedHMACMiddleware(r.cfg))
	ingestGroup.Use(middleware.AuditLogMiddleware())
	ingestGroup.POST("/alert", r.handlers.ingest.IngestAlert)
}

func (r *routeRegistrar) registerProfileRoutes(v1 *gin.RouterGroup) {
	profileGroup := v1.Group("")
	profileGroup.Use(middleware.CookieAuthMiddleware(r.cfg))
	profileGroup.Use(middleware.AuditLogMiddleware())
	profileGroup.GET("/profile", r.handlers.auth.GetProfile)
	profileGroup.PUT("/profile", r.handlers.auth.UpdateProfile)
	profileGroup.POST("/profile/change-password", r.handlers.auth.ChangePassword)
	profileGroup.GET("/profile/sessions", r.handlers.auth.GetUserSessions)
	profileGroup.DELETE("/profile/sessions/:session_id", r.handlers.auth.RevokeSession)
}

func (r *routeRegistrar) registerQueryRoutes(v1 *gin.RouterGroup) {
	queryGroup := v1.Group("")
	queryGroup.Use(middleware.CookieAuthMiddleware(r.cfg))
	queryGroup.Use(middleware.RequireAdmin())
	queryGroup.Use(middleware.AuditLogMiddleware())

	queryGroup.GET("/clusters/summary", r.handlers.query.GetClustersSummary)
	queryGroup.GET("/clusters", r.handlers.query.GetClusters)
	queryGroup.GET("/clusters/:id", r.handlers.query.GetCluster)

	queryGroup.GET("/alerts", r.handlers.query.GetAlerts)
	queryGroup.GET("/alerts/trend", r.handlers.query.GetAlertTrend)
	queryGroup.GET("/alerts/:id", r.handlers.query.GetAlert)
	queryGroup.GET("/alerts/:id/raw-payload", r.handlers.query.GetAlertRawPayload)

	r.registerRCARoutes(queryGroup)
	queryGroup.GET("/events/stream", r.handlers.query.EventStream)
}

func (r *routeRegistrar) registerRCARoutes(queryGroup *gin.RouterGroup) {
	rcaGroup := queryGroup.Group("/rca")
	rcaGroup.GET("/:alert_id", r.handlers.rca.GetRCAByAlertID)
	rcaGroup.GET("/:alert_id/stream", r.handlers.rca.StreamRCA)
	rcaGroup.GET("/:alert_id/cache", r.handlers.rca.GetRCACacheByAlertID)
	rcaGroup.POST("/:alert_id/trigger", r.handlers.query.TriggerRCA)
	rcaGroup.POST("/trigger", r.handlers.rca.TriggerRCA)
	rcaGroup.GET("/runs", r.handlers.rca.ListRCARuns)
	rcaGroup.GET("/runs/:run_id", r.handlers.rca.GetRCARunStatus)
	rcaGroup.GET("/stats", r.handlers.rca.GetRCAStats)
}

func (r *routeRegistrar) registerHolmesRoutes(v1 *gin.RouterGroup) {
	apiGroup := v1.Group("")
	apiGroup.Use(middleware.APITokenMiddleware(r.cfg))
	apiGroup.Use(middleware.AuditLogMiddleware())
	apiGroup.GET("/holmesgpt/stream/investigate", r.handlers.holmesProxy.StreamInvestigate)
	apiGroup.POST("/holmesgpt/stream/investigate/send", r.handlers.holmesProxy.SendApprovalDecision)
}

func (r *routeRegistrar) registerAPIKeyRoutes(v1 *gin.RouterGroup) {
	apiKeyGroup := v1.Group("/apikeys")
	apiKeyGroup.Use(middleware.CookieAuthMiddleware(r.cfg))
	apiKeyGroup.Use(middleware.AuditLogMiddleware())
	apiKeyGroup.POST("", r.handlers.apiKey.CreateAPIKey)
	apiKeyGroup.GET("", r.handlers.apiKey.ListAPIKeys)
	apiKeyGroup.DELETE("/:id", r.handlers.apiKey.DeleteAPIKey)
	apiKeyGroup.PUT("/:id/status", r.handlers.apiKey.UpdateAPIKeyStatus)
}

func (r *routeRegistrar) registerAdminRoutes(v1 *gin.RouterGroup) {
	adminGroup := v1.Group("/admin")
	adminGroup.Use(middleware.CookieAuthMiddleware(r.cfg))
	adminGroup.Use(middleware.RequireAdmin())
	adminGroup.Use(middleware.AuditLogMiddleware())

	adminGroup.GET("/audit-logs", r.handlers.query.GetAuditLogs)
	adminGroup.POST("/clusters", r.handlers.query.CreateCluster)
	adminGroup.PUT("/clusters/:id", r.handlers.query.UpdateCluster)
	adminGroup.DELETE("/clusters/:id", r.handlers.query.DeleteCluster)

	adminGroup.GET("/users", r.handlers.user.GetUsers)
	adminGroup.GET("/users/:id", r.handlers.user.GetUser)
	adminGroup.PUT("/users/:id/admin", r.handlers.user.SetUserAdmin)
	adminGroup.DELETE("/users/:id", r.handlers.user.DeleteUser)

	adminGroup.GET("/apikeys", r.handlers.apiKey.ListAllAPIKeys)

	adminGroup.GET("/settings", r.handlers.systemSetting.ListSettings)
	adminGroup.PUT("/settings/:key", r.handlers.systemSetting.UpdateSetting)
	adminGroup.GET("/settings/auto-rca", r.handlers.systemSetting.GetAutoRCAConfig)
	adminGroup.POST("/settings/auto-rca/init", r.handlers.systemSetting.InitAutoRCAConfig)
}

func (r *routeRegistrar) registerKnowledgeRoutes(v1 *gin.RouterGroup) {
	kbRead := v1.Group("/knowledge")
	kbRead.Use(middleware.CookieAuthMiddleware(r.cfg))
	kbRead.Use(middleware.AuditLogMiddleware())
	kbRead.GET("", r.handlers.knowledge.List)
	kbRead.GET(":id", r.handlers.knowledge.GetByID)

	kbWrite := v1.Group("/knowledge")
	kbWrite.Use(middleware.CookieAuthMiddleware(r.cfg))
	kbWrite.Use(middleware.RequireAdmin())
	kbWrite.Use(middleware.AuditLogMiddleware())
	kbWrite.POST("", r.handlers.knowledge.Create)
	kbWrite.PUT(":id", r.handlers.knowledge.Update)
	kbWrite.POST(":id/publish", r.handlers.knowledge.Publish)
	kbWrite.POST("/upload/presign", r.handlers.knowledge.PresignUpload)
	kbWrite.DELETE(":id", r.handlers.knowledge.Delete)
}

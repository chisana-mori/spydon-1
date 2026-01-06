package api

import (
	"net/http"
	"net/url"
	"strings"

	"robusta-web/backend/internal/config"
	"robusta-web/backend/internal/constants"
	"robusta-web/backend/internal/db"
	apikeyservice "robusta-web/backend/internal/features/apikey/services"
	apikeyhttp "robusta-web/backend/internal/features/apikey/transport/http"
	authservice "robusta-web/backend/internal/features/auth/services"
	authhttp "robusta-web/backend/internal/features/auth/transport/http"
	healthhttp "robusta-web/backend/internal/features/health/transport/http"
	holmesservice "robusta-web/backend/internal/features/holmes/services"
	holmeshttp "robusta-web/backend/internal/features/holmes/transport/http"
	ingestservice "robusta-web/backend/internal/features/ingest/services"
	ingesthttp "robusta-web/backend/internal/features/ingest/transport/http"
	knowledgeservice "robusta-web/backend/internal/features/knowledge/services"
	knowledgehttp "robusta-web/backend/internal/features/knowledge/transport/http"
	navyservice "robusta-web/backend/internal/features/navy/services"
	navyhttp "robusta-web/backend/internal/features/navy/transport/http"
	pipelineservice "robusta-web/backend/internal/features/pipeline/services"
	pipelinehttp "robusta-web/backend/internal/features/pipeline/transport/http"
	queryservice "robusta-web/backend/internal/features/query/services"
	queryhttp "robusta-web/backend/internal/features/query/transport/http"
	rcaservice "robusta-web/backend/internal/features/rca/services"
	rcahttp "robusta-web/backend/internal/features/rca/transport/http"
	sharedservices "robusta-web/backend/internal/features/shared/services"
	systemsettingservice "robusta-web/backend/internal/features/systemsetting/services"
	systemsettinghttp "robusta-web/backend/internal/features/systemsetting/transport/http"
	userservice "robusta-web/backend/internal/features/user/services"
	userhttp "robusta-web/backend/internal/features/user/transport/http"
	"robusta-web/backend/internal/logger"
	"robusta-web/backend/internal/middleware"
	"robusta-web/backend/internal/pkg/nodesync"

	"robusta-web/backend/pkg/redis"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	cas "gopkg.in/cas.v2"
)

type handlerSet struct {
	cfg              *config.Config
	apiKeyService    *apikeyservice.APIKeyService
	holmesService    *holmesservice.HolmesService
	alertService     *ingestservice.AlertService
	rcaService       *rcaservice.RCAService
	authService      *authservice.AuthService
	casClient        *cas.Client
	userService      *userservice.UserService
	systemSettingSvc *systemsettingservice.SystemSettingService
	ingest           *ingesthttp.Handler
	health           *healthhttp.Handler
	query            *queryhttp.Handler
	knowledge        *knowledgehttp.Handler
	pipeline         *pipelinehttp.Handler
	navy             *navyhttp.Handler
}

func buildHandlerSet(database *db.Database, navyDatabase *db.NavyDatabase, cfg *config.Config, nodesyncManager *nodesync.Manager) (*handlerSet, error) {
	clusterService := queryservice.NewClusterService(database)
	auditService := sharedservices.NewAuditService(database)

	objectStorage, err := sharedservices.NewObjectStorageService(cfg)
	if err != nil {
		return nil, err
	}

	knowledgeService := knowledgeservice.NewKnowledgeService(database, objectStorage)
	systemSettingService := systemsettingservice.NewSystemSettingService(database)
	holmesService := holmesservice.NewHolmesService(database, cfg, objectStorage, knowledgeService)
	rcaService := rcaservice.NewRCAService(database, auditService, objectStorage, systemSettingService, holmesService, knowledgeService)
	alertService := ingestservice.NewAlertService(database, clusterService, auditService, rcaService, objectStorage)
	authService := authservice.NewAuthService(database, cfg)
	userService := userservice.NewUserService(database)
	apiKeyService := apikeyservice.NewAPIKeyService(database)

	casClient, err := buildCASClient(cfg)
	if err != nil {
		return nil, err
	}

	// 创建流水线引擎 (使用 AWX 和 Prometheus 运行时)
	awxRuntime := pipelineservice.NewAWXRuntimeFromConfig(cfg)
	prometheusRuntime := pipelineservice.NewPrometheusRuntimeFromConfig()
	pipelineEngine := pipelineservice.NewPipelineEngine(
		database,
		cfg,
		pipelineservice.WithJobRuntime(awxRuntime),
		pipelineservice.WithMetricsRuntime(prometheusRuntime),
	)

	// Navy 设备服务
	navyDeviceService := navyservice.NewNavyDeviceService(navyDatabase, nodesyncManager)

	// Safe Drain Service
	// Initialize Redis for Safe Drain
	redisHandler, err := redis.NewHandler(cfg.Redis.URL, cfg.Redis.PoolSize)
	if err != nil {
		logger.L().Warn("Failed to initialize Redis for Safe Drain, drain features may be limited", zap.Error(err))
		// Consider failure if Redis is critical or use a no-op/mock
	}

	// Cluster Connection Manager (implements K8sClientFactory for Safe Drain)
	clusterConnectionManager := sharedservices.NewClusterConnectionManager(database.DB)
	if err := clusterConnectionManager.Initialize(); err != nil {
		logger.L().Warn("Failed to initialize ClusterConnectionManager", zap.Error(err))
	}

	// Safe Drain Service (SimpleDrainService)
	safeDrainService := navyservice.NewSimpleDrainService(clusterConnectionManager, redisHandler)

	// 设备批量操作服务
	deviceOpsService := navyservice.NewDeviceOperationsService(
		navyDatabase,
		database,
		nodesyncManager,
		awxRuntime,
		cfg,
		logger.L(),
		safeDrainService,
	)

	// K8s 节点管理服务
	k8sNodeManageService := navyservice.NewK8sNodeManageService(database, nodesyncManager)

	// Streamer needs raw access to AWX Client
	awxStreamer := pipelineservice.NewAWXStreamer(database.DB, awxRuntime.GetClient())

	handlers := &handlerSet{
		cfg:              cfg,
		apiKeyService:    apiKeyService,
		holmesService:    holmesService,
		alertService:     alertService,
		rcaService:       rcaService,
		authService:      authService,
		casClient:        casClient,
		userService:      userService,
		systemSettingSvc: systemSettingService,
		ingest: ingesthttp.New(
			alertService,
			rcaService,
			clusterService,
			auditService,
			objectStorage,
		),
		health:    healthhttp.New(database),
		query:     queryhttp.New(alertService, rcaService, clusterService, objectStorage),
		knowledge: knowledgehttp.New(knowledgeService),
		pipeline:  pipelinehttp.New(pipelineEngine, awxStreamer),
		navy: navyhttp.New(
			cfg,
			navyDatabase,
			navyDeviceService,
			deviceOpsService,
			safeDrainService,
			k8sNodeManageService,
		),
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
	router       *gin.Engine
	cfg          *config.Config
	handlers     *handlerSet
	navyDatabase *db.NavyDatabase
}

func newRouteRegistrar(router *gin.Engine, cfg *config.Config, handlers *handlerSet, navyDatabase *db.NavyDatabase) *routeRegistrar {
	return &routeRegistrar{router: router, cfg: cfg, handlers: handlers, navyDatabase: navyDatabase}
}

func (r *routeRegistrar) register() {
	r.applyGlobalMiddleware()
	r.registerHealthRoutes()

	v1 := r.router.Group(constants.APIVersionV1)

	// Auth & profile routes are now handled by feature-first handler
	authhttp.New(r.cfg, r.handlers.authService, r.handlers.casClient).RegisterRoutes(r.router, v1)

	r.registerWebhookRoute(v1)
	r.registerIngestRoutes(v1)
	r.registerQueryRoutes(v1)
	r.registerHolmesRoutes(v1)
	r.registerAdminRoutes(v1)
	r.registerKnowledgeRoutes(v1)
	r.registerPipelineRoutes(v1)
	r.registerNavyRoutes(v1)
}

func (r *routeRegistrar) applyGlobalMiddleware() {
	r.router.Use(middleware.CORSMiddleware())
	r.router.Use(middleware.SecurityHeadersMiddleware())
	r.router.Use(middleware.RequestIDMiddleware())
	r.router.Use(middleware.ErrorHandler())
	r.router.Use(middleware.RequestResponseLogger())
}

func (r *routeRegistrar) registerHealthRoutes() {
	r.handlers.health.RegisterRoutes(r.router)
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

func (r *routeRegistrar) registerQueryRoutes(v1 *gin.RouterGroup) {
	// Create admin group with required middlewares
	adminGroup := v1.Group("/admin")
	adminGroup.Use(middleware.CookieAuthMiddleware(r.cfg))
	adminGroup.Use(middleware.RequireAdmin())

	// Register query routes (alerts and clusters)
	r.handlers.query.RegisterRoutes(v1, adminGroup)

	// Register RCA routes (preserve existing middleware setup)
	queryGroup := v1.Group("")
	queryGroup.Use(middleware.CookieAuthMiddleware(r.cfg))
	queryGroup.Use(middleware.RequireAdmin())
	queryGroup.Use(middleware.AuditLogMiddleware())
	r.registerRCARoutes(queryGroup)
}

func (r *routeRegistrar) registerRCARoutes(queryGroup *gin.RouterGroup) {
	// RCA routes migrated to feature-first handler (preserve URL paths and middlewares)
	rcahttp.New(r.handlers.holmesService, r.handlers.rcaService, r.handlers.alertService).RegisterRoutes(queryGroup)
}

func (r *routeRegistrar) registerHolmesRoutes(v1 *gin.RouterGroup) {
	// Migrate to feature-first handler (preserve URL paths and middlewares)
	holmeshttp.New(r.cfg, r.handlers.holmesService, r.handlers.alertService).RegisterRoutes(v1)
}

func (r *routeRegistrar) registerAdminRoutes(v1 *gin.RouterGroup) {
	adminGroup := v1.Group("/admin")
	adminGroup.Use(middleware.CookieAuthMiddleware(r.cfg))
	adminGroup.Use(middleware.RequireAdmin())
	adminGroup.Use(middleware.AuditLogMiddleware())

	// User management routes migrated to feature-first handler (preserve URL paths)
	userhttp.New(r.handlers.userService).RegisterRoutes(adminGroup)

	// API Keys: migrate to feature-first handler (preserve URL paths)
	apikeyhttp.New(r.cfg, r.handlers.apiKeyService).RegisterRoutes(v1, adminGroup)

	// System Settings routes migrated to feature-first handler with self-registration
	systemsettinghttp.New(r.handlers.systemSettingSvc, r.handlers.rcaService).RegisterRoutes(adminGroup)
}

func (r *routeRegistrar) registerKnowledgeRoutes(v1 *gin.RouterGroup) {
	// Create admin group with required middlewares
	adminGroup := v1.Group("/admin")
	adminGroup.Use(middleware.CookieAuthMiddleware(r.cfg))
	adminGroup.Use(middleware.RequireAdmin())

	// Register knowledge routes
	r.handlers.knowledge.RegisterRoutes(v1, adminGroup)
}

func (r *routeRegistrar) registerPipelineRoutes(v1 *gin.RouterGroup) {
	// Create admin group with required middlewares
	adminGroup := v1.Group("/admin")
	adminGroup.Use(middleware.CookieAuthMiddleware(r.cfg))
	adminGroup.Use(middleware.RequireAdmin())

	// Register pipeline routes
	r.handlers.pipeline.RegisterRoutes(v1, adminGroup)
}

func (r *routeRegistrar) registerNavyRoutes(v1 *gin.RouterGroup) {
	// Navy routes migrated to feature-first handler while preserving
	// URL paths and middleware semantics.
	r.handlers.navy.RegisterRoutes(v1)
}

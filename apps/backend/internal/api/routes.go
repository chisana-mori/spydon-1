package api

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"time"

	"robusta-web/backend/internal/config"
	"robusta-web/backend/internal/constants"
	"robusta-web/backend/internal/db"
	apikeyservice "robusta-web/backend/internal/features/apikey/services"
	apikeyhttp "robusta-web/backend/internal/features/apikey/transport/http"
	authservice "robusta-web/backend/internal/features/auth/services"
	authhttp "robusta-web/backend/internal/features/auth/transport/http"
	calicohttp "robusta-web/backend/internal/features/calico/transport/http"
	configurationhttp "robusta-web/backend/internal/features/configuration/transport/http"
	healthhttp "robusta-web/backend/internal/features/health/transport/http"
	holmesservice "robusta-web/backend/internal/features/holmes/services"
	holmeshttp "robusta-web/backend/internal/features/holmes/transport/http"
	ingestservice "robusta-web/backend/internal/features/ingest/services"
	ingesthttp "robusta-web/backend/internal/features/ingest/transport/http"
	knowledgeservice "robusta-web/backend/internal/features/knowledge/services"
	knowledgehttp "robusta-web/backend/internal/features/knowledge/transport/http"
	navyservice "robusta-web/backend/internal/features/navy/services"
	navyhttp "robusta-web/backend/internal/features/navy/transport/http"
	notificationservice "robusta-web/backend/internal/features/notification/services"
	notificationhttp "robusta-web/backend/internal/features/notification/transport/http"
	pipelineservice "robusta-web/backend/internal/features/pipeline/services"
	pipelinehttp "robusta-web/backend/internal/features/pipeline/transport/http"
	queryservice "robusta-web/backend/internal/features/query/services"
	queryhttp "robusta-web/backend/internal/features/query/transport/http"
	rcaservice "robusta-web/backend/internal/features/rca/services"
	rcahttp "robusta-web/backend/internal/features/rca/transport/http"
	sharedservices "robusta-web/backend/internal/features/shared/services"
	sharedhttp "robusta-web/backend/internal/features/shared/transport/http"
	systemsettingservice "robusta-web/backend/internal/features/systemsetting/services"
	systemsettinghttp "robusta-web/backend/internal/features/systemsetting/transport/http"
	userservice "robusta-web/backend/internal/features/user/services"
	userhttp "robusta-web/backend/internal/features/user/transport/http"
	"robusta-web/backend/internal/middleware"
	"robusta-web/backend/internal/pkg/calico"
	"robusta-web/backend/internal/pkg/nodesync"
	"robusta-web/backend/pkg/logger"
	"robusta-web/backend/pkg/mailer"
	"robusta-web/backend/pkg/redis"
	"robusta-web/backend/pkg/support"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	cas "gopkg.in/cas.v2"
)

// =============================================================================
// BackgroundServices - 后台服务
// =============================================================================

// BackgroundServices 需要在 main 中启动的后台服务
type BackgroundServices struct {
	ChangeManager *navyservice.ChangeManager
	AWXJobPoller  *navyservice.AWXJobPoller
}

// Start 启动所有后台服务
func (s *BackgroundServices) Start(ctx context.Context) {
	if s.ChangeManager != nil {
		go func() {
			if err := s.ChangeManager.RecoverPendingTickets(ctx); err != nil {
				logger.L().Error("恢复待处理变更单失败", zap.Error(err))
			}
		}()
	}
	if s.AWXJobPoller != nil {
		s.AWXJobPoller.Start(ctx)
	}
}

// Stop 停止所有后台服务
func (s *BackgroundServices) Stop() {
	if s.AWXJobPoller != nil {
		s.AWXJobPoller.Stop()
	}
}

// =============================================================================
// SetupRoutes - Navy 风格的路由注册
// =============================================================================

// SetupRoutes 设置 API 路由，采用 Navy 风格的显式注册
func SetupRoutes(
	router *gin.Engine,
	database *db.Database,
	cfg *config.Config,
	nodesyncManager *nodesync.Manager,
	awxRuntime *pipelineservice.AWXRuntime,
) (*BackgroundServices, error) {
	// Apply global middleware first
	applyGlobalMiddleware(router)

	// Build core services
	coreServices, err := buildCoreServices(database, cfg)
	if err != nil {
		return nil, err
	}

	// Build Navy-specific services
	navyServices, redisHandler := buildNavyServices(database, cfg, nodesyncManager, awxRuntime)

	// Build external dependencies
	externalDeps, err := buildExternalDependencies(cfg, awxRuntime, database)
	if err != nil {
		return nil, err
	}

	// Register all routes
	registerRoutes(
		router, database, cfg, nodesyncManager,
		coreServices, navyServices, externalDeps, redisHandler,
	)

	// Return background services for main to start
	return &BackgroundServices{
		ChangeManager: navyServices.changeMgr,
		AWXJobPoller:  navyservice.NewAWXJobPoller(navyServices.changeMgr, awxRuntime, logger.L()),
	}, nil
}

// =============================================================================
// Helper Functions
// =============================================================================

// coreServices holds core business services shared across features
type coreServices struct {
	objectStorage    *sharedservices.ObjectStorageService
	clusterSvc       *queryservice.ClusterService
	auditSvc         *sharedservices.AuditService
	knowledgeSvc     *knowledgeservice.KnowledgeService
	systemSettingSvc *systemsettingservice.SystemSettingService
	holmesSvc        *holmesservice.HolmesService
	rcaSvc           *rcaservice.RCAService
	alertSvc         *ingestservice.AlertService
	authSvc          *authservice.AuthService
	userSvc          *userservice.UserService
	apiKeySvc        *apikeyservice.APIKeyService
	dictionarySvc    *sharedservices.DictionaryService
}

// navyServices holds Navy-specific device management services
type navyServices struct {
	connMgr         *sharedservices.ClusterConnectionManager
	safeDrainSvc    *navyservice.SimpleDrainService
	deviceSvc       *navyservice.NavyDeviceService
	deviceOpsSvc    *navyservice.DeviceOperationsService
	deviceValidator *navyservice.DeviceValidator
	changeMgr       *navyservice.ChangeManager
	k8sNodeMgrSvc   *navyservice.K8sNodeManageService
	f5Svc           *navyservice.F5InfoService
}

// externalDependencies holds external system dependencies
type externalDependencies struct {
	casClient      *cas.Client
	pipelineEngine *pipelineservice.PipelineEngine
	awxStreamer    *pipelineservice.AWXStreamer
}

// applyGlobalMiddleware applies global middleware to the router
func applyGlobalMiddleware(router *gin.Engine) {
	router.Use(
		middleware.CORSMiddleware(),
		middleware.SecurityHeadersMiddleware(),
		middleware.RequestIDMiddleware(),
		middleware.ErrorHandler(),
		middleware.RequestResponseLogger(),
	)
}

// buildCoreServices constructs core business services
func buildCoreServices(database *db.Database, cfg *config.Config) (*coreServices, error) {
	objectStorage, err := sharedservices.NewObjectStorageService(cfg)
	if err != nil {
		return nil, err
	}

	clusterSvc := queryservice.NewClusterService(database)
	auditSvc := sharedservices.NewAuditService(database)
	knowledgeSvc := knowledgeservice.NewKnowledgeService(database, objectStorage)
	systemSettingSvc := systemsettingservice.NewSystemSettingService(database)
	holmesSvc := holmesservice.NewHolmesService(database, cfg, objectStorage, knowledgeSvc)
	rcaSvc := rcaservice.NewRCAService(database, auditSvc, objectStorage, systemSettingSvc, holmesSvc, knowledgeSvc)
	alertSvc := ingestservice.NewAlertService(database, clusterSvc, auditSvc, rcaSvc, objectStorage)
	authSvc := authservice.NewAuthService(database, cfg)
	userSvc := userservice.NewUserService(database)
	apiKeySvc := apikeyservice.NewAPIKeyService(database)
	dictionarySvc := sharedservices.NewDictionaryService(database)

	return &coreServices{
		objectStorage:    objectStorage,
		clusterSvc:       clusterSvc,
		auditSvc:         auditSvc,
		knowledgeSvc:     knowledgeSvc,
		systemSettingSvc: systemSettingSvc,
		holmesSvc:        holmesSvc,
		rcaSvc:           rcaSvc,
		alertSvc:         alertSvc,
		authSvc:          authSvc,
		userSvc:          userSvc,
		apiKeySvc:        apiKeySvc,
		dictionarySvc:    dictionarySvc,
	}, nil
}

// buildNavyServices constructs Navy-specific services
func buildNavyServices(
	database *db.Database,
	cfg *config.Config,
	nodesyncManager *nodesync.Manager,
	awxRuntime *pipelineservice.AWXRuntime,
) (*navyServices, *redis.Handler) {
	// Initialize Redis (optional)
	redisHandler, _ := redis.NewHandler(cfg.ExternalDependencies.Redis.URL, cfg.ExternalDependencies.Redis.Password, cfg.ExternalDependencies.Redis.PoolSize)
	if redisHandler == nil {
		logger.L().Warn("Redis 初始化失败，部分功能受限")
	}

	// Initialize cluster connection manager
	connMgr := sharedservices.NewClusterConnectionManager(database.DB)
	if err := connMgr.Initialize(); err != nil {
		logger.L().Warn("ClusterConnectionManager 初始化失败", zap.Error(err))
	}

	safeDrainSvc := navyservice.NewSimpleDrainService(connMgr, redisHandler, logger.L(), cfg.ChangeManagement.DragonflyEnabled)
	deviceSvc := navyservice.NewNavyDeviceService(database, nodesyncManager)
	deviceOpsSvc := navyservice.NewDeviceOperationsService(
		database, database, nodesyncManager, awxRuntime, cfg, logger.L(), safeDrainSvc,
	)

	// 创建验证器
	deviceValidator := navyservice.NewDeviceValidator(database, nodesyncManager)

	changeMgrCfg := navyservice.ChangeManagerConfig{
		Enabled:          cfg.ChangeManagement.Enabled,
		DragonflyEnabled: cfg.ChangeManagement.DragonflyEnabled,
		Timeout:          time.Duration(cfg.ChangeManagement.TimeoutMinutes) * time.Minute,
	}
	changeMgr := navyservice.NewChangeManager(changeMgrCfg, redisHandler, nil, logger.L(), database)

	k8sNodeMgrSvc := navyservice.NewK8sNodeManageService(database, nodesyncManager)
	f5Svc := navyservice.NewF5InfoService(database.DB)

	return &navyServices{
		connMgr:         connMgr,
		safeDrainSvc:    safeDrainSvc,
		deviceSvc:       deviceSvc,
		deviceOpsSvc:    deviceOpsSvc,
		deviceValidator: deviceValidator,
		changeMgr:       changeMgr,
		k8sNodeMgrSvc:   k8sNodeMgrSvc,
		f5Svc:           f5Svc,
	}, redisHandler
}

// buildExternalDependencies constructs external system dependencies
func buildExternalDependencies(
	cfg *config.Config,
	awxRuntime *pipelineservice.AWXRuntime,
	database *db.Database,
) (*externalDependencies, error) {
	casClient, err := buildCASClient(cfg)
	if err != nil {
		return nil, err
	}

	pipelineEngine := pipelineservice.NewPipelineEngine(
		database, cfg,
		pipelineservice.WithJobRuntime(awxRuntime),
		pipelineservice.WithMetricsRuntime(pipelineservice.NewPrometheusRuntimeFromConfig()),
	)
	awxStreamer := pipelineservice.NewAWXStreamer(database.DB, awxRuntime.GetClient())

	return &externalDependencies{
		casClient:      casClient,
		pipelineEngine: pipelineEngine,
		awxStreamer:    awxStreamer,
	}, nil
}

// registerRoutes registers all API routes using Navy-style explicit registration
func registerRoutes(
	router *gin.Engine,
	database *db.Database,
	cfg *config.Config,
	nodesyncManager *nodesync.Manager,
	core *coreServices,
	navy *navyServices,
	ext *externalDependencies,
	redisHandler *redis.Handler,
) {
	v1 := router.Group(constants.APIVersionV1)

	// Health check (root routes)
	healthHandler := healthhttp.New(database)
	healthHandler.RegisterRoutes(router)

	// Authentication
	authHandler := authhttp.New(cfg, core.authSvc, ext.casClient)
	authHandler.RegisterRoutes(router, v1)

	// Ingest webhooks (API Key authentication)
	ingestHandler := ingesthttp.New(core.alertSvc, core.rcaSvc, core.clusterSvc, core.auditSvc, core.objectStorage)
	webhookGroup := v1.Group("")
	webhookGroup.Use(middleware.APIKeyMiddleware(cfg, core.apiKeySvc))
	webhookGroup.POST("/ingest/robusta-webhook", ingestHandler.IngestRobustaFinding)
	webhookGroup.POST("/ingest/alertmanager", ingestHandler.IngestAlertmanagerWebhook)

	// Ingest endpoints (HMAC authentication)
	ingestGroup := v1.Group("/ingest")
	ingestGroup.Use(middleware.EnhancedHMACMiddleware(cfg))
	ingestGroup.Use(middleware.AuditLogMiddleware())
	ingestGroup.POST("/alert", ingestHandler.IngestAlert)

	// Query endpoints
	queryHandler := queryhttp.New(core.alertSvc, core.rcaSvc, core.clusterSvc, core.objectStorage)
	adminGroup := createAdminGroup(v1, cfg, false)
	queryHandler.RegisterRoutes(v1, adminGroup)

	// RCA endpoints (with audit logging)
	rcaHandler := rcahttp.New(core.holmesSvc, core.rcaSvc, core.alertSvc)
	rcaGroup := createAdminGroup(v1, cfg, true)
	rcaHandler.RegisterRoutes(rcaGroup)

	// User management
	userHandler := userhttp.New(core.userSvc)
	userAdminGroup := createAdminGroup(v1, cfg, true)
	userHandler.RegisterRoutes(userAdminGroup)

	// API Key management
	apiKeyHandler := apikeyhttp.New(cfg, core.apiKeySvc)
	apiKeyHandler.RegisterRoutes(v1, userAdminGroup)

	// System settings
	systemSettingHandler := systemsettinghttp.New(core.systemSettingSvc, core.rcaSvc)
	systemSettingHandler.RegisterRoutes(userAdminGroup)

	// Holmes AI
	holmesHandler := holmeshttp.New(cfg, core.holmesSvc, core.alertSvc)
	holmesHandler.RegisterRoutes(v1)

	// Knowledge base
	knowledgeHandler := knowledgehttp.New(core.knowledgeSvc)
	knowledgeAdminGroup := createAdminGroup(v1, cfg, false)
	knowledgeHandler.RegisterRoutes(v1, knowledgeAdminGroup)

	// Pipeline execution
	pipelineHandler := pipelinehttp.New(ext.pipelineEngine, ext.awxStreamer)
	pipelineHandler.RegisterRoutes(v1, knowledgeAdminGroup)

	// Navy device management
	navyHandler := navyhttp.New(
		cfg, database, navy.deviceSvc, navy.deviceOpsSvc,
		navy.safeDrainSvc, navy.k8sNodeMgrSvc, navy.changeMgr, navy.f5Svc,
		navy.deviceValidator,
	)
	navyHandler.RegisterRoutes(v1)

	// Shared resources (dictionaries, etc.)
	sharedHandler := sharedhttp.New(cfg, core.dictionarySvc)
	sharedHandler.RegisterRoutes(v1)

	// Configuration management
	configHandler := configurationhttp.NewConfigurationHandler(database)
	configGroup := v1.Group("/configuration")
	configGroup.Use(middleware.CookieAuthMiddleware(cfg))
	configGroup.Use(middleware.RequireAdmin())
	configHandler.RegisterRoutes(configGroup)

	// Email notifications
	notificationHandler := buildNotificationHandler(database, cfg, nodesyncManager, redisHandler)
	notificationHandler.RegisterRoutes(v1)

	// Calico network overview
	calicoSvc := calico.NewService(nodesyncManager, cfg)
	calicoHandler := calicohttp.NewHandler(calicoSvc, nodesyncManager)
	calicoHandler.RegisterRoutes(v1)
}

// createAdminGroup creates a route group with admin authentication
func createAdminGroup(v1 *gin.RouterGroup, cfg *config.Config, withAudit bool) *gin.RouterGroup {
	group := v1.Group("/admin")
	group.Use(middleware.CookieAuthMiddleware(cfg))
	group.Use(middleware.RequireAdmin())
	if withAudit {
		group.Use(middleware.AuditLogMiddleware())
	}
	return group
}

// buildNotificationHandler 构建邮件通知处理器
func buildNotificationHandler(
	database *db.Database,
	cfg *config.Config,
	nodesyncManager *nodesync.Manager,
	redisHandler *redis.Handler,
) *notificationhttp.EmailHandler {
	templateSvc := notificationservice.NewEmailTemplateService(database)
	contactSvc := notificationservice.NewEmailContactService(database)
	emailSvc := sharedservices.NewEmailService(cfg)

	// Initialize notification resource services
	notificationservice.InitDeployService(nodesyncManager, database)
	notificationservice.InitNodeService(nodesyncManager, database)
	notificationservice.InitExComponentService(nodesyncManager)
	notificationservice.InitPodService(nodesyncManager, database, redisHandler)

	notificationSvc := notificationservice.NewEmailNotificationService(database, cfg, emailSvc, templateSvc)
	notificationSvc.SetResourceFetcher(nodesyncManager)

	// Use global Mailer from support package
	var m *mailer.Mailer
	if initResult := support.GetInitResult(); initResult != nil {
		m = initResult.Mailer
	}
	// Note: NoticeEmailFe handles nil mailer internally if needed, or we might want to log a warning here
	noticeEmailFe := notificationservice.NewNoticeEmailFe(database, m, nodesyncManager)

	return notificationhttp.NewEmailHandler(templateSvc, contactSvc, notificationSvc, noticeEmailFe)
}

// buildCASClient 构建 CAS 单点登录客户端
func buildCASClient(cfg *config.Config) (*cas.Client, error) {
	if !cfg.CAS.Enabled || cfg.CAS.ServerURL == "" {
		return nil, nil
	}

	casURL, err := url.Parse(cfg.CAS.ServerURL)
	if err != nil {
		return nil, err
	}

	return cas.NewClient(&cas.Options{
		URL:          casURL,
		SessionStore: cas.NewMemorySessionStore(),
		SendService:  true,
		Cookie: &http.Cookie{
			Path:     "/",
			HttpOnly: true,
			Secure:   strings.HasPrefix(strings.ToLower(cfg.CAS.RedirectURL), "https"),
			SameSite: http.SameSiteLaxMode,
		},
	}), nil
}

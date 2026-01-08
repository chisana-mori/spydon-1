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
	"robusta-web/backend/internal/pkg/nodesync"
	"robusta-web/backend/pkg/logger"
	"robusta-web/backend/pkg/redis"

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
	navyDatabase *db.NavyDatabase,
	cfg *config.Config,
	nodesyncManager *nodesync.Manager,
	awxRuntime *pipelineservice.AWXRuntime,
) (*BackgroundServices, error) {
	// =========================================================================
	// 1. 构建核心业务服务
	// =========================================================================
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

	// =========================================================================
	// 2. 构建 Navy 相关服务
	// =========================================================================
	redisHandler, redisErr := redis.NewHandler(cfg.Redis.URL, cfg.Redis.PoolSize)
	if redisErr != nil {
		logger.L().Warn("Redis 初始化失败，部分功能受限", zap.Error(redisErr))
	}

	connMgr := sharedservices.NewClusterConnectionManager(database.DB)
	if initErr := connMgr.Initialize(); initErr != nil {
		logger.L().Warn("ClusterConnectionManager 初始化失败", zap.Error(initErr))
	}

	safeDrainSvc := navyservice.NewSimpleDrainService(connMgr, redisHandler)
	deviceSvc := navyservice.NewNavyDeviceService(navyDatabase, nodesyncManager)
	deviceOpsSvc := navyservice.NewDeviceOperationsService(
		navyDatabase, database, nodesyncManager, awxRuntime, cfg, logger.L(), safeDrainSvc,
	)
	changeMgrCfg := navyservice.ChangeManagerConfig{
		Enabled: cfg.ChangeManagement.Enabled,
		Timeout: time.Duration(cfg.ChangeManagement.TimeoutMinutes) * time.Minute,
	}
	changeMgr := navyservice.NewChangeManager(changeMgrCfg, redisHandler, nil, logger.L())
	k8sNodeMgrSvc := navyservice.NewK8sNodeManageService(database, nodesyncManager)
	f5Svc := navyservice.NewF5InfoService(navyDatabase.DB)

	// =========================================================================
	// 3. 构建外部依赖
	// =========================================================================
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

	// =========================================================================
	// 4. 应用全局中间件
	// =========================================================================
	router.Use(
		middleware.CORSMiddleware(),
		middleware.SecurityHeadersMiddleware(),
		middleware.RequestIDMiddleware(),
		middleware.ErrorHandler(),
		middleware.RequestResponseLogger(),
	)

	// =========================================================================
	// 5. 创建 Handlers 并注册路由 (Navy 风格)
	// =========================================================================
	v1 := router.Group(constants.APIVersionV1)

	// ----- Health (根路由) -----
	healthHandler := healthhttp.New(database)
	healthHandler.RegisterRoutes(router)

	// ----- Auth (认证) -----
	authHandler := authhttp.New(cfg, authSvc, casClient)
	authHandler.RegisterRoutes(router, v1)

	// ----- Ingest Webhooks (API Key 认证) -----
	ingestHandler := ingesthttp.New(alertSvc, rcaSvc, clusterSvc, auditSvc, objectStorage)
	webhookGroup := v1.Group("")
	webhookGroup.Use(middleware.APIKeyMiddleware(cfg, apiKeySvc))
	webhookGroup.POST("/ingest/robusta-webhook", ingestHandler.IngestRobustaFinding)
	webhookGroup.POST("/ingest/alertmanager", ingestHandler.IngestAlertmanagerWebhook)

	// ----- Ingest (HMAC 认证) -----
	ingestGroup := v1.Group("/ingest")
	ingestGroup.Use(middleware.EnhancedHMACMiddleware(cfg))
	ingestGroup.Use(middleware.AuditLogMiddleware())
	ingestGroup.POST("/alert", ingestHandler.IngestAlert)

	// ----- Query -----
	queryHandler := queryhttp.New(alertSvc, rcaSvc, clusterSvc, objectStorage)
	adminGroup := createAdminGroup(v1, cfg, false)
	queryHandler.RegisterRoutes(v1, adminGroup)

	// ----- RCA (审计日志) -----
	rcaHandler := rcahttp.New(holmesSvc, rcaSvc, alertSvc)
	rcaGroup := createAdminGroup(v1, cfg, true)
	rcaHandler.RegisterRoutes(rcaGroup)

	// ----- User -----
	userHandler := userhttp.New(userSvc)
	userAdminGroup := createAdminGroup(v1, cfg, true)
	userHandler.RegisterRoutes(userAdminGroup)

	// ----- API Key -----
	apiKeyHandler := apikeyhttp.New(cfg, apiKeySvc)
	apiKeyHandler.RegisterRoutes(v1, userAdminGroup)

	// ----- System Setting -----
	systemSettingHandler := systemsettinghttp.New(systemSettingSvc, rcaSvc)
	systemSettingHandler.RegisterRoutes(userAdminGroup)

	// ----- Holmes AI -----
	holmesHandler := holmeshttp.New(cfg, holmesSvc, alertSvc)
	holmesHandler.RegisterRoutes(v1)

	// ----- Knowledge -----
	knowledgeHandler := knowledgehttp.New(knowledgeSvc)
	knowledgeAdminGroup := createAdminGroup(v1, cfg, false)
	knowledgeHandler.RegisterRoutes(v1, knowledgeAdminGroup)

	// ----- Pipeline -----
	pipelineHandler := pipelinehttp.New(pipelineEngine, awxStreamer)
	pipelineHandler.RegisterRoutes(v1, knowledgeAdminGroup)

	// ----- Navy (设备管理) -----
	navyHandler := navyhttp.New(
		cfg, navyDatabase, deviceSvc, deviceOpsSvc,
		safeDrainSvc, k8sNodeMgrSvc, changeMgr, f5Svc,
	)
	navyHandler.RegisterRoutes(v1)

	// ----- Shared (字典等) -----
	sharedHandler := sharedhttp.New(cfg, dictionarySvc)
	sharedHandler.RegisterRoutes(v1)

	// ----- Configuration -----
	configHandler := configurationhttp.NewConfigurationHandler(navyDatabase)
	configGroup := v1.Group("/configuration")
	configGroup.Use(middleware.CookieAuthMiddleware(cfg))
	configGroup.Use(middleware.RequireAdmin())
	configHandler.RegisterRoutes(configGroup)

	// ----- Notification (邮件通知) -----
	notificationHandler := buildNotificationHandler(database, cfg, nodesyncManager)
	notificationHandler.RegisterRoutes(v1)

	// =========================================================================
	// 6. 返回后台服务
	// =========================================================================
	bgServices := &BackgroundServices{
		ChangeManager: changeMgr,
		AWXJobPoller:  navyservice.NewAWXJobPoller(changeMgr, awxRuntime, logger.L()),
	}

	return bgServices, nil
}

// =============================================================================
// Helper Functions
// =============================================================================

// createAdminGroup 创建带管理员权限的路由组
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
func buildNotificationHandler(database *db.Database, cfg *config.Config, nodesyncManager *nodesync.Manager) *notificationhttp.EmailHandler {
	templateSvc := notificationservice.NewEmailTemplateService(database)
	contactSvc := notificationservice.NewEmailContactService(database)
	emailSvc := sharedservices.NewEmailService(cfg)

	notificationSvc := notificationservice.NewEmailNotificationService(database, cfg, emailSvc, templateSvc)
	notificationSvc.SetResourceFetcher(nodesyncManager)

	return notificationhttp.NewEmailHandler(templateSvc, contactSvc, notificationSvc)
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

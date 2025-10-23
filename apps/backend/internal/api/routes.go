package api

import (
	"net/http"
	"net/url"
	"strings"

	"robusta-web/backend/internal/config"
	"robusta-web/backend/internal/db"
	"robusta-web/backend/internal/middleware"
	"robusta-web/backend/internal/services"

	"github.com/gin-gonic/gin"
	cas "gopkg.in/cas.v2"
)

// SetupRoutes 设置API路由
func SetupRoutes(router *gin.Engine, database *db.Database, cfg *config.Config) error {
	// 创建服务实例
	clusterService := services.NewClusterService(database)
	alertService := services.NewAlertService(database)
	rcaService := services.NewRCAService(database)
	auditService := services.NewAuditService(database)
	objectStorage, err := services.NewObjectStorageService(cfg)
	if err != nil {
		return err
	}
	holmesService := services.NewHolmesService(database, cfg, objectStorage)
	authService := services.NewAuthService(database, cfg)
	userService := services.NewUserService(database)
	apiKeyService := services.NewAPIKeyService(database)

	var casClient *cas.Client
	if cfg.CAS.Enabled && cfg.CAS.ServerURL != "" {
		casURL, err := url.Parse(cfg.CAS.ServerURL)
		if err != nil {
			return err
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

		casClient = cas.NewClient(casOptions)
	}

	// 创建处理器实例
	ingestHandler := NewIngestHandler(alertService, rcaService, clusterService, auditService, objectStorage)
	queryHandler := NewQueryHandler(alertService, rcaService, clusterService, objectStorage)
	rcaHandler := NewRCAHandler(holmesService)
	holmesProxyHandler := NewHolmesProxyHandler(cfg, holmesService)
	authHandler := NewAuthHandler(authService, casClient, cfg)
	userHandler := NewUserHandler(userService)
	apiKeyHandler := NewAPIKeyHandler(apiKeyService)
	healthHandler := NewHealthHandler(database)

	// 全局中间件
	router.Use(middleware.CORSMiddleware())
	router.Use(middleware.SecurityHeadersMiddleware())
	router.Use(middleware.RequestIDMiddleware())
	router.Use(middleware.RequestResponseLogger())

	// 健康检查路由（无需认证）
	router.GET("/health", healthHandler.HealthCheck)
	router.GET("/ready", healthHandler.ReadinessCheck)

	// 认证路由（无需认证）
	authGroup := router.Group("/auth")
	{
		authGroup.GET("/url", authHandler.GetAuthURL)
		authGroup.POST("/callback", authHandler.HandleCallback)
		authGroup.POST("/refresh", authHandler.RefreshToken)
		authGroup.POST("/logout", authHandler.Logout)
	}

	// 无论 CAS 是否启用，都注册登录与登出路由，由处理器自行判断
	router.GET("/auth/cas/login", authHandler.CASLogin)

	callbackPath := cfg.CAS.CallbackPath
	if callbackPath == "" {
		callbackPath = "/auth/cas/callback"
	}

	router.GET(callbackPath, authHandler.CASCallback)
	router.GET("/auth/cas/logout", authHandler.CASLogout)

	// API v1 路由组
	v1 := router.Group("/api/v1")

	// 本地/生产对接：提供一个使用API Key校验的webhook入口（webhook_sink专用）
	// 生产环境请设置 INGEST_API_KEY，并在Robusta sinksConfig.headers 中附带相同的 X-API-Key 或 Authorization: Bearer
	v1.Group("").Use(middleware.APIKeyMiddleware(cfg, apiKeyService)).POST("/ingest/robusta-webhook", ingestHandler.IngestRobustaFinding)

	// Ingest API（需要增强版HMAC验证，包含时间戳）
	ingestGroup := v1.Group("/ingest")
	ingestGroup.Use(middleware.EnhancedHMACMiddleware(cfg))
	ingestGroup.Use(middleware.AuditLogMiddleware())
	{
		ingestGroup.POST("/alert", ingestHandler.IngestAlert)
		ingestGroup.POST("/rca", ingestHandler.IngestRCA)
	}

	// 用户资料API（仅需要登录，不需要管理员权限）
	profileGroup := v1.Group("")
	profileGroup.Use(middleware.CookieAuthMiddleware(cfg))
	profileGroup.Use(middleware.AuditLogMiddleware())
	{
		profileGroup.GET("/profile", authHandler.GetProfile)
		profileGroup.PUT("/profile", authHandler.UpdateProfile)
		profileGroup.POST("/profile/change-password", authHandler.ChangePassword)
		profileGroup.GET("/profile/sessions", authHandler.GetUserSessions)
		profileGroup.DELETE("/profile/sessions/:session_id", authHandler.RevokeSession)
	}

	// Query API（需要管理员权限）
	queryGroup := v1.Group("")
	queryGroup.Use(middleware.CookieAuthMiddleware(cfg))
	queryGroup.Use(middleware.RequireAdmin())
	queryGroup.Use(middleware.AuditLogMiddleware())
	{
		// 集群相关
		queryGroup.GET("/clusters/summary", queryHandler.GetClustersSummary)
		queryGroup.GET("/clusters", queryHandler.GetClusters)
		queryGroup.GET("/clusters/:id", queryHandler.GetCluster)

		// 告警相关
		queryGroup.GET("/alerts", queryHandler.GetAlerts)
		queryGroup.GET("/alerts/trend", queryHandler.GetAlertTrend)
		queryGroup.GET("/alerts/:id", queryHandler.GetAlert)
		queryGroup.GET("/alerts/:id/raw-payload", queryHandler.GetAlertRawPayload)

		// RCA相关
		rcaGroup := queryGroup.Group("/rca")
		rcaGroup.GET("/:alert_id", rcaHandler.GetRCAByAlertID)
		rcaGroup.GET("/:alert_id/cache", rcaHandler.GetRCACacheByAlertID)
		rcaGroup.POST("/:alert_id/trigger", queryHandler.TriggerRCA)
		rcaGroup.POST("/trigger", rcaHandler.TriggerRCA)
		rcaGroup.GET("/runs", rcaHandler.ListRCARuns)
		rcaGroup.GET("/runs/:run_id", rcaHandler.GetRCARunStatus)
		rcaGroup.GET("/stats", rcaHandler.GetRCAStats)

		// 事件流（SSE）
		queryGroup.GET("/events/stream", queryHandler.EventStream)
	}

	// HolmesGPT API（需要API Token或管理员权限）
	apiGroup := v1.Group("")
	apiGroup.Use(middleware.APITokenMiddleware(cfg))
	apiGroup.Use(middleware.AuditLogMiddleware())
	{
		apiGroup.POST("/holmesgpt/stream/investigate", holmesProxyHandler.StreamInvestigate)
	}

	// API Key管理（需要登录）
	apiKeyGroup := v1.Group("/apikeys")
	apiKeyGroup.Use(middleware.CookieAuthMiddleware(cfg))
	apiKeyGroup.Use(middleware.AuditLogMiddleware())
	{
		apiKeyGroup.POST("", apiKeyHandler.CreateAPIKey)
		apiKeyGroup.GET("", apiKeyHandler.ListAPIKeys)
		apiKeyGroup.DELETE("/:id", apiKeyHandler.DeleteAPIKey)
		apiKeyGroup.PUT("/:id/status", apiKeyHandler.UpdateAPIKeyStatus)
	}

	// 管理API（需要管理员权限）
	adminGroup := v1.Group("/admin")
	adminGroup.Use(middleware.CookieAuthMiddleware(cfg))
	adminGroup.Use(middleware.RequireAdmin())
	adminGroup.Use(middleware.AuditLogMiddleware())
	{
		// 审计日志
		adminGroup.GET("/audit-logs", queryHandler.GetAuditLogs)

		// 集群管理
		adminGroup.DELETE("/clusters/:id", queryHandler.DeleteCluster)

		// 用户管理
		adminGroup.GET("/users", userHandler.GetUsers)
		adminGroup.GET("/users/:id", userHandler.GetUser)
		adminGroup.PUT("/users/:id/admin", userHandler.SetUserAdmin)
		adminGroup.DELETE("/users/:id", userHandler.DeleteUser)

		// API Key管理（管理员查看所有）
		adminGroup.GET("/apikeys", apiKeyHandler.ListAllAPIKeys)
	}

	return nil
}

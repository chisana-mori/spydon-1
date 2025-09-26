package api

import (
	"robusta-web/backend/internal/config"
	"robusta-web/backend/internal/db"
	"robusta-web/backend/internal/middleware"
	"robusta-web/backend/internal/services"

	"github.com/gin-gonic/gin"
)

// SetupRoutes 设置API路由
func SetupRoutes(router *gin.Engine, database *db.Database, cfg *config.Config) error {
	// 创建服务实例
	clusterService := services.NewClusterService(database)
	alertService := services.NewAlertService(database)
	rcaService := services.NewRCAService(database)
	auditService := services.NewAuditService(database)
	holmesService := services.NewHolmesService(database, cfg)
	authService := services.NewAuthService(database, cfg)
	objectStorage, err := services.NewObjectStorageService(cfg)
	if err != nil {
		return err
	}

	// 创建处理器实例
	ingestHandler := NewIngestHandler(alertService, rcaService, clusterService, auditService, objectStorage)
	queryHandler := NewQueryHandler(alertService, rcaService, clusterService, objectStorage)
	rcaHandler := NewRCAHandler(holmesService)
	authHandler := NewAuthHandler(authService)
	healthHandler := NewHealthHandler(database)
	holmesStreamHandler := NewHolmesStreamHandler(cfg, holmesService, objectStorage)

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

	// API v1 路由组
	v1 := router.Group("/api/v1")

	// 本地/生产对接：提供一个使用API Key校验的webhook入口（webhook_sink专用）
	// 生产环境请设置 INGEST_API_KEY，并在Robusta sinksConfig.headers 中附带相同的 X-API-Key 或 Authorization: Bearer
	v1.Group("").Use(middleware.APIKeyMiddleware(cfg)).POST("/ingest/robusta-webhook", ingestHandler.IngestRobustaFinding)

	// Ingest API（需要增强版HMAC验证，包含时间戳）
	ingestGroup := v1.Group("/ingest")
	ingestGroup.Use(middleware.EnhancedHMACMiddleware(cfg))
	ingestGroup.Use(middleware.AuditLogMiddleware())
	{
		ingestGroup.POST("/alert", ingestHandler.IngestAlert)
		ingestGroup.POST("/rca", ingestHandler.IngestRCA)
	}

	// 集群心跳API（需要增强版HMAC验证，包含时间戳）
	clusterGroup := v1.Group("/clusters")
	clusterGroup.Use(middleware.EnhancedHMACMiddleware(cfg))
	{
		clusterGroup.POST("/heartbeat", ingestHandler.ClusterHeartbeat)
	}

	// Query API（开发阶段暂时禁用JWT认证）
	queryGroup := v1.Group("")
	// TODO: 生产环境需要启用认证中间件
	// queryGroup.Use(middleware.AuthMiddleware(cfg))
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
		queryGroup.GET("/rca/:alert_id", queryHandler.GetRCAByAlertID)
		queryGroup.POST("/rca/:alert_id/trigger", queryHandler.TriggerRCA)
		queryGroup.POST("/rca/trigger", rcaHandler.TriggerRCA)
		queryGroup.GET("/rca/runs", rcaHandler.ListRCARuns)
		queryGroup.GET("/rca/runs/:run_id", rcaHandler.GetRCARunStatus)
		queryGroup.GET("/rca/runs/:run_id/stream", holmesStreamHandler.StreamRunReplay)
		queryGroup.POST("/holmesgpt/stream/investigate", holmesStreamHandler.StreamInvestigate)
		queryGroup.GET("/rca/stats", rcaHandler.GetRCAStats)

		// 事件流（SSE）
		queryGroup.GET("/events/stream", queryHandler.EventStream)

		// 用户资料相关
		queryGroup.GET("/profile", authHandler.GetProfile)
		queryGroup.PUT("/profile", authHandler.UpdateProfile)
		queryGroup.POST("/profile/change-password", authHandler.ChangePassword)
		queryGroup.GET("/profile/sessions", authHandler.GetUserSessions)
		queryGroup.DELETE("/profile/sessions/:session_id", authHandler.RevokeSession)
	}

	// 管理API（开发阶段暂时禁用权限检查）
	adminGroup := v1.Group("/admin")
	// TODO: 生产环境需要启用认证和权限中间件
	// adminGroup.Use(middleware.AuthMiddleware(cfg))
	// adminGroup.Use(middleware.RequireRole("admin"))
	adminGroup.Use(middleware.AuditLogMiddleware())
	{
		adminGroup.GET("/audit-logs", queryHandler.GetAuditLogs)
		adminGroup.DELETE("/clusters/:id", queryHandler.DeleteCluster)
	}

	return nil
}

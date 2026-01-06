package http

import (
	ingestservice "robusta-web/backend/internal/features/ingest/services"
	queryservice "robusta-web/backend/internal/features/query/services"
	rcaservice "robusta-web/backend/internal/features/rca/services"
	sharedservices "robusta-web/backend/internal/features/shared/services"
	"robusta-web/backend/internal/middleware"

	"github.com/gin-gonic/gin"
)

// Handler 查询处理器（聚合 alerts 和 clusters 查询）
type Handler struct {
	alertService   *ingestservice.AlertService
	rcaService     *rcaservice.RCAService
	clusterService *queryservice.ClusterService
	storageService sharedservices.PayloadStorage
}

// New 创建新的查询处理器
func New(
	alertService *ingestservice.AlertService,
	rcaService *rcaservice.RCAService,
	clusterService *queryservice.ClusterService,
	storageService sharedservices.PayloadStorage,
) *Handler {
	return &Handler{
		alertService:   alertService,
		rcaService:     rcaService,
		clusterService: clusterService,
		storageService: storageService,
	}
}

// RegisterRoutes 注册查询路由
// v1 is /api/v1 group; admin is /api/v1/admin group with admin middlewares already applied.
func (h *Handler) RegisterRoutes(v1, admin *gin.RouterGroup) {
	// Clusters routes
	clustersGroup := v1.Group("/clusters")
	{
		clustersGroup.GET("/summary", h.GetClustersSummary)
		clustersGroup.GET("", h.GetClusters)
		clustersGroup.GET("/:id", h.GetCluster)
		clustersGroup.GET("/:id/nodes", h.GetClusterNodes)
	}

	// Admin cluster management
	adminClustersGroup := admin.Group("/clusters")
	adminClustersGroup.Use(middleware.AuditLogMiddleware())
	{
		adminClustersGroup.POST("", h.CreateCluster)
		adminClustersGroup.PUT("/:id", h.UpdateCluster)
		adminClustersGroup.DELETE("/:id", h.DeleteCluster)
	}

	// Alerts routes
	alertsGroup := v1.Group("/alerts")
	{
		alertsGroup.GET("", h.GetAlerts)
		alertsGroup.GET("/trend", h.GetAlertTrend)
		alertsGroup.GET("/:id", h.GetAlert)
		alertsGroup.GET("/:id/raw", h.GetAlertRawPayload)
		alertsGroup.GET("/:id/rca", h.GetRCAByAlertID)
		alertsGroup.POST("/:id/rca", h.TriggerRCA)
	}

	// SSE event stream
	v1.GET("/events", h.EventStream)

	// Admin audit logs
	admin.GET("/audit-logs", h.GetAuditLogs)
}

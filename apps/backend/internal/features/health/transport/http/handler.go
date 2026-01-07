package http

import (
	"net/http"
	"time"

	"robusta-web/backend/internal/db"
	httpx "robusta-web/backend/internal/transport/httpx"

	"github.com/gin-gonic/gin"
)

// Handler 健康检查处理器
type Handler struct {
	database *db.Database
}

// New 创建新的健康检查处理器
func New(database *db.Database) *Handler {
	return &Handler{
		database: database,
	}
}

// RegisterRoutes 注册健康检查路由
func (h *Handler) RegisterRoutes(router *gin.Engine) {
	router.GET("/health", h.HealthCheck)
	router.GET("/ready", h.ReadinessCheck)
}

// HealthCheck 基础健康检查
// @Summary 基础健康检查
// @Description 该接口用于验证Robusta Central Hub服务是否正在运行。它返回服务的基本状态信息、当前时间戳、服务名称以及版本号。此接口通常被负载均衡器或监控系统调用，以确保服务进程处于活动状态并能响应请求。
// @Tags Health
// @Produce json
// @Success 200 {object} httpx.Response{data=object}
// @Router /health [get]
func (h *Handler) HealthCheck(c *gin.Context) {
	httpx.Success(c, gin.H{
		"status":    "ok",
		"timestamp": time.Now().Format(time.RFC3339),
		"service":   "robusta-central-hub",
		"version":   "1.0.0",
	})
}

// ReadinessCheck 就绪检查（包括数据库连接检查）
// @Summary 服务就绪检查
// @Description 该接口执行深度的服务就绪检查，包括验证数据库连接池的可用性以及通过Ping操作测试实际的连通性。如果所有关键依赖项均正常，则返回ready状态。此接口常被用于K8s的Readiness Probe，以决定是否将流量分发到该容器。
// @Tags Health
// @Produce json
// @Success 200 {object} httpx.Response{data=object}
// @Failure 503 {object} httpx.ErrorResponse
// @Router /ready [get]
func (h *Handler) ReadinessCheck(c *gin.Context) {
	// 检查数据库连接
	sqlDB, err := h.database.DB.DB()
	if err != nil {
		httpx.Error(c, http.StatusServiceUnavailable, "DATABASE_CONNECTION_ERROR", "数据库连接获取失败")
		return
	}

	if err := sqlDB.Ping(); err != nil {
		httpx.Error(c, http.StatusServiceUnavailable, "DATABASE_PING_ERROR", "数据库连接测试失败")
		return
	}

	httpx.Success(c, gin.H{
		"status":    "ready",
		"timestamp": time.Now().Format(time.RFC3339),
		"checks": gin.H{
			"database": "ok",
		},
	})
}

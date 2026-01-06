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
func (h *Handler) HealthCheck(c *gin.Context) {
	httpx.Success(c, gin.H{
		"status":    "ok",
		"timestamp": time.Now().Format(time.RFC3339),
		"service":   "robusta-central-hub",
		"version":   "1.0.0",
	})
}

// ReadinessCheck 就绪检查（包括数据库连接检查）
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

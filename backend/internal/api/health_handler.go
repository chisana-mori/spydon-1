package api

import (
	"net/http"
	"time"

	"robusta-web/backend/internal/db"

	"github.com/gin-gonic/gin"
)

// HealthHandler 健康检查处理器
type HealthHandler struct {
	database *db.Database
}

// NewHealthHandler 创建新的健康检查处理器
func NewHealthHandler(database *db.Database) *HealthHandler {
	return &HealthHandler{
		database: database,
	}
}

// HealthCheck 基础健康检查
func (h *HealthHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"timestamp": time.Now().Format(time.RFC3339),
		"service":   "robusta-central-hub",
		"version":   "1.0.0",
	})
}

// ReadinessCheck 就绪检查（包括数据库连接检查）
func (h *HealthHandler) ReadinessCheck(c *gin.Context) {
	// 检查数据库连接
	sqlDB, err := h.database.DB.DB()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "error",
			"error":  "数据库连接获取失败",
			"code":   "DATABASE_CONNECTION_ERROR",
		})
		return
	}

	if err := sqlDB.Ping(); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "error",
			"error":  "数据库连接测试失败",
			"code":   "DATABASE_PING_ERROR",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":    "ready",
		"timestamp": time.Now().Format(time.RFC3339),
		"checks": gin.H{
			"database": "ok",
		},
	})
}

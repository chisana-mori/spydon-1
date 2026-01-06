package http

import (
	"robusta-web/backend/internal/features/pipeline/services"
	"robusta-web/backend/internal/middleware"

	"github.com/gin-gonic/gin"
)

// Handler 流水线API处理器
type Handler struct {
	engine      *services.PipelineEngine
	awxStreamer *services.AWXStreamer
}

// New 创建流水线处理器
func New(engine *services.PipelineEngine, awxStreamer *services.AWXStreamer) *Handler {
	return &Handler{
		engine:      engine,
		awxStreamer: awxStreamer,
	}
}

// RegisterRoutes 注册流水线路由
// v1 is /api/v1 group; admin is /api/v1/admin group with admin middlewares already applied.
func (h *Handler) RegisterRoutes(v1, admin *gin.RouterGroup) {
	// /api/v1/pipelines (public read, admin write)
	pipelineGroup := v1.Group("/pipelines")
	{
		// Templates (read-only for non-admin)
		pipelineGroup.GET("/templates", h.ListTemplates)
		pipelineGroup.GET("/templates/:id", h.GetTemplate)

		// Executions (read-only for non-admin)
		pipelineGroup.GET("/executions", h.GetExecutionHistory)
		pipelineGroup.GET("/executions/active", h.GetActiveExecutions)
		pipelineGroup.GET("/executions/:id", h.GetExecution)
		pipelineGroup.GET("/executions/:id/sop", h.GenerateSOPFlow)
		pipelineGroup.GET("/executions/:id/logs/stream", h.StreamLogs)
		pipelineGroup.GET("/executions/:id/progress/stream", h.StreamProgress)

		// AWX templates (read-only)
		pipelineGroup.GET("/awx/templates", h.ListAWXTemplates)
		pipelineGroup.GET("/awx/templates/:id", h.GetAWXTemplate)
		pipelineGroup.GET("/awx/inventories/:name/variables", h.GetInventoryVariables)
	}

	// Admin routes for pipeline management
	adminPipelineGroup := admin.Group("/pipelines")
	adminPipelineGroup.Use(middleware.AuditLogMiddleware())
	{
		// Template management
		adminPipelineGroup.POST("/templates", h.CreateTemplate)
		adminPipelineGroup.PUT("/templates/:id", h.UpdateTemplate)
		adminPipelineGroup.DELETE("/templates/:id", h.DeleteTemplate)

		// Execution management
		adminPipelineGroup.POST("/executions", h.StartExecution)
		adminPipelineGroup.POST("/executions/:id/pause", h.PauseExecution)
		adminPipelineGroup.POST("/executions/:id/resume", h.ResumeExecution)
		adminPipelineGroup.POST("/executions/:id/cancel", h.CancelExecution)
		adminPipelineGroup.POST("/executions/:id/rollback", h.RollbackExecution)
		adminPipelineGroup.POST("/executions/:id/run", h.RunPendingExecution)
		adminPipelineGroup.POST("/executions/:id/clone", h.CloneExecution)

		// AWX inventory management
		adminPipelineGroup.PUT("/awx/inventories/:name/variables", h.UpdateInventoryVariables)
	}
}

// getUserIDFromContext 从上下文获取用户ID
func getUserIDFromContext(c *gin.Context) uint64 {
	if userID, exists := c.Get("user_id"); exists {
		if id, ok := userID.(uint64); ok {
			return id
		}
	}
	return 0
}

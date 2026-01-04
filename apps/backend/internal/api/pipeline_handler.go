package api

import (
	"net/http"
	"strconv"

	"robusta-web/backend/internal/models"
	"robusta-web/backend/internal/services"

	"github.com/gin-gonic/gin"
)

// PipelineHandler 流水线API处理器
type PipelineHandler struct {
	engine *services.PipelineEngine
}

// NewPipelineHandler 创建流水线处理器
func NewPipelineHandler(engine *services.PipelineEngine) *PipelineHandler {
	return &PipelineHandler{engine: engine}
}

// 请求/响应结构

// CreateTemplateRequest 创建模板请求
type CreateTemplateRequest struct {
	Name        string                   `json:"name" binding:"required"`
	Description string                   `json:"description"`
	Stages      []models.StageDefinition `json:"stages" binding:"required,min=1"`
}

// StartExecutionRequest 启动执行请求
type StartExecutionRequest struct {
	TemplateID          uint64                 `json:"template_id" binding:"required"`
	ClusterID           uint64                 `json:"cluster_id" binding:"required"`
	Parameters          map[string]interface{} `json:"parameters"`
	TargetNodes         []string               `json:"target_nodes"`
	BatchSize           int                    `json:"batch_size"` // Deprecated: Use Batches logic if Batches is nil
	Batches             [][]string             `json:"batches"`    // New: Explicit batch definitions
	PauseBetweenBatches bool                   `json:"pause_between_batches"`
	AutoStart           bool                   `json:"auto_start"` // New: If false, creates pending execution without starting
}

// ApprovalRequest 审批请求
type ApprovalRequest struct {
	Notes string `json:"notes"`
}

// ListTemplates 获取流水线模板列表
// @Summary 获取流水线模板列表
// @Tags Pipeline
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Param keyword query string false "搜索关键字"
// @Success 200 {object} map[string]interface{}
// @Router /api/pipelines/templates [get]
func (h *PipelineHandler) ListTemplates(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	keyword := c.Query("keyword")

	templates, total, err := h.engine.ListTemplates(page, pageSize, keyword)
	if err != nil {
		InternalError(c, "", "获取模板列表失败: "+err.Error())
		return
	}

	totalPages := 0
	if pageSize > 0 {
		totalPages = int((total + int64(pageSize) - 1) / int64(pageSize))
	}

	c.JSON(http.StatusOK, gin.H{
		"data": templates,
		"pagination": gin.H{
			"total":       total,
			"page":        page,
			"page_size":   pageSize,
			"total_pages": totalPages,
		},
	})
}

// GetTemplate 获取单个模板详情
// @Summary 获取流水线模板详情
// @Tags Pipeline
// @Param id path int true "Template ID"
// @Success 200 {object} models.PipelineTemplate
// @Router /api/pipelines/templates/{id} [get]
func (h *PipelineHandler) GetTemplate(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "", "无效的模板ID")
		return
	}

	template, err := h.engine.GetTemplate(id)
	if err != nil {
		NotFound(c, "", "模板不存在")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": template})
}

// CreateTemplate 创建流水线模板
// @Summary 创建流水线模板
// @Tags Pipeline
// @Accept json
// @Param request body CreateTemplateRequest true "模板配置"
// @Success 201 {object} models.PipelineTemplate
// @Router /api/pipelines/templates [post]
func (h *PipelineHandler) CreateTemplate(c *gin.Context) {
	var req CreateTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "", "请求参数错误: "+err.Error())
		return
	}

	userID := getUserIDFromContext(c)
	template, err := h.engine.CreateTemplate(req.Name, req.Description, req.Stages, userID)
	if err != nil {
		InternalError(c, "", "创建模板失败: "+err.Error())
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": template})
}

// UpdateTemplateRequest 更新模板请求
type UpdateTemplateRequest struct {
	Name        string                   `json:"name" binding:"required"`
	Description string                   `json:"description"`
	Stages      []models.StageDefinition `json:"stages" binding:"required,min=1"`
}

// UpdateTemplate 更新流水线模板
// @Summary 更新流水线模板
// @Tags Pipeline
// @Accept json
// @Param id path int true "Template ID"
// @Param request body UpdateTemplateRequest true "模板配置"
// @Success 200 {object} models.PipelineTemplate
// @Router /api/pipelines/templates/{id} [put]
func (h *PipelineHandler) UpdateTemplate(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "", "无效的模板ID")
		return
	}

	var req UpdateTemplateRequest
	if err = c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "", "请求参数错误: "+err.Error())
		return
	}

	template, err := h.engine.UpdateTemplate(id, req.Name, req.Description, req.Stages)
	if err != nil {
		InternalError(c, "", "更新模板失败: "+err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": template})
}

// DeleteTemplate 删除流水线模板
// @Summary 删除流水线模板
// @Tags Pipeline
// @Param id path int true "Template ID"
// @Success 200 {object} map[string]string
// @Router /api/pipelines/templates/{id} [delete]
func (h *PipelineHandler) DeleteTemplate(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "", "无效的模板ID")
		return
	}

	if err := h.engine.DeleteTemplate(id); err != nil {
		InternalError(c, "", "删除模板失败: "+err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "模板已删除"})
}

// StartExecution 启动流水线执行
// @Summary 启动流水线执行
// @Tags Pipeline
// @Accept json
// @Param request body StartExecutionRequest true "执行配置"
// @Success 201 {object} models.PipelineExecution
// @Router /api/pipelines/executions [post]
func (h *PipelineHandler) StartExecution(c *gin.Context) {
	var req StartExecutionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "", "请求参数错误: "+err.Error())
		return
	}

	// 获取当前用户ID
	userID := getUserIDFromContext(c)

	execution, err := h.engine.StartExecution(c.Request.Context(), req.TemplateID, req.ClusterID, req.Parameters, userID, req.TargetNodes, req.BatchSize, req.Batches, req.PauseBetweenBatches, req.AutoStart)
	if err != nil {
		InternalError(c, "", "启动执行失败: "+err.Error())
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": execution})
}

// GetExecution 获取执行详情
// @Summary 获取流水线执行详情
// @Tags Pipeline
// @Param id path int true "Execution ID"
// @Success 200 {object} models.PipelineExecution
// @Router /api/pipelines/executions/{id} [get]
func (h *PipelineHandler) GetExecution(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "", "无效的执行ID")
		return
	}

	execution, err := h.engine.GetExecution(id)
	if err != nil {
		NotFound(c, "", "执行记录不存在")
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": execution})
}

// PauseExecution 暂停执行
// @Summary 暂停流水线执行
// @Tags Pipeline
// @Param id path int true "Execution ID"
// @Success 200 {object} map[string]string
// @Router /api/pipelines/executions/{id}/pause [post]
func (h *PipelineHandler) PauseExecution(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "", "无效的执行ID")
		return
	}

	if err := h.engine.PauseExecution(id); err != nil {
		InternalError(c, "", "暂停失败: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "执行已暂停"})
}

// ResumeExecution 恢复执行 (审批通过)
// @Summary 恢复流水线执行
// @Tags Pipeline
// @Accept json
// @Param id path int true "Execution ID"
// @Param request body ApprovalRequest true "审批信息"
// @Success 200 {object} map[string]string
// @Router /api/pipelines/executions/{id}/resume [post]
func (h *PipelineHandler) ResumeExecution(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "", "无效的执行ID")
		return
	}

	var req ApprovalRequest
	_ = c.ShouldBindJSON(&req) // notes是可选的

	userID := getUserIDFromContext(c)

	if err := h.engine.ResumeExecution(c.Request.Context(), id, userID, req.Notes); err != nil {
		InternalError(c, "", "恢复失败: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "执行已恢复"})
}

// CancelExecution 取消执行
// @Summary 取消流水线执行
// @Tags Pipeline
// @Param id path int true "Execution ID"
// @Success 200 {object} map[string]string
// @Router /api/pipelines/executions/{id}/cancel [post]
func (h *PipelineHandler) CancelExecution(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "", "无效的执行ID")
		return
	}

	if err := h.engine.CancelExecution(id); err != nil {
		InternalError(c, "", "取消失败: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "执行已取消"})
}

// RollbackExecution 触发回滚
// @Summary 触发流水线回滚
// @Tags Pipeline
// @Param id path int true "Execution ID"
// @Success 200 {object} map[string]string
// @Router /api/pipelines/executions/{id}/rollback [post]
func (h *PipelineHandler) RollbackExecution(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "", "无效的执行ID")
		return
	}

	if err := h.engine.RollbackExecution(c.Request.Context(), id); err != nil {
		InternalError(c, "", "回滚失败: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "回滚已触发"})
}

// RunPendingExecution 启动待执行的任务
// @Summary 启动待执行的流水线任务
// @Tags Pipeline
// @Param id path int true "Execution ID"
// @Success 200 {object} map[string]string
// @Router /api/pipelines/executions/{id}/run [post]
func (h *PipelineHandler) RunPendingExecution(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "", "无效的执行ID")
		return
	}

	if err := h.engine.RunPendingExecution(c.Request.Context(), id); err != nil {
		InternalError(c, "", "启动失败: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "任务已启动"})
}

// GetExecutionHistory 获取流水线执行历史
// @Summary 获取流水线执行历史
// @Tags Pipeline
// @Param cluster_id query int false "Cluster ID (可选，不传则获取全部)"
// @Param limit query int false "数量限制" default(50)
// @Success 200 {array} models.PipelineExecution
// @Router /api/pipelines/executions [get]
func (h *PipelineHandler) GetExecutionHistory(c *gin.Context) {
	limit := 50
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, errConv := strconv.Atoi(limitStr); errConv == nil && l > 0 {
			limit = l
		}
	}

	clusterIDStr := c.Query("cluster_id")

	// 如果有 cluster_id，获取指定集群的历史
	if clusterIDStr != "" {
		clusterID, err := strconv.ParseUint(clusterIDStr, 10, 64)
		if err != nil {
			BadRequest(c, "", "无效的cluster_id")
			return
		}
		executions, err := h.engine.GetExecutionHistory(clusterID, limit)
		if err != nil {
			InternalError(c, "", "获取历史失败: "+err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": executions})
		return
	}

	// 否则获取全部执行历史
	executions, err := h.engine.GetAllExecutions(limit)
	if err != nil {
		InternalError(c, "", "获取历史失败: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": executions})
}

// GetActiveExecutions 获取所有活跃的执行任务
// @Summary 获取活跃的流水线执行
// @Tags Pipeline
// @Param limit query int false "数量限制" default(50)
// @Success 200 {array} models.PipelineExecution
// @Router /api/pipelines/executions/active [get]
func (h *PipelineHandler) GetActiveExecutions(c *gin.Context) {
	limit := 50
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, errConv := strconv.Atoi(limitStr); errConv == nil && l > 0 {
			limit = l
		}
	}

	executions, err := h.engine.GetActiveExecutions(limit)
	if err != nil {
		InternalError(c, "", "获取活跃执行失败: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": executions})
}

// ListAWXTemplates 获取任务模板列表
// @Summary 获取任务模板列表（AWX Job Templates 等）
// @Tags Pipeline
// @Success 200 {array} services.JobTemplateInfo
// @Router /api/pipelines/awx/templates [get]
func (h *PipelineHandler) ListAWXTemplates(c *gin.Context) {
	templates, err := h.engine.ListJobTemplates(c.Request.Context())
	if err != nil {
		InternalError(c, "", "获取任务模板列表失败: "+err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": templates})
}

// GetAWXTemplate 获取单个任务模板详情
// @Summary 获取单个任务模板详情
// @Tags Pipeline
// @Param id path int true "Template ID"
// @Success 200 {object} services.JobTemplateInfo
// @Router /api/pipelines/awx/templates/{id} [get]
func (h *PipelineHandler) GetAWXTemplate(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		BadRequest(c, "", "无效的模板ID")
		return
	}

	template, err := h.engine.GetJobTemplate(c.Request.Context(), id)
	if err != nil {
		InternalError(c, "", "获取任务模板详情失败: "+err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": template})
}

// 辅助函数
func getUserIDFromContext(c *gin.Context) uint64 {
	if userID, exists := c.Get("user_id"); exists {
		if id, ok := userID.(uint64); ok {
			return id
		}
	}
	return 0
}

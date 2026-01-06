package http

import (
	"io"
	"net/http"
	"strconv"

	httpx "robusta-web/backend/internal/transport/httpx"

	"github.com/gin-gonic/gin"
)

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

// StartExecution 启动流水线执行
func (h *Handler) StartExecution(c *gin.Context) {
	var req StartExecutionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "", "请求参数错误: "+err.Error())
		return
	}

	userID := getUserIDFromContext(c)

	execution, err := h.engine.StartExecution(c.Request.Context(), req.TemplateID, req.ClusterID, req.Parameters, userID, req.TargetNodes, req.BatchSize, req.Batches, req.PauseBetweenBatches, req.AutoStart)
	if err != nil {
		httpx.InternalError(c, "", "启动执行失败: "+err.Error())
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": execution})
}

// GetExecution 获取执行详情
func (h *Handler) GetExecution(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httpx.BadRequest(c, "", "无效的执行ID")
		return
	}

	execution, err := h.engine.GetExecution(id)
	if err != nil {
		httpx.NotFound(c, "", "执行记录不存在")
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": execution})
}

// PauseExecution 暂停执行
func (h *Handler) PauseExecution(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httpx.BadRequest(c, "", "无效的执行ID")
		return
	}

	if err := h.engine.PauseExecution(id); err != nil {
		httpx.InternalError(c, "", "暂停失败: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "执行已暂停"})
}

// ResumeExecution 恢复执行 (审批通过)
func (h *Handler) ResumeExecution(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httpx.BadRequest(c, "", "无效的执行ID")
		return
	}

	var req ApprovalRequest
	_ = c.ShouldBindJSON(&req) // notes是可选的

	userID := getUserIDFromContext(c)

	if err := h.engine.ResumeExecution(c.Request.Context(), id, userID, req.Notes); err != nil {
		httpx.InternalError(c, "", "恢复失败: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "执行已恢复"})
}

// CancelExecution 取消执行
func (h *Handler) CancelExecution(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httpx.BadRequest(c, "", "无效的执行ID")
		return
	}

	if err := h.engine.CancelExecution(id); err != nil {
		httpx.InternalError(c, "", "取消失败: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "执行已取消"})
}

// RollbackExecution 触发回滚
func (h *Handler) RollbackExecution(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httpx.BadRequest(c, "", "无效的执行ID")
		return
	}

	if err := h.engine.RollbackExecution(c.Request.Context(), id); err != nil {
		httpx.InternalError(c, "", "回滚失败: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "回滚已触发"})
}

// RunPendingExecution 启动待执行的任务
func (h *Handler) RunPendingExecution(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httpx.BadRequest(c, "", "无效的执行ID")
		return
	}

	if err := h.engine.RunPendingExecution(c.Request.Context(), id); err != nil {
		httpx.InternalError(c, "", "启动失败: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "任务已启动"})
}

// CloneExecution 克隆任务 (Copy to New Pending)
func (h *Handler) CloneExecution(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httpx.BadRequest(c, "", "无效的执行ID")
		return
	}

	userID := getUserIDFromContext(c)
	execution, err := h.engine.CloneExecution(c.Request.Context(), id, userID)
	if err != nil {
		httpx.InternalError(c, "", "克隆失败: "+err.Error())
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": execution})
}

// GetExecutionHistory 获取流水线执行历史
func (h *Handler) GetExecutionHistory(c *gin.Context) {
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
			httpx.BadRequest(c, "", "无效的cluster_id")
			return
		}
		executions, err := h.engine.GetExecutionHistory(clusterID, limit)
		if err != nil {
			httpx.InternalError(c, "", "获取历史失败: "+err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": executions})
		return
	}

	// 否则获取全部执行历史
	executions, err := h.engine.GetAllExecutions(limit)
	if err != nil {
		httpx.InternalError(c, "", "获取历史失败: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": executions})
}

// GetActiveExecutions 获取所有活跃的执行任务
func (h *Handler) GetActiveExecutions(c *gin.Context) {
	limit := 50
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, errConv := strconv.Atoi(limitStr); errConv == nil && l > 0 {
			limit = l
		}
	}

	executions, err := h.engine.GetActiveExecutions(limit)
	if err != nil {
		httpx.InternalError(c, "", "获取活跃执行失败: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": executions})
}

// GenerateSOPFlow 生成 AI-SOP 流程定义
func (h *Handler) GenerateSOPFlow(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httpx.BadRequest(c, "", "无效的 Execution ID")
		return
	}

	sopFlow, err := h.engine.GenerateSOPFlow(c.Request.Context(), id)
	if err != nil {
		httpx.InternalError(c, "", "生成 SOP Flow 失败: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": sopFlow})
}

// StreamLogs SSE endpoint for raw log streaming
func (h *Handler) StreamLogs(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httpx.BadRequest(c, "", "Invalid Execution ID")
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Transfer-Encoding", "chunked")

	stream, err := h.awxStreamer.StreamJobLogs(c.Request.Context(), id)
	if err != nil {
		httpx.InternalError(c, "", "Failed to start log stream: "+err.Error())
		return
	}

	c.Stream(func(w io.Writer) bool {
		select {
		case line, ok := <-stream:
			if !ok {
				return false
			}
			c.SSEvent("log", line)
			return true
		case <-c.Request.Context().Done():
			return false
		}
	})
}

// StreamProgress SSE endpoint for structured task progress
func (h *Handler) StreamProgress(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httpx.BadRequest(c, "", "Invalid Execution ID")
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	stream, err := h.awxStreamer.StreamJobProgress(c.Request.Context(), id)
	if err != nil {
		httpx.InternalError(c, "", "Failed to start progress stream: "+err.Error())
		return
	}

	c.Stream(func(w io.Writer) bool {
		select {
		case node, ok := <-stream:
			if !ok {
				return false
			}
			c.SSEvent("task", node)
			return true
		case <-c.Request.Context().Done():
			return false
		}
	})
}

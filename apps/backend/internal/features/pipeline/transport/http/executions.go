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
// @Summary 启动或创建流水线作业
// @Description 根据指定的模板和集群启动一个新的流水线执行任务。支持传入自定义参数、目标节点列表以及分批执行策略（Batch Size 或显式 Batches）。通过 AutoStart 标志控制是立即触发执行，还是仅创建一个处于待命状态的任务记录。
// @Tags Admin,Pipeline,Executions
// @Accept json
// @Produce json
// @Param request body StartExecutionRequest true "作业启动配置"
// @Success 201 {object} object
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /admin/pipelines/executions [post]
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
// @Summary 获取流水线作业状态详情
// @Description 查询特定流水线执行任务的当前生存状态。返回结果包含各阶段的执行进度、作业耗时、输出摘要以及当前所处的批次信息。该接口是监控异步长耗时作业执行进度的主要数据来源。
// @Tags Pipeline,Executions
// @Produce json
// @Param id path int true "执行ID"
// @Success 200 {object} object
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Router /pipelines/executions/{id} [get]
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
// @Summary 暂停正在运行的作业
// @Description 立即中断当前正在执行的流水线任务。系统会记录当前的中断点，确保作业不再向后续节点下发新任务。常用于发现预警异常或需要临时人工干预介入的运维场景，为排查问题争取响应时间。
// @Tags Admin,Pipeline,Executions
// @Param id path int true "执行ID"
// @Success 200 {object} object
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /admin/pipelines/executions/{id}/pause [post]
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
// @Summary 恢复或审批通过作业
// @Description 恢复之前被暂停的流水线任务，或正式提交针对处于等待审批状态阶段的确认。用户可以提供审批备注供审计使用。执行恢复后，流水线将从断点处继续向下推进至下一个批次或最终完成。
// @Tags Admin,Pipeline,Executions
// @Accept json
// @Produce json
// @Param id path int true "执行ID"
// @Param request body ApprovalRequest false "审批/恢复备注"
// @Success 200 {object} object
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /admin/pipelines/executions/{id}/resume [post]
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
// @Summary 强制取消流水线任务
// @Description 彻底终止指定的流水线执行流程。由于该操作可能涉及底层AWX作业的强行中断，系统会尝试进行优雅清理，但状态将被标记为“已取消”。建议仅在确认作业无法继续运行或配置错误时执行此操作。
// @Tags Admin,Pipeline,Executions
// @Param id path int true "执行ID"
// @Success 200 {object} object
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /admin/pipelines/executions/{id}/cancel [post]
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
// @Summary 触发自动回滚流程
// @Description 针对已失败或异常的流水线任务执行回滚逻辑。回滚操作会根据模板中定义的逆向处理阶段对系统状态进行恢复，最大限度减少故障对生产环境的影响。回滚作业本身作为一个新的子任务进行生命周期管理和状态追踪。
// @Tags Admin,Pipeline,Executions
// @Param id path int true "执行ID"
// @Success 200 {object} object
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /admin/pipelines/executions/{id}/rollback [post]
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
// @Summary 启动待命状态的作业
// @Description 手动触发一个之前已创建但并未立即开始的流水线任务。该接口支持在完成所有前置资源准备和人工预检确认后，精准控制作业的下发时机，是实现变更窗口精细化管理的有效手段。
// @Tags Admin,Pipeline,Executions
// @Param id path int true "执行ID"
// @Success 200 {object} object
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /admin/pipelines/executions/{id}/run [post]
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
// @Summary 克隆历史作业任务
// @Description 以某个历史作业为蓝本，复制其所有的参数配置、目标节点及执行策略，创建一个全新的处于“待命”状态的执行任务。这在需要针对不同集群重复执行相同变更，或在回退失败后快速重试修正后的配置时非常高效。
// @Tags Admin,Pipeline,Executions
// @Param id path int true "原执行ID"
// @Success 201 {object} object
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /admin/pipelines/executions/{id}/clone [post]
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
// @Summary 查询作业执行历史
// @Description 检索流水线作业的历史记录列表。支持按集群ID进行过滤，以获取特定环境下的所有审计与执行记录。返回数据包含作业的起止时间、最终状态以及操作人，是进行合规性审计和故障回溯的核心数据接口。
// @Tags Pipeline,Executions
// @Produce json
// @Param cluster_id query int false "集群ID"
// @Param limit query int false "条数限制"
// @Success 200 {object} object
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /pipelines/executions [get]
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
// @Summary 获取当前活跃作业列表
// @Description 实时查询系统中所有正在运行、暂停中或处于审批等待状态的流水线任务。该接口展示了当前全平台的并发变更活动概况，帮助运维总控中心掌握集群资源的使用情况及潜在的变更冲突。
// @Tags Pipeline,Executions
// @Produce json
// @Param limit query int false "条数限制"
// @Success 200 {object} object
// @Failure 500 {object} httpx.ErrorResponse
// @Router /pipelines/executions/active [get]
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
// @Summary 生成AI引导的SOP流程图
// @Description 基于流水线执行的任务逻辑和当前上下文，通过AI模型动态生成标准作业程序（SOP）的图形化或流程化定义。该接口为用户提供可视化的操作指引，帮助理解复杂的自动化执行链路及关键决策分支。
// @Tags Pipeline,Executions
// @Produce json
// @Param id path int true "执行ID"
// @Success 200 {object} object
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /pipelines/executions/{id}/sop [get]
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
// @Summary 实时流式传输作业日志
// @Description 使用Server-Sent Events (SSE) 技术将流水线作业的底层原始执行日志实时推送到客户端。该接口能够展示来自AWX或K8s作业的逐行标准输出，让管理员能像在本地终端一样监控远程自动化任务的细节输出。
// @Tags Pipeline,Logs
// @Produce text/event-stream
// @Param id path int true "执行ID"
// @Success 200 {string} string "Event: log"
// @Router /pipelines/executions/{id}/logs/stream [get]
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
// @Summary 实时推送结构化执行进度
// @Description 建立流式连接以实时追踪流水线中各节点/任务的结构化状态变更。该接口会主动推送任务开始、成功、失败及进度更新消息，相比原始日志，它提供了更高维、更易于前端可视化的作业进度展现形式。
// @Tags Pipeline,Executions
// @Produce text/event-stream
// @Param id path int true "执行ID"
// @Success 200 {string} string "Event: task"
// @Router /pipelines/executions/{id}/progress/stream [get]
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

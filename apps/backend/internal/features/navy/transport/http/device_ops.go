package http

import (
	"context"
	"net/http"

	"robusta-web/backend/internal/features/navy/services"
	pipelineservice "robusta-web/backend/internal/features/pipeline/services"

	"github.com/gin-gonic/gin"
)

// BatchNodeOperationRequest 批量节点操作请求
type BatchNodeOperationRequest struct {
	CICodes []string `json:"ci_codes" binding:"required,min=1"`
}

// DrainNodesRequest Drain 操作请求
type DrainNodesRequest struct {
	CICodes          []string `json:"ci_codes" binding:"required,min=1"`
	Force            bool     `json:"force"`
	IgnoreDaemonsets bool     `json:"ignore_daemonsets"`
	DeleteLocalData  bool     `json:"delete_local_data"`
	Timeout          int      `json:"timeout"` // 秒
}

// TaintNodesRequest Taint 操作请求
type TaintNodesRequest struct {
	CICodes []string `json:"ci_codes" binding:"required,min=1"`
	Key     string   `json:"key" binding:"required"`
	Value   string   `json:"value"`
	Effect  string   `json:"effect" binding:"required,oneof=NoSchedule PreferNoSchedule NoExecute"`
	Action  string   `json:"action" binding:"required,oneof=add remove"`
}

// LabelNodesRequest Label 操作请求
type LabelNodesRequest struct {
	CICodes []string          `json:"ci_codes" binding:"required,min=1"`
	Labels  map[string]string `json:"labels" binding:"required,min=1"`
	Action  string            `json:"action" binding:"required,oneof=add remove"`
}

// helper: handle batch operations that return BatchOperationResult
func (h *Handler) handleBatchResultOp(c *gin.Context, opType services.ChangeOperationType, fn func(ctx context.Context, ciCodes []string) (*services.BatchOperationResult, error)) {
	var req BatchNodeOperationRequest
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": bindErr.Error()})
		return
	}

	var result *services.BatchOperationResult
	var opErr error

	_, err := h.changeManager.WithChange(
		c.Request.Context(),
		opType,
		req.CICodes,
		nil,
		func(ticketID string) error {
			result, opErr = fn(c.Request.Context(), req.CICodes)
			return opErr
		},
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if opErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": opErr.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

// helper: handle batch operations that return AWX Job handles
func (h *Handler) handleBatchJobOp(c *gin.Context, opType services.ChangeOperationType, fn func(ctx context.Context, ciCodes []string) (*pipelineservice.JobHandle, error), message string) {
	var req BatchNodeOperationRequest
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": bindErr.Error()})
		return
	}

	var handle *pipelineservice.JobHandle
	var opErr error

	ticketID, err := h.changeManager.WithChange(
		c.Request.Context(),
		opType,
		req.CICodes,
		nil,
		func(tid string) error {
			handle, opErr = fn(c.Request.Context(), req.CICodes)
			return opErr
		},
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if opErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": opErr.Error()})
		return
	}

	// Attach AWX job ID for async tracking
	if handle != nil && ticketID != "" {
		_ = h.changeManager.AttachAWXJob(ticketID, handle.JobID)
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"job_id":  handle.JobID,
			"message": message,
		},
	})
}

// CordonNodes 设置节点为不可调度
// @Summary 批量设置节点不可调度 (Cordon)
// @Description 针对指定的一组CI编码（设备标识），在K8s集群中将其对应的节点标记为不可调度状态。新的Pod将不会被指派到这些节点上，但已运行的Pod不受影响。此操作通常作为节点维护或下线流程的第一步。
// @Tags Navy,NodeOps
// @Accept json
// @Produce json
// @Param request body BatchNodeOperationRequest true "待操作节点CI列表"
// @Success 200 {object} gin.H{data=services.BatchOperationResult}
// @Failure 400 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /navy/nodes/cordon [post]
func (h *Handler) CordonNodes(c *gin.Context) {
	h.handleBatchResultOp(c, services.ChangeOpCordon, func(ctx context.Context, ciCodes []string) (*services.BatchOperationResult, error) {
		return h.deviceOpsService.CordonNodes(ctx, ciCodes)
	})
}

// UncordonNodes 设置节点为可调度
// @Summary 批量取消节点不可调度 (Uncordon)
// @Description 恢复指定CI编码对应节点的可调度状态。执行后，K8s调度器将重新开始向这些节点指派新的负载。该接口常用于维护任务结束后的资源恢复阶段，确保集群计算能力得到及时重新利用。
// @Tags Navy,NodeOps
// @Accept json
// @Produce json
// @Param request body BatchNodeOperationRequest true "待恢复节点CI列表"
// @Success 200 {object} gin.H{data=services.BatchOperationResult}
// @Failure 400 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /navy/nodes/uncordon [post]
func (h *Handler) UncordonNodes(c *gin.Context) {
	h.handleBatchResultOp(c, services.ChangeOpUncordon, func(ctx context.Context, ciCodes []string) (*services.BatchOperationResult, error) {
		return h.deviceOpsService.UncordonNodes(ctx, ciCodes)
	})
}

// DrainNodes 驱逐节点上的 Pod
// @Summary 批量驱逐节点Pod (Drain)
// @Description 安全地从指定节点上驱逐所有运行中的Pod。支持配置是否强制执行、是否忽略守护进程集（DaemonSets）以及是否删除本地数据。该接口集成了变更管理流程，确保在大规模驱逐操作前进行必要的合规性校验。
// @Tags Navy,NodeOps
// @Accept json
// @Produce json
// @Param request body DrainNodesRequest true "驱逐配置及节点列表"
// @Success 200 {object} gin.H{data=services.BatchOperationResult}
// @Failure 400 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /navy/nodes/drain [post]
func (h *Handler) DrainNodes(c *gin.Context) {
	var req DrainNodesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var result *services.BatchOperationResult
	var opErr error

	ticketID, err := h.changeManager.WithChange(
		c.Request.Context(),
		services.ChangeOpDrain,
		req.CICodes,
		map[string]any{"force": req.Force, "ignore_daemonsets": req.IgnoreDaemonsets},
		func(tid string) error {
			result, opErr = h.deviceOpsService.DrainNodes(c.Request.Context(), req.CICodes, services.DrainOptions{
				Force:            req.Force,
				IgnoreDaemonsets: req.IgnoreDaemonsets,
				DeleteLocalData:  req.DeleteLocalData,
				Timeout:          req.Timeout,
			})
			return opErr
		},
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if opErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": opErr.Error()})
		return
	}

	// Attach drain IDs to the change ticket for async tracking
	if result != nil && ticketID != "" {
		var drainIDs []string
		for _, r := range result.Results {
			if r.DrainID != "" {
				drainIDs = append(drainIDs, r.DrainID)
			}
		}
		if len(drainIDs) > 0 {
			_ = h.changeManager.AttachDrainIDs(ticketID, drainIDs)
		}
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

// TaintNodes 添加或移除节点 Taint
// @Summary 批量管理节点污点 (Taint)
// @Description 在指定的K8s节点上批量添加或移除特定的污点。管理员需定义污点的Key、Value以及调度效果（如NoSchedule）。该功能通过变更管理系统进行审计，是实现集群逻辑隔离和节点生命周期管理的常用技术手段。
// @Tags Navy,NodeOps
// @Accept json
// @Produce json
// @Param request body TaintNodesRequest true "污点配置参数"
// @Success 200 {object} gin.H{data=services.BatchOperationResult}
// @Failure 400 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /navy/nodes/taint [post]
func (h *Handler) TaintNodes(c *gin.Context) {
	var req TaintNodesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var result *services.BatchOperationResult
	var opErr error

	_, err := h.changeManager.WithChange(
		c.Request.Context(),
		services.ChangeOpTaint,
		req.CICodes,
		map[string]any{"key": req.Key, "value": req.Value, "effect": req.Effect, "action": req.Action},
		func(ticketID string) error {
			result, opErr = h.deviceOpsService.TaintNodes(c.Request.Context(), req.CICodes, services.TaintOperation{
				Key:    req.Key,
				Value:  req.Value,
				Effect: req.Effect,
				Action: req.Action,
			})
			return opErr
		},
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if opErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": opErr.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

// LabelNodes 添加或移除节点 Label
// @Summary 批量管理节点标签 (Label)
// @Description 对一组指定的节点执行标签的批量更新或删除操作。用户可以一次性定义多个KV对。此操作通过LabelOperation模型进行解析，确保标签变更的一致性，广泛应用于节点分类、调度策略调整及自动化运维打标场景。
// @Tags Navy,NodeOps
// @Accept json
// @Produce json
// @Param request body LabelNodesRequest true "标签配置参数"
// @Success 200 {object} gin.H{data=services.BatchOperationResult}
// @Failure 400 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /navy/nodes/label [post]
func (h *Handler) LabelNodes(c *gin.Context) {
	var req LabelNodesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var result *services.BatchOperationResult
	var opErr error

	_, err := h.changeManager.WithChange(
		c.Request.Context(),
		services.ChangeOpLabel,
		req.CICodes,
		map[string]any{"labels": req.Labels, "action": req.Action},
		func(ticketID string) error {
			result, opErr = h.deviceOpsService.LabelNodes(c.Request.Context(), req.CICodes, services.LabelOperation{
				Labels: req.Labels,
				Action: req.Action,
			})
			return opErr
		},
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if opErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": opErr.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

// ShutdownNodes 关机节点 (AWX)
// @Summary 批量执行节点关机
// @Description 通过调用底层的AWX自动化作业流程，对指定的物理或虚拟节点执行硬关机操作。此请求会被记录在变更管理系统中，并返回一个异步作业句柄。常用于紧急维护、能源优化或数据中心退役等运维任务。
// @Tags Navy,NodeOps
// @Accept json
// @Produce json
// @Param request body BatchNodeOperationRequest true "待关机节点列表"
// @Success 200 {object} gin.H{data=object}
// @Failure 400 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /navy/nodes/shutdown [post]
func (h *Handler) ShutdownNodes(c *gin.Context) {
	h.handleBatchJobOp(c, services.ChangeOpShutdown, func(ctx context.Context, ciCodes []string) (*pipelineservice.JobHandle, error) {
		return h.deviceOpsService.ShutdownNodes(ctx, ciCodes)
	}, "关机任务已提交")
}

// RebootNodes 重启节点 (AWX)
// @Summary 批量执行节点重启
// @Description 触发预定义的自动化脚本对选定节点进行批量重启。该操作通过AWX作业引擎异步执行，确保节点在重启过程中能够遵循标准的预检和后验逻辑。接口返回追踪ID，便于管理员实时监控重启作业的执行进度。
// @Tags Navy,NodeOps
// @Accept json
// @Produce json
// @Param request body BatchNodeOperationRequest true "待重启节点列表"
// @Success 200 {object} gin.H{data=object}
// @Failure 400 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /navy/nodes/reboot [post]
func (h *Handler) RebootNodes(c *gin.Context) {
	h.handleBatchJobOp(c, services.ChangeOpReboot, func(ctx context.Context, ciCodes []string) (*pipelineservice.JobHandle, error) {
		return h.deviceOpsService.RebootNodes(ctx, ciCodes)
	}, "重启任务已提交")
}

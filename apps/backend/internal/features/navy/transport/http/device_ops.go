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
func (h *Handler) CordonNodes(c *gin.Context) {
	h.handleBatchResultOp(c, services.ChangeOpCordon, func(ctx context.Context, ciCodes []string) (*services.BatchOperationResult, error) {
		return h.deviceOpsService.CordonNodes(ctx, ciCodes)
	})
}

// UncordonNodes 设置节点为可调度
func (h *Handler) UncordonNodes(c *gin.Context) {
	h.handleBatchResultOp(c, services.ChangeOpUncordon, func(ctx context.Context, ciCodes []string) (*services.BatchOperationResult, error) {
		return h.deviceOpsService.UncordonNodes(ctx, ciCodes)
	})
}

// DrainNodes 驱逐节点上的 Pod
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
func (h *Handler) ShutdownNodes(c *gin.Context) {
	h.handleBatchJobOp(c, services.ChangeOpShutdown, func(ctx context.Context, ciCodes []string) (*pipelineservice.JobHandle, error) {
		return h.deviceOpsService.ShutdownNodes(ctx, ciCodes)
	}, "关机任务已提交")
}

// RebootNodes 重启节点 (AWX)
func (h *Handler) RebootNodes(c *gin.Context) {
	h.handleBatchJobOp(c, services.ChangeOpReboot, func(ctx context.Context, ciCodes []string) (*pipelineservice.JobHandle, error) {
		return h.deviceOpsService.RebootNodes(ctx, ciCodes)
	}, "重启任务已提交")
}

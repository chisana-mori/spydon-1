package http

import (
	"net/http"

	"robusta-web/backend/internal/features/navy/services"

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

// CordonNodes 设置节点为不可调度
func (h *Handler) CordonNodes(c *gin.Context) {
	var req BatchNodeOperationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.deviceOpsService.CordonNodes(c.Request.Context(), req.CICodes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

// UncordonNodes 设置节点为可调度
func (h *Handler) UncordonNodes(c *gin.Context) {
	var req BatchNodeOperationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.deviceOpsService.UncordonNodes(c.Request.Context(), req.CICodes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

// DrainNodes 驱逐节点上的 Pod
func (h *Handler) DrainNodes(c *gin.Context) {
	var req DrainNodesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.deviceOpsService.DrainNodes(c.Request.Context(), req.CICodes, services.DrainOptions{
		Force:            req.Force,
		IgnoreDaemonsets: req.IgnoreDaemonsets,
		DeleteLocalData:  req.DeleteLocalData,
		Timeout:          req.Timeout,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
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

	result, err := h.deviceOpsService.TaintNodes(c.Request.Context(), req.CICodes, services.TaintOperation{
		Key:    req.Key,
		Value:  req.Value,
		Effect: req.Effect,
		Action: req.Action,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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

	result, err := h.deviceOpsService.LabelNodes(c.Request.Context(), req.CICodes, services.LabelOperation{
		Labels: req.Labels,
		Action: req.Action,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

// ShutdownNodes 关机节点 (AWX)
func (h *Handler) ShutdownNodes(c *gin.Context) {
	var req BatchNodeOperationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	handle, err := h.deviceOpsService.ShutdownNodes(c.Request.Context(), req.CICodes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"job_id":  handle.JobID,
			"message": "关机任务已提交",
		},
	})
}

// RebootNodes 重启节点 (AWX)
func (h *Handler) RebootNodes(c *gin.Context) {
	var req BatchNodeOperationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	handle, err := h.deviceOpsService.RebootNodes(c.Request.Context(), req.CICodes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"job_id":  handle.JobID,
			"message": "重启任务已提交",
		},
	})
}

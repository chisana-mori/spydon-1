package api

import (
	"net/http"

	"robusta-web/backend/internal/services"

	"github.com/gin-gonic/gin"
)

// DeviceOperationsHandler 设备批量操作处理器
type DeviceOperationsHandler struct {
	service *services.DeviceOperationsService
}

// NewDeviceOperationsHandler 创建设备操作处理器
func NewDeviceOperationsHandler(service *services.DeviceOperationsService) *DeviceOperationsHandler {
	return &DeviceOperationsHandler{
		service: service,
	}
}

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
// @Summary Cordon 节点
// @Tags DeviceOperations
// @Accept json
// @Produce json
// @Param body body BatchNodeOperationRequest true "请求体"
// @Success 200 {object} services.BatchOperationResult
// @Router /navy/device-ops/cordon [post]
func (h *DeviceOperationsHandler) CordonNodes(c *gin.Context) {
	var req BatchNodeOperationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.service.CordonNodes(c.Request.Context(), req.CICodes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

// UncordonNodes 设置节点为可调度
// @Summary Uncordon 节点
// @Tags DeviceOperations
// @Accept json
// @Produce json
// @Param body body BatchNodeOperationRequest true "请求体"
// @Success 200 {object} services.BatchOperationResult
// @Router /navy/device-ops/uncordon [post]
func (h *DeviceOperationsHandler) UncordonNodes(c *gin.Context) {
	var req BatchNodeOperationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.service.UncordonNodes(c.Request.Context(), req.CICodes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

// DrainNodes 驱逐节点上的 Pod
// @Summary Drain 节点
// @Tags DeviceOperations
// @Accept json
// @Produce json
// @Param body body DrainNodesRequest true "请求体"
// @Success 200 {object} services.BatchOperationResult
// @Router /navy/device-ops/drain [post]
func (h *DeviceOperationsHandler) DrainNodes(c *gin.Context) {
	var req DrainNodesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.service.DrainNodes(c.Request.Context(), req.CICodes, services.DrainOptions{
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
// @Summary Taint 节点
// @Tags DeviceOperations
// @Accept json
// @Produce json
// @Param body body TaintNodesRequest true "请求体"
// @Success 200 {object} services.BatchOperationResult
// @Router /navy/device-ops/taint [post]
func (h *DeviceOperationsHandler) TaintNodes(c *gin.Context) {
	var req TaintNodesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.service.TaintNodes(c.Request.Context(), req.CICodes, services.TaintOperation{
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
// @Summary Label 节点
// @Tags DeviceOperations
// @Accept json
// @Produce json
// @Param body body LabelNodesRequest true "请求体"
// @Success 200 {object} services.BatchOperationResult
// @Router /navy/device-ops/label [post]
func (h *DeviceOperationsHandler) LabelNodes(c *gin.Context) {
	var req LabelNodesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.service.LabelNodes(c.Request.Context(), req.CICodes, services.LabelOperation{
		Labels: req.Labels,
		Action: req.Action,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

// ShutdownNodes 关机节点
// @Summary 关机节点 (AWX)
// @Tags DeviceOperations
// @Accept json
// @Produce json
// @Param body body BatchNodeOperationRequest true "请求体"
// @Success 200 {object} map[string]int
// @Router /navy/device-ops/shutdown [post]
func (h *DeviceOperationsHandler) ShutdownNodes(c *gin.Context) {
	var req BatchNodeOperationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	handle, err := h.service.ShutdownNodes(c.Request.Context(), req.CICodes)
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

// RebootNodes 重启节点
// @Summary 重启节点 (AWX)
// @Tags DeviceOperations
// @Accept json
// @Produce json
// @Param body body BatchNodeOperationRequest true "请求体"
// @Success 200 {object} map[string]int
// @Router /navy/device-ops/reboot [post]
func (h *DeviceOperationsHandler) RebootNodes(c *gin.Context) {
	var req BatchNodeOperationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	handle, err := h.service.RebootNodes(c.Request.Context(), req.CICodes)
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

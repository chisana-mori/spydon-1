package api

import (
	"net/http"

	"robusta-web/backend/internal/services"

	"github.com/gin-gonic/gin"
)

// K8sNodeManageHandler K8s 节点标签/污点管理处理器
type K8sNodeManageHandler struct {
	service *services.K8sNodeManageService
}

// NewK8sNodeManageHandler 创建处理器
func NewK8sNodeManageHandler(service *services.K8sNodeManageService) *K8sNodeManageHandler {
	return &K8sNodeManageHandler{service: service}
}

// GetNodeLabelsAndTaints 获取节点的实时标签和污点
// GET /navy/k8s-nodes/labels-taints?cluster=ny-3&ciCode=ny-3-worker2
func (h *K8sNodeManageHandler) GetNodeLabelsAndTaints(c *gin.Context) {
	clusterName := c.Query("cluster")
	ciCode := c.Query("ciCode")

	if clusterName == "" || ciCode == "" {
		BadRequest(c, "MISSING_PARAMS", "cluster 和 ciCode 是必需的")
		return
	}

	resp, err := h.service.GetNodeLabelsAndTaints(c.Request.Context(), clusterName, ciCode)
	if err != nil {
		InternalError(c, "GET_NODE_LABELS_TAINTS_ERROR", err.Error())
		return
	}

	Success(c, resp)
}

// ListClusterNodes 列出集群的所有节点及其标签/污点
// GET /navy/k8s-nodes?cluster=ny-3
func (h *K8sNodeManageHandler) ListClusterNodes(c *gin.Context) {
	clusterName := c.Query("cluster")
	if clusterName == "" {
		BadRequest(c, "MISSING_CLUSTER", "cluster 是必需的")
		return
	}

	resp, err := h.service.ListClusterNodes(c.Request.Context(), clusterName)
	if err != nil {
		InternalError(c, "LIST_CLUSTER_NODES_ERROR", err.Error())
		return
	}

	Success(c, resp)
}

// AddLabel 添加节点标签
// POST /navy/k8s-nodes/labels
func (h *K8sNodeManageHandler) AddLabel(c *gin.Context) {
	var req services.AddLabelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "INVALID_REQUEST", err.Error())
		return
	}

	if err := h.service.AddLabel(c.Request.Context(), &req); err != nil {
		InternalError(c, "ADD_LABEL_ERROR", err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "标签添加成功",
	})
}

// RemoveLabel 删除节点标签
// DELETE /navy/k8s-nodes/labels
func (h *K8sNodeManageHandler) RemoveLabel(c *gin.Context) {
	var req services.RemoveLabelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "INVALID_REQUEST", err.Error())
		return
	}

	if err := h.service.RemoveLabel(c.Request.Context(), &req); err != nil {
		InternalError(c, "REMOVE_LABEL_ERROR", err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "标签删除成功",
	})
}

// AddTaint 添加节点污点
// POST /navy/k8s-nodes/taints
func (h *K8sNodeManageHandler) AddTaint(c *gin.Context) {
	var req services.AddTaintRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "INVALID_REQUEST", err.Error())
		return
	}

	if err := h.service.AddTaint(c.Request.Context(), &req); err != nil {
		InternalError(c, "ADD_TAINT_ERROR", err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "污点添加成功",
	})
}

// RemoveTaint 删除节点污点
// DELETE /navy/k8s-nodes/taints
func (h *K8sNodeManageHandler) RemoveTaint(c *gin.Context) {
	var req services.RemoveTaintRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "INVALID_REQUEST", err.Error())
		return
	}

	if err := h.service.RemoveTaint(c.Request.Context(), &req); err != nil {
		InternalError(c, "REMOVE_TAINT_ERROR", err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "污点删除成功",
	})
}

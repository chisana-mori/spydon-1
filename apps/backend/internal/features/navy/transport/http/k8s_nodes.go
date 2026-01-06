package http

import (
	"net/http"

	"robusta-web/backend/internal/features/navy/services"
	"robusta-web/backend/internal/transport/httpx"

	"github.com/gin-gonic/gin"
)

// GetNodeLabelsAndTaints 获取节点的实时标签和污点
// GET /navy/k8s-nodes/labels-taints?cluster=ny-3&ciCode=ny-3-worker2
func (h *Handler) GetNodeLabelsAndTaints(c *gin.Context) {
	clusterName := c.Query("cluster")
	ciCode := c.Query("ciCode")

	if clusterName == "" || ciCode == "" {
		httpx.BadRequest(c, "MISSING_PARAMS", "cluster 和 ciCode 是必需的")
		return
	}

	resp, err := h.k8sNodeManageService.GetNodeLabelsAndTaints(c.Request.Context(), clusterName, ciCode)
	if err != nil {
		httpx.InternalError(c, "GET_NODE_LABELS_TAINTS_ERROR", err.Error())
		return
	}

	httpx.Success(c, resp)
}

// ListClusterNodes 列出集群的所有节点及其标签/污点
// GET /navy/k8s-nodes?cluster=ny-3
func (h *Handler) ListClusterNodes(c *gin.Context) {
	clusterName := c.Query("cluster")
	if clusterName == "" {
		httpx.BadRequest(c, "MISSING_CLUSTER", "cluster 是必需的")
		return
	}

	resp, err := h.k8sNodeManageService.ListClusterNodes(c.Request.Context(), clusterName)
	if err != nil {
		httpx.InternalError(c, "LIST_CLUSTER_NODES_ERROR", err.Error())
		return
	}

	httpx.Success(c, resp)
}

// AddLabel 添加节点标签
// POST /navy/k8s-nodes/labels
func (h *Handler) AddLabel(c *gin.Context) {
	var req services.AddLabelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "INVALID_REQUEST", err.Error())
		return
	}

	if err := h.k8sNodeManageService.AddLabel(c.Request.Context(), &req); err != nil {
		httpx.InternalError(c, "ADD_LABEL_ERROR", err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "标签添加成功",
	})
}

// RemoveLabel 删除节点标签
// DELETE /navy/k8s-nodes/labels
func (h *Handler) RemoveLabel(c *gin.Context) {
	var req services.RemoveLabelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "INVALID_REQUEST", err.Error())
		return
	}

	if err := h.k8sNodeManageService.RemoveLabel(c.Request.Context(), &req); err != nil {
		httpx.InternalError(c, "REMOVE_LABEL_ERROR", err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "标签删除成功",
	})
}

// AddTaint 添加节点污点
// POST /navy/k8s-nodes/taints
func (h *Handler) AddTaint(c *gin.Context) {
	var req services.AddTaintRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "INVALID_REQUEST", err.Error())
		return
	}

	if err := h.k8sNodeManageService.AddTaint(c.Request.Context(), &req); err != nil {
		httpx.InternalError(c, "ADD_TAINT_ERROR", err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "污点添加成功",
	})
}

// RemoveTaint 删除节点污点
// DELETE /navy/k8s-nodes/taints
func (h *Handler) RemoveTaint(c *gin.Context) {
	var req services.RemoveTaintRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "INVALID_REQUEST", err.Error())
		return
	}

	if err := h.k8sNodeManageService.RemoveTaint(c.Request.Context(), &req); err != nil {
		httpx.InternalError(c, "REMOVE_TAINT_ERROR", err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "污点删除成功",
	})
}

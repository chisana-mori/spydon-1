package http

import (
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

package http

import (
	"robusta-web/backend/internal/features/navy/services"
	"robusta-web/backend/internal/transport/httpx"

	"github.com/gin-gonic/gin"
)

// =============================================================================
// K8sNodeHandler - K8s 节点管理处理器
// =============================================================================

// K8sNodeHandler 处理 K8s 节点管理相关的 HTTP 请求
type K8sNodeHandler struct {
	svc *services.K8sNodeManageService
}

// NewK8sNodeHandler 创建 K8sNodeHandler
func NewK8sNodeHandler(svc *services.K8sNodeManageService) *K8sNodeHandler {
	return &K8sNodeHandler{svc: svc}
}

// RegisterRoutes 注册 K8s 节点管理路由
func (h *K8sNodeHandler) RegisterRoutes(navyGroup *gin.RouterGroup) {
	k8s := navyGroup.Group(RouteGroupK8sNodes)
	{
		k8s.GET("", h.ListClusterNodes)
		k8s.GET(RouteLabelsAndTaints, h.GetNodeLabelsAndTaints)
	}
}

// GetNodeLabelsAndTaints 获取节点的实时标签和污点
// GET /navy/k8s-nodes/labels-taints?cluster=ny-3&ciCode=ny-3-worker2
func (h *K8sNodeHandler) GetNodeLabelsAndTaints(c *gin.Context) {
	clusterName := c.Query("cluster")
	ciCode := c.Query("ciCode")

	if clusterName == "" || ciCode == "" {
		httpx.BadRequest(c, "MISSING_PARAMS", "cluster 和 ciCode 是必需的")
		return
	}

	resp, err := h.svc.GetNodeLabelsAndTaints(c.Request.Context(), clusterName, ciCode)
	if err != nil {
		httpx.InternalError(c, "GET_NODE_LABELS_TAINTS_ERROR", err.Error())
		return
	}

	httpx.Success(c, resp)
}

// ListClusterNodes 列出集群的所有节点及其标签/污点
// GET /navy/k8s-nodes?cluster=ny-3
func (h *K8sNodeHandler) ListClusterNodes(c *gin.Context) {
	clusterName := c.Query("cluster")
	if clusterName == "" {
		httpx.BadRequest(c, "MISSING_CLUSTER", "cluster 是必需的")
		return
	}

	resp, err := h.svc.ListClusterNodes(c.Request.Context(), clusterName)
	if err != nil {
		httpx.InternalError(c, "LIST_CLUSTER_NODES_ERROR", err.Error())
		return
	}

	httpx.Success(c, resp)
}

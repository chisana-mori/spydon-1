package http

import (
	"robusta-web/backend/internal/models"
	httpx "robusta-web/backend/internal/transport/httpx"

	"github.com/gin-gonic/gin"
)

// GetClustersSummary 获取集群概览
func (h *Handler) GetClustersSummary(c *gin.Context) {
	summary, err := h.clusterService.GetClustersSummary()
	if err != nil {
		httpx.InternalError(c, "GET_CLUSTERS_SUMMARY_ERROR", "获取集群概览失败")
		return
	}

	httpx.Success(c, summary)
}

// GetClusters 获取集群列表
func (h *Handler) GetClusters(c *gin.Context) {
	var query struct {
		httpx.PaginationQuery
		Status string `form:"status"`
	}
	if derr := httpx.BindQuery(c, &query); derr != nil {
		httpx.AbortWithDomainError(c, derr)
		return
	}
	if derr := query.PaginationQuery.Validate(); derr != nil {
		httpx.AbortWithDomainError(c, derr)
		return
	}
	params := query.PaginationQuery.ToParams()

	clusters, total, err := h.clusterService.GetClusters(params.Page, params.PageSize, query.Status)
	if err != nil {
		httpx.InternalError(c, "GET_CLUSTERS_ERROR", "获取集群列表失败")
		return
	}

	pagination := httpx.NewPagination(params.Page, params.PageSize, total)
	pagination.Sort = params.Sort
	httpx.SuccessPaginated(c, clusters, pagination)
}

// GetCluster 获取单个集群详情
func (h *Handler) GetCluster(c *gin.Context) {
	var path struct {
		ID string `uri:"id" binding:"required"`
	}
	if derr := httpx.BindURI(c, &path); derr != nil {
		httpx.AbortWithDomainError(c, derr)
		return
	}

	cluster, err := h.clusterService.GetClusterByID(path.ID)
	if err != nil {
		httpx.NotFound(c, "CLUSTER_NOT_FOUND", "集群不存在")
		return
	}

	httpx.Success(c, cluster)
}

// GetClusterNodes 获取集群节点列表
func (h *Handler) GetClusterNodes(c *gin.Context) {
	var path struct {
		ID string `uri:"id" binding:"required"`
	}
	if derr := httpx.BindURI(c, &path); derr != nil {
		httpx.AbortWithDomainError(c, derr)
		return
	}

	nodes, err := h.clusterService.GetClusterNodes(c.Request.Context(), path.ID)
	if err != nil {
		httpx.InternalError(c, "GET_CLUSTER_NODES_ERROR", "获取集群节点失败: "+err.Error())
		return
	}

	httpx.Success(c, gin.H{"data": nodes})
}

// CreateCluster 创建集群（管理员功能）
func (h *Handler) CreateCluster(c *gin.Context) {
	var req struct {
		Name          string `json:"name" binding:"required"`
		ClusterID     string `json:"cluster_id"`
		Description   string `json:"description"`
		Config        string `json:"kube_config"` // KubeConfig
		PrometheusURL string `json:"prometheus_url"`
		Status        string `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "INVALID_REQUEST", "请求参数无效")
		return
	}

	cluster := &models.Cluster{
		Name:          req.Name,
		ClusterID:     req.ClusterID,
		Description:   req.Description,
		Config:        models.KiteSecretString(req.Config),
		PrometheusURL: req.PrometheusURL,
		Status:        req.Status,
	}

	if cluster.Status == "" {
		cluster.Status = "active"
	}

	if err := h.clusterService.CreateCluster(cluster); err != nil {
		httpx.InternalError(c, "CREATE_CLUSTER_ERROR", "创建集群失败: "+err.Error())
		return
	}

	httpx.SuccessWithMessage(c, "集群创建成功", cluster)
}

// UpdateCluster 更新集群（管理员功能）
func (h *Handler) UpdateCluster(c *gin.Context) {
	var path struct {
		ID string `uri:"id" binding:"required"`
	}
	if derr := httpx.BindURI(c, &path); derr != nil {
		httpx.AbortWithDomainError(c, derr)
		return
	}

	var req struct {
		Description   string `json:"description"`
		Config        string `json:"kube_config"` // KubeConfig
		PrometheusURL string `json:"prometheus_url"`
		Status        string `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "INVALID_REQUEST", "请求参数无效")
		return
	}

	cluster := &models.Cluster{
		Name:          path.ID, // Using Name as ID based on existing logic
		Description:   req.Description,
		Config:        models.KiteSecretString(req.Config),
		PrometheusURL: req.PrometheusURL,
		Status:        req.Status,
	}

	if err := h.clusterService.UpdateCluster(cluster); err != nil {
		httpx.InternalError(c, "UPDATE_CLUSTER_ERROR", "更新集群失败: "+err.Error())
		return
	}

	httpx.SuccessWithMessage(c, "集群更新成功", cluster)
}

// DeleteCluster 删除集群（管理员功能）
func (h *Handler) DeleteCluster(c *gin.Context) {
	var path struct {
		ID string `uri:"id" binding:"required"`
	}
	if derr := httpx.BindURI(c, &path); derr != nil {
		httpx.AbortWithDomainError(c, derr)
		return
	}

	if err := h.clusterService.DeleteCluster(path.ID); err != nil {
		httpx.InternalError(c, "DELETE_CLUSTER_ERROR", "删除集群失败")
		return
	}

	httpx.SuccessWithMessage(c, "集群删除成功", nil)
}

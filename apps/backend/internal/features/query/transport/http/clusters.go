package http

import (
	"robusta-web/backend/internal/models"
	httpx "robusta-web/backend/internal/transport/httpx"

	"github.com/gin-gonic/gin"
)

// GetClustersSummary 获取集群概览
// @Summary 获取多集群统计概览
// @Description 汇总查询系统中所有已纳管K8s集群的整体健康状况。返回结果包含集群总数、处于活跃状态的集群比例以及各集群的资源压力概况。该数据通常用于全局大盘展示，让管理员能一眼识别出存在潜在风险的集群环境。
// @Tags Query,Clusters
// @Produce json
// @Success 200 {object} httpx.Response{data=object}
// @Failure 500 {object} httpx.ErrorResponse
// @Router /clusters/summary [get]
func (h *Handler) GetClustersSummary(c *gin.Context) {
	summary, err := h.clusterService.GetClustersSummary()
	if err != nil {
		httpx.InternalError(c, "GET_CLUSTERS_SUMMARY_ERROR", "获取集群概览失败")
		return
	}

	httpx.Success(c, summary)
}

// GetClusters 获取集群列表
// @Summary 分页获取集群列表
// @Description 从数据库中检索当前系统纳管的所有K8s集群的基本信息。支持按集群名称或运行状态进行过滤，并提供分页展示功能。该接口是集群资产管理的基础，展示了各集群的唯一ID、名称、描述及其对应的Prometheus监控地址。
// @Tags Query,Clusters
// @Produce json
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Param status query string false "集群状态"
// @Success 200 {object} httpx.Response{data=[]models.Cluster}
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /clusters [get]
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
// @Summary 获取特定集群详情
// @Description 通过集群ID或名称获取其完整的配置细节，但不包含敏感的KubeConfig全文。返回结果涵盖了集群的标签定义、监控接入配置以及其健康状态。此接口支撑了集群详情页的数据加载，帮助运维人员深入了解单一集群的配置基准。
// @Tags Query,Clusters
// @Produce json
// @Param id path string true "集群名称或ID"
// @Success 200 {object} httpx.Response{data=models.Cluster}
// @Failure 404 {object} httpx.ErrorResponse
// @Router /clusters/{id} [get]
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
// @Summary 查询集群节点信息
// @Description 通过直接调用K8s API Server或缓存数据，实时检索指定集群下的所有节点（Node）清单。返回结果包含各节点的名称、角色分类、CPU与内存负载、内部IP以及对应的健康状态，是进行大规模节点批量运维操作的前置数据准备接口。
// @Tags Query,Clusters
// @Produce json
// @Param id path string true "集群名称或ID"
// @Success 200 {object} httpx.Response{data=[]object}
// @Failure 500 {object} httpx.ErrorResponse
// @Router /clusters/{id}/nodes [get]
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
// @Summary 注册新K8s集群
// @Description 向系统中添加一个新的K8s集群实例。管理员需要提供集群名称、KubeConfig认证信息以及关联的Prometheus监控地址。创建后，系统将尝试与集群建立连接并初始化基础的告警同步与资源采集流程。
// @Tags Admin,Clusters
// @Accept json
// @Produce json
// @Param request body object true "集群创建参数"
// @Success 200 {object} httpx.Response{data=models.Cluster}
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /admin/clusters [post]
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
// @Summary 修改集群接入配置
// @Description 更新已有集群的元数据信息。管理员可以修改集群的描述文本、刷新过期的KubeConfig凭据或调整Prometheus监控URL。此操作对于维护长期运行的集群接入链条、确保SSO对接和资源透视的准确性至关重要。
// @Tags Admin,Clusters
// @Accept json
// @Produce json
// @Param id path string true "集群标识"
// @Param request body object true "集群更新内容"
// @Success 200 {object} httpx.Response{data=models.Cluster}
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /admin/clusters/{id} [put]
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
// @Summary 物理移除集群纳管
// @Description 彻底从Robusta中央控制台移除指定的K8s集群。执行此操作后，系统将停止对该集群的一切监控审计、告警接收及自动化作业下发。关联的KubeConfig记录将被永久物理删除，请谨慎执行。
// @Tags Admin,Clusters
// @Param id path string true "集群标识"
// @Success 200 {object} httpx.Response
// @Failure 500 {object} httpx.ErrorResponse
// @Router /admin/clusters/{id} [delete]
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

package api

import (
	"net/http"
	"time"

	"robusta-web/backend/internal/models"
	"robusta-web/backend/internal/services"

	"github.com/gin-gonic/gin"
)

// QueryHandler 查询处理器
type QueryHandler struct {
	alertService   *services.AlertService
	rcaService     *services.RCAService
	clusterService *services.ClusterService
	storageService services.PayloadStorage
}

// NewQueryHandler 创建新的查询处理器
func NewQueryHandler(
	alertService *services.AlertService,
	rcaService *services.RCAService,
	clusterService *services.ClusterService,
	storageService services.PayloadStorage,
) *QueryHandler {
	return &QueryHandler{
		alertService:   alertService,
		rcaService:     rcaService,
		clusterService: clusterService,
		storageService: storageService,
	}
}

// GetClustersSummary 获取集群概览
func (h *QueryHandler) GetClustersSummary(c *gin.Context) {
	summary, err := h.clusterService.GetClustersSummary()
	if err != nil {
		InternalError(c, "GET_CLUSTERS_SUMMARY_ERROR", "获取集群概览失败")
		return
	}

	Success(c, summary)
}

// GetClusters 获取集群列表
func (h *QueryHandler) GetClusters(c *gin.Context) {
	var query struct {
		PaginationQuery
		Status string `form:"status"`
	}
	if derr := bindQuery(c, &query); derr != nil {
		AbortWithDomainError(c, derr)
		return
	}
	if derr := query.PaginationQuery.validate(); derr != nil {
		AbortWithDomainError(c, derr)
		return
	}
	params := query.PaginationQuery.ToParams()

	clusters, total, err := h.clusterService.GetClusters(params.Page, params.PageSize, query.Status)
	if err != nil {
		InternalError(c, "GET_CLUSTERS_ERROR", "获取集群列表失败")
		return
	}

	pagination := NewPagination(params.Page, params.PageSize, total)
	pagination.Sort = params.Sort
	SuccessPaginated(c, clusters, pagination)
}

// GetCluster 获取单个集群详情
func (h *QueryHandler) GetCluster(c *gin.Context) {
	var path struct {
		ID string `uri:"id" binding:"required"`
	}
	if derr := bindURI(c, &path); derr != nil {
		AbortWithDomainError(c, derr)
		return
	}

	cluster, err := h.clusterService.GetClusterByID(path.ID)
	if err != nil {
		NotFound(c, "CLUSTER_NOT_FOUND", "集群不存在")
		return
	}

	Success(c, cluster)
}

// GetAlerts 获取告警列表
func (h *QueryHandler) GetAlerts(c *gin.Context) {
	var query struct {
		PaginationQuery
		ClusterName string `form:"cluster_name"`
		Severity    string `form:"severity"`
		Status      string `form:"status"`
		Keyword     string `form:"keyword"`
		Since       string `form:"since"`
	}
	if derr := bindQuery(c, &query); derr != nil {
		AbortWithDomainError(c, derr)
		return
	}
	if derr := query.PaginationQuery.validate(); derr != nil {
		AbortWithDomainError(c, derr)
		return
	}
	params := query.PaginationQuery.ToParams()

	var since *time.Time
	if query.Since != "" {
		if parsedTime, err := time.Parse(time.RFC3339, query.Since); err == nil {
			since = &parsedTime
		}
	}

	filters := services.AlertFilters{
		ClusterName: query.ClusterName,
		Severity:    query.Severity,
		Status:      query.Status,
		Keyword:     query.Keyword,
		Since:       since,
	}

	alerts, total, err := h.alertService.GetAlerts(params.Page, params.PageSize, filters)
	if err != nil {
		InternalError(c, "GET_ALERTS_ERROR", "获取告警列表失败")
		return
	}

	pagination := NewPagination(params.Page, params.PageSize, total)
	pagination.Sort = params.Sort
	SuccessPaginated(c, alerts, pagination)
}

// GetAlertTrend 获取告警趋势数据
func (h *QueryHandler) GetAlertTrend(c *gin.Context) {
	var query struct {
		Days int `form:"days" binding:"omitempty,gte=1"`
	}
	if derr := bindQuery(c, &query); derr != nil {
		AbortWithDomainError(c, derr)
		return
	}
	days := query.Days
	if days == 0 {
		days = 30
	}

	trend, err := h.alertService.GetAlertTrend(days)
	if err != nil {
		InternalError(c, "GET_ALERT_TREND_ERROR", "获取告警趋势失败")
		return
	}

	Success(c, trend)
}

// GetAlert 获取单个告警详情
func (h *QueryHandler) GetAlert(c *gin.Context) {
	var path struct {
		ID uint64 `uri:"id" binding:"required,gt=0"`
	}
	if err := c.ShouldBindUri(&path); err != nil {
		BadRequest(c, "INVALID_ALERT_ID", "无效的告警ID")
		return
	}

	alert, err := h.alertService.GetAlertByID(path.ID)
	if err != nil {
		NotFound(c, "ALERT_NOT_FOUND", "告警不存在")
		return
	}

	Success(c, alert)
}

// GetAlertRawPayload 获取告警的原始数据
func (h *QueryHandler) GetAlertRawPayload(c *gin.Context) {
	var path struct {
		ID uint64 `uri:"id" binding:"required,gt=0"`
	}
	if err := c.ShouldBindUri(&path); err != nil {
		BadRequest(c, "INVALID_ALERT_ID", "无效的告警ID")
		return
	}

	// 获取告警信息
	alert, err := h.alertService.GetAlertByID(path.ID)
	if err != nil {
		NotFound(c, "ALERT_NOT_FOUND", "告警不存在")
		return
	}

	// 检查是否有原始数据
	if alert.RawPayloadKey == "" {
		NotFound(c, "NO_RAW_PAYLOAD", "该告警没有原始数据")
		return
	}

	// 从存储服务获取原始数据
	rawData, err := h.storageService.Get(c.Request.Context(), alert.RawPayloadKey)
	if err != nil {
		ErrorWithDetails(c, http.StatusInternalServerError, "GET_RAW_PAYLOAD_ERROR", "获取原始数据失败", err.Error())
		return
	}

	// 设置响应头
	c.Header("Content-Type", "application/json")
	c.Header("X-Raw-Payload-Key", alert.RawPayloadKey)

	// 返回原始数据
	c.Data(http.StatusOK, "application/json", rawData)
}

// GetRCAByAlertID 根据告警ID获取RCA报告
func (h *QueryHandler) GetRCAByAlertID(c *gin.Context) {
	var path struct {
		AlertID uint64 `uri:"alert_id" binding:"required,gt=0"`
	}
	if err := c.ShouldBindUri(&path); err != nil {
		BadRequest(c, "INVALID_ALERT_ID", "无效的告警ID")
		return
	}

	rcaRuns, err := h.rcaService.GetRCARunsByAlertID(path.AlertID)
	if err != nil {
		InternalError(c, "GET_RCA_ERROR", "获取RCA报告失败")
		return
	}

	Success(c, rcaRuns)
}

// TriggerRCA 手动触发RCA分析
func (h *QueryHandler) TriggerRCA(c *gin.Context) {
	var path struct {
		AlertID uint64 `uri:"alert_id" binding:"required,gt=0"`
	}
	if err := c.ShouldBindUri(&path); err != nil {
		BadRequest(c, "INVALID_ALERT_ID", "无效的告警ID")
		return
	}

	// 检查告警是否存在
	alert, err := h.alertService.GetAlertByID(path.AlertID)
	if err != nil {
		NotFound(c, "ALERT_NOT_FOUND", "告警不存在")
		return
	}

	// 触发RCA分析（这里应该调用HolmesGPT服务）
	rcaRun, err := h.rcaService.TriggerRCAManual(alert)
	if err != nil {
		InternalError(c, "TRIGGER_RCA_ERROR", "触发RCA分析失败")
		return
	}

	SuccessWithMessage(c, "RCA分析已触发", gin.H{
		"rca_id": rcaRun.ID,
	})
}

// EventStream SSE事件流
func (h *QueryHandler) EventStream(c *gin.Context) {
	// 设置SSE头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")

	// 获取客户端断开连接的通道
	clientGone := c.Request.Context().Done()

	// 创建事件通道
	eventChan := make(chan interface{}, 10)

	// 启动事件监听器（这里应该实现实际的事件监听逻辑）
	go func() {
		defer close(eventChan)

		// 模拟事件推送
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-clientGone:
				return
			case <-ticker.C:
				// 发送心跳事件
				eventChan <- map[string]interface{}{
					"type":      "heartbeat",
					"timestamp": time.Now().Format(time.RFC3339),
				}
			}
		}
	}()

	// 发送事件到客户端
	for {
		select {
		case <-clientGone:
			return
		case event, ok := <-eventChan:
			if !ok {
				return
			}

			c.SSEvent("message", event)
			c.Writer.Flush()
		}
	}
}

// GetAuditLogs 获取审计日志（管理员功能）
func (h *QueryHandler) GetAuditLogs(c *gin.Context) {
	var query struct {
		PaginationQuery
		UserID string `form:"user_id"`
		Action string `form:"action"`
	}
	if derr := bindQuery(c, &query); derr != nil {
		AbortWithDomainError(c, derr)
		return
	}
	if derr := query.PaginationQuery.validate(); derr != nil {
		AbortWithDomainError(c, derr)
		return
	}
	params := query.PaginationQuery.ToParams()

	filters := gin.H{}
	if query.UserID != "" {
		filters["user_id"] = query.UserID
	}
	if query.Action != "" {
		filters["action"] = query.Action
	}

	// 这里应该调用审计服务获取日志
	pagination := NewPagination(params.Page, params.PageSize, 0)
	pagination.Sort = params.Sort
	SuccessPaginated(c, []interface{}{}, pagination)
}

// DeleteCluster 删除集群（管理员功能）
func (h *QueryHandler) DeleteCluster(c *gin.Context) {
	var path struct {
		ID string `uri:"id" binding:"required"`
	}
	if derr := bindURI(c, &path); derr != nil {
		AbortWithDomainError(c, derr)
		return
	}

	if err := h.clusterService.DeleteCluster(path.ID); err != nil {
		InternalError(c, "DELETE_CLUSTER_ERROR", "删除集群失败")
		return
	}

	SuccessWithMessage(c, "集群删除成功", nil)
}

// CreateCluster 创建集群（管理员功能）
func (h *QueryHandler) CreateCluster(c *gin.Context) {
	var req struct {
		Name          string `json:"name" binding:"required"`
		ClusterID     string `json:"cluster_id"`
		Description   string `json:"description"`
		KubeConfig    string `json:"kube_config"`
		PrometheusURL string `json:"prometheus_url"`
		Status        string `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "INVALID_REQUEST", "请求参数无效")
		return
	}

	cluster := &models.Cluster{
		Name:          req.Name,
		ClusterID:     req.ClusterID,
		Description:   req.Description,
		KubeConfig:    req.KubeConfig,
		PrometheusURL: req.PrometheusURL,
		Status:        req.Status,
	}

	if cluster.Status == "" {
		cluster.Status = "active"
	}

	if err := h.clusterService.CreateCluster(cluster); err != nil {
		InternalError(c, "CREATE_CLUSTER_ERROR", "创建集群失败: "+err.Error())
		return
	}

	SuccessWithMessage(c, "集群创建成功", cluster)
}

// UpdateCluster 更新集群（管理员功能）
func (h *QueryHandler) UpdateCluster(c *gin.Context) {
	var path struct {
		ID string `uri:"id" binding:"required"`
	}
	if derr := bindURI(c, &path); derr != nil {
		AbortWithDomainError(c, derr)
		return
	}

	var req struct {
		Description   string `json:"description"`
		KubeConfig    string `json:"kube_config"`
		PrometheusURL string `json:"prometheus_url"`
		Status        string `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "INVALID_REQUEST", "请求参数无效")
		return
	}

	cluster := &models.Cluster{
		Name:          path.ID, // Using Name as ID based on existing logic
		Description:   req.Description,
		KubeConfig:    req.KubeConfig,
		PrometheusURL: req.PrometheusURL,
		Status:        req.Status,
	}

	if err := h.clusterService.UpdateCluster(cluster); err != nil {
		InternalError(c, "UPDATE_CLUSTER_ERROR", "更新集群失败: "+err.Error())
		return
	}

	SuccessWithMessage(c, "集群更新成功", cluster)
}

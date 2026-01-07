package http

import (
	"net/http"
	"time"

	ingestservice "robusta-web/backend/internal/features/ingest/services"
	httpx "robusta-web/backend/internal/transport/httpx"

	"github.com/gin-gonic/gin"
)

// GetAlerts 获取告警列表
// @Summary 分页查询告警历史
// @Description 从系统中分页检索所有已接收的告警记录。支持通过集群名、严重程度、当前状态、时间范围以及关键字进行多维度组合过滤。该接口为系统的告警控制台提供核心数据，展示故障发生的上下文及处理状态。
// @Tags Query,Alerts
// @Produce json
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Param cluster_name query string false "按集群名称过滤"
// @Param severity query string false "按严重级别过滤"
// @Param status query string false "按状态过滤"
// @Param since query string false "起始时间 (RFC3339)"
// @Success 200 {object} httpx.Response{data=[]models.Alert}
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /alerts [get]
func (h *Handler) GetAlerts(c *gin.Context) {
	var query struct {
		httpx.PaginationQuery
		ClusterName string `form:"cluster_name"`
		Severity    string `form:"severity"`
		Status      string `form:"status"`
		Keyword     string `form:"keyword"`
		Since       string `form:"since"`
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

	var since *time.Time
	if query.Since != "" {
		if parsedTime, err := time.Parse(time.RFC3339, query.Since); err == nil {
			since = &parsedTime
		}
	}

	filters := ingestservice.AlertFilters{
		ClusterName: query.ClusterName,
		Severity:    query.Severity,
		Status:      query.Status,
		Keyword:     query.Keyword,
		Since:       since,
	}

	alerts, total, err := h.alertService.GetAlerts(params.Page, params.PageSize, filters)
	if err != nil {
		httpx.InternalError(c, "GET_ALERTS_ERROR", "获取告警列表失败")
		return
	}

	pagination := httpx.NewPagination(params.Page, params.PageSize, total)
	pagination.Sort = params.Sort
	httpx.SuccessPaginated(c, alerts, pagination)
}

// GetAlertTrend 获取告警趋势数据
// @Summary 获取告警数量趋势
// @Description 按天统计过去一段时间（默认30天）内不同严重级别的告警分布趋势。该数据通过时间序列分析生成，常用于仪表盘展示，帮助运维团队直观了解业务系统的稳定性波动情况及告警治理的长期效果。
// @Tags Query,Alerts
// @Produce json
// @Param days query int false "统计天数 (默认30)"
// @Success 200 {object} httpx.Response{data=object}
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /alerts/trend [get]
func (h *Handler) GetAlertTrend(c *gin.Context) {
	var query struct {
		Days int `form:"days" binding:"omitempty,gte=1"`
	}
	if derr := httpx.BindQuery(c, &query); derr != nil {
		httpx.AbortWithDomainError(c, derr)
		return
	}
	days := query.Days
	if days == 0 {
		days = 30
	}

	trend, err := h.alertService.GetAlertTrend(days)
	if err != nil {
		httpx.InternalError(c, "GET_ALERT_TREND_ERROR", "获取告警趋势失败")
		return
	}

	httpx.Success(c, trend)
}

// GetAlert 获取单个告警详情
// @Summary 获取特定告警详情
// @Description 根据唯一数据库ID调取特定告警的完整条目。返回结果涵盖告警的所有标签（Labels）、注释（Annotations）、产生的集群环境以及精确的时间戳。此接口为告警详情侧边栏提供展示所需的详尽上下文信息。
// @Tags Query,Alerts
// @Produce json
// @Param id path int true "告警ID"
// @Success 200 {object} httpx.Response{data=models.Alert}
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Router /alerts/{id} [get]
func (h *Handler) GetAlert(c *gin.Context) {
	var path struct {
		ID uint64 `uri:"id" binding:"required,gt=0"`
	}
	if err := c.ShouldBindUri(&path); err != nil {
		httpx.BadRequest(c, "INVALID_ALERT_ID", "无效的告警ID")
		return
	}

	alert, err := h.alertService.GetAlertByID(path.ID)
	if err != nil {
		httpx.NotFound(c, "ALERT_NOT_FOUND", "告警不存在")
		return
	}

	httpx.Success(c, alert)
}

// GetAlertRawPayload 获取告警的原始数据
// @Summary 调取原始Webhook数据
// @Description 从关联的对象存储中提取该告警最初到达系统时的原始JSON请求体。这通常包含Alertmanager或自定义探针发送的所有未过滤字段。该功能主要用于高级故障排查，帮助确认数据接收和解析阶段是否存在信息漏损。
// @Tags Query,Alerts
// @Produce json
// @Param id path int true "告警ID"
// @Success 200 {string} string "JSON 原始数据流"
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /alerts/{id}/raw [get]
func (h *Handler) GetAlertRawPayload(c *gin.Context) {
	var path struct {
		ID uint64 `uri:"id" binding:"required,gt=0"`
	}
	if err := c.ShouldBindUri(&path); err != nil {
		httpx.BadRequest(c, "INVALID_ALERT_ID", "无效的告警ID")
		return
	}

	// 获取告警信息
	alert, err := h.alertService.GetAlertByID(path.ID)
	if err != nil {
		httpx.NotFound(c, "ALERT_NOT_FOUND", "告警不存在")
		return
	}

	// 检查是否有原始数据
	if alert.RawPayloadKey == "" {
		httpx.NotFound(c, "NO_RAW_PAYLOAD", "该告警没有原始数据")
		return
	}

	// 从存储服务获取原始数据
	rawData, err := h.storageService.Get(c.Request.Context(), alert.RawPayloadKey)
	if err != nil {
		httpx.ErrorWithDetails(c, http.StatusInternalServerError, "GET_RAW_PAYLOAD_ERROR", "获取原始数据失败", err.Error())
		return
	}

	// 设置响应头
	c.Header("Content-Type", "application/json")
	c.Header("X-Raw-Payload-Key", alert.RawPayloadKey)

	// 返回原始数据
	c.Data(http.StatusOK, "application/json", rawData)
}

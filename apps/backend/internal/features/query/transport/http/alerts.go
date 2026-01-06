package http

import (
	"net/http"
	"time"

	ingestservice "robusta-web/backend/internal/features/ingest/services"
	httpx "robusta-web/backend/internal/transport/httpx"

	"github.com/gin-gonic/gin"
)

// GetAlerts 获取告警列表
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

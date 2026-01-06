package http

import (
	"time"

	httpx "robusta-web/backend/internal/transport/httpx"

	"github.com/gin-gonic/gin"
)

// GetRCAByAlertID 根据告警ID获取RCA报告
func (h *Handler) GetRCAByAlertID(c *gin.Context) {
	var path struct {
		AlertID uint64 `uri:"id" binding:"required,gt=0"`
	}
	if err := c.ShouldBindUri(&path); err != nil {
		httpx.BadRequest(c, "INVALID_ALERT_ID", "无效的告警ID")
		return
	}

	rcaRuns, err := h.rcaService.GetRCARunsByAlertID(path.AlertID)
	if err != nil {
		httpx.InternalError(c, "GET_RCA_ERROR", "获取RCA报告失败")
		return
	}

	httpx.Success(c, rcaRuns)
}

// TriggerRCA 手动触发RCA分析
func (h *Handler) TriggerRCA(c *gin.Context) {
	var path struct {
		AlertID uint64 `uri:"id" binding:"required,gt=0"`
	}
	if err := c.ShouldBindUri(&path); err != nil {
		httpx.BadRequest(c, "INVALID_ALERT_ID", "无效的告警ID")
		return
	}

	// 检查告警是否存在
	alert, err := h.alertService.GetAlertByID(path.AlertID)
	if err != nil {
		httpx.NotFound(c, "ALERT_NOT_FOUND", "告警不存在")
		return
	}

	// 触发RCA分析
	rcaRun, err := h.rcaService.TriggerRCAManual(alert)
	if err != nil {
		httpx.InternalError(c, "TRIGGER_RCA_ERROR", "触发RCA分析失败")
		return
	}

	httpx.SuccessWithMessage(c, "RCA分析已触发", gin.H{
		"rca_id": rcaRun.ID,
	})
}

// EventStream SSE事件流
func (h *Handler) EventStream(c *gin.Context) {
	// 设置SSE头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")

	// 获取客户端断开连接的通道
	clientGone := c.Request.Context().Done()

	// 创建事件通道
	eventChan := make(chan interface{}, 10)

	// 启动事件监听器
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
func (h *Handler) GetAuditLogs(c *gin.Context) {
	var query struct {
		httpx.PaginationQuery
		UserID string `form:"user_id"`
		Action string `form:"action"`
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

	filters := gin.H{}
	if query.UserID != "" {
		filters["user_id"] = query.UserID
	}
	if query.Action != "" {
		filters["action"] = query.Action
	}

	// TODO: 调用审计服务获取日志
	pagination := httpx.NewPagination(params.Page, params.PageSize, 0)
	pagination.Sort = params.Sort
	httpx.SuccessPaginated(c, []interface{}{}, pagination)
}

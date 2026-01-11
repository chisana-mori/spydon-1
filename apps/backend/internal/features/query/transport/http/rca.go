package http

import (
	"time"

	httpx "robusta-web/backend/internal/transport/httpx"

	"github.com/gin-gonic/gin"
)

// GetRCAByAlertID 根据告警ID获取RCA报告
// @Summary 获取告警根因分析报告
// @Description 检索特定告警关联的所有根因分析（RCA）运行记录。返回结果包含AI模型生成的故障结论、可能的诱因以及修复建议。该接口是实现“告警-根因-修复”闭环运维流程中的关键数据透视点。
// @Tags Query,RCA
// @Produce json
// @Param id path int true "告警ID"
// @Success 200 {object} httpx.Response{data=[]object}
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /alerts/{id}/rca [get]
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
// @Summary 手工发起告警根因分析
// @Description 针对某个特定的存量告警，手动触发AI根因分析流程。系统将基于告警发生时的实时上下文、相关指标和日志，调用AI引擎进行深度故障诊断，并异步生成新的RCA分析报告。适用于自动分析未覆盖或需要重新诊断的异常场景。
// @Tags Query,RCA
// @Produce json
// @Param id path int true "告警ID"
// @Success 200 {object} httpx.Response{data=object}
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /alerts/{id}/rca [post]
func (h *Handler) TriggerRCA(c *gin.Context) {
	var path struct {
		AlertID uint64 `uri:"id" binding:"required,gt=0"`
	}
	if err := c.ShouldBindUri(&path); err != nil {
		httpx.BadRequest(c, "INVALID_ALERT_ID", "无效的告警ID")
		return
	}

	alert, err := h.alertService.GetAlertByID(path.AlertID)
	if err != nil {
		httpx.NotFound(c, "ALERT_NOT_FOUND", "告警不存在")
		return
	}

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
// @Summary 实时系统事件流
// @Description 建立基于Server-Sent Events (SSE) 的持久连接，以便实时接收来自后端的全局事件推送（如告警产生、任务完成提醒等）。该流式接口确保前端界面能够即时响应后台状态变更，提供流畅的交互体验，并包含自动的心跳保活机制。
// @Tags Query,Events
// @Produce text/event-stream
// @Success 200 {string} string "Event: message"
// @Router /events [get]
func (h *Handler) EventStream(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")

	clientGone := c.Request.Context().Done()

	eventChan := make(chan interface{}, 10)

	go func() {
		defer close(eventChan)

		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-clientGone:
				return
			case <-ticker.C:
				eventChan <- map[string]interface{}{
					"type":      "heartbeat",
					"timestamp": time.Now().Format(time.RFC3339),
				}
			}
		}
	}()

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
// @Summary 分页查询操作审计日志
// @Description 检索系统中全局的操作审计记录。管理员可以通过用户ID、操作动作等维度对敏感操作进行追溯。该功能是满足合规性审计和安全回溯要求的重要手段，记录了所有对集群、告警及配置进行的增删改操作详情。
// @Tags Admin,Audit
// @Produce json
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Param user_id query string false "按操作人ID过滤"
// @Param action query string false "按操作动词过滤"
// @Success 200 {object} httpx.Response{data=[]object}
// @Failure 400 {object} httpx.ErrorResponse
// @Router /admin/audit-logs [get]
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

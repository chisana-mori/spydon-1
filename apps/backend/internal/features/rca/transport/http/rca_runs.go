package http

import (
	"net/http"

	httpx "robusta-web/backend/internal/transport/httpx"

	"github.com/gin-gonic/gin"
)

// TriggerRCAByAlertID triggers an RCA run for a specific alert id (manual mode).
// @Summary 按ID发起手动RCA分析
// @Description 针对指定的单一告警ID，强制系统发起一次即时的根因分析。该行为类似于用户在Web端点击“诊断”按钮，系统将同步收集相关环境数据并提交给AI后端。分析ID会在响应中立即返回，以便前端进行后续的状态追显。
// @Tags RCA
// @Produce json
// @Param alert_id path int true "内部告警ID"
// @Success 200 {object} httpx.Response{data=object}
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /rca/{alert_id}/trigger [post]
func (h *Handler) TriggerRCAByAlertID(c *gin.Context) {
	var path struct {
		AlertID uint64 `uri:"alert_id" binding:"required,gt=0"`
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

// GetRCARunStatus returns status of a specific RCA run.
// @Summary 查询RCA任务当前状态
// @Description 根据运行ID精确查询一次RCA分析任务的执行进度及最终结论。该接口透出了任务的起止时间、当前步骤（由于分析过程可能较长）以及是否已产出最终报告，是实现前端长耗时异步任务状态轮询的关键接口。
// @Tags RCA
// @Produce json
// @Param run_id path string true "分析运行ID"
// @Success 200 {object} httpx.Response{data=object}
// @Failure 404 {object} httpx.ErrorResponse
// @Router /rca/runs/{run_id} [get]
func (h *Handler) GetRCARunStatus(c *gin.Context) {
	var uri struct {
		RunID string `uri:"run_id" binding:"required"`
	}
	if err := c.ShouldBindUri(&uri); err != nil {
		httpx.BadRequest(c, "MISSING_RUN_ID", "运行ID不能为空")
		return
	}

	rcaRun, err := h.getRCARunByID(uri.RunID)
	if err != nil {
		httpx.NotFound(c, "RCA_RUN_NOT_FOUND", "未找到RCA运行记录")
		return
	}

	httpx.Success(c, rcaRun)
}

// ListRCARuns lists RCA runs with pagination.
// @Summary 分页列表显示RCA运行记录
// @Description 分页展现系统内所有的RCA分析历史记录。支持通过集群名称及运行状态（如：运行中、已完成、失败等）进行检索过滤。返回结果集供审计与追溯使用，展现了系统自动告警诊断及历史人力修复建议的历史全貌。
// @Tags RCA
// @Produce json
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Param cluster_name query string false "集群名过滤"
// @Param status query string false "任务状态过滤"
// @Success 200 {object} httpx.Response{data=[]object}
// @Failure 500 {object} httpx.ErrorResponse
// @Router /rca/runs [get]
func (h *Handler) ListRCARuns(c *gin.Context) {
	params, derr := httpx.ParsePaginationParams(c)
	if derr != nil {
		httpx.AbortWithDomainError(c, derr)
		return
	}

	var query struct {
		ClusterName string `form:"cluster_name"`
		Status      string `form:"status"`
	}
	if err := c.ShouldBindQuery(&query); err != nil {
		httpx.BadRequest(c, "INVALID_QUERY", "无效的查询参数")
		return
	}

	runs, total, err := h.listRCARuns(params.Page, params.PageSize, query.ClusterName, query.Status)
	if err != nil {
		httpx.ErrorWithDetails(c, http.StatusInternalServerError, "LIST_RCA_RUNS_FAILED", "获取RCA运行列表失败", err.Error())
		return
	}

	pagination := httpx.NewPagination(params.Page, params.PageSize, total)
	pagination.Sort = params.Sort
	httpx.SuccessPaginated(c, runs, pagination)
}

// findAlertByFingerprint looks up an alert by fingerprint and cluster.
// TODO: replace stub with real implementation using AlertService once available.
func (h *Handler) findAlertByFingerprint(fingerprint, clusterName string) (*struct {
	ID uint64 `json:"id"`
}, error) {
	// Temporary stub implementation to keep API compatible with existing handler.
	return &struct {
		ID uint64 `json:"id"`
	}{
		ID: 1,
	}, nil
}

// getRCARunByID returns a single RCA run by id (stub implementation).
func (h *Handler) getRCARunByID(runID string) (interface{}, error) {
	// TODO: wire to real RCA run storage when ready.
	return gin.H{
		"id":           runID,
		"status":       "completed",
		"started_at":   "2024-01-01T00:00:00Z",
		"completed_at": "2024-01-01T00:05:00Z",
	}, nil
}

// listRCARuns returns a list of RCA runs (stub implementation).
func (h *Handler) listRCARuns(page, limit int, clusterName, status string) ([]interface{}, int64, error) {
	// TODO: wire to real RCA run storage when ready.
	runs := []interface{}{
		gin.H{
			"id":           "run-1",
			"alert_id":     "alert-1",
			"status":       "completed",
			"started_at":   "2024-01-01T00:00:00Z",
			"completed_at": "2024-01-01T00:05:00Z",
		},
		gin.H{
			"id":         "run-2",
			"alert_id":   "alert-2",
			"status":     "running",
			"started_at": "2024-01-01T01:00:00Z",
		},
	}

	return runs, int64(len(runs)), nil
}

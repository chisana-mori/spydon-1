package http

import (
	"net/http"

	httpx "robusta-web/backend/internal/transport/httpx"

	"github.com/gin-gonic/gin"
)

// TriggerRCAByAlertID triggers an RCA run for a specific alert id (manual mode).
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

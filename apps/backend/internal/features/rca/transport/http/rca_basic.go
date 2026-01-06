package http

import (
	"errors"
	"net/http"
	"strconv"

	"robusta-web/backend/internal/apperrors"
	holmesservice "robusta-web/backend/internal/features/holmes/services"
	"robusta-web/backend/internal/models"
	httpx "robusta-web/backend/internal/transport/httpx"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// TriggerRCARequest is the payload to trigger an RCA run by alert fingerprint.
type TriggerRCARequest struct {
	AlertFingerprint string                 `json:"alert_fingerprint" binding:"required"`
	ClusterName      string                 `json:"cluster_name" binding:"required"`
	TimeoutSeconds   int                    `json:"timeout_seconds,omitempty"`
	Context          map[string]interface{} `json:"context,omitempty"`
}

// TriggerRCA triggers RCA analysis by alert fingerprint.
func (h *Handler) TriggerRCA(c *gin.Context) {
	var req TriggerRCARequest
	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.AbortWithDomainError(c, err)
		return
	}

	alert, err := h.findAlertByFingerprint(req.AlertFingerprint, req.ClusterName)
	if err != nil {
		domainErr := apperrors.NotFound(
			"",
			apperrors.WithCode("ALERT_NOT_FOUND"),
			apperrors.WithMessage("未找到对应的告警"),
			apperrors.WithDetails(err.Error()),
			apperrors.WithCause(err),
		)
		httpx.AbortWithDomainError(c, domainErr)
		return
	}

	rcaRun, err := h.holmesService.TriggerAnalysis(c.Request.Context(), alert.ID)
	if err != nil {
		domainErr := apperrors.New(
			http.StatusInternalServerError,
			"TRIGGER_RCA_FAILED",
			"触发RCA分析失败",
			apperrors.WithDetails(err.Error()),
			apperrors.WithCause(err),
		)
		httpx.AbortWithDomainError(c, domainErr)
		return
	}

	httpx.SuccessWithMessage(c, "RCA分析已触发", gin.H{
		"id":         models.FormatID(rcaRun.ID),
		"alert_id":   models.FormatID(rcaRun.AlertID),
		"status":     rcaRun.Status,
		"started_at": rcaRun.StartedAt,
	})
}

// GetRCAByAlertID returns RCA runs by alert id, with cache metadata.
func (h *Handler) GetRCAByAlertID(c *gin.Context) {
	var path struct {
		AlertID uint64 `uri:"alert_id" binding:"required,gt=0"`
	}
	if err := c.ShouldBindUri(&path); err != nil {
		httpx.BadRequest(c, "INVALID_ALERT_ID", "无效的告警ID")
		return
	}

	rcaRuns, err := h.holmesService.GetAnalysisByAlertID(path.AlertID)
	if err != nil {
		httpx.ErrorWithDetails(c, http.StatusInternalServerError, "GET_RCA_FAILED", "获取RCA结果失败", err.Error())
		return
	}

	cacheResult, cacheErr := h.holmesService.GetCachedResult(c.Request.Context(), path.AlertID)

	extras := gin.H{"cache_hit": cacheResult != nil}
	if cacheResult != nil {
		extras["cached_result"] = cacheResult
	}
	if cacheErr != nil {
		extras["cache_error"] = cacheErr.Error()
	}

	httpx.Success(c, rcaRuns, extras)
}

// GetRCACacheByAlertID returns cached RCA result for an alert (for frontend replay).
func (h *Handler) GetRCACacheByAlertID(c *gin.Context) {
	var path struct {
		AlertID uint64 `uri:"alert_id" binding:"required,gt=0"`
	}
	if err := c.ShouldBindUri(&path); err != nil {
		httpx.BadRequest(c, "INVALID_ALERT_ID", "无效的告警ID")
		return
	}

	var query struct {
		RunID string `form:"run_id"`
	}
	if derr := httpx.BindQuery(c, &query); derr != nil {
		httpx.AbortWithDomainError(c, derr)
		return
	}

	var (
		cacheResult *holmesservice.RCACachedResult
		err         error
	)

	if query.RunID != "" {
		parsedRunID, parseErr := strconv.ParseUint(query.RunID, 10, 64)
		if parseErr != nil {
			domainErr := apperrors.Validation(
				"无效的运行ID",
				map[string]string{"run_id": "格式不正确"},
				apperrors.WithCode("INVALID_RUN_ID"),
				apperrors.WithCause(parseErr),
			)
			httpx.AbortWithDomainError(c, domainErr)
			return
		}

		cacheResult, err = h.holmesService.GetCachedResultByRunID(c.Request.Context(), parsedRunID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				httpx.NotFound(c, "RCA_RUN_NOT_FOUND", "运行记录不存在或尚未产生缓存")
				return
			}
			httpx.ErrorWithDetails(c, http.StatusInternalServerError, "GET_RCA_CACHE_FAILED", "获取RCA缓存失败", err.Error())
			return
		}

		if cacheResult == nil {
			httpx.NotFound(c, "RCA_CACHE_NOT_READY", "该运行记录尚未生成RCA缓存数据")
			return
		}

		if cacheResult.AlertID != path.AlertID {
			httpx.NotFound(c, "RCA_CACHE_MISMATCH", "缓存记录不属于该告警")
			return
		}
	} else {
		cacheResult, err = h.holmesService.GetCachedResult(c.Request.Context(), path.AlertID)
	}
	if err != nil {
		httpx.ErrorWithDetails(c, http.StatusInternalServerError, "GET_RCA_CACHE_FAILED", "获取RCA缓存失败", err.Error())
		return
	}

	if cacheResult == nil {
		httpx.NotFound(c, "RCA_CACHE_NOT_FOUND", "该告警尚未进行过RCA分析，或分析结果未缓存")
		return
	}

	payload := gin.H{
		"run_id":        cacheResult.RunID,
		"alert_id":      cacheResult.AlertID,
		"cached_at":     cacheResult.CachedAt,
		"stream_chunks": cacheResult.StreamChunks,
		"metadata":      cacheResult.Metadata,
		"version":       cacheResult.Version,
	}

	httpx.Success(c, payload, gin.H{"cache_hit": true})
}

// GetRCAStats returns aggregated RCA statistics.
func (h *Handler) GetRCAStats(c *gin.Context) {
	var query struct {
		ClusterName string `form:"cluster_name"`
	}
	if derr := httpx.BindQuery(c, &query); derr != nil {
		httpx.AbortWithDomainError(c, derr)
		return
	}

	stats, err := h.holmesService.GetAnalysisStats(query.ClusterName)
	if err != nil {
		httpx.ErrorWithDetails(c, http.StatusInternalServerError, "GET_RCA_STATS_FAILED", "获取RCA统计失败", err.Error())
		return
	}

	httpx.Success(c, stats)
}

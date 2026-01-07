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
// @Summary 按指纹触发RCA分析
// @Description 通过提供告警的唯一指纹（Fingerprint）和集群名称，手动触发一次根因分析流程。该接口主要用于第三方系统集成，通过已知的告警标识符远程发起AI诊断任务，并支持配置自定义的上下文参数以增强分析效果。
// @Tags RCA
// @Accept json
// @Produce json
// @Param request body TriggerRCARequest true "触发请求详情"
// @Success 200 {object} httpx.Response{data=object}
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /rca/trigger [post]
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
// @Summary 获取告警的所有RCA记录
// @Description 根据指定的内部告警ID，检索该告警关联的所有历史分析运行记录。接口除了返回基础的运行状态和列表，还会包含缓存命中元数据。用户可以据此了解该故障是否已经过AI诊断，以及诊断的具体结论。
// @Tags RCA
// @Produce json
// @Param alert_id path int true "内部告警ID"
// @Success 200 {object} httpx.Response{data=[]object}
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /rca/{alert_id} [get]
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
// @Summary 获取RCA缓存后的分析报告
// @Description 提取已完成分析任务的完整缓存数据，主要用于在前端即时回放（Replay）AI诊断的流式过程。用户可以通过RunID指定特定的分析纪录或直接获取该告警最新的成功分析结论。包含由AI生成的结论分块、建议及关联元数据。
// @Tags RCA
// @Produce json
// @Param alert_id path int true "内部告警ID"
// @Param run_id query string false "特定运行ID"
// @Success 200 {object} httpx.Response{data=object}
// @Failure 404 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /rca/{alert_id}/cache [get]
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
// @Summary 获取RCA汇总统计
// @Description 统计系统中所有集群或特定集群下的根因分析覆盖情况。返回数据涵盖总分析次数、平均耗时、成功率以及AI发现的关键故障类别分布。该接口为运维总控中心提供了资产治理和告警质量评估的量化依据。
// @Tags RCA
// @Produce json
// @Param cluster_name query string false "集群名称过滤"
// @Success 200 {object} httpx.Response{data=object}
// @Failure 500 {object} httpx.ErrorResponse
// @Router /rca/stats [get]
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

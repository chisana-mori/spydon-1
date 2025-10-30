package api

import (
	"errors"
	"net/http"

	"robusta-web/backend/internal/apperrors"
	"robusta-web/backend/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// RCAHandler 处理RCA相关的API请求
type RCAHandler struct {
	holmesService *services.HolmesService
}

// NewRCAHandler 创建RCAHandler实例
func NewRCAHandler(holmesService *services.HolmesService) *RCAHandler {
	return &RCAHandler{
		holmesService: holmesService,
	}
}

// TriggerRCARequest 触发RCA分析请求
type TriggerRCARequest struct {
	AlertFingerprint string                 `json:"alert_fingerprint" binding:"required"`
	ClusterID        string                 `json:"cluster_id" binding:"required"`
	Depth            string                 `json:"depth,omitempty"`
	TimeoutSeconds   int                    `json:"timeout_seconds,omitempty"`
	Context          map[string]interface{} `json:"context,omitempty"`
}

var allowedRCADepths = map[string]struct{}{
	"quick":    {},
	"standard": {},
	"deep":     {},
}

func normalizeRCADepth(depth string) string {
	if depth == "" {
		return "standard"
	}
	return depth
}

func validateRCADepth(depth string) apperrors.DomainError {
	if depth == "" {
		return nil
	}
	if _, ok := allowedRCADepths[depth]; ok {
		return nil
	}

	return apperrors.Validation(
		"无效的分析深度",
		map[string]string{
			"depth": "仅支持 quick、standard、deep",
		},
	)
}

// TriggerRCA 触发RCA分析
func (h *RCAHandler) TriggerRCA(c *gin.Context) {
	var req TriggerRCARequest
	if err := bindJSON(c, &req); err != nil {
		AbortWithDomainError(c, err)
		return
	}

	req.Depth = normalizeRCADepth(req.Depth)
	if derr := validateRCADepth(req.Depth); derr != nil {
		AbortWithDomainError(c, derr)
		return
	}

	alert, err := h.findAlertByFingerprint(req.AlertFingerprint, req.ClusterID)
	if err != nil {
		domainErr := apperrors.NotFound(
			"",
			apperrors.WithCode("ALERT_NOT_FOUND"),
			apperrors.WithMessage("未找到对应的告警"),
			apperrors.WithDetails(err.Error()),
			apperrors.WithCause(err),
		)
		AbortWithDomainError(c, domainErr)
		return
	}

	// 触发分析
	rcaRun, err := h.holmesService.TriggerAnalysis(c.Request.Context(), alert.ID, req.Depth)
	if err != nil {
		domainErr := apperrors.New(
			http.StatusInternalServerError,
			"TRIGGER_RCA_FAILED",
			"触发RCA分析失败",
			apperrors.WithDetails(err.Error()),
			apperrors.WithCause(err),
		)
		AbortWithDomainError(c, domainErr)
		return
	}

	SuccessWithMessage(c, "RCA分析已触发", gin.H{
		"id":         rcaRun.ID,
		"alert_id":   rcaRun.AlertID,
		"status":     rcaRun.Status,
		"started_at": rcaRun.StartedAt,
	})
}

// GetRCAByAlertID 根据告警ID获取RCA结果
func (h *RCAHandler) GetRCAByAlertID(c *gin.Context) {
	alertID := c.Param("alert_id")
	if alertID == "" {
		BadRequest(c, "MISSING_ALERT_ID", "告警ID不能为空")
		return
	}

	rcaRuns, err := h.holmesService.GetAnalysisByAlertID(alertID)
	if err != nil {
		ErrorWithDetails(c, http.StatusInternalServerError, "GET_RCA_FAILED", "获取RCA结果失败", err.Error())
		return
	}

	cacheResult, cacheErr := h.holmesService.GetCachedResult(c.Request.Context(), alertID)

	extras := gin.H{
		"cache_hit": cacheResult != nil,
	}
	if cacheResult != nil {
		extras["cached_result"] = cacheResult
	}

	if cacheErr != nil {
		extras["cache_error"] = cacheErr.Error()
	}

	Success(c, rcaRuns, extras)
}

// GetRCACacheByAlertID 获取告警的RCA缓存结果（用于前端回放）
func (h *RCAHandler) GetRCACacheByAlertID(c *gin.Context) {
	alertID := c.Param("alert_id")
	if alertID == "" {
		BadRequest(c, "MISSING_ALERT_ID", "告警ID不能为空")
		return
	}

	runID := c.Query("run_id")

	// 获取缓存结果
	var (
		cacheResult *services.RCACachedResult
		err         error
	)

	if runID != "" {
		if _, parseErr := uuid.Parse(runID); parseErr != nil {
			domainErr := apperrors.Validation(
				"无效的运行ID",
				map[string]string{"run_id": "格式不正确"},
				apperrors.WithCode("INVALID_RUN_ID"),
				apperrors.WithCause(parseErr),
			)
			AbortWithDomainError(c, domainErr)
			return
		}

		cacheResult, err = h.holmesService.GetCachedResultByRunID(c.Request.Context(), runID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				NotFound(c, "RCA_RUN_NOT_FOUND", "运行记录不存在或尚未产生缓存")
				return
			}
			ErrorWithDetails(c, http.StatusInternalServerError, "GET_RCA_CACHE_FAILED", "获取RCA缓存失败", err.Error())
			return
		}

		if cacheResult == nil {
			NotFound(c, "RCA_CACHE_NOT_READY", "该运行记录尚未生成RCA缓存数据")
			return
		}

		if cacheResult.AlertID != "" && cacheResult.AlertID != alertID {
			NotFound(c, "RCA_CACHE_MISMATCH", "缓存记录不属于该告警")
			return
		}
	} else {
		cacheResult, err = h.holmesService.GetCachedResult(c.Request.Context(), alertID)
	}
	if err != nil {
		ErrorWithDetails(c, http.StatusInternalServerError, "GET_RCA_CACHE_FAILED", "获取RCA缓存失败", err.Error())
		return
	}

	if cacheResult == nil {
		NotFound(c, "RCA_CACHE_NOT_FOUND", "该告警尚未进行过RCA分析，或分析结果未缓存")
		return
	}

	// 返回格式化的缓存数据
	payload := gin.H{
		"run_id":        cacheResult.RunID,
		"alert_id":      cacheResult.AlertID,
		"cached_at":     cacheResult.CachedAt,
		"depth":         cacheResult.Depth,
		"stream_chunks": cacheResult.StreamChunks,
		"metadata":      cacheResult.Metadata,
		"version":       cacheResult.Version,
	}

	Success(c, payload, gin.H{"cache_hit": true})
}

// GetRCAStats 获取RCA统计信息
func (h *RCAHandler) GetRCAStats(c *gin.Context) {
	clusterID := c.Query("cluster_id")

	stats, err := h.holmesService.GetAnalysisStats(clusterID)
	if err != nil {
		ErrorWithDetails(c, http.StatusInternalServerError, "GET_RCA_STATS_FAILED", "获取RCA统计失败", err.Error())
		return
	}

	Success(c, stats)
}

// TriggerRCAByAlertID 根据告警ID触发RCA分析
func (h *RCAHandler) TriggerRCAByAlertID(c *gin.Context) {
	alertID := c.Param("alert_id")
	if alertID == "" {
		BadRequest(c, "MISSING_ALERT_ID", "告警ID不能为空")
		return
	}

	depth := normalizeRCADepth(c.DefaultQuery("depth", "standard"))
	if derr := validateRCADepth(depth); derr != nil {
		AbortWithDomainError(c, derr)
		return
	}

	// 触发分析
	rcaRun, err := h.holmesService.TriggerAnalysis(c.Request.Context(), alertID, depth)
	if err != nil {
		ErrorWithDetails(c, http.StatusInternalServerError, "TRIGGER_RCA_FAILED", "触发RCA分析失败", err.Error())
		return
	}

	SuccessWithMessage(c, "RCA分析已触发", gin.H{
		"id":         rcaRun.ID,
		"alert_id":   rcaRun.AlertID,
		"status":     rcaRun.Status,
		"started_at": rcaRun.StartedAt,
	})
}

// GetRCARunStatus 获取RCA运行状态
func (h *RCAHandler) GetRCARunStatus(c *gin.Context) {
	runID := c.Param("run_id")
	if runID == "" {
		BadRequest(c, "MISSING_RUN_ID", "运行ID不能为空")
		return
	}

	rcaRun, err := h.getRCARunByID(runID)
	if err != nil {
		NotFound(c, "RCA_RUN_NOT_FOUND", "未找到RCA运行记录")
		return
	}

	Success(c, rcaRun)
}

// ListRCARuns 列出RCA运行记录
func (h *RCAHandler) ListRCARuns(c *gin.Context) {
	params, derr := ParsePaginationParams(c)
	if derr != nil {
		AbortWithDomainError(c, derr)
		return
	}

	clusterID := c.Query("cluster_id")
	status := c.Query("status")

	rcaRuns, total, err := h.listRCARuns(params.Page, params.PageSize, clusterID, status)
	if err != nil {
		ErrorWithDetails(c, http.StatusInternalServerError, "LIST_RCA_RUNS_FAILED", "获取RCA运行列表失败", err.Error())
		return
	}

	pagination := NewPagination(params.Page, params.PageSize, total)
	pagination.Sort = params.Sort
	SuccessPaginated(c, rcaRuns, pagination)
}

// 辅助方法

func (h *RCAHandler) findAlertByFingerprint(fingerprint, clusterID string) (*struct {
	ID string `json:"id"`
}, error,
) {
	// 这里应该调用AlertService来查找告警
	// 暂时返回模拟数据
	return &struct {
		ID string `json:"id"`
	}{
		ID: "mock-alert-id",
	}, nil
}

func (h *RCAHandler) getRCARunByID(runID string) (interface{}, error) {
	// 这里应该从数据库查询RCA运行记录
	// 暂时返回模拟数据
	return gin.H{
		"id":           runID,
		"status":       "completed",
		"started_at":   "2024-01-01T00:00:00Z",
		"completed_at": "2024-01-01T00:05:00Z",
	}, nil
}

func (h *RCAHandler) listRCARuns(page, limit int, clusterID, status string) ([]interface{}, int64, error) {
	// 这里应该从数据库查询RCA运行记录列表
	// 暂时返回模拟数据
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

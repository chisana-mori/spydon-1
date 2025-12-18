package api

import (
	"errors"
	"net/http"
	"strconv"

	"time"

	"robusta-web/backend/internal/apperrors"
	"robusta-web/backend/internal/constants"
	"robusta-web/backend/internal/models"
	"robusta-web/backend/internal/services"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RCAHandler 处理RCA相关的API请求
type RCAHandler struct {
	holmesService *services.HolmesService
	rcaService    *services.RCAService
}

// NewRCAHandler 创建RCAHandler实例
func NewRCAHandler(holmesService *services.HolmesService, rcaService *services.RCAService) *RCAHandler {
	return &RCAHandler{
		holmesService: holmesService,
		rcaService:    rcaService,
	}
}

// TriggerRCARequest 触发RCA分析请求
type TriggerRCARequest struct {
	AlertFingerprint string                 `json:"alert_fingerprint" binding:"required"`
	ClusterID        string                 `json:"cluster_id" binding:"required"`
	TimeoutSeconds   int                    `json:"timeout_seconds,omitempty"`
	Context          map[string]interface{} `json:"context,omitempty"`
}

// TriggerRCA 触发RCA分析
func (h *RCAHandler) TriggerRCA(c *gin.Context) {
	var req TriggerRCARequest
	if err := bindJSON(c, &req); err != nil {
		AbortWithDomainError(c, err)
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
	rcaRun, err := h.holmesService.TriggerAnalysis(c.Request.Context(), alert.ID)
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
		"id":         models.FormatID(rcaRun.ID),
		"alert_id":   models.FormatID(rcaRun.AlertID),
		"status":     rcaRun.Status,
		"started_at": rcaRun.StartedAt,
	})
}

// GetRCAByAlertID 根据告警ID获取RCA结果
func (h *RCAHandler) GetRCAByAlertID(c *gin.Context) {
	var path struct {
		AlertID uint64 `uri:"alert_id" binding:"required,gt=0"`
	}
	if err := c.ShouldBindUri(&path); err != nil {
		BadRequest(c, "INVALID_ALERT_ID", "无效的告警ID")
		return
	}

	rcaRuns, err := h.holmesService.GetAnalysisByAlertID(path.AlertID)
	if err != nil {
		ErrorWithDetails(c, http.StatusInternalServerError, "GET_RCA_FAILED", "获取RCA结果失败", err.Error())
		return
	}

	cacheResult, cacheErr := h.holmesService.GetCachedResult(c.Request.Context(), path.AlertID)

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
	var path struct {
		AlertID uint64 `uri:"alert_id" binding:"required,gt=0"`
	}
	if err := c.ShouldBindUri(&path); err != nil {
		BadRequest(c, "INVALID_ALERT_ID", "无效的告警ID")
		return
	}

	var query struct {
		RunID string `form:"run_id"`
	}
	if derr := bindQuery(c, &query); derr != nil {
		AbortWithDomainError(c, derr)
		return
	}

	// 获取缓存结果
	var (
		cacheResult *services.RCACachedResult
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
			AbortWithDomainError(c, domainErr)
			return
		}

		cacheResult, err = h.holmesService.GetCachedResultByRunID(c.Request.Context(), parsedRunID)
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

		if cacheResult.AlertID != path.AlertID {
			NotFound(c, "RCA_CACHE_MISMATCH", "缓存记录不属于该告警")
			return
		}
	} else {
		cacheResult, err = h.holmesService.GetCachedResult(c.Request.Context(), path.AlertID)
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
		"stream_chunks": cacheResult.StreamChunks,
		"metadata":      cacheResult.Metadata,
		"version":       cacheResult.Version,
	}

	Success(c, payload, gin.H{"cache_hit": true})
}

// GetRCAStats 获取RCA统计信息
func (h *RCAHandler) GetRCAStats(c *gin.Context) {
	var query struct {
		ClusterID string `form:"cluster_id"`
	}
	if derr := bindQuery(c, &query); derr != nil {
		AbortWithDomainError(c, derr)
		return
	}

	stats, err := h.holmesService.GetAnalysisStats(query.ClusterID)
	if err != nil {
		ErrorWithDetails(c, http.StatusInternalServerError, "GET_RCA_STATS_FAILED", "获取RCA统计失败", err.Error())
		return
	}

	Success(c, stats)
}

// TriggerRCAByAlertID 根据告警ID触发RCA分析
func (h *RCAHandler) TriggerRCAByAlertID(c *gin.Context) {
	var path struct {
		AlertID uint64 `uri:"alert_id" binding:"required,gt=0"`
	}
	if err := c.ShouldBindUri(&path); err != nil {
		BadRequest(c, "INVALID_ALERT_ID", "无效的告警ID")
		return
	}

	// 触发分析
	rcaRun, err := h.holmesService.TriggerAnalysis(c.Request.Context(), path.AlertID)
	if err != nil {
		ErrorWithDetails(c, http.StatusInternalServerError, "TRIGGER_RCA_FAILED", "触发RCA分析失败", err.Error())
		return
	}

	SuccessWithMessage(c, "RCA分析已触发", gin.H{
		"id":         models.FormatID(rcaRun.ID),
		"alert_id":   models.FormatID(rcaRun.AlertID),
		"status":     rcaRun.Status,
		"started_at": rcaRun.StartedAt,
	})
}

// GetRCARunStatus 获取RCA运行状态
func (h *RCAHandler) GetRCARunStatus(c *gin.Context) {
	var uri struct {
		RunID string `uri:"run_id" binding:"required"`
	}
	if err := c.ShouldBindUri(&uri); err != nil {
		BadRequest(c, "MISSING_RUN_ID", "运行ID不能为空")
		return
	}

	rcaRun, err := h.getRCARunByID(uri.RunID)
	if err != nil {
		NotFound(c, "RCA_RUN_NOT_FOUND", "未找到RCA运行记录")
		return
	}

	Success(c, rcaRun)
}

// ListRCARuns 列出RCA运行记录
func (h *RCAHandler) ListRCARuns(c *gin.Context) {
	var query struct {
		PaginationQuery
		ClusterID string `form:"cluster_id"`
		Status    string `form:"status"`
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

	rcaRuns, total, err := h.listRCARuns(params.Page, params.PageSize, query.ClusterID, query.Status)
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
	ID uint64 `json:"id"`
}, error,
) {
	// 这里应该调用AlertService来查找告警
	// 暂时返回模拟数据
	return &struct {
		ID uint64 `json:"id"`
	}{
		ID: 1,
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

// StreamRCA 获取RCA分析流
func (h *RCAHandler) StreamRCA(c *gin.Context) {
	var uri struct {
		AlertID uint64 `uri:"alert_id" binding:"required,gt=0"`
	}
	if err := c.ShouldBindUri(&uri); err != nil {
		BadRequest(c, "MISSING_ALERT_ID", "告警ID不能为空")
		return
	}
	alertIDStr := models.FormatID(uri.AlertID)

	// 设置 SSE headers（必须在任何写入之前）
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	// CORS: 允许已知来源并与凭证配合使用（不能使用 *）
	origin := c.Request.Header.Get("Origin")
	allowedOrigins := map[string]struct{}{
		constants.CORSPort3000HTTP:  {},
		constants.CORSPort5173HTTP:  {},
		constants.CORSPort3000HTTPS: {},
		constants.CORSPort5173HTTPS: {},
	}
	if _, ok := allowedOrigins[origin]; ok {
		c.Header(constants.HeaderAccessControlAllowOrigin, origin)
		c.Header(constants.HeaderAccessControlAllowCredentials, "true")
	}

	// 1. 订阅广播（先订阅，避免错过状态更新）
	ch, unsubscribe := h.rcaService.Subscribe(alertIDStr)
	defer unsubscribe()

	// 2. 检查RCA状态（订阅后再查询，确保不会错过更新）
	rcaRuns, err := h.rcaService.GetRCARunsByAlertID(uri.AlertID)
	if err != nil {
		c.SSEvent("error", gin.H{"error": "获取RCA状态失败", "details": err.Error()})
		return
	}

	var latestRun *models.RCARun
	if len(rcaRuns) > 0 {
		latestRun = &rcaRuns[0]
	}

	// 3. 发送当前状态
	if latestRun != nil {
		c.SSEvent("status", gin.H{
			"status": latestRun.Status,
			"run_id": models.FormatID(latestRun.ID),
		})
		c.Writer.Flush()

		// 如果已完成或失败，继续监听一小段时间以防有延迟的消息
		if latestRun.Status == string(models.RCAStatusCompleted) || latestRun.Status == string(models.RCAStatusFailed) {
			// 不立即关闭，让客户端有机会接收到可能的后续消息
			// 但设置一个短超时
			timer := time.NewTimer(2 * time.Second)
			defer timer.Stop()

			select {
			case msg, ok := <-ch:
				if ok {
					if _, writeErr := c.Writer.Write([]byte("data: " + msg + "\n\n")); writeErr != nil {
						return
					}
					c.Writer.Flush()
				}
			case <-timer.C:
				return
			case <-c.Request.Context().Done():
				return
			}
			return
		}
	} else {
		c.SSEvent("status", gin.H{"status": "none", "message": "等待 RCA 分析开始"})
		c.Writer.Flush()
	}

	// 5. 监听消息
	for {
		select {
		case msg, ok := <-ch:
			if !ok {
				return
			}
			// msg 是 JSON 字符串，直接写入 data
			if _, writeErr := c.Writer.Write([]byte("data: " + msg + "\n\n")); writeErr != nil {
				return
			}
			c.Writer.Flush()
		case <-c.Request.Context().Done():
			return
		}
	}
}

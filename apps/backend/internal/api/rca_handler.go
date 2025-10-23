package api

import (
	"errors"
	"net/http"
	"strconv"

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

// TriggerRCA 触发RCA分析
func (h *RCAHandler) TriggerRCA(c *gin.Context) {
	var req TriggerRCARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "无效的请求数据",
			"details": err.Error(),
		})
		return
	}

	// 验证深度参数
	if req.Depth == "" {
		req.Depth = "standard"
	}

	validDepths := map[string]bool{
		"quick":    true,
		"standard": true,
		"deep":     true,
	}

	if !validDepths[req.Depth] {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":        "无效的分析深度",
			"valid_depths": []string{"quick", "standard", "deep"},
		})
		return
	}

	// 根据fingerprint查找告警
	alert, err := h.findAlertByFingerprint(req.AlertFingerprint, req.ClusterID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "未找到对应的告警",
			"details": err.Error(),
		})
		return
	}

	// 触发分析
	rcaRun, err := h.holmesService.TriggerAnalysis(c.Request.Context(), alert.ID, req.Depth)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "触发RCA分析失败",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "RCA分析已触发",
		"data": gin.H{
			"id":         rcaRun.ID,
			"alert_id":   rcaRun.AlertID,
			"status":     rcaRun.Status,
			"started_at": rcaRun.StartedAt,
		},
	})
}

// GetRCAByAlertID 根据告警ID获取RCA结果
func (h *RCAHandler) GetRCAByAlertID(c *gin.Context) {
	alertID := c.Param("alert_id")
	if alertID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "告警ID不能为空",
		})
		return
	}

	rcaRuns, err := h.holmesService.GetAnalysisByAlertID(alertID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "获取RCA结果失败",
			"details": err.Error(),
		})
		return
	}

	cacheResult, cacheErr := h.holmesService.GetCachedResult(c.Request.Context(), alertID)

	response := gin.H{
		"data":      rcaRuns,
		"cache_hit": cacheResult != nil,
	}

	if cacheResult != nil {
		response["cached_result"] = cacheResult
	}

	if cacheErr != nil {
		response["cache_error"] = cacheErr.Error()
	}

	c.JSON(http.StatusOK, response)
}

// GetRCACacheByAlertID 获取告警的RCA缓存结果（用于前端回放）
func (h *RCAHandler) GetRCACacheByAlertID(c *gin.Context) {
	alertID := c.Param("alert_id")
	if alertID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "告警ID不能为空",
		})
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
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "无效的运行ID",
				"details": parseErr.Error(),
			})
			return
		}

		cacheResult, err = h.holmesService.GetCachedResultByRunID(c.Request.Context(), runID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusNotFound, gin.H{
					"error":   "未找到指定的分析记录",
					"message": "运行记录不存在或尚未产生缓存",
				})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "获取RCA缓存失败",
				"details": err.Error(),
			})
			return
		}

		if cacheResult == nil {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "未找到缓存",
				"message": "该运行记录尚未生成RCA缓存数据",
			})
			return
		}

		if cacheResult.AlertID != "" && cacheResult.AlertID != alertID {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "缓存记录不属于该告警",
				"message": "请检查运行ID与告警ID是否匹配",
			})
			return
		}
	} else {
		cacheResult, err = h.holmesService.GetCachedResult(c.Request.Context(), alertID)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "获取RCA缓存失败",
			"details": err.Error(),
		})
		return
	}

	if cacheResult == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "未找到RCA缓存结果",
			"message": "该告警尚未进行过RCA分析，或分析结果未缓存",
		})
		return
	}

	// 返回格式化的缓存数据
	c.JSON(http.StatusOK, gin.H{
		"cache_hit": true,
		"data": gin.H{
			"run_id":        cacheResult.RunID,
			"alert_id":      cacheResult.AlertID,
			"cached_at":     cacheResult.CachedAt,
			"depth":         cacheResult.Depth,
			"stream_chunks": cacheResult.StreamChunks,
			"metadata":      cacheResult.Metadata,
			"version":       cacheResult.Version,
		},
	})
}

// GetRCAStats 获取RCA统计信息
func (h *RCAHandler) GetRCAStats(c *gin.Context) {
	clusterID := c.Query("cluster_id")

	stats, err := h.holmesService.GetAnalysisStats(clusterID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "获取RCA统计失败",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": stats,
	})
}

// TriggerRCAByAlertID 根据告警ID触发RCA分析
func (h *RCAHandler) TriggerRCAByAlertID(c *gin.Context) {
	alertID := c.Param("alert_id")
	if alertID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "告警ID不能为空",
		})
		return
	}

	// 获取可选参数
	depth := c.DefaultQuery("depth", "standard")

	// 验证深度参数
	validDepths := map[string]bool{
		"quick":    true,
		"standard": true,
		"deep":     true,
	}

	if !validDepths[depth] {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":        "无效的分析深度",
			"valid_depths": []string{"quick", "standard", "deep"},
		})
		return
	}

	// 触发分析
	rcaRun, err := h.holmesService.TriggerAnalysis(c.Request.Context(), alertID, depth)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "触发RCA分析失败",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "RCA分析已触发",
		"data": gin.H{
			"id":         rcaRun.ID,
			"alert_id":   rcaRun.AlertID,
			"status":     rcaRun.Status,
			"started_at": rcaRun.StartedAt,
		},
	})
}

// GetRCARunStatus 获取RCA运行状态
func (h *RCAHandler) GetRCARunStatus(c *gin.Context) {
	runID := c.Param("run_id")
	if runID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "运行ID不能为空",
		})
		return
	}

	rcaRun, err := h.getRCARunByID(runID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "未找到RCA运行记录",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": rcaRun,
	})
}

// ListRCARuns 列出RCA运行记录
func (h *RCAHandler) ListRCARuns(c *gin.Context) {
	// 分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	// 过滤参数
	clusterID := c.Query("cluster_id")
	status := c.Query("status")

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	rcaRuns, total, err := h.listRCARuns(page, limit, clusterID, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "获取RCA运行列表失败",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": rcaRuns,
		"pagination": gin.H{
			"page":  page,
			"limit": limit,
			"total": total,
		},
	})
}

// 辅助方法

func (h *RCAHandler) findAlertByFingerprint(fingerprint, clusterID string) (*struct {
	ID string `json:"id"`
}, error) {
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

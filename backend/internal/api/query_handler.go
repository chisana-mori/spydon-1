package api

import (
	"net/http"
	"strconv"
	"time"

	"robusta-web/backend/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// QueryHandler 查询处理器
type QueryHandler struct {
	alertService   *services.AlertService
	rcaService     *services.RCAService
	clusterService *services.ClusterService
	storageService services.PayloadStorage
}

// NewQueryHandler 创建新的查询处理器
func NewQueryHandler(
	alertService *services.AlertService,
	rcaService *services.RCAService,
	clusterService *services.ClusterService,
	storageService services.PayloadStorage,
) *QueryHandler {
	return &QueryHandler{
		alertService:   alertService,
		rcaService:     rcaService,
		clusterService: clusterService,
		storageService: storageService,
	}
}

// GetClustersSummary 获取集群概览
func (h *QueryHandler) GetClustersSummary(c *gin.Context) {
	summary, err := h.clusterService.GetClustersSummary()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "获取集群概览失败",
			"code":  "GET_CLUSTERS_SUMMARY_ERROR",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": summary,
	})
}

// GetClusters 获取集群列表
func (h *QueryHandler) GetClusters(c *gin.Context) {
	// 解析查询参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	status := c.Query("status")

	clusters, total, err := h.clusterService.GetClusters(page, limit, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "获取集群列表失败",
			"code":  "GET_CLUSTERS_ERROR",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": clusters,
		"pagination": gin.H{
			"page":  page,
			"limit": limit,
			"total": total,
		},
	})
}

// GetCluster 获取单个集群详情
func (h *QueryHandler) GetCluster(c *gin.Context) {
	clusterID := c.Param("id")

	cluster, err := h.clusterService.GetClusterByID(clusterID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "集群不存在",
			"code":  "CLUSTER_NOT_FOUND",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": cluster,
	})
}

// GetAlerts 获取告警列表
func (h *QueryHandler) GetAlerts(c *gin.Context) {
	// 解析查询参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	clusterID := c.Query("cluster_id")
	severity := c.Query("severity")
	status := c.Query("status")
	keyword := c.Query("keyword")

	// 解析时间范围
	var since *time.Time
	if sinceStr := c.Query("since"); sinceStr != "" {
		if parsedTime, err := time.Parse(time.RFC3339, sinceStr); err == nil {
			since = &parsedTime
		}
	}

	filters := services.AlertFilters{
		ClusterID: clusterID,
		Severity:  severity,
		Status:    status,
		Keyword:   keyword,
		Since:     since,
	}

	alerts, total, err := h.alertService.GetAlerts(page, limit, filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "获取告警列表失败",
			"code":  "GET_ALERTS_ERROR",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": alerts,
		"pagination": gin.H{
			"page":  page,
			"limit": limit,
			"total": total,
		},
	})
}

// GetAlert 获取单个告警详情
func (h *QueryHandler) GetAlert(c *gin.Context) {
	alertIDStr := c.Param("id")
	alertID, err := uuid.Parse(alertIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的告警ID",
			"code":  "INVALID_ALERT_ID",
		})
		return
	}

	alert, err := h.alertService.GetAlertByID(alertID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "告警不存在",
			"code":  "ALERT_NOT_FOUND",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": alert,
	})
}

// GetAlertRawPayload 获取告警的原始数据
func (h *QueryHandler) GetAlertRawPayload(c *gin.Context) {
	alertIDStr := c.Param("id")
	alertID, err := uuid.Parse(alertIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的告警ID",
			"code":  "INVALID_ALERT_ID",
		})
		return
	}

	// 获取告警信息
	alert, err := h.alertService.GetAlertByID(alertID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "告警不存在",
			"code":  "ALERT_NOT_FOUND",
		})
		return
	}

	// 检查是否有原始数据
	if alert.RawPayloadKey == "" {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "该告警没有原始数据",
			"code":  "NO_RAW_PAYLOAD",
		})
		return
	}

	// 从存储服务获取原始数据
	rawData, err := h.storageService.Get(c.Request.Context(), alert.RawPayloadKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "获取原始数据失败",
			"code":    "GET_RAW_PAYLOAD_ERROR",
			"details": err.Error(),
		})
		return
	}

	// 设置响应头
	c.Header("Content-Type", "application/json")
	c.Header("X-Raw-Payload-Key", alert.RawPayloadKey)

	// 返回原始数据
	c.Data(http.StatusOK, "application/json", rawData)
}

// GetRCAByAlertID 根据告警ID获取RCA报告
func (h *QueryHandler) GetRCAByAlertID(c *gin.Context) {
	alertIDStr := c.Param("alert_id")
	alertID, err := uuid.Parse(alertIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的告警ID",
			"code":  "INVALID_ALERT_ID",
		})
		return
	}

	rcaRuns, err := h.rcaService.GetRCARunsByAlertID(alertID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "获取RCA报告失败",
			"code":  "GET_RCA_ERROR",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": rcaRuns,
	})
}

// TriggerRCA 手动触发RCA分析
func (h *QueryHandler) TriggerRCA(c *gin.Context) {
	alertIDStr := c.Param("alert_id")
	alertID, err := uuid.Parse(alertIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的告警ID",
			"code":  "INVALID_ALERT_ID",
		})
		return
	}

	// 检查告警是否存在
	alert, err := h.alertService.GetAlertByID(alertID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "告警不存在",
			"code":  "ALERT_NOT_FOUND",
		})
		return
	}

	// 触发RCA分析（这里应该调用HolmesGPT服务）
	rcaRun, err := h.rcaService.TriggerRCAAnalysis(alert)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "触发RCA分析失败",
			"code":  "TRIGGER_RCA_ERROR",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "RCA分析已触发",
		"rca_id":  rcaRun.ID,
	})
}

// EventStream SSE事件流
func (h *QueryHandler) EventStream(c *gin.Context) {
	// 设置SSE头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")

	// 获取客户端断开连接的通道
	clientGone := c.Request.Context().Done()

	// 创建事件通道
	eventChan := make(chan interface{}, 10)

	// 启动事件监听器（这里应该实现实际的事件监听逻辑）
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
func (h *QueryHandler) GetAuditLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	userID := c.Query("user_id")
	action := c.Query("action")

	filters := gin.H{}
	if userID != "" {
		filters["user_id"] = userID
	}
	if action != "" {
		filters["action"] = action
	}

	// 这里应该调用审计服务获取日志
	c.JSON(http.StatusOK, gin.H{
		"data": []interface{}{},
		"pagination": gin.H{
			"page":  page,
			"limit": limit,
			"total": 0,
		},
		"filters": filters,
	})
}

// DeleteCluster 删除集群（管理员功能）
func (h *QueryHandler) DeleteCluster(c *gin.Context) {
	clusterID := c.Param("id")

	if err := h.clusterService.DeleteCluster(clusterID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "删除集群失败",
			"code":  "DELETE_CLUSTER_ERROR",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "集群删除成功",
	})
}

package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"robusta-web/backend/internal/models"
	"robusta-web/backend/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// IngestHandler 数据接收处理器
type IngestHandler struct {
	alertService   *services.AlertService
	rcaService     *services.RCAService
	clusterService *services.ClusterService
	auditService   *services.AuditService
	storageService services.PayloadStorage
}

// NewIngestHandler 创建新的数据接收处理器
func NewIngestHandler(
	alertService *services.AlertService,
	rcaService *services.RCAService,
	clusterService *services.ClusterService,
	auditService *services.AuditService,
	storageService services.PayloadStorage,
) *IngestHandler {
	return &IngestHandler{
		alertService:   alertService,
		rcaService:     rcaService,
		clusterService: clusterService,
		auditService:   auditService,
		storageService: storageService,
	}
}

// IngestAlert 接收告警数据
func (h *IngestHandler) IngestAlert(c *gin.Context) {
	rawBody, err := readAndRestoreBody(c)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "READ_BODY_ERROR", "读取请求体失败")
		return
	}

	if len(rawBody) > 0 {
		log.Printf("[Webhook] /ingest/alert 接收到原始数据: %s", string(rawBody))
	}

	var req IngestAlertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_REQUEST", "无效的请求数据", err.Error())
		return
	}

	// 验证严重级别
	if !isValidSeverity(req.Severity) {
		respondError(c, http.StatusBadRequest, "INVALID_SEVERITY", "无效的严重级别")
		return
	}

	// 保存原始payload（可选）
	payloadKey, err := savePayload(c.Request.Context(), h.storageService, fmt.Sprintf("alerts/%s", req.ClusterID), rawBody, "application/json")
	if err != nil {
		respondError(c, http.StatusInternalServerError, "SAVE_RAW_PAYLOAD_ERROR", "保存原始告警数据失败", err.Error())
		return
	}

	alert := &models.Alert{
		Fingerprint:   req.Fingerprint,
		ClusterID:     req.ClusterID,
		Title:         req.Title,
		Description:   req.Description,
		Severity:      req.Severity,
		Status:        getOrDefault(req.Status, "firing"),
		Labels:        toJSON(req.Labels),
		Annotations:   toJSON(req.Annotations),
		StartsAt:      req.StartsAt,
		EndsAt:        req.EndsAt,
		RawPayloadKey: payloadKey,
	}

	// 保存告警
	if err := h.alertService.CreateOrUpdateAlert(alert); err != nil {
		respondError(c, http.StatusInternalServerError, "SAVE_ALERT_ERROR", "保存告警失败")
		return
	}

	// 记录审计日志
	h.auditService.LogAction("system", "alert_ingested", "alert", alert.ID.String(), map[string]interface{}{
		"cluster_id":  req.ClusterID,
		"fingerprint": req.Fingerprint,
		"severity":    req.Severity,
	}, c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{
		"message":  "告警接收成功",
		"alert_id": alert.ID,
	})
}

// IngestRobustaFinding 接收Robusta默认webhook_sink的Finding并转换为内部Alert
// 该接口用于本地/快速对接，不进行HMAC校验（由routes选择不加中间件）
func (h *IngestHandler) IngestRobustaFinding(c *gin.Context) {
	// 读取原始请求体（用于存储到MinIO）
	rawBody, err := readAndRestoreBody(c)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "READ_BODY_ERROR", "读取请求体失败")
		return
	}

	if len(rawBody) > 0 {
		log.Printf("[Webhook] /ingest/robusta-webhook 接收到原始数据长度: %d bytes", len(rawBody))
	}

	var finding RobustaFinding
	if err := c.ShouldBindJSON(&finding); err != nil {
		log.Printf("[Webhook] 解析RobustaFinding失败: %v", err)
		log.Printf("[Webhook] 原始数据前200字符: %s", string((rawBody)))
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "无效的Finding格式",
			"code":    "INVALID_FINDING_FORMAT",
			"details": err.Error(),
		})
		return
	}

	// 使用后台上下文进行数据转换和保存，避免HTTP请求超时影响
	// 创建一个独立的上下文，不受HTTP请求生命周期影响
	bgCtx := context.Background()
	
	// 创建转换器并转换为Alert（使用后台上下文）
	converter := NewFindingToAlertConverterWithContext(bgCtx, h, &finding, rawBody)
	alert, err := converter.Convert()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "转换Finding失败",
			"code":    "CONVERT_FINDING_ERROR",
			"details": err.Error(),
		})
		return
	}

	// 确保集群存在
	_ = h.clusterService.UpdateHeartbeat(&models.Cluster{
		ClusterID: alert.ClusterID,
		Name:      alert.ClusterID,
		Status:    "active",
	})

	// 保存告警
	if err := h.alertService.CreateOrUpdateAlert(alert); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "保存告警失败",
			"code":    "SAVE_ALERT_ERROR",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "Robusta告警接收成功",
		"alert_id": alert.ID,
	})
}

// IngestRCA 接收RCA结果
func (h *IngestHandler) IngestRCA(c *gin.Context) {
	rawBody, err := readAndRestoreBody(c)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "READ_BODY_ERROR", "读取请求体失败")
		return
	}

	var req IngestRCARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_REQUEST", "无效的请求数据", err.Error())
		return
	}

	// 解析AlertID
	alertID, err := uuid.Parse(req.AlertID)
	if err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_ALERT_ID", "无效的告警ID")
		return
	}

	// 保存原始payload（可选）
	payloadKey, err := savePayload(c.Request.Context(), h.storageService, fmt.Sprintf("rca/%s", alertID.String()), rawBody, "application/json")
	if err != nil {
		respondError(c, http.StatusInternalServerError, "SAVE_RAW_PAYLOAD_ERROR", "保存RCA原始数据失败", err.Error())
		return
	}

	// 验证RCA状态
	if !isValidRCAStatus(req.Status) {
		respondError(c, http.StatusBadRequest, "INVALID_RCA_STATUS", "无效的RCA状态")
		return
	}

	// 创建RCA运行记录
	rcaRun := &models.RCARun{
		AlertID:         alertID,
		Status:          req.Status,
		Summary:         &req.Summary,
		Suspects:        toJSON(req.Suspects),
		Recommendations: toJSON(req.Recommendations),
		Attachments:     toJSON(req.Attachments),
		ErrorMessage:    &req.ErrorMessage,
		RawPayloadKey:   payloadKey,
	}

	// 如果状态是completed或failed/timeout，设置完成时间
	if req.Status == "completed" || req.Status == "failed" || req.Status == "timeout" {
		now := time.Now()
		rcaRun.CompletedAt = &now
	}

	// 保存RCA记录
	if err := h.rcaService.CreateRCARun(rcaRun); err != nil {
		respondError(c, http.StatusInternalServerError, "SAVE_RCA_ERROR", "保存RCA记录失败")
		return
	}

	// 记录审计日志
	h.auditService.LogAction("system", "rca_ingested", "rca_run", rcaRun.ID.String(), map[string]interface{}{
		"alert_id": req.AlertID,
		"status":   req.Status,
	}, c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{
		"message": "RCA结果接收成功",
		"rca_id":  rcaRun.ID,
	})
}

func readAndRestoreBody(c *gin.Context) ([]byte, error) {
	b, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return nil, err
	}
	c.Request.Body = io.NopCloser(bytes.NewBuffer(b))
	return b, nil
}

func respondError(c *gin.Context, status int, code, message string, details ...string) {
	resp := gin.H{"error": message, "code": code}
	if len(details) > 0 && details[0] != "" {
		resp["details"] = details[0]
	}
	c.JSON(status, resp)
}

func toJSON(m map[string]interface{}) datatypes.JSON {
	if len(m) == 0 {
		return nil
	}
	if b, err := json.Marshal(m); err == nil {
		return datatypes.JSON(b)
	}
	return nil
}

func isValidSeverity(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "low", "medium", "high", "critical":
		return true
	default:
		return false
	}
}

func isValidRCAStatus(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "pending", "running", "completed", "failed", "timeout":
		return true
	default:
		return false
	}
}

func savePayload(ctx context.Context, storage services.PayloadStorage, path string, data []byte, contentType string) (string, error) {
	if storage == nil || len(data) == 0 {
		return "", nil
	}
	return storage.Save(ctx, path, data, contentType)
}

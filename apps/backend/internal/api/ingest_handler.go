package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"robusta-web/backend/internal/logger"
	"robusta-web/backend/internal/models"
	"robusta-web/backend/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
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
		InternalError(c, "READ_BODY_ERROR", "读取请求体失败")
		return
	}

	if len(rawBody) > 0 {
		logger.L().Debug("接收到告警原始数据", zap.Int("length", len(rawBody)))
	}

	var req IngestAlertRequest
	if derr := bindJSON(c, &req); derr != nil {
		AbortWithDomainError(c, derr)
		return
	}

	// 验证严重级别
	if !isValidSeverity(req.Severity) {
		BadRequest(c, "INVALID_SEVERITY", "无效的严重级别")
		return
	}

	// 保存原始payload（可选）
	payloadKey, err := savePayload(c.Request.Context(), h.storageService, fmt.Sprintf("alerts/%s", req.ClusterID), rawBody, "application/json")
	if err != nil {
		ErrorWithDetails(c, http.StatusInternalServerError, "SAVE_RAW_PAYLOAD_ERROR", "保存原始告警数据失败", err.Error())
		return
	}

	// 设置默认状态
	status := req.Status
	if status == "" {
		status = "firing"
	}

	alert := &models.Alert{
		Fingerprint:   req.Fingerprint,
		ClusterID:     req.ClusterID,
		Title:         req.Title,
		Description:   req.Description,
		Severity:      req.Severity,
		Status:        status,
		Labels:        toJSON(req.Labels),
		Annotations:   toJSON(req.Annotations),
		StartsAt:      req.StartsAt,
		EndsAt:        req.EndsAt,
		RawPayloadKey: payloadKey,
	}

	// 保存告警
	if err := h.alertService.CreateOrUpdateAlert(c.Request.Context(), alert); err != nil {
		InternalError(c, "SAVE_ALERT_ERROR", "保存告警失败")
		return
	}

	// 记录审计日志
	if err := h.auditService.LogAction("system", "alert_ingested", "alert", alert.ID.String(), map[string]interface{}{
		"cluster_id":  req.ClusterID,
		"fingerprint": req.Fingerprint,
		"severity":    req.Severity,
	}, c.ClientIP(), c.Request.UserAgent()); err != nil {
		// 记录日志但不影响主流程
		logger.L().Warn("记录审计日志失败", zap.Error(err))
	}

	SuccessWithMessage(c, "告警接收成功", gin.H{
		"alert_id": alert.ID,
	})
}

// IngestRobustaFinding 接收Robusta默认webhook_sink的Finding并转换为内部Alert
// 该接口用于本地/快速对接，不进行HMAC校验（由routes选择不加中间件）
func (h *IngestHandler) IngestRobustaFinding(c *gin.Context) {
	// 读取原始请求体（用于存储到MinIO）
	rawBody, err := readAndRestoreBody(c)
	if err != nil {
		InternalError(c, "READ_BODY_ERROR", "读取请求体失败")
		return
	}

	if len(rawBody) > 0 {
		logger.L().Debug("接收到Robusta原始数据", zap.Int("length", len(rawBody)))
	}

	var finding RobustaFinding
	if derr := bindJSON(c, &finding); derr != nil {
		preview := string(rawBody)
		if len(preview) > 200 {
			preview = preview[:200]
		}
		logger.L().Warn("解析RobustaFinding失败",
			zap.String("payload_preview", preview),
			zap.Any("details", derr.Details()),
			zap.Error(derr.Unwrap()))
		AbortWithDomainError(c, derr)
		return
	}

	// 使用后台上下文进行数据转换和保存，避免HTTP请求超时影响
	// 创建一个独立的上下文，不受HTTP请求生命周期影响
	bgCtx := context.Background()

	// 创建转换器并转换为Alert（使用后台上下文）
	converter := NewFindingToAlertConverterWithContext(bgCtx, h, &finding, rawBody)
	alert, err := converter.Convert()
	if err != nil {
		ErrorWithDetails(c, http.StatusInternalServerError, "CONVERT_FINDING_ERROR", "转换Finding失败", err.Error())
		return
	}

	// 确保集群存在
	_ = h.clusterService.UpdateHeartbeat(c.Request.Context(), &models.Cluster{
		ClusterID: alert.ClusterID,
		Name:      alert.ClusterID,
		Status:    "active",
	})

	// 保存告警
	if err := h.alertService.CreateOrUpdateAlert(c.Request.Context(), alert); err != nil {
		ErrorWithDetails(c, http.StatusInternalServerError, "SAVE_ALERT_ERROR", "保存告警失败", err.Error())
		return
	}

	SuccessWithMessage(c, "Robusta告警接收成功", gin.H{
		"alert_id": alert.ID,
	})
}

// IngestRCA 接收RCA结果
func (h *IngestHandler) IngestRCA(c *gin.Context) {
	rawBody, err := readAndRestoreBody(c)
	if err != nil {
		InternalError(c, "READ_BODY_ERROR", "读取请求体失败")
		return
	}

	var req IngestRCARequest
	if derr := bindJSON(c, &req); derr != nil {
		AbortWithDomainError(c, derr)
		return
	}

	// 解析AlertID
	alertID, err := uuid.Parse(req.AlertID)
	if err != nil {
		BadRequest(c, "INVALID_ALERT_ID", "无效的告警ID")
		return
	}

	// 保存原始payload（可选）
	payloadKey, err := savePayload(c.Request.Context(), h.storageService, fmt.Sprintf("rca/%s", alertID.String()), rawBody, "application/json")
	if err != nil {
		ErrorWithDetails(c, http.StatusInternalServerError, "SAVE_RAW_PAYLOAD_ERROR", "保存RCA原始数据失败", err.Error())
		return
	}

	// 验证RCA状态
	if !isValidRCAStatus(req.Status) {
		BadRequest(c, "INVALID_RCA_STATUS", "无效的RCA状态")
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
		InternalError(c, "SAVE_RCA_ERROR", "保存RCA记录失败")
		return
	}

	// 记录审计日志
	if err := h.auditService.LogAction("system", "rca_ingested", "rca_run", rcaRun.ID.String(), map[string]interface{}{
		"alert_id": req.AlertID,
		"status":   req.Status,
	}, c.ClientIP(), c.Request.UserAgent()); err != nil {
		// 记录日志但不影响主流程
		logger.L().Warn("记录审计日志失败", zap.Error(err))
	}

	SuccessWithMessage(c, "RCA结果接收成功", gin.H{
		"rca_id": rcaRun.ID,
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

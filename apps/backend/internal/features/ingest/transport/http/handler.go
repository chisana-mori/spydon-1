package http

import (
	"context"
	"net/http"

	ingestservice "robusta-web/backend/internal/features/ingest/services"
	queryservice "robusta-web/backend/internal/features/query/services"
	rcaservice "robusta-web/backend/internal/features/rca/services"
	sharedservices "robusta-web/backend/internal/features/shared/services"
	"robusta-web/backend/internal/logger"
	"robusta-web/backend/internal/models"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Handler handles alert and RCA ingestion endpoints
type Handler struct {
	alertService   *ingestservice.AlertService
	rcaService     *rcaservice.RCAService
	clusterService *queryservice.ClusterService
	auditService   *sharedservices.AuditService
	storageService sharedservices.PayloadStorage
}

// New creates a new ingest handler
func New(
	alertService *ingestservice.AlertService,
	rcaService *rcaservice.RCAService,
	clusterService *queryservice.ClusterService,
	auditService *sharedservices.AuditService,
	storageService sharedservices.PayloadStorage,
) *Handler {
	return &Handler{
		alertService:   alertService,
		rcaService:     rcaService,
		clusterService: clusterService,
		auditService:   auditService,
		storageService: storageService,
	}
}

// RegisterRoutes registers all ingest routes
func (h *Handler) RegisterRoutes(v1 *gin.RouterGroup) {
	// Webhook routes (with API key middleware applied in router_builder)
	// These are registered separately in router_builder.go
}

// IngestAlert handles standard alert ingestion
// @Summary Ingest alert
// @Tags Ingest
// @Accept json
// @Produce json
// @Param request body IngestAlertRequest true "Alert data"
// @Success 200 {object} map[string]any "Success"
// @Failure 400 {object} map[string]any "Bad request"
// @Failure 500 {object} map[string]any "Internal error"
// @Router /api/v1/ingest/alert [post]
func (h *Handler) IngestAlert(c *gin.Context) {
	rawBody, err := readAndRestoreBody(c)
	if err != nil {
		internalError(c, "READ_BODY_ERROR", "读取请求体失败")
		return
	}

	var req IngestAlertRequest
	if derr := bindJSON(c, &req); derr != nil {
		abortWithDomainError(c, derr)
		return
	}

	// 调用 Service 处理告警
	alertID, err := h.alertService.ProcessAlert(c.Request.Context(), ingestservice.ProcessAlertRequest{
		Fingerprint: req.Fingerprint,
		ClusterName: req.ClusterName,
		Title:       req.Title,
		Description: req.Description,
		Severity:    req.Severity,
		Status:      req.Status,
		Labels:      req.Labels,
		Annotations: req.Annotations,
		StartsAt:    req.StartsAt,
		EndsAt:      req.EndsAt,
		RawBody:     rawBody,
		ClientIP:    c.ClientIP(),
		UserAgent:   c.Request.UserAgent(),
	})

	if err != nil {
		// 判断是否是验证错误
		if err.Error() == "无效的严重级别" {
			badRequest(c, "INVALID_SEVERITY", err.Error())
			return
		}
		internalError(c, "SAVE_ALERT_ERROR", "保存告警失败")
		return
	}

	successWithMessage(c, "告警接收成功", gin.H{
		"alert_id": alertID,
	})
}

// IngestRobustaFinding handles Robusta webhook_sink Finding ingestion
// @Summary Ingest Robusta finding
// @Tags Ingest
// @Accept json
// @Produce json
// @Param request body RobustaFinding true "Robusta finding data"
// @Success 200 {object} map[string]any "Success"
// @Failure 400 {object} map[string]any "Bad request"
// @Failure 500 {object} map[string]any "Internal error"
// @Router /api/v1/ingest/robusta-webhook [post]
func (h *Handler) IngestRobustaFinding(c *gin.Context) {
	// 读取原始请求体（用于存储到MinIO）
	rawBody, err := readAndRestoreBody(c)
	if err != nil {
		internalError(c, "READ_BODY_ERROR", "读取请求体失败")
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
		abortWithDomainError(c, derr)
		return
	}

	// 使用后台上下文进行数据转换和保存，避免HTTP请求超时影响
	bgCtx := context.Background()

	// 创建转换器并转换为Alert（使用后台上下文）
	converter := NewFindingToAlertConverterWithContext(bgCtx, h, &finding, rawBody)
	alert, err := converter.Convert()
	if err != nil {
		errorWithDetails(c, http.StatusInternalServerError, "CONVERT_FINDING_ERROR", "转换Finding失败", err.Error())
		return
	}

	// 使用 Service 处理告警（心跳、保存、审计）
	if err := h.alertService.IngestConvertedAlert(c.Request.Context(), alert, c.ClientIP(), c.Request.UserAgent()); err != nil {
		errorWithDetails(c, http.StatusInternalServerError, "SAVE_ALERT_ERROR", "保存告警失败", err.Error())
		return
	}

	successWithMessage(c, "Robusta告警接收成功", gin.H{
		"alert_id": alert.ID,
	})
}

// IngestAlertmanagerWebhook handles Alertmanager webhook ingestion
// @Summary Ingest Alertmanager webhook
// @Tags Ingest
// @Accept json
// @Produce json
// @Param request body AlertmanagerWebhookRequest true "Alertmanager webhook data"
// @Success 200 {object} map[string]any "Success"
// @Failure 400 {object} map[string]any "Bad request"
// @Failure 500 {object} map[string]any "Internal error"
// @Router /api/v1/ingest/alertmanager [post]
func (h *Handler) IngestAlertmanagerWebhook(c *gin.Context) {
	var webhook AlertmanagerWebhookRequest
	if derr := bindJSON(c, &webhook); derr != nil {
		abortWithDomainError(c, derr)
		return
	}

	for idx, amAlert := range webhook.Alerts {
		// 使用转换器转换告警数据
		converter := NewAlertmanagerAlertConverter(amAlert, webhook.Status, idx)
		alertData := converter.Convert()

		// 创建Alert模型
		alert := &models.Alert{
			Fingerprint: alertData.Fingerprint,
			ClusterName: alertData.ClusterName,
			Title:       alertData.Title,
			Description: alertData.Description,
			Severity:    alertData.Severity,
			Status:      alertData.Status,
			Labels:      alertData.Labels,
			Annotations: alertData.Annotations,
			StartsAt:    alertData.StartsAt,
			EndsAt:      alertData.EndsAt,
		}

		// 使用 Service 处理告警（心跳、保存、审计）
		if err := h.alertService.IngestConvertedAlert(c.Request.Context(), alert, c.ClientIP(), c.Request.UserAgent()); err != nil {
			internalError(c, "SAVE_ALERT_ERROR", "保存告警失败")
			return
		}
	}

	successWithMessage(c, "Alertmanager告警接收成功", gin.H{
		"count": len(webhook.Alerts),
	})
}

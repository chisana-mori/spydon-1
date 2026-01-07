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
// @Summary 接收标准格式告警
// @Description 接收并处理来自底层探针或监控组件发送的标准JSON格式告警。该接口解析告警的指纹、集群名称、严重级别和状态信息，并将其持久化到数据库中，同时触发后续的告警转发、聚合及通知等核心业务逻辑。
// @Tags Ingest
// @Accept json
// @Produce json
// @Param request body IngestAlertRequest true "标准告警数据结构"
// @Success 200 {object} map[string]any "告警接收成功"
// @Failure 400 {object} map[string]any "无效的请求参数"
// @Failure 500 {object} map[string]any "系统内部处理失败"
// @Router /ingest/alert [post]
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
// @Summary 接收Robusta Finding数据
// @Description 处理来自Robusta webhook_sink发送的Finding事件数据。接口会自动提取Finding中的K8s资源信息、告警消息及相关上下文。接收到的原始数据将存储在对象存储中，并转换为系统内部的统一告警模型以便进行后续分析。
// @Tags Ingest
// @Accept json
// @Produce json
// @Param request body RobustaFinding true "Robusta Finding事件结构"
// @Success 200 {object} map[string]any "Finding接收成功"
// @Failure 400 {object} map[string]any "数据格式解析失败"
// @Failure 500 {object} map[string]any "转换或持久化过程中发生内部错误"
// @Router /ingest/robusta-webhook [post]
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
// @Summary 接收Alertmanager告警
// @Description 接收来自Prometheus Alertmanager的标准Webhook推送请求。该接口支持处理单个Webhook请求中包含的多个告警条目。它会自动遍历告警列表，将每一项转换为系统内部的统一告警格式，并记录告警的产生集群、开始和结束时间及严重程度。
// @Tags Ingest
// @Accept json
// @Produce json
// @Param request body AlertmanagerWebhookRequest true "Alertmanager Webhook推送数据"
// @Success 200 {object} map[string]any "Alertmanager告警处理成功"
// @Failure 400 {object} map[string]any "Webhook数据不符合规范"
// @Failure 500 {object} map[string]any "数据库保存过程中发生异常"
// @Router /ingest/alertmanager [post]
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

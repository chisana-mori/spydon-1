package api

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"

	"robusta-web/backend/internal/apperrors"
	"robusta-web/backend/internal/config"
	"robusta-web/backend/internal/constants"
	"robusta-web/backend/internal/logger"
	"robusta-web/backend/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// HolmesProxyHandler 负责将前端流式请求转发到 HolmesGPT
type HolmesProxyHandler struct {
	cfg           *config.Config
	holmesService *services.HolmesService
	alertService  *services.AlertService
}

// NewHolmesProxyHandler 创建代理处理器
func NewHolmesProxyHandler(cfg *config.Config, holmesService *services.HolmesService, alertService *services.AlertService) *HolmesProxyHandler {
	return &HolmesProxyHandler{
		cfg:           cfg,
		holmesService: holmesService,
		alertService:  alertService,
	}
}

type streamInvestigateRequest struct {
	AlertID          string `json:"alert_id"`
	ForceRefresh     bool   `json:"force_refresh"`
	PreferCache      bool   `json:"prefer_cache"`
	Language         string `json:"language"`
	IncludeToolCalls *bool  `json:"include_tool_calls"`
}

// StreamInvestigate 透传 HolmesGPT 的 SSE 流
func (h *HolmesProxyHandler) StreamInvestigate(c *gin.Context) {
	if !h.cfg.HolmesGPT.Enabled {
		AbortWithDomainError(c, apperrors.New(
			http.StatusServiceUnavailable,
			"HOLMESGPT_DISABLED",
			"HolmesGPT 功能未启用",
		))
		return
	}

	var reqBody streamInvestigateRequest
	if err := c.ShouldBindJSON(&reqBody); err != nil {
		ErrorWithDetails(c, http.StatusBadRequest, constants.ErrorCodeBadRequest, "解析HolmesGPT请求参数失败", err.Error())
		return
	}

	alertID := strings.TrimSpace(reqBody.AlertID)
	if alertID == "" {
		BadRequest(c, "MISSING_ALERT_ID", "告警ID不能为空")
		return
	}

	// 验证告警ID格式
	if _, err := uuid.Parse(alertID); err != nil {
		BadRequest(c, "INVALID_ALERT_ID", "无效的告警ID")
		return
	}

	preferCache := reqBody.PreferCache
	forceRefresh := reqBody.ForceRefresh

	if queryForce := c.Query("force"); queryForce != "" {
		forceRefresh = forceRefresh || strings.EqualFold(queryForce, "true") || queryForce == "1"
	}

	if headerForce := c.GetHeader("X-RCA-Force-Refresh"); headerForce != "" {
		forceRefresh = forceRefresh || strings.EqualFold(headerForce, "true") || headerForce == "1"
	}

	if queryPrefer := c.Query("cache"); queryPrefer != "" {
		preferCache = preferCache || strings.EqualFold(queryPrefer, "true") || queryPrefer == "1"
	}

	if headerPrefer := c.GetHeader("X-RCA-Prefer-Cache"); headerPrefer != "" {
		preferCache = preferCache || strings.EqualFold(headerPrefer, "true") || headerPrefer == "1"
	}

	language := reqBody.Language
	if language == "" {
		language = "zh-CN"
	}

	includeToolCalls := true
	if reqBody.IncludeToolCalls != nil {
		includeToolCalls = *reqBody.IncludeToolCalls
	}

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		InternalError(c, "STREAM_UNSUPPORTED", "响应流不支持刷新")
		return
	}

	c.Status(http.StatusOK)
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Header("Access-Control-Allow-Origin", "*")

	opts := services.InvestigateOptions{
		ForceRefresh:     forceRefresh,
		PreferCache:      preferCache,
		Language:         language,
		IncludeToolCalls: &includeToolCalls,
		Source:           "webui",
	}

	wroteAny := false
	err := h.holmesService.Investigate(c.Request.Context(), alertID, opts, func(chunk string) error {
		wroteAny = true
		if _, err := io.WriteString(c.Writer, chunk); err != nil {
			return err
		}
		flusher.Flush()
		return nil
	})

	if err != nil {
		// 如果已经写入了部分响应，我们无法更改状态码，只能记录日志或中断流
		if wroteAny {
			logger.L().Error("流式响应中断", zap.Error(err), zap.String("alert_id", alertID))
			return
		}

		// 如果尚未写入任何响应，可以返回错误响应
		if errors.Is(err, context.Canceled) || errors.Is(err, io.EOF) {
			return
		}

		// 尝试判断是否是告警不存在
		if strings.Contains(err.Error(), "record not found") || strings.Contains(err.Error(), "告警不存在") {
			NotFound(c, "ALERT_NOT_FOUND", "告警不存在")
			return
		}

		ErrorWithDetails(c, http.StatusBadGateway, "HOLMESGPT_STREAM_ERROR", "HolmesGPT 分析失败", err.Error())
		return
	}
}

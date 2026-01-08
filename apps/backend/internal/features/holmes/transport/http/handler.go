package http

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"robusta-web/backend/internal/config"
	"robusta-web/backend/internal/features/holmes/services"
	ingestservice "robusta-web/backend/internal/features/ingest/services"
	"robusta-web/backend/internal/middleware"
	"robusta-web/backend/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/go-resty/resty/v2"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

type Handler struct {
	cfg           *config.Config
	holmesService *services.HolmesService
	alertService  *ingestservice.AlertService
}

func New(cfg *config.Config, holmesService *services.HolmesService, alertService *ingestservice.AlertService) *Handler {
	return &Handler{cfg: cfg, holmesService: holmesService, alertService: alertService}
}

// RegisterRoutes registers HolmesGPT proxy routes under /api/v1, preserving original paths
func (h *Handler) RegisterRoutes(v1 *gin.RouterGroup) {
	g := v1.Group(RouteGroup)
	g.Use(middleware.APITokenMiddleware(h.cfg))
	g.Use(middleware.AuditLogMiddleware())
	{
		g.GET(PathStreamInvestigate, h.StreamInvestigate)
		g.POST(PathSendApprovalDecision, h.SendApprovalDecision)
	}
}

type streamInvestigateRequest struct {
	AlertID          uint64 `json:"alert_id" binding:"required,gt=0"`
	ForceRefresh     bool   `json:"force_refresh"`
	PreferCache      bool   `json:"prefer_cache"`
	Language         string `json:"language"`
	IncludeToolCalls *bool  `json:"include_tool_calls"`
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// StreamInvestigate proxies WebSocket stream to HolmesGPT via HolmesService
// @Summary HolmesGPT 智能分析流
// @Description 建立WebSocket连接以启动针对特定告警的AI智能分析流程。该接口将实时推送来自HolmesGPT的分析结果、工具调用过程以及最终的修复建议。支持通过初始化参数控制强制刷新、缓存优先顺序以及输出语言等分析选项。
// @Tags Holmes
// @Param alert_id body int true "告警ID"
// @Param language body string false "语言 (默认zh-CN)"
// @Success 101 {string} string "Switching Protocols"
// @Failure 400 {object} gin.H
// @Failure 503 {object} gin.H
// @Router /holmes/investigate [get]
func (h *Handler) StreamInvestigate(c *gin.Context) {
	if !h.cfg.HolmesGPT.Enabled {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "HolmesGPT 功能未启用"})
		return
	}

	// Upgrade to WebSocket
	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logger.L().Error("WebSocket 升级失败", zap.Error(err))
		return
	}
	defer func() {
		if closeErr := ws.Close(); closeErr != nil {
			logger.L().Warn("关闭 WebSocket 连接失败", zap.Error(closeErr))
		}
	}()

	// First message as params
	_, msg, err := ws.ReadMessage()
	if err != nil {
		logger.L().Error("读取 WebSocket 初始化消息失败", zap.Error(err))
		return
	}

	var reqBody streamInvestigateRequest
	if unmarshalErr := json.Unmarshal(msg, &reqBody); unmarshalErr != nil {
		_ = ws.WriteJSON(gin.H{"error": "无效的初始化参数", "details": unmarshalErr.Error()})
		return
	}

	language := reqBody.Language
	if language == "" {
		language = "zh-CN"
	}

	includeToolCalls := true
	if reqBody.IncludeToolCalls != nil {
		includeToolCalls = *reqBody.IncludeToolCalls
	}

	opts := services.InvestigateOptions{
		ForceRefresh:     reqBody.ForceRefresh,
		PreferCache:      reqBody.PreferCache,
		Language:         language,
		IncludeToolCalls: &includeToolCalls,
		Source:           "webui",
	}

	// Keepalive
	ws.SetPingHandler(func(string) error {
		if deadlineErr := ws.SetWriteDeadline(time.Now().Add(10 * time.Second)); deadlineErr != nil {
			logger.L().Warn("设置 WebSocket 写入超时失败", zap.Error(deadlineErr))
		}
		return ws.WriteMessage(websocket.PongMessage, nil)
	})

	// Stream via service callback
	err = h.holmesService.Investigate(c.Request.Context(), reqBody.AlertID, opts, func(chunk string) error {
		if deadlineErr := ws.SetWriteDeadline(time.Now().Add(10 * time.Second)); deadlineErr != nil {
			logger.L().Warn("设置 WebSocket 写入超时失败", zap.Error(deadlineErr))
		}
		return ws.WriteMessage(websocket.TextMessage, []byte(chunk))
	})
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, io.EOF) {
			return
		}
		_ = ws.WriteJSON(gin.H{"error": "HolmesGPT 分析失败", "details": err.Error()})
		return
	}
}

// ApprovalDecisionRequest mirrors original struct
type ApprovalDecisionRequest struct {
	ConnectionID string `json:"connectionId" binding:"required"`
	ID           int    `json:"id"`
	Result       struct {
		Decision string `json:"decision" binding:"required,oneof=approved denied decline abort accept"`
	} `json:"result" binding:"required"`
}

// SendApprovalDecision forwards decision to agent backend
// @Summary 发送审批决策
// @Description 当HolmesGPT执行敏感操作（如执行修复命令）需要人工介入时，调用此接口发送审批结果（通过或拒绝）。该接口会将用户的决策实时透传给底层的Agent服务，以决定是否继续执行后续的自动化故障修复或资源变更步骤。
// @Tags Holmes
// @Accept json
// @Produce json
// @Param request body ApprovalDecisionRequest true "审批决策数据"
// @Success 200 {object} object
// @Failure 400 {object} gin.H
// @Failure 502 {object} gin.H
// @Router /holmes/send-decision [post]
func (h *Handler) SendApprovalDecision(c *gin.Context) {
	var req ApprovalDecisionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数", "details": err.Error()})
		return
	}

	agentURL := services.NormalizeURL(h.cfg.HolmesGPT.URL) + "/api/stream/investigate/send"

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	client := resty.New().SetTimeout(10 * time.Second)
	resp, err := client.R().
		SetContext(ctx).
		SetHeader("Content-Type", "application/json").
		SetBody(req).
		Post(agentURL)
	if err != nil {
		logger.L().Error("发送审批决策到 agent 失败", zap.Error(err), zap.String("agent_url", agentURL))
		c.JSON(http.StatusBadGateway, gin.H{"error": "转发审批决策失败", "details": err.Error()})
		return
	}

	c.Data(resp.StatusCode(), resp.Header().Get("Content-Type"), resp.Body())
	logger.L().Info("审批决策已转发",
		zap.String("connection_id", req.ConnectionID),
		zap.Int("request_id", req.ID),
		zap.String("decision", req.Result.Decision),
		zap.Int("status_code", resp.StatusCode()))
}

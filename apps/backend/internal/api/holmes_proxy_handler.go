package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"robusta-web/backend/internal/config"
	"robusta-web/backend/internal/logger"
	"robusta-web/backend/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/go-resty/resty/v2"
	"github.com/gorilla/websocket"
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
	AlertID          uint64 `json:"alert_id" binding:"required,gt=0"`
	ForceRefresh     bool   `json:"force_refresh"`
	PreferCache      bool   `json:"prefer_cache"`
	Language         string `json:"language"`
	IncludeToolCalls *bool  `json:"include_tool_calls"`
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许所有来源，生产环境应根据需要配置
	},
}

// StreamInvestigate 透传 HolmesGPT 的 WebSocket 流
func (h *HolmesProxyHandler) StreamInvestigate(c *gin.Context) {
	if !h.cfg.HolmesGPT.Enabled {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "HolmesGPT 功能未启用"})
		return
	}

	// 升级 HTTP 连接为 WebSocket
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

	// 读取第一条消息作为配置参数
	_, msg, err := ws.ReadMessage()
	if err != nil {
		logger.L().Error("读取 WebSocket 初始化消息失败", zap.Error(err))
		return
	}

	var reqBody streamInvestigateRequest
	if unmarshalErr := json.Unmarshal(msg, &reqBody); unmarshalErr != nil {
		if writeErr := ws.WriteJSON(gin.H{"error": "无效的初始化参数", "details": unmarshalErr.Error()}); writeErr != nil {
			logger.L().Warn("写入 WebSocket 错误消息失败", zap.Error(writeErr))
		}
		return
	}

	alertID := reqBody.AlertID

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

	// 设置 ping handler 以保持连接活跃
	ws.SetPingHandler(func(string) error {
		if deadlineErr := ws.SetWriteDeadline(time.Now().Add(10 * time.Second)); deadlineErr != nil {
			logger.L().Warn("设置 WebSocket 写入超时失败", zap.Error(deadlineErr))
		}
		return ws.WriteMessage(websocket.PongMessage, nil)
	})

	// 调用 Service 进行调查，通过回调发送 WebSocket 消息
	err = h.holmesService.Investigate(c.Request.Context(), alertID, opts, func(chunk string) error {
		if deadlineErr := ws.SetWriteDeadline(time.Now().Add(10 * time.Second)); deadlineErr != nil {
			logger.L().Warn("设置 WebSocket 写入超时失败", zap.Error(deadlineErr))
		}
		// 直接发送文本消息
		return ws.WriteMessage(websocket.TextMessage, []byte(chunk))
	})

	if err != nil {
		// 如果是上下文取消或 EOF，通常意味着正常结束或客户端断开
		if errors.Is(err, context.Canceled) || errors.Is(err, io.EOF) {
			return
		}

		// 尝试判断是否是告警不存在
		if strings.Contains(err.Error(), "record not found") || strings.Contains(err.Error(), "告警不存在") {
			if writeErr := ws.WriteJSON(gin.H{"error": "告警不存在"}); writeErr != nil {
				logger.L().Warn("写入 WebSocket 错误消息失败", zap.Error(writeErr))
			}
			return
		}

		logger.L().Error("HolmesGPT 分析失败", zap.Error(err), zap.Uint64("alert_id", alertID))
		if writeErr := ws.WriteJSON(gin.H{"error": "HolmesGPT 分析失败", "details": err.Error()}); writeErr != nil {
			logger.L().Warn("写入 WebSocket 错误消息失败", zap.Error(writeErr))
		}
		return
	}
}

// ApprovalDecisionRequest 审批决策请求
type ApprovalDecisionRequest struct {
	ConnectionID string `json:"connectionId" binding:"required"`
	ID           int    `json:"id"`
	Result       struct {
		Decision string `json:"decision" binding:"required,oneof=approved denied decline abort accept"`
	} `json:"result" binding:"required"`
}

// SendApprovalDecision 发送审批决策到 agent 后端
func (h *HolmesProxyHandler) SendApprovalDecision(c *gin.Context) {
	var req ApprovalDecisionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数", "details": err.Error()})
		return
	}

	// 构建转发到 agent 后端的 URL
	agentURL := services.NormalizeURL(h.cfg.HolmesGPT.URL) + "/api/stream/investigate/send"

	// 使用 resty 转发请求到 agent 后端
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

	// 转发响应状态码和内容
	c.Data(resp.StatusCode(), resp.Header().Get("Content-Type"), resp.Body())

	logger.L().Info("审批决策已转发",
		zap.String("connection_id", req.ConnectionID),
		zap.Int("request_id", req.ID),
		zap.String("decision", req.Result.Decision),
		zap.Int("status_code", resp.StatusCode()))
}

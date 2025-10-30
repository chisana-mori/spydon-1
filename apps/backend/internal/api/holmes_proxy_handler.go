package api

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"robusta-web/backend/internal/apperrors"
	"robusta-web/backend/internal/config"
	"robusta-web/backend/internal/constants"
	"robusta-web/backend/internal/logger"
	"robusta-web/backend/internal/models"
	"robusta-web/backend/internal/services"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// HolmesProxyHandler 负责将前端流式请求转发到 HolmesGPT
type HolmesProxyHandler struct {
	cfg           *config.Config
	client        *http.Client
	holmesService *services.HolmesService
}

// NewHolmesProxyHandler 创建代理处理器
func NewHolmesProxyHandler(cfg *config.Config, holmesService *services.HolmesService) *HolmesProxyHandler {
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DisableCompression:    true,
		DisableKeepAlives:     true,                                                       // 避免 SSE 重用断开的连接
		TLSNextProto:          make(map[string]func(string, *tls.Conn) http.RoundTripper), // 禁用 HTTP/2，确保与上游代理兼容
	}

	return &HolmesProxyHandler{
		cfg:           cfg,
		holmesService: holmesService,
		client: &http.Client{
			Timeout:   120 * time.Second,
			Transport: transport,
		},
	}
}

type investigatePayload struct {
	Subject struct {
		AlertID string `json:"alert_id"`
	} `json:"subject"`
	Depth        string `json:"depth"`
	ForceRefresh bool   `json:"force_refresh"`
	PreferCache  bool   `json:"prefer_cache"`
	CacheControl struct {
		BypassCache bool `json:"bypass_cache"`
		PreferCache bool `json:"prefer_cache"`
	} `json:"cache_control"`
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

	requestBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		ErrorWithDetails(c, http.StatusBadRequest, constants.ErrorCodeBadRequest, "读取请求体失败", err.Error())
		return
	}

	var payload investigatePayload
	alertID := ""
	depth := ""
	preferCache := false
	forceRefresh := false

	if err := json.Unmarshal(requestBody, &payload); err != nil {
		logger.L().Warn("解析HolmesGPT请求体失败，将以透传方式继续", zap.Error(err))
	} else {
		alertID = strings.TrimSpace(payload.Subject.AlertID)
		depth = strings.TrimSpace(payload.Depth)
		forceRefresh = payload.ForceRefresh || payload.CacheControl.BypassCache
		preferCache = payload.PreferCache || payload.CacheControl.PreferCache
	}

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

	if depth == "" {
		depth = strings.TrimSpace(h.cfg.HolmesGPT.DefaultDepth)
		if depth == "" {
			depth = "standard"
		}
	}

	if alertID != "" && preferCache && !forceRefresh && h.holmesService != nil {
		if cached, cacheErr := h.holmesService.GetCachedResult(c.Request.Context(), alertID); cacheErr == nil && cached != nil && len(cached.StreamChunks) > 0 {
			h.streamFromCache(c, cached)
			return
		} else if cacheErr != nil {
			logger.L().Warn("读取RCA缓存失败，继续走实时分析", zap.Error(cacheErr), zap.String("alert_id", alertID))
		}
	}

	upstreamURL := strings.TrimRight(h.cfg.HolmesGPT.URL, "/") + "/api/stream/investigate"
	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, upstreamURL, bytes.NewReader(requestBody))
	if err != nil {
		ErrorWithDetails(c, http.StatusInternalServerError, constants.ErrorCodeInternal, "构造上游请求失败", err.Error())
		return
	}

	// SSE 要求使用独立连接，防止复用旧连接导致流被截断
	req.Close = true
	req.Header.Set("Connection", "close")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")

	if h.cfg.HolmesGPT.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+h.cfg.HolmesGPT.APIKey)
	}

	var rcaRun *models.RCARun
	if alertID != "" && h.holmesService != nil {
		if run, runErr := h.holmesService.StartStreamRun(alertID); runErr == nil {
			rcaRun = run
		} else {
			logger.L().Warn("创建RCA流式运行记录失败", zap.Error(runErr), zap.String("alert_id", alertID))
		}
	}

	streamChunks := make([]string, 0, 128)
	bufferedSSE := ""
	var streamErr error
	metadata := make(map[string]interface{})

	resp, err := h.client.Do(req)
	if err != nil {
		streamErr = err
		if rcaRun != nil {
			h.holmesService.FinalizeStreamRun(c.Request.Context(), rcaRun, depth, nil, metadata, streamErr)
		}

		select {
		case <-c.Request.Context().Done():
			return
		default:
		}

		ErrorWithDetails(c, http.StatusBadGateway, "HOLMESGPT_UPSTREAM_FAILURE", "HolmesGPT 请求失败", err.Error())
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		streamErr = fmt.Errorf("status %d: %s", resp.StatusCode, string(body))
		if rcaRun != nil {
			h.holmesService.FinalizeStreamRun(c.Request.Context(), rcaRun, depth, nil, metadata, streamErr)
		}
		ErrorWithDetails(c, resp.StatusCode, "HOLMESGPT_ERROR", fmt.Sprintf("HolmesGPT 返回错误: %s", http.StatusText(resp.StatusCode)), string(body))
		return
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

	buffer := make([]byte, 4096)
	defer func() {
		if rcaRun != nil {
			if len(bufferedSSE) > 0 {
				streamChunks = append(streamChunks, bufferedSSE)
				bufferedSSE = ""
			}
			logger.L().Info("流式RCA分析完成，准备保存到MinIO", zap.Int("chunk_count", len(streamChunks)), zap.Error(streamErr))
			h.holmesService.FinalizeStreamRun(c.Request.Context(), rcaRun, depth, streamChunks, metadata, streamErr)
		}
	}()

	for {
		if err := c.Request.Context().Err(); err != nil {
			streamErr = err
			return
		}

		n, readErr := resp.Body.Read(buffer)
		if n > 0 {
			if rcaRun != nil {
				bufferedSSE += string(buffer[:n])

				for {
					idx := strings.Index(bufferedSSE, "\n\n")
					if idx < 0 {
						break
					}
					event := bufferedSSE[:idx+2]
					streamChunks = append(streamChunks, event)
					bufferedSSE = bufferedSSE[idx+2:]
				}
			}
			if _, writeErr := c.Writer.Write(buffer[:n]); writeErr != nil {
				streamErr = writeErr
				return
			}
			flusher.Flush()
		}

		if readErr != nil {
			if readErr == io.EOF {
				streamErr = nil
				return
			}
			// 非 EOF 错误直接返回，前端会收到不完整 SSE
			streamErr = readErr
			return
		}
	}
}

func (h *HolmesProxyHandler) streamFromCache(c *gin.Context, cached *services.RCACachedResult) {
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
	c.Header("X-RCA-Cache-Hit", "true")

	chunks := cached.StreamChunks
	if len(chunks) == 0 && cached.Analysis != nil {
		if payload, err := json.Marshal(gin.H{
			"type": "analysis",
			"data": cached.Analysis,
		}); err == nil {
			chunks = append(chunks, fmt.Sprintf("data: %s\n\n", string(payload)))
		}
		chunks = append(chunks, "data: {\"type\":\"complete\",\"data\":{}}\n\n")
		chunks = append(chunks, "data: [DONE]\n\n")
	}

	for _, chunk := range chunks {
		if _, err := io.WriteString(c.Writer, chunk); err != nil {
			return
		}
		flusher.Flush()
	}
}

package api

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"robusta-web/backend/internal/config"

	"github.com/gin-gonic/gin"
)

// HolmesProxyHandler 负责将前端流式请求转发到 HolmesGPT
type HolmesProxyHandler struct {
	cfg    *config.Config
	client *http.Client
}

// NewHolmesProxyHandler 创建代理处理器
func NewHolmesProxyHandler(cfg *config.Config) *HolmesProxyHandler {
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DisableCompression:    true,
	}

	return &HolmesProxyHandler{
		cfg: cfg,
		client: &http.Client{
			Timeout:   0,
			Transport: transport,
		},
	}
}

// StreamInvestigate 透传 HolmesGPT 的 SSE 流
func (h *HolmesProxyHandler) StreamInvestigate(c *gin.Context) {
	if !h.cfg.HolmesGPT.Enabled {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "HolmesGPT 功能未启用",
		})
		return
	}

	requestBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "读取请求体失败",
			"details": err.Error(),
		})
		return
	}

	upstreamURL := strings.TrimRight(h.cfg.HolmesGPT.URL, "/") + "/api/stream/investigate"
	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, upstreamURL, bytes.NewReader(requestBody))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "构造上游请求失败",
			"details": err.Error(),
		})
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")

	if h.cfg.HolmesGPT.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+h.cfg.HolmesGPT.APIKey)
	}

	resp, err := h.client.Do(req)
	if err != nil {
		select {
		case <-c.Request.Context().Done():
			return
		default:
		}

		c.JSON(http.StatusBadGateway, gin.H{
			"error":   "HolmesGPT 请求失败",
			"details": err.Error(),
		})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		c.JSON(resp.StatusCode, gin.H{
			"error":   fmt.Sprintf("HolmesGPT 返回错误: %s", http.StatusText(resp.StatusCode)),
			"details": string(body),
		})
		return
	}

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "响应流不支持刷新",
		})
		return
	}

	c.Status(http.StatusOK)
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Header("Access-Control-Allow-Origin", "*")

	buffer := make([]byte, 4096)
	for {
		if err := c.Request.Context().Err(); err != nil {
			return
		}

		n, readErr := resp.Body.Read(buffer)
		if n > 0 {
			if _, writeErr := c.Writer.Write(buffer[:n]); writeErr != nil {
				return
			}
			flusher.Flush()
		}

		if readErr != nil {
			if readErr == io.EOF {
				return
			}
			// 非 EOF 错误直接返回，前端会收到不完整 SSE
			return
		}
	}
}

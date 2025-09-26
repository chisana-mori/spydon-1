package api

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"robusta-web/backend/internal/config"
	"robusta-web/backend/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// HolmesStreamHandler 负责代理 HolmesGPT 的流式分析
// 同时将原始 SSE 数据存储到对象存储中，供后续回放使用。
type HolmesStreamHandler struct {
	cfg            *config.Config
	holmesService  *services.HolmesService
	storageService services.PayloadStorage
	httpClient     *http.Client
}

// NewHolmesStreamHandler 构造函数
func NewHolmesStreamHandler(cfg *config.Config, holmesService *services.HolmesService, storageService services.PayloadStorage) *HolmesStreamHandler {
	timeout := time.Duration(cfg.HolmesGPT.TimeoutSeconds)
	if timeout <= 0 {
		timeout = 300
	}

	return &HolmesStreamHandler{
		cfg:            cfg,
		holmesService:  holmesService,
		storageService: storageService,
		httpClient: &http.Client{
			Timeout: timeout * time.Second,
		},
	}
}

// StreamInvestigate 代理 HolmesGPT 的 /stream/investigate 接口，并实时转发给前端
func (h *HolmesStreamHandler) StreamInvestigate(c *gin.Context) {
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "读取请求失败", "code": "READ_BODY_FAILED"})
		return
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的JSON请求", "code": "INVALID_JSON"})
		return
	}

	alertIDStr := extractAlertID(payload)
	if alertIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求体中缺少 subject.alert_id", "code": "MISSING_ALERT_ID"})
		return
	}

	alertID, err := uuid.Parse(alertIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的告警ID", "code": "INVALID_ALERT_ID"})
		return
	}

	run, err := h.holmesService.CreateStreamRun(alertID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建RCA运行记录失败", "code": "CREATE_RUN_FAILED", "details": err.Error()})
		return
	}

	// 设置SSE响应头
	c.Writer.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-RCA-Run-ID", run.ID.String())
	c.Status(http.StatusOK)

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		h.holmesService.MarkStreamRunFailed(run.ID, "响应不支持SSE刷新")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "不支持SSE刷新", "code": "NO_FLUSHER"})
		return
	}

	// 准备上游请求
	upstreamURL := strings.TrimRight(h.cfg.HolmesGPT.URL, "/") + "/api/stream/investigate"
	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, upstreamURL, bytes.NewReader(bodyBytes))
	if err != nil {
		h.holmesService.MarkStreamRunFailed(run.ID, fmt.Sprintf("构造HolmesGPT请求失败: %v", err))
		writeSSEError(c, flusher, fmt.Sprintf("构造HolmesGPT请求失败: %v", err))
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	if token := strings.TrimSpace(h.cfg.HolmesGPT.APIKey); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := h.httpClient.Do(req)
	if err != nil {
		h.holmesService.MarkStreamRunFailed(run.ID, fmt.Sprintf("请求HolmesGPT失败: %v", err))
		writeSSEError(c, flusher, fmt.Sprintf("请求HolmesGPT失败: %v", err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		responseBody, _ := io.ReadAll(resp.Body)
		errMsg := fmt.Sprintf("HolmesGPT返回状态 %d: %s", resp.StatusCode, strings.TrimSpace(string(responseBody)))
		h.holmesService.MarkStreamRunFailed(run.ID, errMsg)
		writeSSEError(c, flusher, errMsg)
		return
	}

	// 缓存SSE原始内容
	var sseBuffer bytes.Buffer
	reader := bufio.NewReader(resp.Body)
	var streamErr error
	clientClosed := false

	for {
		line, err := reader.ReadString('\n')
		if len(line) > 0 {
			sseBuffer.WriteString(line)
			if !clientClosed {
				if _, writeErr := c.Writer.Write([]byte(line)); writeErr != nil {
					clientClosed = true
				} else {
					flusher.Flush()
				}
			}
		}

		if err != nil {
			if err == io.EOF {
				break
			}
			streamErr = err
			break
		}
	}

	// 若缓冲中没有 complete 事件，补一个
	if !strings.Contains(sseBuffer.String(), "\"type\":\"complete\"") {
		completeLine := "data: {\"type\":\"complete\",\"data\":{}}\n\n"
		sseBuffer.WriteString(completeLine)
		if !clientClosed {
			c.Writer.Write([]byte(completeLine))
			flusher.Flush()
		}
	}

	storageKey, saveErr := h.storageService.Save(c.Request.Context(), fmt.Sprintf("rca-runs/%s/%s", alertID.String(), run.ID.String()), sseBuffer.Bytes(), "text/event-stream")
	if saveErr != nil {
		streamErr = fmt.Errorf("保存SSE结果失败: %w", saveErr)
	}

	if streamErr != nil {
		h.holmesService.MarkStreamRunFailed(run.ID, streamErr.Error())
	} else {
		if err := h.holmesService.MarkStreamRunCompleted(run.ID, storageKey); err != nil {
			h.holmesService.MarkStreamRunFailed(run.ID, fmt.Sprintf("更新运行状态失败: %v", err))
		}
	}
}

// StreamRunReplay 将已存储的SSE原文回放给前端
func (h *HolmesStreamHandler) StreamRunReplay(c *gin.Context) {
	runIDStr := c.Param("run_id")
	runID, err := uuid.Parse(runIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的RCA运行ID", "code": "INVALID_RUN_ID"})
		return
	}

	run, err := h.holmesService.GetRunByID(runID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "RCA运行记录不存在", "code": "RUN_NOT_FOUND"})
		return
	}

	if strings.TrimSpace(run.AnalysisPayloadKey) == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "该RCA运行未存储SSE结果", "code": "NO_ANALYSIS_PAYLOAD"})
		return
	}

	data, err := h.storageService.Get(c.Request.Context(), run.AnalysisPayloadKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取SSE结果失败", "code": "LOAD_ANALYSIS_FAILED", "details": err.Error()})
		return
	}

	// 设置SSE响应头
	c.Writer.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Status(http.StatusOK)

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "不支持SSE刷新", "code": "NO_FLUSHER"})
		return
	}

	reader := bufio.NewReader(bytes.NewReader(data))
	for {
		line, err := reader.ReadString('\n')
		if len(line) > 0 {
			if _, writeErr := c.Writer.Write([]byte(line)); writeErr != nil {
				return
			}
			flusher.Flush()
		}

		if err != nil {
			return
		}
	}
}

// 从请求中提取 alert_id，兼容 subject.alert_id / subject.alertId 等写法
func extractAlertID(payload map[string]interface{}) string {
	subjectRaw, ok := payload["subject"]
	if !ok {
		return ""
	}
	subject, ok := subjectRaw.(map[string]interface{})
	if !ok {
		return ""
	}

	if v, ok := subject["alert_id"].(string); ok && v != "" {
		return v
	}
	if v, ok := subject["alertId"].(string); ok && v != "" {
		return v
	}
	if v, ok := subject["alertID"].(string); ok && v != "" {
		return v
	}
	return ""
}

func writeSSEError(c *gin.Context, flusher http.Flusher, message string) {
	payload := map[string]interface{}{
		"type": "error",
		"data": map[string]interface{}{
			"message": message,
		},
	}
	data, _ := json.Marshal(payload)
	c.Writer.Write([]byte("data: " + string(data) + "\n\n"))
	flusher.Flush()
}

package api

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
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

// IngestAlertRequest 接收告警请求结构
type IngestAlertRequest struct {
	Fingerprint string                 `json:"fingerprint" binding:"required"`
	ClusterID   string                 `json:"cluster_id" binding:"required"`
	Title       string                 `json:"title" binding:"required"`
	Description string                 `json:"description"`
	Severity    string                 `json:"severity" binding:"required"`
	Status      string                 `json:"status"`
	Labels      map[string]interface{} `json:"labels"`
	Annotations map[string]interface{} `json:"annotations"`
	StartsAt    *time.Time             `json:"starts_at"`
	EndsAt      *time.Time             `json:"ends_at"`
}

// IngestAlert 接收告警数据
func (h *IngestHandler) IngestAlert(c *gin.Context) {
	rawBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "读取请求体失败",
			"code":  "READ_BODY_ERROR",
		})
		return
	}

	c.Request.Body = io.NopCloser(bytes.NewBuffer(rawBody))
	if len(rawBody) > 0 {
		log.Printf("[Webhook] /ingest/alert 接收到原始数据: %s", string(rawBody))
	}
	var req IngestAlertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "无效的请求数据",
			"code":    "INVALID_REQUEST",
			"details": err.Error(),
		})
		return
	}

	// 验证严重级别
	validSeverities := []string{"low", "medium", "high", "critical"}
	if !contains(validSeverities, req.Severity) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的严重级别",
			"code":  "INVALID_SEVERITY",
		})
		return
	}

	// 创建告警模型
	// 转换JSON字段
	var lblJSON, annJSON datatypes.JSON
	if req.Labels != nil {
		if b, err := json.Marshal(req.Labels); err == nil {
			lblJSON = datatypes.JSON(b)
		}
	}
	if req.Annotations != nil {
		if b, err := json.Marshal(req.Annotations); err == nil {
			annJSON = datatypes.JSON(b)
		}
	}

	payloadKey := ""
	if h.storageService != nil {
		if key, err := h.storageService.Save(c.Request.Context(), fmt.Sprintf("alerts/%s", req.ClusterID), rawBody, "application/json"); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "保存原始告警数据失败",
				"code":    "SAVE_RAW_PAYLOAD_ERROR",
				"details": err.Error(),
			})
			return
		} else {
			payloadKey = key
		}
	}

	alert := &models.Alert{
		Fingerprint:   req.Fingerprint,
		ClusterID:     req.ClusterID,
		Title:         req.Title,
		Description:   req.Description,
		Severity:      req.Severity,
		Status:        getOrDefault(req.Status, "firing"),
		Labels:        lblJSON,
		Annotations:   annJSON,
		StartsAt:      req.StartsAt,
		EndsAt:        req.EndsAt,
		RawPayloadKey: payloadKey,
	}

	// 保存告警
	if err := h.alertService.CreateOrUpdateAlert(alert); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "保存告警失败",
			"code":  "SAVE_ALERT_ERROR",
		})
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
	// 读取原始请求体，兼容Robusta webhook_sink可能发送的多种格式
	raw, _ := io.ReadAll(c.Request.Body)
	// 恢复Body供后续使用
	c.Request.Body = io.NopCloser(bytes.NewBuffer(raw))
	if len(raw) > 0 {
		log.Printf("[Webhook] /ingest/robusta-webhook 接收到原始数据: %s", string(raw))
	}

	// 优先按JSON解析；否则按纯文本处理
	body := map[string]interface{}{}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &body)
	}
	if len(body) == 0 && len(raw) > 0 {
		body["title"] = "Robusta Webhook"
		body["description"] = string(raw)
		body["severity"] = "medium"
	}

	// 从头或payload中推断cluster_id
	clusterID := c.GetHeader("X-Robusta-Cluster-ID")
	// 备选：payload中的显式字段
	if clusterID == "" {
		if v, ok := body["cluster_name"].(string); ok {
			clusterID = v
		}
	}
	// 备选：labels/annotations中的常见字段
	if clusterID == "" {
		// 尝试从labels
		if lb, ok := body["labels"].(map[string]interface{}); ok {
			for _, k := range []string{"robusta_cluster", "cluster", "kubernetes_cluster", "cluster_id"} {
				if s, ok2 := lb[k].(string); ok2 && s != "" {
					clusterID = s
					break
				}
			}
		}
		// 尝试从annotations
		if clusterID == "" {
			if an, ok := body["annotations"].(map[string]interface{}); ok {
				for _, k := range []string{"robusta_cluster", "cluster", "kubernetes_cluster", "cluster_id"} {
					if s, ok2 := an[k].(string); ok2 && s != "" {
						clusterID = s
						break
					}
				}
			}
		}
	}
	// 备选：从description文本中解析类似 "Source: kind"/"cluster: kind"
	if clusterID == "" {
		if desc, _ := body["description"].(string); desc != "" {
			re := regexp.MustCompile(`(?i)(source|cluster|cluster_name)\s*:\s*([A-Za-z0-9\-_.:]+)`)
			if m := re.FindStringSubmatch(desc); len(m) >= 3 {
				clusterID = strings.TrimSpace(m[2])
			}
		}
	}
	if clusterID == "" {
		// 最终兜底：开发环境默认 kind，防止外键约束失败
		clusterID = "kind"
	}

	// 提取基础字段（兼容多种键）
	pickString := func(m map[string]interface{}, keys ...string) string {
		for _, k := range keys {
			if v, ok := m[k].(string); ok && v != "" {
				return v
			}
		}
		return ""
	}
	title := pickString(body, "title", "summary", "text", "message")
	description := pickString(body, "description", "details")
	// 预先声明严重级别与聚合键，供后续规则使用
	sev, _ := body["severity"].(string)
	if sev == "" {
		sev = "medium"
	}
	// 推断 aggregation_key（尽量稳定）
	aggKey, _ := body["aggregation_key"].(string)
	if title == "" && description != "" {
		// 取描述首行作为标题
		if idx := strings.IndexByte(description, '\n'); idx > 0 {
			title = strings.TrimSpace(description[:idx])
		} else {
			title = strings.TrimSpace(description)
		}
	}
	// 先从labels中直接标准化（优先级最高）
	var lblMap map[string]interface{}
	if lb, ok := body["labels"].(map[string]interface{}); ok {
		lblMap = lb
		// Prometheus alertname → 标准标题
		if an, ok := lb["alertname"].(string); ok && an != "" {
			title = fmt.Sprintf("Prometheus Alert: %s", an)
		}
		// severity 优先从 labels 覆盖
		if sv, ok := lb["severity"].(string); ok && sv != "" {
			svl := strings.ToLower(sv)
			switch svl {
			case "info", "informational":
				sev = "low"
			case "low":
				sev = "low"
			case "medium", "moderate":
				sev = "medium"
			case "high":
				sev = "high"
			case "critical", "urgent":
				sev = "critical"
			default:
				// 保持现有 sev
			}
		}
	}

	// 针对常见文本格式做标准化标题与聚合键提取（当labels未提供时）
	// 1) "Failed to pull at least one image in pod <pod> in namespace <ns>"
	if description != "" {
		if m := regexp.MustCompile(`(?i)Failed to pull at least one image in pod\s+([A-Za-z0-9\-_.]+)\s+in namespace\s+([A-Za-z0-9\-_.]+)`).FindStringSubmatch(description); len(m) >= 3 {
			pod := m[1]
			ns := m[2]
			title = fmt.Sprintf("ImagePullBackOff: %s (%s)", pod, ns)
			if aggKey == "" {
				aggKey = fmt.Sprintf("%s/%s", ns, pod)
			}
			if sev == "" || sev == "medium" {
				sev = "high"
			}
		}
	}
	// 2) K8s Warning: <name> (<namespace>)
	if description != "" && !strings.HasPrefix(strings.ToLower(title), "prometheus alert:") {
		// 允许 name 中包含点（如 bad-pod.12345），聚合键使用点前部分
		reWarn := regexp.MustCompile(`(?i)K8s\s+Warning:\s*([^\(\n]+)\(([^\)]+)\)`)
		if m := reWarn.FindStringSubmatch(description); len(m) >= 3 {
			rawName := strings.TrimSpace(m[1])
			ns := strings.TrimSpace(m[2])
			baseName := strings.TrimSpace(strings.SplitN(rawName, ".", 2)[0])
			if baseName == "" {
				baseName = rawName
			}
			title = fmt.Sprintf("K8s Warning: %s (%s)", baseName, ns)
			if aggKey == "" {
				aggKey = fmt.Sprintf("%s/%s", ns, baseName)
			}
			// 若未显式设置高等级，这类资源级警告默认提高到high
			if sev == "" || sev == "medium" || sev == "low" {
				sev = "high"
			}
		}
	}

	// 3) Prometheus Alert: <name>
	if description != "" {
		if m := regexp.MustCompile(`(?i)Prometheus Alert:\s*([^\n]+)`).FindStringSubmatch(description); len(m) >= 2 {
			name := strings.TrimSpace(m[1])
			if name != "" && !strings.HasPrefix(strings.ToLower(title), "prometheus alert:") {
				title = fmt.Sprintf("Prometheus Alert: %s", name)
			}
		}
	}
	if title == "" {
		title = "Robusta Webhook"
	}
	// sev 已在前面标准化

	// labels/annotations（如果存在）
	// JSON字段序列化为 datatypes.JSON
	var labelsJSON datatypes.JSON
	if lb, ok := body["labels"].(map[string]interface{}); ok {
		if b, err := json.Marshal(lb); err == nil {
			labelsJSON = datatypes.JSON(b)
		}
	}
	var annotationsJSON datatypes.JSON
	if an, ok := body["annotations"].(map[string]interface{}); ok {
		if b, err := json.Marshal(an); err == nil {
			annotationsJSON = datatypes.JSON(b)
		}
	}

	// 指纹（优先使用payload自带，其次用title+cluster+aggregation_key生成）
	fingerprint, _ := body["fingerprint"].(string)
	// aggKey 已在前面声明，以下做补全推断
	if aggKey == "" {
		// 从labels推断：pod/namespace
		if lblMap != nil {
			ns, _ := lblMap["namespace"].(string)
			pod, _ := lblMap["pod"].(string)
			// 兼容不同键名
			if ns == "" {
				ns, _ = lblMap["kubernetes_namespace"].(string)
			}
			if pod == "" {
				pod, _ = lblMap["kubernetes_pod"].(string)
			}
			if ns != "" && pod != "" {
				aggKey = fmt.Sprintf("%s/%s", ns, pod)
			}
		}
		// 从annotations备选
		if aggKey == "" {
			if an, ok := body["annotations"].(map[string]interface{}); ok {
				ns, _ := an["namespace"].(string)
				pod, _ := an["pod"].(string)
				if ns != "" && pod != "" {
					aggKey = fmt.Sprintf("%s/%s", ns, pod)
				}
			}
		}
		// 从描述中解析 "pod <name> in namespace <ns>"
		if aggKey == "" && description != "" {
			re := regexp.MustCompile(`(?i)pod\s+([A-Za-z0-9\-_.]+).*namespace\s+([A-Za-z0-9\-_.]+)`)
			if m := re.FindStringSubmatch(description); len(m) >= 3 {
				aggKey = fmt.Sprintf("%s/%s", m[2], m[1])
			}
		}
	}
	if fingerprint == "" {
		sum := md5.Sum([]byte(fmt.Sprintf("%s:%s:%s", title, clusterID, aggKey)))
		fingerprint = hex.EncodeToString(sum[:])
	}

	// 时间
	var startsAt *time.Time
	if s, ok := body["creation_time"].(string); ok && s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			startsAt = &t
		}
	}

	payloadKey := ""
	if h.storageService != nil {
		if key, err := h.storageService.Save(c.Request.Context(), fmt.Sprintf("alerts/%s", clusterID), raw, "application/json"); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "保存原始告警数据失败",
				"code":    "SAVE_RAW_PAYLOAD_ERROR",
				"details": err.Error(),
			})
			return
		} else {
			payloadKey = key
		}
	}

	alert := &models.Alert{
		Fingerprint:   fingerprint,
		ClusterID:     clusterID,
		Title:         title,
		Description:   description,
		Severity:      sev,
		Status:        "firing",
		Labels:        labelsJSON,
		Annotations:   annotationsJSON,
		StartsAt:      startsAt,
		RawPayloadKey: payloadKey,
	}

	// 确保集群存在（避免外键失败）
	_ = h.clusterService.UpdateHeartbeat(&models.Cluster{ClusterID: clusterID, Name: clusterID, Status: "active"})

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

// IngestRCARequest 接收RCA请求结构
type IngestRCARequest struct {
	AlertID         string                 `json:"alert_id" binding:"required"`
	Status          string                 `json:"status" binding:"required"`
	Summary         string                 `json:"summary"`
	Suspects        map[string]interface{} `json:"suspects"`
	Recommendations map[string]interface{} `json:"recommendations"`
	Attachments     map[string]interface{} `json:"attachments"`
	ErrorMessage    string                 `json:"error_message"`
}

// IngestRCA 接收RCA结果
func (h *IngestHandler) IngestRCA(c *gin.Context) {
	rawBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "读取请求体失败",
			"code":  "READ_BODY_ERROR",
		})
		return
	}

	c.Request.Body = io.NopCloser(bytes.NewBuffer(rawBody))
	var req IngestRCARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "无效的请求数据",
			"code":    "INVALID_REQUEST",
			"details": err.Error(),
		})
		return
	}

	// 解析AlertID
	alertID, err := uuid.Parse(req.AlertID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的告警ID",
			"code":  "INVALID_ALERT_ID",
		})
		return
	}

	payloadKey := ""
	if h.storageService != nil {
		if key, err := h.storageService.Save(c.Request.Context(), fmt.Sprintf("rca/%s", alertID.String()), rawBody, "application/json"); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "保存RCA原始数据失败",
				"code":    "SAVE_RAW_PAYLOAD_ERROR",
				"details": err.Error(),
			})
			return
		} else {
			payloadKey = key
		}
	}

	// 验证RCA状态
	validStatuses := []string{"pending", "running", "completed", "failed", "timeout"}
	if !contains(validStatuses, req.Status) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的RCA状态",
			"code":  "INVALID_RCA_STATUS",
		})
		return
	}

	// 创建RCA运行记录
	var suspectsJSON, recsJSON, attachJSON datatypes.JSON
	if req.Suspects != nil {
		if b, err := json.Marshal(req.Suspects); err == nil {
			suspectsJSON = datatypes.JSON(b)
		}
	}
	if req.Recommendations != nil {
		if b, err := json.Marshal(req.Recommendations); err == nil {
			recsJSON = datatypes.JSON(b)
		}
	}
	if req.Attachments != nil {
		if b, err := json.Marshal(req.Attachments); err == nil {
			attachJSON = datatypes.JSON(b)
		}
	}

	rcaRun := &models.RCARun{
		AlertID:         alertID,
		Status:          req.Status,
		Summary:         &req.Summary,
		Suspects:        suspectsJSON,
		Recommendations: recsJSON,
		Attachments:     attachJSON,
		ErrorMessage:    &req.ErrorMessage,
		RawPayloadKey:   payloadKey,
	}

	// 如果状态是completed或failed，设置完成时间
	if req.Status == "completed" || req.Status == "failed" || req.Status == "timeout" {
		now := time.Now()
		rcaRun.CompletedAt = &now
	}

	// 保存RCA记录
	if err := h.rcaService.CreateRCARun(rcaRun); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "保存RCA记录失败",
			"code":  "SAVE_RCA_ERROR",
		})
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

// ClusterHeartbeatRequest 集群心跳请求结构
type ClusterHeartbeatRequest struct {
	ClusterID   string `json:"cluster_id" binding:"required"`
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Status      string `json:"status"`
}

// ClusterHeartbeat 集群心跳
func (h *IngestHandler) ClusterHeartbeat(c *gin.Context) {
	var req ClusterHeartbeatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "无效的请求数据",
			"code":    "INVALID_REQUEST",
			"details": err.Error(),
		})
		return
	}

	// 更新集群心跳
	cluster := &models.Cluster{
		ClusterID:   req.ClusterID,
		Name:        req.Name,
		Description: req.Description,
		Status:      getOrDefault(req.Status, "active"),
	}

	if err := h.clusterService.UpdateHeartbeat(cluster); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "更新集群心跳失败",
			"code":  "UPDATE_HEARTBEAT_ERROR",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "心跳更新成功",
	})
}

// 辅助函数
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func getOrDefault(value, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}

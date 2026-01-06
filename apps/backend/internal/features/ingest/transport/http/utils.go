package http

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
)

// Severity 严重级别常量
const (
	SeverityLow      = "low"
	SeverityMedium   = "medium"
	SeverityHigh     = "high"
	SeverityCritical = "critical"
)

// RCAStatus RCA状态常量
const (
	RCAStatusPending   = "pending"
	RCAStatusRunning   = "running"
	RCAStatusCompleted = "completed"
	RCAStatusFailed    = "failed"
	RCAStatusTimeout   = "timeout"
)

// AlertStatus 告警状态常量
const (
	AlertStatusFiring   = "firing"
	AlertStatusResolved = "resolved"
)

// readAndRestoreBody 读取并恢复请求体
func readAndRestoreBody(c *gin.Context) ([]byte, error) {
	b, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return nil, err
	}
	c.Request.Body = io.NopCloser(bytes.NewBuffer(b))
	return b, nil
}

// toJSON 将map转换为JSON
func toJSON(m map[string]interface{}) datatypes.JSON {
	if len(m) == 0 {
		return nil
	}
	if b, err := json.Marshal(m); err == nil {
		return datatypes.JSON(b)
	}
	return nil
}

// stringMapToInterface 将string map转换为interface map
func stringMapToInterface(src map[string]string) map[string]interface{} {
	if len(src) == 0 {
		return nil
	}
	result := make(map[string]interface{}, len(src))
	for k, v := range src {
		result[k] = v
	}
	return result
}

// timePtr 将time.Time转换为指针，零值返回nil
func timePtr(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	tCopy := t
	return &tCopy
}

// firstNonEmpty 返回第一个非空字符串
func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	}
	return ""
}

// normalizeSeverity 标准化严重级别（通用版本）
func normalizeSeverityGeneric(severity string) string {
	sev := strings.ToLower(strings.TrimSpace(severity))
	switch sev {
	case "0", "debug":
		return SeverityLow
	case "1", "info", "informational", "notice":
		return SeverityLow
	case "2", "low":
		return SeverityLow
	case "3", "high", "major":
		return SeverityHigh
	case "medium", "moderate", "warn", "warning":
		return SeverityMedium
	case "critical", "urgent", "fatal", "emergency", "disaster":
		return SeverityCritical
	case "error":
		return SeverityHigh
	default:
		return SeverityMedium
	}
}

// normalizeAlertmanagerSeverity 标准化Alertmanager严重级别
func normalizeAlertmanagerSeverity(severity string) string {
	return normalizeSeverityGeneric(severity)
}

// extractClusterNameFromLabels 从标签中提取集群名称
func extractClusterNameFromLabels(labels map[string]string) string {
	for _, key := range []string{"cluster_name", "cluster", "kubernetes_cluster", "robusta_cluster"} {
		if v := strings.TrimSpace(labels[key]); v != "" {
			return v
		}
	}
	return "default"
}

// generateFingerprint 生成指纹哈希
func generateFingerprint(parts ...string) string {
	joined := strings.Join(parts, "|")
	hash := sha256.Sum256([]byte(joined))
	return hex.EncodeToString(hash[:])
}

// generateAlertmanagerFingerprint 生成Alertmanager告警指纹
func generateAlertmanagerFingerprint(clusterName string, labels map[string]string, annotations map[string]string) string {
	parts := []string{strings.TrimSpace(clusterName)}

	// 添加排序后的标签
	if len(labels) > 0 {
		labelKeys := make([]string, 0, len(labels))
		for k := range labels {
			labelKeys = append(labelKeys, k)
		}
		sort.Strings(labelKeys)
		for _, key := range labelKeys {
			parts = append(parts, fmt.Sprintf("label:%s=%s", key, labels[key]))
		}
	}

	// 只添加关键的注释
	if len(annotations) > 0 {
		annotationKeys := make([]string, 0, len(annotations))
		for k := range annotations {
			annotationKeys = append(annotationKeys, k)
		}
		sort.Strings(annotationKeys)
		for _, key := range annotationKeys {
			if key == "summary" || key == "description" {
				parts = append(parts, fmt.Sprintf("annotation:%s=%s", key, annotations[key]))
			}
		}
	}

	return generateFingerprint(parts...)
}

// AlertmanagerAlertConverter Alertmanager告警转换器
type AlertmanagerAlertConverter struct {
	alert           AlertmanagerAlert
	webhookStatus   string
	index           int
	clusterName     string
	title           string
	description     string
	severity        string
	status          string
	fingerprint     string
	labelsJSON      datatypes.JSON
	annotationsJSON datatypes.JSON
	startsAt        *time.Time
	endsAt          *time.Time
}

// NewAlertmanagerAlertConverter 创建Alertmanager告警转换器
func NewAlertmanagerAlertConverter(alert AlertmanagerAlert, webhookStatus string, index int) *AlertmanagerAlertConverter {
	return &AlertmanagerAlertConverter{
		alert:         alert,
		webhookStatus: webhookStatus,
		index:         index,
	}
}

// Convert 转换为内部Alert模型
func (c *AlertmanagerAlertConverter) Convert() *AlertData {
	c.extractClusterName()
	c.buildTitle()
	c.buildDescription()
	c.normalizeSeverity()
	c.determineStatus()
	c.buildFingerprint()
	c.buildMetadata()
	c.parseTimes()

	return &AlertData{
		ClusterName: c.clusterName,
		Title:       c.title,
		Description: c.description,
		Severity:    c.severity,
		Status:      c.status,
		Fingerprint: c.fingerprint,
		Labels:      c.labelsJSON,
		Annotations: c.annotationsJSON,
		StartsAt:    c.startsAt,
		EndsAt:      c.endsAt,
	}
}

// extractClusterName 提取集群名称
func (c *AlertmanagerAlertConverter) extractClusterName() {
	c.clusterName = extractClusterNameFromLabels(c.alert.Labels)
}

// buildTitle 构建标题
func (c *AlertmanagerAlertConverter) buildTitle() {
	c.title = strings.TrimSpace(c.alert.Labels["alertname"])
	if c.title == "" {
		c.title = fmt.Sprintf("Alertmanager 告警-%d", c.index+1)
	}
}

// buildDescription 构建描述
func (c *AlertmanagerAlertConverter) buildDescription() {
	c.description = firstNonEmpty(
		c.alert.Annotations["description"],
		c.alert.Annotations["summary"],
		c.alert.Annotations["message"],
	)
}

// normalizeSeverity 标准化严重级别
func (c *AlertmanagerAlertConverter) normalizeSeverity() {
	c.severity = normalizeAlertmanagerSeverity(c.alert.Labels["severity"])
}

// determineStatus 确定状态
func (c *AlertmanagerAlertConverter) determineStatus() {
	c.status = strings.ToLower(strings.TrimSpace(c.alert.Status))
	if c.status == "" {
		c.status = strings.ToLower(strings.TrimSpace(c.webhookStatus))
	}
	if c.status == "" {
		c.status = AlertStatusFiring
	}
}

// buildFingerprint 构建指纹
// Alertmanager 告警总是生成自己的指纹，不使用 Alertmanager 提供的
func (c *AlertmanagerAlertConverter) buildFingerprint() {
	if fp := strings.TrimSpace(c.alert.Fingerprint); fp != "" {
		c.fingerprint = fp
		return
	}
	c.fingerprint = generateAlertmanagerFingerprint(c.clusterName, c.alert.Labels, c.alert.Annotations)
}

// buildMetadata 构建元数据
func (c *AlertmanagerAlertConverter) buildMetadata() {
	c.labelsJSON = toJSON(stringMapToInterface(c.alert.Labels))

	annotationMap := stringMapToInterface(c.alert.Annotations)
	if annotationMap == nil {
		annotationMap = make(map[string]interface{})
	}
	if c.alert.GeneratorURL != "" {
		annotationMap["generatorURL"] = c.alert.GeneratorURL
	}
	c.annotationsJSON = toJSON(annotationMap)
}

// parseTimes 解析时间
func (c *AlertmanagerAlertConverter) parseTimes() {
	c.startsAt = timePtr(c.alert.StartsAt)
	c.endsAt = timePtr(c.alert.EndsAt)
}

// AlertData 告警数据结构
type AlertData struct {
	ClusterName string
	Title       string
	Description string
	Severity    string
	Status      string
	Fingerprint string
	Labels      datatypes.JSON
	Annotations datatypes.JSON
	StartsAt    *time.Time
	EndsAt      *time.Time
}

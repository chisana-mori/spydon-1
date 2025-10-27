package api

import (
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
)

// contains 检查字符串切片是否包含指定项
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// getOrDefault 返回值或默认值
func getOrDefault(value, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}

// min 返回两个整数中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// extractClusterID 从多个来源提取cluster_id
func extractClusterID(c *gin.Context, finding *RobustaFinding) string {
	// 如果有gin.Context，尝试从header获取
	if c != nil {
		if clusterID := c.GetHeader("X-Robusta-Cluster-ID"); clusterID != "" {
			return clusterID
		}
	}

	if finding.Subject.Labels != nil {
		for _, key := range []string{"robusta_cluster", "cluster", "kubernetes_cluster", "cluster_id", "cluster_name"} {
			if v, ok := finding.Subject.Labels[key].(string); ok && v != "" {
				return v
			}
		}
	}

	if finding.SilenceLabels != nil {
		for _, key := range []string{"robusta_cluster", "cluster", "kubernetes_cluster", "cluster_id", "cluster_name"} {
			if v, ok := finding.SilenceLabels[key].(string); ok && v != "" {
				return v
			}
		}
	}

	if finding.Description != "" {
		re := regexp.MustCompile(`(?i)(source|cluster|cluster_name)\s*:\s*([A-Za-z0-9\-_.:]+)`)
		if m := re.FindStringSubmatch(finding.Description); len(m) >= 3 {
			return strings.TrimSpace(m[2])
		}
	}

	return "kind"
}

// normalizeSeverity 标准化严重级别
func normalizeSeverity(severity string) string {
	sev := strings.ToLower(strings.TrimSpace(severity))
	switch sev {
	case "0", "debug":
		return "low"
	case "1", "info", "informational":
		return "low"
	case "2", "low":
		return "low"
	case "3", "high":
		return "high"
	case "medium", "moderate":
		return "medium"
	case "critical", "urgent":
		return "critical"
	default:
		return "medium"
	}
}

// ValidSeverities 有效的严重级别列表
var ValidSeverities = []string{"low", "medium", "high", "critical"}

// ValidRCAStatuses 有效的RCA状态列表
var ValidRCAStatuses = []string{"pending", "running", "completed", "failed", "timeout"}

package api

import (
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
)

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

// normalizeSeverity 标准化严重级别（兼容旧代码）
func normalizeSeverity(severity string) string {
	return normalizeSeverityGeneric(severity)
}

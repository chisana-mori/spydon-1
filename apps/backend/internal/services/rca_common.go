package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"robusta-web/backend/internal/models"

	"gorm.io/datatypes"
)

var severityPriorityOrder = []models.AlertSeverity{
	models.AlertSeverityCritical,
	models.AlertSeverityHigh,
	models.AlertSeverityError,
	models.AlertSeverityMedium,
	models.AlertSeverityWarning,
	models.AlertSeverityLow,
	models.AlertSeverityInfo,
}

var severityPriorityCaseExpr = buildSeverityPriorityCaseExpr()

func buildSeverityPriorityCaseExpr() string {
	var builder strings.Builder
	builder.WriteString("CASE alerts.severity ")
	maxRank := len(severityPriorityOrder)
	for idx, severity := range severityPriorityOrder {
		builder.WriteString(fmt.Sprintf("WHEN '%s' THEN %d ", severity, maxRank-idx))
	}
	builder.WriteString("ELSE 0 END")
	return builder.String()
}

func normalizeSeverity(severity string) models.AlertSeverity {
	return models.AlertSeverity(strings.ToLower(strings.TrimSpace(severity)))
}

// AutoRCAConfig Auto-RCA配置
type AutoRCAConfig struct {
	Enabled           bool                   `json:"enabled"`
	RateLimit         int                    `json:"rate_limit"`
	Period            int                    `json:"period"`
	AllowedSeverities []models.AlertSeverity `json:"allowed_severities"`
}

func defaultAllowedSeverities() []models.AlertSeverity {
	return []models.AlertSeverity{
		models.AlertSeverityCritical,
		models.AlertSeverityHigh,
		models.AlertSeverityError,
	}
}

// fallbackAutoRCAConfig 降级配置（仅在数据库查询失败时使用）
func fallbackAutoRCAConfig() *AutoRCAConfig {
	return &AutoRCAConfig{
		Enabled:           false, // 降级时禁用 Auto-RCA
		RateLimit:         10,
		Period:            60,
		AllowedSeverities: defaultAllowedSeverities(),
	}
}

// GetDefaultAutoRCAConfig 获取默认配置（用于初始化）
func GetDefaultAutoRCAConfig() *AutoRCAConfig {
	return &AutoRCAConfig{
		Enabled:           false, // 默认关闭，需要手动开启
		RateLimit:         10,
		Period:            60,
		AllowedSeverities: defaultAllowedSeverities(),
	}
}

// ProcessRCARequest 处理RCA请求参数
type ProcessRCARequest struct {
	AlertID         uint64
	Status          string
	Summary         string
	Suspects        map[string]interface{}
	Recommendations map[string]interface{}
	Attachments     map[string]interface{}
	ErrorMessage    string
	RawBody         []byte
	ClientIP        string
	UserAgent       string
}

// savePayload 保存原始数据
func (s *RCAService) savePayload(ctx context.Context, keyPrefix string, data []byte, contentType string) (string, error) {
	if s.payloadStorage == nil {
		return "", nil
	}

	key, err := s.payloadStorage.Save(ctx, keyPrefix, data, contentType)
	if err != nil {
		return "", err
	}

	return key, nil
}

// isValidRCAStatus 验证RCA状态
func isValidRCAStatus(status string) bool {
	switch models.RCAStatus(status) {
	case models.RCAStatusPending,
		models.RCAStatusRunning,
		models.RCAStatusCompleted,
		models.RCAStatusFailed,
		models.RCAStatusTimeout,
		models.RCAStatusQueued:
		return true
	default:
		return false
	}
}

func toJSON(v interface{}) datatypes.JSON {
	if v == nil {
		return datatypes.JSON([]byte("{}"))
	}
	b, err := json.Marshal(v)
	if err != nil {
		return datatypes.JSON([]byte("{}"))
	}
	return datatypes.JSON(b)
}

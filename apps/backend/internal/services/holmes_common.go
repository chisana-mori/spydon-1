package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"robusta-web/backend/internal/models"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func (s *HolmesService) buildInvestigatePayload(alert *models.Alert, opts InvestigateOptions) ([]byte, error) {
	language := opts.Language
	if language == "" {
		language = "zh-CN"
	}
	includeToolCalls := true
	if opts.IncludeToolCalls != nil {
		includeToolCalls = *opts.IncludeToolCalls
	}
	source := opts.Source
	if source == "" {
		source = "robusta"
	}

	description := strings.TrimSpace(alert.Description)
	if description != "" {
		description += "\n\n"
	}
	description += fmt.Sprintf("请用%s回答全部内容。", languageDisplayName(language))

	// context 只包含辅助信息，不重复告警数据
	contextData := map[string]interface{}{
		"response_language": language,
	}

	payload := map[string]interface{}{
		"source":                    source,
		"title":                     alert.Title,
		"description":               description,
		"force_refresh":             opts.ForceRefresh,
		"prefer_cache":              opts.PreferCache,
		"include_tool_calls":        includeToolCalls,
		"include_tool_call_results": includeToolCalls,
		"source_instance_id":        "backend",
		"prompt_template":           "builtin://generic_investigation.jinja2",
		"cache_control":             map[string]bool{"bypass_cache": opts.ForceRefresh, "prefer_cache": opts.PreferCache},
		"context":                   contextData,
		// subject 包含完整的告警信息，作为分析的主体对象
		"subject": s.buildSubjectContext(alert),
	}

	// 知识库内容添加到 context 中
	if opts.KnowledgeBase != "" {
		contextData["knowledge_base"] = opts.KnowledgeBase
	}

	return json.Marshal(payload)
}

// buildSubjectContext 构建告警主体信息，包含完整的告警数据供 AI 分析
func (s *HolmesService) buildSubjectContext(alert *models.Alert) map[string]interface{} {
	return map[string]interface{}{
		"alert_id":    alert.ID.String(),
		"fingerprint": alert.Fingerprint,
		"title":       alert.Title,
		"description": alert.Description,
		"cluster_id":  alert.ClusterID,
		"severity":    alert.Severity,
		"status":      alert.Status,
		"labels":      decodeJSONMap(alert.Labels),
		"annotations": decodeJSONMap(alert.Annotations),
		"created_at":  alert.CreatedAt.Format(time.RFC3339),
		"starts_at":   formatTimePtr(alert.StartsAt),
		"ends_at":     formatTimePtr(alert.EndsAt),
	}
}

func decodeJSONMap(data datatypes.JSON) map[string]interface{} {
	if len(data) == 0 {
		return map[string]interface{}{}
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return map[string]interface{}{}
	}
	return result
}

func formatTimePtr(t *time.Time) interface{} {
	if t == nil {
		return nil
	}
	return t.Format(time.RFC3339)
}

func languageDisplayName(code string) string {
	switch strings.ToLower(code) {
	case "zh-cn", "zh":
		return "中文"
	case "en", "en-us":
		return "英文"
	default:
		return "中文"
	}
}

// GetAnalysisByAlertID 根据告警ID获取分析结果
func (s *HolmesService) GetAnalysisByAlertID(alertID string) ([]*models.RCARun, error) {
	var rcaRuns []*models.RCARun

	err := s.db.Where("alert_id = ?", alertID).
		Order("created_at DESC").
		Find(&rcaRuns).Error
	if err != nil {
		return nil, fmt.Errorf("查询RCA运行记录失败: %w", err)
	}

	return rcaRuns, nil
}

func (s *HolmesService) getAlertByID(alertID string) (*models.Alert, error) {
	var alert models.Alert
	err := s.db.Where("id = ?", alertID).First(&alert).Error
	if err != nil {
		return nil, err
	}
	return &alert, nil
}

func (s *HolmesService) getRunningAnalysis(alertID string) (*models.RCARun, error) {
	var rcaRun models.RCARun
	err := s.db.Where("alert_id = ? AND status IN (?)", alertID, []string{string(models.RCAStatusPending), string(models.RCAStatusRunning)}).
		First(&rcaRun).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &rcaRun, nil
}

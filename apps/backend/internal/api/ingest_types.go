package api

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

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

// FindingSeverity 严重级别枚举
type FindingSeverity struct {
	Value int `json:"_value_"`
}

const (
	FindingSeverityDebug string = "DEBUG"
	FindingSeverityInfo  string = "INFO"
	FindingSeverityLow   string = "LOW"
	FindingSeverityHigh  string = "HIGH"
)

// FindingStatus 状态枚举
type FindingStatus string

const (
	FindingStatusFiring   FindingStatus = "FIRING"
	FindingStatusResolved FindingStatus = "RESOLVED"
)

// LinkType 链接类型枚举
type LinkType string

const (
	LinkTypeVideo                    LinkType = "video"
	LinkTypePrometheusGeneratorURL   LinkType = "prometheus_generator_url"
	LinkTypeOpsgenieListAlertByAlias LinkType = "opsgenie_list_alert_by_alias"
)

// EnrichmentType Enrichment类型枚举
type EnrichmentType interface{}

const (
	EnrichmentTypeGraph                string = "graph"
	EnrichmentTypeAIAnalysis           string = "ai_analysis"
	EnrichmentTypeNodeInfo             string = "node_info"
	EnrichmentTypeContainerInfo        string = "container_info"
	EnrichmentTypeK8sEvents            string = "k8s_events"
	EnrichmentTypeAlertLabels          string = "alert_labels"
	EnrichmentTypeDiff                 string = "diff"
	EnrichmentTypeTextFile             string = "text_file"
	EnrichmentTypeCrashInfo            string = "crash_info"
	EnrichmentTypeImagePullBackoffInfo string = "image_pull_backoff_info"
	EnrichmentTypePendingPodInfo       string = "pending_pod_info"
)

// FindingSource 来源类型
type FindingSource struct {
	Value string `json:"_value_"`
}

// FindingType Finding类型
type FindingType struct {
	Value string `json:"_value_"`
}

// FindingSubjectType Subject类型
type FindingSubjectType struct {
	Value string `json:"_value_"`
}

// EnumValue 通用的 Enum 值解析器（兼容旧格式）
type EnumValue struct {
	Value interface{} `json:"_value_"`
}

func (e *EnumValue) String() string {
	if e.Value == nil {
		return ""
	}
	switch v := e.Value.(type) {
	case string:
		return v
	case float64:
		return fmt.Sprintf("%.0f", v)
	case int:
		return fmt.Sprintf("%d", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func (e *EnumValue) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		e.Value = str
		return nil
	}

	var num float64
	if err := json.Unmarshal(data, &num); err == nil {
		e.Value = num
		return nil
	}

	var obj struct {
		Value interface{} `json:"_value_"`
	}
	if err := json.Unmarshal(data, &obj); err == nil {
		e.Value = obj.Value
		return nil
	}

	e.Value = ""
	return nil
}

// FlexibleTime 灵活的时间类型，支持多种格式
type FlexibleTime struct {
	time.Time
}

// UnmarshalJSON 自定义 JSON 反序列化，支持多种时间格式
func (ft *FlexibleTime) UnmarshalJSON(data []byte) error {
	// 去除引号
	str := strings.Trim(string(data), `"`)
	if str == "" || str == "null" {
		return nil
	}

	// 支持的时间格式列表
	formats := []string{
		time.RFC3339,                       // "2006-01-02T15:04:05Z07:00"
		time.RFC3339Nano,                   // "2006-01-02T15:04:05.999999999Z07:00"
		"2006-01-02 15:04:05.999999Z07:00", // Robusta 格式: "2025-10-25 07:27:42.395545+00:00"
		"2006-01-02 15:04:05Z07:00",        // 无微秒版本
		"2006-01-02 15:04:05",              // 无时区版本
		time.DateTime,                      // "2006-01-02 15:04:05"
	}

	var lastErr error
	for _, format := range formats {
		t, err := time.Parse(format, str)
		if err == nil {
			ft.Time = t
			return nil
		}
		lastErr = err
	}

	return fmt.Errorf("无法解析时间 '%s': %w", str, lastErr)
}

// RobustaFinding Robusta Finding完整结构
type RobustaFinding struct {
	ID string `json:"id"`

	Title          string          `json:"title"`
	FindingType    FindingType     `json:"finding_type,omitempty"`
	Failure        bool            `json:"failure"`
	Description    string          `json:"description,omitempty"`
	Source         FindingSource   `json:"source,omitempty"`
	AggregationKey string          `json:"aggregation_key,omitempty"`
	Severity       FindingSeverity `json:"severity,omitempty"`
	Category       string          `json:"category,omitempty"` // TODO: Python sets None and comments "fill real category"
	Subject        RobustaSubject  `json:"subject"`

	Enrichments []RobustaEnrichment `json:"enrichments,omitempty"`
	Links       []RobustaLink       `json:"links,omitempty"`

	// Service resolved via TopServiceResolver in Python
	Service    *Service `json:"service,omitempty"`
	ServiceKey string   `json:"service_key,omitempty"`

	InvestigateURI string `json:"investigate_uri,omitempty"`

	AddSilenceURL bool                   `json:"add_silence_url,omitempty"`
	SilenceLabels map[string]interface{} `json:"silence_labels,omitempty"` // Python used Dict[Any,Any]

	CreationDate *string `json:"creation_date,omitempty"` // Python accepts str or None
	Fingerprint  string  `json:"fingerprint,omitempty"`

	StartsAt FlexibleTime  `json:"starts_at"`
	EndsAt   *FlexibleTime `json:"ends_at,omitempty"`

	Dirty bool `json:"dirty"`
}

// RobustaSubject Subject信息
type RobustaSubject struct {
	Name        string                 `json:"name"`
	SubjectType EnumValue              `json:"subject_type"`
	Namespace   string                 `json:"namespace"`
	Node        string                 `json:"node"`
	Container   string                 `json:"container"`
	Labels      map[string]interface{} `json:"labels"`
	Annotations map[string]interface{} `json:"annotations"`
}

// RobustaEnrichment Enrichment数据
type RobustaEnrichment struct {
	Blocks         []RobustaBlock         `json:"blocks"`
	Annotations    map[string]interface{} `json:"annotations"`
	EnrichmentType *EnrichmentType        `json:"enrichment_type,omitempty"`
	Title          string                 `json:"title"`
}

// RobustaBlock Enrichment中的Block（支持多种类型）
type RobustaBlock struct {
	Type        string                   `json:"type"`
	Hidden      bool                     `json:"hidden,omitempty"`
	HTMLClass   string                   `json:"html_class,omitempty"`
	Text        string                   `json:"text,omitempty"`
	Filename    string                   `json:"filename,omitempty"`
	Contents    string                   `json:"contents,omitempty"`
	Headers     []string                 `json:"headers,omitempty"`
	Rows        [][]interface{}          `json:"rows,omitempty"`
	ColumnWidth []interface{}            `json:"column_width,omitempty"`
	TableName   string                   `json:"table_name,omitempty"`
	Metadata    map[string]interface{}   `json:"metadata,omitempty"`
	Events      []map[string]interface{} `json:"events,omitempty"`
	JSON        interface{}              `json:"json,omitempty"`
	Title       string                   `json:"title,omitempty"`
	Items       []string                 `json:"items,omitempty"`
	Diffs       []map[string]interface{} `json:"diffs,omitempty"`
	Old         string                   `json:"old,omitempty"`
	New         string                   `json:"new,omitempty"`
}

// RobustaLink 链接信息
type RobustaLink struct {
	URL  string    `json:"url"`
	Name string    `json:"name"`
	Type *LinkType `json:"type,omitempty"`
}

// Service 服务信息（可选）
type Service struct {
	ResourceType string `json:"resource_type,omitempty"`
	Name         string `json:"name,omitempty"`
	ResourceKey  string `json:"resource_key,omitempty"`
}

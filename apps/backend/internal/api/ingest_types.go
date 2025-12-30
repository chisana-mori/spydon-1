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
	ClusterName string                 `json:"cluster_name" binding:"required"`
	Title       string                 `json:"title" binding:"required"`
	Description string                 `json:"description"`
	Severity    string                 `json:"severity" binding:"required,oneof=low medium high critical"`
	Status      string                 `json:"status" binding:"omitempty,oneof=firing resolved silenced"`
	Labels      map[string]interface{} `json:"labels"`
	Annotations map[string]interface{} `json:"annotations"`
	StartsAt    *time.Time             `json:"starts_at"`
	EndsAt      *time.Time             `json:"ends_at"`
}

// AlertmanagerWebhookRequest Alertmanager webhook 的请求体
type AlertmanagerWebhookRequest struct {
	Receiver          string                       `json:"receiver"`
	Status            string                       `json:"status"`
	Alerts            []AlertmanagerAlert          `json:"alerts" binding:"required,dive"`
	GroupLabels       map[string]string            `json:"groupLabels"`
	CommonLabels      map[string]string            `json:"commonLabels"`
	CommonAnnotations map[string]string            `json:"commonAnnotations"`
	ExternalURL       string                       `json:"externalURL"`
	Version           string                       `json:"version"`
	GroupKey          string                       `json:"groupKey"`
	TruncatedAlerts   int                          `json:"truncatedAlerts"`
	Metadata          map[string]map[string]string `json:"metadata"`
}

// AlertmanagerAlert 单条 Alertmanager 告警
type AlertmanagerAlert struct {
	Status       string            `json:"status" binding:"required"`
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations"`
	StartsAt     time.Time         `json:"startsAt"`
	EndsAt       time.Time         `json:"endsAt"`
	GeneratorURL string            `json:"generatorURL"`
	Fingerprint  string            `json:"fingerprint"`
}

// IngestRCARequest 接收RCA请求结构
type IngestRCARequest struct {
	AlertID         uint64                 `json:"alert_id" binding:"required,gt=0"`
	Status          string                 `json:"status" binding:"required,oneof=pending running completed failed timeout"`
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

// Finding严重级别常量
const (
	FindingSeverityDebug = "DEBUG"
	FindingSeverityInfo  = "INFO"
	FindingSeverityLow   = "LOW"
	FindingSeverityHigh  = "HIGH"
)

// FindingStatus 状态枚举
type FindingStatus string

// Finding状态常量
const (
	FindingStatusFiring   FindingStatus = "FIRING"
	FindingStatusResolved FindingStatus = "RESOLVED"
)

// LinkType 链接类型枚举
type LinkType string

// 链接类型常量
const (
	LinkTypeVideo                    LinkType = "video"
	LinkTypePrometheusGeneratorURL   LinkType = "prometheus_generator_url"
	LinkTypeOpsgenieListAlertByAlias LinkType = "opsgenie_list_alert_by_alias"
)

// EnrichmentType Enrichment类型枚举
type EnrichmentType interface{}

// Enrichment类型常量
const (
	EnrichmentTypeGraph                = "graph"
	EnrichmentTypeAIAnalysis           = "ai_analysis"
	EnrichmentTypeNodeInfo             = "node_info"
	EnrichmentTypeContainerInfo        = "container_info"
	EnrichmentTypeK8sEvents            = "k8s_events"
	EnrichmentTypeAlertLabels          = "alert_labels"
	EnrichmentTypeDiff                 = "diff"
	EnrichmentTypeTextFile             = "text_file"
	EnrichmentTypeCrashInfo            = "crash_info"
	EnrichmentTypeImagePullBackoffInfo = "image_pull_backoff_info"
	EnrichmentTypePendingPodInfo       = "pending_pod_info"
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

// 支持的时间格式列表
var supportedTimeFormats = []string{
	time.RFC3339,                       // "2006-01-02T15:04:05Z07:00"
	time.RFC3339Nano,                   // "2006-01-02T15:04:05.999999999Z07:00"
	"2006-01-02 15:04:05.999999Z07:00", // Robusta 格式: "2025-10-25 07:27:42.395545+00:00"
	"2006-01-02 15:04:05Z07:00",        // 无微秒版本
	"2006-01-02 15:04:05",              // 无时区版本
	time.DateTime,                      // "2006-01-02 15:04:05"
}

// UnmarshalJSON 自定义 JSON 反序列化，支持多种时间格式
func (ft *FlexibleTime) UnmarshalJSON(data []byte) error {
	// 去除引号
	str := strings.Trim(string(data), `"`)
	if str == "" || str == "null" {
		return nil
	}

	// 尝试所有支持的格式
	for _, format := range supportedTimeFormats {
		if t, err := time.Parse(format, str); err == nil {
			ft.Time = t
			return nil
		}
	}

	// 所有格式都失败，返回详细错误
	return fmt.Errorf("无法解析时间字符串 '%s'，支持的格式包括: RFC3339, RFC3339Nano, 以及多种自定义格式", str)
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

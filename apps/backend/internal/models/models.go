package models

import (
	"strconv"
	"time"

	"gorm.io/datatypes"
)

// ParseID 将字符串解析为uint64 ID
func ParseID(id string) (uint64, error) {
	return strconv.ParseUint(id, 10, 64)
}

// FormatID 将uint64 ID转换为字符串
func FormatID(id uint64) string {
	return strconv.FormatUint(id, 10)
}

// BaseModel 基础模型，包含通用字段
type BaseModel struct {
	ID        uint64    `json:"id" gorm:"primaryKey;autoIncrement"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Cluster 集群模型 (与 Kite 兼容的格式)
type Cluster struct {
	ID            uint             `json:"id" gorm:"primaryKey;autoIncrement"`
	CreatedAt     time.Time        `json:"created_at"`
	UpdatedAt     time.Time        `json:"updated_at"`
	Name          string           `json:"name" gorm:"type:varchar(100);uniqueIndex;not null;column:clustername"`
	Description   string           `json:"description" gorm:"type:text;column:desc"`
	Config        KiteSecretString `json:"config" gorm:"type:text;column:config"` // KubeConfig - 自动加密
	PrometheusURL string           `json:"prometheus_url" gorm:"type:varchar(255)"`
	InCluster     bool             `json:"in_cluster" gorm:"type:boolean;default:false"`
	IsDefault     bool             `json:"is_default" gorm:"type:boolean;default:false"`
	Enable        bool             `json:"enable" gorm:"type:boolean;default:true"`

	// Spydon 特有字段 (Kite 不使用，但不影响兼容性)
	ClusterID string `json:"cluster_id" gorm:"column:cluster_id;type:varchar(255)"`
	Status    string `json:"status" gorm:"type:varchar(32);default:active"`

	// navy字段
	ClusterVersion string   `json:"cluster_version" gorm:"default:'';size:128;column:cluster_version"`
	Idc            string   `json:"idc" gorm:"default:'';size:36;column:idc"`   // gl ft wg qf
	Zone           string   `json:"zone" gorm:"default:'';size:36;column:zone"` // egt
	KubeConfig     string   `json:"kube_config" gorm:"default:'';size:1024;column:kubeconfig"`
	Updator        string   `json:"updator" gorm:"default:'';size:128;column:updater"`
	FlowType       string   `json:"flow_type" gorm:"default:'';size:255"`
	ClusterGroup   string   `json:"cluster_group" gorm:"default:'';size:128"` // 同IDC中上的集群分组信息
	Purpose        string   `json:"purpose" gorm:"default:'';size:255"`
	Arch           string   `json:"arch" gorm:"default:'';size:255"`
	Priority       int      `json:"priority" gorm:"default:0;column:priority"`
	MasterIPs      []string `json:"master_ips" gorm:"-"`
	EtcdIPs        []string `json:"etcd_ips" gorm:"-"`
	EtcdEventIPs   []string `json:"etcd_event_ips" gorm:"-"`
}

type Alert struct {
	BaseModel
	Fingerprint   string         `json:"fingerprint" gorm:"type:varchar(191);not null"`
	ClusterName   string         `json:"cluster_name" gorm:"column:cluster_name;type:varchar(255);not null"`
	ClusterID     string         `json:"cluster_id" gorm:"column:cluster_id;type:varchar(255)"`
	Title         string         `json:"title" gorm:"type:text;not null"`
	Description   string         `json:"description" gorm:"type:text"`
	Severity      string         `json:"severity" gorm:"type:varchar(32);not null"`
	Status        string         `json:"status" gorm:"type:varchar(32);default:firing"`
	Labels        datatypes.JSON `json:"labels" gorm:"type:json"`
	Annotations   datatypes.JSON `json:"annotations" gorm:"type:json"`
	StartsAt      *time.Time     `json:"starts_at"`
	EndsAt        *time.Time     `json:"ends_at"`
	RawPayloadKey string         `json:"raw_payload_key" gorm:"type:text"`

	// 关联关系
	RCARuns []RCARun `json:"rca_runs,omitempty" gorm:"foreignKey:AlertID;constraint:OnDelete:CASCADE,OnUpdate:CASCADE;"`
}

// RCARun RCA运行记录模型
type RCARun struct {
	BaseModel
	AlertID         uint64         `json:"alert_id" gorm:"type:bigint unsigned;not null"`
	Status          string         `json:"status" gorm:"type:varchar(32);not null"`
	Summary         *string        `json:"summary"`
	Suspects        datatypes.JSON `json:"suspects" gorm:"type:json"`
	Recommendations datatypes.JSON `json:"recommendations" gorm:"type:json"`
	Attachments     datatypes.JSON `json:"attachments" gorm:"type:json"`
	StartedAt       time.Time      `json:"started_at"`
	CompletedAt     *time.Time     `json:"completed_at"`
	ErrorMessage    *string        `json:"error_message"`
	RawPayloadKey   string         `json:"raw_payload_key" gorm:"type:text"`

	// 关联关系 - 不使用数据库外键约束
	Alert *Alert `json:"alert,omitempty" gorm:"foreignKey:AlertID;constraint:OnDelete:CASCADE,OnUpdate:CASCADE;"`
}

// AuditLog 审计日志模型
type AuditLog struct {
	BaseModel
	UserID       *uint64        `json:"user_id" gorm:"type:bigint unsigned"`
	Action       string         `json:"action" gorm:"type:varchar(64);not null"`
	ResourceType string         `json:"resource_type" gorm:"type:varchar(64)"`
	ResourceID   *uint64        `json:"resource_id" gorm:"type:bigint unsigned"`
	Details      datatypes.JSON `json:"details" gorm:"type:json"`
	IPAddress    string         `json:"ip_address"`
	UserAgent    string         `json:"user_agent"`
}

// AlertStatus 告警状态枚举
type AlertStatus string

const (
	AlertStatusFiring   AlertStatus = "firing"
	AlertStatusResolved AlertStatus = "resolved"
	AlertStatusSilenced AlertStatus = "silenced"
)

// AlertSeverity 告警严重级别枚举
type AlertSeverity string

const (
	AlertSeverityInfo     AlertSeverity = "info"
	AlertSeverityWarning  AlertSeverity = "warning"
	AlertSeverityError    AlertSeverity = "error"
	AlertSeverityLow      AlertSeverity = "low"
	AlertSeverityMedium   AlertSeverity = "medium"
	AlertSeverityHigh     AlertSeverity = "high"
	AlertSeverityCritical AlertSeverity = "critical"
)

// RCAStatus RCA状态枚举
type RCAStatus string

const (
	RCAStatusPending   RCAStatus = "pending"
	RCAStatusRunning   RCAStatus = "running"
	RCAStatusCompleted RCAStatus = "completed"
	RCAStatusFailed    RCAStatus = "failed"
	RCAStatusTimeout   RCAStatus = "timeout"
	RCAStatusQueued    RCAStatus = "queued"
)

// ClusterStatus 集群状态枚举
type ClusterStatus string

const (
	ClusterStatusInit     ClusterStatus = "Init"     // 集群初始化
	ClusterStatusPending  ClusterStatus = "Pending"  // 集群维护中
	ClusterStatusRunning  ClusterStatus = "Running"  // 运行中
	ClusterStatusOffline  ClusterStatus = "Offline"  // 已下线
)

// TableName 方法用于指定表名
// Cluster 与 Kite 共用，不加前缀
func (Cluster) TableName() string {
	return "k8s_clusters"
}

// 以下表加 spydon_ 前缀以避免与 Kite 冲突
func (Alert) TableName() string {
	return "spydon_alerts"
}

func (RCARun) TableName() string {
	return "spydon_rca_runs"
}

func (AuditLog) TableName() string {
	return "spydon_audit_logs"
}

// RCAStats RCA统计信息
type RCAStats struct {
	Total              int64                    `json:"total"`
	ByStatus           []map[string]interface{} `json:"by_status"`
	SuccessRate        float64                  `json:"success_rate"`
	AvgDurationSeconds float64                  `json:"avg_duration_seconds"`
}

// User 用户模型
type User struct {
	BaseModel
	Username      string     `json:"username" gorm:"type:varchar(191);uniqueIndex;not null"`
	Email         string     `json:"email" gorm:"type:varchar(191);uniqueIndex;not null"`
	Name          string     `json:"name" gorm:"type:varchar(255)"`
	Picture       string     `json:"picture" gorm:"type:text"`
	IsAdmin       bool       `json:"is_admin" gorm:"default:false"` // 是否为管理员，默认为普通用户
	EmailVerified bool       `json:"email_verified" gorm:"default:false"`
	Provider      string     `json:"provider" gorm:"type:varchar(64);default:local"`
	ProviderID    string     `json:"provider_id" gorm:"type:varchar(191)"`
	LastLoginAt   *time.Time `json:"last_login_at"`
}

// RefreshToken 刷新令牌模型
type RefreshToken struct {
	BaseModel
	UserID    uint64    `json:"user_id" gorm:"type:bigint unsigned;not null"`
	TokenHash string    `json:"-" gorm:"column:token;type:varchar(255);uniqueIndex;not null"`
	ExpiresAt time.Time `json:"expires_at" gorm:"not null"`
	User      User      `json:"user" gorm:"foreignKey:UserID"`
}

// TableName 指定表名
func (User) TableName() string {
	return "spydon_users"
}

func (RefreshToken) TableName() string {
	return "spydon_refresh_tokens"
}

// APIKey API密钥模型
type APIKey struct {
	BaseModel
	UserID      uint64     `json:"user_id" gorm:"type:bigint unsigned;not null"`
	Name        string     `json:"name" gorm:"type:varchar(255);not null"`            // API Key名称/描述
	Key         string     `json:"key" gorm:"type:varchar(255);uniqueIndex;not null"` // API Key值（加密存储）
	KeyPrefix   string     `json:"key_prefix" gorm:"type:varchar(64);not null"`       // Key前缀（用于显示）
	LastUsedAt  *time.Time `json:"last_used_at"`                                      // 最后使用时间
	ExpiresAt   *time.Time `json:"expires_at"`                                        // 过期时间（可选）
	IsActive    bool       `json:"is_active" gorm:"default:true"`                     // 是否激活
	Permissions string     `json:"permissions" gorm:"type:varchar(32);default:read"`  // 权限范围（read/write/admin）
	User        User       `json:"user,omitempty" gorm:"foreignKey:UserID"`           // 关联用户
}

// TableName 指定表名
func (APIKey) TableName() string {
	return "spydon_api_keys"
}

// Dictionary 字典表 - 用于管理枚举值、下拉选项等配置数据
type Dictionary struct {
	BaseModel
	Code           string           `json:"code" gorm:"type:varchar(100);uniqueIndex;not null"` // 字典编码（唯一标识）
	Name           string           `json:"name" gorm:"type:varchar(255);not null"`             // 字典名称
	Module         string           `json:"module" gorm:"type:varchar(100)"`                    // 所属模块
	Description    string           `json:"description" gorm:"type:text"`                       // 描述
	IsEnabled      bool             `json:"is_enabled" gorm:"type:boolean;default:true"`        // 是否启用
	KeySameAsValue bool             `json:"key_same_as_value" gorm:"type:boolean"`              // key 与 value 是否相同
	SortOrder      int              `json:"sort_order" gorm:"type:int;default:0"`               // 排序顺序
	Items          []DictionaryItem `json:"items,omitempty" gorm:"foreignKey:DictionaryID;constraint:OnDelete:CASCADE"`
}

// DictionaryItem 字典项表 - 具体的枚举选项
type DictionaryItem struct {
	BaseModel
	DictionaryID uint64 `json:"dictionary_id" gorm:"type:bigint unsigned;index;not null"` // 关联字典 ID
	Key          string `json:"key" gorm:"type:varchar(255);not null"`                    // 选项 Key（存储值）
	Value        string `json:"value" gorm:"type:varchar(255);not null"`                  // 选项 Value（显示值）
	Description  string `json:"description" gorm:"type:text"`                             // 描述
	IsDefault    bool   `json:"is_default" gorm:"type:boolean;default:false"`             // 是否默认选中
	IsEnabled    bool   `json:"is_enabled" gorm:"type:boolean;default:true"`              // 是否启用
	SortOrder    int    `json:"sort_order" gorm:"type:int;default:0"`                     // 排序顺序
	Extra        string `json:"extra,omitempty" gorm:"type:text"`                         // 扩展字段（JSON）
}

// TableName 指定表名
func (Dictionary) TableName() string {
	return "spydon_dictionaries"
}

// TableName 指定表名
func (DictionaryItem) TableName() string {
	return "spydon_dictionary_items"
}

// EmailTemplate 邮件模板
type EmailTemplate struct {
	BaseModel
	Name      string `json:"name" gorm:"type:varchar(100);uniqueIndex;not null"` // 模板名称
	Title     string `json:"title" gorm:"type:varchar(255);not null"`            // 邮件标题模板
	Body      string `json:"body" gorm:"type:text;not null"`                     // HTML 模板内容
	Params    string `json:"params" gorm:"type:text"`                            // 参数定义 (JSON)
	IsEnabled bool   `json:"is_enabled" gorm:"type:boolean;default:true"`        // 是否启用
}

// TableName 指定表名
func (EmailTemplate) TableName() string {
	return "email_template"
}

// EmailContact 邮件联系人
type EmailContact struct {
	BaseModel
	Name    string `json:"name" gorm:"type:varchar(100);not null" query:"like"`    // 联系人/组名称
	Address string `json:"address" gorm:"type:varchar(500);not null" query:"like"` // 邮箱地址（多个用逗号分隔）
}

// TableName 指定表名
func (EmailContact) TableName() string {
	return "email_address"
}

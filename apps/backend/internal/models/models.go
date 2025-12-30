package models

import (
	"strconv"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
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
	ID        uint64         `json:"id" gorm:"primaryKey;autoIncrement"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

// Cluster 集群模型
type Cluster struct {
	BaseModel
	Name          string     `json:"name" gorm:"type:varchar(255);unique;not null"`
	ClusterID     string     `json:"cluster_id" gorm:"column:cluster_id;type:varchar(255)"`
	Description   string     `json:"description" gorm:"type:text"`
	KubeConfig    string     `json:"kube_config" gorm:"type:text"`
	PrometheusURL string     `json:"prometheus_url" gorm:"type:varchar(255)"`
	Status        string     `json:"status" gorm:"type:varchar(32);default:active"`
	LastHeartbeat *time.Time `json:"last_heartbeat"`

	// 关联关系 - 不使用数据库外键约束
	Alerts []Alert `json:"alerts,omitempty" gorm:"foreignKey:ClusterName;references:Name;constraint:OnDelete:SET NULL,OnUpdate:CASCADE;"`
}

// Alert 告警模型
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

	// 关联关系 - 不使用数据库外键约束，由应用层保证数据完整性
	Cluster *Cluster `json:"cluster,omitempty" gorm:"foreignKey:ClusterName;references:Name;constraint:OnDelete:SET NULL,OnUpdate:CASCADE;"`
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
	ClusterStatusActive   ClusterStatus = "active"
	ClusterStatusInactive ClusterStatus = "inactive"
)

// TableName 方法用于指定表名
func (Cluster) TableName() string {
	return "clusters"
}

func (Alert) TableName() string {
	return "alerts"
}

func (RCARun) TableName() string {
	return "rca_runs"
}

func (AuditLog) TableName() string {
	return "audit_logs"
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
	return "users"
}

func (RefreshToken) TableName() string {
	return "refresh_tokens"
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
	return "api_keys"
}

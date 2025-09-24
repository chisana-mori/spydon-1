package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// BaseModel 基础模型，包含通用字段
type BaseModel struct {
	ID        uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

// Cluster 集群模型
type Cluster struct {
	BaseModel
	ClusterID     string     `json:"cluster_id" gorm:"type:varchar(255);unique;not null"`
	Name          string     `json:"name" gorm:"not null"`
	Description   string     `json:"description"`
	Status        string     `json:"status" gorm:"default:active"`
	LastHeartbeat *time.Time `json:"last_heartbeat"`

	// 关联关系
	Alerts []Alert `json:"alerts,omitempty" gorm:"foreignKey:ClusterID;references:ClusterID"`
}

// Alert 告警模型
type Alert struct {
	BaseModel
	Fingerprint   string         `json:"fingerprint" gorm:"not null"`
	ClusterID     string         `json:"cluster_id" gorm:"type:varchar(255);not null"`
	Title         string         `json:"title" gorm:"not null"`
	Description   string         `json:"description"`
	Severity      string         `json:"severity" gorm:"not null"`
	Status        string         `json:"status" gorm:"default:firing"`
	Labels        datatypes.JSON `json:"labels" gorm:"type:jsonb"`
	Annotations   datatypes.JSON `json:"annotations" gorm:"type:jsonb"`
	StartsAt      *time.Time     `json:"starts_at"`
	EndsAt        *time.Time     `json:"ends_at"`
	RawPayloadKey string         `json:"raw_payload_key" gorm:"type:text"`

	// 关联关系
	Cluster *Cluster `json:"cluster,omitempty" gorm:"foreignKey:ClusterID;references:ClusterID"`
	RCARuns []RCARun `json:"rca_runs,omitempty" gorm:"foreignKey:AlertID"`
}

// RCARun RCA运行记录模型
type RCARun struct {
	BaseModel
	AlertID         uuid.UUID      `json:"alert_id" gorm:"not null"`
	Status          string         `json:"status" gorm:"not null"`
	Summary         *string        `json:"summary"`
	Suspects        datatypes.JSON `json:"suspects" gorm:"type:jsonb"`
	Recommendations datatypes.JSON `json:"recommendations" gorm:"type:jsonb"`
	Attachments     datatypes.JSON `json:"attachments" gorm:"type:jsonb"`
	StartedAt       time.Time      `json:"started_at" gorm:"default:now()"`
	CompletedAt     *time.Time     `json:"completed_at"`
	ErrorMessage    *string        `json:"error_message"`
	RawPayloadKey   string         `json:"raw_payload_key" gorm:"type:text"`

	// 关联关系
	Alert *Alert `json:"alert,omitempty" gorm:"foreignKey:AlertID"`
}

// AuditLog 审计日志模型
type AuditLog struct {
	BaseModel
	UserID       string         `json:"user_id"`
	Action       string         `json:"action" gorm:"not null"`
	ResourceType string         `json:"resource_type"`
	ResourceID   string         `json:"resource_id"`
	Details      datatypes.JSON `json:"details" gorm:"type:jsonb"`
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
)

// ClusterStatus 集群状态枚举
type ClusterStatus string

const (
	ClusterStatusActive      ClusterStatus = "active"
	ClusterStatusInactive    ClusterStatus = "inactive"
	ClusterStatusMaintenance ClusterStatus = "maintenance"
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
	Username      string     `json:"username" gorm:"uniqueIndex;not null"`
	Email         string     `json:"email" gorm:"uniqueIndex;not null"`
	Name          string     `json:"name"`
	Picture       string     `json:"picture"`
	Roles         []string   `json:"roles" gorm:"type:text[]"`
	EmailVerified bool       `json:"email_verified" gorm:"default:false"`
	Provider      string     `json:"provider" gorm:"default:local"`
	ProviderID    string     `json:"provider_id"`
	LastLoginAt   *time.Time `json:"last_login_at"`
}

// RefreshToken 刷新令牌模型
type RefreshToken struct {
	BaseModel
	UserID    uuid.UUID `json:"user_id" gorm:"not null"`
	Token     string    `json:"token" gorm:"uniqueIndex;not null"`
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

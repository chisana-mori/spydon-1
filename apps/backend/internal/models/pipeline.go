package models

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// PipelineTemplate 流水线模板 - 管理员定义的可复用流程
type PipelineTemplate struct {
	ID          uint64         `json:"id" gorm:"primaryKey;autoIncrement"`
	Name        string         `json:"name" gorm:"type:varchar(255);unique;not null"`
	Description string         `json:"description" gorm:"type:text"`
	Stages      datatypes.JSON `json:"stages" gorm:"type:json;not null"` // []StageDefinition
	CreatedBy   uint64         `json:"created_by" gorm:"type:bigint unsigned"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

// StageDefinition 阶段定义 (存储在JSON中)
type StageDefinition struct {
	ID        string          `json:"id"`   // Stage唯一标识
	Name      string          `json:"name"` // 阶段名称
	Type      StageType       `json:"type"` // 阶段类型
	Config    StageConfig     `json:"config"`
	OnFailure FailureStrategy `json:"on_failure"`
	DependsOn []string        `json:"depends_on,omitempty"` // 依赖的前置Stage ID
}

// StageType 阶段类型
type StageType string

const (
	StageTypeAWXJob     StageType = "awx_job"     // 执行AWX Job Template
	StageTypeManualGate StageType = "manual_gate" // 人工审批
	StageTypeDelay      StageType = "delay"       // 延时等待
	StageTypeCondition  StageType = "condition"   // 条件判断
	StageTypePreCheck   StageType = "pre_check"   // 预检查 (Prometheus查询)
	StageTypePostCheck  StageType = "post_check"  // 后检查
	StageTypeRollback   StageType = "rollback"    // 回滚阶段
)

// StageConfig 阶段配置 (根据Type不同使用不同字段)
type StageConfig struct {
	// AWX Job 相关
	AWXTemplateID   int                `json:"awx_template_id,omitempty"`
	AWXTemplateName string             `json:"awx_template_name,omitempty"`
	ExtraVars       map[string]string  `json:"extra_vars,omitempty"`
	DryRun          bool               `json:"dry_run,omitempty"`    // Check Mode
	Parameters      []ParameterBinding `json:"parameters,omitempty"` // 参数绑定

	// Manual Gate 相关
	ApproverRoles  []string `json:"approver_roles,omitempty"`
	TimeoutMinutes int      `json:"timeout_minutes,omitempty"`

	// Delay 相关
	DelaySeconds int `json:"delay_seconds,omitempty"`

	// Condition / Check 相关
	PromQuery     string `json:"prom_query,omitempty"`
	ExpectedValue string `json:"expected_value,omitempty"`
	Operator      string `json:"operator,omitempty"` // eq, ne, gt, lt
}

// ParameterBinding 参数绑定定义
type ParameterBinding struct {
	Name         string   `json:"name"`                    // 参数名称（对应AWX extra_vars的key）
	Label        string   `json:"label"`                   // 显示标签
	Description  string   `json:"description,omitempty"`   // 参数说明
	InputType    string   `json:"input_type"`              // 输入类型: fixed, text, select, multi_select
	DefaultValue string   `json:"default_value,omitempty"` // 默认值
	Options      []string `json:"options,omitempty"`       // 可选值列表（用于select/multi_select）
	Required     bool     `json:"required"`                // 是否必填
}

// FailureStrategy 失败策略
type FailureStrategy string

const (
	FailureStrategyAbort    FailureStrategy = "abort"    // 终止流水线
	FailureStrategyRollback FailureStrategy = "rollback" // 触发回滚
	FailureStrategyContinue FailureStrategy = "continue" // 继续执行
	FailureStrategyPause    FailureStrategy = "pause"    // 暂停等待人工介入
)

// PipelineExecution 流水线执行实例
type PipelineExecution struct {
	ID                 uint64          `json:"id" gorm:"primaryKey;autoIncrement"`
	PipelineTemplateID uint64          `json:"pipeline_template_id" gorm:"type:bigint unsigned;not null"`
	ClusterID          uint64          `json:"cluster_id" gorm:"type:bigint unsigned"`
	ClusterName        string          `json:"cluster_name" gorm:"type:varchar(255)"`
	Status             ExecutionStatus `json:"status" gorm:"type:varchar(32);not null;default:pending"`
	Parameters         datatypes.JSON  `json:"parameters" gorm:"type:json"`       // 运行时参数
	EffectiveStages    datatypes.JSON  `json:"effective_stages" gorm:"type:json"` // 实际执行的阶段定义（支持动态生成/批次）
	CurrentStageID     string          `json:"current_stage_id" gorm:"type:varchar(64)"`
	StartedAt          *time.Time      `json:"started_at"`
	CompletedAt        *time.Time      `json:"completed_at"`
	TriggeredBy        uint64          `json:"triggered_by" gorm:"type:bigint unsigned"`
	ErrorMessage       *string         `json:"error_message" gorm:"type:text"`
	CreatedAt          time.Time       `json:"created_at"`
	UpdatedAt          time.Time       `json:"updated_at"`

	// 关联
	Template  *PipelineTemplate `json:"template,omitempty" gorm:"foreignKey:PipelineTemplateID"`
	StageRuns []StageRun        `json:"stage_runs,omitempty" gorm:"foreignKey:ExecutionID"`
}

// ExecutionStatus 执行状态
type ExecutionStatus string

const (
	ExecutionStatusPending     ExecutionStatus = "pending"      // 等待开始
	ExecutionStatusRunning     ExecutionStatus = "running"      // 运行中
	ExecutionStatusPaused      ExecutionStatus = "paused"       // 已暂停 (等待审批/手动继续)
	ExecutionStatusSuccessful  ExecutionStatus = "successful"   // 成功完成
	ExecutionStatusFailed      ExecutionStatus = "failed"       // 失败
	ExecutionStatusCanceled    ExecutionStatus = "canceled"     // 已取消
	ExecutionStatusRollingBack ExecutionStatus = "rolling_back" // 回滚中
	ExecutionStatusRolledBack  ExecutionStatus = "rolled_back"  // 已回滚
)

// StageRun 阶段执行记录
type StageRun struct {
	ID            uint64         `json:"id" gorm:"primaryKey;autoIncrement"`
	ExecutionID   uint64         `json:"execution_id" gorm:"type:bigint unsigned;not null"`
	StageID       string         `json:"stage_id" gorm:"type:varchar(64);not null"`
	StageName     string         `json:"stage_name" gorm:"type:varchar(255)"`
	StageType     StageType      `json:"stage_type" gorm:"type:varchar(32)"`
	Status        StageRunStatus `json:"status" gorm:"type:varchar(32);not null;default:pending"`
	AWXJobID      *int           `json:"awx_job_id" gorm:"type:int"`
	AWXJobStatus  string         `json:"awx_job_status" gorm:"type:varchar(32)"`
	Output        datatypes.JSON `json:"output" gorm:"type:json"` // 阶段输出/结果
	ErrorMessage  *string        `json:"error_message" gorm:"type:text"`
	StartedAt     *time.Time     `json:"started_at"`
	CompletedAt   *time.Time     `json:"completed_at"`
	ApprovedBy    *uint64        `json:"approved_by" gorm:"type:bigint unsigned"`
	ApprovalNotes string         `json:"approval_notes" gorm:"type:text"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

// StageRunStatus 阶段运行状态
type StageRunStatus string

const (
	StageRunStatusPending         StageRunStatus = "pending"          // 待执行
	StageRunStatusRunning         StageRunStatus = "running"          // 执行中
	StageRunStatusWaitingApproval StageRunStatus = "waiting_approval" // 等待审批
	StageRunStatusSuccessful      StageRunStatus = "successful"       // 成功
	StageRunStatusFailed          StageRunStatus = "failed"           // 失败
	StageRunStatusSkipped         StageRunStatus = "skipped"          // 已跳过
	StageRunStatusCanceled        StageRunStatus = "canceled"         // 已取消
)

// TableName 指定表名
func (PipelineTemplate) TableName() string {
	return "spydon_pipeline_templates"
}

func (PipelineExecution) TableName() string {
	return "spydon_pipeline_executions"
}

func (StageRun) TableName() string {
	return "spydon_stage_runs"
}

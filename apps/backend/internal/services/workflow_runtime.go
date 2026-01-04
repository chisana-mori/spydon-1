package services

import (
	"context"
	"time"

	"robusta-web/backend/internal/models"
)

// JobRuntime 任务运行时接口
// 定义通用的外部任务执行能力，AWX、SSH、K8s Job 等都可以实现此接口
type JobRuntime interface {
	// Name 返回运行时名称（如 "awx", "ssh", "k8s_job"）
	Name() string

	// LaunchJob 启动任务，返回任务句柄
	LaunchJob(ctx context.Context, config JobConfig) (*JobHandle, error)

	// WaitForJob 等待任务完成（阻塞式轮询）
	WaitForJob(ctx context.Context, handle *JobHandle, pollInterval time.Duration) (*JobResult, error)

	// GetJobStatus 获取任务当前状态
	GetJobStatus(ctx context.Context, handle *JobHandle) (string, error)

	// CancelJob 取消正在运行的任务
	CancelJob(ctx context.Context, handle *JobHandle) error

	// ListTemplates 列出可用的任务模板（如 AWX Job Templates）
	// 部分 runtime 可能不支持此功能，返回空列表即可
	ListTemplates(ctx context.Context) ([]JobTemplateInfo, error)

	// GetTemplate 获取单个任务模板详情
	GetTemplate(ctx context.Context, id int) (*JobTemplateInfo, error)
}

// JobConfig 任务启动配置
type JobConfig struct {
	TemplateID   int                    `json:"template_id"`   // 模板 ID（AWX template_id 等）
	TemplateName string                 `json:"template_name"` // 模板名称（用于日志/显示）
	ExtraVars    map[string]interface{} `json:"extra_vars"`    // 额外变量
	DryRun       bool                   `json:"dry_run"`       // 检查模式（不实际执行）
	Limit        string                 `json:"limit"`         // 限制执行范围（主机列表等）
	ClusterName  string                 `json:"cluster_name"`  // 目标集群名称
}

// JobHandle 任务句柄，用于跟踪和管理已启动的任务
type JobHandle struct {
	RuntimeName string `json:"runtime_name"` // 运行时名称
	JobID       int    `json:"job_id"`       // 任务 ID（runtime 内部使用）
	ExternalID  string `json:"external_id"`  // 外部系统 ID（可选，用于跨系统追踪）
}

// JobResult 任务执行结果
type JobResult struct {
	Status  string `json:"status"`  // 状态（successful, failed, canceled 等）
	Success bool   `json:"success"` // 是否成功
	Output  string `json:"output"`  // 输出内容（日志摘要等）
}

// JobTemplateInfo 任务模板信息
type JobTemplateInfo struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	ExtraVars   string `json:"extra_vars,omitempty"` // 模板变量定义(JSON/YAML)
}

// MetricsRuntime 指标查询运行时接口
// 用于执行检查阶段的指标查询，Prometheus 只是其中一个实现
type MetricsRuntime interface {
	// Name 返回运行时名称（如 "prometheus", "influxdb"）
	Name() string

	// Query 执行指标查询，返回数值结果
	// endpoint: 查询端点 URL
	// query: 查询语句（如 PromQL）
	Query(ctx context.Context, endpoint string, query string) (float64, error)

	// QueryRaw 执行指标查询，返回原始结果（用于复杂场景）
	QueryRaw(ctx context.Context, endpoint string, query string) (interface{}, error)
}

// PipelineHookContext 上下文信息，传递给钩子函数
type PipelineHookContext struct {
	ExecutionID    uint64
	Stage          models.StageDefinition
	PreviousResult *StageResult // 上一个阶段的执行结果 (如果有)
}

// PipelineHook 流水线钩子接口
// 允许在阶段执行前后拦截并执行自定义逻辑 (如 AI Agent 审核)
type PipelineHook interface {
	// Name Hook 名称
	Name() string

	// BeforeStage 阶段执行前调用
	// 如果返回 error，将阻止阶段执行 (视作阶段失败)
	BeforeStage(ctx context.Context, hookCtx PipelineHookContext) error

	// AfterStage 阶段执行后调用
	// result 包含了阶段的执行结果，Hook 可以修改 result 或基于 result 执行后续操作
	AfterStage(ctx context.Context, hookCtx PipelineHookContext, result *StageResult) error
}

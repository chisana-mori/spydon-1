// Pipeline流水线相关类型

// 阶段类型
export type StageType =
  | 'awx_job'
  | 'manual_gate'
  | 'delay'
  | 'condition'
  | 'pre_check'
  | 'post_check'
  | 'rollback'

// 失败策略
export type FailureStrategy = 'abort' | 'rollback' | 'continue' | 'pause'

// 阶段配置
export interface StageConfig {
  awx_template_id?: number
  awx_template_name?: string
  extra_vars?: Record<string, string>
  limit?: string
  dry_run?: boolean
  approver_roles?: string[]
  timeout_minutes?: number
  delay_seconds?: number
  prom_query?: string
  expected_value?: string
  operator?: 'eq' | 'ne' | 'gt' | 'lt'
}

// 阶段定义
export interface StageDefinition {
  id: string
  name: string
  type: StageType
  config: StageConfig
  on_failure: FailureStrategy
  depends_on?: string[]
}

// 流水线模板
export interface PipelineTemplate {
  id: string
  name: string
  description?: string
  stages: StageDefinition[]
  created_by?: string
  created_at: string
  updated_at: string
}

// 执行状态
export type ExecutionStatus =
  | 'pending'
  | 'running'
  | 'paused'
  | 'successful'
  | 'failed'
  | 'canceled'
  | 'rolling_back'
  | 'rolled_back'

// 阶段运行状态
export type StageRunStatus =
  | 'pending'
  | 'running'
  | 'waiting_approval'
  | 'successful'
  | 'failed'
  | 'skipped'
  | 'canceled'

// 阶段运行记录
export interface StageRun {
  id: string
  execution_id: string
  stage_id: string
  stage_name: string
  stage_type: StageType
  status: StageRunStatus
  awx_job_id?: number
  awx_job_status?: string
  output?: Record<string, any>
  error_message?: string
  started_at?: string
  completed_at?: string
  approved_by?: string
  approval_notes?: string
  created_at: string
  updated_at: string
}

// 流水线执行实例
export interface PipelineExecution {
  id: string
  pipeline_template_id: string
  cluster_id: string
  cluster_name: string
  status: ExecutionStatus
  parameters?: Record<string, any>
  current_stage_id?: string
  started_at?: string
  completed_at?: string
  triggered_by?: string
  error_message?: string
  created_at: string
  updated_at: string
  template?: PipelineTemplate
  stage_runs?: StageRun[]
}

// 创建模板请求
export interface CreateTemplateRequest {
  name: string
  description?: string
  stages: StageDefinition[]
}

// 启动执行请求
export interface StartExecutionRequest {
  template_id: string
  cluster_id: string
  parameters?: Record<string, any>
}

// 审批请求
export interface ApprovalRequest {
  notes?: string
}

// 执行历史查询参数
export interface ExecutionHistoryParams {
  cluster_id: string
  limit?: number
}

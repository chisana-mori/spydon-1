// Pipeline流水线相关类型

// AWX Job Template
export interface AWXJobTemplate {
  id: number
  name: string
  description: string
  job_type: string
  inventory: number
  project: number
  playbook: string
  ask_variables_on_launch: boolean
  ask_limit_on_launch: boolean
  extra_vars?: string
}

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

// 参数输入类型
export type ParameterInputType = 'fixed' | 'text' | 'select' | 'multi_select'

// 参数绑定定义
export interface ParameterBinding {
  name: string           // 参数名称（对应AWX extra_vars的key）
  label: string          // 显示标签
  description?: string   // 参数说明
  input_type: ParameterInputType  // 输入类型
  default_value?: string // 默认值
  options?: string[]     // 可选值列表（用于select/multi_select）
  required: boolean      // 是否必填
}

// 阶段配置
export interface StageConfig {
  awx_template_id?: number
  awx_template_name?: string
  extra_vars?: Record<string, string>
  dry_run?: boolean
  parameters?: ParameterBinding[]  // 参数绑定
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

// 更新模板请求
export interface UpdateTemplateRequest {
  name: string
  description?: string
  stages: StageDefinition[]
}

// 启动执行请求
export interface StartExecutionRequest {
  template_id: number
  cluster_id: number
  parameters?: Record<string, any>
  target_nodes?: string[]
  batch_size?: number
  batches?: string[][]
  pause_between_batches?: boolean
  auto_start?: boolean  // 是否自动启动，默认 true；设为 false 则创建待执行任务
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

export interface PipelineNodeStatus {
  task_name: string
  task_id: string
  status: 'running' | 'success' | 'failed' | 'skipped'
  start_time: string
  end_time?: string
  host?: string
}

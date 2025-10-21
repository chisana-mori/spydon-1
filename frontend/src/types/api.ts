// API响应基础类型
export interface ApiResponse<T = any> {
  data?: T
  error?: string
  code?: string
  message?: string
}

export interface PaginationResponse<T> {
  data: T[]
  pagination: {
    page: number
    limit: number
    total: number
  }
}

// 集群相关类型
export interface Cluster {
  id: string
  cluster_id: string
  name: string
  description?: string
  status: 'active' | 'inactive' | 'maintenance'
  last_heartbeat?: string
  created_at: string
  updated_at: string
}

export interface ClusterStats {
  cluster_id: string
  name: string
  status: string
  alert_count: number
  critical_count: number
  last_heartbeat?: string
}

export interface ClusterSummary {
  total_clusters: number
  active_clusters: number
  total_alerts: number
  critical_alerts: number
  cluster_stats: ClusterStats[]
  alert_trends: AlertTrendData[]
}

export interface AlertTrendData {
  date: string
  count: number
}

// 告警相关类型
export interface Alert {
  id: string
  fingerprint: string
  cluster_id: string
  title: string
  description?: string
  severity: 'low' | 'medium' | 'high' | 'critical'
  status: 'firing' | 'resolved' | 'silenced'
  labels?: Record<string, any>
  annotations?: Record<string, any>
  starts_at?: string
  ends_at?: string
  raw_payload_key?: string
  created_at: string
  updated_at: string
  cluster?: Cluster
  rca_runs?: RCARun[]
}

export interface AlertFilters {
  cluster_id?: string
  severity?: string
  status?: string
  keyword?: string
  since?: string
}

// RCA相关类型
export interface RCARun {
  id: string
  alert_id: string
  status: 'pending' | 'running' | 'completed' | 'failed' | 'timeout'
  summary?: string
  suspects?: Record<string, any>
  recommendations?: Record<string, any>
  attachments?: Record<string, any>
  started_at: string
  completed_at?: string
  error_message?: string
  created_at: string
  alert?: Alert
}

// 审计日志类型
export interface AuditLog {
  id: string
  user_id?: string
  action: string
  resource_type?: string
  resource_id?: string
  details?: Record<string, any>
  ip_address?: string
  user_agent?: string
  created_at: string
}

// 统计信息类型
export interface AlertStats {
  by_severity: Array<{ severity: string; count: number }>
  by_status: Array<{ status: string; count: number }>
  total: number
  recent_24h: number
}

export interface RCAStats {
  by_status: Array<{ status: string; count: number }>
  total: number
  success_rate: number
  avg_duration_seconds: number
}

// 事件流类型
export interface StreamEvent {
  type: 'alert' | 'rca' | 'heartbeat' | 'cluster'
  data: any
  timestamp: string
}

// 用户认证类型
export interface User {
  id: string
  username?: string
  email: string
  name?: string
  picture?: string
  is_admin: boolean
  email_verified?: boolean
  provider?: string
  last_login_at?: string
  created_at?: string
  updated_at?: string
}

export interface AuthToken {
  access_token: string
  token_type: string
  expires_in: number
}

// 错误类型
export interface ApiError {
  error: string
  code: string
  details?: string
}

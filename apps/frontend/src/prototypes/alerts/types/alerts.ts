/**
 * 告警相关的类型定义
 */

export interface Alert {
  id: string;
  title: string;
  description?: string;
  severity: 'low' | 'medium' | 'high' | 'critical';
  status: 'active' | 'resolved' | 'acknowledged' | 'firing';
  source?: string;
  timestamp?: string;
  created_at: string;
  updated_at: string;
  labels: Record<string, string> | null;
  annotations: Record<string, string> | null;

  // 原始数据字段
  raw_payload_key?: string;

  // 其他字段
  fingerprint?: string;
  cluster_id?: string;
  generatorURL?: string;
  starts_at?: string | null;
  ends_at?: string | null;

  // 集群信息
  cluster?: {
    id: string;
    cluster_id: string;
    name: string;
    description: string;
    status: string;
    created_at: string;
    updated_at: string;
    last_heartbeat: string;
  };

  // RCA 运行记录
  rca_runs?: RCARun[];
}

// RCA 状态
export type RCAStatus = 'none' | 'pending' | 'queued' | 'running' | 'completed' | 'failed' | 'timeout';

// RCA 运行记录
export interface RCARun {
  id: string;
  alert_id: string;
  status: RCAStatus;
  summary?: string | null;
  suspects?: Record<string, any> | null;
  recommendations?: Record<string, any> | null;
  attachments?: Record<string, any> | null;
  error_message?: string | null;
  started_at: string;
  completed_at?: string | null;
  created_at: string;
  updated_at: string;
}

// RCA SSE 消息类型
export type RCAMessageType = 'status' | 'log' | 'progress' | 'error';

// RCA SSE 消息
export interface RCAMessage {
  type: RCAMessageType;
  status?: RCAStatus;
  message?: string;
  percentage?: number;
  error?: string;
  timestamp?: string;
}

// RCA 进度日志
export interface RCAProgressLog {
  timestamp: Date;
  message: string;
  type: 'info' | 'success' | 'warning' | 'error';
}

export interface AlertsResponse {
  alerts: Alert[];
  total: number;
  page: number;
  pageSize: number;
  hasMore: boolean;
}

export interface AlertsQueryParams {
  page?: number;
  pageSize?: number;
  severity?: string[];
  status?: string[];
  source?: string;
  search?: string;
  startTime?: string;
  endTime?: string;
}

// 原始数据相关类型
export interface RawPayloadData {
  key: string;
  data: any;
  contentType: string;
  lastModified: Date;
  size: number;
}

export interface RawPayloadDisplayProps {
  rawPayloadKey: string;
  alertId: string;
  alertTitle: string;
}

// 原始数据展示模式
export type RawDataViewMode = 'json' | 'yaml' | 'raw' | 'formatted';

// 原始数据状态
export interface RawDataState {
  loading: boolean;
  data: RawPayloadData | null;
  error: string | null;
  viewMode: RawDataViewMode;
}

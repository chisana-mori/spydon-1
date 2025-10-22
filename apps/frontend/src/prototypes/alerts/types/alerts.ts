/**
 * 告警相关的类型定义
 */

export interface Alert {
  id: string;
  title: string;
  description?: string;
  severity: 'low' | 'medium' | 'high' | 'critical';
  status: 'active' | 'resolved' | 'acknowledged' | 'firing'; // 添加 firing 状态
  source?: string;
  timestamp?: string;
  created_at: string; // 实际返回的时间字段
  updated_at: string;
  labels: Record<string, string> | null; // 可能为 null
  annotations: Record<string, string> | null; // 可能为 null

  // 新增的原始数据字段
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

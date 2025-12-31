/**
 * 告警 API 客户端
 */

import { Alert, AlertsResponse, AlertsQueryParams } from '@prototypes/alerts/types/alerts';

export class AlertsAPI {
  private baseUrl: string;
  private apiToken?: string;

  constructor(baseUrl?: string, apiToken?: string) {
    this.baseUrl = baseUrl || process.env.NEXT_PUBLIC_API_BASE_URL || '/api';
    this.apiToken = apiToken || process.env.NEXT_PUBLIC_API_TOKEN;
  }

  /**
   * 获取告警列表
   */
  async getAlerts(params: AlertsQueryParams = {}): Promise<AlertsResponse> {
    const searchParams = new URLSearchParams();

    // 构建查询参数
    if (params.page) searchParams.set('page', params.page.toString());
    if (params.pageSize) searchParams.set('pageSize', params.pageSize.toString());
    if (params.severity?.length) searchParams.set('severity', params.severity.join(','));
    if (params.status?.length) searchParams.set('status', params.status.join(','));
    if (params.source) searchParams.set('source', params.source);
    if (params.search) searchParams.set('search', params.search);
    if (params.startTime) searchParams.set('startTime', params.startTime);
    if (params.endTime) searchParams.set('endTime', params.endTime);

    const url = `${this.baseUrl}/v1/alerts?${searchParams.toString()}`;

    const response = await fetch(url, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
        ...(this.apiToken && { 'Authorization': `Bearer ${this.apiToken}` }),
      },
    });

    if (!response.ok) {
      throw new Error(`Failed to fetch alerts: ${response.status} ${response.statusText}`);
    }

    const data = await response.json();

    // 确保返回的数据符合 AlertsResponse 接口
    return {
      alerts: data.alerts || [],
      total: data.total || 0,
      page: data.page || 1,
      pageSize: data.pageSize || 20,
      hasMore: data.hasMore || false,
    };
  }

  /**
   * 获取单个告警详情
   */
  async getAlert(alertId: string): Promise<Alert> {
    const url = `${this.baseUrl}/v1/alerts/${alertId}`;

    const response = await fetch(url, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
        ...(this.apiToken && { 'Authorization': `Bearer ${this.apiToken}` }),
      },
    });

    if (!response.ok) {
      throw new Error(`Failed to fetch alert: ${response.status} ${response.statusText}`);
    }

    return await response.json();
  }

  /**
   * 更新告警状态
   */
  async updateAlertStatus(alertId: string, status: 'acknowledged' | 'resolved'): Promise<Alert> {
    const url = `${this.baseUrl}/v1/alerts/${alertId}/status`;

    const response = await fetch(url, {
      method: 'PATCH',
      headers: {
        'Content-Type': 'application/json',
        ...(this.apiToken && { 'Authorization': `Bearer ${this.apiToken}` }),
      },
      body: JSON.stringify({ status }),
    });

    if (!response.ok) {
      throw new Error(`Failed to update alert status: ${response.status} ${response.statusText}`);
    }

    return await response.json();
  }

  /**
   * 批量更新告警状态
   */
  async batchUpdateAlertStatus(
    alertIds: string[],
    status: 'acknowledged' | 'resolved'
  ): Promise<{ updated: number; failed: string[] }> {
    const url = `${this.baseUrl}/v1/alerts/batch/status`;

    const response = await fetch(url, {
      method: 'PATCH',
      headers: {
        'Content-Type': 'application/json',
        ...(this.apiToken && { 'Authorization': `Bearer ${this.apiToken}` }),
      },
      body: JSON.stringify({
        alertIds,
        status
      }),
    });

    if (!response.ok) {
      throw new Error(`Failed to batch update alert status: ${response.status} ${response.statusText}`);
    }

    return await response.json();
  }

  /**
   * 获取告警统计信息
   */
  async getAlertStats(): Promise<{
    total: number;
    active: number;
    resolved: number;
    acknowledged: number;
    bySeverity: Record<string, number>;
    bySource: Record<string, number>;
  }> {
    const url = `${this.baseUrl}/v1/alerts/stats`;

    const response = await fetch(url, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
        ...(this.apiToken && { 'Authorization': `Bearer ${this.apiToken}` }),
      },
    });

    if (!response.ok) {
      throw new Error(`Failed to fetch alert stats: ${response.status} ${response.statusText}`);
    }

    return await response.json();
  }
}

// 默认 API 客户端实例
export const alertsAPI = new AlertsAPI();

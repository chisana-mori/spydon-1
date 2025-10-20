import axios from 'axios'
import type { AxiosInstance, AxiosResponse } from 'axios'
import type {
  ApiResponse,
  PaginationResponse,
  Cluster,
  ClusterSummary,
  Alert,
  AlertFilters,
  RCARun,
  AuditLog,
  AlertStats,
  RCAStats,
  AlertTrendData,
  User,
} from '@/types/api'

const API_BASE = process.env.NEXT_PUBLIC_API_BASE_URL || 'http://localhost:8080/api/v1'
const DEFAULT_CAS_LOGIN_URL = 'https://localhost:8443/cas/login'
const CAS_LOGIN_PATH = process.env.NEXT_PUBLIC_CAS_LOGIN_PATH || DEFAULT_CAS_LOGIN_URL

const BACKEND_ORIGIN = (() => {
  try {
    return new URL(API_BASE).origin
  } catch (error) {
    return 'http://localhost:8080'
  }
})()

let isRedirectingToCAS = false

const resolveCasLoginUrl = () => {
  const target = CAS_LOGIN_PATH.trim() || DEFAULT_CAS_LOGIN_URL
  if (target.startsWith('http://') || target.startsWith('https://')) {
    return target
  }
  return `${BACKEND_ORIGIN}${target}`
}

// 创建axios实例
const createApiClient = (): AxiosInstance => {
  const client = axios.create({
    baseURL: API_BASE,
    timeout: 30000,
    headers: {
      'Content-Type': 'application/json',
    },
    withCredentials: true,
  })

  // 请求拦截器 - 开发阶段暂时禁用认证
  client.interceptors.request.use(
    (config) => {
      // TODO: 生产环境需要启用认证token
      // const token = localStorage.getItem('auth_token')
      // if (token) {
      //   config.headers.Authorization = `Bearer ${token}`
      // }
      return config
    },
    (error) => {
      return Promise.reject(error)
    }
  )

  // 响应拦截器 - 处理错误（开发阶段禁用认证重定向）
  client.interceptors.response.use(
    (response: AxiosResponse) => {
      return response
    },
    (error) => {
      if (
        typeof window !== 'undefined' &&
        error?.response?.status === 401 &&
        !isRedirectingToCAS
      ) {
        isRedirectingToCAS = true
        // 保存当前页面URL，用于登录后跳转
        sessionStorage.setItem('cas_return_url', window.location.href)

        const callbackUrl = encodeURIComponent(`${window.location.origin}/auth/cas/callback`)
        const casLoginUrl = resolveCasLoginUrl()
        const separator = casLoginUrl.includes('?') ? '&' : '?'
        window.location.href = `${casLoginUrl}${separator}service=${callbackUrl}`
      }
      return Promise.reject(error)
    }
  )

  return client
}

const apiClient = createApiClient()

// API类
export class RobustaAPI {
  // 集群相关API
  static async getClusters(page = 1, limit = 20, status?: string): Promise<PaginationResponse<Cluster>> {
    const params = new URLSearchParams({ page: page.toString(), limit: limit.toString() })
    if (status) params.append('status', status)
    
    const response = await apiClient.get(`/clusters?${params}`)
    return response.data
  }

  static async getCluster(clusterId: string): Promise<ApiResponse<Cluster>> {
    const response = await apiClient.get(`/clusters/${clusterId}`)
    return response.data
  }

  static async getClustersSummary(): Promise<ApiResponse<ClusterSummary>> {
    const response = await apiClient.get('/clusters/summary')
    return response.data
  }

  static async deleteCluster(clusterId: string): Promise<ApiResponse> {
    const response = await apiClient.delete(`/admin/clusters/${clusterId}`)
    return response.data
  }

  // 告警相关API
  static async getAlerts(
    page = 1,
    limit = 20,
    filters: AlertFilters = {}
  ): Promise<PaginationResponse<Alert>> {
    const params = new URLSearchParams({ page: page.toString(), limit: limit.toString() })
    
    Object.entries(filters).forEach(([key, value]) => {
      if (value) params.append(key, value)
    })
    
    const response = await apiClient.get(`/alerts?${params}`)
    return response.data
  }

  static async getAlert(alertId: string): Promise<ApiResponse<Alert>> {
    const response = await apiClient.get(`/alerts/${alertId}`)
    return response.data
  }

  static async getAlertRawPayload(alertId: string): Promise<any> {
    const response = await apiClient.get(`/alerts/${alertId}/raw-payload`)
    return response.data
  }

  static async getAlertStats(clusterId?: string): Promise<ApiResponse<AlertStats>> {
    const params = clusterId ? `?cluster_id=${clusterId}` : ''
    const response = await apiClient.get(`/alerts/stats${params}`)
    return response.data
  }

  static async getAlertTrend(days = 30): Promise<ApiResponse<AlertTrendData[]>> {
    const params = new URLSearchParams({ days: days.toString() })
    const response = await apiClient.get(`/alerts/trend?${params}`)
    return response.data
  }

  static async getProfile(): Promise<User | null> {
    const response = await apiClient.get<{ user?: User }>('/profile')
    return response.data?.user ?? null
  }

  static async logout(): Promise<void> {
    await apiClient.post('/auth/logout')
  }

  // RCA相关API
  static async getRCAByAlertId(alertId: string): Promise<ApiResponse<RCARun[]>> {
    const response = await apiClient.get(`/rca/${alertId}`)
    return response.data
  }

  static async triggerRCA(alertId: string): Promise<ApiResponse<RCARun>> {
    const response = await apiClient.post(`/rca/${alertId}/trigger`)
    return response.data
  }

  static async getRCAStats(clusterId?: string): Promise<ApiResponse<RCAStats>> {
    const params = clusterId ? `?cluster_id=${clusterId}` : ''
    const response = await apiClient.get(`/rca/stats${params}`)
    return response.data
  }

  // 审计日志API
  static async getAuditLogs(
    page = 1,
    limit = 50,
    userId?: string,
    action?: string
  ): Promise<PaginationResponse<AuditLog>> {
    const params = new URLSearchParams({ page: page.toString(), limit: limit.toString() })
    if (userId) params.append('user_id', userId)
    if (action) params.append('action', action)
    
    const response = await apiClient.get(`/admin/audit-logs?${params}`)
    return response.data
  }

  // 事件流API（开发阶段禁用认证）
  static createEventStream(onMessage: (event: MessageEvent) => void): EventSource {
    // TODO: 生产环境需要启用认证token
    // const token = localStorage.getItem('auth_token')
    const url = new URL('/events/stream', apiClient.defaults.baseURL)

    // if (token) {
    //   url.searchParams.append('token', token)
    // }

    const eventSource = new EventSource(url.toString())
    eventSource.onmessage = onMessage

    return eventSource
  }

  // 健康检查API
  static async healthCheck(): Promise<ApiResponse> {
    const response = await axios.get(`${apiClient.defaults.baseURL?.replace('/api/v1', '')}/health`)
    return response.data
  }
}

export default RobustaAPI

import axios, { AxiosInstance, AxiosError } from 'axios'
import { appConfig } from '@/config'
import type {
    ApiResponse,
    PaginationResponse,
    Cluster,
    ClusterSummary,
    Alert,
    AlertFilters,
    AlertStats,
    RCARun,
    RCAStats,
    AuditLog,
    User,
    KnowledgeArticle,
    KnowledgeManifest,
} from '@/types/api'
import type {
    PipelineTemplate,
    PipelineExecution,
    CreateTemplateRequest,
    UpdateTemplateRequest,
    StartExecutionRequest,
    AWXJobTemplate,
} from '@/types/pipeline'
import type {
    NavyDevice,
    DeviceQuery,
    NavyDeviceQueryRequest,
    NavyFilterOptions,
    QueryTemplate,
    DeviceFeatureDetails,
} from '@/types/navy'

// 获取 API Base URL
function getApiBaseUrl(): string {
    return appConfig.apiBaseUrl
}

// 创建 axios 实例
function createApiClient(): AxiosInstance {
    const client = axios.create({
        baseURL: getApiBaseUrl(),
        withCredentials: true,
        headers: {
            'Content-Type': 'application/json',
        },
    })

    // 请求拦截器
    client.interceptors.request.use((config) => {
        // 动态获取最新的 baseURL (支持运行时配置注入)
        config.baseURL = getApiBaseUrl()
        return config
    })

    // 响应拦截器
    client.interceptors.response.use(
        (response) => response,
        (error: AxiosError) => {
            if (error.response?.status === 401) {
                // 未授权，重定向到 CAS 登录页
                if (typeof window !== 'undefined') {
                    const currentPath = window.location.pathname
                    // 避免在登录相关页面循环重定向
                    if (!currentPath.includes('/auth') && !currentPath.includes('/unauthorized')) {
                        // CAS 登录路径已包含完整后端地址
                        const casLoginPath = appConfig.casLoginPath
                        // 添加 service 参数，登录后返回当前页面
                        const serviceUrl = encodeURIComponent(window.location.href)
                        window.location.href = `${casLoginPath}?service=${serviceUrl}`
                    }
                }
            }
            return Promise.reject(error)
        }
    )

    return client
}

const apiClient = createApiClient()

// API 响应处理
async function handleResponse<T>(promise: Promise<any>): Promise<T> {
    const response = await promise
    return response.data?.data ?? response.data
}

// API 类
export class RobustaAPI {
    // ============ Profile ============
    static async getProfile(): Promise<User> {
        const response = await apiClient.get('/profile')
        // 后端返回 { data: { user: {...} } }，需要提取 user 对象
        return response.data?.data?.user ?? response.data?.user ?? response.data
    }

    static async updateProfile(data: Partial<User>): Promise<User> {
        return handleResponse(apiClient.put('/profile', data))
    }

    static async logout(): Promise<void> {
        // Auth 路由在 /auth 下，不是 /api/v1
        await axios.post('/auth/logout', {}, { withCredentials: true })
    }

    // ============ Clusters ============
    static async getClustersSummary(): Promise<ClusterSummary> {
        return handleResponse(apiClient.get('/clusters/summary'))
    }

    static async getClusters(page = 1, pageSize = 20, status?: string): Promise<PaginationResponse<Cluster>> {
        const params: Record<string, any> = { page, page_size: pageSize }
        if (status) params.status = status
        const response = await apiClient.get('/clusters', { params })
        return response.data
    }

    static async getCluster(id: string): Promise<Cluster> {
        return handleResponse(apiClient.get(`/clusters/${id}`))
    }

    static async getClusterNodes(id: string): Promise<string[]> {
        return handleResponse(apiClient.get(`/clusters/${id}/nodes`))
    }

    static async createCluster(data: Partial<Cluster>): Promise<Cluster> {
        return handleResponse(apiClient.post('/admin/clusters', data))
    }

    static async updateCluster(name: string, data: Partial<Cluster>): Promise<Cluster> {
        return handleResponse(apiClient.put(`/admin/clusters/${name}`, data))
    }

    static async deleteCluster(name: string): Promise<void> {
        await apiClient.delete(`/admin/clusters/${name}`)
    }

    // ============ Alerts ============
    static async getAlerts(
        page = 1,
        pageSize = 20,
        filters?: AlertFilters
    ): Promise<PaginationResponse<Alert>> {
        const params: Record<string, any> = { page, page_size: pageSize, ...filters }
        const response = await apiClient.get('/alerts', { params })
        return response.data
    }

    static async getAlert(id: string): Promise<Alert> {
        return handleResponse(apiClient.get(`/alerts/${id}`))
    }

    static async getAlertStats(clusterName?: string): Promise<AlertStats> {
        const params = clusterName ? { cluster_name: clusterName } : {}
        return handleResponse(apiClient.get('/alerts/stats', { params }))
    }

    static async getAlertTrend(days = 30): Promise<{ date: string; count: number }[]> {
        return handleResponse(apiClient.get('/alerts/trend', { params: { days } }))
    }

    // ============ RCA ============
    static async getRCAStats(clusterName?: string): Promise<RCAStats> {
        const params = clusterName ? { cluster_name: clusterName } : {}
        return handleResponse(apiClient.get('/rca/stats', { params }))
    }

    static async getRCARuns(page = 1, pageSize = 20): Promise<PaginationResponse<RCARun>> {
        const response = await apiClient.get('/rca/runs', { params: { page, page_size: pageSize } })
        return response.data
    }

    static async triggerRCA(alertId: string): Promise<RCARun> {
        return handleResponse(apiClient.post(`/rca/${alertId}/trigger`))
    }

    // ============ Users ============
    static async getUsers(page = 1, pageSize = 20, keyword?: string): Promise<PaginationResponse<User>> {
        const params: Record<string, any> = { page, page_size: pageSize }
        if (keyword) params.keyword = keyword
        const response = await apiClient.get('/admin/users', { params })
        return response.data
    }

    static async setUserAdmin(userId: string, isAdmin: boolean): Promise<void> {
        await apiClient.put(`/admin/users/${userId}/admin`, { is_admin: isAdmin })
    }

    static async deleteUser(userId: string): Promise<void> {
        await apiClient.delete(`/admin/users/${userId}`)
    }

    // ============ Audit Logs ============
    static async getAuditLogs(page = 1, pageSize = 20): Promise<PaginationResponse<AuditLog>> {
        const response = await apiClient.get('/admin/audit-logs', { params: { page, page_size: pageSize } })
        return response.data
    }

    // ============ Knowledge ============
    static async listKnowledge(params: {
        page?: number
        page_size?: number
        status?: string
        keyword?: string
        rule_name?: string
    }): Promise<PaginationResponse<KnowledgeArticle>> {
        const response = await apiClient.get('/knowledge', { params })
        return response.data
    }

    static async getKnowledgeById(id: string, includeContent = false): Promise<KnowledgeManifest> {
        return handleResponse(apiClient.get(`/knowledge/${id}`, { params: { include_content: includeContent } }))
    }

    static async queryKnowledgeByRule(ruleName: string, version?: number): Promise<KnowledgeArticle[]> {
        const params: Record<string, any> = { rule_name: ruleName }
        if (version) params.version = version
        return handleResponse(apiClient.get('/knowledge', { params }))
    }

    static async createKnowledge(data: Partial<KnowledgeArticle>): Promise<KnowledgeArticle> {
        return handleResponse(apiClient.post('/knowledge', data))
    }

    static async updateKnowledge(id: string, data: Partial<KnowledgeManifest>): Promise<KnowledgeArticle> {
        return handleResponse(apiClient.put(`/knowledge/${id}`, data))
    }

    static async publishKnowledge(id: string, notes?: string): Promise<void> {
        await apiClient.post(`/knowledge/${id}/publish`, { notes })
    }

    static async deleteKnowledge(id: string): Promise<void> {
        await apiClient.delete(`/knowledge/${id}`)
    }

    // ============ Pipelines ============
    static async listPipelineTemplates(
        page = 1,
        pageSize = 20,
        keyword?: string
    ): Promise<PaginationResponse<PipelineTemplate>> {
        const params: Record<string, any> = { page, page_size: pageSize }
        if (keyword) params.keyword = keyword
        const response = await apiClient.get('/pipelines/templates', { params })
        return response.data
    }

    static async getPipelineTemplate(id: string): Promise<PipelineTemplate> {
        return handleResponse(apiClient.get(`/pipelines/templates/${id}`))
    }

    static async createPipelineTemplate(data: CreateTemplateRequest): Promise<PipelineTemplate> {
        return handleResponse(apiClient.post('/pipelines/templates', data))
    }

    static async updatePipelineTemplate(id: string, data: UpdateTemplateRequest): Promise<PipelineTemplate> {
        return handleResponse(apiClient.put(`/pipelines/templates/${id}`, data))
    }

    static async deletePipelineTemplate(id: string): Promise<void> {
        await apiClient.delete(`/pipelines/templates/${id}`)
    }

    static async getActiveExecutions(limit = 50): Promise<PipelineExecution[]> {
        return handleResponse(apiClient.get('/pipelines/executions/active', { params: { limit } }))
    }

    static async getExecutionHistory(params?: { cluster_id?: string; limit?: number }): Promise<PipelineExecution[]> {
        return handleResponse(apiClient.get('/pipelines/executions', { params }))
    }

    static async getExecution(id: string): Promise<PipelineExecution> {
        return handleResponse(apiClient.get(`/pipelines/executions/${id}`))
    }

    static async startExecution(data: StartExecutionRequest): Promise<PipelineExecution> {
        return handleResponse(apiClient.post('/pipelines/executions', data))
    }

    static async pauseExecution(id: string): Promise<void> {
        await apiClient.post(`/pipelines/executions/${id}/pause`)
    }

    static async resumeExecution(id: string, notes?: string): Promise<void> {
        await apiClient.post(`/pipelines/executions/${id}/resume`, { notes })
    }

    static async cancelExecution(id: string): Promise<void> {
        await apiClient.post(`/pipelines/executions/${id}/cancel`)
    }

    static async rollbackExecution(id: string): Promise<void> {
        await apiClient.post(`/pipelines/executions/${id}/rollback`)
    }

    static async runPendingExecution(id: string): Promise<void> {
        await apiClient.post(`/pipelines/executions/${id}/run`)
    }

    // AWX Templates
    static async listAWXTemplates(): Promise<AWXJobTemplate[]> {
        return handleResponse(apiClient.get('/pipelines/awx/templates'))
    }

    static async getAWXTemplate(id: number): Promise<AWXJobTemplate> {
        return handleResponse(apiClient.get(`/pipelines/awx/templates/${id}`))
    }

    // ============ Navy Devices ============
    static async listNavyDevices(query: DeviceQuery): Promise<PaginationResponse<NavyDevice>> {
        const response = await apiClient.get('/navy/devices', { params: query })
        return response.data
    }

    static async queryNavyDevices(req: NavyDeviceQueryRequest): Promise<PaginationResponse<NavyDevice>> {
        const response = await apiClient.post('/navy/devices/query', req)
        return response.data
    }

    static async getNavyFilterOptions(): Promise<NavyFilterOptions> {
        return handleResponse(apiClient.get('/navy/devices/filter-options'))
    }

    static async getLabelValues(labelKey: string): Promise<string[]> {
        return handleResponse(apiClient.get('/navy/devices/label-values', { params: { key: labelKey } }))
    }

    static async getTaintValues(taintKey: string): Promise<{ value: string; effect: string }[]> {
        return handleResponse(apiClient.get('/navy/devices/taint-values', { params: { key: taintKey } }))
    }

    static async getDeviceFieldValues(field: string): Promise<string[]> {
        return handleResponse(apiClient.get('/navy/devices/device-field-values', { params: { field } }))
    }

    static async getNavyDeviceFeatures(ciCode: string): Promise<DeviceFeatureDetails> {
        return handleResponse(apiClient.get('/navy/devices/feature-details', { params: { ci_code: ciCode } }))
    }

    static async updateNavyDeviceRole(id: number, role: string): Promise<void> {
        await apiClient.patch(`/navy/devices/${id}/role`, { role })
    }

    static async updateNavyDeviceGroup(id: number, group: string): Promise<void> {
        await apiClient.patch(`/navy/devices/${id}/group`, { group })
    }

    static async listNavyTemplates(page = 1, pageSize = 10): Promise<PaginationResponse<QueryTemplate>> {
        const response = await apiClient.get('/navy/templates', { params: { page, size: pageSize } })
        return response.data
    }

    static async getNavyTemplate(id: number): Promise<QueryTemplate> {
        return handleResponse(apiClient.get(`/navy/templates/${id}`))
    }

    static async saveNavyTemplate(template: QueryTemplate): Promise<void> {
        await apiClient.post('/navy/templates', template)
    }

    static async deleteNavyTemplate(id: number): Promise<void> {
        await apiClient.delete(`/navy/templates/${id}`)
    }

    static getNavyExportUrl(): string {
        return `${getApiBaseUrl()}/navy/devices/export`
    }
}


// 默认导出
export default RobustaAPI

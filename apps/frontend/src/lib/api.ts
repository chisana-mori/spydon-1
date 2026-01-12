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
    NodeLabelTaintResponse,
} from '@/types/navy'
import type { F5Info, F5InfoQuery, F5InfoListResponse, F5InfoUpdateDTO } from '@/types/f5'
import type {
    EmailTemplate,
    EmailContact,
    CreateEmailTemplateRequest,
    UpdateEmailTemplateRequest,
    CreateEmailContactRequest,
    UpdateEmailContactRequest,
    PreviewEmailRequest,
    PreviewEmailResponse,
    SendEmailRequest,
    AffectedResource,
    MailGenReq,
} from '@/types/email'
import type { OverviewResponse, ClusterDetailResponse } from '@/types/calico'

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
export const api = apiClient


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

    static async syncClusterConfig(): Promise<{ updated_count: number }> {
        return handleResponse(apiClient.post('/admin/clusters/sync-config'))
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

    static async runPendingExecution(id: string): Promise<{ message: string }> {
        const res = await fetch(`${getApiBaseUrl()}/pipelines/executions/${id}/run`, {
            method: 'POST',
        })
        if (!res.ok) throw new Error('Failed to run pending execution')
        return res.json()
    }

    static async cloneExecution(id: string): Promise<{ data: PipelineExecution }> {
        const res = await fetch(`${getApiBaseUrl()}/pipelines/executions/${id}/clone`, {
            method: 'POST',
        })
        if (!res.ok) throw new Error('Failed to clone execution')
        return res.json()
    }

    static getLogStreamUrl(id: string): string {
        return `${getApiBaseUrl()}/pipelines/executions/${id}/progress/stream`
    }

    static getProgressStreamUrl(id: string): string {
        return `${getApiBaseUrl()}/pipelines/executions/${id}/progress/stream`
    }

    static async generateSOPFlow(id: string): Promise<any> {
        return handleResponse(apiClient.get(`/pipelines/executions/${id}/sop`))
    }

    // AWX Templates
    static async listAWXTemplates(): Promise<AWXJobTemplate[]> {
        return handleResponse(apiClient.get('/pipelines/awx/templates'))
    }

    static async getAWXTemplate(id: number): Promise<AWXJobTemplate> {
        return handleResponse(apiClient.get(`/pipelines/awx/templates/${id}`))
    }

    // AWX Inventory Variables
    static async getInventoryVariables(clusterName: string): Promise<string> {
        const response = await apiClient.get(`/pipelines/awx/inventories/${clusterName}/variables`)
        return response.data?.data?.variables ?? ''
    }

    static async updateInventoryVariables(clusterName: string, variables: string): Promise<void> {
        await apiClient.put(`/pipelines/awx/inventories/${clusterName}/variables`, { variables })
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

    static async getBatchDeviceFeatures(ciCodes: string[]): Promise<DeviceFeatureDetails> {
        return handleResponse(apiClient.post('/navy/devices/features', { ci_codes: ciCodes }))
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

    // ============ Device Bulk Operations ============
    static async cordonNodes(ciCodes: string[]): Promise<BatchOperationResult> {
        return handleResponse(apiClient.post('/navy/device-ops/cordon', { ci_codes: ciCodes }))
    }

    static async uncordonNodes(ciCodes: string[]): Promise<BatchOperationResult> {
        return handleResponse(apiClient.post('/navy/device-ops/uncordon', { ci_codes: ciCodes }))
    }

    static async drainNodes(
        ciCodes: string[],
        options?: {
            force?: boolean
            ignore_daemonsets?: boolean
            delete_local_data?: boolean
            timeout?: number
        }
    ): Promise<BatchOperationResult> {
        return handleResponse(
            apiClient.post('/navy/device-ops/drain', {
                ci_codes: ciCodes,
                ...options,
            })
        )
    }

    static async taintNodes(
        ciCodes: string[],
        key: string,
        value: string,
        effect: 'NoSchedule' | 'PreferNoSchedule' | 'NoExecute',
        action: 'add' | 'remove'
    ): Promise<BatchOperationResult> {
        return handleResponse(
            apiClient.post('/navy/device-ops/taint', {
                ci_codes: ciCodes,
                key,
                value,
                effect,
                action,
            })
        )
    }

    static async labelNodes(
        ciCodes: string[],
        labels: Record<string, string>,
        action: 'add' | 'remove'
    ): Promise<BatchOperationResult> {
        return handleResponse(
            apiClient.post('/navy/device-ops/label', {
                ci_codes: ciCodes,
                labels,
                action,
            })
        )
    }

    static async shutdownNodes(ciCodes: string[]): Promise<{ job_id: number; message: string }> {
        return handleResponse(apiClient.post('/navy/device-ops/shutdown', { ci_codes: ciCodes }))
    }

    static async rebootNodes(ciCodes: string[]): Promise<{ job_id: number; message: string }> {
        return handleResponse(apiClient.post('/navy/device-ops/reboot', { ci_codes: ciCodes }))
    }

    // ============ F5 Load Balancer Management ============
    static async listF5Infos(query: F5InfoQuery): Promise<F5InfoListResponse> {
        const response = await apiClient.get('/navy/f5', { params: query })
        return response.data
    }

    static async getF5Info(id: number): Promise<F5Info> {
        return handleResponse(apiClient.get(`/navy/f5/${id}`))
    }

    static async updateF5Info(id: number, data: Partial<F5InfoUpdateDTO>): Promise<void> {
        await apiClient.put(`/navy/f5/${id}`, data)
    }

    static async deleteF5Info(id: number): Promise<void> {
        await apiClient.delete(`/navy/f5/${id}`)
    }

    // ============ K8s Node Real-Time Management ============
    // 这些 API 直接从 K8s API 获取实时数据，用于 Taint/Label 管理
    static async getNodeLabelsAndTaints(clusterName: string, ciCode: string): Promise<NodeLabelTaintResponse> {
        return handleResponse(
            apiClient.get('/navy/k8s-nodes/labels-taints', {
                params: { cluster: clusterName, ciCode },
            })
        )
    }

    static async listClusterNodes(clusterName: string): Promise<NodeLabelTaintResponse[]> {
        return handleResponse(apiClient.get('/navy/k8s-nodes', { params: { cluster: clusterName } }))
    }

    // ============ Email Templates ============
    static async listEmailTemplates(page = 1, pageSize = 20, keyword?: string): Promise<PaginationResponse<EmailTemplate>> {
        const params: Record<string, any> = { page, size: pageSize }
        if (keyword) params.keyword = keyword
        const response = await apiClient.get('/email/templates', { params })
        return response.data
    }

    static async getEmailTemplate(id: number): Promise<EmailTemplate> {
        return handleResponse(apiClient.get(`/email/templates/${id}`))
    }

    static async createEmailTemplate(data: CreateEmailTemplateRequest): Promise<EmailTemplate> {
        return handleResponse(apiClient.post('/email/templates', data))
    }

    static async updateEmailTemplate(id: number, data: UpdateEmailTemplateRequest): Promise<EmailTemplate> {
        return handleResponse(apiClient.put(`/email/templates/${id}`, data))
    }

    static async deleteEmailTemplate(id: number): Promise<void> {
        await apiClient.delete(`/email/templates/${id}`)
    }

    // ============ Email Contacts ============
    static async listEmailContacts(page = 1, pageSize = 20, keyword?: string): Promise<PaginationResponse<EmailContact>> {
        const params: Record<string, any> = { page, size: pageSize }
        if (keyword) params.keyword = keyword
        const response = await apiClient.get('/email/contacts', { params })
        return response.data
    }

    static async getEmailContact(id: number): Promise<EmailContact> {
        return handleResponse(apiClient.get(`/email/contacts/${id}`))
    }

    static async createEmailContact(data: CreateEmailContactRequest): Promise<EmailContact> {
        return handleResponse(apiClient.post('/email/contacts', data))
    }

    static async updateEmailContact(id: number, data: UpdateEmailContactRequest): Promise<EmailContact> {
        return handleResponse(apiClient.put(`/email/contacts/${id}`, data))
    }

    static async deleteEmailContact(id: number): Promise<void> {
        await apiClient.delete(`/email/contacts/${id}`)
    }

    // ============ Email Sending ============
    static async previewEmail(data: MailGenReq): Promise<PreviewEmailResponse> {
        return handleResponse(apiClient.post('/email/preview', data))
    }

    static async sendEmail(data: SendEmailRequest): Promise<void> {
        await apiClient.post('/email/send', data)
    }

    static async getAffectedResources(clusterName: string, nodes?: string[]): Promise<AffectedResource[]> {
        const params: Record<string, any> = { cluster: clusterName }
        if (nodes && nodes.length > 0) params.nodes = nodes.join(',')
        return handleResponse(apiClient.get('/email/affected-resources', { params }))
    }

    // ============ Calico Network Observability ============
    static async getCalicoOverview(): Promise<OverviewResponse> {
        return handleResponse(apiClient.get('/calico/overview'))
    }

    static async getCalicoClusterDetail(clusterName: string): Promise<ClusterDetailResponse> {
        return handleResponse(apiClient.get(`/calico/clusters/${clusterName}`))
    }

    static async syncIPPoolsToWayne(clusterName: string): Promise<{ created: number; updated: number; deleted: number }> {
        return handleResponse(apiClient.post(`/calico/clusters/${clusterName}/sync-wayne`))
    }

}

// Batch operation result type
export interface BatchOperationResult {
    total: number
    succeeded: number
    failed: number
    results: Array<{
        ci_code: string
        success: boolean
        message?: string
        error?: string
        drain_id?: string
    }>
}


// 默认导出
export default RobustaAPI

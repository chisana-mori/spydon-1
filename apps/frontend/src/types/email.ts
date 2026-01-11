// Email Notification Types

export interface ParamDefinition {
    name: string
    title: string
    type: 'input' | 'select' | 'datetime' | 'resource'
    value?: string
    required?: boolean
    dictCode?: string        // type=select 时关联的字典编码
    resourceType?: string    // type=resource 时的资源类型 (nodes/pods/deployments)
    placeholder?: string
    _id?: string
}

export interface TemplateConfig {
    tables: string[]
    definitions: ParamDefinition[]
}

export interface EmailTemplate {
    id: number
    name: string
    title: string
    body: string
    params: TemplateConfig
    is_enabled: boolean
    created_at: string
    updated_at: string
}

export interface EmailContact {
    id: number
    name: string
    address: string
    created_at: string
    updated_at: string
}

export interface CreateEmailTemplateRequest {
    name: string
    title: string
    body: string
    params: TemplateConfig
    is_enabled?: boolean
}

export interface UpdateEmailTemplateRequest {
    name?: string
    title?: string
    body?: string
    params?: TemplateConfig
    is_enabled?: boolean
}

export interface CreateEmailContactRequest {
    name: string
    address: string
}

export interface UpdateEmailContactRequest {
    name?: string
    address?: string
}

export interface PreviewEmailRequest {
    template_id: number
    cluster_name?: string
    nodes?: string[]
    params?: Record<string, unknown>
}

export interface PreviewEmailResponse {
    subject: string
    html_body: string
    affected_resources: AffectedResource[]
    attachment_name?: string  // Excel 附件文件名
}

export interface AffectedResource {
    type: string        // node, pod, deployment
    name: string
    namespace?: string
    status?: string
    ip?: string
    app?: string
}

export interface SendEmailRequest {
    template_id: number
    cluster_name?: string
    nodes?: string[]
    params?: Record<string, unknown>
    recipients: string[]
}

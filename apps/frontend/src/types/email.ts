// Email Notification Types

export interface ParamOption {
    label: string
    value: string
}

export interface ParamDefinition {
    name: string
    title: string
    type: 'string' | 'select' | 'date' | 'datetime' | 'component'
    value?: string           // 默认值（用于所有类型）
    required?: boolean
    options?: ParamOption[]  // type=select 时的手动选项
    isValueSeparated?: boolean // 是否键值分离 (全局控制选项)
    resourceType?: string    // type=component 时的资源类型 (nodes/pods/deployments)
    placeholder?: string
    _id?: string
    defaultTime?: string     // @deprecated: 使用 value 代替
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

export interface MailGenReq {
    emailTemplateId: number
    addressId: number
    additional: Record<string, any>
}

export interface PreviewEmailRequest {
    // Legacy preview params, kept for backward compatibility if needed,
    // but the new flow uses MailGenReq structure for preview/build
    template_id: number
    cluster_name?: string
    nodes?: string[]
    params?: Record<string, unknown>
}

// Response from BuildEmail (which returns SendEmailReq structure)
export interface BuildEmailResponse {
    templateId: number
    subject: string
    content: string
    attachFiles: AttachFile[]
    addresses: string[]
    appid: string[]
}

export interface AttachFile {
    Name: string
    Content: string // base64
}

export interface PreviewEmailResponse extends BuildEmailResponse {
    // Mapping backend response to frontend expectations if we need adapter layer
    // But currently backend BuildEmail returns SendEmailReq struct directly
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
    subject?: string
    body?: string
}

import { PaginationResponse } from './api'

export interface NavyDevice {
    id: number
    ci_code: string
    ip: string
    arch_type: string
    idc: string
    room: string
    cabinet: string
    cabinet_no: string
    infra_type: string
    is_localization: boolean
    net_zone: string
    group: string
    appid: string
    app_name: string
    os_create_time: string
    cpu: number
    memory: number
    model: string
    kvm_ip: string
    os: string
    company: string
    os_name: string
    os_issue: string
    os_kernel: string
    status: string
    role: string
    k8s_status: string
    cluster: string
    cluster_id: number
    acceptance_time: string
    disk_count: number
    disk_detail: string
    network_speed: string
    is_special: boolean
    feature_count: number
    created_at: string
    updated_at: string
    // Virtual Row Properties
    isVirtual?: boolean
    isExactMatch?: boolean
    originalKeyword?: string
    isMissing?: boolean
}

export interface DeviceQuery {
    page?: number
    size?: number
    keyword?: string
    onlySpecial?: boolean
}

// 筛选类型
export type FilterType = 'device' | 'nodeLabel' | 'taint'

// 条件类型 (12种)
export type ConditionType =
    | 'equal'
    | 'notEqual'
    | 'contains'
    | 'notContains'
    | 'exists'
    | 'notExists'
    | 'in'
    | 'notIn'
    | 'greaterThan'
    | 'lessThan'
    | 'isEmpty'
    | 'isNotEmpty'

// 逻辑运算符
export type LogicalOperator = 'and' | 'or'

export interface FilterBlock {
    id: string
    type: FilterType
    conditionType: ConditionType
    key: string
    value: string | string[]
    operator: LogicalOperator
    isActive: boolean
    label?: string // 用于显示
}

export interface FilterGroup {
    id: string
    blocks: FilterBlock[]
    operator: LogicalOperator
}

export interface NavyDeviceQueryRequest {
    groups: FilterGroup[]
    page?: number
    size?: number
}

// 设备字段定义
export interface DeviceFieldDefinition {
    id: string
    label: string
    dataType: 'string' | 'number' | 'boolean' | 'date'
}

export interface FilterOption {
    id: string
    label: string
    value: string
}

export interface DeviceFieldValues {
    field: string
    values: FilterOption[]
}

// 筛选选项响应
export interface NavyFilterOptions {
    deviceFields: DeviceFieldDefinition[]
    deviceFieldValues: DeviceFieldValues[]
    labelKeys: string[]
    taintKeys: string[]
}

export interface QueryTemplate {
    id?: number
    name: string
    description?: string
    groups: FilterGroup[]
    created_at?: string
    updated_at?: string
}

export interface DeviceFeatureDetails {
    labels: { key: string; value: string; nodes: string[] }[]
    taints: { key: string; value: string; effect: string; nodes: string[] }[]
}

// K8s 节点实时标签/污点响应 (使用新的 /k8s-nodes API)
export interface NodeLabelTaintResponse {
    nodeName: string
    clusterName: string
    clusterId: number
    labels: { key: string; value: string }[]
    taints: { key: string; value: string; effect: string; timeAdded?: string }[]
    updatedAt: string
    conditions: string[]
}

// 污点值
export interface TaintValue {
    key: string
    value: string
    effect: string
}

// 条件类型配置
export const CONDITION_TYPES: Record<ConditionType, { label: string; requiresValue: boolean }> = {
    equal: { label: '等于', requiresValue: true },
    notEqual: { label: '不等于', requiresValue: true },
    contains: { label: '包含', requiresValue: true },
    notContains: { label: '不包含', requiresValue: true },
    exists: { label: '存在', requiresValue: false },
    notExists: { label: '不存在', requiresValue: false },
    in: { label: '在列表中', requiresValue: true },
    notIn: { label: '不在列表中', requiresValue: true },
    greaterThan: { label: '大于', requiresValue: true },
    lessThan: { label: '小于', requiresValue: true },
    isEmpty: { label: '为空', requiresValue: false },
    isNotEmpty: { label: '不为空', requiresValue: false },
}

// 根据筛选类型获取可用的条件类型
export const getConditionTypesForFilterType = (filterType: FilterType): ConditionType[] => {
    switch (filterType) {
        case 'device':
            return ['equal', 'notEqual', 'contains', 'notContains', 'in', 'notIn', 'greaterThan', 'lessThan', 'isEmpty', 'isNotEmpty']
        case 'nodeLabel':
            return ['equal', 'notEqual', 'contains', 'exists', 'notExists', 'in']
        case 'taint':
            return ['equal', 'exists', 'notExists', 'in']
        default:
            return ['equal', 'contains', 'exists']
    }
}

// 生成唯一ID
export const generateId = (): string => {
    return `${Date.now()}-${Math.random().toString(36).substr(2, 9)}`
}

// 创建默认筛选块
export const createDefaultFilterBlock = (type: FilterType = 'device'): FilterBlock => ({
    id: generateId(),
    type,
    conditionType: 'equal',
    key: '',
    value: '',
    operator: 'and',
    isActive: true,
})

// 创建默认筛选组
export const createDefaultFilterGroup = (): FilterGroup => ({
    id: generateId(),
    blocks: [createDefaultFilterBlock()],
    operator: 'and',
})

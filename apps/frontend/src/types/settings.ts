/**
 * 系统设置相关类型定义
 */

// 系统设置项
export interface SystemSetting {
    id: string;
    key: string;
    value: any;
    description: string;
    created_at: string;
    updated_at: string;
}

// Auto-RCA 配置
export interface AutoRCAConfig {
    enabled: boolean;
    rate_limit: number;  // 周期内允许的请求数
    period: number;      // 周期（秒）
    allowed_severities: string[]; // 允许自动分析的告警级别
}

// 系统设置键
export const SETTING_KEYS = {
    AUTO_RCA: 'auto_rca',
} as const;

// 更新设置请求
export interface UpdateSettingRequest {
    value: any;
    description?: string;
}

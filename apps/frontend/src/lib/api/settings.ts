/**
 * 系统设置 API 客户端
 */

import { appConfig } from '@/config';
import { SystemSetting, UpdateSettingRequest } from '@/types/settings';

const basePath = appConfig.basePath || '';

/**
 * 获取所有系统设置
 */
export async function listSettings(): Promise<SystemSetting[]> {
    const response = await fetch(`${basePath}/api/v1/admin/settings`, {
        method: 'GET',
        credentials: 'include',
        headers: {
            'Content-Type': 'application/json',
        },
    });

    if (!response.ok) {
        throw new Error(`获取设置列表失败: ${response.statusText}`);
    }

    const data = await response.json();
    return data.data || [];
}

/**
 * 更新指定设置
 */
export async function updateSetting(
    key: string,
    request: UpdateSettingRequest
): Promise<SystemSetting> {
    const response = await fetch(`${basePath}/api/v1/admin/settings/${key}`, {
        method: 'PUT',
        credentials: 'include',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify(request),
    });

    if (!response.ok) {
        throw new Error(`更新设置失败: ${response.statusText}`);
    }

    const data = await response.json();
    return data.data;
}

/**
 * 获取指定设置
 */
export async function getSetting(key: string): Promise<SystemSetting | null> {
    const settings = await listSettings();
    return settings.find(s => s.key === key) || null;
}

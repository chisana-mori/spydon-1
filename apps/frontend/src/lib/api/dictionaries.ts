
import { appConfig } from '@/config';
import { Dictionary, DictionaryItem, DictionaryListParams, DictionaryListResponse } from '@/types/dictionary';

const basePath = appConfig.basePath || '';

/**
 * 获取字典列表
 */
export async function listDictionaries(params: DictionaryListParams = {}): Promise<DictionaryListResponse> {
    const query = new URLSearchParams();
    if (params.page) query.append('page', params.page.toString());
    if (params.page_size) query.append('page_size', params.page_size.toString());
    if (params.module) query.append('module', params.module);
    if (params.keyword) query.append('keyword', params.keyword);
    if (params.is_enabled !== undefined) query.append('is_enabled', params.is_enabled.toString());

    // 注意：后端 API 路径为 /api/v1/shared/dictionaries
    const response = await fetch(`${basePath}/api/v1/shared/dictionaries?${query.toString()}`, {
        method: 'GET',
        credentials: 'include',
        headers: {
            'Content-Type': 'application/json',
        },
    });

    if (!response.ok) {
        throw new Error(`获取字典列表失败: ${response.statusText}`);
    }

    const data = await response.json();
    return data.data; // 假设后端返回结构 { data: { items: [], total: ... } }，需确认。
    // 根据 DTO，Response 结构是 ListDictionariesResponse (包含 items, total 等)
    // 如果后端统一封装了 { code: 0, msg: "ok", data: ... }，则这里返回 data.data
}

/**
 * 获取单个字典详情
 */
export async function getDictionary(id: number): Promise<Dictionary> {
    const response = await fetch(`${basePath}/api/v1/shared/dictionaries/${id}`, {
        method: 'GET',
        credentials: 'include',
        headers: {
            'Content-Type': 'application/json',
        },
    });

    if (!response.ok) {
        throw new Error(`获取字典详情失败: ${response.statusText}`);
    }

    const data = await response.json();
    return data.data;
}

/**
 * 创建字典
 */
export async function createDictionary(dictionary: Partial<Dictionary>): Promise<Dictionary> {
    const response = await fetch(`${basePath}/api/v1/shared/dictionaries`, {
        method: 'POST',
        credentials: 'include',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify(dictionary),
    });

    if (!response.ok) {
        throw new Error(`创建字典失败: ${response.statusText}`);
    }

    const data = await response.json();
    return data.data;
}

/**
 * 更新字典
 */
export async function updateDictionary(id: number, dictionary: Partial<Dictionary>): Promise<Dictionary> {
    const response = await fetch(`${basePath}/api/v1/shared/dictionaries/${id}`, {
        method: 'PUT',
        credentials: 'include',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify(dictionary),
    });

    if (!response.ok) {
        throw new Error(`更新字典失败: ${response.statusText}`);
    }

    const data = await response.json();
    return data.data;
}

/**
 * 删除字典
 */
export async function deleteDictionary(id: number): Promise<void> {
    const response = await fetch(`${basePath}/api/v1/shared/dictionaries/${id}`, {
        method: 'DELETE',
        credentials: 'include',
        headers: {
            'Content-Type': 'application/json',
        },
    });

    if (!response.ok) {
        throw new Error(`删除字典失败: ${response.statusText}`);
    }
}

/**
 * 批量更新字典项
 */
export async function batchUpdateDictionaryItems(dictionaryId: number, items: Partial<DictionaryItem>[]): Promise<void> {
    // 预处理：将 id 为 0 的项的 id 设为 undefined，这样 JSON 序列化时不会包含 id 字段
    // 这能让后端（即使是旧代码）识别为新项（ID 为 nil）
    const processedItems = items.map(item => {
        if (item.id === 0) {
            const { id, ...rest } = item;
            return rest;
        }
        return item;
    });

    const response = await fetch(`${basePath}/api/v1/shared/dictionaries/${dictionaryId}/items`, {
        method: 'PUT',
        credentials: 'include',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({ items: processedItems }),
    });

    if (!response.ok) {
        throw new Error(`批量更新字典项失败: ${response.statusText}`);
    }
}

/**
 * 根据字典编码获取字典项列表
 */
export async function getDictionaryItemsByCode(code: string): Promise<DictionaryItem[]> {
    const response = await fetch(`${basePath}/api/v1/shared/dict/${code}/items`, {
        method: 'GET',
        credentials: 'include',
        headers: {
            'Content-Type': 'application/json',
        },
    });

    if (!response.ok) {
        throw new Error(`获取字典项失败: ${response.statusText}`);
    }

    const data = await response.json();
    return data.data || [];
}

/**
 * 解析字典值：将逗号分隔的 keys 转换为用分隔符分隔的 values
 * @param keys - 逗号分隔的 key 字符串，如 "APLUS,BPLUS"
 * @param items - 字典项列表
 * @param separator - 分隔符，默认为 " | "
 * @returns 解析后的值字符串，如 "A+ | B+"
 */
export function parseDictionaryValues(
    keys: string | undefined,
    items: DictionaryItem[],
    separator: string = ' | '
): string {
    if (!keys) return '-'

    const keyArray = keys.split(',').filter(Boolean)
    if (keyArray.length === 0) return '-'

    const values = keyArray
        .map(key => {
            const item = items.find(item => item.key === key)
            return item?.value || key
        })
        .filter(Boolean)

    return values.length > 0 ? values.join(separator) : '-'
}

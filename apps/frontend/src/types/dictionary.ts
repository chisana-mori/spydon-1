
export interface DictionaryItem {
    id: number;
    dictionary_id?: number;
    key: string;
    value: string;
    description?: string;
    is_default: boolean;
    is_enabled: boolean;
    sort_order: number;
    extra?: string; // JSON string
    created_at?: string;
    updated_at?: string;
}

export interface Dictionary {
    id: number;
    code: string;
    name: string;
    module: string;
    description?: string;
    is_enabled: boolean;
    key_same_as_value: boolean;
    sort_order: number;
    items?: DictionaryItem[];
    created_at?: string;
    updated_at?: string;
}

export interface DictionaryListParams {
    page?: number;
    page_size?: number;
    module?: string;
    keyword?: string;
    is_enabled?: boolean;
}

export interface DictionaryListResponse {
    items: Dictionary[];
    total: number;
    page: number;
    page_size: number;
    total_pages: number;
}

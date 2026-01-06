/**
 * Safe Drain API 服务
 * 提供与后端 Safe Drain 功能的通信接口
 */
import type {
    SafeDrainRequest,
    SafeDrainResponse,
    DrainPodMigrationInfo,
} from '@/types/safe-drain';

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || '/api';

// API 错误类型
export class APIError extends Error {
    constructor(
        message: string,
        public readonly statusCode: number,
        public readonly endpoint: string,
        public readonly originalError?: Error
    ) {
        super(message);
        this.name = 'APIError';
    }
}

export class TimeoutError extends Error {
    constructor(endpoint: string) {
        super(`API request timeout: ${endpoint}`);
        this.name = 'TimeoutError';
    }
}

export class NetworkError extends Error {
    constructor(message: string, public readonly originalError?: Error) {
        super(message);
        this.name = 'NetworkError';
    }
}

// Safe Drain API 服务类
class SafeDrainAPIService {
    private async request<T>(
        endpoint: string,
        options?: RequestInit & { timeoutMs?: number }
    ): Promise<T> {
        const headers: HeadersInit = {
            'Content-Type': 'application/json',
            'Accept': 'application/json',
        };

        const controller = new AbortController();
        const timeoutMs = options?.timeoutMs;
        let timeoutId: ReturnType<typeof setTimeout> | undefined;
        if (timeoutMs && timeoutMs > 0) {
            timeoutId = setTimeout(
                () => controller.abort(new DOMException('Request timed out', 'AbortError')),
                timeoutMs
            );
        }

        let response: Response;
        try {
            response = await fetch(`${API_BASE_URL}${endpoint}`, {
                cache: 'no-store',
                credentials: 'include',
                ...options,
                headers: {
                    ...headers,
                    ...options?.headers,
                },
                signal: controller.signal,
            });
        } catch (err) {
            if (timeoutId) clearTimeout(timeoutId);

            if (err instanceof Error) {
                if (err.name === 'AbortError') {
                    throw new TimeoutError(endpoint);
                }
                throw new NetworkError(`Network request failed: ${err.message}`, err);
            }
            throw new NetworkError(`Unknown network error: ${String(err)}`);
        } finally {
            if (timeoutId) clearTimeout(timeoutId);
        }

        if (!response.ok) {
            let errorMessage = `HTTP ${response.status}`;
            try {
                const errorData = await response.json();
                errorMessage = errorData.message || errorData.error || errorMessage;
            } catch {
                // 如果无法解析错误响应，使用默认的HTTP状态码错误
            }
            throw new APIError(errorMessage, response.status, endpoint);
        }

        const fullResponse = await response.json();

        // 验证响应格式
        if (!fullResponse || typeof fullResponse !== 'object') {
            throw new APIError('Invalid response format: expected object', response.status, endpoint);
        }

        // 处理后端返回的统一格式 {success, message, data}
        if (fullResponse.success === true && fullResponse.data !== undefined) {
            return fullResponse.data;
        }

        // 也检查 code 格式（兼容性）
        if (fullResponse.code === 200 && fullResponse.data !== undefined) {
            return fullResponse.data;
        }

        // 如果响应格式不符合预期，返回完整响应
        return fullResponse;
    }

    /**
     * 开始 drain 操作
     */
    async startDrain(request: SafeDrainRequest): Promise<SafeDrainResponse> {
        return this.request<SafeDrainResponse>('/v1/navy/drain/start', {
            method: 'POST',
            body: JSON.stringify(request),
        });
    }

    /**
     * 取消 drain 操作
     */
    async cancelDrain(drainId: string): Promise<{ success: boolean; message: string }> {
        return this.request<{ success: boolean; message: string }>(`/v1/navy/drain/${drainId}/cancel`, {
            method: 'POST',
        });
    }

    /**
     * 获取详细的迁移信息
     */
    async getDrainMigrations(drainId: string): Promise<{ drainId: string; migrations: DrainPodMigrationInfo[] }> {
        return this.request<{ drainId: string; migrations: DrainPodMigrationInfo[] }>(
            `/v1/navy/drain/${drainId}/migrations`,
            { timeoutMs: 8000 }
        );
    }
}

// 单例实例
export const safeDrainAPI = new SafeDrainAPIService();

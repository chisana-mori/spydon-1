/**
 * MinIO 客户端工具类
 * 用于获取告警的原始数据
 */

export interface MinIOConfig {
  endpoint: string;
  accessKey?: string;
  secretKey?: string;
  useSSL?: boolean;
  region?: string;
  bucket?: string;
}

export interface RawPayloadData {
  key: string;
  data: any;
  contentType: string;
  size: number;
  lastModified: Date;
}

export class MinIOClient {
  private config: MinIOConfig;

  constructor(config?: Partial<MinIOConfig>) {
    // 从统一配置中读取默认值
    const { appConfig } = require('@/config')
    this.config = {
      endpoint: appConfig.minio.endpoint,
      accessKey: appConfig.minio.accessKey,
      secretKey: appConfig.minio.secretKey,
      bucket: appConfig.minio.bucket,
      useSSL: appConfig.minio.useSSL,
      region: appConfig.minio.region,
      ...config,
    };
  }

  /**
   * 获取 API 基础 URL，支持运行时配置动态更新
   */
  private getApiBaseUrl(): string {
    const { appConfig } = require('@/config')
    return appConfig.apiBaseUrl || 'http://localhost:8080/api/v1'
  }

  /**
   * 通过 raw_payload_key 获取原始数据
   * 现在通过后端 API 获取，而不是直接访问 MinIO
   */
  async getRawPayload(rawPayloadKey: string): Promise<RawPayloadData | null> {
    try {
      // 从 raw_payload_key 中提取告警 ID
      // 假设 key 格式为: alerts/cluster-id/alert-uuid
      const alertId = this.extractAlertIdFromKey(rawPayloadKey);
      if (!alertId) {
        return null;
      }

      // 通过后端 API 获取原始数据
      const url = `${this.getApiBaseUrl()}/alerts/${alertId}/raw-payload`;

      const response = await fetch(url, {
        method: 'GET',
        headers: {
          'Accept': 'application/json, text/plain, */*',
        },
      });

      if (!response.ok) {
        return null;
      }

      const contentType = response.headers.get('content-type') || 'application/json';
      const contentLength = response.headers.get('content-length');
      const payloadKey = response.headers.get('x-raw-payload-key') || rawPayloadKey;

      let data: any;
      if (contentType.includes('application/json')) {
        data = await response.json();
      } else if (contentType.includes('text/')) {
        data = await response.text();
      } else {
        data = await response.arrayBuffer();
      }

      return {
        key: payloadKey,
        data,
        contentType,
        size: contentLength ? parseInt(contentLength) : 0,
        lastModified: new Date(),
      };
    } catch (error) {
      return null;
    }
  }

  /**
   * 从 raw_payload_key 中提取告警 ID
   * 这是一个临时方案，理想情况下应该直接传递告警 ID
   */
  private extractAlertIdFromKey(rawPayloadKey: string): string | null {
    // 尝试从 key 中提取 UUID
    // 格式可能是: alerts/cluster-id/uuid 或其他格式
    const uuidRegex = /[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}/i;
    const match = rawPayloadKey.match(uuidRegex);
    return match ? match[0] : null;
  }

  /**
   * 构建 MinIO 对象 URL
   */
  private buildObjectUrl(objectKey: string): string {
    const { endpoint, bucket } = this.config;

    // 移除 endpoint 末尾的斜杠
    const cleanEndpoint = endpoint.replace(/\/$/, '');

    // 确保 objectKey 不以斜杠开头
    const cleanObjectKey = objectKey.replace(/^\//, '');

    return `${cleanEndpoint}/${bucket}/${cleanObjectKey}`;
  }

  /**
   * 测试 MinIO 连接
   */
  async testConnection(): Promise<boolean> {
    try {
      const healthUrl = `${this.config.endpoint}/minio/health/live`;
      const response = await fetch(healthUrl);
      return response.ok;
    } catch (error) {
      return false;
    }
  }

  /**
   * 获取配置信息（用于调试）
   */
  getConfig(): MinIOConfig {
    return { ...this.config };
  }
}

// 创建默认实例
export const minioClient = new MinIOClient();

// 导出便捷方法
export const getRawPayload = (rawPayloadKey: string) => minioClient.getRawPayload(rawPayloadKey);
export const testMinIOConnection = () => minioClient.testConnection();

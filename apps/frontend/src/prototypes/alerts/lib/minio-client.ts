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
}

export interface RawPayloadData {
  key: string;
  data: any;
  contentType: string;
  lastModified: Date;
  size: number;
}

export class MinIOClient {
  private config: MinIOConfig;

  constructor(config: MinIOConfig) {
    this.config = {
      useSSL: true,
      region: 'us-east-1',
      ...config,
    };
  }

  /**
   * 通过 raw_payload_key 获取原始数据
   */
  async getRawPayload(rawPayloadKey: string): Promise<RawPayloadData | null> {
    try {
      // 构建 MinIO 对象 URL
      const url = this.buildObjectUrl(rawPayloadKey);

      const response = await fetch(url, {
        method: 'GET',
        headers: {
          'Accept': 'application/json, text/plain, */*',
        },
      });

      if (!response.ok) {
        return null;
      }

      const contentType = response.headers.get('content-type') || 'application/octet-stream';
      const lastModified = new Date(response.headers.get('last-modified') || Date.now());
      const size = parseInt(response.headers.get('content-length') || '0');

      let data: any;

      // 根据内容类型解析数据
      if (contentType.includes('application/json')) {
        data = await response.json();
      } else if (contentType.includes('text/')) {
        data = await response.text();
      } else {
        data = await response.blob();
      }

      return {
        key: rawPayloadKey,
        data,
        contentType,
        lastModified,
        size,
      };
    } catch (error) {
      return null;
    }
  }

  /**
   * 构建 MinIO 对象访问 URL
   */
  private buildObjectUrl(key: string): string {
    const protocol = this.config.useSSL ? 'https' : 'http';
    const baseUrl = `${protocol}://${this.config.endpoint}`;

    // 如果 key 已经包含 bucket 信息，直接使用
    if (key.startsWith('http')) {
      return key;
    }

    // 否则构建完整的 URL
    return `${baseUrl}/${key}`;
  }

  /**
   * 获取预签名 URL（如果需要认证）
   */
  async getPresignedUrl(rawPayloadKey: string, expiry: number = 3600): Promise<string | null> {
    try {
      // 这里需要调用后端 API 来获取预签名 URL
      const response = await fetch('/api/minio/presigned-url', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          key: rawPayloadKey,
          expiry,
        }),
      });

      if (!response.ok) {
        throw new Error(`Failed to get presigned URL: ${response.statusText}`);
      }

      const { url } = await response.json();
      return url;
    } catch (error) {
      return null;
    }
  }
}

// 默认 MinIO 客户端实例
export const minioClient = new MinIOClient({
  endpoint: process.env.NEXT_PUBLIC_MINIO_ENDPOINT || 'localhost:9000',
  useSSL: process.env.NEXT_PUBLIC_MINIO_USE_SSL === 'true',
});

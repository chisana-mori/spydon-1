import { NextRequest, NextResponse } from 'next/server';

/**
 * 获取 MinIO 预签名 URL 的 API 路由
 * POST /api/minio/presigned-url
 */
export async function POST(request: NextRequest) {
  try {
    const { key, expiry = 3600 } = await request.json();

    if (!key) {
      return NextResponse.json(
        { error: 'Missing required parameter: key' },
        { status: 400 }
      );
    }

    // 这里需要根据您的后端 API 来获取预签名 URL
    // 示例：调用后端服务获取预签名 URL
    const backendUrl = process.env.BACKEND_API_URL || 'http://localhost:8000';

    const response = await fetch(`${backendUrl}/api/v1/minio/presigned-url`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        // 如果需要认证，添加认证头
        'Authorization': `Bearer ${process.env.API_TOKEN || ''}`,
      },
      body: JSON.stringify({
        key,
        expiry,
      }),
    });

    if (!response.ok) {
      throw new Error(`Backend API error: ${response.status} ${response.statusText}`);
    }

    const data = await response.json();

    return NextResponse.json({
      url: data.url,
      expiry: data.expiry,
    });

  } catch (error) {
    return NextResponse.json(
      {
        error: 'Failed to get presigned URL',
        details: error instanceof Error ? error.message : 'Unknown error'
      },
      { status: 500 }
    );
  }
}

/**
 * 健康检查
 * GET /api/minio/presigned-url
 */
export async function GET() {
  return NextResponse.json({
    status: 'ok',
    service: 'MinIO Presigned URL API',
    timestamp: new Date().toISOString(),
  });
}

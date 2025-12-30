import { NextResponse } from 'next/server';

/**
 * 健康检测端点
 * 用于 Kubernetes liveness/readiness 探针
 */
export async function GET() {
    return NextResponse.json({ status: 'ok', timestamp: new Date().toISOString() });
}

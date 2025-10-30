import { fileURLToPath } from 'node:url'
import { dirname } from 'node:path'

const __filename = fileURLToPath(import.meta.url)
const __dirname = dirname(__filename)

const normalizeBasePath = (value) => {
  if (!value) {
    return ''
  }
  const trimmed = value.trim()
  if (!trimmed || trimmed === '/') {
    return ''
  }
  const withLeadingSlash = trimmed.startsWith('/') ? trimmed : `/${trimmed}`
  return withLeadingSlash.replace(/\/+$/, '')
}

const basePath = normalizeBasePath(process.env.NEXT_PUBLIC_BASE_PATH)

/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  typedRoutes: true,
  turbopack: {
    root: __dirname,
  },
  env: {
    NEXT_MANAGED_BASE_PATH: basePath ? 'true' : 'false',
  },
  ...(basePath
    ? {
        basePath,
        assetPrefix: basePath,
      }
    : {}),
  async rewrites() {
    // 从环境变量读取后端地址
    // NEXT_PUBLIC_BACKEND_HOST: 仅域名和端口，如 http://localhost:8080
    // NEXT_PUBLIC_BACKEND_BASE_URL: 包含 basePath，如 http://your-domain/spydon
    const backendHost = process.env.NEXT_PUBLIC_BACKEND_HOST || 
                        process.env.NEXT_PUBLIC_BACKEND_BASE_URL || 
                        'http://localhost:8080';

    return [
      {
        source: '/api/:path*',
        destination: `${backendHost}/api/:path*`,
      },
      {
        source: '/auth/:path*',
        destination: `${backendHost}/auth/:path*`,
      },
    ];
  },
};

export default nextConfig;

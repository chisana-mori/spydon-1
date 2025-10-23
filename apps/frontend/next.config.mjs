/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  experimental: {
    typedRoutes: true,
  },
  async rewrites() {
    // 从环境变量读取后端地址，默认为 localhost:8080
    // 注意：这里只需要 host:port，不包含 /api 路径
    const backendHost = process.env.NEXT_PUBLIC_BACKEND_HOST || 'http://localhost:8080';

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

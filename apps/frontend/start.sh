#!/bin/sh

# 启动脚本：同时运行 nginx 和 Next.js

# 等待后端服务就绪
echo "Waiting for backend service..."
while ! nc -z backend 8080; do
    sleep 1
done
echo "Backend service is ready!"

# 启动 Next.js 服务（SSR）
echo "Starting Next.js server..."
NODE_ENV=production npm start &

# 等待 Next.js 启动
sleep 5

# 启动 nginx
echo "Starting nginx..."
nginx -g "daemon off;"

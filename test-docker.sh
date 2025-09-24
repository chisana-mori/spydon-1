#!/bin/bash

# Robusta Web Docker测试脚本

set -e

echo "🚀 开始Docker环境测试..."

# 清理之前的容器
echo "📦 清理之前的容器..."
docker-compose down -v 2>/dev/null || true

# 构建镜像
echo "🔨 构建Docker镜像..."
docker-compose build

# 启动服务
echo "🌟 启动服务..."
docker-compose up -d

# 等待服务启动
echo "⏳ 等待服务启动..."
sleep 30

# 检查服务状态
echo "🔍 检查服务状态..."
docker-compose ps

# 测试后端健康检查
echo "🏥 测试后端健康检查..."
if curl -f http://localhost:8080/health; then
    echo "✅ 后端服务健康检查通过"
else
    echo "❌ 后端服务健康检查失败"
    docker-compose logs backend
    exit 1
fi

# 测试前端服务
echo "🌐 测试前端服务..."
if curl -f http://localhost:3000/health; then
    echo "✅ 前端服务健康检查通过"
else
    echo "❌ 前端服务健康检查失败"
    docker-compose logs frontend
    exit 1
fi

# 测试数据库连接
echo "🗄️ 测试数据库连接..."
if docker-compose exec -T postgres pg_isready -U postgres; then
    echo "✅ 数据库连接正常"
else
    echo "❌ 数据库连接失败"
    docker-compose logs postgres
    exit 1
fi

# 测试MinIO服务
echo "📦 测试MinIO服务..."
if curl -f http://localhost:9000/minio/health/live; then
    echo "✅ MinIO服务正常"
else
    echo "❌ MinIO服务异常"
    docker-compose logs minio
    exit 1
fi

# 测试API端点
echo "🔌 测试API端点..."
if curl -f http://localhost:8080/api/v1/clusters/summary; then
    echo "✅ API端点响应正常"
else
    echo "❌ API端点响应异常"
    docker-compose logs backend
    exit 1
fi

echo "🎉 所有测试通过！"
echo "📊 服务访问地址："
echo "  - 前端: http://localhost:3000"
echo "  - 后端API: http://localhost:8080"
echo "  - MinIO控制台: http://localhost:9001"

echo "🛑 停止服务..."
docker-compose down

echo "✨ 测试完成！"

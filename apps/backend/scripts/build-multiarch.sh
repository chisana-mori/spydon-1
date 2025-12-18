#!/bin/bash
# 多架构 Docker 镜像构建脚本 (后端)
# 使用 buildx 直接构建多架构镜像

set -e

# 默认配置
REGISTRY="${REGISTRY:-}"
IMAGE_NAME="${IMAGE_NAME:-robusta-backend}"
VERSION="${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo 'dev')}"
COMMIT_SHA="${COMMIT_SHA:-$(git rev-parse --short HEAD 2>/dev/null || echo 'unknown')}"
BUILD_TIME="${BUILD_TIME:-$(date -u +"%Y-%m-%dT%H:%M:%SZ")}"
PLATFORMS="${PLATFORMS:-linux/amd64,linux/arm64}"
PUSH="${PUSH:-false}"

# Go 代理配置 (国内可设置为 https://goproxy.cn,direct)
GOPROXY="${GOPROXY:-https://proxy.golang.org,direct}"

# 构建完整镜像名称
if [ -n "$REGISTRY" ]; then
    FULL_IMAGE="${REGISTRY}/${IMAGE_NAME}"
else
    FULL_IMAGE="${IMAGE_NAME}"
fi

echo "============================================"
echo "🐳 Building Multi-Arch Backend Image"
echo "============================================"
echo "📦 Image: ${FULL_IMAGE}:${VERSION}"
echo "🖥️  Platforms: ${PLATFORMS}"
echo "🔧 GOPROXY: ${GOPROXY}"
echo "📤 Push: ${PUSH}"
echo "============================================"

# 确保 builder 存在
BUILDER_NAME="robusta-multiarch"
if ! docker buildx inspect ${BUILDER_NAME} &> /dev/null; then
    docker buildx create --name ${BUILDER_NAME} --driver docker-container --bootstrap
fi
docker buildx use ${BUILDER_NAME}

# 构建命令
if [ "$PUSH" = "true" ]; then
    # 多架构推送到仓库
    docker buildx build \
        --platform "${PLATFORMS}" \
        --build-arg VERSION="${VERSION}" \
        --build-arg COMMIT_SHA="${COMMIT_SHA}" \
        --build-arg BUILD_TIME="${BUILD_TIME}" \
        --build-arg GOPROXY="${GOPROXY}" \
        --tag "${FULL_IMAGE}:${VERSION}" \
        --tag "${FULL_IMAGE}:latest" \
        --file Dockerfile.multiarch \
        --push .
else
    # 本地加载（仅当前架构）
    docker buildx build \
        --build-arg VERSION="${VERSION}" \
        --build-arg COMMIT_SHA="${COMMIT_SHA}" \
        --build-arg BUILD_TIME="${BUILD_TIME}" \
        --build-arg GOPROXY="${GOPROXY}" \
        --tag "${FULL_IMAGE}:${VERSION}" \
        --tag "${FULL_IMAGE}:latest" \
        --file Dockerfile.multiarch \
        --load .
fi

echo "✅ Build completed: ${FULL_IMAGE}:${VERSION}"

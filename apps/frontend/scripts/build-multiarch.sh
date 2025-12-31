#!/bin/bash
# 多架构 Docker 镜像构建脚本 (前端)
# 使用 buildx 直接构建多架构镜像

set -e

# 默认配置
REGISTRY="${REGISTRY:-}"
IMAGE_NAME="${IMAGE_NAME:-robusta-frontend}"
VERSION="${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo 'dev')}"
COMMIT_SHA="${COMMIT_SHA:-$(git rev-parse --short HEAD 2>/dev/null || echo 'unknown')}"
BUILD_TIME="${BUILD_TIME:-$(date -u +"%Y-%m-%dT%H:%M:%SZ")}"
PLATFORMS="${PLATFORMS:-linux/amd64,linux/arm64}"
PUSH="${PUSH:-false}"

# NPM 镜像源配置 (国内可设置为 https://registry.npmmirror.com)
NPM_REGISTRY="${NPM_REGISTRY:-https://registry.npmjs.org}"

# 构建完整镜像名称
if [ -n "$REGISTRY" ]; then
    FULL_IMAGE="${REGISTRY}/${IMAGE_NAME}"
else
    FULL_IMAGE="${IMAGE_NAME}"
fi

echo "============================================"
echo "🐳 Building Multi-Arch Frontend Image"
echo "============================================"
echo "📦 Image: ${FULL_IMAGE}:${VERSION}"
echo "🖥️  Platforms: ${PLATFORMS}"
echo "🔧 NPM Registry: ${NPM_REGISTRY}"
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
        --build-arg NPM_REGISTRY="${NPM_REGISTRY}" \
        --tag "${FULL_IMAGE}:${VERSION}" \
        --tag "${FULL_IMAGE}:latest" \
        --file Dockerfile.multiarch \
        --push .
else
    # 本地加载（强制 linux/arm64 并保存为 frontend.tar）
    docker buildx build \
        --platform linux/arm64 \
        --build-arg VERSION="${VERSION}" \
        --build-arg COMMIT_SHA="${COMMIT_SHA}" \
        --build-arg BUILD_TIME="${BUILD_TIME}" \
        --build-arg NPM_REGISTRY="${NPM_REGISTRY}" \
        --tag "${FULL_IMAGE}:${VERSION}" \
        --tag "${FULL_IMAGE}:latest" \
        --file Dockerfile.multiarch \
        --load .

    echo "💾 Saving image to frontend.tar..."
    docker save -o frontend.tar "${FULL_IMAGE}:${VERSION}"
fi

echo "✅ Build completed: ${FULL_IMAGE}:${VERSION}"

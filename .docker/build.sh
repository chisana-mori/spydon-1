#!/bin/bash

# Docker 镜像构建脚本
# 支持多架构构建和缓存优化

set -euo pipefail

# 默认配置
REGISTRY="${REGISTRY:-ghcr.io}"
REPO="${REPO:-$(git config --get remote.origin.url | sed 's|.*github.com[:/]||; s|\.git||')}"
BACKEND_IMAGE="${REGISTRY}/${REPO}/backend"
FRONTEND_IMAGE="${REGISTRY}/${REPO}/frontend"
VERSION="${VERSION:-$(git rev-parse --short HEAD)}"
BUILD_TIME="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
COMMIT_SHA="$(git rev-parse HEAD)"

# 构建参数
PLATFORM="${PLATFORM:-linux/amd64}"
CACHE_FROM="${CACHE_FROM:-type=gha}"
CACHE_TO="${CACHE_TO:-type=gha,mode=max}"
PUSH="${PUSH:-false}"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
    exit 1
}

# 检查依赖
check_deps() {
    log "Checking dependencies..."

    if ! command -v docker &> /dev/null; then
        error "Docker is not installed"
    fi

    if ! docker buildx version &> /dev/null; then
        error "Docker Buildx is not available"
    fi

    # 检查 BuildKit
    if ! docker buildx inspect default &> /dev/null; then
        log "Creating new Buildx builder..."
        docker buildx create --name multiarch --driver docker-container --use
        docker buildx inspect --bootstrap
    fi
}

# 构建后端镜像
build_backend() {
    log "Building backend image..."

    cd apps/backend

    docker buildx build \
        --platform "${PLATFORM}" \
        --file Dockerfile.optimized \
        --tag "${BACKEND_IMAGE}:${VERSION}" \
        --tag "${BACKEND_IMAGE}:latest" \
        --build-arg VERSION="${VERSION}" \
        --build-arg COMMIT_SHA="${COMMIT_SHA}" \
        --build-arg BUILD_TIME="${BUILD_TIME}" \
        --cache-from "${CACHE_FROM}" \
        --cache-to "${CACHE_TO}" \
        ${PUSH:+--push} \
        .

    cd - > /dev/null

    log "Backend image built successfully: ${BACKEND_IMAGE}:${VERSION}"
}

# 构建前端镜像
build_frontend() {
    log "Building frontend image..."

    cd apps/frontend

    docker buildx build \
        --platform "${PLATFORM}" \
        --file Dockerfile.optimized \
        --tag "${FRONTEND_IMAGE}:${VERSION}" \
        --tag "${FRONTEND_IMAGE}:latest" \
        --build-arg VERSION="${VERSION}" \
        --build-arg COMMIT_SHA="${COMMIT_SHA}" \
        --build-arg BUILD_TIME="${BUILD_TIME}" \
        --build-arg NEXT_PUBLIC_API_URL="${NEXT_PUBLIC_API_URL:-http://localhost:8080}" \
        --cache-from "${CACHE_FROM}" \
        --cache-to "${CACHE_TO}" \
        ${PUSH:+--push} \
        .

    cd - > /dev/null

    log "Frontend image built successfully: ${FRONTEND_IMAGE}:${VERSION}"
}

# 安全扫描
security_scan() {
    log "Running security scan..."

    # 后端镜像扫描
    log "Scanning backend image..."
    trivy image --exit-code 0 --severity HIGH,CRITICAL "${BACKEND_IMAGE}:${VERSION}" || true

    # 前端镜像扫描
    log "Scanning frontend image..."
    trivy image --exit-code 0 --severity HIGH,CRITICAL "${FRONTEND_IMAGE}:${VERSION}" || true
}

# 验证镜像
validate_images() {
    log "Validating built images..."

    # 检查镜像是否存在
    if ! docker image inspect "${BACKEND_IMAGE}:${VERSION}" &> /dev/null; then
        error "Backend image validation failed"
    fi

    if ! docker image inspect "${FRONTEND_IMAGE}:${VERSION}" &> /dev/null; then
        error "Frontend image validation failed"
    fi

    # 检查镜像大小
    BACKEND_SIZE=$(docker image inspect "${BACKEND_IMAGE}:${VERSION}" --format='{{.Size}}')
    FRONTEND_SIZE=$(docker image inspect "${FRONTEND_IMAGE}:${VERSION}" --format='{{.Size}}')

    log "Backend image size: $(numfmt --to=iec $BACKEND_SIZE)"
    log "Frontend image size: $(numfmt --to=iec $FRONTEND_SIZE)"
}

# 显示构建信息
show_build_info() {
    log "Build Information:"
    echo "  Version: ${VERSION}"
    echo "  Commit: ${COMMIT_SHA}"
    echo "  Build Time: ${BUILD_TIME}"
    echo "  Platform: ${PLATFORM}"
    echo "  Backend Image: ${BACKEND_IMAGE}"
    echo "  Frontend Image: ${FRONTEND_IMAGE}"
    echo "  Push: ${PUSH}"
}

# 主函数
main() {
    log "Starting Docker build process..."

    show_build_info
    check_deps

    # 构建镜像
    build_backend
    build_frontend

    # 验证镜像
    validate_images

    # 安全扫描（如果安装了 trivy）
    if command -v trivy &> /dev/null; then
        security_scan
    else
        warn "Trivy not found, skipping security scan"
    fi

    log "Docker build process completed successfully!"
}

# 命令行参数解析
while [[ $# -gt 0 ]]; do
    case $1 in
        --push)
            PUSH=true
            shift
            ;;
        --platform)
            PLATFORM="$2"
            shift 2
            ;;
        --version)
            VERSION="$2"
            shift 2
            ;;
        --registry)
            REGISTRY="$2"
            shift 2
            ;;
        --backend-only)
            BUILD_BACKEND_ONLY=true
            shift
            ;;
        --frontend-only)
            BUILD_FRONTEND_ONLY=true
            shift
            ;;
        -h|--help)
            cat << EOF
Usage: $0 [OPTIONS]

Options:
  --push              Push images to registry
  --platform PLATFORM Build platform (default: linux/amd64)
  --version VERSION   Image version (default: git commit SHA)
  --registry REGISTRY Container registry (default: ghcr.io)
  --backend-only      Build only backend image
  --frontend-only     Build only frontend image
  -h, --help          Show this help message

Examples:
  $0 --push --version v1.0.0
  $0 --platform linux/amd64,linux/arm64
  $0 --backend-only --push
EOF
            exit 0
            ;;
        *)
            error "Unknown option: $1"
            ;;
    esac
done

# 根据参数执行特定构建
if [[ "${BUILD_BACKEND_ONLY:-}" == "true" ]]; then
    log "Building backend only..."
    show_build_info
    check_deps
    build_backend
    validate_images
    log "Backend build completed!"
    exit 0
fi

if [[ "${BUILD_FRONTEND_ONLY:-}" == "true" ]]; then
    log "Building frontend only..."
    show_build_info
    check_deps
    build_frontend
    validate_images
    log "Frontend build completed!"
    exit 0
fi

# 执行完整构建流程
main

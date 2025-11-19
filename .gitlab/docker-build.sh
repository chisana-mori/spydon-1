#!/bin/bash

# GitLab CI Docker 构建脚本
# 适配 GitLab CI 环境的 Docker 镜像构建

set -euo pipefail

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

info() {
    echo -e "${BLUE}[DOCKER]${NC} $1"
}

# GitLab CI 变量
REGISTRY="${CI_REGISTRY:-}"
BACKEND_IMAGE="${CI_REGISTRY_IMAGE}/backend"
FRONTEND_IMAGE="${CI_REGISTRY_IMAGE}/frontend"
COMMIT_SHA="${CI_COMMIT_SHA:-}"
COMMIT_REF_SLUG="${CI_COMMIT_REF_SLUG:-}"
PROJECT_ID="${CI_PROJECT_ID:-}"
JOB_TOKEN="${CI_JOB_TOKEN:-}"

# 构建参数
PLATFORM="${PLATFORM:-linux/amd64,linux/arm64}"
PUSH="${PUSH:-true}"
BUILD_CONTEXT="${BUILD_CONTEXT:-.}"

# 检查 GitLab CI 环境
check_gitlab_env() {
    log "Checking GitLab CI environment..."

    if [ -z "$REGISTRY" ]; then
        error "CI_REGISTRY is not set"
        exit 1
    fi

    if [ -z "$COMMIT_SHA" ]; then
        error "CI_COMMIT_SHA is not set"
        exit 1
    fi

    log "GitLab CI environment verified"
    log "Registry: $REGISTRY"
    log "Backend Image: $BACKEND_IMAGE"
    log "Frontend Image: $FRONTEND_IMAGE"
    log "Commit: $COMMIT_SHA"
}

# 设置 Docker BuildKit
setup_buildkit() {
    log "Setting up Docker BuildKit..."

    # 启用 BuildKit
    export DOCKER_BUILDKIT=1

    # 创建 buildx 构建器（如果不存在）
    if ! docker buildx inspect gitlab-builder &> /dev/null; then
        log "Creating buildx builder..."
        docker buildx create --name gitlab-builder --driver docker-container --use
        docker buildx inspect --bootstrap
    else
        log "Using existing buildx builder..."
        docker buildx use gitlab-builder
    fi
}

# 登录 GitLab Container Registry
login_registry() {
    log "Logging into GitLab Container Registry..."

    if [ -n "${CI_REGISTRY_USER:-}" ] && [ -n "${CI_REGISTRY_PASSWORD:-}" ]; then
        echo "$CI_REGISTRY_PASSWORD" | docker login -u "$CI_REGISTRY_USER" --password-stdin "$REGISTRY"
        log "Successfully logged into $REGISTRY"
    else
        warn "CI_REGISTRY_USER or CI_REGISTRY_PASSWORD not set, skipping registry login"
    fi
}

# 构建后端镜像
build_backend() {
    log "Building backend Docker image..."

    local backend_context="apps/backend"
    if [ ! -d "$backend_context" ]; then
        error "Backend context directory not found: $backend_context"
        exit 1
    fi

    # 构建参数
    local build_args=(
        "--platform" "$PLATFORM"
        "--file" "$backend_context/Dockerfile.optimized"
        "--tag" "$BACKEND_IMAGE:$COMMIT_SHA"
        "--tag" "$BACKEND_IMAGE:$COMMIT_REF_SLUG"
        "--build-arg" "VERSION=$COMMIT_REF_SLUG"
        "--build-arg" "COMMIT_SHA=$COMMIT_SHA"
        "--build-arg" "BUILD_TIME=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
        "--build-arg" "CI_PROJECT_ID=$PROJECT_ID"
        "--build-arg" "CI_PIPELINE_ID=${CI_PIPELINE_ID:-unknown}"
        "--cache-from" "type=registry,ref=$BACKEND_IMAGE:cache"
        "--cache-to" "type=registry,ref=$BACKEND_IMAGE:cache,mode=max"
    )

    # 如果需要推送
    if [ "$PUSH" = "true" ]; then
        build_args+=("--push")
    else
        build_args+=("--load")
    fi

    log "Building backend with args: ${build_args[*]}"
    docker buildx build "${build_args[@]}" "$backend_context"

    # 验证镜像
    if [ "$PUSH" = "true" ]; then
        log "Verifying backend image..."
        docker buildx imagetools inspect "$BACKEND_IMAGE:$COMMIT_SHA"
    fi

    log "Backend image built successfully: $BACKEND_IMAGE:$COMMIT_SHA"
}

# 构建前端镜像
build_frontend() {
    log "Building frontend Docker image..."

    local frontend_context="apps/frontend"
    if [ ! -d "$frontend_context" ]; then
        error "Frontend context directory not found: $frontend_context"
        exit 1
    fi

    # 构建参数
    local build_args=(
        "--platform" "$PLATFORM"
        "--file" "$frontend_context/Dockerfile.optimized"
        "--tag" "$FRONTEND_IMAGE:$COMMIT_SHA"
        "--tag" "$FRONTEND_IMAGE:$COMMIT_REF_SLUG"
        "--build-arg" "VERSION=$COMMIT_REF_SLUG"
        "--build-arg" "COMMIT_SHA=$COMMIT_SHA"
        "--build-arg" "BUILD_TIME=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
        "--build-arg" "CI_PROJECT_ID=$PROJECT_ID"
        "--build-arg" "CI_PIPELINE_ID=${CI_PIPELINE_ID:-unknown}"
        "--build-arg" "NEXT_PUBLIC_API_URL=${NEXT_PUBLIC_API_URL:-http://localhost:8080}"
        "--cache-from" "type=registry,ref=$FRONTEND_IMAGE:cache"
        "--cache-to" "type=registry,ref=$FRONTEND_IMAGE:cache,mode=max"
    )

    # 如果需要推送
    if [ "$PUSH" = "true" ]; then
        build_args+=("--push")
    else
        build_args+=("--load")
    fi

    log "Building frontend with args: ${build_args[*]}"
    docker buildx build "${build_args[@]}" "$frontend_context"

    # 验证镜像
    if [ "$PUSH" = "true" ]; then
        log "Verifying frontend image..."
        docker buildx imagetools inspect "$FRONTEND_IMAGE:$COMMIT_SHA"
    fi

    log "Frontend image built successfully: $FRONTEND_IMAGE:$COMMIT_SHA"
}

# 安全扫描
security_scan() {
    log "Running security scans..."

    # 检查是否安装了 trivy
    if command -v trivy &> /dev/null; then
        # 后端镜像扫描
        log "Scanning backend image..."
        trivy image --exit-code 0 --format json --output backend-trivy.json "$BACKEND_IMAGE:$COMMIT_SHA" || true

        # 前端镜像扫描
        log "Scanning frontend image..."
        trivy image --exit-code 0 --format json --output frontend-trivy.json "$FRONTEND_IMAGE:$COMMIT_SHA" || true

        # 生成报告摘要
        log "Generating security scan summary..."
        if [ -f "backend-trivy.json" ]; then
            local backend_vulns
            backend_vulns=$(jq -r '.Results[]?.Vulnerabilities | length' backend-trivy.json | awk '{sum+=$1} END {print sum+0}')
            log "Backend vulnerabilities found: $backend_vulns"
        fi

        if [ -f "frontend-trivy.json" ]; then
            local frontend_vulns
            frontend_vulns=$(jq -r '.Results[]?.Vulnerabilities | length' frontend-trivy.json | awk '{sum+=$1} END {print sum+0}')
            log "Frontend vulnerabilities found: $frontend_vulns"
        fi
    else
        warn "Trivy not found, skipping security scan"
    fi
}

# 生成构建信息
generate_build_info() {
    log "Generating build information..."

    local build_info_file="build-info.json"
    cat > "$build_info_file" << EOF
{
  "build": {
    "version": "$COMMIT_REF_SLUG",
    "commit_sha": "$COMMIT_SHA",
    "build_time": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
    "pipeline_id": "${CI_PIPELINE_ID:-unknown}",
    "job_id": "${CI_JOB_ID:-unknown}",
    "project_id": "$PROJECT_ID"
  },
  "images": {
    "backend": {
      "repository": "$BACKEND_IMAGE",
      "tag": "$COMMIT_SHA"
    },
    "frontend": {
      "repository": "$FRONTEND_IMAGE",
      "tag": "$COMMIT_SHA"
    }
  },
  "platform": "$PLATFORM"
}
EOF

    log "Build information written to: $build_info_file"
    cat "$build_info_file"
}

# 验证构建结果
verify_build() {
    log "Verifying build results..."

    # 检查镜像是否存在
    if [ "$PUSH" = "true" ]; then
        # 验证镜像在注册表中
        if command -v curl &> /dev/null && [ -n "$JOB_TOKEN" ]; then
            # 验证后端镜像
            local backend_manifest_url="https://$REGISTRY/v2/$CI_PROJECT_PATH/backend/manifests/$COMMIT_SHA"
            if curl --header "Authorization: Bearer $JOB_TOKEN" "$backend_manifest_url" &> /dev/null; then
                log "✅ Backend image verified in registry"
            else
                warn "⚠️ Backend image verification failed"
            fi

            # 验证前端镜像
            local frontend_manifest_url="https://$REGISTRY/v2/$CI_PROJECT_PATH/frontend/manifests/$COMMIT_SHA"
            if curl --header "Authorization: Bearer $JOB_TOKEN" "$frontend_manifest_url" &> /dev/null; then
                log "✅ Frontend image verified in registry"
            else
                warn "⚠️ Frontend image verification failed"
            fi
        fi
    else
        # 本地验证
        if docker image inspect "$BACKEND_IMAGE:$COMMIT_SHA" &> /dev/null; then
            log "✅ Backend image verified locally"
        else
            error "❌ Backend image verification failed"
            exit 1
        fi

        if docker image inspect "$FRONTEND_IMAGE:$COMMIT_SHA" &> /dev/null; then
            log "✅ Frontend image verified locally"
        else
            error "❌ Frontend image verification failed"
            exit 1
        fi
    fi
}

# 清理函数
cleanup() {
    log "Cleaning up..."
    # 清理临时文件
    rm -f backend-trivy.json frontend-trivy.json
}

# 显示帮助信息
show_help() {
    cat << EOF
Usage: $0 [OPTIONS]

GitLab CI Docker Build Script

Options:
  --backend-only       Build only backend image
  --frontend-only      Build only frontend image
  --no-push            Build images without pushing to registry
  --platform PLATFORM  Target platform (default: linux/amd64,linux/arm64)
  --context PATH       Build context (default: .)
  --help               Show this help message

Environment Variables:
  CI_REGISTRY          GitLab Container Registry URL
  CI_REGISTRY_USER     Registry username
  CI_REGISTRY_PASSWORD Registry password
  CI_REGISTRY_IMAGE    Base image name
  CI_COMMIT_SHA        Commit SHA
  CI_COMMIT_REF_SLUG   Commit ref slug
  CI_PROJECT_ID        Project ID
  NEXT_PUBLIC_API_URL  Frontend API URL

Examples:
  $0                    Build and push all images
  $0 --backend-only     Build only backend image
  $0 --no-push          Build images locally
EOF
}

# 主函数
main() {
    local backend_only=false
    local frontend_only=false

    # 解析命令行参数
    while [[ $# -gt 0 ]]; do
        case $1 in
            --backend-only)
                backend_only=true
                shift
                ;;
            --frontend-only)
                frontend_only=true
                shift
                ;;
            --no-push)
                PUSH=false
                shift
                ;;
            --platform)
                PLATFORM="$2"
                shift 2
                ;;
            --context)
                BUILD_CONTEXT="$2"
                shift 2
                ;;
            --help)
                show_help
                exit 0
                ;;
            *)
                error "Unknown option: $1"
                show_help
                exit 1
                ;;
        esac
    done

    # 设置错误处理
    trap cleanup EXIT

    log "Starting GitLab CI Docker build process..."

    # 检查环境
    check_gitlab_env
    setup_buildkit
    login_registry

    # 构建镜像
    if [ "$backend_only" = true ]; then
        build_backend
        security_scan
        verify_build
        generate_build_info
        log "Backend build completed!"
        exit 0
    fi

    if [ "$frontend_only" = true ]; then
        build_frontend
        security_scan
        verify_build
        generate_build_info
        log "Frontend build completed!"
        exit 0
    fi

    # 构建所有镜像
    build_backend
    build_frontend

    # 安全扫描
    security_scan

    # 验证结果
    verify_build

    # 生成构建信息
    generate_build_info

    log "All Docker images built successfully!"
}

# 执行主函数
main "$@"

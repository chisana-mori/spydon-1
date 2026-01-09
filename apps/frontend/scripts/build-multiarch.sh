#!/bin/bash
# ==============================================================================
# 多架构 Docker 镜像构建脚本 (前端)
# 支持 Docker Desktop, OrbStack, Linux 原生 Docker
# 支持 buildx 多平台构建和本地单架构构建
# ==============================================================================

set -e

# ========================= 配置与参数 =========================
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

# 镜像配置
REGISTRY="${REGISTRY:-}"
IMAGE_NAME="${IMAGE_NAME:-robusta-frontend}"
VERSION="${VERSION:-$(git -C "$PROJECT_DIR" describe --tags --always --dirty 2>/dev/null || echo 'dev')}"
COMMIT_SHA="${COMMIT_SHA:-$(git -C "$PROJECT_DIR" rev-parse --short HEAD 2>/dev/null || echo 'unknown')}"
BUILD_TIME="${BUILD_TIME:-$(date -u +"%Y-%m-%dT%H:%M:%SZ")}"

# 构建模式: buildx | local | auto
BUILD_MODE="${BUILD_MODE:-auto}"

# 平台配置
PLATFORMS="${PLATFORMS:-linux/amd64,linux/arm64}"  # buildx 多平台
PLATFORM="${PLATFORM:-}"                            # 单平台覆盖（优先级高于 PLATFORMS）

# 输出配置
PUSH="${PUSH:-false}"
LOAD="${LOAD:-false}"

# 构建选项
PROGRESS="${PROGRESS:-auto}"
NO_CACHE="${NO_CACHE:-false}"
CACHE_FROM="${CACHE_FROM:-}"
CACHE_TO="${CACHE_TO:-type=inline}"

# Builder 配置: auto | default | container | <自定义名称>
BUILDER="${BUILDER:-auto}"
BUILDER_NAME="${BUILDER_NAME:-multiarch-builder}"

# NPM 镜像源配置
NPM_REGISTRY="${NPM_REGISTRY:-https://registry.npmjs.org}"



# Next.js 构建参数
NEXT_PUBLIC_BACKEND_BASE_URL="${NEXT_PUBLIC_BACKEND_BASE_URL:-http://localhost:8080}"
NEXT_PUBLIC_CAS_LOGIN_PATH="${NEXT_PUBLIC_CAS_LOGIN_PATH:-/auth/cas/login}"
NEXT_PUBLIC_CAS_LOGOUT_PATH="${NEXT_PUBLIC_CAS_LOGOUT_PATH:-/auth/cas/logout}"
NEXT_PUBLIC_BASE_PATH="${NEXT_PUBLIC_BASE_PATH:-}"

# Dockerfile 选择
DOCKERFILE="${DOCKERFILE:-Dockerfile.multiarch}"

# ========================= 颜色输出 =========================
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
MAGENTA='\033[0;35m'
NC='\033[0m'

log_info()    { echo -e "${BLUE}ℹ ${1}${NC}"; }
log_success() { echo -e "${GREEN}✅ ${1}${NC}"; }
log_warning() { echo -e "${YELLOW}⚠️  ${1}${NC}"; }
log_error()   { echo -e "${RED}❌ ${1}${NC}"; }
log_step()    { echo -e "${CYAN}▶ ${1}${NC}"; }
log_debug()   { [[ "${DEBUG:-false}" == "true" ]] && echo -e "${MAGENTA}🔍 ${1}${NC}"; }

# ========================= 帮助信息 =========================
print_help() {
    cat << EOF
用法: $0 [选项]

多架构 Docker 镜像构建脚本 (前端)

构建模式:
  BUILD_MODE=buildx    使用 docker buildx 构建（支持多平台）
  BUILD_MODE=local     使用普通 docker build（单平台，最兼容）
  BUILD_MODE=auto      自动检测最佳构建方式（默认）

输出选项:
  PUSH=true            推送镜像到 Registry
  LOAD=true            加载镜像到本地 Docker（仅支持单平台）

平台配置:
  PLATFORMS=linux/amd64,linux/arm64    多平台列表（buildx 模式）
  PLATFORM=linux/arm64                 单平台构建（优先级更高）

Builder 配置:
  BUILDER=auto         自动选择最佳 builder（默认，OrbStack 环境使用 orbstack builder）
  BUILDER=orbstack     使用 OrbStack 内置 builder（仅 OrbStack 环境可用）
  BUILDER=default      使用 Docker 默认 builder
  BUILDER=container    使用 docker-container driver builder（Docker Desktop/Linux 推荐）


Next.js 配置:
  NEXT_PUBLIC_BACKEND_BASE_URL=xxx    后端 API 地址
  NEXT_PUBLIC_CAS_LOGIN_PATH=xxx      CAS 登录路径
  NEXT_PUBLIC_CAS_LOGOUT_PATH=xxx     CAS 登出路径
  NEXT_PUBLIC_BASE_PATH=xxx           应用基础路径

其他选项:
  REGISTRY=xxx         镜像仓库地址
  IMAGE_NAME=xxx       镜像名称（默认: robusta-frontend）
  VERSION=xxx          版本号
  NPM_REGISTRY=xxx     NPM 镜像源
  NO_CACHE=true        禁用构建缓存
  PROGRESS=plain       构建进度显示模式
  DEBUG=true           显示调试信息

使用示例:
  # 本地单架构构建（最简单）
  BUILD_MODE=local $0

  # 单架构 buildx 构建并加载到本地
  PLATFORM=linux/amd64 LOAD=true $0

  # 多架构构建并推送
  PUSH=true REGISTRY=myregistry.com $0

  # 指定 arm64 单平台构建并推送
  PLATFORM=linux/arm64 PUSH=true REGISTRY=myregistry.com $0

  # 使用国内 NPM 镜像源
  NPM_REGISTRY=https://registry.npmmirror.com $0


  # 使用 OrbStack 默认 builder
  BUILDER=default PUSH=true $0
EOF
}

# ========================= 环境检测 =========================
detect_docker_environment() {
    local env_type="docker"

    # 检测 OrbStack
    if docker context show 2>/dev/null | grep -qi "orbstack"; then
        env_type="orbstack"
    elif docker info 2>/dev/null | grep -qi "orbstack"; then
        env_type="orbstack"
    # 检测 Docker Desktop
    elif docker context show 2>/dev/null | grep -qi "desktop"; then
        env_type="docker-desktop"
    elif docker info 2>/dev/null | grep -qi "Docker Desktop"; then
        env_type="docker-desktop"
    fi

    echo "$env_type"
}

detect_host_platform() {
    local os=$(uname -s | tr '[:upper:]' '[:lower:]')
    local arch=$(uname -m)

    case "$arch" in
        x86_64|amd64) arch="amd64" ;;
        aarch64|arm64) arch="arm64" ;;
        *) arch="amd64" ;;
    esac

    echo "linux/${arch}"
}

check_buildx_available() {
    if docker buildx version &>/dev/null; then
        return 0
    fi
    return 1
}

check_builder_exists() {
    local builder_name="$1"
    docker buildx ls 2>/dev/null | grep -q "^${builder_name}[[:space:]]"
}

# ========================= Builder 管理 =========================
ensure_builder() {
    local docker_env=$(detect_docker_environment)
    local selected_builder=""

    log_step "配置 Builder (环境: ${docker_env}, 配置: ${BUILDER})"

    case "$BUILDER" in
        auto)
            # 自动选择策略
            case "$docker_env" in
                orbstack)
                    # OrbStack: 使用 orbstack builder，它原生支持多架构
                    if check_builder_exists "orbstack"; then
                        selected_builder="orbstack"
                        log_info "OrbStack 环境检测到，使用 orbstack builder"
                    else
                        # 回退到 default
                        selected_builder="default"
                        log_info "OrbStack 环境检测到，使用 default builder"
                    fi
                    ;;
                docker-desktop)
                    # Docker Desktop: 使用 docker-container driver
                    if check_builder_exists "$BUILDER_NAME"; then
                        selected_builder="$BUILDER_NAME"
                    else
                        create_container_builder
                        selected_builder="$BUILDER_NAME"
                    fi
                    ;;
                *)
                    # Linux 原生 Docker: 使用 docker-container driver
                    if check_builder_exists "$BUILDER_NAME"; then
                        selected_builder="$BUILDER_NAME"
                    else
                        create_container_builder
                        selected_builder="$BUILDER_NAME"
                    fi
                    ;;
            esac
            ;;
        default)
            selected_builder="default"
            log_info "使用 Docker 默认 builder"
            ;;
        orbstack)
            if check_builder_exists "orbstack"; then
                selected_builder="orbstack"
            else
                log_error "orbstack builder 不存在"
                exit 1
            fi
            ;;
        container)
            if check_builder_exists "$BUILDER_NAME"; then
                selected_builder="$BUILDER_NAME"
            else
                create_container_builder
                selected_builder="$BUILDER_NAME"
            fi
            ;;
        *)
            # 自定义 builder 名称
            if check_builder_exists "$BUILDER"; then
                selected_builder="$BUILDER"
            else
                log_error "Builder '$BUILDER' 不存在"
                exit 1
            fi
            ;;
    esac

    # 切换到选中的 builder
    log_debug "切换到 builder: ${selected_builder}"
    if ! docker buildx use "$selected_builder" 2>&1; then
        log_warning "无法切换到 builder: ${selected_builder}，尝试继续..."
    fi

    # 验证当前 builder
    local current_builder=$(docker buildx ls 2>/dev/null | grep '\*' | awk '{print $1}')
    log_success "当前使用 Builder: ${current_builder}"

    ACTIVE_BUILDER="$selected_builder"
}

create_container_builder() {
    log_info "创建 docker-container builder: ${BUILDER_NAME}"

    # 先删除可能存在的同名 builder
    docker buildx rm "$BUILDER_NAME" 2>/dev/null || true

    # 创建新的 builder
    docker buildx create \
        --name "$BUILDER_NAME" \
        --driver docker-container \
        --driver-opt network=host \
        --bootstrap

    log_success "Builder ${BUILDER_NAME} 创建成功"
}

# ========================= 构建函数 =========================
build_with_buildx() {
    local full_image="$1"
    local effective_platforms="$2"

    log_step "使用 docker buildx 构建..."
    log_info "镜像: ${full_image}:${VERSION}"
    log_info "平台: ${effective_platforms}"
    echo ""

    # 构建参数
    local build_args=(
        "--build-arg" "VERSION=${VERSION}"
        "--build-arg" "COMMIT_SHA=${COMMIT_SHA}"
        "--build-arg" "BUILD_TIME=${BUILD_TIME}"
        "--build-arg" "NPM_REGISTRY=${NPM_REGISTRY}"
        "--build-arg" "NEXT_PUBLIC_BACKEND_BASE_URL=${NEXT_PUBLIC_BACKEND_BASE_URL}"
        "--build-arg" "NEXT_PUBLIC_CAS_LOGIN_PATH=${NEXT_PUBLIC_CAS_LOGIN_PATH}"
        "--build-arg" "NEXT_PUBLIC_CAS_LOGOUT_PATH=${NEXT_PUBLIC_CAS_LOGOUT_PATH}"
        "--build-arg" "NEXT_PUBLIC_BASE_PATH=${NEXT_PUBLIC_BASE_PATH}"
    )



    # 输出参数
    local output_args=()
    if [[ "$PUSH" == "true" ]]; then
        output_args+=("--push")
    elif [[ "$LOAD" == "true" ]]; then
        output_args+=("--load")
    fi

    # 缓存参数
    local cache_args=()
    if [[ -n "$CACHE_FROM" ]]; then
        cache_args+=("--cache-from" "$CACHE_FROM")
    fi
    if [[ -n "$CACHE_TO" ]]; then
        cache_args+=("--cache-to" "$CACHE_TO")
    fi

    # 其他参数
    local extra_args=()
    if [[ "$NO_CACHE" == "true" ]]; then
        extra_args+=("--no-cache")
    fi

    # 执行构建
    docker buildx build \
        --platform="${effective_platforms}" \
        --progress="${PROGRESS}" \
        --tag="${full_image}:${VERSION}" \
        --tag="${full_image}:latest" \
        --file="${PROJECT_DIR}/${DOCKERFILE}" \
        "${build_args[@]}" \
        "${output_args[@]}" \
        "${cache_args[@]}" \
        "${extra_args[@]}" \
        "$PROJECT_DIR"
}

build_with_docker() {
    local full_image="$1"
    local target_platform="$2"

    log_step "使用普通 docker build 构建..."
    log_info "镜像: ${full_image}:${VERSION}"
    log_info "平台: ${target_platform}"
    echo ""

    # 构建参数
    local build_args=(
        "--build-arg" "VERSION=${VERSION}"
        "--build-arg" "COMMIT_SHA=${COMMIT_SHA}"
        "--build-arg" "BUILD_TIME=${BUILD_TIME}"
        "--build-arg" "NPM_REGISTRY=${NPM_REGISTRY}"
        "--build-arg" "NEXT_PUBLIC_BACKEND_BASE_URL=${NEXT_PUBLIC_BACKEND_BASE_URL}"
        "--build-arg" "NEXT_PUBLIC_CAS_LOGIN_PATH=${NEXT_PUBLIC_CAS_LOGIN_PATH}"
        "--build-arg" "NEXT_PUBLIC_CAS_LOGOUT_PATH=${NEXT_PUBLIC_CAS_LOGOUT_PATH}"
        "--build-arg" "NEXT_PUBLIC_BASE_PATH=${NEXT_PUBLIC_BASE_PATH}"
    )



    # 平台参数（如果指定）
    local platform_args=()
    if [[ -n "$target_platform" ]] && [[ "$target_platform" != "$(detect_host_platform)" ]]; then
        platform_args+=("--platform" "$target_platform")
    fi

    # 其他参数
    local extra_args=()
    if [[ "$NO_CACHE" == "true" ]]; then
        extra_args+=("--no-cache")
    fi

    # 执行构建
    docker build \
        --tag="${full_image}:${VERSION}" \
        --tag="${full_image}:latest" \
        --file="${PROJECT_DIR}/${DOCKERFILE}" \
        "${build_args[@]}" \
        "${platform_args[@]}" \
        "${extra_args[@]}" \
        "$PROJECT_DIR"

    # 推送（如果需要）
    if [[ "$PUSH" == "true" ]]; then
        log_step "推送镜像..."
        docker push "${full_image}:${VERSION}"
        docker push "${full_image}:latest"
    fi
}

# ========================= 验证函数 =========================
validate_dockerfile() {
    if [[ ! -f "${PROJECT_DIR}/${DOCKERFILE}" ]]; then
        log_error "Dockerfile 不存在: ${PROJECT_DIR}/${DOCKERFILE}"
        exit 1
    fi
    log_success "Dockerfile 验证通过"
}

validate_nextjs_config() {
    if [[ ! -f "${PROJECT_DIR}/next.config.mjs" ]]; then
        log_warning "next.config.mjs 不存在"
        return
    fi

    if ! grep -q "output: 'standalone'" "${PROJECT_DIR}/next.config.mjs"; then
        log_warning "next.config.mjs 建议配置 'output: standalone' 以优化多架构构建"
    fi

    log_success "Next.js 配置验证通过"
}

validate_config() {
    # 检查推送时是否配置了 registry
    if [[ "$PUSH" == "true" ]] && [[ -z "$REGISTRY" ]]; then
        log_warning "PUSH=true 但未配置 REGISTRY，将推送到 Docker Hub"
    fi

    # 检查 load + 多平台的兼容性
    if [[ "$LOAD" == "true" ]] && [[ "$PLATFORMS" == *","* ]] && [[ -z "$PLATFORM" ]]; then
        log_warning "--load 不支持多平台，自动切换到主机原生平台"
        PLATFORM=$(detect_host_platform)
    fi
}

# ========================= 输出函数 =========================
print_header() {
    echo ""
    echo "════════════════════════════════════════════════════════════════"
    echo "🚀 前端多架构镜像构建"
    echo "════════════════════════════════════════════════════════════════"
}

print_config() {
    local full_image="$1"
    local effective_platforms="$2"

    echo ""
    log_info "构建配置:"
    echo "  📦 镜像:       ${full_image}:${VERSION}"
    echo "  🖥️  平台:       ${effective_platforms}"
    echo "  🔧 构建模式:   ${BUILD_MODE}"
    echo "  📤 推送:       ${PUSH}"
    echo "  📥 加载:       ${LOAD}"
    echo "  🔗 NPM源:      ${NPM_REGISTRY}"
    echo "  📄 Dockerfile: ${DOCKERFILE}"
    echo ""



    log_info "Next.js 配置:"
    echo "  🔗 Backend:    ${NEXT_PUBLIC_BACKEND_BASE_URL}"
    echo "  🔐 CAS Login:  ${NEXT_PUBLIC_CAS_LOGIN_PATH}"
    echo "  🚪 CAS Logout: ${NEXT_PUBLIC_CAS_LOGOUT_PATH}"
    echo "  📍 Base Path:  ${NEXT_PUBLIC_BASE_PATH:-'/'}"
    echo ""
}

print_summary() {
    local full_image="$1"
    local start_time="$2"
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))

    echo ""
    echo "════════════════════════════════════════════════════════════════"
    log_success "构建完成!"
    echo "════════════════════════════════════════════════════════════════"
    echo ""
    echo "  📦 镜像: ${full_image}:${VERSION}"
    echo "  ⏱️  耗时: $((duration / 60))m $((duration % 60))s"
    echo ""

    if [[ "$PUSH" == "true" ]]; then
        echo "💡 拉取镜像:"
        echo "   docker pull ${full_image}:${VERSION}"
    elif [[ "$LOAD" == "true" ]] || [[ "$BUILD_MODE" == "local" ]]; then
        echo "💡 运行镜像:"
        echo "   docker run -it -p 3000:3000 ${full_image}:${VERSION}"
    else
        echo "💡 下一步:"
        echo "   推送: PUSH=true $0"
        echo "   加载: PLATFORM=linux/amd64 LOAD=true $0"
    fi
    echo ""
}

# ========================= 主流程 =========================
main() {
    # 解析帮助参数
    for arg in "$@"; do
        case $arg in
            -h|--help|help)
                print_help
                exit 0
                ;;
        esac
    done

    # 记录开始时间
    local start_time=$(date +%s)

    # 切换到项目目录
    cd "$PROJECT_DIR"

    # 构建完整镜像名称
    local full_image
    if [[ -n "$REGISTRY" ]]; then
        full_image="${REGISTRY}/${IMAGE_NAME}"
    else
        full_image="${IMAGE_NAME}"
    fi

    # 确定有效平台
    local effective_platforms
    if [[ -n "$PLATFORM" ]]; then
        effective_platforms="$PLATFORM"
    else
        effective_platforms="$PLATFORMS"
    fi

    # 打印头部和配置
    print_header
    print_config "$full_image" "$effective_platforms"

    # 验证配置
    validate_config
    validate_dockerfile
    validate_nextjs_config

    # 检测 Docker 环境
    local docker_env=$(detect_docker_environment)
    log_info "Docker 环境: ${docker_env}"

    # 确定构建模式
    local final_mode="$BUILD_MODE"
    if [[ "$BUILD_MODE" == "auto" ]]; then
        if check_buildx_available; then
            final_mode="buildx"
            log_info "自动选择 buildx 模式"
        else
            final_mode="local"
            log_warning "buildx 不可用，使用本地构建模式"
        fi
    fi

    # 执行构建
    case "$final_mode" in
        buildx)
            ensure_builder
            build_with_buildx "$full_image" "$effective_platforms"
            ;;
        local)
            build_with_docker "$full_image" "${PLATFORM:-$(detect_host_platform)}"
            ;;
        *)
            log_error "未知构建模式: ${final_mode}"
            exit 1
            ;;
    esac

    # 打印摘要
    print_summary "$full_image" "$start_time"
}

# 执行主流程
main "$@"

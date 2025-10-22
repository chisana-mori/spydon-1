#!/bin/bash

# Robusta Central Hub 部署脚本
# 支持Docker Compose和Kubernetes部署

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
ROOT_DIR=$(cd "${SCRIPT_DIR}/../.." && pwd)
COMPOSE_FILE="${ROOT_DIR}/infrastructure/docker/docker-compose.yml"

# 日志函数
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

ensure_docker_compose() {
    if ! docker compose version >/dev/null 2>&1; then
        log_error "docker compose 未安装或版本过低"
        exit 1
    fi
}

# 显示帮助信息
show_help() {
    cat << EOF
Robusta Central Hub 部署脚本

用法:
    $0 [选项] <命令>

命令:
    docker-dev      使用Docker Compose启动开发环境
    docker-prod     使用Docker Compose启动生产环境
    k8s-deploy      部署到Kubernetes集群
    k8s-delete      从Kubernetes集群删除
    build           构建Docker镜像
    test            运行测试
    clean           清理资源

选项:
    -h, --help      显示此帮助信息
    --with-holmes   包含HolmesGPT服务
    --skip-build    跳过镜像构建
    --namespace     Kubernetes命名空间 (默认: robusta-central-hub)

示例:
    $0 docker-dev                    # 启动开发环境
    $0 docker-prod --with-holmes     # 启动生产环境并包含HolmesGPT
    $0 k8s-deploy --namespace prod   # 部署到Kubernetes
    $0 build                         # 构建所有镜像
    $0 test                          # 运行测试

EOF
}

# 检查依赖
check_dependencies() {
    local deps=("$@")
    for dep in "${deps[@]}"; do
        if ! command -v "$dep" &> /dev/null; then
            log_error "依赖 $dep 未安装"
            exit 1
        fi
    done
}

# 构建Docker镜像
build_images() {
    log_info "构建Docker镜像..."
    
    # 构建后端镜像
    log_info "构建后端镜像..."
    docker build -t robusta-central-hub/backend:latest "${ROOT_DIR}/apps/backend"
    
    # 构建前端镜像
    log_info "构建前端镜像..."
    docker build -t robusta-central-hub/frontend:latest "${ROOT_DIR}/apps/frontend"
    
    log_success "镜像构建完成"
}

# Docker Compose部署
deploy_docker() {
    local env=$1
    local with_holmes=$2
    
    log_info "使用Docker Compose部署 ($env 环境)..."
    
    # 检查依赖
    check_dependencies "docker"
    ensure_docker_compose
    
    # 构建镜像（除非跳过）
    if [[ "$SKIP_BUILD" != "true" ]]; then
        build_images
    fi
    
    # 设置环境变量
    export COMPOSE_PROJECT_NAME=robusta-central-hub
    
    # 启动服务
    local compose_args=(-f "$COMPOSE_FILE")
    if [[ "$with_holmes" == "true" ]]; then
        compose_args+=("--profile" "holmes")
    fi
    
    if [[ "$env" == "prod" ]]; then
        # 生产环境配置
        export ENVIRONMENT=production
        docker compose "${compose_args[@]}" up -d
    else
        # 开发环境配置
        export ENVIRONMENT=development
        docker compose "${compose_args[@]}" up -d
    fi
    
    log_success "Docker Compose部署完成"
    log_info "访问地址:"
    log_info "  前端: http://localhost:3000"
    log_info "  后端API: http://localhost:8080"
    log_info "  MinIO控制台: http://localhost:9001"
    if [[ "$with_holmes" == "true" ]]; then
        log_info "  HolmesGPT: http://localhost:8081"
    fi
}

# Kubernetes部署
deploy_k8s() {
    local namespace=${K8S_NAMESPACE:-robusta-central-hub}
    
    log_info "部署到Kubernetes集群 (命名空间: $namespace)..."
    
    # 检查依赖
    check_dependencies "kubectl"
    
    # 检查集群连接
    if ! kubectl cluster-info &> /dev/null; then
        log_error "无法连接到Kubernetes集群"
        exit 1
    fi
    
    # 构建并推送镜像（除非跳过）
    if [[ "$SKIP_BUILD" != "true" ]]; then
        build_images
        # 这里需要根据实际情况推送到镜像仓库
        log_warning "请确保镜像已推送到可访问的镜像仓库"
    fi
    
    # 应用Kubernetes配置
    log_info "创建命名空间..."
    kubectl apply -f "${ROOT_DIR}/infrastructure/k8s/namespace.yaml"
    
    log_info "应用配置和密钥..."
    kubectl apply -f "${ROOT_DIR}/infrastructure/k8s/configmap.yaml"
    
    log_info "部署PostgreSQL..."
    kubectl apply -f "${ROOT_DIR}/infrastructure/k8s/postgres.yaml"
    
    log_info "部署后端服务..."
    kubectl apply -f "${ROOT_DIR}/infrastructure/k8s/backend.yaml"
    
    # 等待部署完成
    log_info "等待部署完成..."
    kubectl wait --for=condition=available --timeout=300s deployment/postgres -n "$namespace"
    kubectl wait --for=condition=available --timeout=300s deployment/robusta-backend -n "$namespace"
    
    log_success "Kubernetes部署完成"
    
    # 显示服务状态
    kubectl get pods -n "$namespace"
    kubectl get services -n "$namespace"
}

# 删除Kubernetes部署
delete_k8s() {
    local namespace=${K8S_NAMESPACE:-robusta-central-hub}
    
    log_info "从Kubernetes集群删除部署 (命名空间: $namespace)..."
    
    kubectl delete -f "${ROOT_DIR}/infrastructure/k8s/" --ignore-not-found=true
    
    log_success "Kubernetes部署已删除"
}

# 运行测试
run_tests() {
    log_info "运行测试..."
    
    # 后端测试
    log_info "运行后端测试..."
    pushd "${ROOT_DIR}/apps/backend" > /dev/null
    go test -v ./...
    popd > /dev/null
    
    # 前端测试
    log_info "运行前端测试..."
    pushd "${ROOT_DIR}/apps/frontend" > /dev/null
    npm test -- --watchAll=false
    popd > /dev/null
    
    log_success "所有测试通过"
}

# 清理资源
clean_resources() {
    log_info "清理资源..."
    
    # 停止Docker Compose
    ensure_docker_compose
    docker compose -f "$COMPOSE_FILE" down -v --remove-orphans 2>/dev/null || true
    
    # 删除镜像
    docker rmi robusta-central-hub/backend:latest 2>/dev/null || true
    docker rmi robusta-central-hub/frontend:latest 2>/dev/null || true
    
    # 清理未使用的Docker资源
    docker system prune -f
    
    log_success "资源清理完成"
}

# 解析命令行参数
WITH_HOLMES=false
SKIP_BUILD=false
K8S_NAMESPACE="robusta-central-hub"

while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            show_help
            exit 0
            ;;
        --with-holmes)
            WITH_HOLMES=true
            shift
            ;;
        --skip-build)
            SKIP_BUILD=true
            shift
            ;;
        --namespace)
            K8S_NAMESPACE="$2"
            shift 2
            ;;
        docker-dev)
            COMMAND="docker-dev"
            shift
            ;;
        docker-prod)
            COMMAND="docker-prod"
            shift
            ;;
        k8s-deploy)
            COMMAND="k8s-deploy"
            shift
            ;;
        k8s-delete)
            COMMAND="k8s-delete"
            shift
            ;;
        build)
            COMMAND="build"
            shift
            ;;
        test)
            COMMAND="test"
            shift
            ;;
        clean)
            COMMAND="clean"
            shift
            ;;
        *)
            log_error "未知参数: $1"
            show_help
            exit 1
            ;;
    esac
done

# 执行命令
case $COMMAND in
    docker-dev)
        deploy_docker "dev" "$WITH_HOLMES"
        ;;
    docker-prod)
        deploy_docker "prod" "$WITH_HOLMES"
        ;;
    k8s-deploy)
        deploy_k8s
        ;;
    k8s-delete)
        delete_k8s
        ;;
    build)
        build_images
        ;;
    test)
        run_tests
        ;;
    clean)
        clean_resources
        ;;
    *)
        log_error "请指定一个命令"
        show_help
        exit 1
        ;;
esac

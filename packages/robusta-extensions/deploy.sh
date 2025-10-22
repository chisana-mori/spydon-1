#!/bin/bash

# Robusta Central Hub Extensions 部署脚本

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
EXTENSIONS_DIR="$SCRIPT_DIR"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

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

# 显示帮助信息
show_help() {
    cat << EOF
Robusta Central Hub Extensions 部署脚本

用法: $0 [选项]

选项:
    -h, --help              显示此帮助信息
    -m, --method METHOD     部署方法: local|git|package (默认: local)
    -u, --url URL          Git仓库URL (当method=git时使用)
    -b, --branch BRANCH    Git分支 (默认: main)
    -n, --namespace NS     Robusta命名空间 (默认: robusta)
    -c, --config FILE      配置文件路径
    --dry-run              仅显示将要执行的命令，不实际执行

示例:
    $0 --method local                    # 本地部署
    $0 --method git --url https://...    # 从Git仓库部署
    $0 --config my-config.yaml          # 使用自定义配置
    $0 --dry-run                         # 预览部署命令

EOF
}

# 默认参数
METHOD="local"
GIT_URL=""
GIT_BRANCH="main"
NAMESPACE="robusta"
CONFIG_FILE=""
DRY_RUN=false

# 解析命令行参数
while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            show_help
            exit 0
            ;;
        -m|--method)
            METHOD="$2"
            shift 2
            ;;
        -u|--url)
            GIT_URL="$2"
            shift 2
            ;;
        -b|--branch)
            GIT_BRANCH="$2"
            shift 2
            ;;
        -n|--namespace)
            NAMESPACE="$2"
            shift 2
            ;;
        -c|--config)
            CONFIG_FILE="$2"
            shift 2
            ;;
        --dry-run)
            DRY_RUN=true
            shift
            ;;
        *)
            log_error "未知参数: $1"
            show_help
            exit 1
            ;;
    esac
done

# 执行命令（支持dry-run）
execute_command() {
    local cmd="$1"
    local description="$2"
    
    if [ "$DRY_RUN" = true ]; then
        log_info "[DRY-RUN] $description"
        echo "  命令: $cmd"
    else
        log_info "$description"
        eval "$cmd"
    fi
}

# 检查依赖
check_dependencies() {
    log_info "检查依赖..."
    
    local missing_deps=()
    
    if ! command -v kubectl &> /dev/null; then
        missing_deps+=("kubectl")
    fi
    
    if ! command -v robusta &> /dev/null; then
        missing_deps+=("robusta")
    fi
    
    if [ ${#missing_deps[@]} -ne 0 ]; then
        log_error "缺少依赖: ${missing_deps[*]}"
        log_error "请安装缺少的依赖后重试"
        exit 1
    fi
    
    log_success "依赖检查通过"
}

# 验证Robusta安装
verify_robusta() {
    log_info "验证Robusta安装..."
    
    if ! kubectl get namespace "$NAMESPACE" &> /dev/null; then
        log_error "Robusta命名空间 '$NAMESPACE' 不存在"
        log_error "请先安装Robusta或指定正确的命名空间"
        exit 1
    fi
    
    if ! kubectl get deployment robusta-runner -n "$NAMESPACE" &> /dev/null; then
        log_error "Robusta runner未找到"
        log_error "请确认Robusta已正确安装"
        exit 1
    fi
    
    log_success "Robusta安装验证通过"
}

# 本地部署
deploy_local() {
    log_info "使用本地方法部署扩展..."
    
    # 检查本地文件
    if [ ! -f "$EXTENSIONS_DIR/pyproject.toml" ]; then
        log_error "未找到pyproject.toml文件"
        exit 1
    fi
    
    # 推送到Robusta
    execute_command "robusta playbooks push '$EXTENSIONS_DIR'" "推送扩展到Robusta"
    
    log_success "本地部署完成"
}

# Git部署
deploy_git() {
    log_info "使用Git方法部署扩展..."
    
    if [ -z "$GIT_URL" ]; then
        log_error "Git URL未指定，请使用 --url 参数"
        exit 1
    fi
    
    # 创建临时配置
    local temp_config=$(mktemp)
    cat > "$temp_config" << EOF
playbookRepos:
  central_hub_extensions:
    url: "$GIT_URL"
    branch: "$GIT_BRANCH"
    pip_install: true
EOF
    
    execute_command "helm upgrade robusta robusta/robusta -n '$NAMESPACE' -f '$temp_config'" "更新Robusta配置"
    
    # 清理临时文件
    if [ "$DRY_RUN" = false ]; then
        rm -f "$temp_config"
    fi
    
    log_success "Git部署完成"
}

# 应用配置
apply_config() {
    if [ -n "$CONFIG_FILE" ]; then
        log_info "应用配置文件: $CONFIG_FILE"
        
        if [ ! -f "$CONFIG_FILE" ]; then
            log_error "配置文件不存在: $CONFIG_FILE"
            exit 1
        fi
        
        execute_command "helm upgrade robusta robusta/robusta -n '$NAMESPACE' -f '$CONFIG_FILE'" "应用配置"
        log_success "配置应用完成"
    fi
}

# 验证部署
verify_deployment() {
    log_info "验证部署..."
    
    if [ "$DRY_RUN" = true ]; then
        log_info "[DRY-RUN] 跳过部署验证"
        return
    fi
    
    # 等待Pod重启
    log_info "等待Robusta Pod重启..."
    kubectl rollout status deployment/robusta-runner -n "$NAMESPACE" --timeout=300s
    
    # 检查日志
    log_info "检查扩展加载状态..."
    sleep 10
    
    local logs=$(kubectl logs deployment/robusta-runner -n "$NAMESPACE" --tail=50)
    
    if echo "$logs" | grep -q "central_hub_extensions"; then
        log_success "扩展加载成功"
    else
        log_warning "未在日志中找到扩展加载信息，请手动检查"
    fi
    
    # 显示Pod状态
    kubectl get pods -n "$NAMESPACE" -l app.kubernetes.io/name=robusta
}

# 显示后续步骤
show_next_steps() {
    log_success "部署完成！"
    echo
    log_info "后续步骤:"
    echo "1. 检查Robusta日志:"
    echo "   kubectl logs -f deployment/robusta-runner -n $NAMESPACE"
    echo
    echo "2. 测试告警发送:"
    echo "   kubectl apply -f https://raw.githubusercontent.com/robusta-dev/kubernetes-demos/main/crashpod/broken.yaml"
    echo
    echo "3. 查看Central Hub界面确认告警接收"
    echo
    echo "4. 如需修改配置，编辑values.yaml并重新运行:"
    echo "   helm upgrade robusta robusta/robusta -n $NAMESPACE -f your-values.yaml"
}

# 主函数
main() {
    log_info "开始部署Robusta Central Hub Extensions"
    echo "部署方法: $METHOD"
    echo "命名空间: $NAMESPACE"
    echo
    
    # 检查依赖
    check_dependencies
    
    # 验证Robusta
    verify_robusta
    
    # 根据方法部署
    case $METHOD in
        local)
            deploy_local
            ;;
        git)
            deploy_git
            ;;
        *)
            log_error "不支持的部署方法: $METHOD"
            exit 1
            ;;
    esac
    
    # 应用配置
    apply_config
    
    # 验证部署
    verify_deployment
    
    # 显示后续步骤
    show_next_steps
}

# 运行主函数
main "$@"

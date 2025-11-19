#!/bin/bash

# GitOps 同步脚本
# 用于手动触发 GitOps 同步和状态检查

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
    echo -e "${BLUE}[GITOPS]${NC} $1"
}

# 默认配置
ARGOCD_SERVER="${ARGOCD_SERVER:-argocd-server.argocd.svc.cluster.local}"
ARGOCD_NAMESPACE="${ARGOCD_NAMESPACE:-argocd}"
ARGOCD_AUTH_TOKEN="${ARGOCD_AUTH_TOKEN:-}"
TARGET_ENVIRONMENT="${TARGET_ENVIRONMENT:-}"
DRY_RUN="${DRY_RUN:-false}"

# 检查依赖
check_deps() {
    log "Checking dependencies..."

    local tools=("kubectl" "argocd")
    for tool in "${tools[@]}"; do
        if ! command -v "$tool" &> /dev/null; then
            error "$tool is not installed"
        fi
    done

    # 检查 ArgoCD 连接
    if ! argocd cluster list &> /dev/null; then
        error "Cannot connect to ArgoCD server: $ARGOCD_SERVER"
    fi

    log "All dependencies are available"
}

# 登录 ArgoCD
login_argocd() {
    if [ -n "$ARGOCD_AUTH_TOKEN" ]; then
        log "Using provided auth token"
        argocd login "$ARGOCD_SERVER" --auth-token "$ARGOCD_AUTH_TOKEN" --insecure
        return
    fi

    # 尝试使用当前上下文
    if argocd account get-info &> /dev/null; then
        log "Already logged in to ArgoCD"
        return
    fi

    # 使用 SSO 登录
    if command -v argocd &> /dev/null; then
        log "Attempting SSO login to ArgoCD..."
        argocd login "$ARGOCD_SERVER" --sso --insecure
    else
        error "No valid authentication method found"
    fi
}

# 获取应用列表
get_applications() {
    local environment="${1:-}"
    local apps

    if [ -n "$environment" ]; then
        apps=$(argocd app list --output name | grep "spydon-$environment" || true)
    else
        apps=$(argocd app list --output name | grep "spydon-" || true)
    fi

    echo "$apps"
}

# 检查应用状态
check_app_status() {
    local app="$1"
    info "Checking status of $app..."

    local status
    status=$(argocd app get "$app" --output json | jq -r '.status.health.status')

    local sync_status
    sync_status=$(argocd app get "$app" --output json | jq -r '.status.sync.status')

    echo "  Health: $status"
    echo "  Sync: $sync_status"

    if [ "$status" != "Healthy" ] || [ "$sync_status" != "Synced" ]; then
        warn "Application $app is not in desired state"
        return 1
    fi

    log "Application $app is healthy and synced"
    return 0
}

# 同步应用
sync_application() {
    local app="$1"
    local dry_run="$2"

    info "Syncing application: $app"

    if [ "$dry_run" = "true" ]; then
        warn "DRY RUN: Would sync $app"
        return 0
    fi

    # 获取当前同步状态
    local current_status
    current_status=$(argocd app get "$app" --output json | jq -r '.status.sync.status')

    if [ "$current_status" = "Synced" ]; then
        info "Application $app is already synced"
        return 0
    fi

    # 执行同步
    argocd app sync "$app" --force --retry 3

    # 等待同步完成
    local timeout=600
    local interval=10
    local elapsed=0

    while [ $elapsed -lt $timeout ]; do
        local sync_status
        sync_status=$(argocd app get "$app" --output json | jq -r '.status.sync.status')

        if [ "$sync_status" = "Synced" ]; then
            log "Application $app synced successfully"
            return 0
        elif [ "$sync_status" = "OutOfSync" ]; then
            warn "Application $app is still out of sync"
        elif [ "$sync_status" = "Failed" ]; then
            error "Application $app sync failed"
            return 1
        fi

        sleep $interval
        elapsed=$((elapsed + interval))
        info "Waiting for sync... ($elapsed/${timeout}s)"
    done

    error "Timeout waiting for $app to sync"
    return 1
}

# 回滚应用
rollback_application() {
    local app="$1"
    local dry_run="$2"

    info "Rolling back application: $app"

    if [ "$dry_run" = "true" ]; then
        warn "DRY RUN: Would rollback $app"
        return 0
    fi

    # 获取可用的历史版本
    local revisions
    revisions=$(argocd app history "$app" --output json | jq -r '.items[1].revision' 2>/dev/null || echo "")

    if [ -z "$revisions" ]; then
        error "No previous revision found for $app"
        return 1
    fi

    info "Rolling back to revision: $revisions"

    # 执行回滚
    argocd app rollback "$app" "$revisions"

    # 等待回滚完成
    sync_application "$app" "$dry_run"
}

# 显示应用详情
show_app_details() {
    local app="$1"

    info "Showing details for: $app"
    argocd app get "$app" --output yaml | less
}

# 部署新版本
deploy_version() {
    local environment="$1"
    local version="$2"
    local dry_run="$3"

    info "Deploying version $version to $environment environment"

    if [ "$dry_run" = "true" ]; then
        warn "DRY RUN: Would deploy version $version to $environment"
        return 0
    fi

    # 更新 GitOps 仓库中的镜像标签
    local gitops_repo="${GITOPS_REPO:-https://github.com/your-org/spydon-gitops.git}"
    local gitops_dir="${GITOPS_DIR:-/tmp/spydon-gitops}"

    log "Cloning GitOps repository..."
    git clone "$gitops_repo" "$gitops_dir"
    cd "$gitops_dir"

    # 更新环境配置
    local env_dir="environments/$environment"
    if [ ! -d "$env_dir" ]; then
        error "Environment directory not found: $env_dir"
        return 1
    fi

    # 更新 kustomization.yaml 中的镜像标签
    sed -i "s|newTag:.*|newTag: $version|g" "$env_dir/kustomization.yaml"

    # 提交和推送更改
    git config user.email "ci@spydon.com"
    git config user.name "CI Bot"
    git add "$env_dir/kustomization.yaml"
    git commit -m "Deploy version $version to $environment"
    git push

    log "GitOps repository updated, waiting for ArgoCD sync..."

    cd - > /dev/null
    rm -rf "$gitops_dir"

    # 等待 ArgoCD 自动同步
    local apps
    apps=$(get_applications "$environment")

    for app in $apps; do
        sync_application "$app" "$dry_run"
    done
}

# 清理临时资源
cleanup() {
    log "Cleaning up..."
    rm -rf /tmp/spydon-gitops
}

# 显示帮助信息
show_help() {
    cat << EOF
Usage: $0 [OPTIONS] [COMMAND]

Commands:
  status [env]         Show application status for all or specific environment
  sync [env]           Sync applications for all or specific environment
  rollback [env]       Rollback applications for all or specific environment
  deploy <env> <ver>   Deploy new version to environment
  details <app>        Show detailed information about an application
  list                 List all Spydon applications

Options:
  -n, --namespace NAMESPACE    ArgoCD namespace (default: argocd)
  -s, --server SERVER          ArgoCD server address
  -t, --token TOKEN            ArgoCD auth token
  -d, --dry-run                Perform dry run (no changes)
  -h, --help                   Show this help message

Environment Variables:
  ARGOCD_SERVER                ArgoCD server address
  ARGOCD_NAMESPACE             ArgoCD namespace
  ARGOCD_AUTH_TOKEN            ArgoCD authentication token
  GITOPS_REPO                  GitOps repository URL
  GITOPS_DIR                   Temporary GitOps directory

Examples:
  $0 status                    Show status of all environments
  $0 status production         Show production environment status
  $0 sync staging              Sync staging environment
  $0 deploy production v1.2.3  Deploy v1.2.3 to production
  $0 rollback production       Rollback production environment
  $0 --dry-run deploy staging v1.2.3  Dry run deployment
EOF
}

# 主函数
main() {
    local command="${1:-}"
    local environment="${2:-}"
    local version="${3:-}"

    # 设置错误处理
    trap cleanup EXIT

    # 检查依赖
    check_deps

    # 登录 ArgoCD
    login_argocd

    case "$command" in
        status)
            local apps
            apps=$(get_applications "$environment")

            for app in $apps; do
                check_app_status "$app" || true
                echo
            done
            ;;
        sync)
            local apps
            apps=$(get_applications "$environment")

            for app in $apps; do
                sync_application "$app" "$DRY_RUN"
                echo
            done
            ;;
        rollback)
            local apps
            apps=$(get_applications "$environment")

            for app in $apps; do
                rollback_application "$app" "$DRY_RUN"
                echo
            done
            ;;
        deploy)
            if [ -z "$environment" ] || [ -z "$version" ]; then
                error "deploy command requires environment and version"
                exit 1
            fi
            deploy_version "$environment" "$version" "$DRY_RUN"
            ;;
        details)
            if [ -z "$environment" ]; then
                error "details command requires application name"
                exit 1
            fi
            show_app_details "$environment"
            ;;
        list)
            get_applications
            ;;
        -h|--help|"")
            show_help
            ;;
        *)
            error "Unknown command: $command"
            show_help
            exit 1
            ;;
    esac
}

# 解析命令行参数
while [[ $# -gt 0 ]]; do
    case $1 in
        -n|--namespace)
            ARGOCD_NAMESPACE="$2"
            shift 2
            ;;
        -s|--server)
            ARGOCD_SERVER="$2"
            shift 2
            ;;
        -t|--token)
            ARGOCD_AUTH_TOKEN="$2"
            shift 2
            ;;
        -d|--dry-run)
            DRY_RUN=true
            shift
            ;;
        *)
            break
            ;;
    esac
done

# 执行主函数
main "$@"

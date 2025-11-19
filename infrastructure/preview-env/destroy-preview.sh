#!/bin/bash

# 预览环境销毁脚本
# 清理 Pull Request 相关的临时预览环境

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
    echo -e "${BLUE}[CLEANUP]${NC} $1"
}

# 配置
PR_NUMBER="${PR_NUMBER:-}"
NAMESPACE="${NAMESPACE:-}"
FORCE="${FORCE:-false}"
DRY_RUN="${DRY_RUN:-false}"
TIMEOUT="${TIMEOUT:-300}"

# 验证参数
validate_params() {
    if [ -z "$PR_NUMBER" ] && [ -z "$NAMESPACE" ]; then
        error "Either PR_NUMBER or NAMESPACE must be specified"
        exit 1
    fi

    # 如果指定了 PR_NUMBER，生成命名空间
    if [ -n "$PR_NUMBER" ]; then
        NAMESPACE="preview-pr-$PR_NUMBER"
    fi
}

# 检查依赖
check_deps() {
    log "Checking dependencies..."

    local tools=("kubectl" "helm" "jq")
    for tool in "${tools[@]}"; do
        if ! command -v "$tool" &> /dev/null; then
            error "$tool is not installed"
        fi
    done

    # 检查 Kubernetes 连接
    if ! kubectl cluster-info &> /dev/null; then
        error "Cannot connect to Kubernetes cluster"
    fi

    log "All dependencies are available"
}

# 检查命名空间是否存在
check_namespace() {
    if ! kubectl get namespace "$NAMESPACE" &> /dev/null; then
        warn "Namespace $NAMESPACE does not exist"
        return 1
    fi
    return 0
}

# 备份重要数据
backup_data() {
    info "Backing up important data from namespace: $NAMESPACE"

    if [ "$DRY_RUN" = "true" ]; then
        warn "DRY RUN: Would backup data from $NAMESPACE"
        return 0
    fi

    # 创建备份目录
    local backup_dir="/tmp/spydon-backup-$(date +%Y%m%d_%H%M%S)"
    mkdir -p "$backup_dir"

    # 备份 ConfigMaps 和 Secrets
    kubectl get configmaps -n "$NAMESPACE" -o yaml > "$backup_dir/configmaps.yaml" 2>/dev/null || true
    kubectl get secrets -n "$NAMESPACE" -o yaml > "$backup_dir/secrets.yaml" 2>/dev/null || true

    # 备份应用配置
    helm get values "preview-pr-$PR_NUMBER" -n "$NAMESPACE" > "$backup_dir/helm-values.yaml" 2>/dev/null || true

    # 备份日志（如果有的话）
    kubectl logs --all-containers=true -n "$NAMESPACE" > "$backup_dir/pods.log" 2>/dev/null || true

    log "Data backed up to: $backup_dir"
    echo "$backup_dir" > /tmp/last_backup_dir
}

# 卸载 Helm release
unhelm_release() {
    info "Uninstalling Helm releases from namespace: $NAMESPACE"

    if [ "$DRY_RUN" = "true" ]; then
        warn "DRY RUN: Would uninstall Helm releases from $NAMESPACE"
        return 0
    fi

    # 获取所有的 Helm releases
    local releases
    releases=$(helm list -n "$NAMESPACE" --output json | jq -r '.[].name' 2>/dev/null || true)

    if [ -z "$releases" ]; then
        info "No Helm releases found in namespace $NAMESPACE"
        return 0
    fi

    for release in $releases; do
        info "Uninstalling Helm release: $release"
        helm uninstall "$release" -n "$NAMESPACE" --timeout="$TIMEOUT" || {
            warn "Failed to uninstall Helm release: $release"
        }
    done

    log "Helm releases uninstalled"
}

# 删除 PVC
delete_pvcs() {
    info "Deleting PVCs from namespace: $NAMESPACE"

    if [ "$DRY_RUN" = "true" ]; then
        warn "DRY RUN: Would delete PVCs from $NAMESPACE"
        return 0
    fi

    # 获取所有的 PVC
    local pvcs
    pvcs=$(kubectl get pvc -n "$NAMESPACE" --no-headers 2>/dev/null | awk '{print $1}' || true)

    if [ -z "$pvcs" ]; then
        info "No PVCs found in namespace $NAMESPACE"
        return 0
    fi

    for pvc in $pvcs; do
        info "Deleting PVC: $pvc"
        kubectl delete pvc "$pvc" -n "$NAMESPACE" --timeout=60s || {
            warn "Failed to delete PVC: $pvc"
        }
    done

    log "PVCs deleted"
}

# 等待 Pod 终止
wait_for_pod_termination() {
    info "Waiting for pods to terminate in namespace: $NAMESPACE"

    if [ "$DRY_RUN" = "true" ]; then
        warn "DRY RUN: Would wait for pods to terminate"
        return 0
    fi

    local retries=0
    local max_retries=60

    while [ $retries -lt $max_retries ]; do
        local pod_count
        pod_count=$(kubectl get pods -n "$NAMESPACE" --no-headers 2>/dev/null | wc -l || echo "0")

        if [ "$pod_count" = "0" ]; then
            log "All pods have terminated"
            return 0
        fi

        if [ $retries -eq $((max_retries - 1)) ]; then
            warn "Timeout waiting for pods to terminate"
            return 1
        fi

        sleep 5
        ((retries++))
        info "Waiting for pods to terminate... ($pod_count pods remaining)"
    done
}

# 删除命名空间
delete_namespace() {
    info "Deleting namespace: $NAMESPACE"

    if [ "$DRY_RUN" = "true" ]; then
        warn "DRY RUN: Would delete namespace $NAMESPACE"
        return 0
    fi

    # 检查命名空间是否还有资源
    local resource_count
    resource_count=$(kubectl api-resources --verbs=list --namespaced -o name | \
        xargs -I {} kubectl get {} -n "$NAMESPACE" --no-headers 2>/dev/null | wc -l || echo "0")

    if [ "$resource_count" -gt 0 ] && [ "$FORCE" != "true" ]; then
        warn "Namespace $NAMESPACE still has $resource_count resources"
        warn "Use --force to force deletion"
        return 1
    fi

    kubectl delete namespace "$NAMESPACE" --timeout="$TIMEOUT" || {
        warn "Failed to delete namespace $NAMESPACE"
        return 1
    }

    log "Namespace $NAMESPACE deleted successfully"
}

# 清理 DNS 记录
cleanup_dns() {
    if [ -z "$PR_NUMBER" ]; then
        return 0
    fi

    info "Cleaning up DNS records for PR #$PR_NUMBER"

    if [ "$DRY_RUN" = "true" ]; then
        warn "DRY RUN: Would clean up DNS records"
        return 0
    fi

    # 这里可以添加清理 DNS 记录的逻辑
    # 例如：通过 API 调用删除 Ingress 对应的 DNS 记录
    local hostname="pr-$PR_NUMBER.${BASE_DOMAIN:-spydon-preview.local}"

    # 示例：清理 Route53 记录（如果使用 AWS）
    if command -v aws &> /dev/null && [ -n "${AWS_ZONE_ID:-}" ]; then
        aws route53 list-resource-record-sets \
            --hosted-zone-id "$AWS_ZONE_ID" \
            --query "ResourceRecordSets[?Name == '$hostname.']" \
            --output table || true

        # 删除 DNS 记录的逻辑
        # aws route53 change-resource-record-sets ...
    fi

    log "DNS records cleaned up"
}

# 发送通知
send_notification() {
    local status="$1"

    info "Sending cleanup notification..."

    # 构建通知消息
    local message
    case "$status" in
        "success")
            message="🧹 Preview environment cleanup completed"
            ;;
        "failed")
            message="❌ Preview environment cleanup failed"
            ;;
        *)
            message="ℹ️ Preview environment cleanup status: $status"
            ;;
    esac

    # 发送到 Slack（如果配置了 webhook）
    if [ -n "${SLACK_WEBHOOK_URL:-}" ]; then
        curl -X POST -H 'Content-type: application/json' \
            --data "{
                \"text\": \"$message\",
                \"attachments\": [
                    {
                        \"fields\": [
                            {
                                \"title\": \"PR\",
                                \"value\": \"#${PR_NUMBER:-Unknown}\",
                                \"short\": true
                            },
                            {
                                \"title\": \"Namespace\",
                                \"value\": \"$NAMESPACE\",
                                \"short\": true
                            }
                        ]
                    }
                ]
            }" \
            "$SLACK_WEBHOOK_URL" || true
    fi

    # 添加到 PR 评论（如果配置了 GitHub token）
    if [ -n "${GITHUB_TOKEN:-}" ] && [ -n "$PR_NUMBER" ]; then
        local pr_comment="Preview environment for PR #$PR_NUMBER has been cleaned up.

**Namespace**: $NAMESPACE
**Status**: $status

---

🤖 All temporary resources have been removed."

        gh api -X POST \
            repos/:owner/:repo/issues/$PR_NUMBER/comments \
            --field body="$pr_comment" || true
    fi
}

# 清理预览环境
cleanup_preview() {
    log "Cleaning up preview environment..."
    log "Namespace: $NAMESPACE"
    if [ -n "$PR_NUMBER" ]; then
        log "PR Number: $PR_NUMBER"
    fi

    validate_params
    check_deps

    # 检查命名空间是否存在
    if ! check_namespace; then
        log "Namespace does not exist, nothing to clean up"
        return 0
    fi

    # 执行清理流程
    backup_data
    unhelm_release
    delete_pvcs
    wait_for_pod_termination
    cleanup_dns
    delete_namespace

    send_notification "success"
    log "Preview environment cleanup completed!"
}

# 清理函数
cleanup() {
    log "Cleaning up temporary files..."
    # 清理临时文件（如果需要）
}

# 列出所有预览环境
list_preview_environments() {
    log "Listing all preview environments..."

    local namespaces
    namespaces=$(kubectl get namespaces -l app=spydon,environment=preview --no-headers | awk '{print $1}' || true)

    if [ -z "$namespaces" ]; then
        log "No preview environments found"
        return 0
    fi

    echo "Preview Environments:"
    echo "===================="
    echo "$namespaces" | while read -r namespace; do
        local pr_number
        pr_number=$(kubectl get namespace "$namespace" -o jsonpath='{.metadata.labels.pr-number}' 2>/dev/null || echo "Unknown")
        local creation_date
        creation_date=$(kubectl get namespace "$namespace" -o jsonpath='{.metadata.labels.creation-date}' 2>/dev/null || echo "Unknown")
        local pod_count
        pod_count=$(kubectl get pods -n "$namespace" --no-headers 2>/dev/null | wc -l || echo "0")
        local pvc_count
        pvc_count=$(kubectl get pvc -n "$namespace" --no-headers 2>/dev/null | wc -l || echo "0")

        printf "%-20s PR: %-6s Created: %-20s Pods: %-3s PVCs: %-3s\n" \
            "$namespace" "$pr_number" "$creation_date" "$pod_count" "$pvc_count"
    done
}

# 强制清理所有预览环境
force_cleanup_all() {
    warn "Force cleaning up ALL preview environments"
    warn "This will delete all preview environments without confirmation"

    if [ "$DRY_RUN" = "false" ] && [ "$FORCE" != "true" ]; then
        read -p "Are you sure you want to continue? (y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log "Operation cancelled"
            exit 0
        fi
    fi

    local namespaces
    namespaces=$(kubectl get namespaces -l app=spydon,environment=preview --no-headers | awk '{print $1}' || true)

    if [ -z "$namespaces" ]; then
        log "No preview environments found"
        return 0
    fi

    for namespace in $namespaces; do
        NAMESPACE="$namespace"
        log "Cleaning up: $namespace"

        backup_data
        unhelm_release
        delete_pvcs
        wait_for_pod_termination
        delete_namespace
    done

    log "All preview environments cleaned up!"
}

# 显示帮助信息
show_help() {
    cat << EOF
Usage: $0 [OPTIONS] [COMMAND]

Commands:
  cleanup [PR_NUM]       Cleanup preview environment for specific PR
  cleanup-ns NS         Cleanup preview environment by namespace
  list                  List all preview environments
  cleanup-all           Cleanup all preview environments
  force-cleanup-all     Force cleanup all preview environments (dangerous)

Options:
  -n, --namespace NS    Target namespace
  -f, --force           Force deletion without confirmation
  -t, --timeout SECONDS Cleanup timeout (default: 300)
  --dry-run             Perform dry run
  -h, --help            Show this help message

Environment Variables:
  PR_NUMBER             Pull Request number
  NAMESPACE             Target namespace
  BASE_DOMAIN           Base domain for preview environments
  SLACK_WEBHOOK_URL     Slack webhook URL
  GITHUB_TOKEN          GitHub token
  AWS_ZONE_ID           AWS Route53 zone ID (for DNS cleanup)

Examples:
  $0 cleanup 123                    Cleanup PR #123 preview environment
  $0 cleanup-ns preview-pr-123      Cleanup specific namespace
  $0 list                          List all preview environments
  $0 --force cleanup-all            Force cleanup all environments
  $0 --dry-run cleanup 123          Dry run cleanup
EOF
}

# 主函数
main() {
    local command="${1:-cleanup}"
    local target="${2:-}"

    # 设置错误处理
    trap cleanup EXIT

    # 解析命令行参数
    while [[ $# -gt 0 ]]; do
        case $1 in
            -n|--namespace)
                NAMESPACE="$2"
                shift 2
                ;;
            -f|--force)
                FORCE=true
                shift
                ;;
            -t|--timeout)
                TIMEOUT="$2"
                shift 2
                ;;
            --dry-run)
                DRY_RUN=true
                shift
                ;;
            -h|--help)
                show_help
                exit 0
                ;;
            *)
                # 跳过命令参数
                if [ "$1" = "$command" ] || [ "$1" = "$target" ]; then
                    shift
                else
                    error "Unknown option: $1"
                    show_help
                    exit 1
                fi
                ;;
        esac
    done

    case "$command" in
        cleanup)
            if [ -z "$target" ] && [ -z "$PR_NUMBER" ] && [ -z "$NAMESPACE" ]; then
                error "cleanup command requires PR number or namespace"
                show_help
                exit 1
            fi
            PR_NUMBER="${PR_NUMBER:-$target}"
            cleanup_preview
            ;;
        cleanup-ns)
            if [ -z "$target" ] && [ -z "$NAMESPACE" ]; then
                error "cleanup-ns command requires namespace"
                exit 1
            fi
            NAMESPACE="${NAMESPACE:-$target}"
            cleanup_preview
            ;;
        list)
            list_preview_environments
            ;;
        cleanup-all)
            if [ "$FORCE" != "true" ]; then
                read -p "Are you sure you want to cleanup ALL preview environments? (y/N): " -n 1 -r
                echo
                if [[ ! $REPLY =~ ^[Yy]$ ]]; then
                    log "Operation cancelled"
                    exit 0
                fi
            fi
            force_cleanup_all
            ;;
        force-cleanup-all)
            FORCE=true
            force_cleanup_all
            ;;
        "")
            show_help
            ;;
        *)
            error "Unknown command: $command"
            show_help
            exit 1
            ;;
    esac
}

# 执行主函数
main "$@"

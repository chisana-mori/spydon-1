#!/bin/bash

# 预览环境创建脚本
# 为 Pull Request 创建临时预览环境

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
    echo -e "${BLUE}[PREVIEW]${NC} $1"
}

# 配置
PR_NUMBER="${PR_NUMBER:-}"
COMMIT_SHA="${COMMIT_SHA:-}"
BACKEND_IMAGE="${BACKEND_IMAGE:-}"
FRONTEND_IMAGE="${FRONTEND_IMAGE:-}"
BASE_DOMAIN="${BASE_DOMAIN:-spydon-preview.local}"
GIT_REPO="${GIT_REPO:-$(git config --get remote.origin.url)}"
DRY_RUN="${DRY_RUN:-false}"
TIMEOUT="${TIMEOUT:-600}"

# 验证必需参数
validate_params() {
    if [ -z "$PR_NUMBER" ]; then
        error "PR_NUMBER is required"
        exit 1
    fi

    if [ -z "$COMMIT_SHA" ]; then
        error "COMMIT_SHA is required"
        exit 1
    fi

    if [ -z "$BACKEND_IMAGE" ]; then
        error "BACKEND_IMAGE is required"
        exit 1
    fi

    if [ -z "$FRONTEND_IMAGE" ]; then
        error "FRONTEND_IMAGE is required"
        exit 1
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

# 创建命名空间
create_namespace() {
    local namespace="preview-pr-$PR_NUMBER"
    info "Creating namespace: $namespace"

    if [ "$DRY_RUN" = "true" ]; then
        warn "DRY RUN: Would create namespace $namespace"
        return 0
    fi

    kubectl create namespace "$namespace" --dry-run=client -o yaml | kubectl apply -f -

    # 添加标签
    kubectl label namespace "$namespace" \
        app=spydon \
        environment=preview \
        pr-number="$PR_NUMBER" \
        created-by="preview-env" \
        creation-date="$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
        --overwrite

    log "Namespace $namespace created successfully"
}

# 创建证书
create_certificates() {
    local namespace="preview-pr-$PR_NUMBER"
    local hostname="pr-$PR_NUMBER.$BASE_DOMAIN"
    info "Creating certificates for: $hostname"

    if [ "$DRY_RUN" = "true" ]; then
        warn "DRY RUN: Would create certificates for $hostname"
        return 0
    fi

    # 创建自签名证书（生产环境使用 cert-manager）
    cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Secret
metadata:
  name: preview-tls
  namespace: $namespace
type: kubernetes.io/tls
data:
  tls.crt: $(openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
    -keyout /tmp/key.pem -out /tmp/cert.pem \
    -subj "/CN=$hostname" 2>/dev/null && \
    cat /tmp/cert.pem | base64 -w 0)
  tls.key: $(cat /tmp/key.pem | base64 -w 0)
EOF

    # 清理临时文件
    rm -f /tmp/key.pem /tmp/cert.pem

    log "Certificates created for $hostname"
}

# 部署应用
deploy_application() {
    local namespace="preview-pr-$PR_NUMBER"
    local hostname="pr-$PR_NUMBER.$BASE_DOMAIN"

    info "Deploying application to namespace: $namespace"

    if [ "$DRY_RUN" = "true" ]; then
        warn "DRY RUN: Would deploy application"
        return 0
    fi

    # 使用 Helm 部署
    helm upgrade --install "preview-pr-$PR_NUMBER" \
        ../../helm/spydon \
        --namespace "$namespace" \
        --set environment=preview \
        --set backend.image.repository="$BACKEND_IMAGE" \
        --set backend.image.tag="$COMMIT_SHA" \
        --set frontend.image.repository="$FRONTEND_IMAGE" \
        --set frontend.image.tag="$COMMIT_SHA" \
        --set ingress.host="$hostname" \
        --set backend.replicas=1 \
        --set frontend.replicas=1 \
        --set database.enabled=true \
        --set redis.enabled=true \
        --set minio.enabled=false \
        --set monitoring.enabled=false \
        --set backup.enabled=false \
        --wait \
        --timeout="$TIMEOUT" \
        --values - <<EOF
environment: preview
replicaCount: 1

backend:
  image:
    repository: $BACKEND_IMAGE
    tag: $COMMIT_SHA
  resources:
    requests:
      memory: "256Mi"
      cpu: "250m"
    limits:
      memory: "512Mi"
      cpu: "500m"

frontend:
  image:
    repository: $FRONTEND_IMAGE
    tag: $COMMIT_SHA
  resources:
    requests:
      memory: "128Mi"
      cpu: "100m"
    limits:
      memory: "256Mi"
      cpu: "200m"

database:
  enabled: true
  auth:
    database: spydon_preview_$PR_NUMBER
  primary:
    resources:
      requests:
        memory: "256Mi"
        cpu: "100m"
      limits:
        memory: "512Mi"
        cpu: "200m"
    persistence:
      size: 5Gi

redis:
  enabled: true
  auth:
    enabled: false
  master:
    resources:
      requests:
        memory: "64Mi"
        cpu: "50m"
      limits:
        memory: "128Mi"
        cpu: "100m"

ingress:
  enabled: true
  className: nginx
  host: $hostname
  tls:
    enabled: true
    secretName: preview-tls

monitoring:
  enabled: false

backup:
  enabled: false

autoscaling:
  enabled: false
EOF

    log "Application deployed successfully"
}

# 等待部署完成
wait_for_deployment() {
    local namespace="preview-pr-$PR_NUMBER"
    local hostname="pr-$PR_NUMBER.$BASE_DOMAIN"

    info "Waiting for deployment to be ready..."

    if [ "$DRY_RUN" = "true" ]; then
        warn "DRY RUN: Would wait for deployment"
        return 0
    fi

    # 等待 Deployment 就绪
    kubectl wait --for=condition=available --timeout=300s \
        deployment/robusta-backend -n "$namespace"

    kubectl wait --for=condition=available --timeout=300s \
        deployment/robusta-frontend -n "$namespace"

    # 等待数据库就绪
    kubectl wait --for=condition=ready --timeout=180s \
        pod -l app.kubernetes.io/name=postgresql -n "$namespace"

    # 等待 Ingress 生效
    info "Waiting for Ingress to be ready..."
    local retries=0
    local max_retries=60

    while [ $retries -lt $max_retries ]; do
        if curl -f -k "https://$hostname/health" &> /dev/null; then
            log "Preview environment is ready!"
            break
        fi

        if [ $retries -eq $((max_retries - 1)) ]; then
            error "Timeout waiting for preview environment to be ready"
            return 1
        fi

        sleep 10
        ((retries++))
        info "Waiting for Ingress... ($retries/$max_retries)"
    done
}

# 运行健康检查
run_health_checks() {
    local namespace="preview-pr-$PR_NUMBER"
    local hostname="pr-$PR_NUMBER.$BASE_DOMAIN"

    info "Running health checks..."

    if [ "$DRY_RUN" = "true" ]; then
        warn "DRY RUN: Would run health checks"
        return 0
    fi

    # 检查 API 健康状态
    local api_health
    api_health=$(curl -s -k -f "https://$hostname/api/v1/health" || echo "failed")

    if [ "$api_health" != "failed" ]; then
        log "API health check passed"
    else
        warn "API health check failed"
    fi

    # 检查前端可访问性
    if curl -s -k -f "https://$hostname/" &> /dev/null; then
        log "Frontend health check passed"
    else
        warn "Frontend health check failed"
    fi

    # 检查 Pod 状态
    local unhealthy_pods
    unhealthy_pods=$(kubectl get pods -n "$namespace" --no-headers | grep -v "Running\|Completed" || true)

    if [ -z "$unhealthy_pods" ]; then
        log "All pods are healthy"
    else
        warn "Found unhealthy pods:"
        echo "$unhealthy_pods"
    fi
}

# 发送通知
send_notification() {
    local namespace="preview-pr-$PR_NUMBER"
    local hostname="pr-$PR_NUMBER.$BASE_DOMAIN"
    local status="$1"

    info "Sending notification..."

    # 构建通知消息
    local message
    case "$status" in
        "created")
            message="✅ Preview environment created successfully"
            ;;
        "failed")
            message="❌ Preview environment creation failed"
            ;;
        *)
            message="ℹ️ Preview environment status: $status"
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
                                \"value\": \"#$PR_NUMBER\",
                                \"short\": true
                            },
                            {
                                \"title\": \"Environment\",
                                \"value\": \"$namespace\",
                                \"short\": true
                            },
                            {
                                \"title\": \"URL\",
                                \"value\": \"https://$hostname\",
                                \"short\": false
                            }
                        ]
                    }
                ]
            }" \
            "$SLACK_WEBHOOK_URL" || true
    fi

    # 添加到 PR 评论（如果配置了 GitHub token）
    if [ -n "${GITHUB_TOKEN:-}" ]; then
        local pr_comment="Preview environment for PR #$PR_NUMBER

**Status**: $status
**Environment**: $namespace
**URL**: https://$hostname
**Commit**: $COMMIT_SHA

---

🤖 This preview environment will be automatically cleaned up when the PR is closed."

        gh api -X POST \
            repos/:owner/:repo/issues/$PR_NUMBER/comments \
            --field body="$pr_comment" || true
    fi
}

# 创建预览环境
create_preview() {
    log "Creating preview environment for PR #$PR_NUMBER"
    log "Commit: $COMMIT_SHA"
    log "Backend Image: $BACKEND_IMAGE"
    log "Frontend Image: $FRONTEND_IMAGE"

    validate_params
    check_deps

    # 创建资源
    create_namespace
    create_certificates
    deploy_application

    # 等待和验证
    if [ "$DRY_RUN" = "false" ]; then
        wait_for_deployment
        run_health_checks
        send_notification "created"
    else
        send_notification "dry-run"
    fi

    log "Preview environment creation completed!"
    log "Environment URL: https://pr-$PR_NUMBER.$BASE_DOMAIN"
}

# 清理函数
cleanup() {
    log "Cleaning up temporary files..."
    rm -f /tmp/key.pem /tmp/cert.pem
}

# 显示帮助信息
show_help() {
    cat << EOF
Usage: $0 [OPTIONS]

Required Options:
  -p, --pr-number NUM      Pull Request number
  -c, --commit SHA         Commit SHA
  -b, --backend-image IMAGE Backend Docker image
  -f, --frontend-image IMAGE Frontend Docker image

Optional Options:
  -d, --domain DOMAIN      Base domain for preview environments
  -t, --timeout SECONDS    Deployment timeout (default: 600)
  -r, --repo URL           Git repository URL
  --dry-run                Perform dry run
  -h, --help               Show this help message

Environment Variables:
  PR_NUMBER                Pull Request number
  COMMIT_SHA               Commit SHA
  BACKEND_IMAGE            Backend Docker image
  FRONTEND_IMAGE           Frontend Docker image
  BASE_DOMAIN              Base domain for preview environments
  GIT_REPO                 Git repository URL
  SLACK_WEBHOOK_URL        Slack webhook URL
  GITHUB_TOKEN             GitHub token

Examples:
  $0 -p 123 -c abc123def -b ghcr.io/org/spydon/backend -f ghcr.io/org/spydon/frontend
  $0 --dry-run -p 123 -c abc123def -b backend:latest -f frontend:latest
EOF
}

# 主函数
main() {
    # 设置错误处理
    trap cleanup EXIT

    # 解析命令行参数
    while [[ $# -gt 0 ]]; do
        case $1 in
            -p|--pr-number)
                PR_NUMBER="$2"
                shift 2
                ;;
            -c|--commit)
                COMMIT_SHA="$2"
                shift 2
                ;;
            -b|--backend-image)
                BACKEND_IMAGE="$2"
                shift 2
                ;;
            -f|--frontend-image)
                FRONTEND_IMAGE="$2"
                shift 2
                ;;
            -d|--domain)
                BASE_DOMAIN="$2"
                shift 2
                ;;
            -t|--timeout)
                TIMEOUT="$2"
                shift 2
                ;;
            -r|--repo)
                GIT_REPO="$2"
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
                error "Unknown option: $1"
                show_help
                exit 1
                ;;
        esac
    done

    # 执行创建流程
    create_preview
}

# 执行主函数
main "$@"

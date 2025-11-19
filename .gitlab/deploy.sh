#!/bin/bash

# GitLab CI 部署脚本
# 适配 GitLab CI 环境的自动化部署

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
    echo -e "${BLUE}[DEPLOY]${NC} $1"
}

# GitLab CI 变量
CI_PROJECT_ID="${CI_PROJECT_ID:-}"
CI_PIPELINE_ID="${CI_PIPELINE_ID:-}"
CI_JOB_ID="${CI_JOB_ID:-}"
CI_COMMIT_SHA="${CI_COMMIT_SHA:-}"
CI_COMMIT_REF_SLUG="${CI_COMMIT_REF_SLUG:-}"
CI_ENVIRONMENT_NAME="${CI_ENVIRONMENT_NAME:-}"
CI_ENVIRONMENT_URL="${CI_ENVIRONMENT_URL:-}"
CI_MERGE_REQUEST_IID="${CI_MERGE_REQUEST_IID:-}"

# 部署配置
DEPLOY_ENV="${DEPLOY_ENV:-}"
VERSION="${VERSION:-$CI_COMMIT_SHA}"
STRATEGY="${STRATEGY:-}"  # preview, staging, production
DRY_RUN="${DRY_RUN:-false}"

# GitOps 配置
GITOPS_REPO="${GITOPS_REPO:-}"
GITOPS_TOKEN="${GITOPS_TOKEN:-}"
GITOPS_BRANCH="${GITOPS_BRANCH:-main}"

# K8s 配置
KUBECONFIG_DATA="${KUBECONFIG_DATA:-}"
NAMESPACE_PREFIX="${NAMESPACE_PREFIX:-spydon}"

# 检查部署环境
check_deploy_env() {
    log "Checking deployment environment..."

    if [ -z "$CI_PROJECT_ID" ]; then
        warn "Not running in GitLab CI, using local environment"
    else
        log "Running in GitLab CI (Project: $CI_PROJECT_ID, Pipeline: $CI_PIPELINE_ID)"
        log "Environment: ${CI_ENVIRONMENT_NAME:-not set}"
        log "URL: ${CI_ENVIRONMENT_URL:-not set}"
    fi

    # 检查必需工具
    local tools=("kubectl" "helm" "git" "curl" "jq")
    for tool in "${tools[@]}"; do
        if ! command -v "$tool" &> /dev/null; then
            error "$tool is not installed"
            exit 1
        fi
    done

    log "Deployment environment verified"
}

# 设置 Kubernetes 配置
setup_k8s_config() {
    log "Setting up Kubernetes configuration..."

    if [ -n "$KUBECONFIG_DATA" ]; then
        log "Using provided kubeconfig data..."
        echo "$KUBECONFIG_DATA" | base64 -d > kubeconfig
        export KUBECONFIG=kubeconfig
    elif [ -n "${KUBECONFIG:-}" ]; then
        log "Using existing kubeconfig: $KUBECONFIG"
    else
        error "Kubeconfig not provided"
        exit 1
    fi

    # 验证集群连接
    if ! kubectl cluster-info &> /dev/null; then
        error "Cannot connect to Kubernetes cluster"
        exit 1
    fi

    log "Kubernetes configuration ready"
}

# 设置 GitOps 仓库
setup_gitops() {
    log "Setting up GitOps repository..."

    if [ -z "$GITOPS_REPO" ]; then
        warn "GitOps repository not configured, skipping GitOps setup"
        return 0
    fi

    local gitops_dir="gitops-$(date +%s)"
    mkdir -p "$gitops_dir"
    cd "$gitops_dir"

    # 克隆 GitOps 仓库
    log "Cloning GitOps repository: $GITOPS_REPO"
    git clone "$GITOPS_REPO" .
    git checkout "$GITOPS_BRANCH"

    log "GitOps repository ready at: $(pwd)"
}

# 部署预览环境
deploy_preview() {
    log "Deploying preview environment..."

    if [ -z "$CI_MERGE_REQUEST_IID" ]; then
        error "CI_MERGE_REQUEST_IID is required for preview deployment"
        exit 1
    fi

    local namespace="${NAMESPACE_PREFIX}-preview-pr-$CI_MERGE_REQUEST_IID"
    local backend_image="$CI_REGISTRY_IMAGE/backend"
    local frontend_image="$CI_REGISTRY_IMAGE/frontend"

    log "Preview environment configuration:"
    log "  Namespace: $namespace"
    log "  Backend: $backend_image:$VERSION"
    log "  Frontend: $frontend_image:$VERSION"

    if [ "$DRY_RUN" = "true" ]; then
        warn "DRY RUN: Would deploy preview environment"
        return 0
    fi

    # 使用预览环境部署脚本
    if [ -f "../infrastructure/preview-env/create-preview.sh" ]; then
        ../infrastructure/preview-env/create-preview.sh \
            --pr-number "$CI_MERGE_REQUEST_IID" \
            --commit "$VERSION" \
            --backend-image "$backend_image" \
            --frontend-image "$frontend_image" \
            --domain "${PREVIEW_DOMAIN:-spydon-preview.local}"
    else
        error "Preview deployment script not found"
        exit 1
    fi

    # 更新 GitLab CI 环境 URL
    if [ -n "$PREVIEW_DOMAIN" ]; then
        local preview_url="https://pr-$CI_MERGE_REQUEST_IID.$PREVIEW_DOMAIN"
        log "Preview environment URL: $preview_url"
        echo "CI_ENVIRONMENT_URL=$preview_url" >> deploy.env
    fi

    log "Preview environment deployed successfully"
}

# 更新 GitOps 配置
update_gitops_config() {
    if [ ! -d "environments/$DEPLOY_ENV" ]; then
        error "Environment directory not found: environments/$DEPLOY_ENV"
        exit 1
    fi

    log "Updating GitOps configuration for $DEPLOY_ENV environment..."

    cd "environments/$DEPLOY_ENV"

    # 更新 kustomization.yaml
    if [ -f "kustomization.yaml" ]; then
        log "Updating kustomization.yaml..."
        sed -i "s|newTag: .*|newTag: $VERSION|g" kustomization.yaml

        # 更新镜像仓库
        sed -i "s|ghcr.io/your-org/spydon|$CI_REGISTRY_IMAGE|g" kustomization.yaml
    fi

    # 更新配置文件
    local config_file="config.yaml"
    if [ ! -f "$config_file" ]; then
        config_file="values.yaml"
    fi

    if [ -f "$config_file" ]; then
        log "Updating $config_file..."
        # 更新版本信息
        if command -v yq &> /dev/null; then
            yq eval ".version = \"$VERSION\"" -i "$config_file"
            yq eval ".commitSha = \"$CI_COMMIT_SHA\"" -i "$config_file"
            yq eval ".buildTime = \"$(date -u +%Y-%m-%dT%H:%M:%SZ)\"" -i "$config_file"
        else
            # 使用 sed 进行简单替换
            sed -i "s/version: .*/version: $VERSION/g" "$config_file"
            sed -i "s/commitSha: .*/commitSha: $CI_COMMIT_SHA/g" "$config_file"
        fi
    fi

    # 提交更改
    log "Committing GitOps changes..."
    git config user.email "ci@gitlab.com"
    git config user.name "GitLab CI"

    git add .
    git diff --staged --quiet || git commit -m "Deploy $VERSION to $DEPLOY_ENV

- Version: $VERSION
- Commit: $CI_COMMIT_SHA
- Pipeline: $CI_PIPELINE_URL
- Environment: $CI_ENVIRONMENT_NAME"

    # 推送更改
    if [ -n "$GITOPS_TOKEN" ]; then
        log "Pushing to GitOps repository..."
        git remote set-url origin "https://gitlab-ci-token:$GITOPS_TOKEN@$(echo "$GITOPS_REPO" | sed 's|https://||')"
        git push origin "$GITOPS_BRANCH"
    else
        warn "GitOps token not provided, changes committed but not pushed"
    fi

    cd - > /dev/null
    log "GitOps configuration updated"
}

# 部署到 Staging 环境
deploy_staging() {
    log "Deploying to staging environment..."

    DEPLOY_ENV="staging"
    NAMESPACE="${NAMESPACE_PREFIX}-staging"

    if [ -n "$GITOPS_REPO" ]; then
        setup_gitops
        update_gitops_config
    else
        # 直接使用 GitOps 同步脚本
        if [ -f "infrastructure/gitops/scripts/sync.sh" ]; then
            log "Using GitOps sync script..."
            ./infrastructure/gitops/scripts/sync.sh \
                deploy staging "$VERSION"
        else
            error "GitOps sync script not found"
            exit 1
        fi
    fi

    # 等待部署完成
    log "Waiting for staging deployment to complete..."
    if [ -n "${STAGING_URL:-}" ]; then
        timeout 600 bash -c 'until curl -f "$STAGING_URL/health"; do sleep 10; done'
    fi

    log "Staging environment deployed successfully"
}

# 部署到生产环境
deploy_production() {
    log "Deploying to production environment..."

    DEPLOY_ENV="production"
    NAMESPACE="${NAMESPACE_PREFIX}-production"

    # 生产部署需要额外确认
    if [ "$DRY_RUN" != "true" ] && [ -z "${CI_COMMIT_TAG:-}" ] && [ "$CI_COMMIT_BRANCH" != "main" ]; then
        error "Production deployment requires tag or main branch"
        exit 1
    fi

    log "Production deployment configuration:"
    log "  Namespace: $NAMESPACE"
    log "  Version: $VERSION"
    log "  Strategy: Blue-Green"

    if [ -n "$GITOPS_REPO" ]; then
        setup_gitops
        update_gitops_config
    else
        # 直接使用 GitOps 同步脚本
        if [ -f "infrastructure/gitops/scripts/sync.sh" ]; then
            log "Using GitOps sync script..."
            ./infrastructure/gitops/scripts/sync.sh \
                deploy production "$VERSION"
        else
            error "GitOps sync script not found"
            exit 1
        fi
    fi

    # 等待蓝环境就绪
    if [ -n "${PRODUCTION_GREEN_URL:-}" ]; then
        log "Waiting for green environment to be ready..."
        timeout 900 bash -c 'until curl -f "$PRODUCTION_GREEN_URL/health"; do sleep 15; done'

        # 切换流量
        log "Switching traffic to green environment..."
        # 这里需要实现流量切换逻辑

        # 验证生产环境
        log "Verifying production deployment..."
        if [ -n "${PRODUCTION_URL:-}" ]; then
            timeout 300 bash -c 'until curl -f "$PRODUCTION_URL/health"; do sleep 10; done'
        fi
    fi

    log "Production environment deployed successfully"
}

# 发送部署通知
send_deployment_notification() {
    log "Sending deployment notification..."

    local status="$1"
    local message

    case "$status" in
        "started")
            message="🚀 Deployment started for $DEPLOY_ENV environment"
            ;;
        "success")
            message="✅ Deployment completed successfully for $DEPLOY_ENV environment"
            ;;
        "failed")
            message="❌ Deployment failed for $DEPLOY_ENV environment"
            ;;
        *)
            message="ℹ️ Deployment status for $DEPLOY_ENV: $status"
            ;;
    esac

    # 发送到 Slack
    if [ -n "${SLACK_WEBHOOK_URL:-}" ]; then
        curl -X POST -H 'Content-type: application/json' \
            --data "{
                \"text\": \"$message\",
                \"attachments\": [
                    {
                        \"fields\": [
                            {\"title\": \"Environment\", \"value\": \"$DEPLOY_ENV\", \"short\": true},
                            {\"title\": \"Version\", \"value\": \"$VERSION\", \"short\": true},
                            {\"title\": \"Pipeline\", \"value\": \"$CI_PIPELINE_URL\", \"short\": false}
                        ]
                    }
                ]
            }" \
            "$SLACK_WEBHOOK_URL" || true
    fi

    # 发送 Teams
    if [ -n "${TEAMS_WEBHOOK_URL:-}" ]; then
        curl -X POST -H 'Content-Type: application/json' \
            --data "{
                \"@type\": \"MessageCard\",
                \"@context\": \"https://schema.org/extensions\",
                \"themeColor\": \"$([ \"$status\" = \"failed\" ] && echo \"FF0000\" || echo \"00FF00\")\",
                \"summary\": \"$message\",
                \"sections\": [{
                    \"activityTitle\": \"$message\",
                    \"facts\": [
                        {\"name\": \"Environment\", \"value\": \"$DEPLOY_ENV\"},
                        {\"name\": \"Version\", \"value\": \"$VERSION\"},
                        {\"name\": \"Pipeline\", \"value\": \"$CI_PIPELINE_URL\"}
                    ]
                }]
            }" \
            "$TEAMS_WEBHOOK_URL" || true
    fi

    log "Deployment notification sent"
}

# 验证部署
verify_deployment() {
    log "Verifying deployment..."

    local namespace="${NAMESPACE_PREFIX}-${DEPLOY_ENV}"

    # 检查命名空间
    if ! kubectl get namespace "$namespace" &> /dev/null; then
        error "Namespace $namespace not found"
        return 1
    fi

    # 检查部署状态
    local deployments=("robusta-backend" "robusta-frontend")
    for deployment in "${deployments[@]}"; do
        local ready_replicas
        ready_replicas=$(kubectl get deployment "$deployment" -n "$namespace" -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo "0")

        local replicas
        replicas=$(kubectl get deployment "$deployment" -n "$namespace" -o jsonpath='{.spec.replicas}' 2>/dev/null || echo "0")

        if [ "$ready_replicas" = "$replicas" ] && [ "$replicas" -gt 0 ]; then
            log "✅ Deployment $deployment is ready ($ready_replicas/$replicas replicas)"
        else
            warn "⚠️ Deployment $deployment is not ready ($ready_replicas/$replicas replicas)"
            return 1
        fi
    done

    log "Deployment verification completed"
}

# 清理函数
cleanup() {
    log "Cleaning up..."
    # 清理临时目录
    rm -rf gitops-*
    rm -f kubeconfig
}

# 显示帮助信息
show_help() {
    cat << EOF
Usage: $0 [OPTIONS] [ENVIRONMENT]

GitLab CI Deployment Script

ENVIRONMENT:
  preview        Deploy preview environment
  staging        Deploy to staging
  production     Deploy to production

OPTIONS:
  --dry-run      Perform dry run deployment
  --version VER  Deploy specific version
  --strategy STR Deployment strategy
  --namespace NS Target namespace prefix
  --help         Show this help message

Environment Variables:
  CI_PROJECT_ID        GitLab project ID
  CI_PIPELINE_ID       GitLab pipeline ID
  CI_ENVIRONMENT_NAME  GitLab environment name
  KUBECONFIG_DATA      Kubernetes config (base64 encoded)
  GITOPS_REPO          GitOps repository URL
  GITOPS_TOKEN         GitOps repository token
  SLACK_WEBHOOK_URL    Slack webhook URL

Examples:
  $0 preview                      Deploy preview environment
  $0 staging                      Deploy to staging
  $0 --dry-run production        Dry run production deployment
  $0 --version v1.2.3 staging    Deploy specific version to staging
EOF
}

# 主函数
main() {
    local env_type="${1:-}"

    # 解析命令行参数
    while [[ $# -gt 0 ]]; do
        case $1 in
            --dry-run)
                DRY_RUN=true
                shift
                ;;
            --version)
                VERSION="$2"
                shift 2
                ;;
            --strategy)
                STRATEGY="$2"
                shift 2
                ;;
            --namespace)
                NAMESPACE_PREFIX="$2"
                shift 2
                ;;
            --help)
                show_help
                exit 0
                ;;
            preview|staging|production)
                env_type="$1"
                shift
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

    if [ -z "$env_type" ]; then
        error "Environment type is required"
        show_help
        exit 1
    fi

    log "Starting GitLab CI deployment..."
    log "Environment: $env_type"
    log "Version: $VERSION"
    log "Strategy: $STRATEGY"

    # 检查环境
    check_deploy_env
    setup_k8s_config

    # 发送部署开始通知
    send_deployment_notification "started"

    # 执行部署
    case "$env_type" in
        preview)
            deploy_preview
            ;;
        staging)
            deploy_staging
            ;;
        production)
            deploy_production
            ;;
        *)
            error "Unknown environment type: $env_type"
            exit 1
            ;;
    esac

    # 验证部署
    if [ "$DRY_RUN" != "true" ]; then
        verify_deployment
    fi

    # 发送成功通知
    send_deployment_notification "success"

    log "Deployment completed successfully!"
}

# 执行主函数
main "$@"

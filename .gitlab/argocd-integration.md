# ArgoCD 集成指南

## 架构说明

### CI/CD 职责分离

```
┌─────────────────┐      ┌──────────────────┐      ┌─────────────────┐
│   GitLab CI     │      │  GitOps Repo     │      │    ArgoCD       │
│                 │      │                  │      │                 │
│ • 代码检查      │──────▶│ • Manifests      │◀─────│ • 监控变更      │
│ • 单元测试      │ Push │ • Kustomize      │ Watch│ • 自动同步      │
│ • 构建镜像      │      │ • Helm Charts    │      │ • 健康检查      │
│ • 安全扫描      │      │ • Version Info   │      │ • 回滚管理      │
└─────────────────┘      └──────────────────┘      └─────────────────┘
                                                             │
                                                             ▼
                                                    ┌─────────────────┐
                                                    │   Kubernetes    │
                                                    │                 │
                                                    │ • Staging       │
                                                    │ • Production    │
                                                    └─────────────────┘
```

### 工作流程

1. **开发者推送代码** → GitLab
2. **GitLab CI 执行**：
   - 代码质量检查
   - 单元测试
   - 安全扫描
   - 构建 Docker 镜像
   - 推送镜像到 Registry
3. **更新 GitOps 仓库**：
   - 修改 Kubernetes manifests
   - 更新镜像标签
   - 提交版本信息
4. **ArgoCD 自动检测**：
   - 监控 GitOps 仓库变更
   - 对比期望状态与实际状态
   - 自动同步到 Kubernetes
5. **部署完成**：
   - 健康检查
   - 发送通知

## ArgoCD 应用配置

### 1. Staging 环境

```yaml
# argocd/applications/spydon-staging.yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: spydon-staging
  namespace: argocd
  finalizers:
    - resources-finalizer.argocd.argoproj.io
spec:
  project: spydon

  source:
    repoURL: https://gitlab.com/your-org/spydon-gitops.git
    targetRevision: main
    path: environments/staging

    # 使用 Kustomize
    kustomize:
      version: v5.0.0
      commonLabels:
        app.kubernetes.io/managed-by: argocd
        app.kubernetes.io/part-of: spydon
        environment: staging

  destination:
    server: https://kubernetes.default.svc
    namespace: spydon-staging

  syncPolicy:
    automated:
      prune: true      # 自动删除不再需要的资源
      selfHeal: true   # 自动修复配置漂移
      allowEmpty: false

    syncOptions:
      - CreateNamespace=true
      - PrunePropagationPolicy=foreground
      - PruneLast=true

    retry:
      limit: 5
      backoff:
        duration: 5s
        factor: 2
        maxDuration: 3m

  revisionHistoryLimit: 10

  # 健康检查
  ignoreDifferences:
    - group: apps
      kind: Deployment
      jsonPointers:
        - /spec/replicas  # 忽略 HPA 管理的副本数
```

### 2. Production 环境

```yaml
# argocd/applications/spydon-production.yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: spydon-production
  namespace: argocd
  finalizers:
    - resources-finalizer.argocd.argoproj.io
spec:
  project: spydon

  source:
    repoURL: https://gitlab.com/your-org/spydon-gitops.git
    targetRevision: main
    path: environments/production

    kustomize:
      version: v5.0.0
      commonLabels:
        app.kubernetes.io/managed-by: argocd
        app.kubernetes.io/part-of: spydon
        environment: production

  destination:
    server: https://kubernetes.default.svc
    namespace: spydon-production

  syncPolicy:
    # 生产环境：手动同步，需要审批
    automated: null

    syncOptions:
      - CreateNamespace=true
      - PrunePropagationPolicy=foreground
      - PruneLast=true

    retry:
      limit: 3
      backoff:
        duration: 10s
        factor: 2
        maxDuration: 5m

  revisionHistoryLimit: 20
```

### 3. ArgoCD Project

```yaml
# argocd/projects/spydon.yaml
apiVersion: argoproj.io/v1alpha1
kind: AppProject
metadata:
  name: spydon
  namespace: argocd
spec:
  description: Spydon Alert Management Platform

  # 允许的源仓库
  sourceRepos:
    - https://gitlab.com/your-org/spydon-gitops.git

  # 允许的目标集群和命名空间
  destinations:
    - namespace: 'spydon-*'
      server: https://kubernetes.default.svc

  # 允许的资源类型
  clusterResourceWhitelist:
    - group: ''
      kind: Namespace
    - group: ''
      kind: PersistentVolume
    - group: 'rbac.authorization.k8s.io'
      kind: ClusterRole
    - group: 'rbac.authorization.k8s.io'
      kind: ClusterRoleBinding

  namespaceResourceWhitelist:
    - group: '*'
      kind: '*'

  # RBAC 策略
  roles:
    - name: developer
      description: Developers can view and sync
      policies:
        - p, proj:spydon:developer, applications, get, spydon/*, allow
        - p, proj:spydon:developer, applications, sync, spydon/spydon-staging, allow
      groups:
        - spydon-developers

    - name: operator
      description: Operators can manage all environments
      policies:
        - p, proj:spydon:operator, applications, *, spydon/*, allow
      groups:
        - spydon-operators
```

## 部署 ArgoCD 应用

### 方法 1: 使用 kubectl

```bash
# 应用 Project 配置
kubectl apply -f argocd/projects/spydon.yaml

# 应用 Application 配置
kubectl apply -f argocd/applications/spydon-staging.yaml
kubectl apply -f argocd/applications/spydon-production.yaml
```

### 方法 2: 使用 ArgoCD CLI

```bash
# 登录 ArgoCD
argocd login argocd.example.com

# 创建 Project
argocd proj create spydon \
  --description "Spydon Alert Management Platform" \
  --src https://gitlab.com/your-org/spydon-gitops.git \
  --dest https://kubernetes.default.svc,spydon-*

# 创建 Staging Application
argocd app create spydon-staging \
  --project spydon \
  --repo https://gitlab.com/your-org/spydon-gitops.git \
  --path environments/staging \
  --dest-server https://kubernetes.default.svc \
  --dest-namespace spydon-staging \
  --sync-policy automated \
  --auto-prune \
  --self-heal

# 创建 Production Application
argocd app create spydon-production \
  --project spydon \
  --repo https://gitlab.com/your-org/spydon-gitops.git \
  --path environments/production \
  --dest-server https://kubernetes.default.svc \
  --dest-namespace spydon-production
```

## GitOps 仓库结构

```
spydon-gitops/
├── README.md
├── environments/
│   ├── base/                    # 基础配置
│   │   ├── backend/
│   │   │   ├── deployment.yaml
│   │   │   ├── service.yaml
│   │   │   └── kustomization.yaml
│   │   └── frontend/
│   │       ├── deployment.yaml
│   │       ├── service.yaml
│   │       └── kustomization.yaml
│   │
│   ├── development/             # 开发环境
│   │   ├── backend/
│   │   │   └── kustomization.yaml
│   │   ├── frontend/
│   │   │   └── kustomization.yaml
│   │   └── version.yaml
│   │
│   ├── staging/                 # 预生产环境
│   │   ├── backend/
│   │   │   ├── deployment.yaml  # 覆盖配置
│   │   │   └── kustomization.yaml
│   │   ├── frontend/
│   │   │   ├── deployment.yaml
│   │   │   └── kustomization.yaml
│   │   ├── ingress.yaml
│   │   └── version.yaml
│   │
│   └── production/              # 生产环境
│       ├── backend/
│       │   ├── deployment.yaml
│       │   ├── hpa.yaml
│       │   └── kustomization.yaml
│       ├── frontend/
│       │   ├── deployment.yaml
│       │   ├── hpa.yaml
│       │   └── kustomization.yaml
│       ├── ingress.yaml
│       └── version.yaml
│
└── argocd/                      # ArgoCD 配置
    ├── projects/
    │   └── spydon.yaml
    └── applications/
        ├── spydon-staging.yaml
        └── spydon-production.yaml
```

## 监控和通知

### ArgoCD 通知配置

```yaml
# argocd-notifications-cm.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: argocd-notifications-cm
  namespace: argocd
data:
  # Slack 通知模板
  template.app-deployed: |
    message: |
      ✅ Application {{.app.metadata.name}} deployed successfully
      Version: {{.app.status.sync.revision}}
      Environment: {{.app.metadata.labels.environment}}

  template.app-health-degraded: |
    message: |
      ⚠️ Application {{.app.metadata.name}} health degraded
      Status: {{.app.status.health.status}}

  template.app-sync-failed: |
    message: |
      ❌ Application {{.app.metadata.name}} sync failed
      Error: {{.app.status.operationState.message}}

  # Slack 服务配置
  service.slack: |
    token: $slack-token

  # 订阅配置
  subscriptions: |
    - recipients:
      - slack:spydon-deployments
      triggers:
      - on-deployed
      - on-health-degraded
      - on-sync-failed
```

## 常用操作

### 查看应用状态

```bash
# 查看所有应用
argocd app list

# 查看特定应用
argocd app get spydon-staging

# 查看同步历史
argocd app history spydon-staging
```

### 手动同步

```bash
# 同步应用
argocd app sync spydon-staging

# 强制同步（忽略差异）
argocd app sync spydon-staging --force

# 同步特定资源
argocd app sync spydon-staging --resource apps:Deployment:backend
```

### 回滚

```bash
# 查看历史版本
argocd app history spydon-production

# 回滚到特定版本
argocd app rollback spydon-production 5
```

### 暂停/恢复自动同步

```bash
# 暂停自动同步
argocd app set spydon-staging --sync-policy none

# 恢复自动同步
argocd app set spydon-staging --sync-policy automated
```

## 生产环境部署流程

### 1. Staging 自动部署

```bash
# GitLab CI 自动触发
main 分支 → 构建镜像 → 更新 GitOps → ArgoCD 自动同步
```

### 2. Production 手动审批

```bash
# 1. 在 ArgoCD UI 中查看 staging 部署状态
argocd app get spydon-staging

# 2. 确认无问题后，更新 production manifests
cd spydon-gitops
git checkout main
cp environments/staging/version.yaml environments/production/version.yaml
sed -i 's/staging/production/g' environments/production/version.yaml
git commit -m "Promote staging to production"
git push

# 3. 在 ArgoCD UI 中手动同步 production
argocd app sync spydon-production

# 或使用 CLI
argocd app sync spydon-production --prune
```

## 故障排查

### 同步失败

```bash
# 查看详细错误
argocd app get spydon-staging --show-operation

# 查看资源差异
argocd app diff spydon-staging

# 查看事件
kubectl get events -n spydon-staging --sort-by='.lastTimestamp'
```

### 健康检查失败

```bash
# 查看应用健康状态
argocd app get spydon-staging --show-health

# 查看 Pod 状态
kubectl get pods -n spydon-staging

# 查看 Pod 日志
kubectl logs -n spydon-staging deployment/backend
```

## 最佳实践

1. **环境隔离**：每个环境使用独立的 namespace 和 ArgoCD Application
2. **自动化分级**：
   - Development: 完全自动化
   - Staging: 自动同步 + 自我修复
   - Production: 手动审批
3. **版本追踪**：在 GitOps 仓库中记录详细的版本信息
4. **回滚策略**：保留足够的历史版本用于快速回滚
5. **监控告警**：配置 ArgoCD 通知，及时发现部署问题
6. **权限管理**：使用 RBAC 控制不同团队的访问权限

## 安全建议

1. **密钥管理**：使用 Sealed Secrets 或 External Secrets Operator
2. **镜像签名**：验证镜像签名，确保镜像来源可信
3. **网络策略**：配置 NetworkPolicy 限制 Pod 间通信
4. **审计日志**：启用 ArgoCD 审计日志，记录所有操作
5. **最小权限**：ArgoCD 使用最小权限的 ServiceAccount

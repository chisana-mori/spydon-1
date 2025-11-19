# Spydon Helm Chart 部署指南

## 概述

Spydon 项目采用 Helm Chart 作为标准交付物，通过 GitLab CI/CD 自动构建和发布。

## 架构设计

### CI/CD 流程

```
代码提交 → 质量检查 → 测试 → 构建镜像 → 打包 Chart → 发布 Chart
                                              ↓
                                         GitLab Package Registry
                                              ↓
                                         ArgoCD 自动同步
                                              ↓
                                         Kubernetes 集群
```

### Chart 结构

```
spydon/
├── Chart.yaml              # Chart 元数据
├── values.yaml             # 默认配置
├── templates/
│   ├── backend-deployment.yaml
│   ├── backend-service.yaml
│   ├── backend-config.yaml      # 后端 config.yaml ConfigMap
│   ├── frontend-deployment.yaml
│   ├── frontend-service.yaml
│   ├── frontend-config.yaml     # 前端 runtime.json ConfigMap
│   ├── ingress.yaml
│   └── ...
└── charts/                 # 依赖 Charts (PostgreSQL, Redis, MinIO)
```

## 配置文件管理

### 后端配置 (config.yaml)

**位置**: `apps/backend/config/config.yaml`

**Chart 中的处理**:
1. 配置模板化到 `templates/backend-config.yaml`
2. 通过 `values.yaml` 中的 `backend.config` 部分配置
3. 作为 ConfigMap 挂载到 Pod 的 `/app/config/config.yaml`

**示例**:
```yaml
# values.yaml
backend:
  config:
    databaseUrl: "postgres://..."
    jwtSecret: "your-secret"
    holmesGpt:
      url: "http://holmes-gpt:8081"
      apiKey: "your-key"
```

### 前端配置 (runtime.json)

**位置**: `apps/frontend/public/runtime.json`

**Chart 中的处理**:
1. 配置模板化到 `templates/frontend-config.yaml`
2. 通过 `values.yaml` 中的 `frontend.config` 部分配置
3. 作为 ConfigMap 挂载到 Pod 的 `/app/public/runtime.json`

**示例**:
```yaml
# values.yaml
frontend:
  config:
    apiUrl: "https://spydon.example.com/api"
    wsUrl: "wss://spydon.example.com/ws"
    minioEndpoint: "minio.example.com"
```

## GitLab CI/CD 配置

### Pipeline 阶段

1. **validate** - 代码质量检查
   - pre-commit hooks
   - golangci-lint
   - ESLint + TypeScript

2. **test** - 单元测试
   - Go 后端测试
   - React 前端测试
   - 覆盖率报告

3. **security** - 安全扫描
   - Trivy 漏洞扫描
   - npm audit

4. **build** - 构建镜像
   - 多架构构建 (amd64, arm64)
   - 推送到 Container Registry
   - 镜像安全扫描

5. **publish** - 发布 Chart
   - 打包 Helm Chart
   - 更新镜像标签
   - 发布到 Package Registry

### 关键 Job: package-chart

```yaml
package-chart:
  stage: publish
  image: alpine/helm:3.13.0
  script:
    # 1. 设置版本号
    - export CHART_VERSION="${CI_COMMIT_TAG:-0.1.0-${CI_COMMIT_SHORT_SHA}}"
    - export APP_VERSION="${CI_COMMIT_SHA}"

    # 2. 创建配置模板
    - 生成 backend-config.yaml
    - 生成 frontend-config.yaml

    # 3. 更新 values.yaml
    - 更新镜像标签
    - 添加配置默认值

    # 4. 打包 Chart
    - helm package spydon
    - helm repo index .

  artifacts:
    paths:
      - helm-package/*.tgz
      - helm-package/index.yaml
```

## 部署流程

### 1. 开发环境部署

```bash
# 添加 Helm 仓库
helm repo add spydon https://gitlab.com/api/v4/projects/PROJECT_ID/packages/helm/stable

# 创建配置文件
cat > values-dev.yaml <<EOF
global:
  environment: development

backend:
  config:
    databaseUrl: "postgres://dev:dev@postgres:5432/spydon_dev"
    jwtSecret: "dev-jwt-secret"
    # ... 其他配置
EOF

# 安装
helm install spydon-dev spydon/spydon \
  --namespace development \
  --create-namespace \
  --values values-dev.yaml
```

### 2. Staging 环境部署

```bash
# 使用特定版本
helm install spydon-staging spydon/spydon \
  --version 0.1.0-abc123 \
  --namespace staging \
  --create-namespace \
  --values values-staging.yaml
```

### 3. Production 环境部署

```bash
# 使用 Secrets 管理敏感信息
kubectl create secret generic spydon-secrets \
  --from-literal=jwt-secret=xxx \
  --from-literal=database-password=xxx \
  --namespace production

# 部署
helm install spydon spydon/spydon \
  --version 0.1.0 \
  --namespace production \
  --create-namespace \
  --values values-production.yaml \
  --wait \
  --timeout 10m
```

## ArgoCD 集成

### Application 配置

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: spydon-production
  namespace: argocd
spec:
  project: default

  source:
    # Helm Chart 来源
    repoURL: https://gitlab.com/api/v4/projects/PROJECT_ID/packages/helm/stable
    chart: spydon
    targetRevision: 0.1.0

    helm:
      # 使用 Git 仓库中的 values 文件
      valueFiles:
        - $values/infrastructure/k8s/helm/values-production.yaml

  # 额外的 values 仓库
  sources:
    - repoURL: https://gitlab.com/your-org/spydon-config.git
      targetRevision: main
      ref: values

  destination:
    server: https://kubernetes.default.svc
    namespace: production

  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
```

### 自动同步流程

1. **代码提交** → GitLab CI 构建新版本
2. **Chart 发布** → 推送到 Package Registry
3. **ArgoCD 检测** → 发现新版本
4. **自动同步** → 部署到 Kubernetes
5. **健康检查** → 验证部署状态

## 配置管理最佳实践

### 1. 使用 Secrets

```bash
# 创建 Secret
kubectl create secret generic spydon-secrets \
  --from-literal=jwt-secret=$(openssl rand -base64 32) \
  --from-literal=hmac-secret=$(openssl rand -base64 32) \
  --from-literal=database-password=$(openssl rand -base64 32) \
  --namespace production

# 在 values.yaml 中引用
backend:
  existingSecret: spydon-secrets
  config:
    jwtSecret: ""  # 从 Secret 读取
```

### 2. 环境分离

```
infrastructure/k8s/helm/
├── values-development.yaml    # 开发环境
├── values-staging.yaml        # 预发布环境
└── values-production.yaml     # 生产环境
```

### 3. 配置验证

```bash
# 渲染模板检查
helm template spydon spydon/spydon \
  --values values-production.yaml \
  --debug

# 验证配置
helm lint spydon/spydon \
  --values values-production.yaml
```

## 故障排查

### 查看 Chart 信息

```bash
# 列出已安装的 Release
helm list -n production

# 查看 Release 详情
helm get all spydon -n production

# 查看 values
helm get values spydon -n production
```

### 查看配置

```bash
# 查看后端配置
kubectl get configmap spydon-backend-config -n production -o yaml

# 查看前端配置
kubectl get configmap spydon-frontend-config -n production -o yaml

# 进入 Pod 验证
kubectl exec -it deployment/spydon-backend -n production -- \
  cat /app/config/config.yaml
```

### 回滚

```bash
# 查看历史
helm history spydon -n production

# 回滚到上一个版本
helm rollback spydon -n production

# 回滚到指定版本
helm rollback spydon 3 -n production
```

## 升级策略

### 滚动升级

```bash
helm upgrade spydon spydon/spydon \
  --version 0.2.0 \
  --namespace production \
  --values values-production.yaml \
  --wait \
  --timeout 10m
```

### 蓝绿部署

```bash
# 部署新版本到新命名空间
helm install spydon-blue spydon/spydon \
  --version 0.2.0 \
  --namespace production-blue \
  --values values-production.yaml

# 验证后切换流量
kubectl patch ingress spydon -n production \
  -p '{"spec":{"rules":[{"host":"spydon.example.com","http":{"paths":[{"backend":{"service":{"name":"spydon-blue-frontend"}}}]}}]}}'
```

### 金丝雀发布

```yaml
# 使用 Argo Rollouts
apiVersion: argoproj.io/v1alpha1
kind: Rollout
metadata:
  name: spydon-backend
spec:
  strategy:
    canary:
      steps:
        - setWeight: 20
        - pause: {duration: 5m}
        - setWeight: 50
        - pause: {duration: 5m}
        - setWeight: 100
```

## 监控和告警

### Prometheus 集成

```yaml
# values.yaml
backend:
  metrics:
    enabled: true
    serviceMonitor:
      enabled: true
```

### 日志收集

```yaml
# values.yaml
backend:
  logging:
    format: json
    level: info

  # Fluentd sidecar
  fluentd:
    enabled: true
```

## 参考资料

- [Helm 官方文档](https://helm.sh/docs/)
- [GitLab Helm Repository](https://docs.gitlab.com/ee/user/packages/helm_repository/)
- [ArgoCD Helm 支持](https://argo-cd.readthedocs.io/en/stable/user-guide/helm/)
- [Kubernetes 配置最佳实践](https://kubernetes.io/docs/concepts/configuration/overview/)

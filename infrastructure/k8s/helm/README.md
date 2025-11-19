# Spydon Helm Chart

## 概述

Spydon 应用的 Helm Chart，包含后端、前端和所有依赖服务的完整部署配置。

## 特性

- ✅ 完整的应用栈部署（Backend + Frontend + PostgreSQL + Redis + MinIO）
- ✅ 配置文件模板化（config.yaml 和 runtime.json）
- ✅ 多环境支持（development, staging, production）
- ✅ 自动配置管理
- ✅ Ingress 和服务发现
- ✅ 资源限制和健康检查

## 安装

### 1. 添加 Helm 仓库

```bash
# 添加 GitLab Helm 仓库
helm repo add spydon ${CI_API_V4_URL}/projects/${CI_PROJECT_ID}/packages/helm/stable

# 更新仓库
helm repo update
```

### 2. 准备配置文件

创建 `values-production.yaml` 文件：

```yaml
# 全局配置
global:
  domain: spydon.example.com
  environment: production

# 后端配置
backend:
  image:
    repository: registry.gitlab.com/your-org/spydon/backend
    tag: "latest"

  config:
    # 数据库配置（必需）
    databaseUrl: "postgres://user:password@postgres:5432/spydon?sslmode=disable"

    # 安全密钥（必需）
    jwtSecret: "your-secure-jwt-secret-here"
    hmacSecret: "your-secure-hmac-secret-here"
    ingestApiKey: "your-ingest-api-key"

    # OIDC 配置
    oidc:
      issuer: "https://auth.example.com"
      clientId: "spydon-client"
      clientSecret: "your-oidc-secret"
      redirectUrl: "https://spydon.example.com/auth/callback"

    # HolmesGPT 配置
    holmesGpt:
      url: "http://holmes-gpt:8081"
      apiKey: "your-holmes-api-key"
      enabled: true
      model: "deepseek-reasoner"

    # CAS 配置（可选）
    cas:
      enabled: true
      serverUrl: "https://cas.example.com/cas"
      redirectUrl: "https://spydon.example.com"

    # MinIO 配置
    minio:
      endpoint: "minio:9000"
      accessKey: "minioadmin"
      secretKey: "minioadmin"
      bucketName: "robusta-artifacts"
      useSsl: false

    # 邮件配置（可选）
    email:
      smtpHost: "smtp.example.com"
      smtpPort: 587
      smtpUser: "noreply@example.com"
      smtpPass: "your-smtp-password"
      from: "Spydon <noreply@example.com>"
      rcaTo: "alerts@example.com"
      enabled: true

# 前端配置
frontend:
  image:
    repository: registry.gitlab.com/your-org/spydon/frontend
    tag: "latest"

  config:
    apiUrl: "https://spydon.example.com/api"
    wsUrl: "wss://spydon.example.com/ws"
    minioEndpoint: "minio.example.com"
    minioPort: "443"
    minioUseSsl: "true"

# Ingress 配置
ingress:
  enabled: true
  className: nginx
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod
  hosts:
    - host: spydon.example.com
      paths:
        - path: /
          pathType: Prefix
  tls:
    - secretName: spydon-tls
      hosts:
        - spydon.example.com

# PostgreSQL 配置
postgresql:
  enabled: true
  auth:
    username: spydon
    password: your-postgres-password
    database: spydon
  primary:
    persistence:
      enabled: true
      size: 20Gi

# Redis 配置
redis:
  enabled: true
  auth:
    enabled: false
  master:
    persistence:
      enabled: true
      size: 8Gi

# MinIO 配置
minio:
  enabled: true
  auth:
    rootUser: minioadmin
    rootPassword: minioadmin
  persistence:
    enabled: true
    size: 50Gi
```

### 3. 安装 Chart

```bash
# 安装到 production 命名空间
helm install spydon spydon/spydon \
  --namespace production \
  --create-namespace \
  --values values-production.yaml

# 或者指定版本
helm install spydon spydon/spydon \
  --version 0.1.0 \
  --namespace production \
  --create-namespace \
  --values values-production.yaml
```

### 4. 升级 Chart

```bash
# 升级到新版本
helm upgrade spydon spydon/spydon \
  --namespace production \
  --values values-production.yaml

# 升级并等待就绪
helm upgrade spydon spydon/spydon \
  --namespace production \
  --values values-production.yaml \
  --wait \
  --timeout 10m
```

### 5. 卸载

```bash
helm uninstall spydon --namespace production
```

## 配置说明

### 后端配置文件 (config.yaml)

后端配置通过 ConfigMap 挂载到 `/app/config/config.yaml`，包含：

- **数据库连接**：PostgreSQL 连接字符串
- **认证配置**：JWT、HMAC、OIDC、CAS
- **外部服务**：HolmesGPT、MinIO、Email
- **应用设置**：日志级别、限流等

### 前端配置文件 (runtime.json)

前端配置通过 ConfigMap 挂载到 `/app/public/runtime.json`，包含：

- **API 地址**：后端 API 和 WebSocket 地址
- **MinIO 配置**：对象存储访问配置
- **环境标识**：当前部署环境

## 环境变量覆盖

可以通过环境变量覆盖配置文件中的值：

```yaml
backend:
  env:
    - name: DATABASE_URL
      valueFrom:
        secretKeyRef:
          name: database-credentials
          key: url
    - name: JWT_SECRET
      valueFrom:
        secretKeyRef:
          name: app-secrets
          key: jwt-secret
```

## 使用 Secrets

推荐将敏感信息存储在 Kubernetes Secrets 中：

```bash
# 创建 Secret
kubectl create secret generic app-secrets \
  --from-literal=jwt-secret=your-jwt-secret \
  --from-literal=hmac-secret=your-hmac-secret \
  --from-literal=database-password=your-db-password \
  --namespace production

# 在 values.yaml 中引用
backend:
  existingSecret: app-secrets
```

## 多环境部署

### Development

```bash
helm install spydon-dev spydon/spydon \
  --namespace development \
  --create-namespace \
  --values values-development.yaml
```

### Staging

```bash
helm install spydon-staging spydon/spydon \
  --namespace staging \
  --create-namespace \
  --values values-staging.yaml
```

### Production

```bash
helm install spydon spydon/spydon \
  --namespace production \
  --create-namespace \
  --values values-production.yaml
```

## 故障排查

### 查看 Pod 状态

```bash
kubectl get pods -n production
kubectl describe pod <pod-name> -n production
kubectl logs <pod-name> -n production
```

### 查看配置

```bash
# 查看 ConfigMap
kubectl get configmap -n production
kubectl describe configmap spydon-backend-config -n production

# 查看实际配置内容
kubectl get configmap spydon-backend-config -n production -o yaml
```

### 测试配置

```bash
# 进入 Pod 查看配置文件
kubectl exec -it <backend-pod> -n production -- cat /app/config/config.yaml
kubectl exec -it <frontend-pod> -n production -- cat /app/public/runtime.json
```

## CI/CD 集成

Chart 会自动通过 GitLab CI 构建和发布：

1. **代码提交** → 触发 CI Pipeline
2. **构建镜像** → 推送到 Container Registry
3. **打包 Chart** → 更新镜像标签和版本
4. **发布 Chart** → 推送到 Helm Repository

### 在 ArgoCD 中使用

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: spydon
  namespace: argocd
spec:
  project: default
  source:
    repoURL: ${CI_API_V4_URL}/projects/${CI_PROJECT_ID}/packages/helm/stable
    chart: spydon
    targetRevision: 0.1.0
    helm:
      valueFiles:
        - values-production.yaml
  destination:
    server: https://kubernetes.default.svc
    namespace: production
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
```

## 参考

- [Helm 文档](https://helm.sh/docs/)
- [Kubernetes 配置最佳实践](https://kubernetes.io/docs/concepts/configuration/)
- [ArgoCD Helm 集成](https://argo-cd.readthedocs.io/en/stable/user-guide/helm/)

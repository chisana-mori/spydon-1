# Spydon GitOps 配置

## 目录结构

```
infrastructure/gitops/
├── README.md                    # 本文件
├── base/                        # 基础配置（所有环境共享）
│   ├── backend/
│   │   ├── deployment.yaml
│   │   ├── service.yaml
│   │   ├── configmap.yaml
│   │   └── kustomization.yaml
│   ├── frontend/
│   │   ├── deployment.yaml
│   │   ├── service.yaml
│   │   └── kustomization.yaml
│   └── common/
│       ├── namespace.yaml
│       └── kustomization.yaml
│
├── environments/                # 环境特定配置
│   ├── development/
│   │   ├── kustomization.yaml
│   │   ├── backend-patch.yaml
│   │   ├── frontend-patch.yaml
│   │   └── version.yaml
│   ├── staging/
│   │   ├── kustomization.yaml
│   │   ├── backend-patch.yaml
│   │   ├── frontend-patch.yaml
│   │   ├── ingress.yaml
│   │   └── version.yaml
│   └── production/
│       ├── kustomization.yaml
│       ├── backend-patch.yaml
│       ├── frontend-patch.yaml
│       ├── ingress.yaml
│       ├── hpa.yaml
│       └── version.yaml
│
└── scripts/                     # 辅助脚本
    ├── sync.sh                  # 同步脚本
    └── validate.sh              # 验证脚本
```

## 工作流程

### 1. GitLab CI 更新配置

```bash
# 在 .gitlab-ci.yml 中
cd infrastructure/gitops/environments/staging
sed -i "s|newTag:.*|newTag: $CI_COMMIT_SHA|g" kustomization.yaml
git commit -am "Update staging to $CI_COMMIT_SHA"
git push
```

### 2. ArgoCD 自动同步

ArgoCD 监控 `infrastructure/gitops/environments/staging` 目录，检测到变更后：

1. 执行 `kustomize build`
2. 生成 Kubernetes manifests
3. 应用到集群

## 环境说明

### Development
- **命名空间**: `spydon-dev`
- **域名**: `spydon-dev.example.com`
- **副本数**: 1
- **自动同步**: 是
- **自我修复**: 是

### Staging
- **命名空间**: `spydon-staging`
- **域名**: `spydon-staging.example.com`
- **副本数**: 2
- **自动同步**: 是
- **自我修复**: 是

### Production
- **命名空间**: `spydon-prod`
- **域名**: `spydon.example.com`
- **副本数**: 3
- **自动同步**: 否（需要手动审批）
- **自我修复**: 是

## 版本追踪

每次部署都会更新 `version.yaml` 文件：

```yaml
version: abc123def456
branch: main
pipeline: https://gitlab.com/your-org/robusta-web/-/pipelines/12345
timestamp: 2025-01-15T10:30:00Z
author: John Doe
message: "Fix memory leak in backend"
```

## ArgoCD 应用配置

### Staging

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: spydon-staging
spec:
  source:
    repoURL: https://gitlab.com/your-org/robusta-web.git
    path: infrastructure/gitops/environments/staging
    targetRevision: main
  destination:
    server: https://kubernetes.default.svc
    namespace: spydon-staging
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
```

### Production

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: spydon-production
spec:
  source:
    repoURL: https://gitlab.com/your-org/robusta-web.git
    path: infrastructure/gitops/environments/production
    targetRevision: main
  destination:
    server: https://kubernetes.default.svc
    namespace: spydon-prod
  syncPolicy:
    # 生产环境需要手动同步
    automated: null
```

## 常用命令

### 验证配置

```bash
# 验证 Kustomize 配置
kustomize build infrastructure/gitops/environments/staging

# 验证 YAML 语法
yamllint infrastructure/gitops/
```

### 本地测试

```bash
# 生成 manifests
kustomize build infrastructure/gitops/environments/staging > /tmp/staging-manifests.yaml

# 查看差异
kubectl diff -f /tmp/staging-manifests.yaml

# 应用（谨慎！）
kubectl apply -f /tmp/staging-manifests.yaml
```

## 最佳实践

1. **不要直接修改 base/**：所有环境特定的配置都应该在 `environments/` 中通过 patch 覆盖
2. **使用 version.yaml**：记录每次部署的详细信息，便于追踪和回滚
3. **小步提交**：每次只更新一个环境，便于问题定位
4. **测试先行**：先在 development 测试，再到 staging，最后到 production
5. **保持简洁**：避免过度复杂的 Kustomize 配置

## 故障排查

### ArgoCD 不同步

```bash
# 检查 ArgoCD 应用状态
argocd app get spydon-staging

# 手动刷新
argocd app sync spydon-staging --force

# 查看差异
argocd app diff spydon-staging
```

### Kustomize 构建失败

```bash
# 本地验证
kustomize build infrastructure/gitops/environments/staging

# 检查语法
yamllint infrastructure/gitops/environments/staging/
```

## 迁移到独立 GitOps 仓库（可选）

如果将来需要迁移到独立的 GitOps 仓库：

```bash
# 1. 创建新仓库
git init robusta-web-gitops
cd robusta-web-gitops

# 2. 复制 gitops 目录
cp -r ../robusta-web/infrastructure/gitops/* .

# 3. 提交
git add .
git commit -m "Initial GitOps repository"
git push origin main

# 4. 更新 ArgoCD 配置
# 修改 repoURL 指向新仓库
```

## 参考资料

- [Kustomize 文档](https://kustomize.io/)
- [ArgoCD 文档](https://argo-cd.readthedocs.io/)
- [GitOps 最佳实践](https://www.gitops.tech/)

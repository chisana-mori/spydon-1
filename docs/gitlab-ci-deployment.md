# Spydon GitLab CI/CD 部署指南

## 概述

本文档详细介绍了 Spydon 项目在 GitLab CI/CD 环境下的完整部署流程。通过 GitLab 的强大 CI/CD 功能，实现了从代码提交到生产部署的全自动化流程。

## 🏗️ 系统架构

```mermaid
graph TB
    subgraph "GitLab 代码仓库"
        A[Main Repository] --> B[GitOps Repository]
    end

    subgraph "GitLab CI/CD"
        C[Pipeline Trigger] --> D[Code Quality]
        D --> E[Security Scan]
        E --> F[Build Images]
        F --> G[Run Tests]
        G --> H[Deploy Preview]
    end

    subgraph "部署系统"
        I[Argo CD] --> J[Kubernetes]
        J --> K[Helm Charts]
    end

    subgraph "应用环境"
        L[开发环境] --> M[预览环境]
        M --> N[预生产环境]
        N --> O[生产环境]
    end

    A --> C
    B --> I
    H --> M
    J --> L
    J --> N
    J --> O
```

## 🚀 快速开始

### 1. GitLab 项目配置

#### 创建项目和环境

1. **创建 GitLab 项目**
   ```bash
   # 克隆项目
   git clone https://gitlab.com/your-org/spydon.git
   cd spydon

   # 推送到 GitLab
   git remote set-url origin https://gitlab.com/your-org/spydon.git
   git push -u origin main
   ```

2. **配置环境**
   - 进入 `Settings > Environments`
   - 创建以下环境：
     - `development` - 开发环境
     - `staging` - 预生产环境
     - `production` - 生产环境

3. **配置 CI/CD 变量**
   - 进入 `Settings > CI/CD > Variables`
   - 配置项目级别变量（参考 `.gitlab/gitlab-config.yml`）
   - 配置受保护的变量用于生产环境

### 2. CI/CD 变量配置

#### 通用变量

```bash
# 在 GitLab CI/CD Variables 中配置
CI_REGISTRY=registry.gitlab.com
CI_REGISTRY_IMAGE=registry.gitlab.com/your-org/spydon
DOCKER_PLATFORM=linux/amd64,linux/arm64
BUILD_TIMEOUT=600
COVERAGE_THRESHOLD=70

# 通知配置（可选）
SLACK_WEBHOOK_URL=https://hooks.slack.com/services/...
```

#### Kubernetes 配置

```bash
# 将 kubeconfig 文件进行 Base64 编码
base64 kubeconfig > kubeconfig.base64

# 在 GitLab CI/CD Variables 中添加
KUBECONFIG_DATA= # 粘贴 Base64 编码的内容
```

#### 外部服务配置

```bash
# 数据库配置
DATABASE_HOST=postgres.internal
DATABASE_USER=spydon
DATABASE_PASSWORD=your-secure-password
DATABASE_NAME=robusta

# Redis 配置
REDIS_HOST=redis.internal
REDIS_PASSWORD=your-redis-password

# MinIO 配置
MINIO_ENDPOINT=minio.internal
MINIO_ACCESS_KEY=your-access-key
MINIO_SECRET_KEY=your-secret-key

# HolmesGPT 配置
HOLMES_GPT_URL=https://holmesgpt.example.com
HOLMES_GPT_API_KEY=your-api-key
```

### 3. 基础设施设置

#### 安装 Kubernetes 工具

```bash
# 在 GitLab Runner 所在机器上安装
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
sudo install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl

# 安装 Helm
curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash
```

#### 配置 GitLab Runner

```bash
# 安装 GitLab Runner
sudo apt-get update
sudo apt-get install -y gitlab-runner

# 注册 Runner
sudo gitlab-runner register \
  --url https://gitlab.com/ \
  --registration-token YOUR_TOKEN \
  --description "spydon-docker-runner" \
  --executor "docker" \
  --docker-image "docker:24.0.6" \
  --docker-privileged
```

## 🔄 CI/CD 流程详解

### 流水线阶段

```yaml
stages:
  - validate          # 代码质量检查
  - test             # 自动化测试
  - security         # 安全扫描
  - build            # Docker 镜像构建
  - integration-test # 集成测试
  - deploy-preview   # 预览环境部署
  - deploy-staging   # 预生产环境部署
  - deploy-production # 生产环境部署
```

### 1. 代码质量检查

**作业**: `code-quality`

**功能**：
- Go 代码格式化和静态分析
- Node.js 代码 linting 和类型检查
- 生成质量报告

**配置**：
```yaml
code-quality:
  stage: validate
  image: alpine:3.18
  script:
    - echo "🔍 Running code quality checks..."
    # Backend 检查
    - go fmt ./...
    - go vet ./...
    - golangci-lint run --timeout=5m
    # Frontend 检查
    - npm ci
    - npm run lint
    - npm run type-check
```

### 2. 安全扫描

**作业**: `security-scan`

**功能**：
- 代码仓库漏洞扫描
- 依赖安全检查
- 生成安全报告

**配置**：
```yaml
security-scan:
  stage: security
  image:
    name: aquasec/trivy:latest
    entrypoint: [""]
  script:
    - trivy fs --exit-code 0 --format sarif --output trivy-repo.sarif .
    - npm audit --audit-level moderate
  artifacts:
    reports:
      sast: gl-sast-report.json
      dependency_scanning: gl-dependency-scanning-report.json
```

### 3. 自动化测试

#### 后端测试

**作业**: `backend-test`

**功能**：
- Go 单元测试执行
- 代码覆盖率分析
- JUnit 报告生成

**配置**：
```yaml
backend-test:
  stage: test
  image: golang:1.23-alpine
  services:
    - postgres:15-alpine
    - redis:7-alpine
  variables:
    DATABASE_URL: postgres://postgres:postgres@postgres:5432/test_db
    REDIS_URL: redis://redis:6379
  script:
    - go test -v -race -coverprofile=coverage.out ./...
    - go tool cover -html=coverage.out -o coverage.html
  coverage: '/total:.*?(\d+\.\d+)%/'
```

#### 前端测试

**作业**: `frontend-test`

**功能**：
- Node.js 单元测试
- 组件测试
- 覆盖率报告

**配置**：
```yaml
frontend-test:
  stage: test
  image: node:20-alpine
  script:
    - npm ci --cache .npm --prefer-offline
    - npm run test:coverage
  coverage: '/All files[^|]*\|[^|]*\s+([\d\.]+)/'
```

### 4. Docker 镜像构建

**作业**: `build-images`

**功能**：
- 多架构 Docker 镜像构建
- 镜像安全扫描
- 推送到 GitLab Registry

**配置**：
```yaml
build-images:
  stage: build
  script:
    - docker buildx build \
      --platform linux/amd64,linux/arm64 \
      --file apps/backend/Dockerfile.optimized \
      --tag $BACKEND_IMAGE:$CI_COMMIT_SHA \
      --build-arg VERSION=$CI_COMMIT_REF_SLUG \
      --push \
      apps/backend/
```

### 5. 集成测试

**作业**: `integration-test`

**功能**：
- 端到端测试环境搭建
- API 集成测试
- 前端集成测试

**配置**：
```yaml
integration-test:
  stage: integration-test
  services:
    - docker:24.0.6-dind
    - postgres:15-alpine
    - redis:7-alpine
  script:
    - docker-compose -f docker-compose.test.yml up -d
    - timeout 300 bash -c 'until curl -f http://localhost:8080/health; do sleep 5; done'
    - curl -f http://localhost:8080/api/v1/health
    - docker-compose -f docker-compose.test.yml down -v
```

## 🌍 环境部署策略

### 预览环境部署

**触发条件**：创建/更新 Merge Request

**自动化流程**：
1. 构建镜像完成
2. 创建临时 Kubernetes 命名空间
3. 部署应用到预览环境
4. 配置 Ingress 和域名
5. 发送部署通知

**配置示例**：
```yaml
deploy-preview:
  stage: deploy-preview
  environment:
    name: preview/$CI_MERGE_REQUEST_IID
    url: https://pr-$CI_MERGE_REQUEST_IID.$PREVIEW_DOMAIN
    on_stop: stop-preview
  script:
    - ./infrastructure/preview-env/create-preview.sh \
      --pr-number $CI_MERGE_REQUEST_IID \
      --commit $CI_COMMIT_SHA \
      --backend-image $BACKEND_IMAGE \
      --frontend-image $FRONTEND_IMAGE
```

**访问预览环境**：
- URL: `https://pr-{MR_IID}.spydon-preview.local`
- 生命周期：MR 合并后自动清理

### 预生产环境部署

**触发条件**：main 分支更新

**部署步骤**：
1. 手动触发部署
2. 更新 GitOps 仓库
3. Argo CD 自动同步
4. 健康检查验证
5. 发送部署通知

**配置示例**：
```yaml
deploy-staging:
  stage: deploy-staging
  environment:
    name: staging
    url: https://spydon-staging.example.com
  script:
    - ./infrastructure/gitops/scripts/sync.sh deploy staging $CI_COMMIT_SHA
  when: manual
```

### 生产环境部署

**触发条件**：手动审批

**蓝绿部署流程**：
1. 部署到绿环境
2. 验证绿环境健康
3. 切换流量到绿环境
4. 验证生产环境
5. 清理蓝环境

**配置示例**：
```yaml
deploy-production:
  stage: deploy-production
  environment:
    name: production
    url: https://spydon.example.com
  script:
    - ./infrastructure/gitops/scripts/sync.sh deploy production $CI_COMMIT_SHA
    - # 蓝绿部署逻辑
    - kubectl patch service robusta-frontend -p '{"spec":{"selector":{"version":"green"}}}'
  when: manual
```

## 📊 监控和告警

### GitLab CI 监控

#### 流水线监控

```mermaid
graph TD
    A[Pipeline Start] --> B[Stage Execution]
    B --> C{Job Success?}
    C -->|Yes| D[Next Stage]
    C -->|No| E[Failure Handling]
    E --> F[Notify Team]
    D --> G{Pipeline Complete?}
    G -->|Yes| H[Success Notification]
    G -->|No| B
```

#### 告警配置

**Slack 集成**：
```yaml
# 在 .gitlab-ci.yml 中配置
deploy-production:
  script:
    - ./deploy.sh production
  after_script:
    - |
      if [ "$CI_JOB_STATUS" = "success" ]; then
        curl -X POST -H 'Content-type: application/json' \
          --data '{"text":"✅ Production deployment successful"}' \
          $SLACK_WEBHOOK_URL
      else
        curl -X POST -H 'Content-type: application/json' \
          --data '{"text":"❌ Production deployment failed"}' \
          $SLACK_WEBHOOK_URL
      fi
```

### 应用监控

#### Prometheus 指标

```yaml
# 监控指标配置
metrics:
  - name: spydon_deployment_duration_seconds
    type: histogram
    description: Deployment duration in seconds

  - name: spydon_build_success_total
    type: counter
    description: Number of successful builds

  - name: spydon_test_coverage_percent
    type: gauge
    description: Test coverage percentage
```

#### 告警规则

```yaml
# 告警规则配置
alerts:
  - name: DeploymentFailure
    condition: spydon_deployment_failure_total > 0
    for: 5m
    severity: critical

  - name: LowTestCoverage
    condition: spydon_test_coverage_percent < 70
    for: 10m
    severity: warning
```

## 🔧 故障处理

### 常见问题诊断

#### 1. 流水线失败

**诊断步骤**：
```bash
# 检查作业日志
# 在 GitLab UI 中查看失败作业的详细日志

# 本地调试
gitlab-runner exec docker <job-name>

# 检查 Runner 状态
sudo gitlab-runner status
sudo gitlab-runner verify
```

#### 2. Docker 构建失败

**常见原因**：
- Dockerfile 语法错误
- 依赖下载失败
- 镜像仓库权限问题

**解决方案**：
```bash
# 本地调试 Docker 构建
docker build -f apps/backend/Dockerfile.optimized apps/backend/

# 检查 Registry 连接
docker login $CI_REGISTRY
docker pull $CI_REGISTRY_IMAGE/alpine:latest
```

#### 3. 部署失败

**诊断步骤**：
```bash
# 检查 kubectl 配置
kubectl cluster-info
kubectl get nodes

# 检查资源状态
kubectl get all -n spydon-production
kubectl describe deployment robusta-backend -n spydon-production

# 查看事件
kubectl get events -n spydon-production --sort-by='.lastTimestamp'
```

### 应急响应流程

#### 自动回滚

```yaml
# Argo CD 自动回滚配置
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: spydon-production
spec:
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    retry:
      limit: 5
      backoff:
        duration: 5s
        factor: 2
        maxDuration: 3m
```

#### 手动回滚

```bash
# 使用 GitOps 脚本回滚
./infrastructure/gitops/scripts/sync.sh rollback production

# 直接使用 kubectl 回滚
kubectl rollout undo deployment/robusta-backend -n spydon-production
```

## 💡 最佳实践

### 1. CI/CD 配置优化

- **缓存策略**：合理配置 npm 和 Go mod 缓存
- **并行执行**：合理分配作业并行度
- **资源限制**：设置适当的资源请求和限制

### 2. 安全最佳实践

- **变量保护**：敏感信息使用受保护变量
- **权限最小化**：按需分配 GitLab 权限
- **定期轮换**：定期更新密钥和访问令牌

### 3. 环境管理

- **环境隔离**：完全隔离不同环境的资源
- **命名规范**：统一的命名和标签规范
- **清理策略**：及时清理不再使用的资源

### 4. 监控运维

- **全面监控**：覆盖所有关键指标
- **及时告警**：合理的告警阈值和升级策略
- **文档完善**：保持操作手册和故障处理指南更新

## 📚 相关文档

- [GitLab CI/CD 官方文档](https://docs.gitlab.com/ee/ci/)
- [GitLab Kubernetes 集成](https://docs.gitlab.com/ee/user/infrastructure/)
- [Argo CD 用户指南](https://argo-cd.readthedocs.io/)
- [Docker 构建最佳实践](https://docs.docker.com/develop/dev-best-practices/)

## 🆘 支持和帮助

### 技术支持

- **GitLab 文档**：https://docs.gitlab.com
- **项目 Issues**：https://gitlab.com/your-org/spydon/-/issues
- **项目 Wiki**：https://gitlab.com/your-org/spydon/-/wikis

### 联系方式

- **技术支持**：support@spydon.com
- **DevOps 团队**：devops@spydon.com
- **紧急响应**：emergency@spydon.com

---

## 🎉 总结

通过 GitLab CI/CD，Spydon 项目实现了：

✅ **全自动化流水线**：从代码提交到部署的全流程自动化
✅ **多环境支持**：开发、预览、预生产、生产环境完整支持
✅ **GitOps 部署**：声明式配置，版本控制驱动
✅ **安全集成**：内置安全扫描和依赖检查
✅ **灵活配置**：支持多种部署策略和环境配置
✅ **全面监控**：集成告警和通知系统

这套完整的 GitLab CI/CD 流程确保了 Spydon 项目的高质量交付和稳定运行。

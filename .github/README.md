# GitHub Actions Workflows

本目录包含 Spydon 项目的 GitHub Actions CI/CD 工作流配置。

## 工作流列表

### 1. CI Pipeline (`workflows/ci.yml`)

主要的持续集成流程，包含：

- ✅ 代码质量检查（pre-commit, linting）
- 🔒 安全扫描（Trivy, npm audit）
- 🧪 单元测试（后端 + 前端）
- 🐳 Docker 镜像构建（多平台）
- 📦 Helm Chart 打包
- 📤 发布到 Harbor

**触发条件：**
- Pull Request 到 `main` 或 `develop`
- Push 到 `main` 或 `develop`
- 创建版本标签（`v*`）

### 2. Cleanup (`workflows/cleanup.yml`)

自动清理旧的 Docker 镜像，保留最近 10 个版本。

**触发条件：**
- 每天 UTC 2:00 自动运行
- 手动触发

### 3. Deploy (`workflows/deploy.yml`)

部署工作流（如果存在）。

### 4. PR Cleanup (`workflows/pr-cleanup.yml`)

清理 PR 相关资源（如果存在）。

## 快速开始

### 1. 配置 Secrets

在 GitHub 仓库设置中添加以下 secrets：

#### 必需
无（使用 `GITHUB_TOKEN` 自动认证）

#### 可选
- `CODECOV_TOKEN`: 上传覆盖率到 Codecov
- `HARBOR_REGISTRY`: Harbor 注册表地址（如 `harbor.example.com`）
- `HARBOR_PROJECT`: Harbor 项目名称（如 `spydon`）
- `HARBOR_USERNAME`: Harbor 用户名
- `HARBOR_PASSWORD`: Harbor 密码
- `NEXT_PUBLIC_API_URL`: 前端 API URL

### 2. 启用 GitHub Container Registry

1. 进入仓库设置 → Actions → General
2. 在 "Workflow permissions" 中选择 "Read and write permissions"
3. 保存更改

### 3. 测试工作流

创建一个 Pull Request 或推送到 `develop` 分支，观察工作流运行。

## 工作流架构

```
┌─────────────────────────────────────────────────────────┐
│                    Trigger Event                         │
│         (Push / Pull Request / Tag)                      │
└────────────────────┬────────────────────────────────────┘
                     │
        ┌────────────┴────────────┐
        │   Stage 1: Validate     │
        │  - pre-commit           │
        │  - backend-lint         │
        │  - frontend-lint        │
        └────────────┬────────────┘
                     │
        ┌────────────┴────────────┐
        │   Stage 2: Security     │
        │  - security-scan        │
        └────────────┬────────────┘
                     │
        ┌────────────┴────────────┐
        │   Stage 3: Test         │
        │  - backend-test         │
        │  - frontend-test        │
        └────────────┬────────────┘
                     │
        ┌────────────┴────────────┐
        │   Stage 4: Build        │
        │  - build-backend-image  │
        │  - build-frontend-image │
        └────────────┬────────────┘
                     │
        ┌────────────┴────────────┐
        │   Stage 5: Publish      │
        │  - package-chart        │
        │  - publish-chart        │
        └─────────────────────────┘
```

## 镜像标签策略

构建的 Docker 镜像会自动打上以下标签：

- `sha-<full-sha>`: 完整的 commit SHA（主标签）
- `<branch-name>`: 分支名称（如 `main`, `develop`）
- `pr-<number>`: Pull Request 编号
- `latest`: 仅在 `main` 分支

**示例：**
```
ghcr.io/your-org/robusta-web/backend:sha-abc123def456...
ghcr.io/your-org/robusta-web/backend:main
ghcr.io/your-org/robusta-web/backend:latest
```

## 查看构建的镜像

1. 进入仓库主页
2. 点击右侧的 "Packages"
3. 选择 `backend` 或 `frontend`

## 常见问题

### Q: 为什么镜像推送失败？

A: 检查 Workflow permissions 是否设置为 "Read and write permissions"。

### Q: 如何手动触发工作流？

A: 进入 Actions 标签页，选择工作流，点击 "Run workflow"。

### Q: 如何查看工作流日志？

A: 进入 Actions 标签页，选择具体的运行记录，点击查看详细日志。

### Q: Harbor 发布失败怎么办？

A: 检查 Harbor secrets 是否正确配置。如果不需要发布到 Harbor，工作流会自动跳过该步骤。

### Q: 如何禁用某个工作流？

A: 进入 Actions 标签页，选择工作流，点击右上角的 "..." → "Disable workflow"。

## 性能优化

- ✅ 使用 GitHub Actions Cache 缓存依赖
- ✅ 使用 Docker layer cache 加速构建
- ✅ 并行运行独立的 jobs
- ✅ 仅在必要时运行耗时任务

## 相关文档

- [GitHub Actions 迁移指南](../docs/github-actions-migration.md)
- [CI 系统对比](../docs/ci-comparison.md)
- [GitLab CI 配置](../.gitlab-ci.yml)

## 支持

如有问题，请：
1. 查看 [GitHub Actions 文档](https://docs.github.com/en/actions)
2. 查看工作流运行日志
3. 提交 Issue

# GitHub Actions CI/CD Pipeline

本文档说明从 GitLab CI 迁移到 GitHub Actions 的 CI/CD 流程。

## 概述

GitHub Actions 工作流已根据 `.gitlab-ci.yml` 配置创建，保持相同的阶段和功能：

1. **代码质量检查** (Validate)
2. **安全扫描** (Security)
3. **单元测试** (Test)
4. **构建镜像** (Build)
5. **发布** (Publish)

## 工作流文件

### 主 CI 流程：`.github/workflows/ci.yml`

触发条件：
- Pull Request 到 `main` 或 `develop` 分支
- Push 到 `main` 或 `develop` 分支
- 创建 tag（格式：`v*`）

### 清理流程：`.github/workflows/cleanup.yml`

- 每天自动运行，清理旧的 Docker 镜像
- 保留最近 10 个版本

## 主要差异对比

| 功能 | GitLab CI | GitHub Actions |
|------|-----------|----------------|
| 容器注册表 | GitLab Container Registry | GitHub Container Registry (ghcr.io) |
| 缓存 | GitLab Cache | GitHub Actions Cache |
| Artifacts | GitLab Artifacts | GitHub Actions Artifacts |
| 服务容器 | `services:` | `services:` (相同) |
| 环境变量 | `$CI_*` | `${{ github.* }}` |
| 镜像清理 | GitLab API | GitHub Packages API |

## 工作流阶段详解

### 1. 代码质量检查

#### pre-commit
- 运行 pre-commit hooks
- 使用 Python 3.11
- 缓存 pre-commit 环境

#### backend-lint
- 使用 golangci-lint v1.54.2
- Go 1.23
- 5分钟超时

#### frontend-lint
- Node.js 20
- 运行 ESLint 和 TypeScript 类型检查
- 上传覆盖率报告

### 2. 安全扫描

- 使用 Trivy 扫描代码仓库
- 运行 npm audit
- 上传 SARIF 报告到 GitHub Security
- 依赖：backend-lint, frontend-lint

### 3. 单元测试

#### backend-test
- 服务：PostgreSQL 15, Redis 7
- 运行测试并生成覆盖率
- 上传到 Codecov（需配置 `CODECOV_TOKEN`）
- 依赖：backend-lint

#### frontend-test
- Node.js 20
- 运行 Jest 测试
- 生成覆盖率报告
- 依赖：frontend-lint

### 4. 构建 Docker 镜像

#### build-backend-image
- 多平台构建：linux/amd64, linux/arm64
- 使用 `Dockerfile.optimized`
- 推送到 ghcr.io
- Trivy 镜像扫描
- 依赖：backend-test, frontend-test, security-scan

#### build-frontend-image
- 多平台构建：linux/amd64, linux/arm64
- 使用 `Dockerfile.optimized`
- 推送到 ghcr.io
- Trivy 镜像扫描
- 依赖：frontend-test, security-scan

### 5. Helm Chart 打包和发布

#### package-chart
- 仅在 main/develop 分支或 tag 时运行
- 使用 Helm 3.14.4
- 更新 Chart 版本和镜像标签
- 上传 Chart artifacts

#### publish-chart
- 发布到 Harbor（需配置密钥）
- 为 tag 创建 GitHub Release
- 依赖：package-chart

## 必需的 GitHub Secrets

在 GitHub 仓库设置中配置以下 secrets：

### 可选（推荐）
- `CODECOV_TOKEN`: Codecov 上传 token
- `NEXT_PUBLIC_API_URL`: 前端 API URL（构建时）

### Harbor 发布（可选）
- `HARBOR_REGISTRY`: Harbor 注册表地址
- `HARBOR_PROJECT`: Harbor 项目名称
- `HARBOR_USERNAME`: Harbor 用户名
- `HARBOR_PASSWORD`: Harbor 密码

## 镜像标签策略

GitHub Actions 自动生成以下标签：

- `sha-<full-commit-sha>`: 完整 commit SHA
- `<branch-name>`: 分支名称（如 `main`, `develop`）
- `pr-<number>`: Pull Request 编号
- `latest`: 仅在 main 分支

示例：
```
ghcr.io/your-org/robusta-web/backend:sha-abc123...
ghcr.io/your-org/robusta-web/backend:main
ghcr.io/your-org/robusta-web/backend:latest
```

## 与 GitLab CI 的兼容性

两个 CI 系统可以并行运行：
- GitLab CI 继续使用 GitLab Container Registry
- GitHub Actions 使用 GitHub Container Registry
- 互不干扰，可以逐步迁移

## 迁移检查清单

- [ ] 配置 GitHub Secrets
- [ ] 测试 PR 工作流
- [ ] 验证镜像构建和推送
- [ ] 配置 Harbor（如需要）
- [ ] 更新 ArgoCD 镜像源（如需要）
- [ ] 测试 Helm Chart 发布
- [ ] 配置 Codecov（可选）

## 故障排查

### 镜像推送失败
确保 GitHub Actions 有权限推送到 GHCR：
- 仓库设置 → Actions → General → Workflow permissions
- 选择 "Read and write permissions"

### Helm Chart 发布失败
检查 Harbor secrets 是否正确配置，或者忽略错误（workflow 会跳过）。

### 缓存问题
GitHub Actions 缓存限制为 10GB，可能需要定期清理。

## 参考资源

- [GitHub Actions 文档](https://docs.github.com/en/actions)
- [GitHub Container Registry](https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry)
- [Docker Buildx](https://docs.docker.com/buildx/working-with-buildx/)

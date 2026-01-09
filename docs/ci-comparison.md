# GitLab CI vs GitHub Actions 对比

## 快速对比表

| 特性 | GitLab CI | GitHub Actions |
|------|-----------|----------------|
| **配置文件** | `.gitlab-ci.yml` | `.github/workflows/*.yml` |
| **触发器** | `workflow.rules` | `on:` |
| **阶段** | `stages:` | `jobs:` (隐式阶段) |
| **依赖** | `needs:` | `needs:` (相同) |
| **缓存** | `cache:` | `actions/cache@v4` |
| **制品** | `artifacts:` | `actions/upload-artifact@v4` |
| **环境变量** | `$CI_*` | `${{ github.* }}` |
| **镜像仓库** | GitLab Registry | GitHub Container Registry |
| **Runner** | GitLab Runner | GitHub-hosted runners |

## 语法映射

### 触发条件

**GitLab CI:**
```yaml
workflow:
  rules:
    - if: '$CI_PIPELINE_SOURCE == "merge_request_event"'
    - if: '$CI_COMMIT_BRANCH =~ /^(main|develop)$/'
```

**GitHub Actions:**
```yaml
on:
  pull_request:
    branches: [main, develop]
  push:
    branches: [main, develop]
```

### 环境变量

| GitLab CI | GitHub Actions | 说明 |
|-----------|----------------|------|
| `$CI_COMMIT_SHA` | `${{ github.sha }}` | Commit SHA |
| `$CI_COMMIT_REF_NAME` | `${{ github.ref_name }}` | 分支/标签名 |
| `$CI_COMMIT_REF_SLUG` | `${{ github.ref_name }}` | 分支名（slug） |
| `$CI_COMMIT_SHORT_SHA` | `${{ github.sha }}` (需截取) | 短 SHA |
| `$CI_REGISTRY` | `ghcr.io` | 容器注册表 |
| `$CI_REGISTRY_IMAGE` | `ghcr.io/${{ github.repository }}` | 镜像路径 |
| `$CI_PROJECT_ID` | `${{ github.repository_id }}` | 项目 ID |
| `$CI_PIPELINE_SOURCE` | `${{ github.event_name }}` | 触发源 |

### Job 依赖

**GitLab CI:**
```yaml
backend-test:
  needs:
    - job: backend-lint
```

**GitHub Actions:**
```yaml
backend-test:
  needs: backend-lint
```

### 服务容器

**GitLab CI:**
```yaml
services:
  - postgres:15-alpine
  - redis:7-alpine
variables:
  POSTGRES_DB: test_db
```

**GitHub Actions:**
```yaml
services:
  postgres:
    image: postgres:15-alpine
    env:
      POSTGRES_DB: test_db
    ports:
      - 5432:5432
```

### 缓存

**GitLab CI:**
```yaml
cache:
  key:
    files:
      - go.sum
  paths:
    - vendor/
```

**GitHub Actions:**
```yaml
- uses: actions/cache@v4
  with:
    path: vendor/
    key: ${{ hashFiles('go.sum') }}
```

### Artifacts

**GitLab CI:**
```yaml
artifacts:
  paths:
    - coverage/
  expire_in: 1 week
```

**GitHub Actions:**
```yaml
- uses: actions/upload-artifact@v4
  with:
    name: coverage
    path: coverage/
    retention-days: 7
```

### Docker 构建

**GitLab CI:**
```yaml
script:
  - docker buildx build \
      --platform linux/amd64,linux/arm64 \
      --tag $BACKEND_IMAGE:$CI_COMMIT_SHA \
      --push \
      apps/backend/
```

**GitHub Actions:**
```yaml
- uses: docker/build-push-action@v5
  with:
    context: apps/backend
    platforms: linux/amd64,linux/arm64
    tags: ${{ env.BACKEND_IMAGE }}:${{ github.sha }}
    push: true
```

## 主要优势对比

### GitLab CI 优势
- ✅ 内置 Container Registry
- ✅ 更灵活的 rules 系统
- ✅ 原生支持 GitLab 生态
- ✅ 可自托管 Runner

### GitHub Actions 优势
- ✅ 丰富的 Actions 市场
- ✅ 更好的 UI/UX
- ✅ 免费额度更高（公开仓库无限）
- ✅ 与 GitHub 生态深度集成
- ✅ 更活跃的社区

## 迁移注意事项

1. **镜像仓库地址变更**
   - GitLab: `registry.gitlab.com/org/project`
   - GitHub: `ghcr.io/org/project`

2. **认证方式**
   - GitLab: `$CI_REGISTRY_PASSWORD`
   - GitHub: `${{ secrets.GITHUB_TOKEN }}`

3. **缓存限制**
   - GitLab: 取决于配置
   - GitHub: 10GB 总限制

4. **并发限制**
   - GitLab: 取决于 Runner 配置
   - GitHub: 免费版 20 个并发 jobs

5. **Artifacts 保留**
   - GitLab: 可配置，默认 30 天
   - GitHub: 最多 90 天

## 成本对比

### 公开仓库
- **GitLab**: 400 分钟/月（免费版）
- **GitHub**: 无限制

### 私有仓库
- **GitLab**: 400 分钟/月（免费版）
- **GitHub**: 2000 分钟/月（免费版）

## 推荐策略

1. **双轨运行**：同时保留两个 CI 配置，逐步迁移
2. **分阶段迁移**：先迁移简单的 jobs，再迁移复杂的
3. **监控对比**：对比两个系统的执行时间和成功率
4. **逐步切换**：确认 GitHub Actions 稳定后再完全切换

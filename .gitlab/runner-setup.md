# GitLab Runner 配置指南

## 架构说明

### Docker Executor 模式（推荐）

```
┌─────────────────────────────────────────────────────────┐
│                    GitLab Server                        │
│                                                         │
│  Project A (Go)    Project B (Node)    Project C (Python)│
└────────────┬────────────────┬────────────────┬──────────┘
             │                │                │
             └────────────────┼────────────────┘
                              │
                    ┌─────────▼──────────┐
                    │   GitLab Runner    │
                    │                    │
                    │  只需要安装:        │
                    │  • Docker          │
                    │  • GitLab Runner   │
                    └─────────┬──────────┘
                              │
        ┌─────────────────────┼─────────────────────┐
        │                     │                     │
   ┌────▼────┐          ┌────▼────┐          ┌────▼────┐
   │ golang  │          │  node   │          │ python  │
   │ 容器     │          │  容器    │          │  容器    │
   └─────────┘          └─────────┘          └─────────┘
```

**优势**：
- ✅ Runner 环境简单，只需要 Docker
- ✅ 每个项目使用自己的镜像，互不干扰
- ✅ 环境一致性好，本地和 CI 使用相同镜像
- ✅ 易于维护和扩展

## 安装 GitLab Runner

### 1. 在 Linux 上安装

```bash
# 添加 GitLab 官方仓库
curl -L "https://packages.gitlab.com/install/repositories/runner/gitlab-runner/script.deb.sh" | sudo bash

# 安装 GitLab Runner
sudo apt-get install gitlab-runner

# 安装 Docker
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker gitlab-runner

# 验证安装
gitlab-runner --version
docker --version
```

### 2. 在 macOS 上安装

```bash
# 使用 Homebrew 安装
brew install gitlab-runner

# 安装 Docker Desktop
# 从 https://www.docker.com/products/docker-desktop 下载安装

# 验证安装
gitlab-runner --version
docker --version
```

### 3. 使用 Docker 运行 Runner

```bash
# 创建配置目录
mkdir -p /srv/gitlab-runner/config

# 运行 Runner 容器
docker run -d --name gitlab-runner --restart always \
  -v /srv/gitlab-runner/config:/etc/gitlab-runner \
  -v /var/run/docker.sock:/var/run/docker.sock \
  gitlab/gitlab-runner:latest
```

## 注册 Runner

### 方法 1: 交互式注册（推荐新手）

```bash
# 注册 Runner
sudo gitlab-runner register

# 按提示输入：
# 1. GitLab URL: https://gitlab.com/
# 2. Registration token: 从 GitLab 项目 Settings > CI/CD > Runners 获取
# 3. Description: docker-runner-spydon
# 4. Tags: docker,build,test
# 5. Executor: docker
# 6. Default Docker image: docker:24.0.6
```

### 方法 2: 非交互式注册（推荐自动化）

```bash
sudo gitlab-runner register \
  --non-interactive \
  --url "https://gitlab.com/" \
  --registration-token "YOUR_REGISTRATION_TOKEN" \
  --executor "docker" \
  --docker-image "docker:24.0.6" \
  --description "docker-runner-spydon" \
  --tag-list "docker,build,test" \
  --docker-privileged \
  --docker-volumes "/var/run/docker.sock:/var/run/docker.sock" \
  --docker-volumes "/cache"
```

## Runner 配置文件

### 基础配置

```toml
# /etc/gitlab-runner/config.toml

# 并发执行的 Job 数量
concurrent = 4

# 检查新 Job 的间隔
check_interval = 0

# 会话服务器配置（用于交互式调试）
[session_server]
  session_timeout = 1800

[[runners]]
  name = "docker-runner-spydon"
  url = "https://gitlab.com/"
  token = "YOUR_RUNNER_TOKEN"
  executor = "docker"

  # Runner 标签
  # 在 .gitlab-ci.yml 中使用 tags: [docker] 来选择这个 Runner
  tags = ["docker", "build", "test"]

  [runners.custom_build_dir]

  [runners.cache]
    # 使用本地缓存
    Type = "local"
    Shared = true
    [runners.cache.local]
      Path = "/cache"

  [runners.docker]
    # 默认镜像
    image = "docker:24.0.6"

    # 允许 Docker-in-Docker（构建镜像需要）
    privileged = true

    # 禁用 entrypoint 覆盖
    disable_entrypoint_overwrite = false

    # 禁用 OOM Killer
    oom_kill_disable = false

    # 禁用缓存
    disable_cache = false

    # 挂载卷
    volumes = [
      "/cache",
      "/var/run/docker.sock:/var/run/docker.sock"
    ]

    # 共享内存大小
    shm_size = 0

    # 网络模式
    network_mode = "bridge"

    # 拉取策略
    pull_policy = ["if-not-present"]
```

### 高级配置（多项目场景）

```toml
concurrent = 8

# Runner 1: 通用构建和测试
[[runners]]
  name = "docker-runner-general"
  url = "https://gitlab.com/"
  token = "TOKEN_1"
  executor = "docker"
  tags = ["docker", "general"]

  [runners.docker]
    image = "docker:24.0.6"
    privileged = true
    volumes = ["/cache", "/var/run/docker.sock:/var/run/docker.sock"]

# Runner 2: 专用于重型构建任务
[[runners]]
  name = "docker-runner-build-heavy"
  url = "https://gitlab.com/"
  token = "TOKEN_2"
  executor = "docker"
  tags = ["docker", "build", "heavy"]

  # 限制并发数
  limit = 2

  [runners.docker]
    image = "docker:24.0.6"
    privileged = true
    # 更大的共享内存
    shm_size = 268435456  # 256MB
    # 更多 CPU 和内存
    cpus = "4"
    memory = "8g"
    volumes = ["/cache", "/var/run/docker.sock:/var/run/docker.sock"]

# Runner 3: 专用于部署任务
[[runners]]
  name = "docker-runner-deploy"
  url = "https://gitlab.com/"
  token = "TOKEN_3"
  executor = "docker"
  tags = ["docker", "deploy", "production"]

  # 只允许受保护的分支使用
  access_level = "ref_protected"

  [runners.docker]
    image = "alpine:3.18"
    privileged = false
    volumes = ["/cache"]
```

## 标签策略

### 推荐的标签体系

```yaml
# 基础标签
docker          # 使用 Docker Executor
shell           # 使用 Shell Executor

# 任务类型标签
build           # 构建任务
test            # 测试任务
deploy          # 部署任务
security        # 安全扫描

# 资源标签
light           # 轻量级任务（< 1 CPU, < 2GB RAM）
medium          # 中等任务（2-4 CPU, 2-8GB RAM）
heavy           # 重型任务（> 4 CPU, > 8GB RAM）

# 环境标签
development     # 开发环境
staging         # 预生产环境
production      # 生产环境

# 架构标签
x86_64          # AMD64 架构
arm64           # ARM64 架构
```

### 在 .gitlab-ci.yml 中使用标签

```yaml
# 示例 1: 轻量级测试任务
unit-test:
  tags:
    - docker
    - test
    - light

# 示例 2: 重型构建任务
build-multiarch:
  tags:
    - docker
    - build
    - heavy

# 示例 3: 生产部署任务
deploy-production:
  tags:
    - docker
    - deploy
    - production
```

## 性能优化

### 1. 使用缓存

```toml
[[runners]]
  [runners.cache]
    Type = "s3"
    Shared = true
    [runners.cache.s3]
      ServerAddress = "s3.amazonaws.com"
      AccessKey = "YOUR_ACCESS_KEY"
      SecretKey = "YOUR_SECRET_KEY"
      BucketName = "gitlab-runner-cache"
      BucketLocation = "us-east-1"
```

```yaml
# .gitlab-ci.yml
cache:
  key: "$CI_COMMIT_REF_SLUG"
  paths:
    - apps/backend/vendor/
    - apps/frontend/node_modules/
    - .cache/
```

### 2. 使用镜像缓存

```toml
[[runners]]
  [runners.docker]
    # 优先使用本地镜像
    pull_policy = ["if-not-present"]
```

### 3. 并发控制

```toml
# 全局并发数
concurrent = 10

[[runners]]
  # 单个 Runner 的并发限制
  limit = 3
```

## 监控和维护

### 查看 Runner 状态

```bash
# 查看 Runner 列表
sudo gitlab-runner list

# 查看 Runner 状态
sudo gitlab-runner status

# 查看 Runner 日志
sudo journalctl -u gitlab-runner -f
```

### 更新 Runner

```bash
# 停止 Runner
sudo gitlab-runner stop

# 更新 Runner
sudo apt-get update
sudo apt-get install gitlab-runner

# 启动 Runner
sudo gitlab-runner start
```

### 清理缓存

```bash
# 清理 Docker 缓存
docker system prune -a -f

# 清理 Runner 缓存
sudo rm -rf /cache/*
```

## 故障排查

### 问题 1: Runner 无法连接到 GitLab

```bash
# 检查网络连接
curl -I https://gitlab.com/

# 检查 Runner 配置
sudo gitlab-runner verify

# 重新注册 Runner
sudo gitlab-runner unregister --all-runners
sudo gitlab-runner register
```

### 问题 2: Docker 权限问题

```bash
# 将 gitlab-runner 用户添加到 docker 组
sudo usermod -aG docker gitlab-runner

# 重启 Runner
sudo gitlab-runner restart
```

### 问题 3: 磁盘空间不足

```bash
# 清理 Docker 资源
docker system prune -a -f --volumes

# 清理旧的镜像
docker image prune -a -f

# 清理构建缓存
docker builder prune -a -f
```

### 问题 4: Job 卡住不执行

```bash
# 检查 Runner 状态
sudo gitlab-runner status

# 查看 Runner 日志
sudo journalctl -u gitlab-runner -n 100

# 重启 Runner
sudo gitlab-runner restart
```

## 安全最佳实践

### 1. 使用专用 Runner

```yaml
# 为敏感项目使用专用 Runner
deploy-production:
  tags:
    - docker
    - production
    - dedicated  # 专用 Runner
```

### 2. 限制 Runner 访问

```toml
[[runners]]
  # 只允许受保护的分支使用
  access_level = "ref_protected"

  # 只允许特定项目使用
  locked = true
```

### 3. 使用 Secrets 管理

```yaml
# 不要在 .gitlab-ci.yml 中硬编码密钥
deploy:
  script:
    - echo $DEPLOY_KEY | base64 -d > deploy.key
  variables:
    DEPLOY_KEY: $DEPLOY_KEY  # 从 GitLab CI/CD Variables 获取
```

### 4. 定期更新

```bash
# 定期更新 Runner
sudo apt-get update && sudo apt-get upgrade gitlab-runner

# 定期更新 Docker
sudo apt-get update && sudo apt-get upgrade docker-ce
```

## 多项目共享 Runner

### 场景：团队有多个项目

```toml
concurrent = 10

# 共享 Runner - 所有项目都可以使用
[[runners]]
  name = "shared-docker-runner"
  url = "https://gitlab.com/"
  token = "SHARED_TOKEN"
  executor = "docker"
  tags = ["docker", "shared"]

  # 不锁定到特定项目
  locked = false

  [runners.docker]
    image = "docker:24.0.6"
    privileged = true
```

### 在 GitLab 中配置

1. **Group Runners**（推荐）：
   - Settings > CI/CD > Runners
   - 注册 Group Runner
   - 组内所有项目自动可用

2. **Shared Runners**：
   - Admin Area > Runners
   - 注册 Shared Runner
   - 所有项目都可以使用

3. **Specific Runners**：
   - Project Settings > CI/CD > Runners
   - 注册 Specific Runner
   - 只有该项目可以使用

## 总结

### ✅ 推荐配置

- **Executor**: Docker
- **标签**: docker, build, test
- **并发**: 根据服务器资源调整
- **缓存**: 启用 S3 或本地缓存
- **监控**: 定期检查 Runner 状态

### ❌ 不推荐

- Shell Executor（除非有特殊需求）
- 不使用标签（难以管理）
- 过高的并发数（资源不足）
- 不清理缓存（磁盘爆满）

### 📝 检查清单

- [ ] Runner 已安装并运行
- [ ] Docker 已安装并配置
- [ ] Runner 已注册到 GitLab
- [ ] 标签配置正确
- [ ] 缓存配置已启用
- [ ] 监控和日志已配置
- [ ] 安全策略已实施

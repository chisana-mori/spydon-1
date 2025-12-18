# Docker 多架构构建指南

本文档说明如何构建支持 `amd64` 和 `arm64` 架构的 Docker 镜像，以及如何通过 ConfigMap 注入配置。

## 📋 目录结构

```
apps/
├── backend/
│   ├── Dockerfile.multiarch     # 多架构 Dockerfile
│   ├── config/config.yaml       # 默认配置文件
│   └── scripts/build-multiarch.sh
├── frontend/
│   ├── Dockerfile.multiarch     # 多架构 Dockerfile
│   ├── config/runtime.json      # 默认运行时配置
│   ├── docker/entrypoint.sh     # 容器启动脚本
│   └── scripts/build-multiarch.sh
infrastructure/k8s/
├── backend-multiarch.yaml       # 后端 K8s 部署示例
└── frontend-multiarch.yaml      # 前端 K8s 部署示例
```

## 🚀 快速开始

### 前置条件

1. Docker 20.10+ (支持 buildx)
2. 启用 Docker Buildx 多架构支持

```bash
# 检查 buildx 版本
docker buildx version

# 创建多架构 builder (如果还没有)
docker buildx create --name multiarch-builder --use --driver docker-container
docker buildx inspect --bootstrap
```

### 构建后端镜像

```bash
cd apps/backend

# 本地构建 (仅当前架构)
./scripts/build-multiarch.sh

# 推送多架构镜像到仓库
REGISTRY=your-registry.com PUSH=true ./scripts/build-multiarch.sh

# 自定义版本号
VERSION=v1.0.0 REGISTRY=your-registry.com PUSH=true ./scripts/build-multiarch.sh
```

### 构建前端镜像

```bash
cd apps/frontend

# 本地构建 (仅当前架构)
./scripts/build-multiarch.sh

# 推送多架构镜像到仓库
REGISTRY=your-registry.com PUSH=true ./scripts/build-multiarch.sh

# 带构建时配置
NEXT_PUBLIC_BACKEND_BASE_URL=http://api.example.com \
NEXT_PUBLIC_BASE_PATH=/app \
REGISTRY=your-registry.com PUSH=true ./scripts/build-multiarch.sh
```

## ⚙️ 配置注入 (ConfigMap)

### 后端配置

后端通过 `CONFIG_FILE` 环境变量指定配置文件路径，默认为 `/app/config/config.yaml`。

**方式一：挂载 ConfigMap 为文件**

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: robusta-backend-config
data:
  config.yaml: |
    environment: production
    port: "8080"
    database_url: root:password@tcp(mysql:3306)/robusta_hub
    # ... 其他配置
---
apiVersion: apps/v1
kind: Deployment
spec:
  template:
    spec:
      containers:
      - name: backend
        env:
        - name: CONFIG_FILE
          value: /app/config/config.yaml
        volumeMounts:
        - name: config-volume
          mountPath: /app/config
      volumes:
      - name: config-volume
        configMap:
          name: robusta-backend-config
```

**方式二：通过环境变量覆盖**

后端支持通过环境变量覆盖配置文件中的值（前缀 `ROBUSTA_`）：

```yaml
env:
- name: ROBUSTA_DATABASE_URL
  value: "root:newpassword@tcp(mysql:3306)/robusta_hub"
- name: ROBUSTA_HOLMES_GPT__URL  # 嵌套配置使用双下划线
  value: "http://holmesgpt:8081"
```

### 前端配置

前端通过 `RUNTIME_CONFIG_PATH` 环境变量指定运行时配置路径，默认为 `/app/config/runtime.json`。

**方式一：挂载 ConfigMap 为文件**

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: robusta-frontend-config
data:
  runtime.json: |
    {
      "backendBaseUrl": "http://robusta-backend:8080",
      "casLoginPath": "/auth/cas/login",
      "casLogoutPath": "/auth/cas/logout",
      "basePath": ""
    }
---
apiVersion: apps/v1
kind: Deployment
spec:
  template:
    spec:
      containers:
      - name: frontend
        env:
        - name: RUNTIME_CONFIG_PATH
          value: /app/config/runtime.json
        volumeMounts:
        - name: config-volume
          mountPath: /app/config
      volumes:
      - name: config-volume
        configMap:
          name: robusta-frontend-config
```

**方式二：通过环境变量覆盖**

前端 entrypoint 脚本支持以下覆盖环境变量：

```yaml
env:
- name: OVERRIDE_BACKEND_URL
  value: "http://api.example.com"
- name: OVERRIDE_BASE_PATH
  value: "/spydon"
```

## 📦 部署示例

完整的 K8s 部署示例见：
- `infrastructure/k8s/backend-multiarch.yaml`
- `infrastructure/k8s/frontend-multiarch.yaml`

```bash
# 部署到 K8s
kubectl apply -f infrastructure/k8s/namespace.yaml
kubectl apply -f infrastructure/k8s/backend-multiarch.yaml
kubectl apply -f infrastructure/k8s/frontend-multiarch.yaml
```

## 🔧 构建参数

### 后端构建参数

| 环境变量 | 默认值 | 说明 |
|---------|--------|------|
| `REGISTRY` | (空) | 镜像仓库地址 |
| `IMAGE_NAME` | `robusta-backend` | 镜像名称 |
| `VERSION` | Git tag/dev | 版本号 |
| `PLATFORMS` | `linux/amd64,linux/arm64` | 目标平台 |
| `PUSH` | `false` | 是否推送到仓库 |
| `GOPROXY` | `https://proxy.golang.org,direct` | Go 模块代理 |

### 前端构建参数

| 环境变量 | 默认值 | 说明 |
|---------|--------|------|
| `REGISTRY` | (空) | 镜像仓库地址 |
| `IMAGE_NAME` | `robusta-frontend` | 镜像名称 |
| `VERSION` | Git tag/dev | 版本号 |
| `PLATFORMS` | `linux/amd64,linux/arm64` | 目标平台 |
| `PUSH` | `false` | 是否推送到仓库 |
| `NPM_REGISTRY` | `https://registry.npmjs.org` | NPM 镜像源 |
| `NEXT_PUBLIC_BACKEND_BASE_URL` | (空) | 构建时后端地址 |
| `NEXT_PUBLIC_BASE_PATH` | (空) | 构建时应用基础路径 |

## 🇨🇳 国内镜像源配置

在国内网络环境下构建时，建议使用国内镜像源加速依赖下载：

### 后端 (Go)

```bash
# 使用七牛云 GOPROXY
GOPROXY=https://goproxy.cn,direct ./scripts/build-multiarch.sh

# 或使用阿里云
GOPROXY=https://mirrors.aliyun.com/goproxy/,direct ./scripts/build-multiarch.sh

# 完整示例：国内镜像 + 推送到私有仓库
GOPROXY=https://goproxy.cn,direct \
REGISTRY=harbor.example.com/robusta \
PUSH=true ./scripts/build-multiarch.sh
```

### 前端 (NPM)

```bash
# 使用淘宝 NPM 镜像
NPM_REGISTRY=https://registry.npmmirror.com ./scripts/build-multiarch.sh

# 完整示例：国内镜像 + 推送到私有仓库
NPM_REGISTRY=https://registry.npmmirror.com \
REGISTRY=harbor.example.com/robusta \
PUSH=true ./scripts/build-multiarch.sh
```

### 常用国内镜像源

| 类型 | 镜像源 | 地址 |
|------|--------|------|
| Go (七牛) | GOPROXY | `https://goproxy.cn,direct` |
| Go (阿里) | GOPROXY | `https://mirrors.aliyun.com/goproxy/,direct` |
| NPM (淘宝) | NPM_REGISTRY | `https://registry.npmmirror.com` |
| NPM (华为) | NPM_REGISTRY | `https://mirrors.huaweicloud.com/repository/npm/` |

## 🔍 验证镜像

```bash
# 查看镜像支持的架构
docker buildx imagetools inspect your-registry.com/robusta-backend:latest

# 本地运行测试
docker run --rm -p 8080:8080 robusta-backend:latest
docker run --rm -p 3000:3000 robusta-frontend:latest
```

## 📝 注意事项

1. **本地加载限制**：`docker buildx build --load` 只支持单架构，多架构必须使用 `--push`
2. **配置热更新**：修改 ConfigMap 后需要重启 Pod 或使用配置管理工具
3. **敏感信息**：生产环境中敏感配置（如密码、密钥）应使用 Kubernetes Secret
4. **Next.js standalone**：前端使用 Next.js standalone 模式，减小镜像体积
5. **国内构建**：建议配置 GOPROXY 和 NPM_REGISTRY 使用国内镜像源加速

# 前端配置说明

## 配置文件

前端项目使用 JSON 配置文件进行配置，配置文件位于 `config/` 目录：

- `runtime.json` - 实际使用的配置文件（优先级最高）
- `runtime.sample.json` - 配置模板文件

## 配置项说明

```json
{
  "backendBaseUrl": "http://localhost:8080",  // 后端服务地址
  "casLoginPath": "/auth/cas/login",          // CAS 登录路径
  "casLogoutPath": "/auth/cas/logout",        // CAS 登出路径
  "basePath": ""                              // 应用部署的基础路径（如 /spydon）
}
```

### backendBaseUrl

后端服务的完整地址，包括协议、域名和端口。

- 开发环境：`http://localhost:8080`
- 生产环境：`https://your-domain.com` 或 `https://your-domain.com/spydon`

**注意**：如果后端部署在子路径下（如 `/spydon`），需要包含在 `backendBaseUrl` 中。

### basePath

前端应用部署的基础路径，用于支持子路径部署。

- 根路径部署：`""` 或不设置
- 子路径部署：`"/app"` 或 `"/spydon"`

## 使用方式

### 1. 开发环境

复制 `runtime.sample.json` 为 `runtime.json`：

```bash
cp config/runtime.sample.json config/runtime.json
```

修改 `runtime.json` 中的配置项，然后启动开发服务器：

```bash
npm run dev
```

### 2. 生产环境

#### 方式一：构建时配置

在构建前修改 `config/runtime.json`，然后执行构建：

```bash
npm run build
npm run start
```

#### 方式二：Docker 部署

在 Docker 容器中挂载配置文件：

```yaml
services:
  frontend:
    image: your-frontend-image
    volumes:
      - ./runtime.json:/app/config/runtime.json:ro
```

或使用环境变量覆盖（仍然支持）：

```yaml
services:
  frontend:
    image: your-frontend-image
    environment:
      - NEXT_PUBLIC_BACKEND_BASE_URL=https://your-domain.com
      - NEXT_PUBLIC_BASE_PATH=/spydon
```

## 配置优先级

配置加载的优先级从高到低：

1. 环境变量（`NEXT_PUBLIC_*`）
2. 运行时配置（`window.__ROBUSTA_RUNTIME_CONFIG__`）
3. JSON 配置文件（`runtime.json`）
4. 默认配置

这意味着环境变量可以覆盖 JSON 配置文件中的设置。

## 注意事项

1. `runtime.json` 不应提交到版本控制系统（已添加到 `.gitignore`）
2. 修改配置后需要重启开发服务器或重新构建
3. `backendBaseUrl` 会自动推导出 `apiBaseUrl`（`backendBaseUrl + /api/v1`）
4. 如果同时设置了 `backendBaseUrl` 和 `apiBaseUrl`，以 `apiBaseUrl` 为准

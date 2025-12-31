# Nginx 反向代理

## 配置说明

此 Nginx 容器将 `spydon.com` 的请求反向代理到本地服务：

| 路径 | 目标服务 |
|------|---------|
| `/` | localhost:3000 (前端) |
| `/api` | localhost:8080 (API) |
| `/kite` | localhost:9090 (Kite) |

## 本地访问配置

由于使用 `spydon.com` 作为 server_name，需要在 `/etc/hosts` 中添加：

```bash
# 添加到 /etc/hosts
127.0.0.1 spydon.com
```

## 访问地址

使用 **28080 端口**访问：

- http://spydon.com:28080/
- http://spydon.com:28080/api
- http://spydon.com:28080/kite

## 启动/停止容器

```bash
# 启动
docker compose -f infrastructure/nginx/docker-compose.yml up -d

# 停止
docker compose -f infrastructure/nginx/docker-compose.yml down

# 查看日志
docker logs spydon-nginx-proxy

# 重启
docker compose -f infrastructure/nginx/docker-compose.yml restart
```

## 注意事项

- 确保本地服务 (3000, 8080, 9090) 已启动
- `host.docker.internal` 在 OrbStack 中会自动解析到宿主机

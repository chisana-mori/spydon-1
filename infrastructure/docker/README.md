# Robusta Infrastructure Docker Compose

## 概述

此配置文件管理Robusta Hub项目的核心基础设施服务，包括数据库、对象存储和缓存服务。

## 服务架构

### 容器化服务（由Docker管理）

| 服务 | 端口 | 描述 | 状态 |
|------|------|------|------|
| PostgreSQL | 5432 | 主数据库 | ✅ 运行中 |
| MinIO | 9000, 9001 | 对象存储 | ✅ 运行中 |
| Redis | 6379 | 缓存和会话存储 | ✅ 运行中 |

### 外部服务（本地进程）

| 服务 | 端口 | 描述 | 启动方式 |
|------|------|------|---------|
| Navy Service | 8081 | Navy应用服务 | 本地进程 |
| Wayne Service | 8082 | Wayne应用服务 | 本地进程 |
| Wayne API | 8083 | Wayne API服务 | 本地进程 |

## 快速启动

### 启动容器化服务

```bash
# 启动所有容器化服务
docker-compose up -d

# 查看服务状态
docker-compose ps

# 查看日志
docker-compose logs -f
```

### 启动外部服务

外部服务需要在宿主机上手动启动：

```bash
# 启动Navy服务（端口8081）
# 根据实际的启动命令执行

# 启动Wayne服务（端口8082）
# 根据实际的启动命令执行

# 启动Wayne API（端口8083）
# 根据实际的启动命令执行
```

## 服务详情

### PostgreSQL
- **镜像**: `postgres:15-alpine`
- **数据库**: `robusta_hub`
- **用户**: `postgres`
- **密码**: `password`
- **持久化**: `postgres_data` volume
- **健康检查**: PostgreSQL连接检查

### MinIO
- **镜像**: `minio/minio:latest`
- **访问地址**:
  - API: http://localhost:9000
  - 控制台: http://localhost:9001
- **认证**: minioadmin / minioadmin
- **持久化**: `minio_data` volume
- **健康检查**: MinIO健康检查

### Redis
- **镜像**: `redis:7-alpine`
- **持久化**: `redis_data` volume
- **健康检查**: Redis ping检查

## 网络配置

### Docker网络
- **网络名称**: `robusta-network`
- **驱动**: `bridge`
- **子网**: 自动分配

### 端口映射
- PostgreSQL: `5432:5432`
- MinIO API: `9000:9000`
- MinIO 控制台: `9001:9001`
- Redis: `6379:6379`

## 数据持久化

所有数据都通过Docker volumes持久化存储：

- `postgres_data`: PostgreSQL数据文件
- `minio_data`: MinIO对象存储数据
- `redis_data`: Redis数据文件

## 健康检查

每个容器都配置了相应的健康检查：

- **PostgreSQL**: `pg_isready -U postgres`
- **MinIO**: `curl -f http://localhost:9000/minio/health/live`
- **Redis**: `redis-cli ping`

## 环境变量

### PostgreSQL
- `POSTGRES_DB`: robusta_hub
- `POSTGRES_USER`: postgres
- `POSTGRES_PASSWORD`: password

### MinIO
- `MINIO_ROOT_USER`: minioadmin
- `MINIO_ROOT_PASSWORD`: minioadmin

## 维护操作

### 备份数据
```bash
# 备份PostgreSQL
docker exec robusta-postgres pg_dump -U postgres robusta_hub > backup.sql

# 备份MinIO数据（需要MinIO客户端）
# 使用minio客户端工具备份桶数据

# 备份Redis数据
docker exec robusta-redis redis-cli BGSAVE
```

### 日志管理
```bash
# 查看所有服务日志
docker-compose logs

# 查看特定服务日志
docker-compose logs postgres
docker-compose logs minio
docker-compose logs redis
```

### 服务重启
```bash
# 重启所有服务
docker-compose restart

# 重启特定服务
docker-compose restart postgres
```

## 故障排除

### 常见问题

1. **端口冲突**
   - 确保5432、9000、9001、6379端口未被占用
   - 检查外部服务是否占用相同端口

2. **权限问题**
   - 确保Docker有足够权限访问挂载的目录
   - 检查文件权限设置

3. **健康检查失败**
   - 检查服务是否正常启动
   - 查看具体错误日志

4. **外部服务连接问题**
   - 确保外部服务在对应端口上运行
   - 检查防火墙设置

## 开发环境设置

### 数据库连接
```bash
# PostgreSQL连接信息
Host: localhost
Port: 5432
Database: robusta_hub
Username: postgres
Password: password
```

### MinIO配置
```bash
# MinIO连接信息
Endpoint: http://localhost:9000
Access Key: minioadmin
Secret Key: minioadmin
```

### Redis配置
```bash
# Redis连接信息
Host: localhost
Port: 6379
```

## 生产环境注意事项

1. **安全性**
   - 更改默认密码
   - 限制网络访问
   - 启用TLS/SSL

2. **备份策略**
   - 定期备份PostgreSQL数据
   - 备份MinIO对象存储
   - 设置监控和告警

3. **性能优化**
   - 调整数据库配置
   - 优化Redis内存设置
   - 监控资源使用情况

## 版本信息

- PostgreSQL: 15-alpine
- MinIO: latest
- Redis: 7-alpine
- Docker Compose: 3.8
# 多阶段构建：优化的后端 Dockerfile
# 使用 buildx 和 BuildKit 缓存优化

# 构建阶段 - 基础依赖层
FROM golang:1.23-alpine AS base
WORKDIR /app
RUN apk add --no-cache git ca-certificates tzdata curl

# 依赖层 - 缓存 Go modules
FROM base AS deps
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download && go mod verify

# 构建阶段
FROM base AS builder
COPY --from=deps /go/pkg/mod /go/pkg/mod
COPY --from=deps /root/.cache/go-build /root/.cache/go-build
COPY . .

# 构建参数
ARG VERSION=dev
ARG COMMIT_SHA=unknown
ARG BUILD_TIME=unknown

# 设置构建信息
RUN echo "Version: ${VERSION}" > /tmp/buildinfo && \
    echo "Commit: ${COMMIT_SHA}" >> /tmp/buildinfo && \
    echo "Build time: ${BUILD_TIME}" >> /tmp/buildinfo

# 构建应用 - 使用缓存加速
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.version=${VERSION} -X main.commit=${COMMIT_SHA} -X main.buildTime=${BUILD_TIME}" \
    -a -installsuffix cgo \
    -o main ./cmd/server

# 运行阶段 - 最小化镜像
FROM alpine:3.18 AS runtime

# 安装运行时依赖
RUN apk --no-cache add ca-certificates tzdata curl && \
    addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup && \
    rm -rf /var/cache/apk/*

# 设置时区
ENV TZ=UTC

# 复制 CA 证书
COPY --from=base /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# 复制构建产物
COPY --from=builder /app/main /app/main
COPY --from=builder /app/migrations /app/migrations
COPY --from=builder /tmp/buildinfo /app/buildinfo

# 设置权限
RUN chown -R appuser:appgroup /app && \
    chmod +x /app/main

# 切换到非 root 用户
USER appuser
WORKDIR /app

# 健康检查
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# 暴露端口
EXPOSE 8080

# 安全相关配置
ENV ENVIRONMENT=production
ENV GIN_MODE=release

# 运行应用
ENTRYPOINT ["/app/main"]

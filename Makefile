PRE_COMMIT=uvx pre-commit

# ============== Pre-commit hooks ==============

setup-precommit:
	$(PRE_COMMIT) install --install-hooks
	$(PRE_COMMIT) install --hook-type commit-msg
	@echo "pre-commit 已启用（uvx 运行，无需额外安装）"

pre-commit:
	$(PRE_COMMIT) run --all-files

# ============== 后端开发 targets ==============

# 后端 lint 检查
lint-backend:
	@cd apps/backend && golangci-lint run

# 后端 lint 自动修复
lint-fix-backend:
	@cd apps/backend && bash scripts/fix-lint.sh

# 后端测试
test-backend:
	@cd apps/backend && go test ./...

# 后端构建（本地开发）
build-backend:
	@cd apps/backend && go build -o bin/server ./cmd/server

# 后端运行
run-backend:
	@cd apps/backend && go run ./cmd/server

# ============== 前端开发 targets ==============

# 前端 lint 检查
lint-frontend:
	@cd apps/frontend && bun lint

# 前端 lint 自动修复
lint-fix-frontend:
	@cd apps/frontend && bun lint --fix

# 前端构建
build-frontend:
	@cd apps/frontend && bun run build

# 前端开发服务器
dev-frontend:
	@cd apps/frontend && bun run dev

# ============== 全局 lint targets ==============

# 全部 lint 检查
lint: lint-backend lint-frontend
	@echo "✅ All lint checks completed"

# 全部 lint 自动修复
lint-fix: lint-fix-backend lint-fix-frontend
	@echo "✅ All lint fixes applied"

# ============== Docker 多架构构建 targets ==============

# 后端镜像 - 本地构建（最简单，使用 docker build）
docker-build-backend-local:
	@cd apps/backend && BUILD_MODE=local bash scripts/build-multiarch.sh

# 后端镜像 - buildx 构建（支持多平台）
docker-build-backend:
	@cd apps/backend && bash scripts/build-multiarch.sh

# 后端镜像 - 推送到仓库
docker-push-backend:
	@cd apps/backend && PUSH=true bash scripts/build-multiarch.sh

# 前端镜像 - 本地构建（最简单，使用 docker build）
docker-build-frontend-local:
	@cd apps/frontend && BUILD_MODE=local bash scripts/build-multiarch.sh

# 前端镜像 - buildx 构建（支持多平台）
docker-build-frontend:
	@cd apps/frontend && bash scripts/build-multiarch.sh

# 前端镜像 - 推送到仓库
docker-push-frontend:
	@cd apps/frontend && PUSH=true bash scripts/build-multiarch.sh

# 单架构本地加载 - 后端 (自动检测平台)
docker-load-backend:
	@cd apps/backend && LOAD=true PLATFORM=$$(uname -m | sed 's/x86_64/linux\/amd64/' | sed 's/arm64/linux\/arm64/' | sed 's/aarch64/linux\/arm64/') bash scripts/build-multiarch.sh

# 单架构本地加载 - 前端 (自动检测平台)
docker-load-frontend:
	@cd apps/frontend && LOAD=true PLATFORM=$$(uname -m | sed 's/x86_64/linux\/amd64/' | sed 's/arm64/linux\/arm64/' | sed 's/aarch64/linux\/arm64/') bash scripts/build-multiarch.sh

# 单架构本地加载 - 后端 amd64
docker-load-backend-amd64:
	@cd apps/backend && PLATFORM=linux/amd64 LOAD=true bash scripts/build-multiarch.sh

# 单架构本地加载 - 前端 amd64
docker-load-frontend-amd64:
	@cd apps/frontend && PLATFORM=linux/amd64 LOAD=true bash scripts/build-multiarch.sh

# 单架构本地加载 - 后端 arm64
docker-load-backend-arm64:
	@cd apps/backend && PLATFORM=linux/arm64 LOAD=true bash scripts/build-multiarch.sh

# 单架构本地加载 - 前端 arm64
docker-load-frontend-arm64:
	@cd apps/frontend && PLATFORM=linux/arm64 LOAD=true bash scripts/build-multiarch.sh

# 构建所有镜像（后端 + 前端，buildx）
docker-build-all: docker-build-backend docker-build-frontend
	@echo "✅ All images built successfully"

# 构建所有镜像（本地模式）
docker-build-all-local: docker-build-backend-local docker-build-frontend-local
	@echo "✅ All images built successfully (local mode)"

# 推送所有镜像到仓库
docker-push-all: docker-push-backend docker-push-frontend
	@echo "✅ All images pushed successfully"

# 本地加载所有镜像（自动检测平台）
docker-load-all: docker-load-backend docker-load-frontend
	@echo "✅ All images loaded locally"

# 本地加载所有镜像 (amd64)
docker-load-all-amd64: docker-load-backend-amd64 docker-load-frontend-amd64
	@echo "✅ All images loaded locally (amd64)"

# 本地加载所有镜像 (arm64)
docker-load-all-arm64: docker-load-backend-arm64 docker-load-frontend-arm64
	@echo "✅ All images loaded locally (arm64)"

# 显示 Docker buildx 构建器信息
docker-buildx-info:
	docker buildx ls
	@echo ""
	@echo "💡 To create a builder instance, run:"
	@echo "   docker buildx create --name multiarch-builder --driver docker-container"

# ============== 帮助信息 ==============

help:
	@echo ""
	@echo "🛠️  Robusta Web Makefile Help"
	@echo "=============================="
	@echo ""
	@echo "📝 开发命令:"
	@echo "   make lint                   # 运行所有 lint 检查"
	@echo "   make lint-fix               # 自动修复 lint 问题"
	@echo "   make lint-backend           # 后端 lint 检查"
	@echo "   make lint-fix-backend       # 后端 lint 自动修复"
	@echo "   make lint-frontend          # 前端 lint 检查"
	@echo "   make lint-fix-frontend      # 前端 lint 自动修复"
	@echo "   make test-backend           # 后端测试"
	@echo "   make build-backend          # 后端本地编译"
	@echo "   make run-backend            # 运行后端服务"
	@echo "   make build-frontend         # 前端构建"
	@echo "   make dev-frontend           # 前端开发服务器"
	@echo ""
	@echo "🐳 Docker 构建命令:"
	@echo ""
	@echo "   📦 本地构建 (简单模式，使用 docker build):"
	@echo "      make docker-build-backend-local   # 后端"
	@echo "      make docker-build-frontend-local  # 前端"
	@echo "      make docker-build-all-local       # 全部"
	@echo ""
	@echo "   🔧 Buildx 构建 (支持多平台):"
	@echo "      make docker-build-backend         # 后端"
	@echo "      make docker-build-frontend        # 前端"
	@echo "      make docker-build-all             # 全部"
	@echo ""
	@echo "   📥 本地加载镜像 (自动检测平台):"
	@echo "      make docker-load-backend          # 后端"
	@echo "      make docker-load-frontend         # 前端"
	@echo "      make docker-load-all              # 全部"
	@echo ""
	@echo "   📥 本地加载镜像 (指定架构):"
	@echo "      make docker-load-backend-amd64    # 后端 amd64"
	@echo "      make docker-load-backend-arm64    # 后端 arm64"
	@echo "      make docker-load-frontend-amd64   # 前端 amd64"
	@echo "      make docker-load-frontend-arm64   # 前端 arm64"
	@echo "      make docker-load-all-amd64        # 全部 amd64"
	@echo "      make docker-load-all-arm64        # 全部 arm64"
	@echo ""
	@echo "   📤 推送到 Registry:"
	@echo "      REGISTRY=xxx make docker-push-backend"
	@echo "      REGISTRY=xxx make docker-push-frontend"
	@echo "      REGISTRY=xxx make docker-push-all"
	@echo ""
	@echo "   📊 信息:"
	@echo "      make docker-buildx-info           # 查看 buildx 信息"
	@echo ""
	@echo "⚙️  配置变量:"
	@echo "   REGISTRY=...          # Docker registry URL"
	@echo "   IMAGE_NAME=...        # 镜像名称"
	@echo "   VERSION=...           # 版本号 (默认: git describe)"
	@echo "   PLATFORM=linux/arm64  # 单平台构建"
	@echo "   PLATFORMS=...         # 多平台列表 (默认: linux/amd64,linux/arm64)"
	@echo "   BUILD_MODE=local      # 构建模式 (buildx/local/auto)"
	@echo "   BUILDER=orbstack      # Builder 选择 (auto/orbstack/default/container)"
	@echo ""
	@echo "🔧 Pre-commit:"
	@echo "   make setup-precommit  # 安装 pre-commit hooks"
	@echo "   make pre-commit       # 运行 pre-commit 检查"
	@echo ""

# 保留旧的 docker-help 作为别名
docker-help: help

.PHONY: setup-precommit pre-commit \
        lint-backend lint-fix-backend test-backend build-backend run-backend \
        lint-frontend lint-fix-frontend build-frontend dev-frontend \
        lint lint-fix \
        docker-build-backend-local docker-build-backend docker-push-backend \
        docker-build-frontend-local docker-build-frontend docker-push-frontend \
        docker-load-backend docker-load-frontend \
        docker-load-backend-amd64 docker-load-frontend-amd64 \
        docker-load-backend-arm64 docker-load-frontend-arm64 \
        docker-build-all docker-build-all-local docker-push-all \
        docker-load-all docker-load-all-amd64 docker-load-all-arm64 \
        docker-buildx-info help docker-help

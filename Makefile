.PHONY: help build run test clean docker-build docker-run dev-setup

# 默认目标
help:
	@echo "可用的命令:"
	@echo "  dev-setup     - 设置开发环境"
	@echo "  build         - 构建后端应用"
	@echo "  run           - 运行后端应用"
	@echo "  test          - 运行测试"
	@echo "  clean         - 清理构建文件"
	@echo "  docker-build  - 构建Docker镜像"
	@echo "  docker-run    - 运行Docker容器"
	@echo "  frontend-dev  - 启动前端开发服务器"
	@echo "  frontend-build - 构建前端应用"

# 开发环境设置
dev-setup:
	@echo "设置开发环境..."
	cd backend && cp .env.example .env
	@echo "请编辑 backend/.env 文件配置数据库连接"
	@echo "然后运行: make docker-run 启动依赖服务"

# 构建后端应用
build:
	@echo "构建后端应用..."
	cd backend && go build -o bin/server ./cmd/server

# 运行后端应用
run:
	@echo "运行后端应用..."
	cd backend && go run ./cmd/server

# 运行测试
test:
	@echo "运行后端测试..."
	cd backend && go test -v ./...

# 清理构建文件
clean:
	@echo "清理构建文件..."
	cd backend && rm -rf bin/
	cd frontend && rm -rf dist/

# 构建Docker镜像
docker-build:
	@echo "构建Docker镜像..."
	docker-compose build

# 运行Docker容器
docker-run:
	@echo "启动Docker容器..."
	docker-compose up -d

# 停止Docker容器
docker-stop:
	@echo "停止Docker容器..."
	docker-compose down

# 查看日志
docker-logs:
	@echo "查看Docker容器日志..."
	docker-compose logs -f

# 前端开发服务器
frontend-dev:
	@echo "启动前端开发服务器..."
	cd frontend && npm run dev

# 构建前端应用
frontend-build:
	@echo "构建前端应用..."
	cd frontend && npm run build

# 安装前端依赖
frontend-install:
	@echo "安装前端依赖..."
	cd frontend && npm install

# 后端依赖管理
backend-deps:
	@echo "下载后端依赖..."
	cd backend && go mod download

# 格式化代码
format:
	@echo "格式化Go代码..."
	cd backend && go fmt ./...
	@echo "格式化前端代码..."
	cd frontend && npm run lint:fix

# 数据库迁移
db-migrate:
	@echo "运行数据库迁移..."
	cd backend && go run ./cmd/migrate

# 生成API文档
docs:
	@echo "生成API文档..."
	# 这里可以添加Swagger文档生成命令

# 完整的开发环境启动
dev: docker-run
	@echo "等待数据库启动..."
	sleep 10
	@echo "启动后端服务..."
	cd backend && go run ./cmd/server &
	@echo "启动前端开发服务器..."
	cd frontend && npm run dev

# 生产环境部署
deploy:
	@echo "部署到生产环境..."
	docker-compose -f docker-compose.prod.yml up -d

# 健康检查
health:
	@echo "检查服务健康状态..."
	curl -f http://localhost:8080/health || echo "后端服务不可用"
	curl -f http://localhost:3000/health || echo "前端服务不可用"

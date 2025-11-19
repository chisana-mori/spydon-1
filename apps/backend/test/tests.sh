#!/bin/bash

# 后端测试脚本
# 运行单元测试、集成测试和性能测试

set -euo pipefail

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 配置
TEST_TIMEOUT="${TEST_TIMEOUT:-30s}"
COVERAGE_THRESHOLD="${COVERAGE_THRESHOLD:-70}"
RACE_DETECTOR="${RACE_DETECTOR:-true}"

log() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

info() {
    echo -e "${BLUE}[TEST]${NC} $1"
}

# 检查测试环境
check_test_env() {
    log "Checking test environment..."

    # 检查 Go 环境
    if ! command -v go &> /dev/null; then
        error "Go is not installed"
    fi

    # 检查必要的工具
    local tools=("curl" "jq")
    for tool in "${tools[@]}"; do
        if ! command -v "$tool" &> /dev/null; then
            warn "$tool is not installed, some tests may not work"
        fi
    done

    # 设置测试环境变量
    export CGO_ENABLED=1
    export GO111MODULE=on
    export TEST_ENV=true
    export DATABASE_URL="${DATABASE_URL:-postgres://postgres:postgres@localhost:5432/test_db?sslmode=disable}"
    export REDIS_URL="${REDIS_URL:-redis://localhost:6379}"
}

# 启动测试依赖
start_test_dependencies() {
    log "Starting test dependencies..."

    # 检查 Docker 是否运行
    if ! docker info &> /dev/null; then
        error "Docker is not running"
    fi

    # 启动 PostgreSQL 和 Redis（如果需要）
    if ! curl -s "$DATABASE_URL" &> /dev/null; then
        log "Starting PostgreSQL for tests..."
        docker run -d --name test-postgres \
            -e POSTGRES_DB=test_db \
            -e POSTGRES_USER=postgres \
            -e POSTGRES_PASSWORD=postgres \
            -p 5432:5432 \
            postgres:15-alpine || true

        # 等待 PostgreSQL 启动
        local retries=0
        while ! curl -s "$DATABASE_URL" &> /dev/null && [ $retries -lt 30 ]; do
            sleep 2
            ((retries++))
        done

        if [ $retries -eq 30 ]; then
            error "Failed to start PostgreSQL"
        fi
    fi

    if ! redis-cli -u "$REDIS_URL" ping &> /dev/null; then
        log "Starting Redis for tests..."
        docker run -d --name test-redis \
            -p 6379:6379 \
            redis:7-alpine || true

        # 等待 Redis 启动
        local retries=0
        while ! redis-cli -u "$REDIS_URL" ping &> /dev/null && [ $retries -lt 30 ]; do
            sleep 2
            ((retries++))
        done

        if [ $retries -eq 30 ]; then
            error "Failed to start Redis"
        fi
    fi
}

# 运行数据库迁移
run_migrations() {
    log "Running database migrations..."

    # 检查是否有迁移文件
    if [ -d "migrations" ]; then
        go run ./cmd/migrate/main.go up || {
            warn "Migration failed, continuing with tests..."
        }
    else
        warn "No migrations directory found, skipping migrations"
    fi
}

# 运行单元测试
run_unit_tests() {
    info "Running unit tests..."

    local test_args=(
        "-v"
        "-race=$RACE_DETECTOR"
        "-timeout=$TEST_TIMEOUT"
        "-coverprofile=coverage.out"
        "-covermode=atomic"
        "./..."
    )

    # 排除集成测试
    if [ -d "tests/integration" ]; then
        test_args+=("-skip=^tests/integration/")
    fi

    go test "${test_args[@]}"

    log "Unit tests completed"
    log "Coverage report: coverage.out"
}

# 运行集成测试
run_integration_tests() {
    if [ ! -d "tests/integration" ]; then
        warn "No integration tests found"
        return 0
    fi

    info "Running integration tests..."

    cd tests/integration

    # 运行集成测试
    go test -v -timeout=60s ./...

    cd - > /dev/null

    log "Integration tests completed"
}

# 运行性能测试
run_performance_tests() {
    if [ ! -d "tests/performance" ]; then
        warn "No performance tests found"
        return 0
    fi

    info "Running performance tests..."

    cd tests/performance

    # 运行性能基准测试
    go test -bench=. -benchmem -run=^$ ./...

    cd - > /dev/null

    log "Performance tests completed"
}

# 生成测试报告
generate_reports() {
    log "Generating test reports..."

    # 生成覆盖率报告
    if [ -f "coverage.out" ]; then
        go tool cover -html=coverage.out -o coverage.html
        log "HTML coverage report: coverage.html"

        # 检查覆盖率阈值
        local coverage_percent
        coverage_percent=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')

        if (( $(echo "$coverage_percent < $COVERAGE_THRESHOLD" | bc -l) )); then
            warn "Coverage ($coverage_percent%) is below threshold ($COVERAGE_THRESHOLD%)"
        else
            log "Coverage ($coverage_percent%) meets threshold ($COVERAGE_THRESHOLD%)"
        fi
    fi

    # 生成测试报告（如果使用 gotestsum）
    if command -v gotestsum &> /dev/null && [ -f "test.json" ]; then
        gotestsum --junitfile test-results.xml --format testname < test.json
        log "JUnit report: test-results.xml"
    fi
}

# 清理测试环境
cleanup() {
    log "Cleaning up test environment..."

    # 停止测试容器
    docker stop test-postgres test-redis 2>/dev/null || true
    docker rm test-postgres test-redis 2>/dev/null || true

    # 清理临时文件
    rm -f test.tmp test.json

    log "Cleanup completed"
}

# 显示帮助信息
show_help() {
    cat << EOF
Usage: $0 [OPTIONS]

Options:
  -u, --unit-only        Run only unit tests
  -i, --integration-only Run only integration tests
  -p, --performance-only Run only performance tests
  -c, --coverage-only    Run only coverage analysis
  --skip-deps           Skip dependency checks and setup
  --skip-migrations     Skip database migrations
  -t, --timeout TIME    Set test timeout (default: 30s)
  -r, --race RACE       Enable/disable race detector (default: true)
  --coverage-threshold  Set coverage threshold (default: 70)
  --clean-only          Only cleanup test environment
  -h, --help            Show this help message

Examples:
  $0                    Run all tests
  $0 -u                 Run only unit tests
  $0 --skip-deps        Run tests without dependency setup
  $0 --clean-only       Cleanup test environment
EOF
}

# 主函数
main() {
    local unit_only=false
    local integration_only=false
    local performance_only=false
    local coverage_only=false
    local skip_deps=false
    local skip_migrations=false

    # 解析命令行参数
    while [[ $# -gt 0 ]]; do
        case $1 in
            -u|--unit-only)
                unit_only=true
                shift
                ;;
            -i|--integration-only)
                integration_only=true
                shift
                ;;
            -p|--performance-only)
                performance_only=true
                shift
                ;;
            -c|--coverage-only)
                coverage_only=true
                shift
                ;;
            --skip-deps)
                skip_deps=true
                shift
                ;;
            --skip-migrations)
                skip_migrations=true
                shift
                ;;
            -t|--timeout)
                TEST_TIMEOUT="$2"
                shift 2
                ;;
            -r|--race)
                RACE_DETECTOR="$2"
                shift 2
                ;;
            --coverage-threshold)
                COVERAGE_THRESHOLD="$2"
                shift 2
                ;;
            --clean-only)
                cleanup
                exit 0
                ;;
            -h|--help)
                show_help
                exit 0
                ;;
            *)
                error "Unknown option: $1"
                ;;
        esac
    done

    # 设置错误处理
    trap cleanup EXIT

    log "Starting backend test suite..."

    # 检查环境
    if [ "$skip_deps" = false ]; then
        check_test_env
        start_test_dependencies
    fi

    # 运行迁移
    if [ "$skip_migrations" = false ]; then
        run_migrations
    fi

    # 运行测试
    if [ "$coverage_only" = true ]; then
        run_unit_tests
        generate_reports
        exit 0
    fi

    if [ "$unit_only" = true ]; then
        run_unit_tests
        generate_reports
        exit 0
    fi

    if [ "$integration_only" = true ]; then
        run_integration_tests
        exit 0
    fi

    if [ "$performance_only" = true ]; then
        run_performance_tests
        exit 0
    fi

    # 运行所有测试
    run_unit_tests
    run_integration_tests
    run_performance_tests
    generate_reports

    log "All tests completed successfully!"
}

# 执行主函数
main "$@"

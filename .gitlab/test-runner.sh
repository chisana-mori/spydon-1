#!/bin/bash

# GitLab CI 测试运行脚本
# 适配 GitLab CI 环境的自动化测试

set -euo pipefail

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

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

# GitLab CI 变量
CI_PROJECT_ID="${CI_PROJECT_ID:-}"
CI_PIPELINE_ID="${CI_PIPELINE_ID:-}"
CI_JOB_ID="${CI_JOB_ID:-}"
CI_COMMIT_SHA="${CI_COMMIT_SHA:-}"
CI_COMMIT_REF_SLUG="${CI_COMMIT_REF_SLUG:-}"
CI_MERGE_REQUEST_IID="${CI_MERGE_REQUEST_IID:-}"

# 测试配置
TEST_TYPE="${TEST_TYPE:-}"  # backend, frontend, integration, e2e
TEST_TIMEOUT="${TEST_TIMEOUT:-300}"
COVERAGE_THRESHOLD="${COVERAGE_THRESHOLD:-70}"
GENERATE_REPORTS="${GENERATE_REPORTS:-true}"

# 数据库配置
DATABASE_URL="${DATABASE_URL:-postgres://postgres:postgres@postgres:5432/test_db?sslmode=disable}"
REDIS_URL="${REDIS_URL:-redis://redis:6379}"

# 输出目录
REPORTS_DIR="${CI_PROJECT_DIR:-}/test-reports"
mkdir -p "$REPORTS_DIR"

# 检查测试环境
check_test_env() {
    log "Checking test environment..."

    # 检查 GitLab CI 环境
    if [ -z "$CI_PROJECT_ID" ]; then
        warn "Not running in GitLab CI, using local environment"
    else
        log "Running in GitLab CI (Project: $CI_PROJECT_ID, Pipeline: $CI_PIPELINE_ID)"
    fi

    # 检查必要的工具
    local required_tools=("go" "node" "npm")
    for tool in "${required_tools[@]}"; do
        if ! command -v "$tool" &> /dev/null; then
            error "$tool is not installed"
            exit 1
        fi
    done

    log "Test environment verified"
}

# 设置测试环境变量
setup_env() {
    log "Setting up test environment variables..."

    # 通用测试变量
    export CI=true
    export NODE_ENV=test
    export ENVIRONMENT=test
    export LOG_LEVEL=debug

    # 后端测试变量
    export DATABASE_URL="$DATABASE_URL"
    export REDIS_URL="$REDIS_URL"
    export JWT_SECRET="test-jwt-secret"
    export HMAC_SECRET="test-hmac-secret"

    # 前端测试变量
    export NEXT_PUBLIC_API_URL="${NEXT_PUBLIC_API_URL:-http://localhost:8080}"
    export NEXT_PUBLIC_CI="true"
    export NEXT_PUBLIC_TEST_MODE="true"

    # 测试覆盖率变量
    export GO_COVERAGE_FILE="$REPORTS_DIR/backend-coverage.out"
    export GO_COVERAGE_HTML="$REPORTS_DIR/backend-coverage.html"
    export NPM_COVERAGE_DIR="$REPORTS_DIR/frontend-coverage"

    log "Test environment variables set"
}

# 运行后端测试
run_backend_tests() {
    log "Running backend tests..."

    cd apps/backend

    # 下载依赖
    log "Downloading Go dependencies..."
    go mod download
    go mod verify

    # 运行单元测试
    log "Running Go unit tests..."
    local test_args=(
        "-v"
        "-race"
        "-timeout=${TEST_TIMEOUT}s"
        "-coverprofile=$GO_COVERAGE_FILE"
        "-covermode=atomic"
        "-coverpkg=./..."
        "./..."
    )

    # 生成 JUnit XML 报告
    if command -v gotest &> /dev/null; then
        test_args+=("-json")
        gotest "${test_args[@]}" 2>&1 | tee "$REPORTS_DIR/backend-test.log"

        # 生成 JUnit 报告
        if command -v gotest &> /dev/null; then
            gotest -junitreport -v ./... > "$REPORTS_DIR/backend-junit.xml" 2>/dev/null || true
        fi
    else
        # 安装 gotest 工具
        log "Installing gotest tool..."
        go install github.com/haveyoudebuggedit/gotest@latest

        "${GOBIN:-$HOME/go/bin}/gotest" "${test_args[@]}" 2>&1 | tee "$REPORTS_DIR/backend-test.log"
        "${GOBIN:-$HOME/go/bin}/gotest" -junitreport -v ./... > "$REPORTS_DIR/backend-junit.xml" 2>/dev/null || true
    fi

    # 生成 HTML 覆盖率报告
    log "Generating backend coverage report..."
    go tool cover -html="$GO_COVERAGE_FILE" -o "$GO_COVERAGE_HTML"

    # 生成 Cobertura 格式报告（GitLab CI 支持）
    go tool cover -xml="$GO_COVERAGE_FILE" > "$REPORTS_DIR/backend-cobertura.xml"

    # 检查覆盖率阈值
    local coverage_percent
    coverage_percent=$(go tool cover -func="$GO_COVERAGE_FILE" | grep "total:" | awk '{print $3}' | sed 's/%//')

    if (( $(echo "$coverage_percent < $COVERAGE_THRESHOLD" | bc -l) )); then
        warn "Backend coverage ($coverage_percent%) is below threshold ($COVERAGE_THRESHOLD%)"
    else
        log "Backend coverage ($coverage_percent%) meets threshold ($COVERAGE_THRESHOLD%)"
    fi

    cd - > /dev/null
    log "Backend tests completed"
}

# 运行前端测试
run_frontend_tests() {
    log "Running frontend tests..."

    cd apps/frontend

    # 安装依赖
    log "Installing Node.js dependencies..."
    npm ci --cache .npm --prefer-offline

    # 运行单元测试
    log "Running Node.js unit tests..."

    # 生成测试报告
    if [ "$GENERATE_REPORTS" = "true" ]; then
        npm run test:coverage 2>&1 | tee "$REPORTS_DIR/frontend-test.log"
    else
        npm test 2>&1 | tee "$REPORTS_DIR/frontend-test.log"
    fi

    # 生成 JUnit 报告（如果配置了）
    if [ -f "junit.xml" ]; then
        cp junit.xml "$REPORTS_DIR/frontend-junit.xml"
    fi

    # 生成覆盖率报告
    if [ -d "coverage" ]; then
        cp -r coverage "$NPM_COVERAGE_DIR"

        # 生成 Cobertura 格式报告
        if [ -f "coverage/cobertura-coverage.xml" ]; then
            cp coverage/cobertura-coverage.xml "$REPORTS_DIR/frontend-cobertura.xml"
        fi
    fi

    cd - > /dev/null
    log "Frontend tests completed"
}

# 运行集成测试
run_integration_tests() {
    log "Running integration tests..."

    # 等待服务启动
    log "Waiting for services to start..."
    local retries=0
    local max_retries=30

    while [ $retries -lt $max_retries ]; do
        if curl -f "$DATABASE_URL" &> /dev/null && curl -f "$REDIS_URL/ping" &> /dev/null; then
            log "Services are ready"
            break
        fi

        if [ $retries -eq $((max_retries - 1)) ]; then
            error "Timeout waiting for services to start"
            exit 1
        fi

        sleep 5
        ((retries++))
        info "Waiting for services... ($retries/$max_retries)"
    done

    # 运行后端集成测试
    if [ -d "apps/backend/tests/integration" ]; then
        log "Running backend integration tests..."
        cd apps/backend

        go test -v -timeout=600s ./tests/integration/... 2>&1 | tee "$REPORTS_DIR/backend-integration-test.log"

        cd - > /dev/null
    fi

    # 运行前端集成测试
    if [ -f "apps/frontend/tests/integration.spec.ts" ]; then
        log "Running frontend integration tests..."
        cd apps/frontend

        npx playwright test --config=playwright.config.integration.ts 2>&1 | tee "$REPORTS_DIR/frontend-integration-test.log"

        cd - > /dev/null
    fi

    log "Integration tests completed"
}

# 运行 E2E 测试
run_e2e_tests() {
    log "Running E2E tests..."

    if [ ! -d "apps/frontend/tests/e2e" ]; then
        warn "E2E tests directory not found, skipping"
        return 0
    fi

    cd apps/frontend

    # 安装 Playwright 浏览器
    log "Installing Playwright browsers..."
    npx playwright install --with-deps

    # 运行 E2E 测试
    log "Running Playwright E2E tests..."
    local e2e_args=(
        "--config=playwright.config.ts"
        "--reporter=junit,html"
        "--output=$REPORTS_DIR/playwright-report"
    )

    if [ -n "${CI:-}" ]; then
        e2e_args+=("--reporter=line")
    fi

    npx playwright test "${e2e_args[@]}" 2>&1 | tee "$REPORTS_DIR/e2e-test.log"

    # 复制测试报告
    if [ -d "playwright-report" ]; then
        cp -r playwright-report "$REPORTS_DIR/playwright-html-report"
    fi

    if [ -f "test-results.xml" ]; then
        cp test-results.xml "$REPORTS_DIR/e2e-junit.xml"
    fi

    cd - > /dev/null
    log "E2E tests completed"
}

# 生成测试报告摘要
generate_test_summary() {
    log "Generating test summary..."

    local summary_file="$REPORTS_DIR/test-summary.json"
    cat > "$summary_file" << EOF
{
  "test_run": {
    "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
    "project_id": "$CI_PROJECT_ID",
    "pipeline_id": "$CI_PIPELINE_ID",
    "job_id": "$CI_JOB_ID",
    "commit_sha": "$CI_COMMIT_SHA",
    "test_type": "$TEST_TYPE"
  },
  "results": {
    "backend": {
      "tests_run": 0,
      "passed": 0,
      "failed": 0,
      "coverage_percent": 0
    },
    "frontend": {
      "tests_run": 0,
      "passed": 0,
      "failed": 0,
      "coverage_percent": 0
    }
  }
}
EOF

    # 解析测试结果
    if [ -f "$REPORTS_DIR/backend-junit.xml" ]; then
        local backend_tests
        backend_tests=$(xpath -q -e "//@tests" "$REPORTS_DIR/backend-junit.xml" | cut -d'"' -f2)
        local backend_failures
        backend_failures=$(xpath -q -e "//@failures" "$REPORTS_DIR/backend-junit.xml" | cut -d'"' -f2)

        if [ -n "$backend_tests" ]; then
            local backend_passed=$((backend_tests - backend_failures))
            jq --arg tests "$backend_tests" --arg passed "$backend_passed" --arg failed "$backend_failures" \
                '.results.backend.tests_run = ($tests | tonumber) | .results.backend.passed = ($passed | tonumber) | .results.backend.failed = ($failed | tonumber)' \
                "$summary_file" > "${summary_file}.tmp" && mv "${summary_file}.tmp" "$summary_file"
        fi
    fi

    if [ -f "$REPORTS_DIR/frontend-junit.xml" ]; then
        local frontend_tests
        frontend_tests=$(xpath -q -e "//@tests" "$REPORTS_DIR/frontend-junit.xml" | cut -d'"' -f2)
        local frontend_failures
        frontend_failures=$(xpath -q -e "//@failures" "$REPORTS_DIR/frontend-junit.xml" | cut -d'"' -f2)

        if [ -n "$frontend_tests" ]; then
            local frontend_passed=$((frontend_tests - frontend_failures))
            jq --arg tests "$frontend_tests" --arg passed "$frontend_passed" --arg failed "$frontend_failures" \
                '.results.frontend.tests_run = ($tests | tonumber) | .results.frontend.passed = ($passed | tonumber) | .results.frontend.failed = ($failed | tonumber)' \
                "$summary_file" > "${summary_file}.tmp" && mv "${summary_file}.tmp" "$summary_file"
        fi
    fi

    log "Test summary generated: $summary_file"
    cat "$summary_file"
}

# 清理测试环境
cleanup_test_env() {
    log "Cleaning up test environment..."

    # 停止测试容器（如果有）
    if command -v docker &> /dev/null; then
        docker stop test-postgres test-redis 2>/dev/null || true
        docker rm test-postgres test-redis 2>/dev/null || true
    fi

    # 清理临时文件
    rm -f /tmp/test-*.log

    log "Test environment cleaned up"
}

# 显示帮助信息
show_help() {
    cat << EOF
Usage: $0 [OPTIONS] [TYPE]

GitLab CI Test Runner Script

TYPE:
  backend         Run backend tests only
  frontend        Run frontend tests only
  integration     Run integration tests only
  e2e            Run E2E tests only
  all            Run all tests (default)

OPTIONS:
  --timeout SECONDS     Test timeout (default: 300)
  --coverage PERCENT    Coverage threshold (default: 70)
  --no-reports         Skip generating reports
  --help               Show this help message

Environment Variables:
  DATABASE_URL          PostgreSQL connection URL
  REDIS_URL            Redis connection URL
  CI_PROJECT_ID        GitLab project ID
  CI_PIPELINE_ID       GitLab pipeline ID
  CI_JOB_ID            GitLab job ID
  TEST_TYPE            Type of tests to run

Examples:
  $0                   Run all tests
  $0 backend           Run backend tests only
  $0 --timeout 600 e2e Run E2E tests with 10s timeout
  $0 --no-reports      Run tests without generating reports
EOF
}

# 主函数
main() {
    local test_types=("backend" "frontend" "integration" "e2e")
    local run_all=true

    # 解析命令行参数
    while [[ $# -gt 0 ]]; do
        case $1 in
            --timeout)
                TEST_TIMEOUT="$2"
                shift 2
                ;;
            --coverage)
                COVERAGE_THRESHOLD="$2"
                shift 2
                ;;
            --no-reports)
                GENERATE_REPORTS=false
                shift
                ;;
            --help)
                show_help
                exit 0
                ;;
            backend|frontend|integration|e2e)
                TEST_TYPE="$1"
                run_all=false
                shift
                ;;
            *)
                error "Unknown option: $1"
                show_help
                exit 1
                ;;
        esac
    done

    # 设置错误处理
    trap cleanup_test_env EXIT

    log "Starting GitLab CI test runner..."
    log "Test type: ${TEST_TYPE:-all}"
    log "Timeout: ${TEST_TIMEOUT}s"
    log "Coverage threshold: ${COVERAGE_THRESHOLD}%"

    # 检查环境
    check_test_env
    setup_env

    # 创建报告目录
    mkdir -p "$REPORTS_DIR"

    # 运行测试
    if [ "$run_all" = true ]; then
        run_backend_tests
        run_frontend_tests
        run_integration_tests
        run_e2e_tests
    else
        case "$TEST_TYPE" in
            backend)
                run_backend_tests
                ;;
            frontend)
                run_frontend_tests
                ;;
            integration)
                run_integration_tests
                ;;
            e2e)
                run_e2e_tests
                ;;
            *)
                error "Unknown test type: $TEST_TYPE"
                exit 1
                ;;
        esac
    fi

    # 生成报告
    if [ "$GENERATE_REPORTS" = "true" ]; then
        generate_test_summary
    fi

    log "All tests completed successfully!"
}

# 执行主函数
main "$@"

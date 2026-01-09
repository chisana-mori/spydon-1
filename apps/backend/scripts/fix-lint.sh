#!/bin/bash
# ==============================================================================
# Go 代码 Lint 自动修复脚本
# 自动修复常见的代码风格问题
# ==============================================================================

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info()    { echo -e "${BLUE}ℹ ${1}${NC}"; }
log_success() { echo -e "${GREEN}✅ ${1}${NC}"; }
log_warning() { echo -e "${YELLOW}⚠️  ${1}${NC}"; }
log_error()   { echo -e "${RED}❌ ${1}${NC}"; }
log_step()    { echo -e "▶ ${1}"; }

# 检查命令是否存在
check_command() {
    command -v "$1" &> /dev/null
}

# 安装提示
install_hint() {
    local tool="$1"
    case "$tool" in
        gofumpt)
            echo "   go install mvdan.cc/gofumpt@latest"
            ;;
        goimports)
            echo "   go install golang.org/x/tools/cmd/goimports@latest"
            ;;
        golangci-lint)
            echo "   brew install golangci-lint"
            echo "   # 或: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"
            ;;
    esac
}

echo ""
echo "════════════════════════════════════════════════════════════════"
echo "🔧 Go 代码 Lint 自动修复"
echo "════════════════════════════════════════════════════════════════"
echo ""

cd "$PROJECT_DIR"

# 计数器
fixes_applied=0
warnings=0

# Step 1: gofumpt (更严格的代码格式化)
log_step "检查 gofumpt..."
if check_command gofumpt; then
    log_info "运行 gofumpt (严格格式化)..."
    find . -name "*.go" -type f ! -path "./vendor/*" -exec gofumpt -w {} \;
    log_success "gofumpt 完成"
    ((fixes_applied++))
else
    log_warning "gofumpt 未安装，跳过严格格式化"
    install_hint gofumpt
    ((warnings++))
fi

# Step 2: goimports (自动修复 import)
log_step "检查 goimports..."
if check_command goimports; then
    log_info "运行 goimports (整理 imports)..."
    find . -name "*.go" -type f ! -path "./vendor/*" -exec goimports -w {} \;
    log_success "goimports 完成"
    ((fixes_applied++))
else
    log_warning "goimports 未安装，跳过 imports 整理"
    install_hint goimports
    ((warnings++))
fi

# Step 3: go fmt (标准格式化，作为后备)
log_step "运行 go fmt..."
log_info "运行 go fmt (标准格式化)..."
go fmt ./...
log_success "go fmt 完成"
((fixes_applied++))

# Step 4: go mod tidy (整理依赖)
log_step "运行 go mod tidy..."
log_info "整理 Go 模块依赖..."
go mod tidy
log_success "go mod tidy 完成"
((fixes_applied++))

# Step 5: 运行 golangci-lint 检查剩余问题
echo ""
log_step "检查剩余 lint 问题..."
if check_command golangci-lint; then
    log_info "运行 golangci-lint..."
    echo ""
    # 使用 --fix 尝试自动修复可修复的问题
    if golangci-lint run --fix 2>&1; then
        log_success "没有剩余的 lint 问题!"
    else
        log_warning "仍有一些 lint 问题需要手动修复"
        echo ""
        log_info "运行以下命令查看详细问题:"
        echo "   cd apps/backend && golangci-lint run"
    fi
else
    log_warning "golangci-lint 未安装，跳过检查"
    install_hint golangci-lint
    ((warnings++))
fi

# 打印摘要
echo ""
echo "════════════════════════════════════════════════════════════════"
log_success "Lint 修复完成!"
echo "════════════════════════════════════════════════════════════════"
echo ""
echo "  📊 已应用的修复: ${fixes_applied}"
if [ $warnings -gt 0 ]; then
    echo "  ⚠️  警告数量: ${warnings}"
    echo ""
    echo "💡 安装缺失的工具以获得更完整的修复:"
    if ! check_command gofumpt; then
        echo "   go install mvdan.cc/gofumpt@latest"
    fi
    if ! check_command goimports; then
        echo "   go install golang.org/x/tools/cmd/goimports@latest"
    fi
    if ! check_command golangci-lint; then
        echo "   brew install golangci-lint"
    fi
fi
echo ""

#!/bin/bash
# 从 runtime.json 加载配置并启动开发服务器

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONFIG_FILE="$SCRIPT_DIR/../config/runtime.json"
SAMPLE_CONFIG_FILE="$SCRIPT_DIR/../config/runtime.sample.json"

# 检查配置文件
if [ -f "$CONFIG_FILE" ]; then
  echo "✓ Using config from runtime.json"
  CONFIG_SOURCE="$CONFIG_FILE"
elif [ -f "$SAMPLE_CONFIG_FILE" ]; then
  echo "⚠ Using runtime.sample.json (runtime.json not found)"
  CONFIG_SOURCE="$SAMPLE_CONFIG_FILE"
else
  echo "⚠ No config file found, using defaults"
  npm run dev
  exit 0
fi

# 从 JSON 读取配置并导出为环境变量
export NEXT_PUBLIC_BACKEND_BASE_URL=$(node -p "JSON.parse(require('fs').readFileSync('$CONFIG_SOURCE', 'utf-8')).backendBaseUrl || ''")
export NEXT_PUBLIC_CAS_LOGIN_PATH=$(node -p "JSON.parse(require('fs').readFileSync('$CONFIG_SOURCE', 'utf-8')).casLoginPath || ''")
export NEXT_PUBLIC_CAS_LOGOUT_PATH=$(node -p "JSON.parse(require('fs').readFileSync('$CONFIG_SOURCE', 'utf-8')).casLogoutPath || ''")
export NEXT_PUBLIC_BASE_PATH=$(node -p "JSON.parse(require('fs').readFileSync('$CONFIG_SOURCE', 'utf-8')).basePath || ''")
export NEXT_PUBLIC_KITE_BASE_URL=$(node -p "JSON.parse(require('fs').readFileSync('$CONFIG_SOURCE', 'utf-8')).kiteBaseUrl || ''")

echo "Configuration loaded:"
echo "  BACKEND_BASE_URL: $NEXT_PUBLIC_BACKEND_BASE_URL"
echo "  CAS_LOGIN_PATH: $NEXT_PUBLIC_CAS_LOGIN_PATH"
echo "  CAS_LOGOUT_PATH: $NEXT_PUBLIC_CAS_LOGOUT_PATH"
echo "  BASE_PATH: $NEXT_PUBLIC_BASE_PATH"
echo "  KITE_BASE_URL: $NEXT_PUBLIC_KITE_BASE_URL"
echo ""

# 启动开发服务器
next dev

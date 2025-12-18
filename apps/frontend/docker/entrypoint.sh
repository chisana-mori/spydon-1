#!/bin/bash
# 前端容器启动脚本
# 负责运行时配置注入和应用启动

set -e

echo "🚀 Starting Robusta Frontend..."

# ============== 运行时配置处理 ==============
RUNTIME_CONFIG_PATH="${RUNTIME_CONFIG_PATH:-/app/config/runtime.json}"
PUBLIC_CONFIG_PATH="/app/public/__runtime_config__.json"

if [ -f "$RUNTIME_CONFIG_PATH" ]; then
    echo "📋 Found runtime config at ${RUNTIME_CONFIG_PATH}"
    # 复制运行时配置到 public 目录供前端加载
    cp "$RUNTIME_CONFIG_PATH" "$PUBLIC_CONFIG_PATH"
    echo "✅ Runtime config injected to ${PUBLIC_CONFIG_PATH}"

    # 输出配置内容（隐藏敏感信息）
    echo "📝 Runtime config loaded:"
    cat "$RUNTIME_CONFIG_PATH" | head -20
else
    echo "⚠️  No runtime config found at ${RUNTIME_CONFIG_PATH}, using defaults"
    # 创建默认配置
    cat > "$PUBLIC_CONFIG_PATH" << 'EOF'
{
  "backendBaseUrl": "",
  "casLoginPath": "/auth/cas/login",
  "casLogoutPath": "/auth/cas/logout",
  "basePath": ""
}
EOF
fi

# ============== 环境变量覆盖支持 ==============
# 支持通过环境变量覆盖部分配置
if [ -n "$OVERRIDE_BACKEND_URL" ]; then
    echo "🔧 Overriding backendBaseUrl with: ${OVERRIDE_BACKEND_URL}"
    # 使用 node 来更新 JSON (alpine 中可用)
    node -e "
        const fs = require('fs');
        const config = JSON.parse(fs.readFileSync('$PUBLIC_CONFIG_PATH', 'utf8'));
        config.backendBaseUrl = '$OVERRIDE_BACKEND_URL';
        fs.writeFileSync('$PUBLIC_CONFIG_PATH', JSON.stringify(config, null, 2));
    "
fi

if [ -n "$OVERRIDE_BASE_PATH" ]; then
    echo "🔧 Overriding basePath with: ${OVERRIDE_BASE_PATH}"
    node -e "
        const fs = require('fs');
        const config = JSON.parse(fs.readFileSync('$PUBLIC_CONFIG_PATH', 'utf8'));
        config.basePath = '$OVERRIDE_BASE_PATH';
        fs.writeFileSync('$PUBLIC_CONFIG_PATH', JSON.stringify(config, null, 2));
    "
fi

echo "============================================"
echo "🌐 Starting Next.js server on port ${PORT:-3000}..."
echo "============================================"

# 执行传入的命令 (默认: node server.js)
exec "$@"

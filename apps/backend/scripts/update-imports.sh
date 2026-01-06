#!/bin/bash

# 更新import路径脚本
# 将所有对旧services包的引用更新为新的feature-based包

set -e

echo "开始更新import路径..."

# 定义需要更新的文件列表
files=(
    "./internal/middleware/apikey.go"
    "./internal/features/pipeline/transport/http/handler.go"
    "./internal/features/auth/transport/http/utils.go"
    "./internal/features/ingest/transport/http/handler.go"
    "./internal/features/ingest/transport/http/enrichment.go"
    "./internal/features/systemsetting/transport/http/handler.go"
    "./internal/features/user/transport/http/handler.go"
    "./internal/features/rca/transport/http/handler.go"
    "./internal/features/rca/transport/http/rca_basic.go"
    "./internal/features/apikey/transport/http/handler.go"
    "./internal/features/knowledge/transport/http/handler.go"
    "./internal/features/holmes/transport/http/handler.go"
    "./internal/features/query/transport/http/handler.go"
    "./internal/features/query/transport/http/alerts.go"
    "./internal/features/navy/transport/http/handler.go"
    "./internal/features/navy/transport/http/k8s_nodes.go"
    "./internal/features/navy/transport/http/device_ops.go"
    "./internal/features/navy/transport/http/devices.go"
    "./internal/features/navy/transport/http/safe_drain.go"
)

cd apps/backend

for file in "${files[@]}"; do
    if [ -f "$file" ]; then
        echo "处理: $file"
        # 这个脚本只是标记需要手动更新的文件
        # 因为每个文件需要导入的service包不同
    fi
done

echo "需要手动更新以下文件的import语句："
echo "将 'robusta-web/backend/internal/services' 替换为对应feature的service包"
echo ""
echo "例如："
echo "  APIKeyService -> robusta-web/backend/internal/features/apikey/service"
echo "  UserService -> robusta-web/backend/internal/features/user/service"
echo "  AuthService -> robusta-web/backend/internal/features/auth/service"
echo "  等等..."

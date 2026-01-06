#!/bin/bash

# 更新package声明脚本
# 将features目录下service子目录中的所有.go文件的package声明从services改为service

set -e

echo "开始更新package声明..."

# 查找所有features下service目录中的.go文件（排除vendor和测试文件）
find apps/backend/internal/features/*/service -name "*.go" -type f | while read -r file; do
    # 检查文件是否包含 "package services"
    if grep -q "^package services$" "$file"; then
        echo "更新: $file"
        # 使用sed替换package声明
        sed -i '' 's/^package services$/package service/' "$file"
    fi
done

echo "Package声明更新完成！"

#!/bin/bash

# 从环境变量或参数获取数据库连接信息
DATABASE_URL=${DATABASE_URL:-"postgresql://postgres:postgres@localhost:5432/robusta?sslmode=disable"}

echo "Running migration to remove severity column..."
echo "Database: $DATABASE_URL"

# 使用 psql 执行迁移
psql "$DATABASE_URL" -c "ALTER TABLE knowledge_articles DROP COLUMN IF EXISTS severity;"

if [ $? -eq 0 ]; then
    echo "✅ Migration completed successfully!"
else
    echo "❌ Migration failed!"
    exit 1
fi

#!/bin/bash

# 移除外键约束的脚本
# 这个脚本会连接到数据库并移除外键约束

set -e

# 从环境变量或参数获取数据库连接信息
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_NAME="${DB_NAME:-robusta}"
DB_USER="${DB_USER:-postgres}"
DB_PASSWORD="${DB_PASSWORD:-postgres}"

echo "正在连接到数据库: $DB_HOST:$DB_PORT/$DB_NAME"

# 执行SQL命令移除外键约束
PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME <<EOF
-- 移除 alerts 表的外键约束
ALTER TABLE alerts DROP CONSTRAINT IF EXISTS fk_alerts_cluster;

-- 移除 rca_runs 表的外键约束
ALTER TABLE rca_runs DROP CONSTRAINT IF EXISTS fk_rca_runs_alert;

-- 验证外键已被移除
SELECT
    conname AS constraint_name,
    conrelid::regclass AS table_name
FROM pg_constraint
WHERE contype = 'f'
    AND conrelid::regclass::text IN ('alerts', 'rca_runs');

EOF

echo "外键约束已成功移除！"
echo "注意：数据完整性现在由应用层代码保证"

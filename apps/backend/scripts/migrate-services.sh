#!/bin/bash

# Service迁移脚本
# 将services目录下的文件迁移到features目录下的service子目录

set -e

echo "开始迁移service文件..."

# 迁移Holmes相关service
echo "迁移Holmes相关service..."
mv apps/backend/internal/services/holmes_service.go apps/backend/internal/features/holmes/service/ 2>/dev/null || true
mv apps/backend/internal/services/holmes_common.go apps/backend/internal/features/holmes/service/ 2>/dev/null || true
mv apps/backend/internal/services/holmes_stats.go apps/backend/internal/features/holmes/service/ 2>/dev/null || true
mv apps/backend/internal/services/holmes_storage.go apps/backend/internal/features/holmes/service/ 2>/dev/null || true
mv apps/backend/internal/services/holmes_stream.go apps/backend/internal/features/holmes/service/ 2>/dev/null || true

# 迁移RCA相关service
echo "迁移RCA相关service..."
mv apps/backend/internal/services/rca_service.go apps/backend/internal/features/rca/service/ 2>/dev/null || true
mv apps/backend/internal/services/rca_analysis.go apps/backend/internal/features/rca/service/ 2>/dev/null || true
mv apps/backend/internal/services/rca_auto.go apps/backend/internal/features/rca/service/ 2>/dev/null || true
mv apps/backend/internal/services/rca_broadcast.go apps/backend/internal/features/rca/service/ 2>/dev/null || true
mv apps/backend/internal/services/rca_stats.go apps/backend/internal/features/rca/service/ 2>/dev/null || true
mv apps/backend/internal/services/rca_*_test.go apps/backend/internal/features/rca/service/ 2>/dev/null || true

# 迁移Alert相关service
echo "迁移Alert相关service..."
mv apps/backend/internal/services/alert_service.go apps/backend/internal/features/ingest/service/ 2>/dev/null || true

# 迁移Navy相关service
echo "迁移Navy相关service..."
mv apps/backend/internal/services/navy_device_service.go apps/backend/internal/features/navy/service/ 2>/dev/null || true
mv apps/backend/internal/services/navy_device_dto.go apps/backend/internal/features/navy/service/ 2>/dev/null || true
mv apps/backend/internal/services/device_operations_service.go apps/backend/internal/features/navy/service/ 2>/dev/null || true
mv apps/backend/internal/services/k8s_node_manage_service.go apps/backend/internal/features/navy/service/ 2>/dev/null || true
mv apps/backend/internal/services/simple_drain_service.go apps/backend/internal/features/navy/service/ 2>/dev/null || true
mv apps/backend/internal/services/drain_*.go apps/backend/internal/features/navy/service/ 2>/dev/null || true
mv apps/backend/internal/services/safe_drain_dto.go apps/backend/internal/features/navy/service/ 2>/dev/null || true

# 迁移Pipeline相关service
echo "迁移Pipeline相关service..."
mv apps/backend/internal/services/pipeline_engine.go apps/backend/internal/features/pipeline/service/ 2>/dev/null || true
mv apps/backend/internal/services/pipeline_*_test.go apps/backend/internal/features/pipeline/service/ 2>/dev/null || true
mv apps/backend/internal/services/awx_runtime.go apps/backend/internal/features/pipeline/service/ 2>/dev/null || true
mv apps/backend/internal/services/awx_helper.go apps/backend/internal/features/pipeline/service/ 2>/dev/null || true
mv apps/backend/internal/services/awx_streamer.go apps/backend/internal/features/pipeline/service/ 2>/dev/null || true
mv apps/backend/internal/services/prometheus_runtime.go apps/backend/internal/features/pipeline/service/ 2>/dev/null || true
mv apps/backend/internal/services/prometheus_types.go apps/backend/internal/features/pipeline/service/ 2>/dev/null || true
mv apps/backend/internal/services/workflow_runtime.go apps/backend/internal/features/pipeline/service/ 2>/dev/null || true

echo "Service文件迁移完成！"
echo "接下来需要："
echo "1. 更新所有service文件的package声明为'package service'"
echo "2. 更新所有import路径"
echo "3. 运行测试验证"

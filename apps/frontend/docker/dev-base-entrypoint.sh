#!/bin/bash
# 开发镜像入口脚本：负责在挂载目录缺少依赖时复制离线缓存
set -euo pipefail

WORKSPACE_DIR="${WORKSPACE_DIR:-/workspace}"
CACHE_DIR="/opt/offline-cache"
STAMP_FILE="${WORKSPACE_DIR}/node_modules/.offline-cache"

if [ "${OFFLINE_SKIP_COPY:-0}" = "1" ]; then
  echo "[offline-base] 已设置 OFFLINE_SKIP_COPY=1，跳过 node_modules 同步"
else
  if [ ! -d "${WORKSPACE_DIR}/node_modules" ] || [ ! -f "${STAMP_FILE}" ]; then
    echo "[offline-base] 挂载目录缺少依赖，开始复制离线缓存"
    mkdir -p "${WORKSPACE_DIR}/node_modules"
    cp -a "${CACHE_DIR}/node_modules/." "${WORKSPACE_DIR}/node_modules/"
    cp -f "${CACHE_DIR}/package.json" "${WORKSPACE_DIR}/package.json.cache" || true
    cp -f "${CACHE_DIR}/package-lock.json" "${WORKSPACE_DIR}/package-lock.json.cache" || true
    cp -f "${CACHE_DIR}/bun.lockb" "${WORKSPACE_DIR}/bun.lockb.cache" || true
    touch "${STAMP_FILE}"
    echo "[offline-base] 依赖复制完成"
  else
    echo "[offline-base] 检测到已有离线依赖，跳过复制"
  fi
fi

exec "$@"

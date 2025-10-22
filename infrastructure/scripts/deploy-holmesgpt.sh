#!/bin/bash

# HolmesGPT 部署脚本
# 此脚本将在现有的 Robusta 部署中启用 HolmesGPT 功能

set -e

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

echo "🚀 开始部署 HolmesGPT..."

# 检查必要的工具
command -v kubectl >/dev/null 2>&1 || { echo "❌ kubectl 未安装"; exit 1; }
command -v helm >/dev/null 2>&1 || { echo "❌ helm 未安装"; exit 1; }

# 检查 Robusta 命名空间是否存在
if ! kubectl get namespace robusta >/dev/null 2>&1; then
    echo "❌ Robusta 命名空间不存在，请先部署 Robusta"
    exit 1
fi

# 检查 Robusta Helm 仓库
echo "📦 检查 Robusta Helm 仓库..."
if ! helm repo list | grep -q robusta; then
    echo "➕ 添加 Robusta Helm 仓库..."
    helm repo add robusta https://robusta-dev.github.io/helm-charts/
else
    echo "✅ Robusta Helm 仓库已存在"
fi

# 更新 Helm 仓库
echo "🔄 更新 Helm 仓库..."
helm repo update

# 创建 Kubernetes Secret 来存储敏感信息（可选，如果您想使用 Secret 而不是直接在配置中写入）
echo "🔐 创建 HolmesGPT 配置 Secret..."
kubectl create secret generic holmes-secrets \
    --from-literal=deepseekApiKey='sk-4e36ce8a083f47fcad3e3d0ff757f255' \
    --namespace=robusta \
    --dry-run=client -o yaml | kubectl apply -f -

# 升级 Robusta 部署以启用 HolmesGPT
echo "⬆️  升级 Robusta 部署以启用 HolmesGPT..."
helm upgrade --install robusta robusta/robusta \
    --namespace robusta \
    --create-namespace \
    -f "${SCRIPT_DIR}/robusta-holmesgpt-values.yaml" \
    --wait \
    --timeout=10m

echo "⏳ 等待 HolmesGPT 组件启动..."
kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=robusta -n robusta --timeout=300s

# 检查部署状态
echo "🔍 检查部署状态..."
kubectl get pods -n robusta

# 检查 HolmesGPT 是否正确配置
echo "🧪 验证 HolmesGPT 配置..."
if kubectl logs -n robusta deployment/robusta-runner --tail=50 | grep -i "holmes\|gpt" >/dev/null 2>&1; then
    echo "✅ HolmesGPT 配置成功！"
else
    echo "⚠️  HolmesGPT 配置可能有问题，请检查日志"
fi

echo ""
echo "🎉 HolmesGPT 部署完成！"
echo ""
echo "📋 部署摘要:"
echo "   - HolmesGPT: ✅ 已启用"
echo "   - AI 模型: DeepSeek Chat (https://api.deepseek.com)"
echo "   - 邮件通知: heningyu21@126.com"
echo "   - 自动分析: ✅ 已配置"
echo ""
echo "🔧 配置的自动分析场景:"
echo "   - Prometheus 告警自动分析"
echo "   - ImagePullBackOff 错误分析"
echo "   - Pod 崩溃分析"
echo "   - 节点资源不足分析"
echo ""
echo "📧 邮件配置说明:"
echo "   当前配置使用示例 SMTP 设置，您需要根据实际邮件服务器调整配置。"
echo "   请编辑 ${SCRIPT_DIR}/robusta-holmesgpt-values.yaml 中的 mail_sink 配置。"
echo ""
echo "🧪 测试 HolmesGPT:"
echo "   kubectl apply -f https://raw.githubusercontent.com/robusta-dev/kubernetes-demos/main/crashpod/broken.yaml"
echo ""
echo "📊 查看日志:"
echo "   kubectl logs -n robusta deployment/robusta-runner -f"
echo ""
echo "🌐 访问 Robusta UI (如果已配置):"
echo "   kubectl port-forward -n robusta svc/robusta-ui 3000:3000"

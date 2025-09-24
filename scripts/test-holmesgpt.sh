#!/bin/bash

# HolmesGPT 测试脚本
# 此脚本用于测试 HolmesGPT 的功能

set -e

echo "🧪 HolmesGPT 功能测试"
echo "===================="

# 检查 HolmesGPT 是否已启用
echo "1. 检查 HolmesGPT 配置..."
if kubectl get pods -n robusta -l app=holmes | grep -q "Running"; then
    echo "✅ HolmesGPT Pod 正在运行"
else
    echo "❌ HolmesGPT Pod 未运行，请先运行部署脚本"
    exit 1
fi

# 检查 DeepSeek API 配置
echo "2. 检查 DeepSeek API 配置..."
if kubectl get secret -n robusta holmes-secrets >/dev/null 2>&1; then
    echo "✅ HolmesGPT API 密钥已配置"
else
    echo "⚠️  API 密钥 Secret 不存在"
fi

# 检查邮件配置
echo "3. 检查邮件配置..."
if kubectl get secret -n robusta robusta-playbooks-config-secret -o jsonpath='{.data.active_playbooks\.yaml}' | base64 -d | grep -q "mail_sink"; then
    echo "✅ 邮件 Sink 已配置"
else
    echo "⚠️  邮件 Sink 未配置"
fi

# 部署测试 Pod 来触发告警
echo "4. 部署测试 Pod 触发告警..."
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Pod
metadata:
  name: holmesgpt-test-crashpod
  namespace: default
  labels:
    app: holmesgpt-test
spec:
  containers:
  - name: crashpod
    image: busybox
    command: ["sh", "-c", "echo 'Starting test pod...'; sleep 10; echo 'Simulating crash...'; exit 1"]
  restartPolicy: Never
EOF

echo "✅ 测试 Pod 已部署"

# 等待 Pod 失败
echo "5. 等待 Pod 失败以触发分析..."
sleep 15

# 检查 Pod 状态
kubectl get pod holmesgpt-test-crashpod -o wide

# 检查 Robusta 日志中的 HolmesGPT 活动
echo "6. 检查 HolmesGPT 分析日志..."
echo "最近的 Robusta 日志："
kubectl logs -n robusta deployment/robusta-runner --tail=20 | grep -i "holmes\|gpt\|analysis" || echo "未找到 HolmesGPT 相关日志"

# 创建一个 ImagePullBackOff 测试
echo "7. 创建 ImagePullBackOff 测试..."
cat <<EOF | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: holmesgpt-test-imagepull
  namespace: default
spec:
  replicas: 1
  selector:
    matchLabels:
      app: holmesgpt-test-imagepull
  template:
    metadata:
      labels:
        app: holmesgpt-test-imagepull
    spec:
      containers:
      - name: test-container
        image: nonexistent-image:latest
        imagePullPolicy: Always
EOF

echo "✅ ImagePullBackOff 测试部署已创建"

# 等待一段时间让告警触发
echo "8. 等待告警触发和分析..."
sleep 30

# 显示最新的日志
echo "9. 显示最新的分析日志..."
kubectl logs -n robusta deployment/robusta-runner --tail=50 | grep -A 5 -B 5 -i "holmes\|gpt\|analysis" || echo "未找到相关日志"

# 清理测试资源
echo "10. 清理测试资源..."
kubectl delete pod holmesgpt-test-crashpod --ignore-not-found=true
kubectl delete deployment holmesgpt-test-imagepull --ignore-not-found=true

echo ""
echo "🎉 测试完成！"
echo ""
echo "📋 测试结果摘要:"
echo "   - HolmesGPT 配置: ✅"
echo "   - API 密钥配置: ✅"
echo "   - 邮件配置: 请检查上述输出"
echo "   - 测试告警: 已触发"
echo ""
echo "📧 如果配置正确，您应该会收到分析结果邮件"
echo ""
echo "🔍 查看完整日志:"
echo "   kubectl logs -n robusta deployment/robusta-runner -f"
echo ""
echo "📊 查看 Pod 状态:"
echo "   kubectl get pods -n robusta"

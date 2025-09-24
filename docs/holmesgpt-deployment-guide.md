# HolmesGPT 部署和使用指南

本指南将帮助您在现有的 Robusta 集群中部署和配置 HolmesGPT，实现自动化的 Kubernetes 告警根因分析。

## 🎯 功能概述

HolmesGPT 是 Robusta 的 AI 分析引擎，具备以下功能：
- **自动根因分析**: 当告警发生时自动分析问题原因
- **智能建议**: 提供详细的解决方案和修复建议
- **多场景支持**: 支持 Pod 崩溃、镜像拉取失败、资源不足等场景
- **邮件通知**: 将分析结果通过邮件发送给运维团队
- **中文支持**: 支持中文分析报告

## 🔧 配置说明

### AI 模型配置
- **模型**: DeepSeek Chat
- **API 地址**: https://api.deepseek.com
- **API Key**: sk-4e36ce8a083f47fcad3e3d0ff757f255

### 邮件通知配置
- **收件人**: heningyu21@126.com
- **发送方式**: SMTP（需要配置邮件服务器）

### 自动分析场景
1. **Prometheus 告警**: 所有 Prometheus 告警自动触发分析
2. **ImagePullBackOff**: 镜像拉取失败自动分析
3. **Pod 崩溃**: Pod 异常退出自动分析
4. **节点资源**: CPU/内存/磁盘使用率过高分析

## 🚀 部署步骤

### 1. 检查前置条件

确保您的环境满足以下条件：
```bash
# 检查 Robusta 是否已部署
kubectl get pods -n robusta

# 检查 Helm 是否安装
helm version

# 检查集群连接
kubectl cluster-info
```

### 2. 执行部署脚本

```bash
# 进入项目目录
cd /Users/ny/Documents/robusta-web

# 执行 HolmesGPT 部署脚本
./scripts/deploy-holmesgpt.sh
```

部署脚本将自动完成以下操作：
- 添加/更新 Robusta Helm 仓库
- 创建 API 密钥 Secret
- 升级 Robusta 部署以启用 HolmesGPT
- 配置自动分析规则
- 验证部署状态

### 3. 配置邮件通知

**重要**: 默认配置中的邮件设置需要根据您的实际邮件服务器进行调整。

编辑配置文件：
```bash
vim scripts/robusta-holmesgpt-values.yaml
```

找到 `mail_sink` 配置部分，根据您的邮件服务器修改：

```yaml
# 示例：使用 Gmail
mailto: "mailtos://your-gmail@gmail.com:your-app-password@smtp.gmail.com:587?from=robusta-alerts@yourdomain.com&to=heningyu21@126.com"

# 示例：使用企业邮箱
mailto: "mailtos://username:password@mail.company.com:587?from=alerts@company.com&to=heningyu21@126.com"
```

详细的邮件配置说明请参考：[邮件配置指南](holmesgpt-email-setup.md)

### 4. 重新部署应用配置

如果修改了邮件配置，需要重新部署：
```bash
helm upgrade robusta robusta/robusta \
    --namespace robusta \
    -f scripts/robusta-holmesgpt-values.yaml
```

## 🧪 功能测试

### 1. 运行自动化测试

```bash
./scripts/test-holmesgpt.sh
```

测试脚本将：
- 验证 HolmesGPT 配置
- 创建测试 Pod 触发告警
- 检查分析日志
- 清理测试资源

### 2. 手动测试场景

#### 测试 Pod 崩溃分析
```bash
kubectl apply -f https://raw.githubusercontent.com/robusta-dev/kubernetes-demos/main/crashpod/broken.yaml
```

#### 测试镜像拉取失败分析
```bash
kubectl run test-imagepull --image=nonexistent-image:latest
```

#### 测试资源告警分析
```bash
# 创建高 CPU 使用率的 Pod
kubectl run cpu-stress --image=progrium/stress -- --cpu 2 --timeout 60s
```

### 3. 查看分析结果

```bash
# 查看 HolmesGPT 分析日志
kubectl logs -n robusta deployment/robusta-runner | grep -i holmes

# 实时监控日志
kubectl logs -n robusta deployment/robusta-runner -f
```

## 📊 监控和维护

### 查看 HolmesGPT 状态

```bash
# 检查 Robusta 组件状态
kubectl get pods -n robusta

# 查看 HolmesGPT 配置
kubectl get configmap -n robusta robusta-config -o yaml | grep -A 20 holmes

# 检查 API 密钥 Secret
kubectl get secret -n robusta holmes-secrets
```

### 常用调试命令

```bash
# 查看最近的分析活动
kubectl logs -n robusta deployment/robusta-runner --tail=100 | grep -i "holmes\|analysis"

# 检查邮件发送状态
kubectl logs -n robusta deployment/robusta-runner | grep -i mail

# 查看告警处理情况
kubectl logs -n robusta deployment/robusta-runner | grep -i alert
```

## 🔧 故障排除

### 常见问题

1. **HolmesGPT 未启动**
   - 检查 API 密钥是否正确
   - 验证网络连接到 DeepSeek API
   - 查看 Pod 日志中的错误信息

2. **邮件发送失败**
   - 验证 SMTP 服务器配置
   - 检查邮箱认证信息
   - 确认防火墙和网络策略

3. **分析未触发**
   - 检查告警是否正确产生
   - 验证 Playbook 配置
   - 查看触发条件是否匹配

### 日志分析

```bash
# 查看详细错误日志
kubectl logs -n robusta deployment/robusta-runner --previous

# 检查特定时间段的日志
kubectl logs -n robusta deployment/robusta-runner --since=1h

# 导出日志进行分析
kubectl logs -n robusta deployment/robusta-runner > holmesgpt-logs.txt
```

## 📈 性能优化

### 资源配置调整

如果需要调整 HolmesGPT 的资源配置：

```yaml
holmes:
  resources:
    requests:
      memory: "512Mi"
      cpu: "250m"
    limits:
      memory: "1Gi"
      cpu: "500m"
```

### 分析频率控制

通过 `rate_limit` 参数控制分析频率：

```yaml
triggers:
  - on_pod_crash:
      rate_limit: 5  # 每5分钟最多触发一次
```

## 🔄 升级和维护

### 升级 HolmesGPT

```bash
# 更新 Helm 仓库
helm repo update

# 升级到最新版本
helm upgrade robusta robusta/robusta \
    --namespace robusta \
    -f scripts/robusta-holmesgpt-values.yaml
```

### 备份配置

```bash
# 备份当前配置
kubectl get configmap -n robusta robusta-config -o yaml > robusta-config-backup.yaml

# 备份 Secret
kubectl get secret -n robusta holmes-secrets -o yaml > holmes-secrets-backup.yaml
```

## 📚 相关文档

- [HolmesGPT 邮件配置指南](holmesgpt-email-setup.md)
- [Robusta 官方文档](https://docs.robusta.dev/)
- [HolmesGPT 配置参考](https://docs.robusta.dev/master/configuration/holmesgpt/)
- [DeepSeek API 文档](https://platform.deepseek.com/api-docs/)

## 🆘 获取帮助

如果遇到问题，可以：
1. 查看本文档的故障排除部分
2. 检查 Robusta 官方文档
3. 查看项目日志和错误信息
4. 联系技术支持团队

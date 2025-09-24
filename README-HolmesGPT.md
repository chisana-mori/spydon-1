# HolmesGPT 集成配置说明

## 📋 项目概述

本项目为您的 Kubernetes 集群集成了 HolmesGPT AI 分析引擎，实现了自动化的告警根因分析和邮件通知功能。

## 🎯 核心功能

### ✅ 已配置功能
- **HolmesGPT AI 分析**: 使用 DeepSeek API 进行智能分析
- **自动触发机制**: 告警发生时自动启动分析
- **邮件通知系统**: 分析结果自动发送到指定邮箱
- **多场景支持**: 涵盖常见的 Kubernetes 故障场景
- **中文分析报告**: 提供中文的分析结果和解决建议

### 🔧 配置详情

| 配置项 | 值 | 说明 |
|--------|-----|------|
| AI 模型 | DeepSeek Chat | 使用 DeepSeek 的 GPT 模型 |
| API 地址 | https://api.deepseek.com | DeepSeek API 端点 |
| API Key | sk-4e36ce8a083f47fcad3e3d0ff757f255 | 您提供的 API 密钥 |
| 邮件收件人 | heningyu21@126.com | 分析结果接收邮箱 |
| 部署命名空间 | robusta | Robusta 部署的命名空间 |
| 集群名称 | robusta-kind | Kind 集群标识 |

## 🚀 快速开始

### 1. 部署 HolmesGPT
```bash
# 执行部署脚本
./scripts/deploy-holmesgpt.sh
```

### 2. 配置邮件服务器
编辑 `scripts/robusta-holmesgpt-values.yaml` 中的邮件配置：
```yaml
mailto: "mailtos://your-email@gmail.com:your-password@smtp.gmail.com:587?from=alerts@company.com&to=heningyu21@126.com"
```

### 3. 测试功能
```bash
# 运行测试脚本
./scripts/test-holmesgpt.sh
```

## 📁 文件结构

```
├── scripts/
│   ├── robusta-holmesgpt-values.yaml    # HolmesGPT 配置文件
│   ├── deploy-holmesgpt.sh              # 部署脚本
│   └── test-holmesgpt.sh                # 测试脚本
├── docs/
│   ├── holmesgpt-deployment-guide.md    # 详细部署指南
│   └── holmesgpt-email-setup.md         # 邮件配置指南
└── README-HolmesGPT.md                  # 本文件
```

## 🔍 自动分析场景

### 1. Prometheus 告警分析
- **触发条件**: 任何 Prometheus 告警
- **分析内容**: 告警原因、影响范围、解决方案
- **通知方式**: 邮件 + Hub Webhook

### 2. ImagePullBackOff 分析
- **触发条件**: Pod 镜像拉取失败
- **分析内容**: 镜像问题诊断、网络连接检查
- **频率限制**: 每5分钟最多一次

### 3. Pod 崩溃分析
- **触发条件**: Pod 异常退出或重启
- **分析内容**: 崩溃原因、日志分析、修复建议
- **包含信息**: Pod 日志、集群状态

### 4. 节点资源分析
- **触发条件**: CPU/内存/磁盘使用率告警
- **分析内容**: 资源消耗分析、优化建议
- **监控指标**: NodeMemoryHigh, NodeCPUHigh, NodeDiskSpaceHigh

## 📧 邮件通知配置

### 当前配置状态
- ⚠️ **需要配置**: 邮件服务器设置需要根据实际情况调整
- 📧 **收件人**: heningyu21@126.com
- 🔧 **配置文件**: `scripts/robusta-holmesgpt-values.yaml`

### 支持的邮件服务
- Gmail (需要应用专用密码)
- 163 邮箱
- QQ 邮箱 (需要授权码)
- 企业邮箱
- Amazon SES

详细配置方法请参考：[邮件配置指南](docs/holmesgpt-email-setup.md)

## 🛠️ 运维命令

### 查看状态
```bash
# 检查 HolmesGPT 运行状态
kubectl get pods -n robusta

# 查看分析日志
kubectl logs -n robusta deployment/robusta-runner | grep -i holmes

# 实时监控
kubectl logs -n robusta deployment/robusta-runner -f
```

### 配置管理
```bash
# 查看当前配置
kubectl get configmap -n robusta robusta-config -o yaml

# 更新配置
helm upgrade robusta robusta/robusta -n robusta -f scripts/robusta-holmesgpt-values.yaml

# 重启服务
kubectl rollout restart deployment/robusta-runner -n robusta
```

## 🧪 测试验证

### 手动触发测试
```bash
# 1. 部署崩溃 Pod
kubectl apply -f https://raw.githubusercontent.com/robusta-dev/kubernetes-demos/main/crashpod/broken.yaml

# 2. 创建镜像拉取失败
kubectl run test-fail --image=nonexistent-image:latest

# 3. 模拟资源压力
kubectl run cpu-stress --image=progrium/stress -- --cpu 2 --timeout 60s
```

### 验证分析结果
1. 查看 Robusta 日志中的分析活动
2. 检查邮箱是否收到分析报告
3. 验证分析内容的准确性和实用性

## ⚠️ 重要注意事项

### 安全考虑
- API 密钥已存储在 Kubernetes Secret 中
- 邮件密码建议使用 Secret 管理
- 定期轮换 API 密钥

### 成本控制
- DeepSeek API 按使用量计费
- 通过 `rate_limit` 控制分析频率
- 监控 API 使用量和成本

### 网络要求
- 集群需要访问 https://api.deepseek.com
- 邮件发送需要 SMTP 服务器连接
- 确保防火墙和网络策略允许相关连接

## 🔄 升级和维护

### 定期维护任务
- 检查 HolmesGPT 运行状态
- 监控邮件发送成功率
- 更新 Robusta 和 HolmesGPT 版本
- 备份重要配置

### 升级步骤
```bash
# 1. 更新 Helm 仓库
helm repo update

# 2. 升级部署
helm upgrade robusta robusta/robusta -n robusta -f scripts/robusta-holmesgpt-values.yaml

# 3. 验证升级结果
kubectl get pods -n robusta
```

## 📞 技术支持

### 故障排除
1. 查看 [部署指南](docs/holmesgpt-deployment-guide.md) 的故障排除部分
2. 检查 Robusta 官方文档
3. 分析 Pod 日志和事件

### 获取帮助
- 📖 [Robusta 官方文档](https://docs.robusta.dev/)
- 🤖 [HolmesGPT 配置指南](https://docs.robusta.dev/master/configuration/holmesgpt/)
- 🔧 [DeepSeek API 文档](https://platform.deepseek.com/api-docs/)

---

**配置完成后，您的 Kubernetes 集群将具备智能化的告警分析和通知能力，大大提升运维效率！** 🎉

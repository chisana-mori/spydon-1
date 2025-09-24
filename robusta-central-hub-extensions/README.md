# Robusta Central Hub Extensions

这个包为Robusta提供了与Central Hub集成的扩展，包括：

- **Central Webhook Sink**: 将告警发送到Central Hub进行集中管理
- **Holmes Analyze Action**: 触发HolmesGPT自动根因分析

## 🚀 快速开始

### 1. 安装扩展

#### 方法一：从Git仓库安装

在你的Robusta `values.yaml` 中添加：

```yaml
playbookRepos:
  central_hub_extensions:
    url: "https://github.com/your-org/robusta-central-hub-extensions.git"
    pip_install: true
```

#### 方法二：本地开发安装

```bash
# 推送到Robusta
robusta playbooks push ./robusta-extensions
```

### 2. 配置Central Hub集成

在你的Robusta配置中添加：

```yaml
# 全局配置
globalConfig:
  cluster_name: "your-cluster-name"
  central_hub_url: "https://your-central-hub.example.com"
  central_hub_hmac_secret: "your-hmac-secret"

# Sink配置
sinksConfig:
  - central_webhook:
      name: "central_hub_sink"
      url: "https://your-central-hub.example.com/api/v1/ingest/alert"
      cluster_id: "your-cluster-name"
      hmac_secret: "your-hmac-secret"

# 自定义Playbooks
customPlaybooks:
  - name: "AutoHolmesAnalysis"
    triggers:
      - on_prometheus_alert:
          severity: "critical"
    actions:
      - holmes_analyze:
          central_hub_url: "https://your-central-hub.example.com"
          hmac_secret: "your-hmac-secret"
          cluster_id: "your-cluster-name"
    sinks:
      - "central_hub_sink"
```

### 3. 应用配置

```bash
helm upgrade robusta robusta/robusta -f your-values.yaml
```

## 📋 功能详解

### Central Webhook Sink

将Robusta的告警发送到Central Hub进行集中管理。

**特性：**
- HMAC-SHA256签名验证
- 自动重试机制
- 批量发送支持
- 集群心跳监控

**配置参数：**

| 参数 | 类型 | 默认值 | 描述 |
|------|------|--------|------|
| `url` | string | - | Central Hub webhook URL |
| `cluster_id` | string | - | 集群唯一标识符 |
| `hmac_secret` | string | - | HMAC签名密钥 |
| `timeout` | int | 30 | 请求超时时间（秒） |
| `retry_attempts` | int | 3 | 重试次数 |
| `batch_size` | int | 10 | 批量发送大小 |

### Holmes Analyze Action

触发HolmesGPT进行自动根因分析。

**特性：**
- 自动触发高严重级别告警分析
- 可配置分析深度
- 速率限制防止重复分析
- 结果自动添加到告警enrichment

**配置参数：**

| 参数 | 类型 | 默认值 | 描述 |
|------|------|--------|------|
| `central_hub_url` | string | - | Central Hub基础URL |
| `hmac_secret` | string | - | HMAC签名密钥 |
| `cluster_id` | string | - | 集群标识符 |
| `depth` | string | "standard" | 分析深度：quick/standard/deep |
| `timeout_seconds` | int | 300 | 分析超时时间 |
| `cooldown_minutes` | int | 10 | 同一告警分析冷却时间 |
| `enabled` | bool | true | 启用/禁用分析 |
| `auto_trigger_severities` | list | ["critical", "high"] | 自动触发的严重级别 |

## 🔧 高级配置

### 自定义告警路由

```yaml
customPlaybooks:
  - name: "CriticalAlerts"
    triggers:
      - on_prometheus_alert:
          severity: "critical"
    actions:
      - create_finding:
          title: "🚨 Critical Alert: $alert_name"
          severity: CRITICAL
      - holmes_analyze:
          depth: "deep"
          timeout_seconds: 600
    sinks:
      - "central_hub_sink"
      - "slack_sink"  # 同时发送到Slack

  - name: "PodIssues"
    triggers:
      - on_pod_crash_loop: {}
      - on_pod_oom_killed: {}
    actions:
      - logs_enricher: {}
      - pod_events_enricher: {}
      - holmes_analyze:
          depth: "standard"
    sinks:
      - "central_hub_sink"
```

### 条件触发

```yaml
customPlaybooks:
  - name: "ProductionOnlyAnalysis"
    triggers:
      - on_prometheus_alert:
          namespace_prefix: "prod-"
    actions:
      - holmes_analyze:
          enabled: true
          depth: "deep"
    sinks:
      - "central_hub_sink"

  - name: "StagingQuickAnalysis"
    triggers:
      - on_prometheus_alert:
          namespace_prefix: "staging-"
    actions:
      - holmes_analyze:
          enabled: true
          depth: "quick"
          cooldown_minutes: 5
    sinks:
      - "central_hub_sink"
```

## 🔒 安全配置

### HMAC密钥管理

建议使用Kubernetes Secret管理HMAC密钥：

```bash
# 创建Secret
kubectl create secret generic central-hub-secret \
  --from-literal=hmac-secret=your-secret-key

# 在Robusta配置中引用
globalConfig:
  central_hub_hmac_secret: 
    secretKeyRef:
      name: central-hub-secret
      key: hmac-secret
```

### 网络安全

确保Central Hub的网络访问安全：

```yaml
# 使用内部服务地址
globalConfig:
  central_hub_url: "http://central-hub.robusta.svc.cluster.local:8080"
```

## 🐛 故障排除

### 常见问题

1. **HMAC签名验证失败**
   - 检查密钥是否正确
   - 确认时间同步

2. **连接超时**
   - 检查网络连通性
   - 调整timeout参数

3. **分析不触发**
   - 检查severity配置
   - 验证rate limiting设置

### 日志调试

启用详细日志：

```yaml
# 在Robusta配置中
runner:
  log_level: DEBUG
```

查看日志：

```bash
kubectl logs -f deployment/robusta-runner -n robusta
```

## 📚 API参考

详细的API文档请参考：
- [Central Hub API文档](../docs/api.md)
- [Robusta Action开发指南](https://docs.robusta.dev/master/playbook-reference/actions/develop-actions/)

## 🤝 贡献

欢迎提交Issue和Pull Request！

## 📄 许可证

MIT License

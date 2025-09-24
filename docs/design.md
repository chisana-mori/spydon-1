
---

# 设计说明文档

**项目名称**：Kubernetes 多集群告警集中管理与自动根因分析平台
**版本**：v1.0
**日期**：2025-09-23
**作者**：xxx

---

## 1. 背景与目标

当前使用 **Prometheus + Alertmanager + Robusta OSS** 实现了告警收集与富化，但存在以下问题：

* **缺少统一视图**：多集群告警分散在不同的 Slack/Teams，无法集中查看与检索。
* **RCA 手动触发**：HolmesGPT 需人工调用，告警触发后无法第一时间得到自动根因分析。
* **数据留存不足**：历史 RCA 报告和上下文无法集中存档，影响复盘与知识沉淀。

**目标**：

1. 构建一个集中页面（中央 Hub），统一接收各集群 Robusta 富化后的告警。
2. 在告警触发时，自动调用 **HolmesGPT** 进行根因诊断，并将结果写回富化事件。
3. 保证系统具备 **多集群接入、安全隔离、节流与幂等、历史留存与检索** 的能力。

---

## 2. 系统架构

### 2.1 总体架构图

```
[Cluster A..N]
Prometheus → Alertmanager → Robusta (playbooks + enrich)
   │                                 │
   │ (1) central_webhook sink        │ (2) holmes_analyze action
   ▼                                 ▼
Central Hub Ingest API           HolmesGPT Service
   │                                 │
   └──> PostgreSQL + MinIO           │
          │                          │
          └──> Web UI (Next.js) <────┘
```

### 2.2 架构说明

* **Robusta OSS**：继续作为集群内的富化引擎，负责监听 Alertmanager 告警，增加上下文（Pod/日志/图表），并通过 sink/action 扩展与外部交互。
* **central\_webhook sink**：将富化后的事件推送到中央 Hub，带上 `cluster_id` 标识。
* **holmes\_analyze action**：在告警触发时收集上下文，调用 HolmesGPT 进行根因诊断，将结果富化到当前告警，同时推送到中央 Hub。
* **Central Hub**：轻量后端 + UI：

  * Ingest API：接收事件与 RCA 结果，存库。
  * UI：集中浏览告警、历史 RCA、运行 HolmesGPT。
  * 存储：Postgres（结构化）、MinIO（附件/报告）。
* **HolmesGPT**：自托管服务，暴露 `/analyze` API，输入上下文，输出 RCA 结果。

---

## 3. 数据流设计

### 3.1 告警流

1. Alertmanager 触发告警，推送到 Robusta。
2. Robusta 富化事件，触发 sink：

   * **Slack/Teams** → 通知。
   * **central\_webhook** → Hub。
3. 若满足条件（severity=critical），自动执行 `holmes_analyze` action：

   * 收集上下文（Pod/事件/日志/指标/Grafana 链接）。
   * 调用 HolmesGPT，获取 RCA。
   * 将 RCA 写入富化事件 → 下游通知 + 推送 Hub。

### 3.2 数据存储

* `alerts`：富化后的告警元数据（fingerprint、labels、annotations、cluster\_id、状态）。
* `rca_runs`：HolmesGPT RCA 结果（summary、suspects、recommendations、attachments）。
* `clusters`：集群注册信息。
* `audit_log`：用户操作与系统 RCA 触发记录。
* `artifacts`（MinIO/S3）：日志片段、图表截图、RCA 报告（HTML/PDF）。

---

## 4. 组件设计

### 4.1 Robusta 扩展

#### central\_webhook sink

* 功能：将富化事件 POST 到 Hub。
* 安全：HMAC-SHA256 签名 + gzip 压缩。
* 配置：

```yaml
sinksConfig:
  - central_webhook:
      url: https://hub.example.com/api/v1/ingest
      token: ${CENTRAL_TOKEN}
      cluster_id: cluster-a
```

#### holmes\_analyze action

* 功能：自动触发 HolmesGPT RCA。
* 策略：

  * 按 severity/label 过滤。
  * 指纹去重（Alertmanager fingerprint + cluster\_id）。
  * 冷却时间（默认 10 分钟）。
  * 超时与失败回退为“enrich-only”。
* 配置示例：

```yaml
playbooks:
  - triggers:
      on_prometheus_alert:
        severity: critical
    actions:
      - holmes_analyze:
          depth: standard
          cooldown: 10m
          timeout_seconds: 60
          on_error: enrich-only
```

### 4.2 Central Hub

* **Ingest API**

  * `/api/v1/ingest/alert`：接收告警。
  * `/api/v1/ingest/rca`：接收 RCA 结果。
* **Query API**

  * `/api/v1/alerts?cluster_id=&severity=`：检索告警。
  * `/api/v1/rca/{alert_id}`：查看 RCA 报告。
* **UI**

  * 总览页：集群/严重级别分组。
  * 告警详情页：富化内容 + “RCA 历史”。
  * 集群管理页：心跳、安装指引。

### 4.3 HolmesGPT

* 部署方式：K8s Deployment + Service。
* API：`POST /analyze`，输入上下文 JSON，输出 RCA。
* 限流：每请求最大 60s，队列并发上限 5。

---

## 5. 安全设计

* **通信安全**：

  * sink→Hub：HMAC-SHA256 + TLS。
  * Hub→HolmesGPT：内网调用，禁止公网暴露。
* **权限控制**：

  * Robusta 只读 K8s 资源（Pod、事件、日志）。
  * HolmesGPT 仅采集必要上下文。
* **审计**：所有 RCA 运行记录入库，包含：触发条件、输入上下文概要、输出摘要。

---

## 6. 异常处理与风控

* **去重/冷却**：同一告警指纹 10 分钟内只触发一次 HolmesGPT。
* **超时降级**：HolmesGPT 超时 → 仅保留富化信息。
* **熔断**：连续失败 5 次 → 暂停 30 分钟自动触发。
* **人工补触发**：UI/Slack 命令可手动再跑 RCA。

---

## 7. 部署方案

1. **集群内**：

   * 安装 Robusta（Helm）。
   * 启用 `central_webhook sink` 和 `holmes_analyze action`。
2. **集中服务**：

   * 部署 Central Hub（Gin + Mysql + MinIO + Next.js + shadcnUI + thailand css4）。
   * 配置 OIDC 登录（Keycloak/Auth0）。
3. **HolmesGPT**：

   * 部署至集中集群或独立命名空间。
   * 配置访问 Prometheus/Loki 的凭证。

---

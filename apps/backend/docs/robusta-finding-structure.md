# Robusta Finding 数据结构与处理

## 概述

`IngestRobustaFinding` 方法已优化，支持完整的 Robusta Finding 结构体解析，并将 enrichments 数据存储到 MinIO。

## Finding 结构

```json
{
  "id": "uuid",
  "title": "告警标题",
  "description": "告警描述",
  "severity": "HIGH",
  "source": "PROMETHEUS",
  "aggregation_key": "Alert-xxx",
  "finding_type": "ISSUE",
  "failure": true,
  "fingerprint": "sha256哈希",
  "creation_date": "时间戳",
  "starts_at": "datetime",
  "ends_at": "datetime",
  "subject": {
    "name": "pod-name",
    "subject_type": "pod",
    "namespace": "default",
    "node": "node-name",
    "container": "container-name",
    "labels": {"app": "xxx"},
    "annotations": {"key": "value"}
  },
  "enrichments": [
    {
      "blocks": [
        {
          "type": "MarkdownBlock",
          "text": "日志内容..."
        },
        {
          "type": "FileBlock",
          "filename": "logs.txt",
          "contents": "base64编码的内容"
        },
        {
          "type": "TableBlock",
          "headers": ["Name", "Status", "Age"],
          "rows": [["pod-1", "Running", "2d"]]
        },
        {
          "type": "JsonBlock",
          "json": {"key": "value"}
        },
        {
          "type": "HeaderBlock",
          "title": "Section Title"
        },
        {
          "type": "ListBlock",
          "items": ["item1", "item2"]
        },
        {
          "type": "KubernetesDiffBlock",
          "old": "old yaml",
          "new": "new yaml",
          "diffs": [{"path": "/spec/replicas", "old": 1, "new": 3}]
        }
      ],
      "annotations": {},
      "enrichment_type": "logs",
      "title": "日志信息"
    }
  ],
  "links": [
    {
      "url": "https://...",
      "name": "链接名称",
      "type": "PROMETHEUS_GENERATOR_URL"
    }
  ],
  "service": "服务名",
  "service_key": "namespace/service",
  "investigate_uri": "Robusta UI链接",
  "add_silence_url": true,
  "silence_labels": {"alertname": "xxx"}
}
```

## 架构设计

### 面向对象的转换器模式

代码采用了优雅的转换器模式，而不是使用 map 进行参数解析：

```go
// 1. 解析 Finding 结构体
var finding RobustaFinding
json.Unmarshal(raw, &finding)

// 2. 创建转换器对象
converter := NewFindingToAlertConverter(ctx, handler, &finding, raw)

// 3. 执行转换
alert, err := converter.Convert()
```

### FindingToAlertConverter 转换器

转换器封装了所有转换逻辑，每个步骤都是独立的方法：

- `extractClusterID()` - 提取集群ID
- `buildTitle()` - 构建标题
- `buildDescription()` - 构建描述
- `normalizeSeverity()` - 标准化严重级别
- `buildAggregationKey()` - 构建聚合键
- `buildFingerprint()` - 构建指纹
- `parseTimes()` - 解析时间
- `buildLabels()` - 构建标签
- `buildAnnotations()` - 构建注解
- `saveRawPayload()` - 保存原始数据
- `processEnrichments()` - 处理enrichments
- `serializeMetadata()` - 序列化元数据
- `determineStatus()` - 确定状态

优势：
- **清晰的职责分离**：每个方法只负责一个转换步骤
- **易于测试**：可以单独测试每个转换方法
- **易于扩展**：添加新的转换逻辑只需添加新方法
- **类型安全**：使用结构体属性而不是 map，编译时检查

## 处理流程

### 1. 解析 Finding

- 尝试解析为完整的 `RobustaFinding` 结构体
- 如果解析失败，回退到兼容模式（`ingestRobustaFindingLegacy`）

### 2. 提取 Cluster ID

优先级顺序：
1. HTTP 头 `X-Robusta-Cluster-ID`
2. `subject.labels` 中的集群标识
3. `silence_labels` 中的集群标识
4. `description` 文本解析
5. 默认值 `kind`

### 3. 标准化数据

- **Severity**: 映射为 `low/medium/high/critical`
- **Title**: 优先使用 `finding.title`，否则从 `subject` 构建
- **Labels**: 合并 `subject.labels` 和 `silence_labels`，添加元数据
- **Annotations**: 合并 `subject.annotations`，添加 links 和 URI

### 4. 处理 Enrichments

#### 支持的 Block 类型

| Block 类型 | 描述 | 存储格式 |
|-----------|------|---------|
| **MarkdownBlock** | 文本内容（如日志） | `.md` 文件 |
| **FileBlock** | 文件内容（base64 编码） | 原始文件（解码后） |
| **TableBlock** | 表格数据 | JSON（headers + rows） |
| **JsonBlock** | JSON 数据 | JSON 文件 |
| **HeaderBlock** | 标题 | 仅在 enrichment JSON 中 |
| **ListBlock** | 列表项 | JSON（items 数组） |
| **KubernetesDiffBlock** | YAML 差异 | JSON（old/new/diffs） |

#### Enrichments 存储结构

```
enrichments/{cluster_id}/{fingerprint}/
  ├── {enrichment_type}_0.json              # enrichment 完整数据
  ├── {enrichment_type}_0_markdown_0.md     # MarkdownBlock
  ├── {enrichment_type}_0_files/            # FileBlock 文件
  │   ├── logs.txt
  │   └── metrics.json
  ├── {enrichment_type}_0_table_1.json      # TableBlock
  ├── {enrichment_type}_0_json_2.json       # JsonBlock
  ├── {enrichment_type}_0_list_3.json       # ListBlock
  ├── {enrichment_type}_0_diff_4.json       # KubernetesDiffBlock
  ├── {enrichment_type}_1.json
  └── ...
```

#### Block 处理逻辑

每个 enrichment 包含：
1. **完整 JSON**: 包含所有 blocks、annotations 等
2. **独立 Block 文件**: 根据类型单独存储

- **MarkdownBlock**: 保存为 `.md` 文件，便于阅读
- **FileBlock**: 解码 base64 内容，按原文件名存储
- **TableBlock**: 保存为 JSON，包含 headers 和 rows
- **JsonBlock**: 直接保存 JSON 数据
- **ListBlock**: 保存为 JSON 数组
- **KubernetesDiffBlock**: 保存 old/new YAML 和差异详情
- **HeaderBlock**: 仅作为元数据，不单独存储

#### Enrichment Keys 映射

存储的 key 映射保存在 Alert 的 `annotations.enrichment_keys` 中：
```json
{
  "logs_0": "enrichments/kind/abc123/logs_0.json",
  "logs_0_block_0": "enrichments/kind/abc123/logs_0_markdown_0.md",
  "logs_0_block_1": "enrichments/kind/abc123/logs_0_files/pod-logs.txt",
  "metrics_1": "enrichments/kind/abc123/metrics_1.json",
  "metrics_1_block_0": "enrichments/kind/abc123/metrics_1_table_0.json",
  "diff_2": "enrichments/kind/abc123/diff_2.json",
  "diff_2_block_0": "enrichments/kind/abc123/diff_2_diff_0.json"
}
```

### 5. 创建 Alert

将 Finding 转换为内部 Alert 模型：
- 使用标准化的字段
- 保存原始 payload 到 MinIO
- 关联 enrichment keys
- 确保集群存在

## 调试日志

方法会输出详细的调试日志：
- Finding 基本信息（ID、标题、严重级别等）
- Subject 详情
- Enrichments 数量和类型
- 每个 enrichment 和文件的存储路径

查看日志：
```bash
tail -f apps/backend/server.log | grep "Webhook"
```

## API 响应

成功响应：
```json
{
  "message": "Robusta告警接收成功",
  "alert_id": "uuid"
}
```

## 代码示例

### 扩展转换器

如果需要添加自定义转换逻辑，只需扩展转换器：

```go
// 添加自定义标签
func (c *FindingToAlertConverter) addCustomLabels() {
    // 添加环境标签
    if c.finding.Subject != nil {
        if env, ok := c.finding.Subject.Labels["environment"].(string); ok {
            c.labels["env"] = env
        }
    }

    // 添加团队标签
    if team, ok := c.finding.Subject.Labels["team"].(string); ok {
        c.labels["team"] = team
    }
}

// 在 Convert() 方法中调用
func (c *FindingToAlertConverter) Convert() (*models.Alert, error) {
    c.extractClusterID()
    c.buildTitle()
    // ... 其他步骤
    c.buildLabels()
    c.addCustomLabels()  // 添加自定义逻辑
    // ... 继续其他步骤
}
```

### 自定义 Block 处理

扩展 `processEnrichments` 方法来处理自定义 block 类型：

```go
case "CustomBlock":
    if block.CustomData != nil {
        customJSON, _ := json.MarshalIndent(block.CustomData, "", "  ")
        filePath := fmt.Sprintf("%s/%s_custom_%d.json", basePath, enrichmentID, j)
        if key, err := h.storageService.Save(ctx, filePath, customJSON, "application/json"); err != nil {
            log.Printf("[Webhook] 保存CustomBlock失败: %v", err)
        } else {
            enrichmentKeys[blockKey] = key
        }
    }
```

## 检索 Enrichments

### 从 Alert 获取 Enrichment Keys

```go
// 从 Alert 的 annotations 中获取 enrichment keys
var annotations map[string]interface{}
json.Unmarshal(alert.Annotations, &annotations)

enrichmentKeysJSON, _ := annotations["enrichment_keys"].(string)
var enrichmentKeys map[string]string
json.Unmarshal([]byte(enrichmentKeysJSON), &enrichmentKeys)

// enrichmentKeys 包含所有存储的文件路径
for name, key := range enrichmentKeys {
    // 使用 storageService 检索文件
    content, err := storageService.Get(ctx, key)
    // 处理内容...
}
```

### 示例：获取日志内容

```go
// 获取完整的 enrichment JSON
logsKey := enrichmentKeys["logs_0"]
logsJSON, _ := storageService.Get(ctx, logsKey)

// 获取 Markdown 格式的日志
markdownKey := enrichmentKeys["logs_0_block_0"]
markdownContent, _ := storageService.Get(ctx, markdownKey)

// 获取日志文件
logFileKey := enrichmentKeys["logs_0_block_1"]
logFileContent, _ := storageService.Get(ctx, logFileKey)
```

### 示例：获取表格数据

```go
tableKey := enrichmentKeys["metrics_1_block_0"]
tableJSON, _ := storageService.Get(ctx, tableKey)

var tableData struct {
    Headers []string   `json:"headers"`
    Rows    [][]string `json:"rows"`
}
json.Unmarshal(tableJSON, &tableData)

// 使用表格数据
for _, row := range tableData.Rows {
    // 处理每一行...
}
```

## 兼容性

- 支持完整的 Robusta Finding 结构
- 向后兼容简单格式（通过 `ingestRobustaFindingLegacy`）
- 自动处理缺失字段
- 智能推断 cluster_id 和其他元数据
- 支持所有 7 种 Block 类型
- 自动识别文件类型和内容格式

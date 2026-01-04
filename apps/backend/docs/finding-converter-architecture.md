# Finding 转换器架构设计

## 设计理念

从基于 `map[string]interface{}` 的参数解析重构为面向对象的转换器模式，提供更优雅、可维护的代码结构。

## 核心组件

### 0. Gin ShouldBindJSON

使用 Gin 框架的 `ShouldBindJSON` 方法来解析请求体：

```go
var finding RobustaFinding
if err := c.ShouldBindJSON(&finding); err != nil {
    // 自动处理解析错误
    c.JSON(http.StatusBadRequest, gin.H{
        "error": "无效的Finding格式",
        "details": err.Error(),
    })
    return
}
```

优势：
- **自动解析**：Gin 自动将 JSON 解析为结构体
- **错误处理**：自动捕获 JSON 格式错误
- **验证支持**：可以使用 `binding` 标签进行字段验证
- **性能优化**：Gin 内部优化了 JSON 解析性能

### 1. RobustaFinding 结构体

完整的类型定义，映射 Robusta webhook 的所有字段：

```go
type RobustaFinding struct {
    ID              string
    Title           string
    Description     string
    Severity        string
    Source          string
    Subject         *RobustaSubject
    Enrichments     []RobustaEnrichment
    Links           []RobustaLink
    // ... 更多字段
}
```

### 2. FindingToAlertConverter 转换器

封装所有转换逻辑的对象：

```go
type FindingToAlertConverter struct {
    ctx            *gin.Context
    handler        *IngestHandler
    finding        *RobustaFinding
    rawPayload     []byte

    // 转换后的字段
    clusterID      string
    title          string
    severity       string
    labels         map[string]interface{}
    annotations    map[string]interface{}
    // ... 更多字段
}
```

## 转换流程

### 旧方式（基于 map）

```go
// ❌ 不优雅：使用 map 和类型断言
body := map[string]interface{}{}
json.Unmarshal(raw, &body)

title, _ := body["title"].(string)
if title == "" {
    if subject, ok := body["subject"].(map[string]interface{}); ok {
        name, _ := subject["name"].(string)
        // 复杂的嵌套类型断言...
    }
}
```

问题：
- 大量类型断言，容易出错
- 没有编译时类型检查
- 代码难以理解和维护
- 难以测试

### 新方式（面向对象 + Gin 最佳实践）

```go
// ✅ 优雅：使用 Gin 的 ShouldBindJSON 和转换器
var finding RobustaFinding
if err := c.ShouldBindJSON(&finding); err != nil {
    c.JSON(http.StatusBadRequest, gin.H{
        "error": "无效的Finding格式",
        "details": err.Error(),
    })
    return
}

converter := NewFindingToAlertConverter(ctx, handler, &finding, rawBody)
alert, err := converter.Convert()
```

优势：
- **类型安全**：编译时检查，避免类型断言错误
- **Gin 集成**：使用 `ShouldBindJSON` 自动验证和绑定
- **清晰的职责分离**：每个方法只负责一个转换步骤
- **易于测试和扩展**：独立的方法可以单独测试
- **代码可读性高**：清晰的方法调用链
- **自动验证**：Gin 自动处理 JSON 解析错误

## 转换步骤

每个转换步骤都是独立的方法，按顺序执行：

```go
func (c *FindingToAlertConverter) Convert() (*models.Alert, error) {
    c.extractClusterID()       // 1. 提取集群ID
    c.buildTitle()             // 2. 构建标题
    c.buildDescription()       // 3. 构建描述
    c.normalizeSeverity()      // 4. 标准化严重级别
    c.buildAggregationKey()    // 5. 构建聚合键
    c.buildFingerprint()       // 6. 构建指纹
    c.parseTimes()             // 7. 解析时间
    c.buildLabels()            // 8. 构建标签
    c.buildAnnotations()       // 9. 构建注解

    payloadKey := c.saveRawPayload()        // 10. 保存原始数据
    enrichmentKeys := c.processEnrichments() // 11. 处理enrichments

    labelsJSON, annotationsJSON := c.serializeMetadata() // 12. 序列化
    status := c.determineStatus()                        // 13. 确定状态

    return c.buildAlert(payloadKey, labelsJSON, annotationsJSON, status), nil
}
```

## 方法详解

### extractClusterID()

从多个来源按优先级提取集群ID：

1. HTTP 请求头 `X-Robusta-Cluster-ID`
2. `finding.Subject.Labels` 中的集群标识
3. `finding.SilenceLabels` 中的集群标识
4. 从 `description` 文本解析
5. 默认值 `"kind"`

```go
func (c *FindingToAlertConverter) extractClusterID() {
    if clusterID := c.ctx.GetHeader("X-Robusta-Cluster-ID"); clusterID != "" {
        c.clusterID = clusterID
        return
    }
    // ... 其他来源
    c.clusterID = "kind" // 默认值
}
```

### buildTitle()

智能构建标题：

```go
func (c *FindingToAlertConverter) buildTitle() {
    c.title = c.finding.Title

    if c.title == "" && c.finding.Subject != nil {
        c.title = fmt.Sprintf("%s: %s",
            c.finding.Subject.SubjectType,
            c.finding.Subject.Name)
        if c.finding.Subject.Namespace != "" {
            c.title = fmt.Sprintf("%s (%s)", c.title, c.finding.Subject.Namespace)
        }
    }

    if c.title == "" {
        c.title = "Robusta Finding"
    }
}
```

### buildLabels()

合并多个来源的标签并添加元数据：

```go
func (c *FindingToAlertConverter) buildLabels() {
    // 合并 subject.labels
    if c.finding.Subject != nil && c.finding.Subject.Labels != nil {
        for k, v := range c.finding.Subject.Labels {
            c.labels[k] = v
        }
    }

    // 合并 silence_labels
    if c.finding.SilenceLabels != nil {
        for k, v := range c.finding.SilenceLabels {
            c.labels[k] = v
        }
    }

    // 添加元数据
    c.labels["source"] = c.finding.Source
    c.labels["finding_type"] = c.finding.FindingType
    c.labels["subject_type"] = c.finding.Subject.SubjectType
}
```

## 扩展性

### 添加新的转换步骤

只需添加新方法并在 `Convert()` 中调用：

```go
// 添加自定义标签处理
func (c *FindingToAlertConverter) enrichLabelsWithTeamInfo() {
    if team, ok := c.finding.Subject.Labels["team"].(string); ok {
        c.labels["team"] = team
        c.labels["team_email"] = fmt.Sprintf("%s@company.com", team)
    }
}

func (c *FindingToAlertConverter) Convert() (*models.Alert, error) {
    // ... 现有步骤
    c.buildLabels()
    c.enrichLabelsWithTeamInfo() // 添加新步骤
    // ... 继续
}
```

### 自定义转换逻辑

可以创建转换器的子类或包装器：

```go
type CustomFindingConverter struct {
    *FindingToAlertConverter
}

func (c *CustomFindingConverter) buildTitle() {
    // 自定义标题构建逻辑
    c.title = fmt.Sprintf("[%s] %s", c.clusterID, c.finding.Title)
}
```

## 测试

每个方法都可以独立测试：

```go
func TestBuildTitle(t *testing.T) {
    converter := &FindingToAlertConverter{
        finding: &RobustaFinding{
            Title: "Test Alert",
        },
    }

    converter.buildTitle()

    assert.Equal(t, "Test Alert", converter.title)
}
```

## 性能考虑

- **单次遍历**：每个数据源只遍历一次
- **延迟序列化**：只在最后序列化 JSON
- **避免重复计算**：转换结果存储在转换器字段中

## 最佳实践

1. **保持方法简单**：每个方法只做一件事
2. **使用有意义的命名**：方法名清楚表达其功能
3. **添加日志**：在关键步骤添加调试日志
4. **处理边界情况**：检查 nil 和空值
5. **文档化**：为复杂逻辑添加注释

## 对比总结

| 特性 | 旧方式（map） | 新方式（转换器） |
|------|-------------|----------------|
| 类型安全 | ❌ 运行时检查 | ✅ 编译时检查 |
| 可读性 | ❌ 复杂的类型断言 | ✅ 清晰的方法调用 |
| 可测试性 | ❌ 难以单元测试 | ✅ 易于单元测试 |
| 可维护性 | ❌ 逻辑分散 | ✅ 职责清晰 |
| 扩展性 | ❌ 难以扩展 | ✅ 易于扩展 |
| 性能 | 🟡 多次类型断言 | ✅ 直接访问字段 |

## 结论

通过引入 `FindingToAlertConverter` 转换器模式，代码变得：

- **更优雅**：面向对象设计，清晰的职责分离
- **更安全**：类型安全，减少运行时错误
- **更易维护**：每个方法独立，易于理解和修改
- **更易测试**：可以单独测试每个转换步骤
- **更易扩展**：添加新功能只需添加新方法

这是一个典型的重构案例，展示了如何将过程式代码转换为面向对象的设计。

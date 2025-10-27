# Gin ShouldBindJSON 最佳实践

## 为什么使用 ShouldBindJSON

在 Gin 框架中，`ShouldBindJSON` 是处理 JSON 请求体的推荐方式。

## 对比

### ❌ 不推荐：手动解析

```go
func (h *Handler) HandleRequest(c *gin.Context) {
    // 手动读取和解析
    raw, err := io.ReadAll(c.Request.Body)
    if err != nil {
        c.JSON(500, gin.H{"error": "读取失败"})
        return
    }
    
    var data MyStruct
    if err := json.Unmarshal(raw, &data); err != nil {
        c.JSON(400, gin.H{"error": "解析失败"})
        return
    }
    
    // 处理数据...
}
```

问题：
- 需要手动处理多个错误
- 代码冗长
- 没有利用 Gin 的内置功能
- 需要手动恢复 Body（如果需要多次读取）

### ✅ 推荐：使用 ShouldBindJSON

```go
func (h *Handler) HandleRequest(c *gin.Context) {
    var data MyStruct
    if err := c.ShouldBindJSON(&data); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "无效的请求格式",
            "details": err.Error(),
        })
        return
    }
    
    // 直接使用解析好的数据
    // data.Field1, data.Field2...
}
```

优势：
- 简洁明了
- 自动错误处理
- 利用 Gin 的优化
- 支持字段验证

## 字段验证

使用 `binding` 标签进行自动验证：

```go
type RobustaFinding struct {
    ID          string   `json:"id" binding:"required"`
    Title       string   `json:"title" binding:"required"`
    Severity    string   `json:"severity" binding:"required,oneof=INFO LOW MEDIUM HIGH CRITICAL"`
    Description string   `json:"description"`
    Source      string   `json:"source" binding:"required"`
}
```

验证规则：
- `required` - 必填字段
- `oneof` - 枚举值验证
- `min`, `max` - 长度或数值范围
- `email`, `url` - 格式验证

使用：
```go
var finding RobustaFinding
if err := c.ShouldBindJSON(&finding); err != nil {
    // err 会包含具体的验证错误信息
    c.JSON(http.StatusBadRequest, gin.H{
        "error": "验证失败",
        "details": err.Error(),
    })
    return
}
// 此时 finding 已经通过所有验证
```

## 需要原始数据的场景

如果需要保存原始 JSON（如存储到 MinIO），可以这样做：

```go
func (h *Handler) HandleRequest(c *gin.Context) {
    // 1. 先读取原始数据
    rawBody, err := io.ReadAll(c.Request.Body)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "读取失败"})
        return
    }
    
    // 2. 恢复 Body 供 ShouldBindJSON 使用
    c.Request.Body = io.NopCloser(bytes.NewBuffer(rawBody))
    
    // 3. 使用 ShouldBindJSON 解析
    var data MyStruct
    if err := c.ShouldBindJSON(&data); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "解析失败",
            "details": err.Error(),
        })
        return
    }
    
    // 4. 现在同时拥有原始数据和解析后的结构体
    // rawBody - 用于存储
    // data - 用于业务逻辑
    
    // 保存原始数据
    storageService.Save(ctx, "path", rawBody, "application/json")
    
    // 使用解析后的数据
    processData(data)
}
```

## 实际应用：IngestRobustaFinding

```go
func (h *IngestHandler) IngestRobustaFinding(c *gin.Context) {
    // 读取原始数据（用于存储到MinIO）
    rawBody, err := io.ReadAll(c.Request.Body)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{
            "error": "读取请求体失败",
            "code": "READ_BODY_ERROR",
        })
        return
    }
    
    // 恢复Body
    c.Request.Body = io.NopCloser(bytes.NewBuffer(rawBody))
    
    // 使用ShouldBindJSON解析
    var finding RobustaFinding
    if err := c.ShouldBindJSON(&finding); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "无效的Finding格式",
            "code": "INVALID_FINDING_FORMAT",
            "details": err.Error(),
        })
        return
    }
    
    // 打印调试信息
    finding.LogDebugInfo()
    
    // 创建转换器
    converter := NewFindingToAlertConverter(c, h, &finding, rawBody)
    alert, err := converter.Convert()
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{
            "error": "转换Finding失败",
            "code": "CONVERT_FINDING_ERROR",
            "details": err.Error(),
        })
        return
    }
    
    // 保存告警
    h.alertService.CreateOrUpdateAlert(alert)
    
    c.JSON(http.StatusOK, gin.H{
        "message": "Robusta告警接收成功",
        "alert_id": alert.ID,
    })
}
```

## 最佳实践总结

1. **优先使用 ShouldBindJSON**：除非有特殊需求，否则总是使用 Gin 的绑定方法
2. **添加验证标签**：使用 `binding` 标签进行字段验证
3. **统一错误处理**：为绑定错误提供清晰的错误消息
4. **保存原始数据**：如果需要原始 JSON，先读取再恢复 Body
5. **类型安全**：使用结构体而不是 `map[string]interface{}`
6. **文档化**：为结构体字段添加 JSON 标签和注释

## 性能考虑

- `ShouldBindJSON` 内部使用了优化的 JSON 解析器
- 避免多次解析同一个 JSON
- 如果需要原始数据，只读取一次然后恢复 Body
- 使用结构体比 map 更高效（编译时类型检查）

## 错误处理

```go
if err := c.ShouldBindJSON(&data); err != nil {
    // err 可能包含：
    // - JSON 语法错误
    // - 类型不匹配
    // - 验证失败
    
    // 可以根据错误类型返回不同的响应
    if _, ok := err.(*json.SyntaxError); ok {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "JSON 格式错误",
        })
    } else {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "数据验证失败",
            "details": err.Error(),
        })
    }
    return
}
```

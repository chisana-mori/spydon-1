package api

import (
    "context"
    "encoding/json"
    "net/http"
    "strconv"
    "strings"
    "time"

    "robusta-web/backend/internal/services"

    "github.com/gin-gonic/gin"
)

// KnowledgeHandler 知识库HTTP处理器
type KnowledgeHandler struct {
    svc *services.KnowledgeService
}

func NewKnowledgeHandler(svc *services.KnowledgeService) *KnowledgeHandler {
    return &KnowledgeHandler{svc: svc}
}

// 列表查询（支持分页和可选的规则名过滤）
func (h *KnowledgeHandler) List(c *gin.Context) {
    rule := strings.TrimSpace(c.Query("alert_rule_name"))
    pageStr := c.DefaultQuery("page", "1")
    pageSizeStr := c.DefaultQuery("page_size", "20")
    
    page, _ := strconv.Atoi(pageStr)
    pageSize, _ := strconv.Atoi(pageSizeStr)
    
    if page < 1 {
        page = 1
    }
    if pageSize < 1 || pageSize > 100 {
        pageSize = 20
    }
    
    // 如果有规则名，使用规则名查询
    if rule != "" {
        items, err := h.svc.GetByRule(rule, pageSize)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
            return
        }
        c.JSON(http.StatusOK, gin.H{
            "data": items, 
            "total": len(items),
            "page": page,
            "page_size": pageSize,
        })
        return
    }
    
    // 否则获取全部列表（分页）
    items, total, err := h.svc.List(page, pageSize)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{
        "data": items,
        "total": total,
        "page": page,
        "page_size": pageSize,
    })
}

// 查询（按规则名）- 保留向后兼容
func (h *KnowledgeHandler) QueryByRule(c *gin.Context) {
    rule := strings.TrimSpace(c.Query("alert_rule_name"))
    limitStr := c.DefaultQuery("limit", "1")
    limit, _ := strconv.Atoi(limitStr)
    if rule == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "alert_rule_name 不能为空"})
        return
    }
    items, err := h.svc.GetByRule(rule, limit)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{"data": items, "pagination": gin.H{"page": 1, "limit": limit, "total": len(items)}})
}

// 获取单条（包含manifest，可选）
func (h *KnowledgeHandler) GetByID(c *gin.Context) {
    id := c.Param("id")
    include := c.DefaultQuery("include_manifest", "false") == "true"

    art, err := h.svc.GetByID(id)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "未找到"})
        return
    }
    if !include {
        c.JSON(http.StatusOK, gin.H{"data": art})
        return
    }
    ctx, cancel := context.WithTimeout(c.Request.Context(), 20*time.Second)
    defer cancel()
    raw, err := h.svc.GetManifest(ctx, art.ObjectKey)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    var m map[string]interface{}
    _ = json.Unmarshal(raw, &m)
    c.JSON(http.StatusOK, gin.H{"data": gin.H{"article": art, "manifest": m}})
}

// 创建
func (h *KnowledgeHandler) Create(c *gin.Context) {
    var req struct {
        AlertRuleName string   `json:"alert_rule_name"`
        Title         string   `json:"title"`
        Tags          []string `json:"tags"`
    }
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    user := c.GetString("user_email")
    if user == "" { user = c.GetString("user_name") }
    if user == "" { user = "system" }
    ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
    defer cancel()
    art, err := h.svc.Create(ctx, req.AlertRuleName, req.Title, req.Tags, user)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{"data": art})
}

// 更新（写manifest）
func (h *KnowledgeHandler) Update(c *gin.Context) {
    id := c.Param("id")
    var payload services.Manifest
    if err := c.ShouldBindJSON(&payload); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    user := c.GetString("user_email")
    if user == "" { user = c.GetString("user_name") }
    if user == "" { user = "system" }
    ctx, cancel := context.WithTimeout(c.Request.Context(), 20*time.Second)
    defer cancel()
    art, err := h.svc.Update(ctx, id, payload, user)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{"data": art})
}

// 发布
func (h *KnowledgeHandler) Publish(c *gin.Context) {
    id := c.Param("id")
    var req struct{ ChangeSummary string `json:"change_summary"` }
    _ = json.NewDecoder(c.Request.Body).Decode(&req)
    user := c.GetString("user_email")
    if user == "" { user = c.GetString("user_name") }
    if user == "" { user = "system" }
    ctx, cancel := context.WithTimeout(c.Request.Context(), 20*time.Second)
    defer cancel()
    if err := h.svc.Publish(ctx, id, req.ChangeSummary, user); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{"data": "ok"})
}

// 预签名上传
func (h *KnowledgeHandler) PresignUpload(c *gin.Context) {
    var req struct {
        ArticleID  string `json:"article_id"`
        Filename   string `json:"filename"`
        ContentType string `json:"content_type"`
    }
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Minute)
    defer cancel()
    key, url, err := h.svc.PresignAssetPut(ctx, req.ArticleID, req.Filename, req.ContentType)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{"data": gin.H{"key": key, "url": url}})
}

// Delete 删除知识条目
func (h *KnowledgeHandler) Delete(c *gin.Context) {
    id := c.Param("id")
    
    ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
    defer cancel()
    
    if err := h.svc.Delete(ctx, id); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

package api

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"robusta-web/backend/internal/models"
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

type knowledgeCreateRequest struct {
	AlertRuleName string   `json:"alert_rule_name" binding:"required"`
	Tags          []string `json:"tags"`
}

type knowledgePublishRequest struct {
	ChangeSummary string `json:"change_summary"`
}

type knowledgePresignRequest struct {
	ArticleID   string `json:"article_id" binding:"required"`
	Filename    string `json:"filename" binding:"required"`
	ContentType string `json:"content_type" binding:"required"`
}

type knowledgeURI struct {
	ID uint64 `uri:"id" binding:"required,gt=0"`
}

func formatKnowledgeArticle(art models.KnowledgeArticle) gin.H {
	return gin.H{
		"id":                         models.FormatID(art.ID),
		"alert_rule_name":            art.AlertRuleName,
		"alert_rule_name_normalized": art.AlertRuleNameNormalized,
		"tags":                       art.Tags,
		"status":                     art.Status,
		"object_key":                 art.ObjectKey,
		"version":                    art.Version,
		"created_by":                 art.CreatedBy,
		"updated_by":                 art.UpdatedBy,
		"created_at":                 art.CreatedAt,
		"updated_at":                 art.UpdatedAt,
	}
}

func formatKnowledgeArticles(items []models.KnowledgeArticle) []gin.H {
	formatted := make([]gin.H, len(items))
	for i, item := range items {
		formatted[i] = formatKnowledgeArticle(item)
	}
	return formatted
}

// 列表查询（支持分页和可选的规则名过滤）
func (h *KnowledgeHandler) List(c *gin.Context) {
	var query struct {
		PaginationQuery
		AlertRuleName string `form:"alert_rule_name"`
	}
	if derr := bindQuery(c, &query); derr != nil {
		AbortWithDomainError(c, derr)
		return
	}
	if derr := query.PaginationQuery.validate(); derr != nil {
		AbortWithDomainError(c, derr)
		return
	}
	params := query.PaginationQuery.ToParams()
	rule := strings.TrimSpace(query.AlertRuleName)

	// 如果有规则名，使用规则名查询
	if rule != "" {
		items, err := h.svc.GetByRule(rule, params.PageSize)
		if err != nil {
			InternalError(c, "GET_BY_RULE_FAILED", err.Error())
			return
		}
		pagination := NewPagination(params.Page, params.PageSize, int64(len(items)))
		pagination.Sort = params.Sort
		SuccessPaginated(c, formatKnowledgeArticles(items), pagination)
		return
	}

	// 否则获取全部列表（分页）
	items, total, err := h.svc.List(params.Page, params.PageSize)
	if err != nil {
		InternalError(c, "LIST_FAILED", err.Error())
		return
	}

	pagination := NewPagination(params.Page, params.PageSize, total)
	pagination.Sort = params.Sort
	SuccessPaginated(c, formatKnowledgeArticles(items), pagination)
}

// 查询（按规则名）- 保留向后兼容
func (h *KnowledgeHandler) QueryByRule(c *gin.Context) {
	var query struct {
		AlertRuleName string `form:"alert_rule_name" binding:"required"`
		Limit         *int   `form:"limit" binding:"omitempty,gt=0"`
	}
	if derr := bindQuery(c, &query); derr != nil {
		AbortWithDomainError(c, derr)
		return
	}

	rule := strings.TrimSpace(query.AlertRuleName)
	if rule == "" {
		BadRequest(c, "MISSING_RULE_NAME", "alert_rule_name 不能为空")
		return
	}

	limit := 1
	if query.Limit != nil {
		limit = *query.Limit
	}

	items, err := h.svc.GetByRule(rule, limit)
	if err != nil {
		InternalError(c, "GET_BY_RULE_FAILED", err.Error())
		return
	}
	pagination := NewPagination(1, limit, int64(len(items)))
	SuccessPaginated(c, formatKnowledgeArticles(items), pagination)
}

// 获取单条（包含manifest，可选）
func (h *KnowledgeHandler) GetByID(c *gin.Context) {
	var uri knowledgeURI
	if err := c.ShouldBindUri(&uri); err != nil {
		BadRequest(c, "INVALID_ID", "无效的ID")
		return
	}
	var query struct {
		IncludeManifest bool `form:"include_manifest"`
	}
	if derr := bindQuery(c, &query); derr != nil {
		AbortWithDomainError(c, derr)
		return
	}
	include := query.IncludeManifest

	art, err := h.svc.GetByID(uri.ID)
	if err != nil {
		NotFound(c, "ARTICLE_NOT_FOUND", "未找到")
		return
	}
	if !include {
		Success(c, formatKnowledgeArticle(*art))
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 20*time.Second)
	defer cancel()
	raw, err := h.svc.GetManifest(ctx, art.ObjectKey)
	if err != nil {
		InternalError(c, "GET_MANIFEST_FAILED", err.Error())
		return
	}
	var m map[string]interface{}
	_ = json.Unmarshal(raw, &m)
	Success(c, gin.H{"article": formatKnowledgeArticle(*art), "manifest": m})
}

// 创建
func (h *KnowledgeHandler) Create(c *gin.Context) {
	var req knowledgeCreateRequest
	if err := bindJSON(c, &req); err != nil {
		AbortWithDomainError(c, err)
		return
	}
	user := c.GetString("user_email")
	if user == "" {
		user = c.GetString("user_name")
	}
	if user == "" {
		user = "system"
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()
	art, err := h.svc.Create(ctx, req.AlertRuleName, req.Tags, user)
	if err != nil {
		InternalError(c, "CREATE_FAILED", err.Error())
		return
	}
	Created(c, formatKnowledgeArticle(*art))
}

// 更新（写manifest）
func (h *KnowledgeHandler) Update(c *gin.Context) {
	var uri knowledgeURI
	if err := c.ShouldBindUri(&uri); err != nil {
		BadRequest(c, "INVALID_ID", "无效的ID")
		return
	}
	var payload services.Manifest
	if err := bindJSON(c, &payload); err != nil {
		AbortWithDomainError(c, err)
		return
	}
	user := c.GetString("user_email")
	if user == "" {
		user = c.GetString("user_name")
	}
	if user == "" {
		user = "system"
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 20*time.Second)
	defer cancel()
	art, err := h.svc.Update(ctx, uri.ID, payload, user)
	if err != nil {
		InternalError(c, "UPDATE_FAILED", err.Error())
		return
	}
	Success(c, formatKnowledgeArticle(*art))
}

// 发布
func (h *KnowledgeHandler) Publish(c *gin.Context) {
	var uri knowledgeURI
	if err := c.ShouldBindUri(&uri); err != nil {
		BadRequest(c, "INVALID_ID", "无效的ID")
		return
	}
	var req knowledgePublishRequest
	if err := bindJSON(c, &req); err != nil {
		AbortWithDomainError(c, err)
		return
	}
	user := c.GetString("user_email")
	if user == "" {
		user = c.GetString("user_name")
	}
	if user == "" {
		user = "system"
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 20*time.Second)
	defer cancel()
	if err := h.svc.Publish(ctx, uri.ID, req.ChangeSummary, user); err != nil {
		InternalError(c, "PUBLISH_FAILED", err.Error())
		return
	}
	SuccessWithMessage(c, "发布成功", nil)
}

// 预签名上传
func (h *KnowledgeHandler) PresignUpload(c *gin.Context) {
	var req knowledgePresignRequest
	if err := bindJSON(c, &req); err != nil {
		AbortWithDomainError(c, err)
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Minute)
	defer cancel()
	key, url, err := h.svc.PresignAssetPut(ctx, req.ArticleID, req.Filename, req.ContentType)
	if err != nil {
		InternalError(c, "PRESIGN_FAILED", err.Error())
		return
	}
	Success(c, gin.H{"key": key, "url": url})
}

// Delete 删除知识条目
func (h *KnowledgeHandler) Delete(c *gin.Context) {
	var uri knowledgeURI
	if err := c.ShouldBindUri(&uri); err != nil {
		BadRequest(c, "INVALID_ID", "无效的ID")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	if err := h.svc.Delete(ctx, uri.ID); err != nil {
		InternalError(c, "DELETE_FAILED", err.Error())
		return
	}

	SuccessWithMessage(c, "删除成功", nil)
}

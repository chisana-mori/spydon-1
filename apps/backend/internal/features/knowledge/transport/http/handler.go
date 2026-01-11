package http

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"robusta-web/backend/internal/features/knowledge/services"
	"robusta-web/backend/internal/middleware"
	"robusta-web/backend/internal/models"
	httpx "robusta-web/backend/internal/transport/httpx"

	"github.com/gin-gonic/gin"
)

// Handler 知识库HTTP处理器
type Handler struct {
	svc *services.KnowledgeService
}

// New 创建新的知识库处理器
func New(svc *services.KnowledgeService) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes 注册知识库路由
// v1 is /api/v1 group; admin is /api/v1/admin group with admin middlewares already applied.
func (h *Handler) RegisterRoutes(v1, admin *gin.RouterGroup) {
	// /api/v1/knowledge (public read, admin write)
	knowledgeGroup := v1.Group("/knowledge")
	{
		knowledgeGroup.GET("", h.List)
		knowledgeGroup.GET("/query", h.QueryByRule)
		knowledgeGroup.GET("/:id", h.GetByID)
	}

	adminKnowledgeGroup := admin.Group("/knowledge")
	adminKnowledgeGroup.Use(middleware.AuditLogMiddleware())
	{
		adminKnowledgeGroup.POST("", h.Create)
		adminKnowledgeGroup.PUT("/:id", h.Update)
		adminKnowledgeGroup.POST("/:id/publish", h.Publish)
		adminKnowledgeGroup.POST("/presign-upload", h.PresignUpload)
		adminKnowledgeGroup.DELETE("/:id", h.Delete)
	}
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

// List 列表查询（支持分页和可选的规则名过滤）
// @Summary 分页查询知识库
// @Description 从知识库中分页检索告警相关的文章记录。支持通过告警规则名称进行模糊匹配过滤。返回结果包含文章的基本元数据，如ID、关联规则、状态、版本号及创建信息，帮助用户快速定位所需的运维知识条目。
// @Tags Knowledge
// @Produce json
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Param alert_rule_name query string false "按告警规则名过滤"
// @Success 200 {object} httpx.Response{data=[]object}
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /knowledge [get]
func (h *Handler) List(c *gin.Context) {
	var query struct {
		httpx.PaginationQuery
		AlertRuleName string `form:"alert_rule_name"`
	}
	if derr := httpx.BindQuery(c, &query); derr != nil {
		httpx.AbortWithDomainError(c, derr)
		return
	}
	if derr := query.PaginationQuery.Validate(); derr != nil {
		httpx.AbortWithDomainError(c, derr)
		return
	}
	params := query.PaginationQuery.ToParams()
	rule := strings.TrimSpace(query.AlertRuleName)

	if rule != "" {
		items, err := h.svc.GetByRule(rule, params.PageSize)
		if err != nil {
			httpx.InternalError(c, "GET_BY_RULE_FAILED", err.Error())
			return
		}
		pagination := httpx.NewPagination(params.Page, params.PageSize, int64(len(items)))
		pagination.Sort = params.Sort
		httpx.SuccessPaginated(c, formatKnowledgeArticles(items), pagination)
		return
	}

	items, total, err := h.svc.List(params.Page, params.PageSize)
	if err != nil {
		httpx.InternalError(c, "LIST_FAILED", err.Error())
		return
	}

	pagination := httpx.NewPagination(params.Page, params.PageSize, total)
	pagination.Sort = params.Sort
	httpx.SuccessPaginated(c, formatKnowledgeArticles(items), pagination)
}

// QueryByRule 查询（按规则名）- 保留向后兼容
// @Summary 按规则名精准查询
// @Description 该接口是专为前端向后兼容设计的精准查询路由。通过提供完整的告警规则名称，快速获取该规则下关联的所有知识库文章，并支持限制返回的文章数量。常用于在告警详情页中即时展示对应的排查手册。
// @Tags Knowledge
// @Produce json
// @Param alert_rule_name query string true "完整的告警规则名称"
// @Param limit query int false "限制返回条数"
// @Success 200 {object} httpx.Response{data=[]object}
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /knowledge/query [get]
func (h *Handler) QueryByRule(c *gin.Context) {
	var query struct {
		AlertRuleName string `form:"alert_rule_name" binding:"required"`
		Limit         *int   `form:"limit" binding:"omitempty,gt=0"`
	}
	if derr := httpx.BindQuery(c, &query); derr != nil {
		httpx.AbortWithDomainError(c, derr)
		return
	}

	rule := strings.TrimSpace(query.AlertRuleName)
	if rule == "" {
		httpx.BadRequest(c, "MISSING_RULE_NAME", "alert_rule_name 不能为空")
		return
	}

	limit := 1
	if query.Limit != nil {
		limit = *query.Limit
	}

	items, err := h.svc.GetByRule(rule, limit)
	if err != nil {
		httpx.InternalError(c, "GET_BY_RULE_FAILED", err.Error())
		return
	}
	pagination := httpx.NewPagination(1, limit, int64(len(items)))
	httpx.SuccessPaginated(c, formatKnowledgeArticles(items), pagination)
}

// GetByID 获取单条（包含manifest，可选）
// @Summary 获取知识文章详情
// @Description 根据唯一ID获取特定知识库文章的详细信息。用户可以选择是否包含文章的Manifest内容（即JSON格式的详细排查指引和剧本）。该接口支持深度的资源检索，并通过集成对象存储服务来展示丰富的文章正文数据。
// @Tags Knowledge
// @Produce json
// @Param id path string true "文章ID"
// @Param include_manifest query bool false "是否包含详细配置内容"
// @Success 200 {object} httpx.Response{data=object}
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /knowledge/{id} [get]
func (h *Handler) GetByID(c *gin.Context) {
	var uri knowledgeURI
	if err := c.ShouldBindUri(&uri); err != nil {
		httpx.BadRequest(c, "INVALID_ID", "无效的ID")
		return
	}
	var query struct {
		IncludeManifest bool `form:"include_manifest"`
	}
	if derr := httpx.BindQuery(c, &query); derr != nil {
		httpx.AbortWithDomainError(c, derr)
		return
	}
	include := query.IncludeManifest

	art, err := h.svc.GetByID(uri.ID)
	if err != nil {
		httpx.NotFound(c, "ARTICLE_NOT_FOUND", "未找到")
		return
	}
	if !include {
		httpx.Success(c, formatKnowledgeArticle(*art))
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 20*time.Second)
	defer cancel()
	raw, err := h.svc.GetManifest(ctx, art.ObjectKey)
	if err != nil {
		httpx.InternalError(c, "GET_MANIFEST_FAILED", err.Error())
		return
	}
	var m map[string]interface{}
	_ = json.Unmarshal(raw, &m)
	httpx.Success(c, gin.H{"article": formatKnowledgeArticle(*art), "manifest": m})
}

// Create 创建知识条目
// @Summary 创建新知识条目
// @Description 在系统中初始化一个新的知识库文章。管理员需指定关联的告警规则名称及可选的标签。该操作会在数据库中占位并生成对应的对象存储Key，为后续上传具体的HTML排查内容或配置Manifest数据做好准备。
// @Tags Admin,Knowledge
// @Accept json
// @Produce json
// @Param request body knowledgeCreateRequest true "文章创建基本信息"
// @Success 201 {object} httpx.Response{data=object}
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /admin/knowledge [post]
func (h *Handler) Create(c *gin.Context) {
	var req knowledgeCreateRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.AbortWithDomainError(c, err)
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
		httpx.InternalError(c, "CREATE_FAILED", err.Error())
		return
	}
	httpx.Created(c, formatKnowledgeArticle(*art))
}

// Update 更新（写manifest）
// @Summary 更新文章配置
// @Description 修改已有知识文章的Manifest详细定义。管理员可以通过此接口更新告警排查的具体步骤、运行剧本以及参数说明。新数据会重新同步到对象存储中，并自动增加版本号记录，确保知识库内容的实时性与准确性。
// @Tags Admin,Knowledge
// @Accept json
// @Produce json
// @Param id path string true "文章ID"
// @Param request body services.Manifest true "Manifest配置详情"
// @Success 200 {object} httpx.Response{data=object}
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /admin/knowledge/{id} [put]
func (h *Handler) Update(c *gin.Context) {
	var uri knowledgeURI
	if err := c.ShouldBindUri(&uri); err != nil {
		httpx.BadRequest(c, "INVALID_ID", "无效的ID")
		return
	}
	var payload services.Manifest
	if err := httpx.BindJSON(c, &payload); err != nil {
		httpx.AbortWithDomainError(c, err)
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
		httpx.InternalError(c, "UPDATE_FAILED", err.Error())
		return
	}
	httpx.Success(c, formatKnowledgeArticle(*art))
}

// Publish 发布知识条目
// @Summary 发布知识文章
// @Description 将处于草稿或编辑状态的知识文章正式发布为在线状态。发布过程中可以记录简要的变更说明，用于历史审计和版本追踪。只有发布后的文章才能在前端告警关联中被普通用户查看到。
// @Tags Admin,Knowledge
// @Accept json
// @Produce json
// @Param id path string true "文章ID"
// @Param request body knowledgePublishRequest true "发布变更说明"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /admin/knowledge/{id}/publish [post]
func (h *Handler) Publish(c *gin.Context) {
	var uri knowledgeURI
	if err := c.ShouldBindUri(&uri); err != nil {
		httpx.BadRequest(c, "INVALID_ID", "无效的ID")
		return
	}
	var req knowledgePublishRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.AbortWithDomainError(c, err)
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
		httpx.InternalError(c, "PUBLISH_FAILED", err.Error())
		return
	}
	httpx.SuccessWithMessage(c, "发布成功", nil)
}

// PresignUpload 预签名上传
// @Summary 获取文件上传签名
// @Description 为了支持前端直接向对象存储安全地上传静态资源（如HTML文档或图片），该接口生成一个带有时效性的预签名URL。这避免了将大型二进制数据流通过后端API服务器中转，提高了系统的并发处理能力和传输效率。
// @Tags Admin,Knowledge
// @Accept json
// @Produce json
// @Param request body knowledgePresignRequest true "上传请求详情"
// @Success 200 {object} httpx.Response{data=object}
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /admin/knowledge/presign-upload [post]
func (h *Handler) PresignUpload(c *gin.Context) {
	var req knowledgePresignRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.AbortWithDomainError(c, err)
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Minute)
	defer cancel()
	key, url, err := h.svc.PresignAssetPut(ctx, req.ArticleID, req.Filename, req.ContentType)
	if err != nil {
		httpx.InternalError(c, "PRESIGN_FAILED", err.Error())
		return
	}
	httpx.Success(c, gin.H{"key": key, "url": url})
}

// Delete 删除知识条目
// @Summary 删除知识文章
// @Description 物理删除指定的知识库文章记录及其在对象存储中关联的Manifest内容。这是一项破坏性操作，一旦执行，该告警规则下的所有运维知识指引将立即消失。请确保该条目不再具有参考价值后再进行删除。
// @Tags Admin,Knowledge
// @Param id path string true "文章ID"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /admin/knowledge/{id} [delete]
func (h *Handler) Delete(c *gin.Context) {
	var uri knowledgeURI
	if err := c.ShouldBindUri(&uri); err != nil {
		httpx.BadRequest(c, "INVALID_ID", "无效的ID")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	if err := h.svc.Delete(ctx, uri.ID); err != nil {
		httpx.InternalError(c, "DELETE_FAILED", err.Error())
		return
	}

	httpx.SuccessWithMessage(c, "删除成功", nil)
}

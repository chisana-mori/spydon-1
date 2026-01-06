package http

import (
	"net/http"
	"strconv"

	"robusta-web/backend/internal/models"
	httpx "robusta-web/backend/internal/transport/httpx"

	"github.com/gin-gonic/gin"
)

// CreateTemplateRequest 创建模板请求
type CreateTemplateRequest struct {
	Name        string                   `json:"name" binding:"required"`
	Description string                   `json:"description"`
	Stages      []models.StageDefinition `json:"stages" binding:"required,min=1"`
}

// UpdateTemplateRequest 更新模板请求
type UpdateTemplateRequest struct {
	Name        string                   `json:"name" binding:"required"`
	Description string                   `json:"description"`
	Stages      []models.StageDefinition `json:"stages" binding:"required,min=1"`
}

// ListTemplates 获取流水线模板列表
func (h *Handler) ListTemplates(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	keyword := c.Query("keyword")

	templates, total, err := h.engine.ListTemplates(page, pageSize, keyword)
	if err != nil {
		httpx.InternalError(c, "", "获取模板列表失败: "+err.Error())
		return
	}

	totalPages := 0
	if pageSize > 0 {
		totalPages = int((total + int64(pageSize) - 1) / int64(pageSize))
	}

	c.JSON(http.StatusOK, gin.H{
		"data": templates,
		"pagination": gin.H{
			"total":       total,
			"page":        page,
			"page_size":   pageSize,
			"total_pages": totalPages,
		},
	})
}

// GetTemplate 获取单个模板详情
func (h *Handler) GetTemplate(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httpx.BadRequest(c, "", "无效的模板ID")
		return
	}

	template, err := h.engine.GetTemplate(id)
	if err != nil {
		httpx.NotFound(c, "", "模板不存在")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": template})
}

// CreateTemplate 创建流水线模板
func (h *Handler) CreateTemplate(c *gin.Context) {
	var req CreateTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "", "请求参数错误: "+err.Error())
		return
	}

	userID := getUserIDFromContext(c)
	template, err := h.engine.CreateTemplate(req.Name, req.Description, req.Stages, userID)
	if err != nil {
		httpx.InternalError(c, "", "创建模板失败: "+err.Error())
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": template})
}

// UpdateTemplate 更新流水线模板
func (h *Handler) UpdateTemplate(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httpx.BadRequest(c, "", "无效的模板ID")
		return
	}

	var req UpdateTemplateRequest
	if err = c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "", "请求参数错误: "+err.Error())
		return
	}

	template, err := h.engine.UpdateTemplate(id, req.Name, req.Description, req.Stages)
	if err != nil {
		httpx.InternalError(c, "", "更新模板失败: "+err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": template})
}

// DeleteTemplate 删除流水线模板
func (h *Handler) DeleteTemplate(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httpx.BadRequest(c, "", "无效的模板ID")
		return
	}

	if err := h.engine.DeleteTemplate(id); err != nil {
		httpx.InternalError(c, "", "删除模板失败: "+err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "模板已删除"})
}

package http

import (
	"net/http"
	"strconv"

	httpx "robusta-web/backend/internal/transport/httpx"

	"github.com/gin-gonic/gin"
)

// UpdateInventoryVariablesRequest 更新Inventory变量的请求
type UpdateInventoryVariablesRequest struct {
	Variables string `json:"variables" binding:"required"` // YAML格式的变量
}

// ListAWXTemplates 获取任务模板列表
func (h *Handler) ListAWXTemplates(c *gin.Context) {
	templates, err := h.engine.ListJobTemplates(c.Request.Context())
	if err != nil {
		httpx.InternalError(c, "", "获取任务模板列表失败: "+err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": templates})
}

// GetAWXTemplate 获取单个任务模板详情
func (h *Handler) GetAWXTemplate(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		httpx.BadRequest(c, "", "无效的模板ID")
		return
	}

	template, err := h.engine.GetJobTemplate(c.Request.Context(), id)
	if err != nil {
		httpx.InternalError(c, "", "获取任务模板详情失败: "+err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": template})
}

// GetInventoryVariables 获取Inventory变量
func (h *Handler) GetInventoryVariables(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		httpx.BadRequest(c, "", "集群名称不能为空")
		return
	}

	variables, err := h.engine.GetInventoryVariables(c.Request.Context(), name)
	if err != nil {
		httpx.InternalError(c, "", "获取Inventory变量失败: "+err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"variables": variables}})
}

// UpdateInventoryVariables 更新Inventory变量
func (h *Handler) UpdateInventoryVariables(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		httpx.BadRequest(c, "", "集群名称不能为空")
		return
	}

	var req UpdateInventoryVariablesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "", "请求参数错误: "+err.Error())
		return
	}

	if err := h.engine.UpdateInventoryVariables(c.Request.Context(), name, req.Variables); err != nil {
		httpx.InternalError(c, "", "更新Inventory变量失败: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Inventory变量已更新"})
}

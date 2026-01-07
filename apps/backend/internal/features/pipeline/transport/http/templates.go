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
// @Summary 分页获取流水线模板
// @Description 从系统中分页检索已定义的流水线作业模板列表。支持通过关键字对模板名称进行模糊搜索。返回结果包含模板的基本元数据、创建时间及版本信息，方便运维人员快速定位并选择合适的自动化执行流程模板。
// @Tags Pipeline,Templates
// @Produce json
// @Param page query int false "页码 (默认1)"
// @Param page_size query int false "每页条数 (默认20)"
// @Param keyword query string false "搜素关键字"
// @Success 200 {object} object
// @Failure 500 {object} httpx.ErrorResponse
// @Router /pipelines/templates [get]
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
// @Summary 获取流水线模板详情
// @Description 根据指定的ID加载特定流水线模板的完整定义，包括各个阶段（Stages）的具体步骤、参数要求及环境变量配置。该接口为前端渲染流程图或进行作业启动前的参数校验提供完整的数据支撑结构。
// @Tags Pipeline,Templates
// @Produce json
// @Param id path int true "模板ID"
// @Success 200 {object} object
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Router /pipelines/templates/{id} [get]
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
// @Summary 新增流水线模板
// @Description 用于创建一个包含多个执行阶段的自动化流水线模板。用户需定义模板名称、描述以及具体的阶段逻辑（如调用AWX、K8s命令等）。创建成功的模板可被授权给各业务团队，作为标准化的作业执行蓝本。
// @Tags Admin,Pipeline,Templates
// @Accept json
// @Produce json
// @Param request body CreateTemplateRequest true "模板创建信息"
// @Success 201 {object} object
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /admin/pipelines/templates [post]
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
// @Summary 编辑流水线模板配置
// @Description 修改已有流水线模板的属性或内部阶段定义。系统支持对模板名称、作业步骤及环境配置进行全量更新。更新后的模板将立即生效于后续新发起的作业执行任务，确保运维自动化的灵活性与实时性。
// @Tags Admin,Pipeline,Templates
// @Accept json
// @Produce json
// @Param id path int true "模板ID"
// @Param request body UpdateTemplateRequest true "模板更新内容"
// @Success 200 {object} object
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /admin/pipelines/templates/{id} [put]
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
// @Summary 物理删除流水线模板
// @Description 根据ID从系统中移除指定的自动化流程模板。执行此操作前，请确认该模板未被当前的定时任务或关键生产流程所引用。模板一旦删除，关联的历史作业快照虽然保留，但无法再基于该模板发起新任务。
// @Tags Admin,Pipeline,Templates
// @Param id path int true "模板ID"
// @Success 200 {object} object
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /admin/pipelines/templates/{id} [delete]
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

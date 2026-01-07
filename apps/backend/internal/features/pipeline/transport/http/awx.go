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
// @Summary 获取AWX作业模板列表
// @Description 从集成的AWX/Ansible Tower服务中同步检索所有可用的作业模板条目。返回数据包含模板ID、名称和所属项目。这些模板是构建系统内流水线精确定向任务的基础，允许运维人员直接引用已有的Ansible剧本。
// @Tags Pipeline,AWX
// @Produce json
// @Success 200 {object} object
// @Failure 500 {object} httpx.ErrorResponse
// @Router /pipelines/awx/templates [get]
func (h *Handler) ListAWXTemplates(c *gin.Context) {
	templates, err := h.engine.ListJobTemplates(c.Request.Context())
	if err != nil {
		httpx.InternalError(c, "", "获取任务模板列表失败: "+err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": templates})
}

// GetAWXTemplate 获取单个任务模板详情
// @Summary 获取AWX模板详细配置
// @Description 根据模板ID获取AWX端定义的详细作业参数，包括所需的调查问卷（Survey）变量、执行环境及关联的Inventory。该接口对于在前端动态生成作业启动表单、验证用户输入参数的合法性具有至关重要的作用。
// @Tags Pipeline,AWX
// @Produce json
// @Param id path int true "AWX模板ID"
// @Success 200 {object} object
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /pipelines/awx/templates/{id} [get]
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
// @Summary 获取集群Inventory变量
// @Description 检索指定集群在AWX中对应的Inventory主体变量定义。这些变量通常以YAML格式存储，定义了该集群专属的全局运维配置，如连接凭据、环境路径或自定义的业务逻辑开关参数。
// @Tags Pipeline,AWX
// @Produce json
// @Param name path string true "集群名称"
// @Success 200 {object} object
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /pipelines/awx/inventories/{name}/variables [get]
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
// @Summary 更新集群运维变量配置
// @Description 修改AWX中指定集群Inventory的全局变量内容。该接口允许管理员通过提交YAML字符串来动态调整集群的执行环境参数。更新后的变量将影响所有后续针对该集群发起的Ansible作业执行逻辑。
// @Tags Admin,Pipeline,AWX
// @Accept json
// @Produce json
// @Param name path string true "集群名称"
// @Param request body UpdateInventoryVariablesRequest true "变量更新详情"
// @Success 200 {object} object
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /admin/pipelines/awx/inventories/{name}/variables [put]
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

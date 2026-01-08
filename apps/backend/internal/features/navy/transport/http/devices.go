package http

import (
	"net/http"
	"strconv"

	"robusta-web/backend/internal/features/navy/services"
	"robusta-web/backend/internal/transport/httpx"

	"github.com/gin-gonic/gin"
)

// =============================================================================
// DeviceHandler - 设备查询处理器
// =============================================================================

// DeviceHandler 处理设备查询相关的 HTTP 请求
type DeviceHandler struct {
	svc *services.NavyDeviceService
}

// NewDeviceHandler 创建 DeviceHandler
func NewDeviceHandler(svc *services.NavyDeviceService) *DeviceHandler {
	return &DeviceHandler{svc: svc}
}

// RegisterRoutes 注册设备相关路由
func (h *DeviceHandler) RegisterRoutes(navyGroup *gin.RouterGroup) {
	devices := navyGroup.Group(RouteGroupDevices)
	{
		devices.GET("", h.ListDevices)
		devices.POST(RouteQuery, h.QueryDevices)
		devices.GET(RouteFilterOptions, h.GetFilterOptions)
		devices.GET(RouteLabelValues, h.GetLabelValues)
		devices.GET(RouteTaintValues, h.GetTaintValues)
		devices.GET(RouteDeviceFieldValues, h.GetDeviceFieldValues)
		devices.POST(RouteFeatures, h.GetDeviceFeatures)
		devices.GET(RouteFeatureDetails, h.GetFeatureDetails)
		devices.GET(RouteExport, h.ExportDevices)
		devices.GET(RouteParamID, h.GetDevice)
		devices.PATCH(RouteParamIDRole, h.UpdateDeviceRole)
		devices.PATCH(RouteParamIDGroup, h.UpdateDeviceGroup)
	}

	// 模板管理
	templates := navyGroup.Group(RouteGroupTemplates)
	{
		templates.GET("", h.GetTemplates)
		templates.POST("", h.SaveTemplate)
		templates.GET(RouteParamID, h.GetTemplate)
		templates.DELETE(RouteParamID, h.DeleteTemplate)
	}
}

// ListDevices 获取设备列表
// @Summary 分页获取设备列表
// @Description 根据查询参数分页获取Navy系统中的设备列表。支持通过集群、IDC、子网、角色等多种维度进行简单过滤。返回结果包含设备的基本元数据、状态及所属环境信息，是资产管理模块的核心数据查询接口。
// @Tags Navy,Devices
// @Produce json
// @Param cluster query string false "集群名称"
// @Param idc query string false "机房标识"
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Success 200 {object} httpx.Response{data=[]services.NavyDevice}
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /navy/devices [get]
func (h *DeviceHandler) ListDevices(c *gin.Context) {
	var query services.DeviceQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		httpx.BadRequest(c, "INVALID_QUERY_PARAMS", err.Error())
		return
	}

	res, err := h.svc.ListDevices(c.Request.Context(), &query)
	if err != nil {
		httpx.InternalError(c, "LIST_DEVICES_ERROR", err.Error())
		return
	}

	pagination := httpx.NewPagination(res.Page, res.Size, res.Total)
	httpx.SuccessPaginated(c, res.List, pagination)
}

// QueryDevices 复杂查询
// @Summary 设备多维复杂查询
// @Description 支持通过复杂的条件组合（如嵌套逻辑、高级标签关联等）来精准检索设备。该接口采用POST请求发送JSON格式的查询DSL，能够满足在大规模资产库中进行深度数据挖掘和资源审计的高级性能需求。
// @Tags Navy,Devices
// @Accept json
// @Produce json
// @Param request body services.NavyDeviceQueryRequest true "复杂查询请求结构"
// @Success 200 {object} httpx.Response{data=[]services.NavyDevice}
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /navy/devices/query [post]
func (h *DeviceHandler) QueryDevices(c *gin.Context) {
	var req services.NavyDeviceQueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "INVALID_QUERY_REQUEST", err.Error())
		return
	}

	res, err := h.svc.QueryDevices(c.Request.Context(), &req)
	if err != nil {
		httpx.InternalError(c, "QUERY_DEVICES_ERROR", err.Error())
		return
	}

	pagination := httpx.NewPagination(res.Page, res.Size, res.Total)
	httpx.SuccessPaginated(c, res.List, pagination)
}

// GetFilterOptions 获取筛选选项
// @Summary 获取设备筛选字典项
// @Description 提取当前系统中所有设备的可筛选维度，包括现有的IDC列表、集群集合、硬件角色分类等。前端使用此接口填充搜索框的下拉选项，确保用户在进行设备检索时能够使用有效的预定义元数据关键字。
// @Tags Navy,Devices
// @Produce json
// @Success 200 {object} httpx.Response{data=services.FilterOptions}
// @Failure 500 {object} httpx.ErrorResponse
// @Router /navy/devices/filter-options [get]
func (h *DeviceHandler) GetFilterOptions(c *gin.Context) {
	res, err := h.svc.GetFilterOptions(c.Request.Context())
	if err != nil {
		httpx.InternalError(c, "GET_FILTER_OPTIONS_ERROR", err.Error())
		return
	}
	httpx.Success(c, res)
}

// GetLabelValues 获取指定标签的所有值
// @Summary 获取标签的所有取值
// @Description 根据指定的标签Key，查询并返回系统中所有设备在该标签下出现的唯一值集合。此接口对于了解标签分布情况、发现异常打标数据以及辅助构建动态生成的查询表单具有重要的数据支撑作用。
// @Tags Navy,Devices
// @Produce json
// @Param key query string true "标签键名"
// @Success 200 {object} httpx.Response{data=[]string}
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /navy/devices/labels/values [get]
func (h *DeviceHandler) GetLabelValues(c *gin.Context) {
	labelKey := c.Query("key")
	if labelKey == "" {
		httpx.BadRequest(c, "LABEL_KEY_REQUIRED", "key is required")
		return
	}

	values, err := h.svc.GetLabelValues(c.Request.Context(), labelKey)
	if err != nil {
		httpx.InternalError(c, "GET_LABEL_VALUES_ERROR", err.Error())
		return
	}
	httpx.Success(c, values)
}

// GetTaintValues 获取指定污点的所有值
// @Summary 获取污点的所有取值
// @Description 检索系统中针对特定污点（Taint）Key所应用过的所有唯一Value值。该数据主要用于在节点维护和调度管理中，帮助管理员识别已有的污点应用模式，从而在新的维护任务中选择兼容的配置参数。
// @Tags Navy,Devices
// @Produce json
// @Param key query string true "污点键名"
// @Success 200 {object} httpx.Response{data=[]string}
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /navy/devices/taints/values [get]
func (h *DeviceHandler) GetTaintValues(c *gin.Context) {
	taintKey := c.Query("key")
	if taintKey == "" {
		httpx.BadRequest(c, "TAINT_KEY_REQUIRED", "key is required")
		return
	}

	values, err := h.svc.GetTaintValues(c.Request.Context(), taintKey)
	if err != nil {
		httpx.InternalError(c, "GET_TAINT_VALUES_ERROR", err.Error())
		return
	}
	httpx.Success(c, values)
}

// GetDeviceFieldValues 获取设备字段的所有值
// @Summary 获取设备字段的唯一值
// @Description 查询设备模型中特定基础字段（如厂商、型号、OS版本等）的全局唯一取值列表。该接口有助于维护资产数据的规范性，并为多维度的资产统计报表提供准确的数据字典参考信息。
// @Tags Navy,Devices
// @Produce json
// @Param field query string true "设备字段名称"
// @Success 200 {object} httpx.Response{data=[]string}
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /navy/devices/fields/values [get]
func (h *DeviceHandler) GetDeviceFieldValues(c *gin.Context) {
	field := c.Query("field")
	if field == "" {
		httpx.BadRequest(c, "FIELD_REQUIRED", "field is required")
		return
	}

	values, err := h.svc.GetDeviceFieldValues(c.Request.Context(), field)
	if err != nil {
		httpx.InternalError(c, "GET_DEVICE_FIELD_VALUES_ERROR", err.Error())
		return
	}
	httpx.Success(c, values)
}

// GetDevice 获取单个设备详情
// @Summary 获取设备详细信息
// @Description 通过数据库ID精确获取单台设备的完整配置。返回结果涵盖网络配置、硬件清单、关联的所有标签与污点信息。这是设备运维详情页的后台基础数据接口，提供了对单一物理实例的全方位视角展示。
// @Tags Navy,Devices
// @Produce json
// @Param id path int true "设备数据库ID"
// @Success 200 {object} httpx.Response{data=services.NavyDevice}
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Router /navy/devices/{id} [get]
func (h *DeviceHandler) GetDevice(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		httpx.BadRequest(c, "INVALID_ID", "无效的 ID 格式")
		return
	}

	res, err := h.svc.GetDevice(c.Request.Context(), id)
	if err != nil {
		httpx.NotFound(c, "DEVICE_NOT_FOUND", "设备不存在")
		return
	}

	httpx.Success(c, res)
}

// GetFeatureDetails 获取设备特性详情
// @Summary 查询设备单项特性
// @Description 根据CI编码精准查询该设备的各种运维特性，如K8s版本、内核参数、已挂载磁盘详情等。该接口设计用于在不加载完整设备模型的情况下，快速调取特定的性能指标或配置特征，提升前端渲染效率。
// @Tags Navy,Devices
// @Produce json
// @Param ci_code query string true "设备CI编码"
// @Success 200 {object} httpx.Response{data=object}
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /navy/devices/features/details [get]
func (h *DeviceHandler) GetFeatureDetails(c *gin.Context) {
	ciCode := c.Query("ci_code")
	if ciCode == "" {
		httpx.BadRequest(c, "CI_CODE_REQUIRED", "ci_code is required")
		return
	}

	res, err := h.svc.GetBatchDeviceFeatures(c.Request.Context(), []string{ciCode})
	if err != nil {
		httpx.InternalError(c, "GET_FEATURE_DETAILS_ERROR", err.Error())
		return
	}

	httpx.Success(c, res)
}

// GetDeviceFeatures 批量获取设备特性
// @Summary 批量查询设备特性集
// @Description 接受一组CI编码列表，返回这些设备共同的特性配置快照。该批量处理接口优化了在大规模导出或多设备对比场景下的性能表现，确保用户能够一键获取多台物理节点的底层配置特征汇总数据。
// @Tags Navy,Devices
// @Accept json
// @Produce json
// @Param request body services.DeviceFeaturesRequest true "CI编码列表结构"
// @Success 200 {object} httpx.Response{data=object}
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /navy/devices/features [post]
func (h *DeviceHandler) GetDeviceFeatures(c *gin.Context) {
	var req services.DeviceFeaturesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "INVALID_REQUEST", err.Error())
		return
	}

	res, err := h.svc.GetBatchDeviceFeatures(c.Request.Context(), req.CICodes)
	if err != nil {
		httpx.InternalError(c, "GET_DEVICE_FEATURES_ERROR", err.Error())
		return
	}

	httpx.Success(c, res)
}

// UpdateDeviceRole 更新角色
// @Summary 修改设备运维角色
// @Description 手动调整特定设备在生产环境中的运维角色（如计算节点、存储节点等）。更新角色通常会触发关联的监控策略变更或自动化部署流程的逻辑分配。该操作需要管理员权限，并会在系统中产生审计记录。
// @Tags Navy,Devices
// @Accept json
// @Produce json
// @Param id path int true "设备ID"
// @Param request body services.DeviceRoleUpdateRequest true "新角色配置参数"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /navy/devices/{id}/role [put]
func (h *DeviceHandler) UpdateDeviceRole(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		httpx.BadRequest(c, "INVALID_ID", "无效的 ID 格式")
		return
	}

	var req services.DeviceRoleUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "INVALID_REQUEST", err.Error())
		return
	}

	if err := h.svc.UpdateDeviceRole(c.Request.Context(), id, req.Role); err != nil {
		httpx.InternalError(c, "UPDATE_ROLE_ERROR", err.Error())
		return
	}

	httpx.SuccessWithMessage(c, "角色更新成功", nil)
}

// UpdateDeviceGroup 更新组
// @Summary 修改设备资源分组
// @Description 修改设备所属的逻辑资源池或管理组。分组信息的更新直接影响权限控制、批量任务的筛选范围以及成本中心的统计。在大规模数据中心运营中，此接口是确保存量资产能够根据业务调整进行快速分类流转的关键工具。
// @Tags Navy,Devices
// @Accept json
// @Produce json
// @Param id path int true "设备ID"
// @Param request body services.DeviceGroupUpdateRequest true "新分组配置参数"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /navy/devices/{id}/group [put]
func (h *DeviceHandler) UpdateDeviceGroup(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		httpx.BadRequest(c, "INVALID_ID", "无效的 ID 格式")
		return
	}

	var req services.DeviceGroupUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "INVALID_REQUEST", err.Error())
		return
	}

	if err := h.svc.UpdateDeviceGroup(c.Request.Context(), id, req.Group); err != nil {
		httpx.InternalError(c, "UPDATE_GROUP_ERROR", err.Error())
		return
	}

	httpx.SuccessWithMessage(c, "分组更新成功", nil)
}

// ExportDevices 导出设备
// @Summary 导出全量设备清单
// @Description 生成并下载Navy系统中所有设备的详细CSV报表。该功能常用于资产盘点、离线审计以及向下游CMDB系统提供全量的资产基准数据。导出的数据文件涵盖了设备模型中所有的公开元数据字段。
// @Tags Navy,Devices
// @Produce text/csv
// @Success 200 {string} string "CSV文件流"
// @Failure 500 {object} httpx.ErrorResponse
// @Router /navy/devices/export [get]
func (h *DeviceHandler) ExportDevices(c *gin.Context) {
	data, err := h.svc.ExportDevices(c.Request.Context())
	if err != nil {
		httpx.InternalError(c, "EXPORT_DEVICES_ERROR", err.Error())
		return
	}

	c.Header("Content-Disposition", "attachment; filename=navy_devices.csv")
	c.Data(http.StatusOK, "text/csv", data)
}

// SaveTemplate 保存模板
// @Summary 保存设备查询模板
// @Description 将当前复杂的查询条件（包括DSL、排序规则、显示字段等）保存为可重复使用的模板。用户可以为模板命名，以便后续在资产管理仪表盘中快速切换不同的视图，提升日常运维查询的工作效率。
// @Tags Navy,Templates
// @Accept json
// @Produce json
// @Param request body services.QueryTemplate true "保存模板参数"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /navy/devices/templates [post]
func (h *DeviceHandler) SaveTemplate(c *gin.Context) {
	var req services.QueryTemplate
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "INVALID_REQUEST", err.Error())
		return
	}

	if err := h.svc.SaveQueryTemplate(c.Request.Context(), &req); err != nil {
		httpx.InternalError(c, "SAVE_TEMPLATE_ERROR", err.Error())
		return
	}

	httpx.SuccessWithMessage(c, "模板保存成功", nil)
}

// GetTemplates 获取模板列表
// @Summary 获取个人查询模板列表
// @Description 分页查询当前用户保存的所有资产检索模板。返回的列表包含模板名称、更新时间以及简化的预览参数。该接口支撑了前端资产页面的“视图切换”功能，使得个性化的查询配置能够在多终端同步使用。
// @Tags Navy,Templates
// @Produce json
// @Param page query int false "页码"
// @Param size query int false "每页条数"
// @Success 200 {object} httpx.Response{data=[]services.QueryTemplate}
// @Failure 500 {object} httpx.ErrorResponse
// @Router /navy/devices/templates [get]
func (h *DeviceHandler) GetTemplates(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))

	res, err := h.svc.GetQueryTemplates(c.Request.Context(), page, size)
	if err != nil {
		httpx.InternalError(c, "GET_TEMPLATES_ERROR", err.Error())
		return
	}

	pagination := httpx.NewPagination(res.Page, res.Size, res.Total)
	httpx.SuccessPaginated(c, res.List, pagination)
}

// GetTemplate 获取单个模板
// @Summary 获取查询模板详情
// @Description 根据指定的ID加载完整的查询模板配置。该接口返回模板定义的原始查询条件、布局偏好以及元数据信息，允许前端解析并自动填充复杂查询界面的各个选项，实现“一键还原”历史查询场景的功能。
// @Tags Navy,Templates
// @Produce json
// @Param id path int true "模板ID"
// @Success 200 {object} httpx.Response{data=services.QueryTemplate}
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Router /navy/devices/templates/{id} [get]
func (h *DeviceHandler) GetTemplate(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		httpx.BadRequest(c, "INVALID_ID", "无效的 ID 格式")
		return
	}

	res, err := h.svc.GetQueryTemplate(c.Request.Context(), id)
	if err != nil {
		httpx.NotFound(c, "TEMPLATE_NOT_FOUND", "模板不存在")
		return
	}

	httpx.Success(c, res)
}

// DeleteTemplate 删除模板
// @Summary 物理删除查询模板
// @Description 从系统中永久移除用户之前保存的查询模板记录。该操作不可撤销。移除后，该模板将不再出现在用户的常用视图列表中。这有助于用户通过清理过期的或冗余的配置，保持资产查询界面的整洁与高效。
// @Tags Navy,Templates
// @Param id path int true "模板ID"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /navy/devices/templates/{id} [delete]
func (h *DeviceHandler) DeleteTemplate(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		httpx.BadRequest(c, "INVALID_ID", "无效的 ID 格式")
		return
	}

	if err := h.svc.DeleteQueryTemplate(c.Request.Context(), id); err != nil {
		httpx.InternalError(c, "DELETE_TEMPLATE_ERROR", err.Error())
		return
	}

	httpx.SuccessWithMessage(c, "模板删除成功", nil)
}

package api

import (
	"net/http"
	"robusta-web/backend/internal/services"
	"strconv"

	"github.com/gin-gonic/gin"
)

// NavyDeviceHandler Navy 设备管理处理器
type NavyDeviceHandler struct {
	service *services.NavyDeviceService
}

// NewNavyDeviceHandler 创建 Navy 设备管理处理器
func NewNavyDeviceHandler(service *services.NavyDeviceService) *NavyDeviceHandler {
	return &NavyDeviceHandler{service: service}
}

// List 获取设备列表
func (h *NavyDeviceHandler) List(c *gin.Context) {
	var query services.DeviceQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		BadRequest(c, "INVALID_QUERY_PARAMS", err.Error())
		return
	}

	res, err := h.service.ListDevices(c.Request.Context(), &query)
	if err != nil {
		InternalError(c, "LIST_DEVICES_ERROR", err.Error())
		return
	}

	pagination := NewPagination(res.Page, res.Size, res.Total)
	SuccessPaginated(c, res.List, pagination)
}

// Query 复杂查询
func (h *NavyDeviceHandler) Query(c *gin.Context) {
	var req services.NavyDeviceQueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "INVALID_QUERY_REQUEST", err.Error())
		return
	}

	res, err := h.service.QueryDevices(c.Request.Context(), &req)
	if err != nil {
		InternalError(c, "QUERY_DEVICES_ERROR", err.Error())
		return
	}

	pagination := NewPagination(res.Page, res.Size, res.Total)
	SuccessPaginated(c, res.List, pagination)
}

// GetFilterOptions 获取筛选选项
func (h *NavyDeviceHandler) GetFilterOptions(c *gin.Context) {
	res, err := h.service.GetFilterOptions(c.Request.Context())
	if err != nil {
		InternalError(c, "GET_FILTER_OPTIONS_ERROR", err.Error())
		return
	}
	Success(c, res)
}

// GetLabelValues 获取指定标签的所有值
func (h *NavyDeviceHandler) GetLabelValues(c *gin.Context) {
	labelKey := c.Query("key")
	if labelKey == "" {
		BadRequest(c, "LABEL_KEY_REQUIRED", "key is required")
		return
	}

	values, err := h.service.GetLabelValues(c.Request.Context(), labelKey)
	if err != nil {
		InternalError(c, "GET_LABEL_VALUES_ERROR", err.Error())
		return
	}
	Success(c, values)
}

// GetTaintValues 获取指定污点的所有值
func (h *NavyDeviceHandler) GetTaintValues(c *gin.Context) {
	taintKey := c.Query("key")
	if taintKey == "" {
		BadRequest(c, "TAINT_KEY_REQUIRED", "key is required")
		return
	}

	values, err := h.service.GetTaintValues(c.Request.Context(), taintKey)
	if err != nil {
		InternalError(c, "GET_TAINT_VALUES_ERROR", err.Error())
		return
	}
	Success(c, values)
}

// GetDeviceFieldValues 获取设备字段的所有值
func (h *NavyDeviceHandler) GetDeviceFieldValues(c *gin.Context) {
	field := c.Query("field")
	if field == "" {
		BadRequest(c, "FIELD_REQUIRED", "field is required")
		return
	}

	values, err := h.service.GetDeviceFieldValues(c.Request.Context(), field)
	if err != nil {
		InternalError(c, "GET_DEVICE_FIELD_VALUES_ERROR", err.Error())
		return
	}
	Success(c, values)
}

// Get 获取单个设备详情
func (h *NavyDeviceHandler) Get(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		BadRequest(c, "INVALID_ID", "无效的 ID 格式")
		return
	}

	res, err := h.service.GetDevice(c.Request.Context(), id)
	if err != nil {
		NotFound(c, "DEVICE_NOT_FOUND", "设备不存在")
		return
	}

	Success(c, res)
}

// GetFeatureDetails 获取设备特性详情
func (h *NavyDeviceHandler) GetFeatureDetails(c *gin.Context) {
	ciCode := c.Query("ci_code")
	if ciCode == "" {
		BadRequest(c, "CI_CODE_REQUIRED", "ci_code is required")
		return
	}

	res, err := h.service.GetDeviceFeatureDetails(c.Request.Context(), ciCode)
	if err != nil {
		InternalError(c, "GET_FEATURE_DETAILS_ERROR", err.Error())
		return
	}

	Success(c, res)
}

// UpdateRole 更新角色
func (h *NavyDeviceHandler) UpdateRole(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		BadRequest(c, "INVALID_ID", "无效的 ID 格式")
		return
	}

	var req services.DeviceRoleUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "INVALID_REQUEST", err.Error())
		return
	}

	if err := h.service.UpdateDeviceRole(c.Request.Context(), id, req.Role); err != nil {
		InternalError(c, "UPDATE_ROLE_ERROR", err.Error())
		return
	}

	SuccessWithMessage(c, "角色更新成功", nil)
}

// UpdateGroup 更新组
func (h *NavyDeviceHandler) UpdateGroup(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		BadRequest(c, "INVALID_ID", "无效的 ID 格式")
		return
	}

	var req services.DeviceGroupUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "INVALID_REQUEST", err.Error())
		return
	}

	if err := h.service.UpdateDeviceGroup(c.Request.Context(), id, req.Group); err != nil {
		InternalError(c, "UPDATE_GROUP_ERROR", err.Error())
		return
	}

	SuccessWithMessage(c, "分组更新成功", nil)
}

// Export 导出设备
func (h *NavyDeviceHandler) Export(c *gin.Context) {
	data, err := h.service.ExportDevices(c.Request.Context())
	if err != nil {
		InternalError(c, "EXPORT_DEVICES_ERROR", err.Error())
		return
	}

	c.Header("Content-Disposition", "attachment; filename=navy_devices.csv")
	c.Data(http.StatusOK, "text/csv", data)
}

// SaveTemplate 保存模板
func (h *NavyDeviceHandler) SaveTemplate(c *gin.Context) {
	var req services.QueryTemplate
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "INVALID_REQUEST", err.Error())
		return
	}

	if err := h.service.SaveQueryTemplate(c.Request.Context(), &req); err != nil {
		InternalError(c, "SAVE_TEMPLATE_ERROR", err.Error())
		return
	}

	SuccessWithMessage(c, "模板保存成功", nil)
}

// GetTemplates 获取模板列表
func (h *NavyDeviceHandler) GetTemplates(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))

	res, err := h.service.GetQueryTemplates(c.Request.Context(), page, size)
	if err != nil {
		InternalError(c, "GET_TEMPLATES_ERROR", err.Error())
		return
	}

	pagination := NewPagination(res.Page, res.Size, res.Total)
	SuccessPaginated(c, res.List, pagination)
}

// GetTemplate 获取单个模板
func (h *NavyDeviceHandler) GetTemplate(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		BadRequest(c, "INVALID_ID", "无效的 ID 格式")
		return
	}

	res, err := h.service.GetQueryTemplate(c.Request.Context(), id)
	if err != nil {
		NotFound(c, "TEMPLATE_NOT_FOUND", "模板不存在")
		return
	}

	Success(c, res)
}

// DeleteTemplate 删除模板
func (h *NavyDeviceHandler) DeleteTemplate(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		BadRequest(c, "INVALID_ID", "无效的 ID 格式")
		return
	}

	if err := h.service.DeleteQueryTemplate(c.Request.Context(), id); err != nil {
		InternalError(c, "DELETE_TEMPLATE_ERROR", err.Error())
		return
	}

	SuccessWithMessage(c, "模板删除成功", nil)
}

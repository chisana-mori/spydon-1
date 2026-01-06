package http

import (
	"net/http"
	"strconv"

	"robusta-web/backend/internal/features/navy/services"
	"robusta-web/backend/internal/transport/httpx"

	"github.com/gin-gonic/gin"
)

// ListDevices 获取设备列表
func (h *Handler) ListDevices(c *gin.Context) {
	var query services.DeviceQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		httpx.BadRequest(c, "INVALID_QUERY_PARAMS", err.Error())
		return
	}

	res, err := h.navyDeviceService.ListDevices(c.Request.Context(), &query)
	if err != nil {
		httpx.InternalError(c, "LIST_DEVICES_ERROR", err.Error())
		return
	}

	pagination := httpx.NewPagination(res.Page, res.Size, res.Total)
	httpx.SuccessPaginated(c, res.List, pagination)
}

// QueryDevices 复杂查询
func (h *Handler) QueryDevices(c *gin.Context) {
	var req services.NavyDeviceQueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "INVALID_QUERY_REQUEST", err.Error())
		return
	}

	res, err := h.navyDeviceService.QueryDevices(c.Request.Context(), &req)
	if err != nil {
		httpx.InternalError(c, "QUERY_DEVICES_ERROR", err.Error())
		return
	}

	pagination := httpx.NewPagination(res.Page, res.Size, res.Total)
	httpx.SuccessPaginated(c, res.List, pagination)
}

// GetFilterOptions 获取筛选选项
func (h *Handler) GetFilterOptions(c *gin.Context) {
	res, err := h.navyDeviceService.GetFilterOptions(c.Request.Context())
	if err != nil {
		httpx.InternalError(c, "GET_FILTER_OPTIONS_ERROR", err.Error())
		return
	}
	httpx.Success(c, res)
}

// GetLabelValues 获取指定标签的所有值
func (h *Handler) GetLabelValues(c *gin.Context) {
	labelKey := c.Query("key")
	if labelKey == "" {
		httpx.BadRequest(c, "LABEL_KEY_REQUIRED", "key is required")
		return
	}

	values, err := h.navyDeviceService.GetLabelValues(c.Request.Context(), labelKey)
	if err != nil {
		httpx.InternalError(c, "GET_LABEL_VALUES_ERROR", err.Error())
		return
	}
	httpx.Success(c, values)
}

// GetTaintValues 获取指定污点的所有值
func (h *Handler) GetTaintValues(c *gin.Context) {
	taintKey := c.Query("key")
	if taintKey == "" {
		httpx.BadRequest(c, "TAINT_KEY_REQUIRED", "key is required")
		return
	}

	values, err := h.navyDeviceService.GetTaintValues(c.Request.Context(), taintKey)
	if err != nil {
		httpx.InternalError(c, "GET_TAINT_VALUES_ERROR", err.Error())
		return
	}
	httpx.Success(c, values)
}

// GetDeviceFieldValues 获取设备字段的所有值
func (h *Handler) GetDeviceFieldValues(c *gin.Context) {
	field := c.Query("field")
	if field == "" {
		httpx.BadRequest(c, "FIELD_REQUIRED", "field is required")
		return
	}

	values, err := h.navyDeviceService.GetDeviceFieldValues(c.Request.Context(), field)
	if err != nil {
		httpx.InternalError(c, "GET_DEVICE_FIELD_VALUES_ERROR", err.Error())
		return
	}
	httpx.Success(c, values)
}

// GetDevice 获取单个设备详情
func (h *Handler) GetDevice(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		httpx.BadRequest(c, "INVALID_ID", "无效的 ID 格式")
		return
	}

	res, err := h.navyDeviceService.GetDevice(c.Request.Context(), id)
	if err != nil {
		httpx.NotFound(c, "DEVICE_NOT_FOUND", "设备不存在")
		return
	}

	httpx.Success(c, res)
}

// GetFeatureDetails 获取设备特性详情
func (h *Handler) GetFeatureDetails(c *gin.Context) {
	ciCode := c.Query("ci_code")
	if ciCode == "" {
		httpx.BadRequest(c, "CI_CODE_REQUIRED", "ci_code is required")
		return
	}

	res, err := h.navyDeviceService.GetBatchDeviceFeatures(c.Request.Context(), []string{ciCode})
	if err != nil {
		httpx.InternalError(c, "GET_FEATURE_DETAILS_ERROR", err.Error())
		return
	}

	httpx.Success(c, res)
}

// GetDeviceFeatures 批量获取设备特性
func (h *Handler) GetDeviceFeatures(c *gin.Context) {
	var req services.DeviceFeaturesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "INVALID_REQUEST", err.Error())
		return
	}

	res, err := h.navyDeviceService.GetBatchDeviceFeatures(c.Request.Context(), req.CICodes)
	if err != nil {
		httpx.InternalError(c, "GET_DEVICE_FEATURES_ERROR", err.Error())
		return
	}

	httpx.Success(c, res)
}

// UpdateDeviceRole 更新角色
func (h *Handler) UpdateDeviceRole(c *gin.Context) {
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

	if err := h.navyDeviceService.UpdateDeviceRole(c.Request.Context(), id, req.Role); err != nil {
		httpx.InternalError(c, "UPDATE_ROLE_ERROR", err.Error())
		return
	}

	httpx.SuccessWithMessage(c, "角色更新成功", nil)
}

// UpdateDeviceGroup 更新组
func (h *Handler) UpdateDeviceGroup(c *gin.Context) {
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

	if err := h.navyDeviceService.UpdateDeviceGroup(c.Request.Context(), id, req.Group); err != nil {
		httpx.InternalError(c, "UPDATE_GROUP_ERROR", err.Error())
		return
	}

	httpx.SuccessWithMessage(c, "分组更新成功", nil)
}

// ExportDevices 导出设备
func (h *Handler) ExportDevices(c *gin.Context) {
	data, err := h.navyDeviceService.ExportDevices(c.Request.Context())
	if err != nil {
		httpx.InternalError(c, "EXPORT_DEVICES_ERROR", err.Error())
		return
	}

	c.Header("Content-Disposition", "attachment; filename=navy_devices.csv")
	c.Data(http.StatusOK, "text/csv", data)
}

// SaveTemplate 保存模板
func (h *Handler) SaveTemplate(c *gin.Context) {
	var req services.QueryTemplate
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "INVALID_REQUEST", err.Error())
		return
	}

	if err := h.navyDeviceService.SaveQueryTemplate(c.Request.Context(), &req); err != nil {
		httpx.InternalError(c, "SAVE_TEMPLATE_ERROR", err.Error())
		return
	}

	httpx.SuccessWithMessage(c, "模板保存成功", nil)
}

// GetTemplates 获取模板列表
func (h *Handler) GetTemplates(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))

	res, err := h.navyDeviceService.GetQueryTemplates(c.Request.Context(), page, size)
	if err != nil {
		httpx.InternalError(c, "GET_TEMPLATES_ERROR", err.Error())
		return
	}

	pagination := httpx.NewPagination(res.Page, res.Size, res.Total)
	httpx.SuccessPaginated(c, res.List, pagination)
}

// GetTemplate 获取单个模板
func (h *Handler) GetTemplate(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		httpx.BadRequest(c, "INVALID_ID", "无效的 ID 格式")
		return
	}

	res, err := h.navyDeviceService.GetQueryTemplate(c.Request.Context(), id)
	if err != nil {
		httpx.NotFound(c, "TEMPLATE_NOT_FOUND", "模板不存在")
		return
	}

	httpx.Success(c, res)
}

// DeleteTemplate 删除模板
func (h *Handler) DeleteTemplate(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		httpx.BadRequest(c, "INVALID_ID", "无效的 ID 格式")
		return
	}

	if err := h.navyDeviceService.DeleteQueryTemplate(c.Request.Context(), id); err != nil {
		httpx.InternalError(c, "DELETE_TEMPLATE_ERROR", err.Error())
		return
	}

	httpx.SuccessWithMessage(c, "模板删除成功", nil)
}

package http

import (
	"net/http"
	"strconv"

	"robusta-web/backend/internal/db"
	"robusta-web/backend/internal/features/configuration/services"

	"github.com/gin-gonic/gin"
)

// ConfigurationHandler 配置管理HTTP处理器
type ConfigurationHandler struct {
	labelService     *services.LabelManagementService
	taintService     *services.TaintManagementService
	deviceAppService *services.DeviceAppService
}

// NewConfigurationHandler 创建配置管理处理器
func NewConfigurationHandler(navyDB *db.NavyDatabase) *ConfigurationHandler {
	return &ConfigurationHandler{
		labelService:     services.NewLabelManagementService(navyDB),
		taintService:     services.NewTaintManagementService(navyDB),
		deviceAppService: services.NewDeviceAppService(navyDB),
	}
}

// =====================================================
// Label Management Endpoints
// =====================================================

// ListLabels 获取标签列表
func (h *ConfigurationHandler) ListLabels(c *gin.Context) {
	var query services.LabelListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.labelService.ListLabels(c.Request.Context(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetLabel 获取单个标签
func (h *ConfigurationHandler) GetLabel(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	result, err := h.labelService.GetLabel(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// CreateLabel 创建标签
func (h *ConfigurationHandler) CreateLabel(c *gin.Context) {
	var req services.CreateLabelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.labelService.CreateLabel(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, result)
}

// UpdateLabel 更新标签
func (h *ConfigurationHandler) UpdateLabel(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req services.UpdateLabelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.labelService.UpdateLabel(c.Request.Context(), id, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// DeleteLabel 删除标签
func (h *ConfigurationHandler) DeleteLabel(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	err = h.labelService.DeleteLabel(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "标签删除成功"})
}

// =====================================================
// Taint Management Endpoints
// =====================================================

// ListTaints 获取污点列表
func (h *ConfigurationHandler) ListTaints(c *gin.Context) {
	var query services.TaintListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.taintService.ListTaints(c.Request.Context(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetTaint 获取单个污点
func (h *ConfigurationHandler) GetTaint(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	result, err := h.taintService.GetTaint(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// CreateTaint 创建污点
func (h *ConfigurationHandler) CreateTaint(c *gin.Context) {
	var req services.CreateTaintRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.taintService.CreateTaint(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, result)
}

// UpdateTaint 更新污点
func (h *ConfigurationHandler) UpdateTaint(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req services.UpdateTaintRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.taintService.UpdateTaint(c.Request.Context(), id, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// DeleteTaint 删除污点
func (h *ConfigurationHandler) DeleteTaint(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	err = h.taintService.DeleteTaint(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "污点删除成功"})
}

// =====================================================
// Device App Management Endpoints
// =====================================================

// ListDeviceApps 获取设备应用列表
func (h *ConfigurationHandler) ListDeviceApps(c *gin.Context) {
	var query services.DeviceAppListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.deviceAppService.ListDeviceApps(c.Request.Context(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetDeviceApp 获取单个设备应用
func (h *ConfigurationHandler) GetDeviceApp(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	result, err := h.deviceAppService.GetDeviceApp(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// CreateDeviceApp 创建设备应用
func (h *ConfigurationHandler) CreateDeviceApp(c *gin.Context) {
	var req services.CreateDeviceAppRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.deviceAppService.CreateDeviceApp(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, result)
}

// UpdateDeviceApp 更新设备应用
func (h *ConfigurationHandler) UpdateDeviceApp(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req services.UpdateDeviceAppRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.deviceAppService.UpdateDeviceApp(c.Request.Context(), id, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// DeleteDeviceApp 删除设备应用
func (h *ConfigurationHandler) DeleteDeviceApp(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	err = h.deviceAppService.DeleteDeviceApp(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "设备应用删除成功"})
}

// RegisterRoutes 注册路由
func (h *ConfigurationHandler) RegisterRoutes(r *gin.RouterGroup) {
	// Label Management
	labels := r.Group("/labels")
	{
		labels.GET("", h.ListLabels)
		labels.GET("/:id", h.GetLabel)
		labels.POST("", h.CreateLabel)
		labels.PUT("/:id", h.UpdateLabel)
		labels.DELETE("/:id", h.DeleteLabel)
	}

	// Taint Management
	taints := r.Group("/taints")
	{
		taints.GET("", h.ListTaints)
		taints.GET("/:id", h.GetTaint)
		taints.POST("", h.CreateTaint)
		taints.PUT("/:id", h.UpdateTaint)
		taints.DELETE("/:id", h.DeleteTaint)
	}

	// Device App Management
	deviceApps := r.Group("/device-apps")
	{
		deviceApps.GET("", h.ListDeviceApps)
		deviceApps.GET("/:id", h.GetDeviceApp)
		deviceApps.POST("", h.CreateDeviceApp)
		deviceApps.PUT("/:id", h.UpdateDeviceApp)
		deviceApps.DELETE("/:id", h.DeleteDeviceApp)
	}
}

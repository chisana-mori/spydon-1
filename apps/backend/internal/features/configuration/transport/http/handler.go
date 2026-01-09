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
func NewConfigurationHandler(navyDB *db.Database) *ConfigurationHandler {
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
// @Summary 分页获取标签列表
// @Description 从系统中分页检索预定义的K8s标签及其对应的值列表。支持通过关键字过滤标签名称，返回结果包含标签的ID、Key、Value以及描述信息，主要用于前端在配置设备或资源时提供可选的标签选项。
// @Tags Configuration,Labels
// @Produce json
// @Param name query string false "按名称搜索"
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Success 200 {object} services.LabelListResponse
// @Failure 400 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /labels [get]
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
// @Summary 获取标签详情
// @Description 根据指定的标签ID获取其详细配置。返回的数据包括标签的键名、属性值、分类以及详细的使用说明。该接口常用于在编辑标签前获取旧有的配置信息，确保用户能够根据现有状态进行准确的修改。
// @Tags Configuration,Labels
// @Produce json
// @Param id path int true "标签ID"
// @Success 200 {object} models.Label
// @Failure 400 {object} gin.H
// @Failure 404 {object} gin.H
// @Router /labels/{id} [get]
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
// @Summary 新增配置标签
// @Description 在系统中创建一个新的预定义标签。用户需要提供唯一的标签Key以及对应的Value。创建成功的标签可以被后续分发到各个K8s集群的节点上，实现对基础设施资源的标准化分类与筛选管理。
// @Tags Configuration,Labels
// @Accept json
// @Produce json
// @Param request body services.CreateLabelRequest true "标签创建参数"
// @Success 201 {object} models.Label
// @Failure 400 {object} gin.H
// @Router /labels [post]
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
// @Summary 修改标签配置
// @Description 更新已有标签的属性信息，支持修改标签的显示名称、取值内容或描述性文字。更新操作仅会影响系统内的元数据定义，不会自动同步到已绑定的存量资源上，除非触发相关的资源重新同步流程。
// @Tags Configuration,Labels
// @Accept json
// @Produce json
// @Param id path int true "标签ID"
// @Param request body services.UpdateLabelRequest true "标签更新参数"
// @Success 200 {object} models.Label
// @Failure 400 {object} gin.H
// @Router /labels/{id} [put]
func (h *ConfigurationHandler) UpdateLabel(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req services.UpdateLabelRequest
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": bindErr.Error()})
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
// @Summary 物理删除标签
// @Description 根据ID从数据库中彻底移除指定的标签记录。如果该标签当前正被某些策略或任务引用，删除操作可能会受到约束或导致关联功能异常。请在删除前确认该标签已不再被任何关键业务逻辑所使用。
// @Tags Configuration,Labels
// @Param id path int true "标签ID"
// @Success 200 {object} gin.H
// @Failure 400 {object} gin.H
// @Router /labels/{id} [delete]
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
// @Summary 分页获取污点列表
// @Description 检索系统中定义的所有K8s污点（Taint）模板库。污点包含Key、Value以及Effect（如NoSchedule）。用户可以通过返回的列表选择合适的污点应用到特定的工作节点上，以控制Pod在集群内的调度行为。
// @Tags Configuration,Taints
// @Produce json
// @Param key query string false "按键名搜索"
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Success 200 {object} services.TaintListResponse
// @Failure 400 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /taints [get]
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
// @Summary 获取污点详情
// @Description 获取指定污点记录的详细定义，包括Key、Value以及定义的调度影响效果。通过此接口，管理员可以精确查看某个污点的配置细节，为后续的调度策略优化或故障排查提供基础的数据支持环境。
// @Tags Configuration,Taints
// @Produce json
// @Param id path int true "污点ID"
// @Success 200 {object} models.Taint
// @Failure 400 {object} gin.H
// @Failure 404 {object} gin.H
// @Router /taints/{id} [get]
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
// @Summary 新增污点配置
// @Description 在全局配置库中新增一个污点条目。必须指定Key和Effect属性，Value为可选。新创建的污点可以被用于机器驱逐、节点维护或特定业务隔离场景，帮助实现K8s集群中对Pod调度的高级控制。
// @Tags Configuration,Taints
// @Accept json
// @Produce json
// @Param request body services.CreateTaintRequest true "污点创建参数"
// @Success 201 {object} models.Taint
// @Failure 400 {object} gin.H
// @Router /taints [post]
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
// @Summary 修改污点属性
// @Description 修改系统中已有的污点模板。允许调整污点的Key、Value以及调度策略（Effect）。更新后的污点配置将作为后期新任务的模板，不会对当前已经应用到节点上的存量污点产生自动重写或覆盖的影响。
// @Tags Configuration,Taints
// @Accept json
// @Produce json
// @Param id path int true "污点ID"
// @Param request body services.UpdateTaintRequest true "污点更新参数"
// @Success 200 {object} models.Taint
// @Failure 400 {object} gin.H
// @Router /taints/{id} [put]
func (h *ConfigurationHandler) UpdateTaint(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req services.UpdateTaintRequest
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": bindErr.Error()})
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
// @Summary 物理删除污点
// @Description 永久移除数据库中的污点配置记录。这是一项敏感操作，删除前请确保该污点没有与任何线上自动化流程、维护脚本或资源调度策略强关联。删除操作成功后，该污点将从前端的可选配置项中彻底消失。
// @Tags Configuration,Taints
// @Param id path int true "污点ID"
// @Success 200 {object} gin.H
// @Failure 400 {object} gin.H
// @Router /taints/{id} [delete]
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
// @Summary 获取设备应用白名单
// @Description 获取配置的设备级别应用程序白名单。系统会返回所有支持的应用名称、版本号及对应的描述信息。这些信息用于在设备运维中辅助判断哪些应用程序允许运行或需要被列入特定的资源控制策略名单中。
// @Tags Configuration,DeviceApp
// @Produce json
// @Param name query string false "按名称搜索"
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Success 200 {object} services.DeviceAppListResponse
// @Failure 400 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /device-apps [get]
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
// @Summary 获取应用详情
// @Description 根据ID查询特定设备应用程序的配置详情。返回的信息涵盖了应用的基本定义、配置参数以及预期的运行环境要求。这些数据对于理解应用如何在底层设备上分发和管理起着至关重要的桥梁作用。
// @Tags Configuration,DeviceApp
// @Produce json
// @Param id path int true "应用ID"
// @Success 200 {object} models.DeviceApp
// @Failure 400 {object} gin.H
// @Failure 404 {object} gin.H
// @Router /device-apps/{id} [get]
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
// @Summary 新增设备应用
// @Description 在资产配置库中登记一个新的设备级应用程序。需要输入应用名称、关键字以及详细的业务背景描述。成功创建后，该应用将作为系统内的合规资产，可用于后续的资源打标、自动监控或准入检查流程中。
// @Tags Configuration,DeviceApp
// @Accept json
// @Produce json
// @Param request body services.CreateDeviceAppRequest true "应用创建参数"
// @Success 201 {object} models.DeviceApp
// @Failure 400 {object} gin.H
// @Router /device-apps [post]
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
// @Summary 修改应用配置
// @Description 更新已登记的设备应用元数据。管理员可以随时调整应用的名称或关联的分类关键字，以反映实时的资产变动情况。注意：此接口仅修改管理侧的配置数据，不会实时触发客户端运行时的任何状态更新。
// @Tags Configuration,DeviceApp
// @Accept json
// @Produce json
// @Param id path int true "应用ID"
// @Param request body services.UpdateDeviceAppRequest true "应用更新参数"
// @Success 200 {object} models.DeviceApp
// @Failure 400 {object} gin.H
// @Router /device-apps/{id} [put]
func (h *ConfigurationHandler) UpdateDeviceApp(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req services.UpdateDeviceAppRequest
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": bindErr.Error()})
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
// @Summary 物理删除应用配置
// @Description 从系统中永久删除指定的设备应用定义。此操作将导致以往关联的历史记录失去元数据引用，因此在执行删除前需确保没有活跃的业务逻辑依赖该应用ID。删除操作一经确认完成，相关数据将无法直接恢复。
// @Tags Configuration,DeviceApp
// @Param id path int true "应用ID"
// @Success 200 {object} gin.H
// @Failure 400 {object} gin.H
// @Router /device-apps/{id} [delete]
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

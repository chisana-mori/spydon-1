package http

import (
	"net/http"

	"robusta-web/backend/internal/features/systemsetting/services"
	httpx "robusta-web/backend/internal/transport/httpx"

	"github.com/gin-gonic/gin"
)

// AutoRCARefresher is a minimal interface used to refresh Auto-RCA config.
type AutoRCARefresher interface {
	RefreshAutoRCAConfig()
}

// Handler for system settings.
type Handler struct {
	service    *services.SystemSettingService
	rcaService AutoRCARefresher
}

// New creates a new Handler.
func New(service *services.SystemSettingService, rcaService AutoRCARefresher) *Handler {
	return &Handler{service: service, rcaService: rcaService}
}

// RegisterRoutes registers routes under the admin group.
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	g := r.Group(RouteGroup)
	g.GET("", h.ListSettings)
	g.PUT("/:"+ParamKey, h.UpdateSetting)
	g.GET(SubAutoRCA, h.GetAutoRCAConfig)
	g.POST(SubAutoRCAInit, h.InitAutoRCAConfig)
}

// ListSettings returns all settings.
// @Summary 获取系统设置列表
// @Description 检索系统中所有已定义的全局配置项。这包括平台的基础信息、外部服务的接入参数、以及各种功能模块的开关状态。返回结果支撑了系统设置管理页面的展示，让管理员能全局审视当前平台的运行参数配置详情。
// @Tags Admin,Settings
// @Produce json
// @Success 200 {object} httpx.Response{data=[]object}
// @Failure 500 {object} httpx.ErrorResponse
// @Router /admin/settings [get]
func (h *Handler) ListSettings(c *gin.Context) {
	settings, err := h.service.ListSettings()
	if err != nil {
		httpx.ErrorWithDetails(c, http.StatusInternalServerError, "LIST_SETTINGS_FAILED", "获取设置列表失败", err.Error())
		return
	}
	httpx.Success(c, settings)
}

// UpdateSettingRequest represents the payload to update a setting.
type UpdateSettingRequest struct {
	Value       interface{} `json:"value" binding:"required"`
	Description string      `json:"description"`
}

// UpdateSetting updates the specified setting.
// @Summary 修改系统配置项
// @Description 根据唯一Key值更新特定系统设置的内容。管理员可以修改配置的值（Value）并附带变更说明。针对某些核心设置（如Auto-RCA），该操作会触发后台服务的实时热加载流程，确保新配置能立即在相关业务逻辑中生效应用。
// @Tags Admin,Settings
// @Accept json
// @Produce json
// @Param key path string true "设置项Key"
// @Param request body UpdateSettingRequest true "设置项更新详情"
// @Success 200 {object} httpx.Response{data=object}
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /admin/settings/{key} [put]
func (h *Handler) UpdateSetting(c *gin.Context) {
	var path struct {
		Key string `uri:"key" binding:"required"`
	}
	if err := httpx.BindURI(c, &path); err != nil {
		httpx.BadRequest(c, "MISSING_KEY", "设置Key不能为空")
		return
	}

	var req UpdateSettingRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.AbortWithDomainError(c, err)
		return
	}

	setting, err := h.service.SetSetting(path.Key, req.Value, req.Description)
	if err != nil {
		httpx.ErrorWithDetails(c, http.StatusInternalServerError, "UPDATE_SETTING_FAILED", "更新设置失败", err.Error())
		return
	}

	if h.rcaService != nil && path.Key == services.SettingKeyAutoRCA {
		h.rcaService.RefreshAutoRCAConfig()
	}

	httpx.Success(c, setting)
}

// GetAutoRCAConfig gets the Auto-RCA configuration.
// @Summary 获取Auto-RCA配置详情
// @Description 专门用于获取自动根因分析（Auto-RCA）模块的精细化配置。返回数据包含启用的告警类型、分析阈值及关联的检测规则。通过此接口，运维人员可以确认当前自动诊断功能的运行策略及其最后一次更新的时间。
// @Tags Admin,Settings
// @Produce json
// @Success 200 {object} httpx.Response{data=object}
// @Failure 500 {object} httpx.ErrorResponse
// @Router /admin/settings/auto-rca [get]
func (h *Handler) GetAutoRCAConfig(c *gin.Context) {
	config, updatedAt, err := h.service.GetAutoRCAConfig()
	if err != nil {
		httpx.ErrorWithDetails(c, http.StatusInternalServerError, "AUTO_RCA_CONFIG_FETCH_FAILED", "获取Auto-RCA配置失败", err.Error())
		return
	}
	httpx.Success(c, gin.H{
		"config":     config,
		"updated_at": updatedAt,
	})
}

// InitAutoRCAConfig initializes the Auto-RCA configuration with defaults.
// @Summary 初始化Auto-RCA默认配置
// @Description 在系统首次部署或配置丢失时，用于将自动根因分析模块恢复到出厂默认设置。该操作会执行前置检查以防覆盖现有配置。初始化完成后，Auto-RCA服务将根据这些基准策略自动开始对新产生的告警进行诊断分析。
// @Tags Admin,Settings
// @Produce json
// @Success 200 {object} httpx.Response{data=object}
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /admin/settings/auto-rca/init [post]
func (h *Handler) InitAutoRCAConfig(c *gin.Context) {
	existing, err := h.service.GetSetting(services.SettingKeyAutoRCA)
	if err != nil {
		httpx.ErrorWithDetails(c, http.StatusInternalServerError, "INIT_CONFIG_FAILED", "检查现有配置失败", err.Error())
		return
	}
	if existing != nil {
		httpx.BadRequest(c, "CONFIG_ALREADY_EXISTS", "Auto-RCA配置已存在，请使用更新接口")
		return
	}

	defaultConfig := services.GetDefaultAutoRCAConfig()
	setting, err := h.service.SetSetting(
		services.SettingKeyAutoRCA,
		defaultConfig,
		"Auto-RCA自动根因分析配置",
	)
	if err != nil {
		httpx.ErrorWithDetails(c, http.StatusInternalServerError, "INIT_CONFIG_FAILED", "初始化配置失败", err.Error())
		return
	}

	if h.rcaService != nil {
		h.rcaService.RefreshAutoRCAConfig()
	}

	httpx.SuccessWithMessage(c, "Auto-RCA配置已初始化", setting)
}

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

package api

import (
	"net/http"

	"robusta-web/backend/internal/services"

	"github.com/gin-gonic/gin"
)

// SystemSettingHandler 系统设置处理器
type SystemSettingHandler struct {
	service    *services.SystemSettingService
	rcaService *services.RCAService
}

// NewSystemSettingHandler 创建新的系统设置处理器
func NewSystemSettingHandler(service *services.SystemSettingService, rcaService *services.RCAService) *SystemSettingHandler {
	return &SystemSettingHandler{
		service:    service,
		rcaService: rcaService,
	}
}

// ListSettings 列出所有设置
func (h *SystemSettingHandler) ListSettings(c *gin.Context) {
	settings, err := h.service.ListSettings()
	if err != nil {
		ErrorWithDetails(c, http.StatusInternalServerError, "LIST_SETTINGS_FAILED", "获取设置列表失败", err.Error())
		return
	}
	Success(c, settings)
}

// UpdateSettingRequest 更新设置请求
type UpdateSettingRequest struct {
	Value       interface{} `json:"value" binding:"required"`
	Description string      `json:"description"`
}

// UpdateSetting 更新指定设置
func (h *SystemSettingHandler) UpdateSetting(c *gin.Context) {
	var path struct {
		Key string `uri:"key" binding:"required"`
	}
	if err := c.ShouldBindUri(&path); err != nil {
		BadRequest(c, "MISSING_KEY", "设置Key不能为空")
		return
	}

	var req UpdateSettingRequest
	if err := bindJSON(c, &req); err != nil {
		AbortWithDomainError(c, err)
		return
	}

	setting, err := h.service.SetSetting(path.Key, req.Value, req.Description)
	if err != nil {
		ErrorWithDetails(c, http.StatusInternalServerError, "UPDATE_SETTING_FAILED", "更新设置失败", err.Error())
		return
	}

	// 如果更新的是 Auto-RCA 配置，刷新 RCA 服务配置
	if h.rcaService != nil && path.Key == services.SettingKeyAutoRCA {
		h.rcaService.RefreshAutoRCAConfig()
	}

	Success(c, setting)
}

// GetAutoRCAConfig 获取 Auto-RCA 配置
func (h *SystemSettingHandler) GetAutoRCAConfig(c *gin.Context) {
	config, updatedAt, err := h.service.GetAutoRCAConfig()
	if err != nil {
		ErrorWithDetails(c, http.StatusInternalServerError, "AUTO_RCA_CONFIG_FETCH_FAILED", "获取Auto-RCA配置失败", err.Error())
		return
	}

	Success(c, gin.H{
		"config":     config,
		"updated_at": updatedAt,
	})
}

// InitAutoRCAConfig 初始化 Auto-RCA 配置（使用默认值）
func (h *SystemSettingHandler) InitAutoRCAConfig(c *gin.Context) {
	// 检查是否已存在配置
	existing, err := h.service.GetSetting(services.SettingKeyAutoRCA)
	if err != nil {
		ErrorWithDetails(c, http.StatusInternalServerError, "INIT_CONFIG_FAILED", "检查现有配置失败", err.Error())
		return
	}
	if existing != nil {
		BadRequest(c, "CONFIG_ALREADY_EXISTS", "Auto-RCA配置已存在，请使用更新接口")
		return
	}

	// 使用默认配置初始化
	defaultConfig := services.GetDefaultAutoRCAConfig()
	setting, err := h.service.SetSetting(
		services.SettingKeyAutoRCA,
		defaultConfig,
		"Auto-RCA自动根因分析配置",
	)
	if err != nil {
		ErrorWithDetails(c, http.StatusInternalServerError, "INIT_CONFIG_FAILED", "初始化配置失败", err.Error())
		return
	}

	// 刷新 RCA 服务配置
	if h.rcaService != nil {
		h.rcaService.RefreshAutoRCAConfig()
	}

	SuccessWithMessage(c, "Auto-RCA配置已初始化", setting)
}

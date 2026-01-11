package http

import (
	"fmt"
	"net/http"
	"time"

	"robusta-web/backend/internal/apperrors"
	"robusta-web/backend/internal/config"
	"robusta-web/backend/internal/constants"
	"robusta-web/backend/internal/features/apikey/services"
	"robusta-web/backend/internal/middleware"
	"robusta-web/backend/internal/models"
	httpx "robusta-web/backend/internal/transport/httpx"

	"github.com/gin-gonic/gin"
)

// Handler wires API Key routes, preserving existing URL paths and response shapes.
type Handler struct {
	cfg           *config.Config
	apiKeyService *services.APIKeyService
}

func New(cfg *config.Config, apiKeyService *services.APIKeyService) *Handler {
	return &Handler{cfg: cfg, apiKeyService: apiKeyService}
}

func (h *Handler) RegisterRoutes(v1, admin *gin.RouterGroup) {
	userGroup := v1.Group(RouteGroupUser)
	userGroup.Use(middleware.CookieAuthMiddleware(h.cfg))
	userGroup.Use(middleware.AuditLogMiddleware())
	{
		userGroup.POST("", h.CreateAPIKey)
		userGroup.GET("", h.ListAPIKeys)
		userGroup.DELETE(":id", h.DeleteAPIKey)
		userGroup.PUT(":id/status", h.UpdateAPIKeyStatus)
	}

	admin.GET(RouteGroupAdmin, h.ListAllAPIKeys)
}

type CreateAPIKeyRequest struct {
	Name        string `json:"name" binding:"required"`
	ExpiresIn   *int   `json:"expires_in"`
	Permissions string `json:"permissions" binding:"required,oneof=read write admin"`
}

type CreateAPIKeyResponse struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Key         string     `json:"key"`
	KeyPrefix   string     `json:"key_prefix"`
	ExpiresAt   *time.Time `json:"expires_at"`
	Permissions string     `json:"permissions"`
	CreatedAt   time.Time  `json:"created_at"`
}

type APIKeyListResponse struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	KeyPrefix   string     `json:"key_prefix"`
	LastUsedAt  *time.Time `json:"last_used_at"`
	ExpiresAt   *time.Time `json:"expires_at"`
	IsActive    bool       `json:"is_active"`
	Permissions string     `json:"permissions"`
	CreatedAt   time.Time  `json:"created_at"`
}

// CreateAPIKey 创建API Key
// @Description 为当前登录用户创建一个新的API访问密钥。用户可以指定密钥名称、有效期（以天为单位）以及权限级别（read/write/admin）。生成的密钥将仅在创建时返回一次，用于后续自动化脚本或第三方集成调用API。
// @Tags APIKey
// @Accept json
// @Produce json
// @Param request body CreateAPIKeyRequest true "API Key创建请求参数"
// @Success 201 {object} httpx.Response{data=CreateAPIKeyResponse}
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /apikeys [post]
func (h *Handler) CreateAPIKey(c *gin.Context) {
	var req CreateAPIKeyRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.AbortWithDomainError(c, err)
		return
	}

	uid, derr := h.resolveUserID(c)
	if derr != nil {
		httpx.AbortWithDomainError(c, derr)
		return
	}

	var expiresAt *time.Time
	if req.ExpiresIn != nil && *req.ExpiresIn > 0 {
		expiry := time.Now().AddDate(0, 0, *req.ExpiresIn)
		expiresAt = &expiry
	}

	apiKey, rawKey, err := h.apiKeyService.GenerateAPIKey(uid, req.Name, expiresAt, req.Permissions)
	if err != nil {
		httpx.ErrorWithDetails(c, http.StatusInternalServerError, constants.ErrorCodeCreateFailed, "创建API Key失败", err.Error())
		return
	}

	resp := CreateAPIKeyResponse{
		ID:          models.FormatID(apiKey.ID),
		Name:        apiKey.Name,
		Key:         rawKey, // only return once
		KeyPrefix:   apiKey.KeyPrefix,
		ExpiresAt:   apiKey.ExpiresAt,
		Permissions: apiKey.Permissions,
		CreatedAt:   apiKey.CreatedAt,
	}

	c.AbortWithStatusJSON(constants.StatusCreated, gin.H{"data": resp})
}

// ListAPIKeys 获取API Key列表
// @Description 获取当前用户拥有的所有API访问密钥列表。返回结果包含密钥名称、前缀、最后使用时间、过期时间、当前状态（激活/禁用）以及权限级别。出于安全考虑，完整的密钥内容不会在此接口中返回。
// @Tags APIKey
// @Produce json
// @Success 200 {object} httpx.Response{data=[]APIKeyListResponse}
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /apikeys [get]
func (h *Handler) ListAPIKeys(c *gin.Context) {
	uid, derr := h.resolveUserID(c)
	if derr != nil {
		httpx.AbortWithDomainError(c, derr)
		return
	}

	apiKeys, err := h.apiKeyService.ListAPIKeys(uid)
	if err != nil {
		httpx.ErrorWithDetails(c, http.StatusInternalServerError, constants.ErrorCodeListFailed, "获取API Key列表失败", err.Error())
		return
	}

	resp := make([]APIKeyListResponse, len(apiKeys))
	for i, key := range apiKeys {
		resp[i] = APIKeyListResponse{
			ID:          models.FormatID(key.ID),
			Name:        key.Name,
			KeyPrefix:   key.KeyPrefix,
			LastUsedAt:  key.LastUsedAt,
			ExpiresAt:   key.ExpiresAt,
			IsActive:    key.IsActive,
			Permissions: key.Permissions,
			CreatedAt:   key.CreatedAt,
		}
	}
	httpx.Success(c, resp)
}

// DeleteAPIKey 删除API Key
// @Description 根据指定的ID永久删除当前用户下的某个API访问密钥。删除后，使用该密钥的任何请求都将被拒绝。此操作不可撤销，旨在让用户在密钥泄露或不再需要时能及时清理过期的安全凭证。
// @Tags APIKey
// @Produce json
// @Param id path string true "API Key ID"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /apikeys/{id} [delete]
func (h *Handler) DeleteAPIKey(c *gin.Context) {
	var uri struct {
		ID uint64 `uri:"id" binding:"required,gt=0"`
	}
	if err := c.ShouldBindUri(&uri); err != nil {
		httpx.BadRequest(c, "INVALID_ID", "无效的API Key ID")
		return
	}

	uid, derr := h.resolveUserID(c)
	if derr != nil {
		httpx.AbortWithDomainError(c, derr)
		return
	}

	if err := h.apiKeyService.DeleteAPIKey(uri.ID, uid); err != nil {
		httpx.ErrorWithDetails(c, http.StatusInternalServerError, constants.ErrorCodeDeleteFailed, err.Error(), nil)
		return
	}
	httpx.SuccessWithMessage(c, "API Key删除成功", nil)
}

// UpdateAPIKeyStatus 更新API Key状态
// @Description 启用或禁用当前用户下的指定API访问密钥。禁用密钥可以临时阻止基于该密钥的API访问，而无需永久删除密钥信息。这在进行安全审计或临时调整权限时非常有用，且后续可以随时重新启用。
// @Tags APIKey
// @Accept json
// @Produce json
// @Param id path string true "API Key ID"
// @Param request body object true "状态更新参数 (is_active boolean)"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /apikeys/{id}/status [put]
func (h *Handler) UpdateAPIKeyStatus(c *gin.Context) {
	var uri struct {
		ID uint64 `uri:"id" binding:"required,gt=0"`
	}
	if err := c.ShouldBindUri(&uri); err != nil {
		httpx.BadRequest(c, "INVALID_ID", "无效的API Key ID")
		return
	}
	var req struct {
		IsActive bool `json:"is_active"`
	}
	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.AbortWithDomainError(c, err)
		return
	}

	uid, derr := h.resolveUserID(c)
	if derr != nil {
		httpx.AbortWithDomainError(c, derr)
		return
	}

	if err := h.apiKeyService.UpdateAPIKeyStatus(uri.ID, uid, req.IsActive); err != nil {
		httpx.ErrorWithDetails(c, http.StatusInternalServerError, constants.ErrorCodeUpdateFailed, err.Error(), nil)
		return
	}
	httpx.SuccessWithMessage(c, "API Key状态更新成功", nil)
}

// ListAllAPIKeys 管理员获取所有API Key (分页)
// @Description 管理员权限接口，用于分页列出系统中所有用户创建的API访问密钥。返回数据包含密钥详情以及所属用户信息，支持通过分页参数控制返回数量，方便管理员全局监控和管理系统内的安全凭证使用情况。
// @Tags Admin,APIKey
// @Produce json
// @Param page query int false "页码 (默认1)"
// @Param page_size query int false "每页数量 (默认20)"
// @Success 200 {object} httpx.Response{data=[]map[string]interface{}}
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /admin/apikeys [get]
func (h *Handler) ListAllAPIKeys(c *gin.Context) {
	params, derr := httpx.ParsePaginationParams(c)
	if derr != nil {
		httpx.AbortWithDomainError(c, derr)
		return
	}

	apiKeys, total, err := h.apiKeyService.ListAllAPIKeys(params.Page, params.PageSize)
	if err != nil {
		httpx.ErrorWithDetails(c, http.StatusInternalServerError, constants.ErrorCodeListFailed, "获取API Key列表失败", err.Error())
		return
	}

	data := make([]map[string]interface{}, len(apiKeys))
	for i, key := range apiKeys {
		data[i] = map[string]interface{}{
			"id":           models.FormatID(key.ID),
			"name":         key.Name,
			"key_prefix":   key.KeyPrefix,
			"last_used_at": key.LastUsedAt,
			"expires_at":   key.ExpiresAt,
			"is_active":    key.IsActive,
			"permissions":  key.Permissions,
			"created_at":   key.CreatedAt,
			"user": map[string]interface{}{
				"id":       models.FormatID(key.User.ID),
				"username": key.User.Username,
				"email":    key.User.Email,
				"name":     key.User.Name,
			},
		}
	}

	pagination := httpx.NewPagination(params.Page, params.PageSize, total)
	pagination.Sort = params.Sort
	httpx.SuccessPaginated(c, data, pagination)
}

func (h *Handler) resolveUserID(c *gin.Context) (uint64, apperrors.DomainError) {
	userID, exists := c.Get("user_id")
	if !exists {
		return 0, apperrors.Unauthorized("未授权", apperrors.WithCode(constants.ErrorCodeUnauthorized))
	}
	uid, err := toUint64(userID)
	if err != nil {
		return 0, apperrors.BadRequest("无效的用户ID", nil, apperrors.WithCode("INVALID_USER_ID"), apperrors.WithCause(err))
	}
	return uid, nil
}

func toUint64(value interface{}) (uint64, error) {
	switch v := value.(type) {
	case uint64:
		return v, nil
	case int:
		if v < 0 {
			return 0, fmt.Errorf("negative value")
		}
		return uint64(v), nil
	case int64:
		if v < 0 {
			return 0, fmt.Errorf("negative value")
		}
		return uint64(v), nil
	case float64:
		if v < 0 || v != float64(int64(v)) {
			return 0, fmt.Errorf("invalid float value")
		}
		return uint64(v), nil
	case string:
		return models.ParseID(v)
	default:
		return 0, fmt.Errorf("unsupported type %T", value)
	}
}

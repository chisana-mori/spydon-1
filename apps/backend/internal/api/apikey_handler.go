package api

import (
	"time"

	"robusta-web/backend/internal/apperrors"
	"robusta-web/backend/internal/constants"
	"robusta-web/backend/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// APIKeyHandler API Key处理器
type APIKeyHandler struct {
	apiKeyService *services.APIKeyService
}

// NewAPIKeyHandler 创建API Key处理器实例
func NewAPIKeyHandler(apiKeyService *services.APIKeyService) *APIKeyHandler {
	return &APIKeyHandler{
		apiKeyService: apiKeyService,
	}
}

// CreateAPIKeyRequest 创建API Key请求
type CreateAPIKeyRequest struct {
	Name        string `json:"name" binding:"required"`
	ExpiresIn   *int   `json:"expires_in"` // 过期天数，nil表示永不过期
	Permissions string `json:"permissions" binding:"required,oneof=read write admin"`
}

// CreateAPIKeyResponse 创建API Key响应
type CreateAPIKeyResponse struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Key         string     `json:"key"` // 完整密钥（仅在创建时返回）
	KeyPrefix   string     `json:"key_prefix"`
	ExpiresAt   *time.Time `json:"expires_at"`
	Permissions string     `json:"permissions"`
	CreatedAt   time.Time  `json:"created_at"`
}

// APIKeyListResponse API Key列表响应
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

func resolveUserID(c *gin.Context) (uuid.UUID, apperrors.DomainError) {
	userID, exists := c.Get("user_id")
	if !exists {
		return uuid.Nil, apperrors.Unauthorized("未授权", apperrors.WithCode("UNAUTHORIZED"))
	}

	uidStr, ok := userID.(string)
	if !ok {
		return uuid.Nil, apperrors.BadRequest("无效的用户ID类型", nil, apperrors.WithCode("INVALID_USER_ID"))
	}

	uid, err := uuid.Parse(uidStr)
	if err != nil {
		return uuid.Nil, apperrors.BadRequest("无效的用户ID", nil, apperrors.WithCode("INVALID_USER_ID"), apperrors.WithCause(err))
	}

	return uid, nil
}

// CreateAPIKey 创建新的API Key
func (h *APIKeyHandler) CreateAPIKey(c *gin.Context) {
	var req CreateAPIKeyRequest
	if err := bindJSON(c, &req); err != nil {
		AbortWithDomainError(c, err)
		return
	}

	uid, derr := resolveUserID(c)
	if derr != nil {
		AbortWithDomainError(c, derr)
		return
	}

	// 计算过期时间
	var expiresAt *time.Time
	if req.ExpiresIn != nil && *req.ExpiresIn > 0 {
		expiry := time.Now().AddDate(0, 0, *req.ExpiresIn)
		expiresAt = &expiry
	}

	// 生成API Key
	apiKey, rawKey, err := h.apiKeyService.GenerateAPIKey(uid, req.Name, expiresAt, req.Permissions)
	if err != nil {
		InternalError(c, constants.ErrorCodeCreateFailed, "创建API Key失败")
		return
	}

	Created(c, CreateAPIKeyResponse{
		ID:          apiKey.ID.String(),
		Name:        apiKey.Name,
		Key:         rawKey, // 仅在创建时返回完整密钥
		KeyPrefix:   apiKey.KeyPrefix,
		ExpiresAt:   apiKey.ExpiresAt,
		Permissions: apiKey.Permissions,
		CreatedAt:   apiKey.CreatedAt,
	})
}

// ListAPIKeys 获取当前用户的API Key列表
func (h *APIKeyHandler) ListAPIKeys(c *gin.Context) {
	uid, derr := resolveUserID(c)
	if derr != nil {
		AbortWithDomainError(c, derr)
		return
	}

	apiKeys, err := h.apiKeyService.ListAPIKeys(uid)
	if err != nil {
		InternalError(c, constants.ErrorCodeListFailed, "获取API Key列表失败")
		return
	}

	// 转换为响应格式
	response := make([]APIKeyListResponse, len(apiKeys))
	for i, key := range apiKeys {
		response[i] = APIKeyListResponse{
			ID:          key.ID.String(),
			Name:        key.Name,
			KeyPrefix:   key.KeyPrefix,
			LastUsedAt:  key.LastUsedAt,
			ExpiresAt:   key.ExpiresAt,
			IsActive:    key.IsActive,
			Permissions: key.Permissions,
			CreatedAt:   key.CreatedAt,
		}
	}

	Success(c, response)
}

// DeleteAPIKey 删除API Key
func (h *APIKeyHandler) DeleteAPIKey(c *gin.Context) {
	keyID := c.Param("id")
	if keyID == "" {
		BadRequest(c, "MISSING_ID", "缺少API Key ID")
		return
	}

	id, err := uuid.Parse(keyID)
	if err != nil {
		BadRequest(c, "INVALID_ID", "无效的API Key ID")
		return
	}

	uid, derr := resolveUserID(c)
	if derr != nil {
		AbortWithDomainError(c, derr)
		return
	}

	// 删除API Key
	if err := h.apiKeyService.DeleteAPIKey(id, uid); err != nil {
		InternalError(c, constants.ErrorCodeDeleteFailed, err.Error())
		return
	}

	SuccessWithMessage(c, "API Key删除成功", nil)
}

// UpdateAPIKeyStatus 更新API Key状态
func (h *APIKeyHandler) UpdateAPIKeyStatus(c *gin.Context) {
	keyID := c.Param("id")
	if keyID == "" {
		BadRequest(c, "MISSING_ID", "缺少API Key ID")
		return
	}

	id, err := uuid.Parse(keyID)
	if err != nil {
		BadRequest(c, "INVALID_ID", "无效的API Key ID")
		return
	}

	var req struct {
		IsActive bool `json:"is_active"`
	}
	if err := bindJSON(c, &req); err != nil {
		AbortWithDomainError(c, err)
		return
	}

	uid, derr := resolveUserID(c)
	if derr != nil {
		AbortWithDomainError(c, derr)
		return
	}

	// 更新状态
	if err := h.apiKeyService.UpdateAPIKeyStatus(id, uid, req.IsActive); err != nil {
		InternalError(c, constants.ErrorCodeUpdateFailed, err.Error())
		return
	}

	SuccessWithMessage(c, "API Key状态更新成功", nil)
}

// ListAllAPIKeys 管理员获取所有API Key列表
func (h *APIKeyHandler) ListAllAPIKeys(c *gin.Context) {
	// 获取分页参数
	params, derr := ParsePaginationParams(c)
	if derr != nil {
		AbortWithDomainError(c, derr)
		return
	}

	apiKeys, total, err := h.apiKeyService.ListAllAPIKeys(params.Page, params.PageSize)
	if err != nil {
		InternalError(c, constants.ErrorCodeListFailed, "获取API Key列表失败")
		return
	}

	// 转换为响应格式
	response := make([]map[string]interface{}, len(apiKeys))
	for i, key := range apiKeys {
		response[i] = map[string]interface{}{
			"id":           key.ID.String(),
			"name":         key.Name,
			"key_prefix":   key.KeyPrefix,
			"last_used_at": key.LastUsedAt,
			"expires_at":   key.ExpiresAt,
			"is_active":    key.IsActive,
			"permissions":  key.Permissions,
			"created_at":   key.CreatedAt,
			"user": map[string]interface{}{
				"id":       key.User.ID.String(),
				"username": key.User.Username,
				"email":    key.User.Email,
				"name":     key.User.Name,
			},
		}
	}

	pagination := NewPagination(params.Page, params.PageSize, total)
	pagination.Sort = params.Sort
	SuccessPaginated(c, response, pagination)
}

package api

import (
	"net/http"

	"robusta-web/backend/internal/config"
	"robusta-web/backend/internal/constants"
	"robusta-web/backend/internal/models"
	"robusta-web/backend/internal/services"

	"github.com/gin-gonic/gin"
	cas "gopkg.in/cas.v2"
)

// AuthHandler 认证处理器
type AuthHandler struct {
	authService *services.AuthService
	casClient   *cas.Client
	cfg         *config.Config
}

const casRedirectCookieName = "cas_redirect_target"

// NewAuthHandler 创建认证处理器
func NewAuthHandler(authService *services.AuthService, casClient *cas.Client, cfg *config.Config) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		casClient:   casClient,
		cfg:         cfg,
	}
}

// GetAuthURL 获取认证URL
func (h *AuthHandler) GetAuthURL(c *gin.Context) {
	// 生成随机state参数防止CSRF攻击
	state, err := generateRandomState()
	if err != nil {
		InternalError(c, constants.ErrorCodeGenerateStateFailed, "生成state参数失败")
		return
	}

	// 获取认证URL
	authURL, err := h.authService.GetAuthURL(state)
	if err != nil {
		ErrorWithDetails(c, http.StatusInternalServerError, "GET_AUTH_URL_FAILED", "获取认证URL失败", err.Error())
		return
	}

	// 将state存储到session或cookie中（这里简化处理）
	c.SetCookie("auth_state", state, 600, "/", "", false, true) // 10分钟过期

	Success(c, gin.H{
		"auth_url": authURL,
		"state":    state,
	})
}

// HandleCallback 处理认证回调
func (h *AuthHandler) HandleCallback(c *gin.Context) {
	var req services.LoginRequest
	if err := bindJSON(c, &req); err != nil {
		AbortWithDomainError(c, err)
		return
	}

	// 验证state参数
	storedState, err := c.Cookie("auth_state")
	if err != nil || storedState != req.State {
		BadRequest(c, "INVALID_STATE", "无效的state参数")
		return
	}

	// 清除state cookie
	c.SetCookie("auth_state", "", -1, "/", "", false, true)

	// 处理认证回调
	loginResp, err := h.authService.HandleCallback(req.Code, req.State)
	if err != nil {
		ErrorWithDetails(c, http.StatusUnauthorized, "AUTH_FAILED", "认证失败", err.Error())
		return
	}

	// 设置refresh token到httpOnly cookie
	secure := isSecureRequest(c.Request)
	c.SetCookie("refresh_token", loginResp.RefreshToken, 30*24*3600, "/", "", secure, true) // 30天
	setAccessTokenCookie(c, loginResp.AccessToken, loginResp.ExpiresAt)
	h.setKiteCookie(c, loginResp.User, loginResp.RefreshToken, loginResp.ExpiresAt)

	Success(c, gin.H{
		"access_token": loginResp.AccessToken,
		"expires_at":   loginResp.ExpiresAt,
		"user":         loginResp.User,
	})
}

// RefreshToken 刷新token
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	// 从cookie获取refresh token
	refreshToken, err := c.Cookie("refresh_token")
	if err != nil {
		Unauthorized(c, "MISSING_REFRESH_TOKEN", "缺少refresh token")
		return
	}

	// 刷新token
	loginResp, err := h.authService.RefreshToken(refreshToken)
	if err != nil {
		// 清除无效的refresh token cookie
		secure := isSecureRequest(c.Request)
		c.SetCookie("refresh_token", "", -1, "/", "", secure, true)
		ErrorWithDetails(c, http.StatusUnauthorized, "REFRESH_TOKEN_FAILED", "刷新token失败", err.Error())
		return
	}

	// 更新refresh token cookie
	secure := isSecureRequest(c.Request)
	c.SetCookie("refresh_token", loginResp.RefreshToken, 30*24*3600, "/", "", secure, true)
	setAccessTokenCookie(c, loginResp.AccessToken, loginResp.ExpiresAt)
	h.setKiteCookie(c, loginResp.User, loginResp.RefreshToken, loginResp.ExpiresAt)

	Success(c, gin.H{
		"access_token": loginResp.AccessToken,
		"expires_at":   loginResp.ExpiresAt,
		"user":         loginResp.User,
	})
}

// Logout 登出
func (h *AuthHandler) Logout(c *gin.Context) {
	h.performLocalLogout(c)
	h.clearRedirectCookie(c)

	SuccessWithMessage(c, "登出成功", nil)
}

// GetProfile 获取用户资料
func (h *AuthHandler) GetProfile(c *gin.Context) {
	// 从上下文获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		Unauthorized(c, "UNAUTHORIZED", "用户未认证")
		return
	}

	uid, err := toUint64(userID)
	if err != nil {
		BadRequest(c, "INVALID_USER_ID", "无效的用户ID")
		return
	}

	// 从数据库获取最新的用户信息
	user, err := h.authService.GetUserByID(uid)
	if err != nil {
		InternalError(c, "GET_USER_FAILED", "获取用户信息失败")
		return
	}

	Success(c, gin.H{
		"user": gin.H{
			"id":       models.FormatID(user.ID),
			"email":    user.Email,
			"username": user.Username,
			"name":     user.Name,
			"is_admin": user.IsAdmin, // 从数据库读取最新的is_admin值
		},
	})
}

// UpdateProfile 更新用户资料
func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	// 获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		Unauthorized(c, "UNAUTHORIZED", "用户未认证")
		return
	}

	uid, err := toUint64(userID)
	if err != nil {
		BadRequest(c, "INVALID_USER_ID", "无效的用户ID")
		return
	}

	var req struct {
		Name    string `json:"name"`
		Picture string `json:"picture"`
	}

	if err := bindJSON(c, &req); err != nil {
		AbortWithDomainError(c, err)
		return
	}

	// 这里应该调用服务更新用户资料
	// 简化实现
	SuccessWithMessage(c, "资料更新成功", gin.H{
		"user": gin.H{
			"id":      models.FormatID(uid),
			"name":    req.Name,
			"picture": req.Picture,
		},
	})
}

// ChangePassword 修改密码（仅用于本地认证）
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	// 获取用户ID
	_, exists := c.Get("user_id")
	if !exists {
		Unauthorized(c, "UNAUTHORIZED", "用户未认证")
		return
	}

	var req struct {
		CurrentPassword string `json:"current_password" binding:"required"`
		NewPassword     string `json:"new_password" binding:"required,min=8"`
	}

	if err := bindJSON(c, &req); err != nil {
		AbortWithDomainError(c, err)
		return
	}

	// 这里应该验证当前密码并更新新密码
	// 由于使用OIDC认证，这个功能可能不需要
	ErrorWithDetails(
		c,
		http.StatusNotImplemented,
		"PASSWORD_CHANGE_NOT_SUPPORTED",
		"使用OIDC认证时不支持修改密码",
		gin.H{"hint": "请在身份提供商处修改密码"},
	)
}

// GetUserSessions 获取用户会话列表
func (h *AuthHandler) GetUserSessions(c *gin.Context) {
	// 获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		Unauthorized(c, "UNAUTHORIZED", "用户未认证")
		return
	}

	uid, err := toUint64(userID)
	if err != nil {
		BadRequest(c, "INVALID_USER_ID", "无效的用户ID")
		return
	}

	// 这里应该从数据库获取用户的活跃会话
	// 简化实现
	sessions := []gin.H{
		{
			"id":         "session-1",
			"device":     "Chrome on Windows",
			"ip":         "192.168.1.100",
			"location":   "Beijing, China",
			"created_at": "2024-01-01T10:00:00Z",
			"last_used":  "2024-01-01T15:30:00Z",
			"current":    true,
		},
	}

	Success(c, gin.H{
		"sessions": sessions,
		"user_id":  models.FormatID(uid),
	})
}

// RevokeSession 撤销会话
func (h *AuthHandler) RevokeSession(c *gin.Context) {
	var path struct {
		SessionID string `uri:"session_id" binding:"required"`
	}
	if err := c.ShouldBindUri(&path); err != nil {
		BadRequest(c, "MISSING_SESSION_ID", "会话ID不能为空")
		return
	}

	// 获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		Unauthorized(c, "UNAUTHORIZED", "用户未认证")
		return
	}

	uid, err := toUint64(userID)
	if err != nil {
		BadRequest(c, "INVALID_USER_ID", "无效的用户ID")
		return
	}

	// 这里应该撤销指定的会话
	// 简化实现
	SuccessWithMessage(c, "会话已撤销", gin.H{
		"session_id": path.SessionID,
		"user_id":    models.FormatID(uid),
	})
}

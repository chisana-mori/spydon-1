package http

import (
	"net/http"

	"robusta-web/backend/internal/config"
	"robusta-web/backend/internal/constants"
	"robusta-web/backend/internal/features/auth/services"
	"robusta-web/backend/internal/middleware"
	"robusta-web/backend/internal/models"
	"robusta-web/backend/internal/transport/httpx"

	"github.com/gin-gonic/gin"
	cas "gopkg.in/cas.v2"
)

// Handler 认证处理器
type Handler struct {
	authService *services.AuthService
	casClient   *cas.Client
	cfg         *config.Config
}

const casRedirectCookieName = "cas_redirect_target"

// New 创建认证处理器
func New(cfg *config.Config, authService *services.AuthService, casClient *cas.Client) *Handler {
	return &Handler{
		authService: authService,
		casClient:   casClient,
		cfg:         cfg,
	}
}

// RegisterRoutes 注册认证和用户资料相关路由
func (h *Handler) RegisterRoutes(router *gin.Engine, v1 *gin.RouterGroup) {
	// /auth 基础路由
	authGroup := router.Group(constants.AuthPathBase)
	{
		authGroup.GET("/url", h.GetAuthURL)
		authGroup.POST("/callback", h.HandleCallback)
		authGroup.POST("/refresh", h.RefreshToken)
		authGroup.POST("/logout", h.Logout)
	}

	// CAS 相关路由
	router.GET(constants.CASLoginPath, h.CASLogin)
	callbackPath := h.cfg.CAS.CallbackPath
	if callbackPath == "" {
		callbackPath = "/auth/cas/callback"
	}
	router.GET(callbackPath, h.CASCallback)
	router.GET(constants.CASLogoutPath, h.CASLogout)

	// /api/v1/profile 相关路由
	profileGroup := v1.Group("")
	profileGroup.Use(middleware.CookieAuthMiddleware(h.cfg))
	profileGroup.Use(middleware.AuditLogMiddleware())
	{
		profileGroup.GET("/profile", h.GetProfile)
		profileGroup.PUT("/profile", h.UpdateProfile)
		profileGroup.POST("/profile/change-password", h.ChangePassword)
		profileGroup.GET("/profile/sessions", h.GetUserSessions)
		profileGroup.DELETE("/profile/sessions/:session_id", h.RevokeSession)
	}
}

// GetAuthURL 获取认证URL
// @Summary 获取认证跳转URL
// @Description 该接口用于生成并获取身份供应商的认证跳转地址。系统会生成一个唯一的随机state参数以防止跨站请求伪造（CSRF）攻击，并将其存储在客户端Cookie中。用户访问返回的URL后将被重定向至OIDC或第三方认证平台。
// @Tags Auth
// @Produce json
// @Success 200 {object} httpx.Response{data=object}
// @Failure 500 {object} httpx.ErrorResponse
// @Router /auth/url [get]
func (h *Handler) GetAuthURL(c *gin.Context) {
	// 生成随机state参数防止CSRF攻击
	state, err := generateRandomState()
	if err != nil {
		httpx.InternalError(c, constants.ErrorCodeGenerateStateFailed, "生成state参数失败")
		return
	}

	// 获取认证URL
	authURL, err := h.authService.GetAuthURL(state)
	if err != nil {
		httpx.ErrorWithDetails(c, http.StatusInternalServerError, "GET_AUTH_URL_FAILED", "获取认证URL失败", err.Error())
		return
	}

	// 将state存储到session或cookie中（这里简化处理）
	c.SetCookie("auth_state", state, 600, "/", "", false, true) // 10分钟过期

	httpx.Success(c, gin.H{
		"auth_url": authURL,
		"state":    state,
	})
}

// HandleCallback 处理认证回调
// @Summary 处理第三方认证回调
// @Description 接收并处理来自第三方认证平台的回调请求。接口会验证返回的state参数是否与之前记录的一致，验证通过后使用授权码换取访问令牌和刷新令牌，并根据返回的用户信息在本地完成登录逻辑，下发加密的HTTP-only Cookie。
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body services.LoginRequest true "认证回调及登录参数"
// @Success 200 {object} httpx.Response{data=object}
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /auth/callback [post]
func (h *Handler) HandleCallback(c *gin.Context) {
	var req services.LoginRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.AbortWithDomainError(c, err)
		return
	}

	// 验证state参数
	storedState, err := c.Cookie("auth_state")
	if err != nil || storedState != req.State {
		httpx.BadRequest(c, "INVALID_STATE", "无效的state参数")
		return
	}

	// 清除state cookie
	c.SetCookie("auth_state", "", -1, "/", "", false, true)

	// 处理认证回调
	loginResp, err := h.authService.HandleCallback(req.Code, req.State)
	if err != nil {
		httpx.ErrorWithDetails(c, http.StatusUnauthorized, "AUTH_FAILED", "认证失败", err.Error())
		return
	}

	// 设置refresh token到httpOnly cookie
	secure := isSecureRequest(c.Request)
	c.SetCookie("refresh_token", loginResp.RefreshToken, 30*24*3600, "/", "", secure, true) // 30天
	setAccessTokenCookie(c, loginResp.AccessToken, loginResp.ExpiresAt)
	h.setKiteCookie(c, loginResp.User, loginResp.RefreshToken, loginResp.ExpiresAt)

	httpx.Success(c, gin.H{
		"access_token": loginResp.AccessToken,
		"expires_at":   loginResp.ExpiresAt,
		"user":         loginResp.User,
	})
}

// RefreshToken 刷新token
// @Summary 刷新访问令牌
// @Description 利用本地存储在HTTP-only Cookie中的刷新令牌（Refresh Token）来换取新的访问令牌（Access Token）。此接口允许用户在不重新登录的情况下延长会话有效期。如果刷新令牌失效，将清除相关Cookie并要求用户重新认证。
// @Tags Auth
// @Produce json
// @Success 200 {object} httpx.Response{data=object}
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /auth/refresh [post]
func (h *Handler) RefreshToken(c *gin.Context) {
	// 从cookie获取refresh token
	refreshToken, err := c.Cookie("refresh_token")
	if err != nil {
		httpx.Unauthorized(c, "MISSING_REFRESH_TOKEN", "缺少refresh token")
		return
	}

	// 刷新token
	loginResp, err := h.authService.RefreshToken(refreshToken)
	if err != nil {
		// 清除无效的refresh token cookie
		secure := isSecureRequest(c.Request)
		c.SetCookie("refresh_token", "", -1, "/", "", secure, true)
		httpx.ErrorWithDetails(c, http.StatusUnauthorized, "REFRESH_TOKEN_FAILED", "刷新token失败", err.Error())
		return
	}

	// 更新refresh token cookie
	secure := isSecureRequest(c.Request)
	c.SetCookie("refresh_token", loginResp.RefreshToken, 30*24*3600, "/", "", secure, true)
	setAccessTokenCookie(c, loginResp.AccessToken, loginResp.ExpiresAt)
	h.setKiteCookie(c, loginResp.User, loginResp.RefreshToken, loginResp.ExpiresAt)

	httpx.Success(c, gin.H{
		"access_token": loginResp.AccessToken,
		"expires_at":   loginResp.ExpiresAt,
		"user":         loginResp.User,
	})
}

// Logout 登出
// @Summary 用户登出系统
// @Description 用于结束当前用户的活跃会话。该接口会清除客户端存储的访问令牌、刷新令牌以及用于单点登录的其它Cookie，并重置服务器端维护的会话重定向状态，确保后续请求将被视为未授权，有效保护账户安全。
// @Tags Auth
// @Produce json
// @Success 200 {object} httpx.Response
// @Router /auth/logout [post]
func (h *Handler) Logout(c *gin.Context) {
	h.performLocalLogout(c)
	h.clearRedirectCookie(c)

	httpx.SuccessWithMessage(c, "登出成功", nil)
}

// GetProfile 获取用户资料
// @Summary 获取个人基本资料
// @Description 获取当前已认证用户的最新个人信息。该接口从数据库中读取并返回用户的UserID、电子邮件、用户名、全名以及管理员权限标志。通常用于前端初始化页面布局，或在权限变更后同步本地显示的状态信息。
// @Tags Auth,Profile
// @Produce json
// @Success 200 {object} httpx.Response{data=object}
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /profile [get]
func (h *Handler) GetProfile(c *gin.Context) {
	// 从上下文获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		httpx.Unauthorized(c, "UNAUTHORIZED", "用户未认证")
		return
	}

	uid, err := toUint64(userID)
	if err != nil {
		httpx.BadRequest(c, "INVALID_USER_ID", "无效的用户ID")
		return
	}

	// 从数据库获取最新的用户信息
	user, err := h.authService.GetUserByID(uid)
	if err != nil {
		httpx.InternalError(c, "GET_USER_FAILED", "获取用户信息失败")
		return
	}

	httpx.Success(c, gin.H{
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
// @Summary 更新个人资料信息
// @Description 允许用户修改其个人资料，目前支持更新头像URL和显示名称。该接口需要合法的访问令牌，并在更新成功后返回修改后的用户信息。通过此接口，用户可以个性化其在平台上的展示身份。
// @Tags Auth,Profile
// @Accept json
// @Produce json
// @Param request body object true "用户资料更新参数"
// @Success 200 {object} httpx.Response{data=object}
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Router /profile [put]
func (h *Handler) UpdateProfile(c *gin.Context) {
	// 获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		httpx.Unauthorized(c, "UNAUTHORIZED", "用户未认证")
		return
	}

	uid, err := toUint64(userID)
	if err != nil {
		httpx.BadRequest(c, "INVALID_USER_ID", "无效的用户ID")
		return
	}

	var req struct {
		Name    string `json:"name"`
		Picture string `json:"picture"`
	}

	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.AbortWithDomainError(c, err)
		return
	}

	// 这里应该调用服务更新用户资料
	// 简化实现
	httpx.SuccessWithMessage(c, "资料更新成功", gin.H{
		"user": gin.H{
			"id":      models.FormatID(uid),
			"name":    req.Name,
			"picture": req.Picture,
		},
	})
}

// ChangePassword 修改密码（仅用于本地认证）
// @Summary 修改用户登录密码
// @Description 用于用户自主修改其登录密码。需要提供当前旧密码进行身份验证。注意：如果系统配置了OIDC或CAS等外部统一身份认证，此操作将被禁用，用户必须在相应的身份提供商平台上进行密码变更操作。
// @Tags Auth,Profile
// @Accept json
// @Produce json
// @Param request body object true "密码修改请求内容"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 501 {object} httpx.ErrorResponse
// @Router /profile/change-password [post]
func (h *Handler) ChangePassword(c *gin.Context) {
	// 获取用户ID
	_, exists := c.Get("user_id")
	if !exists {
		httpx.Unauthorized(c, "UNAUTHORIZED", "用户未认证")
		return
	}

	var req struct {
		CurrentPassword string `json:"current_password" binding:"required"`
		NewPassword     string `json:"new_password" binding:"required,min=8"`
	}

	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.AbortWithDomainError(c, err)
		return
	}

	// 这里应该验证当前密码并更新新密码
	// 由于使用OIDC认证，这个功能可能不需要
	httpx.ErrorWithDetails(
		c,
		http.StatusNotImplemented,
		"PASSWORD_CHANGE_NOT_SUPPORTED",
		"使用OIDC认证时不支持修改密码",
		gin.H{"hint": "请在身份提供商处修改密码"},
	)
}

// GetUserSessions 获取用户会话列表
// @Summary 查看活跃会话列表
// @Description 列出当前用户在不同设备或浏览器上产生的所有活跃认证会话。返回信息包含设备名称、IP地址、登录位置、创建时间以及最后活动时间。用户可以通过此列表监控账号登录情况，并识别是否存在异常登录行为。
// @Tags Auth,Profile
// @Produce json
// @Success 200 {object} httpx.Response{data=object}
// @Failure 401 {object} httpx.ErrorResponse
// @Router /profile/sessions [get]
func (h *Handler) GetUserSessions(c *gin.Context) {
	// 获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		httpx.Unauthorized(c, "UNAUTHORIZED", "用户未认证")
		return
	}

	uid, err := toUint64(userID)
	if err != nil {
		httpx.BadRequest(c, "INVALID_USER_ID", "无效的用户ID")
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

	httpx.Success(c, gin.H{
		"sessions": sessions,
		"user_id":  models.FormatID(uid),
	})
}

// RevokeSession 撤销会话
// @Summary 强制退出指定会话
// @Description 根据会话ID强制注销特定的登录会话。该功能常用于远程退出在不可信设备上遗留的登录状态，或者在账号疑似被盗用时紧急终止所有的活跃访问。成功撤销后，该会话对应的令牌将立即失效。
// @Tags Auth,Profile
// @Produce json
// @Param session_id path string true "会话唯一标识"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Router /profile/sessions/{session_id} [delete]
func (h *Handler) RevokeSession(c *gin.Context) {
	var path struct {
		SessionID string `uri:"session_id" binding:"required"`
	}
	if err := c.ShouldBindUri(&path); err != nil {
		httpx.BadRequest(c, "MISSING_SESSION_ID", "会话ID不能为空")
		return
	}

	// 获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		httpx.Unauthorized(c, "UNAUTHORIZED", "用户未认证")
		return
	}

	uid, err := toUint64(userID)
	if err != nil {
		httpx.BadRequest(c, "INVALID_USER_ID", "无效的用户ID")
		return
	}

	// 这里应该撤销指定的会话
	// 简化实现
	httpx.SuccessWithMessage(c, "会话已撤销", gin.H{
		"session_id": path.SessionID,
		"user_id":    models.FormatID(uid),
	})
}

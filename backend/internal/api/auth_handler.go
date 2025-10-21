package api

import (
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"robusta-web/backend/internal/config"
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
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "生成state参数失败",
		})
		return
	}

	// 获取认证URL
	authURL, err := h.authService.GetAuthURL(state)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "获取认证URL失败",
			"details": err.Error(),
		})
		return
	}

	// 将state存储到session或cookie中（这里简化处理）
	c.SetCookie("auth_state", state, 600, "/", "", false, true) // 10分钟过期

	c.JSON(http.StatusOK, gin.H{
		"auth_url": authURL,
		"state":    state,
	})
}

// HandleCallback 处理认证回调
func (h *AuthHandler) HandleCallback(c *gin.Context) {
	var req services.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "无效的请求参数",
			"details": err.Error(),
		})
		return
	}

	// 验证state参数
	storedState, err := c.Cookie("auth_state")
	if err != nil || storedState != req.State {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的state参数",
		})
		return
	}

	// 清除state cookie
	c.SetCookie("auth_state", "", -1, "/", "", false, true)

	// 处理认证回调
	loginResp, err := h.authService.HandleCallback(req.Code, req.State)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "认证失败",
			"details": err.Error(),
		})
		return
	}

	// 设置refresh token到httpOnly cookie
	secure := isSecureRequest(c.Request)
	c.SetCookie("refresh_token", loginResp.RefreshToken, 30*24*3600, "/", "", secure, true) // 30天
	setAccessTokenCookie(c, loginResp.AccessToken, loginResp.ExpiresAt)

	c.JSON(http.StatusOK, gin.H{
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
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "缺少refresh token",
		})
		return
	}

	// 刷新token
	loginResp, err := h.authService.RefreshToken(refreshToken)
	if err != nil {
		// 清除无效的refresh token cookie
		secure := isSecureRequest(c.Request)
		c.SetCookie("refresh_token", "", -1, "/", "", secure, true)
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "刷新token失败",
			"details": err.Error(),
		})
		return
	}

	// 更新refresh token cookie
	secure := isSecureRequest(c.Request)
	c.SetCookie("refresh_token", loginResp.RefreshToken, 30*24*3600, "/", "", secure, true)
	setAccessTokenCookie(c, loginResp.AccessToken, loginResp.ExpiresAt)

	c.JSON(http.StatusOK, gin.H{
		"access_token": loginResp.AccessToken,
		"expires_at":   loginResp.ExpiresAt,
		"user":         loginResp.User,
	})
}

// Logout 登出
func (h *AuthHandler) Logout(c *gin.Context) {
	h.performLocalLogout(c)
	h.clearRedirectCookie(c)

	c.JSON(http.StatusOK, gin.H{
		"message": "登出成功",
	})
}

// CASLogin 触发CAS登录
func (h *AuthHandler) CASLogin(c *gin.Context) {
	if h.casClient == nil || h.cfg == nil || !h.cfg.CAS.Enabled {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "CAS 未启用"})
		return
	}

	redirectTarget := strings.TrimSpace(c.Query("service"))
	if redirectTarget == "" {
		redirectTarget = h.cfg.CAS.RedirectURL
	}
	if redirectTarget == "" {
		redirectTarget = "/"
	}

	secure := isSecureRequest(c.Request)
	encodedTarget := base64.URLEncoding.EncodeToString([]byte(redirectTarget))
	c.SetCookie(casRedirectCookieName, encodedTarget, 600, "/", "", secure, true)

	if strings.TrimSpace(h.cfg.CAS.ServerURL) == "" {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "CAS 服务地址未配置",
		})
		return
	}

	callbackURL, err := h.buildCallbackURL(c.Request)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "构造回调地址失败",
			"details": err.Error(),
		})
		return
	}

	loginURL, err := buildCASLoginURL(h.cfg.CAS.ServerURL, callbackURL.String())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "构造CAS登录地址失败",
			"details": err.Error(),
		})
		return
	}

	c.Redirect(http.StatusFound, loginURL)
}

// CASCallback 处理CAS回调
func (h *AuthHandler) CASCallback(c *gin.Context) {
	if h.casClient == nil || h.cfg == nil || !h.cfg.CAS.Enabled {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "CAS 未启用"})
		return
	}

	ticket := strings.TrimSpace(c.Query("ticket"))
	if ticket == "" {
		h.casClient.RedirectToLogin(c.Writer, c.Request)
		return
	}

	authResp, err := h.validateCASTicket(c.Request, ticket)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "CAS ticket 验证失败",
			"details": err.Error(),
		})
		return
	}

	username := authResp.User
	attributes := cas.UserAttributes(authResp.Attributes)
	loginResp, err := h.authService.LoginWithCAS(username, attributes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "CAS 登录失败",
			"details": err.Error(),
		})
		return
	}

	secure := isSecureRequest(c.Request)
	c.SetCookie("refresh_token", loginResp.RefreshToken, 30*24*3600, "/", "", secure, true)
	setAccessTokenCookie(c, loginResp.AccessToken, loginResp.ExpiresAt)

	redirectTarget := ""
	if value, err := c.Cookie(casRedirectCookieName); err == nil && value != "" {
		if decoded, decodeErr := base64.URLEncoding.DecodeString(value); decodeErr == nil {
			redirectTarget = string(decoded)
		}
	}
	if redirectTarget == "" {
		redirectTarget = h.cfg.CAS.RedirectURL
	}
	if redirectTarget == "" {
		redirectTarget = "/"
	}
	h.clearRedirectCookie(c)

	c.Redirect(http.StatusFound, redirectTarget)
}

// CASValidate 验证 CAS ticket (用于前端 callback 调用)
func (h *AuthHandler) CASValidate(c *gin.Context) {
	// 这个方法应该通过 CAS 中间件调用，而不是直接处理
	// 实际的验证逻辑在 CASCallback 中处理
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": "请使用 /auth/cas/callback 接口",
	})
}

// CASLogout 注销CAS并清理本地会话
func (h *AuthHandler) CASLogout(c *gin.Context) {
	redirectTarget := c.Query("redirect")
	if redirectTarget == "" && h.cfg != nil {
		redirectTarget = h.cfg.CAS.RedirectURL
	}
	if redirectTarget == "" {
		redirectTarget = "/"
	}

	h.performLocalLogout(c)

	if h.casClient == nil || h.cfg == nil || !h.cfg.CAS.Enabled {
		c.Redirect(http.StatusFound, redirectTarget)
		return
	}

	logoutURL, err := h.casClient.LogoutUrlForRequest(c.Request)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "构造CAS登出地址失败",
			"details": err.Error(),
		})
		return
	}

	if redirectTarget != "" {
		if parsed, parseErr := url.Parse(logoutURL); parseErr == nil {
			query := parsed.Query()
			query.Set("service", redirectTarget)
			parsed.RawQuery = query.Encode()
			logoutURL = parsed.String()
		}
	}

	c.Redirect(http.StatusFound, logoutURL)
}

// GetProfile 获取用户资料
func (h *AuthHandler) GetProfile(c *gin.Context) {
	// 从上下文获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "用户未认证",
		})
		return
	}

	// 从数据库获取最新的用户信息（而不是从JWT token中获取）
	// 这样可以确保获取到最新的is_admin状态
	user, err := h.authService.GetUserByID(userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "获取用户信息失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"id":       user.ID.String(),
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
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "用户未认证",
		})
		return
	}

	var req struct {
		Name    string `json:"name"`
		Picture string `json:"picture"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "无效的请求参数",
			"details": err.Error(),
		})
		return
	}

	// 这里应该调用服务更新用户资料
	// 简化实现
	c.JSON(http.StatusOK, gin.H{
		"message": "资料更新成功",
		"user": gin.H{
			"id":      userID,
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
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "用户未认证",
		})
		return
	}

	var req struct {
		CurrentPassword string `json:"current_password" binding:"required"`
		NewPassword     string `json:"new_password" binding:"required,min=8"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "无效的请求参数",
			"details": err.Error(),
		})
		return
	}

	// 这里应该验证当前密码并更新新密码
	// 由于使用OIDC认证，这个功能可能不需要
	c.JSON(http.StatusNotImplemented, gin.H{
		"error":   "使用OIDC认证时不支持修改密码",
		"message": "请在身份提供商处修改密码",
	})
}

// GetUserSessions 获取用户会话列表
func (h *AuthHandler) GetUserSessions(c *gin.Context) {
	// 获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "用户未认证",
		})
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

	c.JSON(http.StatusOK, gin.H{
		"sessions": sessions,
		"user_id":  userID,
	})
}

// RevokeSession 撤销会话
func (h *AuthHandler) RevokeSession(c *gin.Context) {
	sessionID := c.Param("session_id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "会话ID不能为空",
		})
		return
	}

	// 获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "用户未认证",
		})
		return
	}

	// 这里应该撤销指定的会话
	// 简化实现
	c.JSON(http.StatusOK, gin.H{
		"message":    "会话已撤销",
		"session_id": sessionID,
		"user_id":    userID,
	})
}

// 辅助函数

func generateRandomState() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

func (h *AuthHandler) performLocalLogout(c *gin.Context) {
	refreshToken, err := c.Cookie("refresh_token")
	if err == nil && refreshToken != "" {
		h.authService.Logout(refreshToken)
	}

	secure := isSecureRequest(c.Request)
	c.SetCookie("refresh_token", "", -1, "/", "", secure, true)
	c.SetCookie("access_token", "", -1, "/", "", secure, true)
}

func setAccessTokenCookie(c *gin.Context, token string, expiresAt time.Time) {
	if token == "" {
		return
	}

	ttl := int(time.Until(expiresAt).Seconds())
	if ttl <= 0 {
		ttl = 3600
	}

	secure := isSecureRequest(c.Request)

	c.SetCookie("access_token", token, ttl, "/", "", secure, true)
}

func (h *AuthHandler) clearRedirectCookie(c *gin.Context) {
	secure := isSecureRequest(c.Request)
	c.SetCookie(casRedirectCookieName, "", -1, "/", "", secure, true)
}

func (h *AuthHandler) buildCallbackURL(r *http.Request) (*url.URL, error) {
	base := buildRequestBaseURL(r)
	callbackPath := h.cfg.CAS.CallbackPath
	if callbackPath == "" {
		callbackPath = "/auth/cas/callback"
	}

	baseURL, err := url.Parse(base)
	if err != nil {
		return nil, err
	}
	rel, err := url.Parse(callbackPath)
	if err != nil {
		return nil, err
	}
	return baseURL.ResolveReference(rel), nil
}

func buildCASLoginURL(serverURL, serviceURL string) (string, error) {
	server := strings.TrimRight(serverURL, "/") + "/login"
	loginURL, err := url.Parse(server)
	if err != nil {
		return "", err
	}
	query := loginURL.Query()
	query.Set("service", serviceURL)
	loginURL.RawQuery = query.Encode()
	return loginURL.String(), nil
}

func buildRequestBaseURL(r *http.Request) string {
	scheme := "http"
	if isSecureRequest(r) {
		scheme = "https"
	}
	host := r.Host
	if xfHost := r.Header.Get("X-Forwarded-Host"); xfHost != "" {
		parts := strings.Split(xfHost, ",")
		host = strings.TrimSpace(parts[0])
	}
	if host == "" {
		host = "localhost"
	}
	return fmt.Sprintf("%s://%s", scheme, host)
}

func isSecureRequest(r *http.Request) bool {
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		parts := strings.Split(proto, ",")
		return strings.EqualFold(strings.TrimSpace(parts[0]), "https")
	}
	return r.TLS != nil
}

func (h *AuthHandler) validateCASTicket(r *http.Request, ticket string) (*cas.AuthenticationResponse, error) {
	serviceURL, err := h.buildCallbackURL(r)
	if err != nil {
		return nil, fmt.Errorf("构造回调地址失败: %w", err)
	}

	casBase := strings.TrimSpace(h.cfg.CAS.ServerURL)
	if casBase == "" {
		return nil, fmt.Errorf("CAS 服务地址未配置")
	}

	casURL, err := url.Parse(casBase)
	if err != nil {
		return nil, fmt.Errorf("CAS 服务地址无效: %w", err)
	}

	// 创建自定义HTTP客户端，开发环境跳过TLS验证
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{
		Timeout:   10 * time.Second,
		Transport: tr,
	}

	validator := cas.NewServiceTicketValidator(client, casURL)
	resp, err := validator.ValidateTicket(serviceURL, ticket)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, fmt.Errorf("CAS 未返回有效认证信息")
	}

	return resp, nil
}

package api

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"

	"robusta-web/backend/internal/services"

	"github.com/gin-gonic/gin"
)

// AuthHandler 认证处理器
type AuthHandler struct {
	authService *services.AuthService
}

// NewAuthHandler 创建认证处理器
func NewAuthHandler(authService *services.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
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
	c.SetCookie("refresh_token", loginResp.RefreshToken, 30*24*3600, "/", "", true, true) // 30天

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
		c.SetCookie("refresh_token", "", -1, "/", "", true, true)
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "刷新token失败",
			"details": err.Error(),
		})
		return
	}

	// 更新refresh token cookie
	c.SetCookie("refresh_token", loginResp.RefreshToken, 30*24*3600, "/", "", true, true)

	c.JSON(http.StatusOK, gin.H{
		"access_token": loginResp.AccessToken,
		"expires_at":   loginResp.ExpiresAt,
		"user":         loginResp.User,
	})
}

// Logout 登出
func (h *AuthHandler) Logout(c *gin.Context) {
	// 从cookie获取refresh token
	refreshToken, err := c.Cookie("refresh_token")
	if err == nil {
		// 撤销refresh token
		h.authService.Logout(refreshToken)
	}

	// 清除refresh token cookie
	c.SetCookie("refresh_token", "", -1, "/", "", true, true)

	c.JSON(http.StatusOK, gin.H{
		"message": "登出成功",
	})
}

// GetProfile 获取用户资料
func (h *AuthHandler) GetProfile(c *gin.Context) {
	// 从上下文获取用户信息
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "用户未认证",
		})
		return
	}

	userEmail, _ := c.Get("user_email")
	userRoles, _ := c.Get("user_roles")

	c.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"id":    userID,
			"email": userEmail,
			"roles": userRoles,
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

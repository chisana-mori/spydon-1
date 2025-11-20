package api

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"robusta-web/backend/internal/logger"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

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
		if logoutErr := h.authService.Logout(refreshToken); logoutErr != nil {
			// 记录日志但不影响主流程
			logger.L().Warn("登出时发生错误", zap.Error(logoutErr))
		}
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
	base := h.buildRequestBaseURL(r) // 包含 basePath，如 "http://host/spydon"
	callbackPath := h.cfg.CAS.CallbackPath
	if callbackPath == "" {
		callbackPath = "/auth/cas/callback"
	}

	// 确保 callbackPath 以 / 开头
	if !strings.HasPrefix(callbackPath, "/") {
		callbackPath = "/" + callbackPath
	}

	// 直接拼接 base + callbackPath
	// base 已经包含了 basePath，所以最终结果是 http://host/spydon/auth/cas/callback
	fullURL := strings.TrimRight(base, "/") + callbackPath

	return url.Parse(fullURL)
}

func (h *AuthHandler) buildRequestBaseURL(r *http.Request) string {
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

	// 添加 basePath（如果配置了）
	basePath := ""
	if h.cfg != nil && h.cfg.BasePath != "" {
		basePath = strings.TrimRight(h.cfg.BasePath, "/")
		if !strings.HasPrefix(basePath, "/") {
			basePath = "/" + basePath
		}
	}

	return fmt.Sprintf("%s://%s%s", scheme, host, basePath)
}

func isSecureRequest(r *http.Request) bool {
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		parts := strings.Split(proto, ",")
		return strings.EqualFold(strings.TrimSpace(parts[0]), "https")
	}
	return r.TLS != nil
}

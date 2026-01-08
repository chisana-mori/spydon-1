package http

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"robusta-web/backend/internal/config"
	"robusta-web/backend/internal/features/auth/services"
	"robusta-web/backend/internal/models"
	"robusta-web/backend/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

func generateRandomState() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

func (h *Handler) performLocalLogout(c *gin.Context) {
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
	c.SetCookie("auth_token", "", -1, "/", "", secure, true)
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

	// 使用 http.SetCookie 以支持 SameSite 属性
	// 跨域场景 (前端3000, 后端8080) 需要 SameSite=None
	// 注意: SameSite=None 在生产环境需要配合 Secure=true
	cookie := &http.Cookie{
		Name:     "access_token",
		Value:    token,
		MaxAge:   ttl,
		Path:     "/",
		Domain:   "",
		Secure:   secure,
		HttpOnly: true,
		SameSite: http.SameSiteNoneMode,
	}
	// 开发环境允许 SameSite=None 在非 HTTPS 下使用
	if !secure {
		cookie.SameSite = http.SameSiteLaxMode
	}
	http.SetCookie(c.Writer, cookie)
}

func (h *Handler) clearRedirectCookie(c *gin.Context) {
	secure := isSecureRequest(c.Request)
	c.SetCookie(casRedirectCookieName, "", -1, "/", "", secure, true)
}

func (h *Handler) buildCallbackURL(r *http.Request) (*url.URL, error) {
	callbackPath := h.cfg.CAS.CallbackPath
	if callbackPath == "" {
		callbackPath = "/auth/cas/callback"
	}

	// 确保 callbackPath 以 / 开头
	if !strings.HasPrefix(callbackPath, "/") {
		callbackPath = "/" + callbackPath
	}

	// 优先使用配置的 ServiceURL，确保 CAS 登录和回调使用相同的 service URL
	if serviceURL := strings.TrimSpace(h.cfg.CAS.ServiceURL); serviceURL != "" {
		base := strings.TrimRight(serviceURL, "/")
		return url.Parse(base + callbackPath)
	}

	// 回退到基于请求的动态构建
	base := h.buildRequestBaseURL(r)
	fullURL := strings.TrimRight(base, "/") + callbackPath

	return url.Parse(fullURL)
}

func (h *Handler) buildRequestBaseURL(r *http.Request) string {
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

// KiteClaims represents JWT claims compatible with Kite
type KiteClaims struct {
	UserID       uint   `json:"user_id"`
	Username     string `json:"username"`
	Provider     string `json:"provider"`
	RefreshToken string `json:"refresh_token,omitempty"`
	IsAdmin      bool   `json:"is_admin"`
	jwt.RegisteredClaims
}

func generateKiteToken(user services.UserInfo, refreshToken string, cfg *config.Config) (string, error) {
	uid, err := models.ParseID(user.ID)
	if err != nil {
		return "", fmt.Errorf("invalid user id format: %w", err)
	}

	now := time.Now()
	// Kite's default expiration is usually 24h as well, aligning with Robusta
	expirationTime := now.Add(24 * time.Hour)

	claims := KiteClaims{
		UserID:       uint(uid),
		Username:     user.Username,
		Provider:     "robusta", // Set provider to identify source
		RefreshToken: refreshToken,
		IsAdmin:      user.IsAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "Kite",
			Subject:   fmt.Sprintf("%d", uid),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(cfg.JWTSecret))
}

func setKiteAuthCookie(c *gin.Context, token string, expiresAt time.Time) {
	if token == "" {
		return
	}

	ttl := int(time.Until(expiresAt).Seconds())
	if ttl <= 0 {
		ttl = 3600
	}

	secure := isSecureRequest(c.Request)

	// Note: We're setting path to "/" and domain to "" (host only)
	// If Kite is on a different subdomain, Domain needs to be configured.
	// For now assuming same host or localhost.
	c.SetCookie("auth_token", token, ttl, "/", "", secure, true)
}

func (h *Handler) setKiteCookie(c *gin.Context, user services.UserInfo, refreshToken string, expiresAt time.Time) {
	token, err := generateKiteToken(user, refreshToken, h.cfg)
	if err != nil {
		logger.L().Warn("Failed to generate Kite token", zap.Error(err))
		return
	}
	setKiteAuthCookie(c, token, expiresAt)
}

// toUint64 将多种类型转换为 uint64
func toUint64(value interface{}) (uint64, error) {
	switch v := value.(type) {
	case uint64:
		return v, nil
	case uint:
		return uint64(v), nil
	case int64:
		if v < 0 {
			return 0, fmt.Errorf("negative value %d cannot be converted to uint64", v)
		}
		return uint64(v), nil
	case int:
		if v < 0 {
			return 0, fmt.Errorf("negative value %d cannot be converted to uint64", v)
		}
		return uint64(v), nil
	case string:
		id, err := models.ParseID(v)
		if err != nil {
			return 0, err
		}
		return id, nil
	default:
		return 0, fmt.Errorf("unsupported type %T for uint64 conversion", value)
	}
}

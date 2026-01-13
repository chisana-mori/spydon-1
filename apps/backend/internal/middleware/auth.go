package middleware

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"robusta-web/backend/internal/config"
	"robusta-web/backend/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// bypassAuthInDevMode 检查是否应该在开发模式下绕过认证
// 需同时满足：environment=development 且 dev_mode_auth_bypass=true
func bypassAuthInDevMode(cfg *config.Config, c *gin.Context) bool {
	if cfg.Environment == "development" && cfg.DevModeAuthBypass {
		setDevAdmin(c)
		return true
	}
	return false
}

// AuthMiddleware JWT认证中间件
func AuthMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		if bypassAuthInDevMode(cfg, c) {
			return
		}

		tokenString := extractBearerToken(c.GetHeader("Authorization"))
		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "缺少认证头",
				"code":  "MISSING_AUTH_HEADER",
			})
			c.Abort()
			return
		}

		claims, err := parseJWTClaims(cfg, tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "无效的token",
				"code":  "INVALID_TOKEN",
			})
			c.Abort()
			return
		}

		setUserClaims(c, claims)
		c.Next()
	}
}

// OptionalAuthMiddleware 可选认证中间件（用于某些公开接口）
func OptionalAuthMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		if bypassAuthInDevMode(cfg, c) {
			return
		}

		tokenString := extractBearerToken(c.GetHeader("Authorization"))
		if tokenString == "" {
			c.Next()
			return
		}

		claims, err := parseJWTClaims(cfg, tokenString)
		if err == nil {
			setUserClaims(c, claims)
		}

		c.Next()
	}
}

// CookieAuthMiddleware 支持通过Cookie或Authorization头验证JWT
func CookieAuthMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		if bypassAuthInDevMode(cfg, c) {
			return
		}

		tokenString := ""

		if cookieToken, err := c.Cookie("access_token"); err == nil && strings.TrimSpace(cookieToken) != "" {
			tokenString = strings.TrimSpace(cookieToken)
		}

		if tokenString == "" {
			// Fallback: Check auth_token (used by Kite) if access_token is missing
			// Since we added 'sub' claim to Kite token, it can serve as a valid auth token
			if kiteToken, err := c.Cookie("auth_token"); err == nil && strings.TrimSpace(kiteToken) != "" {
				tokenString = strings.TrimSpace(kiteToken)
			}
		}

		if tokenString == "" {
			tokenString = extractBearerToken(c.GetHeader("Authorization"))
		}

		if tokenString == "" {
			if shouldRedirectForCAS(cfg, c.Request) {
				redirectToCASLogin(cfg, c)
				c.Abort()
				return
			}

			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "缺少有效的认证信息",
				"code":  "MISSING_AUTH",
			})
			c.Abort()
			return
		}

		claims, err := parseJWTClaims(cfg, tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "无效的登录状态",
				"code":  "INVALID_SESSION",
			})
			c.Abort()
			return
		}

		setUserClaims(c, claims)
		c.Next()
	}
}

// APITokenMiddleware 针对API访问的Token认证
func APITokenMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		expected := strings.TrimSpace(cfg.HolmesGPT.ProxyAuthToken)
		if expected == "" {
			c.Next()
			return
		}

		token := extractBearerToken(c.GetHeader("Authorization"))
		if token == "" {
			token = strings.TrimSpace(c.GetHeader("X-API-Token"))
		}
		if token == "" {
			if cookieToken, err := c.Cookie("api_token"); err == nil {
				token = strings.TrimSpace(cookieToken)
			}
		}

		if token == "" || token != expected {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "API token无效",
				"code":  "INVALID_API_TOKEN",
			})
			return
		}

		c.Next()
	}
}

func extractBearerToken(header string) string {
	if header == "" {
		return ""
	}
	lower := strings.ToLower(header)
	if !strings.HasPrefix(lower, "bearer ") {
		return ""
	}
	return strings.TrimSpace(header[7:])
}

func parseJWTClaims(cfg *config.Config, tokenString string) (jwt.MapClaims, error) {
	if tokenString == "" {
		return nil, errors.New("empty token")
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(cfg.JWTSecret), nil
	})
	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid claims")
	}

	return claims, nil
}

func setUserClaims(c *gin.Context, claims jwt.MapClaims) {
	if rawSub, ok := claims["sub"]; ok {
		if uid, err := models.ParseID(fmt.Sprint(rawSub)); err == nil {
			c.Set("user_id", uid)
		} else {
			c.Set("user_id", rawSub)
		}
	} else {
	}
	c.Set("user_email", claims["email"])
	c.Set("user_name", claims["name"])
	c.Set("user_roles", claims["roles"])

	// 设置username字段
	if username, ok := claims["username"]; ok {
		c.Set("username", username)
	}

	// 设置is_admin字段
	if isAdmin, ok := claims["is_admin"].(bool); ok {
		c.Set("is_admin", isAdmin)
	} else {
		c.Set("is_admin", false)
	}
}

func setDevAdmin(c *gin.Context) {
	c.Set("user_id", uint64(1))
	c.Set("user_email", "admin@local")
	c.Set("user_name", "Local Admin")
	c.Set("user_roles", []interface{}{"admin"})
	c.Set("username", "admin")
	c.Set("is_admin", true)
}

func shouldRedirectForCAS(cfg *config.Config, r *http.Request) bool {
	if cfg == nil || !cfg.CAS.Enabled {
		return false
	}

	if !strings.EqualFold(r.Method, http.MethodGet) {
		return false
	}

	accept := r.Header.Get("Accept")
	if accept != "" && !strings.Contains(accept, "text/html") {
		return false
	}

	return true
}

func redirectToCASLogin(cfg *config.Config, c *gin.Context) {
	baseURL := buildBaseURL(c.Request)
	serviceTarget := determineServiceTarget(cfg, c.Request)

	loginURL := fmt.Sprintf("%s/auth/cas/login?service=%s", baseURL, url.QueryEscape(serviceTarget))
	c.Redirect(http.StatusFound, loginURL)
}

func determineServiceTarget(cfg *config.Config, r *http.Request) string {
	fullURL := buildFullRequestURL(r)

	if strings.HasPrefix(r.URL.Path, "/api/") {
		if cfg != nil && strings.TrimSpace(cfg.CAS.RedirectURL) != "" {
			return cfg.CAS.RedirectURL
		}
		return buildBaseURL(r)
	}

	return fullURL
}

func buildBaseURL(r *http.Request) string {
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

func buildFullRequestURL(r *http.Request) string {
	base := buildBaseURL(r)
	return base + r.URL.RequestURI()
}

func isSecureRequest(r *http.Request) bool {
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		parts := strings.Split(proto, ",")
		return strings.EqualFold(strings.TrimSpace(parts[0]), "https")
	}
	return r.TLS != nil
}

// RequireRole 角色权限检查中间件（已废弃，保留兼容性）
func RequireRole(requiredRole string) gin.HandlerFunc {
	return func(c *gin.Context) {
		roles, exists := c.Get("user_roles")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "缺少角色信息",
				"code":  "MISSING_ROLES",
			})
			c.Abort()
			return
		}

		userRoles, ok := roles.([]interface{})
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "无效的角色格式",
				"code":  "INVALID_ROLES_FORMAT",
			})
			c.Abort()
			return
		}

		// 检查用户是否具有所需角色
		hasRole := false
		for _, role := range userRoles {
			if roleStr, ok := role.(string); ok && roleStr == requiredRole {
				hasRole = true
				break
			}
		}

		if !hasRole {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "权限不足",
				"code":  "INSUFFICIENT_PERMISSIONS",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireAdmin 管理员权限检查中间件
func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从JWT claims中获取is_admin字段
		isAdmin, exists := c.Get("is_admin")

		if !exists {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "缺少权限信息",
				"code":  "MISSING_PERMISSION",
			})
			c.Abort()
			return
		}

		// 检查是否为管理员
		admin, ok := isAdmin.(bool)
		if !ok || !admin {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "需要管理员权限",
				"code":  "ADMIN_REQUIRED",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

package middleware

import (
	"errors"
	"net/http"
	"strings"

	"robusta-web/backend/internal/config"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// AuthMiddleware JWT认证中间件
func AuthMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
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
		tokenString := ""

		if cookieToken, err := c.Cookie("access_token"); err == nil && strings.TrimSpace(cookieToken) != "" {
			tokenString = strings.TrimSpace(cookieToken)
		}

		if tokenString == "" {
			tokenString = extractBearerToken(c.GetHeader("Authorization"))
		}

		if tokenString == "" {
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
	c.Set("user_id", claims["sub"])
	c.Set("user_email", claims["email"])
	c.Set("user_name", claims["name"])
	c.Set("user_roles", claims["roles"])
}

// RequireRole 角色权限检查中间件
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

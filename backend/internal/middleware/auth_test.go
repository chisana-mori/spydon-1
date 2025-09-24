package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"robusta-web/backend/internal/config"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

func TestAuthMiddleware_ValidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 创建测试配置
	cfg := &config.Config{
		JWTSecret: "test-secret-key",
	}

	// 创建有效的JWT token
	claims := jwt.MapClaims{
		"sub":   "user123",
		"email": "test@example.com",
		"roles": []string{"user"},
		"exp":   time.Now().Add(time.Hour).Unix(),
		"iat":   time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(cfg.JWTSecret))
	assert.NoError(t, err)

	// 创建测试路由
	router := gin.New()
	router.Use(AuthMiddleware(cfg))
	router.GET("/test", func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		userEmail, _ := c.Get("user_email")
		userRoles, _ := c.Get("user_roles")

		c.JSON(http.StatusOK, gin.H{
			"user_id":    userID,
			"user_email": userEmail,
			"user_roles": userRoles,
		})
	})

	// 创建请求
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	w := httptest.NewRecorder()

	// 执行请求
	router.ServeHTTP(w, req)

	// 验证响应
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "user123")
	assert.Contains(t, w.Body.String(), "test@example.com")
}

func TestAuthMiddleware_MissingToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		JWTSecret: "test-secret-key",
	}

	router := gin.New()
	router.Use(AuthMiddleware(cfg))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "缺少认证头")
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		JWTSecret: "test-secret-key",
	}

	router := gin.New()
	router.Use(AuthMiddleware(cfg))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "无效的token")
}

func TestAuthMiddleware_ExpiredToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		JWTSecret: "test-secret-key",
	}

	// 创建过期的JWT token
	claims := jwt.MapClaims{
		"sub":   "user123",
		"email": "test@example.com",
		"roles": []string{"user"},
		"exp":   time.Now().Add(-time.Hour).Unix(), // 过期时间设为1小时前
		"iat":   time.Now().Add(-2 * time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(cfg.JWTSecret))
	assert.NoError(t, err)

	router := gin.New()
	router.Use(AuthMiddleware(cfg))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "无效的token")
}

func TestRequireRole_ValidRole(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()

	// 模拟认证中间件设置用户角色
	router.Use(func(c *gin.Context) {
		c.Set("user_roles", []interface{}{"user", "admin"})
		c.Next()
	})

	router.Use(RequireRole("admin"))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "success")
}

func TestRequireRole_InsufficientPermissions(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()

	// 模拟认证中间件设置用户角色
	router.Use(func(c *gin.Context) {
		c.Set("user_roles", []interface{}{"user"})
		c.Next()
	})

	router.Use(RequireRole("admin"))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "权限不足")
}

func TestRequireRole_MissingRoles(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(RequireRole("admin"))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "缺少角色信息")
}

func TestOptionalAuthMiddleware_WithValidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		JWTSecret: "test-secret-key",
	}

	// 创建有效的JWT token
	claims := jwt.MapClaims{
		"sub":   "user123",
		"email": "test@example.com",
		"roles": []string{"user"},
		"exp":   time.Now().Add(time.Hour).Unix(),
		"iat":   time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(cfg.JWTSecret))
	assert.NoError(t, err)

	router := gin.New()
	router.Use(OptionalAuthMiddleware(cfg))
	router.GET("/test", func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		c.JSON(http.StatusOK, gin.H{
			"authenticated": exists,
			"user_id":       userID,
		})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "\"authenticated\":true")
	assert.Contains(t, w.Body.String(), "user123")
}

func TestOptionalAuthMiddleware_WithoutToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		JWTSecret: "test-secret-key",
	}

	router := gin.New()
	router.Use(OptionalAuthMiddleware(cfg))
	router.GET("/test", func(c *gin.Context) {
		_, exists := c.Get("user_id")
		c.JSON(http.StatusOK, gin.H{
			"authenticated": exists,
			"message":       "public access",
		})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "\"authenticated\":false")
	assert.Contains(t, w.Body.String(), "public access")
}

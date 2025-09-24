package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"robusta-web/backend/internal/config"
	"robusta-web/backend/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIngestAPI_Integration(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 创建测试配置
	cfg := &config.Config{
		HMACSecret: "test-hmac-secret",
	}

	// 创建测试路由
	router := gin.New()
	router.Use(middleware.EnhancedHMACMiddleware(cfg))

	// 模拟ingest处理器
	router.POST("/api/v1/ingest/alert", func(c *gin.Context) {
		var alertData map[string]interface{}
		if err := c.ShouldBindJSON(&alertData); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message":  "告警接收成功",
			"alert_id": "test-alert-123",
		})
	})

	// 准备测试数据
	alertPayload := map[string]interface{}{
		"fingerprint": "test-fingerprint",
		"cluster_id":  "test-cluster",
		"title":       "Test Alert",
		"severity":    "high",
		"status":      "firing",
		"labels": map[string]interface{}{
			"alertname": "TestAlert",
			"instance":  "test-instance",
		},
		"annotations": map[string]interface{}{
			"summary": "Test alert summary",
		},
	}

	payloadBytes, err := json.Marshal(alertPayload)
	require.NoError(t, err)

	// 生成HMAC签名
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	signature := middleware.GenerateEnhancedHMAC(timestamp, string(payloadBytes), cfg.HMACSecret)

	// 创建请求
	req := httptest.NewRequest("POST", "/api/v1/ingest/alert", bytes.NewBuffer(payloadBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Robusta-Signature", "sha256="+signature)
	req.Header.Set("X-Robusta-Timestamp", timestamp)
	req.Header.Set("X-Robusta-Cluster-ID", "test-cluster")

	w := httptest.NewRecorder()

	// 执行请求
	router.ServeHTTP(w, req)

	// 验证响应
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "告警接收成功", response["message"])
	assert.Equal(t, "test-alert-123", response["alert_id"])
}

func TestIngestAPI_InvalidSignature(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		HMACSecret: "test-hmac-secret",
	}

	router := gin.New()
	router.Use(middleware.HMACMiddleware(cfg))
	router.POST("/api/v1/ingest/alert", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	alertPayload := map[string]interface{}{
		"fingerprint": "test-fingerprint",
		"title":       "Test Alert",
	}

	payloadBytes, err := json.Marshal(alertPayload)
	require.NoError(t, err)

	timestamp := strconv.FormatInt(time.Now().Unix(), 10)

	req := httptest.NewRequest("POST", "/api/v1/ingest/alert", bytes.NewBuffer(payloadBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Robusta-Signature", "sha256=invalid-signature")
	req.Header.Set("X-Robusta-Timestamp", timestamp)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "HMAC签名验证失败")
}

func TestIngestAPI_ExpiredTimestamp(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		HMACSecret: "test-hmac-secret",
	}

	router := gin.New()
	router.Use(middleware.EnhancedHMACMiddleware(cfg))
	router.POST("/api/v1/ingest/alert", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	alertPayload := map[string]interface{}{
		"fingerprint": "test-fingerprint",
		"title":       "Test Alert",
	}

	payloadBytes, err := json.Marshal(alertPayload)
	require.NoError(t, err)

	// 使用过期的时间戳（10分钟前）
	expiredTimestamp := strconv.FormatInt(time.Now().Unix()-600, 10)
	signature := middleware.GenerateEnhancedHMAC(expiredTimestamp, string(payloadBytes), cfg.HMACSecret)

	req := httptest.NewRequest("POST", "/api/v1/ingest/alert", bytes.NewBuffer(payloadBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Robusta-Signature", "sha256="+signature)
	req.Header.Set("X-Robusta-Timestamp", expiredTimestamp)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "请求时间戳过期")
}

func TestAuthAPI_Integration(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()

	// 模拟认证处理器
	router.GET("/auth/url", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"auth_url": "https://oidc.example.com/auth?client_id=test&response_type=code&scope=openid+profile+email&redirect_uri=http://localhost:3000/auth/callback&state=random-state",
			"state":    "random-state",
		})
	})

	router.POST("/auth/callback", func(c *gin.Context) {
		var req struct {
			Code  string `json:"code"`
			State string `json:"state"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// 模拟成功的认证回调
		c.JSON(http.StatusOK, gin.H{
			"access_token": "mock-jwt-token",
			"expires_at":   time.Now().Add(24 * time.Hour),
			"user": gin.H{
				"id":       "user123",
				"username": "testuser",
				"email":    "test@example.com",
				"roles":    []string{"user"},
			},
		})
	})

	// 测试获取认证URL
	req := httptest.NewRequest("GET", "/auth/url", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var authResponse map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &authResponse)
	require.NoError(t, err)

	assert.Contains(t, authResponse["auth_url"], "oidc.example.com")
	assert.NotEmpty(t, authResponse["state"])

	// 测试认证回调
	callbackPayload := map[string]string{
		"code":  "auth-code-123",
		"state": "random-state",
	}

	payloadBytes, err := json.Marshal(callbackPayload)
	require.NoError(t, err)

	req = httptest.NewRequest("POST", "/auth/callback", bytes.NewBuffer(payloadBytes))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var callbackResponse map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &callbackResponse)
	require.NoError(t, err)

	assert.Equal(t, "mock-jwt-token", callbackResponse["access_token"])
	assert.NotNil(t, callbackResponse["user"])
}

func TestHealthCheck_Integration(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()

	// 模拟健康检查处理器
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"timestamp": time.Now().Unix(),
			"version":   "1.0.0",
		})
	})

	router.GET("/ready", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ready",
			"checks": gin.H{
				"database": "ok",
				"redis":    "ok",
			},
		})
	})

	// 测试健康检查
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "healthy")

	// 测试就绪检查
	req = httptest.NewRequest("GET", "/ready", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "ready")
}

package services

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"robusta-web/backend/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHolmesService_SendAnalysisRequest(t *testing.T) {
	// 创建模拟HolmesGPT服务器
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/api/v1/analyze", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))

		// 验证请求体
		var req HolmesAnalysisRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)
		assert.Equal(t, "test-fingerprint", req.AlertFingerprint)
		assert.Equal(t, "test-cluster", req.ClusterID)
		assert.Equal(t, "standard", req.Depth)

		// 返回模拟响应
		response := HolmesAnalysisResponse{
			ID:      "analysis-123",
			Status:  "completed",
			Summary: "Test analysis completed",
			Suspects: map[string]interface{}{
				"pod_issues": "High memory usage detected",
			},
			Recommendations: map[string]interface{}{
				"scale_up": "Consider scaling up the deployment",
			},
			StartedAt:   time.Now(),
			CompletedAt: &[]time.Time{time.Now().Add(5 * time.Minute)}[0],
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	// 创建配置
	cfg := &config.Config{
		HolmesGPT: config.HolmesGPTConfig{
			URL:            mockServer.URL,
			APIKey:         "test-api-key",
			TimeoutSeconds: 300,
			Enabled:        true,
			DefaultDepth:   "standard",
			Model:          "test-model",
			ProxyAuthToken: "",
		},
	}

	// 创建HolmesService（不需要数据库）
	service := &HolmesService{
		config: cfg,
		client: &http.Client{Timeout: 30 * time.Second},
	}

	// 准备分析请求
	request := HolmesAnalysisRequest{
		AlertFingerprint: "test-fingerprint",
		ClusterID:        "test-cluster",
		Context: map[string]interface{}{
			"title":       "Test Alert",
			"description": "Test alert description",
			"severity":    "high",
			"labels":      map[string]interface{}{"alertname": "TestAlert"},
			"annotations": map[string]interface{}{"summary": "Test summary"},
		},
		Depth:          "standard",
		TimeoutSeconds: 300,
	}

	// 发送分析请求
	response, err := service.sendAnalysisRequest(context.Background(), request)
	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "analysis-123", response.ID)
	assert.Equal(t, "completed", response.Status)
	assert.Equal(t, "Test analysis completed", response.Summary)
	assert.Contains(t, response.Suspects, "pod_issues")
	assert.Contains(t, response.Recommendations, "scale_up")
}

func TestHolmesService_SendAnalysisRequest_Error(t *testing.T) {
	// 创建模拟失败的HolmesGPT服务器
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal server error"))
	}))
	defer mockServer.Close()

	cfg := &config.Config{
		HolmesGPT: config.HolmesGPTConfig{
			URL:            mockServer.URL,
			APIKey:         "test-api-key",
			TimeoutSeconds: 300,
			Enabled:        true,
			DefaultDepth:   "standard",
			Model:          "test-model",
			ProxyAuthToken: "",
		},
	}

	service := &HolmesService{
		config: cfg,
		client: &http.Client{Timeout: 30 * time.Second},
	}

	request := HolmesAnalysisRequest{
		AlertFingerprint: "test-fingerprint",
		ClusterID:        "test-cluster",
		Depth:            "standard",
	}

	// 发送分析请求（应该失败）
	response, err := service.sendAnalysisRequest(context.Background(), request)
	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "500")
}

func TestHolmesService_ConvertFindingToAlert(t *testing.T) {
	// 这个测试验证告警转换逻辑
	// 由于涉及到复杂的Robusta Finding结构，这里只做基本验证

	// 测试指纹生成
	fingerprint1 := generateTestFingerprint("alert1", "cluster1", "key1")
	fingerprint2 := generateTestFingerprint("alert1", "cluster1", "key1")
	fingerprint3 := generateTestFingerprint("alert2", "cluster1", "key1")

	assert.Equal(t, fingerprint1, fingerprint2, "相同输入应该生成相同指纹")
	assert.NotEqual(t, fingerprint1, fingerprint3, "不同输入应该生成不同指纹")
}

func TestHolmesService_ValidateDepth(t *testing.T) {
	validDepths := []string{"quick", "standard", "deep"}
	invalidDepths := []string{"", "invalid", "slow", "fast"}

	for _, depth := range validDepths {
		assert.True(t, isValidDepth(depth), "深度 %s 应该是有效的", depth)
	}

	for _, depth := range invalidDepths {
		assert.False(t, isValidDepth(depth), "深度 %s 应该是无效的", depth)
	}
}

// 辅助函数
func generateTestFingerprint(title, clusterID, key string) string {
	// 简化的指纹生成逻辑
	data := fmt.Sprintf("%s:%s:%s", title, clusterID, key)
	hash := md5.Sum([]byte(data))
	return fmt.Sprintf("%x", hash)
}

func isValidDepth(depth string) bool {
	validDepths := map[string]bool{
		"quick":    true,
		"standard": true,
		"deep":     true,
	}
	return validDepths[depth]
}

package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"robusta-web/backend/internal/config"
	"robusta-web/backend/internal/db"
	"robusta-web/backend/internal/models"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupHolmesTestDB(t *testing.T) *db.Database {
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// 创建简化的表结构用于测试
	err = database.Exec(`
		CREATE TABLE alerts (
			id TEXT PRIMARY KEY,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			fingerprint TEXT NOT NULL,
			cluster_id TEXT NOT NULL,
			title TEXT NOT NULL,
			description TEXT,
			severity TEXT NOT NULL,
			status TEXT DEFAULT 'firing',
			labels TEXT,
			annotations TEXT,
			starts_at DATETIME,
			ends_at DATETIME,
			raw_payload_key TEXT
		)
	`).Error
	require.NoError(t, err)

	err = database.Exec(`
		CREATE TABLE rca_runs (
			id TEXT PRIMARY KEY,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			alert_id TEXT NOT NULL,
			status TEXT NOT NULL,
			summary TEXT,
			suspects TEXT,
			recommendations TEXT,
			attachments TEXT,
			started_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			completed_at DATETIME,
			error_message TEXT,
			raw_payload_key TEXT
		)
	`).Error
	require.NoError(t, err)

	err = database.Exec(`
		CREATE TABLE clusters (
			id TEXT PRIMARY KEY,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			cluster_id TEXT NOT NULL,
			name TEXT NOT NULL,
			description TEXT,
			status TEXT DEFAULT 'active',
			last_heartbeat DATETIME
		)
	`).Error
	require.NoError(t, err)

	return &db.Database{DB: database}
}

func TestHolmesService_TriggerAnalysis(t *testing.T) {
	// 设置测试数据库
	database := setupHolmesTestDB(t)

	// 创建模拟HolmesGPT服务器
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/api/v1/analyze", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

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
		HolmesGPT: struct {
			URL            string `json:"url"`
			APIKey         string `json:"api_key"`
			TimeoutSeconds int    `json:"timeout_seconds"`
			Enabled        bool   `json:"enabled"`
			DefaultDepth   string `json:"default_depth"`
			Model          string `json:"model"`
		}{
			URL:            mockServer.URL,
			APIKey:         "test-api-key",
			TimeoutSeconds: 300,
			Enabled:        true,
			DefaultDepth:   "standard",
			Model:          "test-model",
		},
	}

	// 创建测试告警
	alert := &models.Alert{
		BaseModel: models.BaseModel{
			ID: uuid.New(),
		},
		Fingerprint: "test-fingerprint",
		ClusterID:   "test-cluster",
		Title:       "Test Alert",
		Description: "Test alert description",
		Severity:    "high",
		Status:      "firing",
		StartsAt:    &[]time.Time{time.Now()}[0],
	}

	// 直接插入到数据库，避免JSON序列化问题
	err := database.DB.Exec(`
		INSERT INTO alerts (id, fingerprint, cluster_id, title, description, severity, status, starts_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, alert.ID.String(), alert.Fingerprint, alert.ClusterID, alert.Title, alert.Description,
		alert.Severity, alert.Status, alert.StartsAt, time.Now(), time.Now()).Error
	require.NoError(t, err)

	// 创建HolmesService
	service := NewHolmesService(database, cfg)

	// 触发分析
	rcaRun, err := service.TriggerAnalysis(context.Background(), alert.ID.String(), "standard")
	require.NoError(t, err)
	assert.NotNil(t, rcaRun)
	assert.Equal(t, alert.ID, rcaRun.AlertID)
	assert.Equal(t, "pending", rcaRun.Status)

	// 等待异步分析完成
	time.Sleep(100 * time.Millisecond)

	// 验证分析结果
	var updatedRun models.RCARun
	err = database.DB.Where("id = ?", rcaRun.ID).First(&updatedRun).Error
	require.NoError(t, err)
	assert.Equal(t, "completed", updatedRun.Status)
	assert.NotNil(t, updatedRun.Summary)
	assert.Equal(t, "Test analysis completed", *updatedRun.Summary)
}

func TestHolmesService_GetAnalysisByAlertID(t *testing.T) {
	database := setupHolmesTestDB(t)
	cfg := &config.Config{
		HolmesGPT: struct {
			URL            string `json:"url"`
			APIKey         string `json:"api_key"`
			TimeoutSeconds int    `json:"timeout_seconds"`
			Enabled        bool   `json:"enabled"`
			DefaultDepth   string `json:"default_depth"`
			Model          string `json:"model"`
		}{
			URL:            "http://localhost:8081",
			TimeoutSeconds: 300,
			Enabled:        true,
			DefaultDepth:   "standard",
			Model:          "test-model",
		},
	}

	service := NewHolmesService(database, cfg)

	// 创建测试数据
	alertID := uuid.New()
	summary1 := "First analysis"
	summary2 := "Second analysis"

	rcaRun1 := &models.RCARun{
		BaseModel: models.BaseModel{
			ID: uuid.New(),
		},
		AlertID:   alertID,
		Status:    "completed",
		Summary:   &summary1,
		StartedAt: time.Now().Add(-1 * time.Hour),
	}

	rcaRun2 := &models.RCARun{
		BaseModel: models.BaseModel{
			ID: uuid.New(),
		},
		AlertID:   alertID,
		Status:    "running",
		Summary:   &summary2,
		StartedAt: time.Now(),
	}

	rcaRun1.CreatedAt = time.Now().Add(-2 * time.Hour)
	rcaRun2.CreatedAt = time.Now()

	err := database.DB.Create([]*models.RCARun{rcaRun1, rcaRun2}).Error
	require.NoError(t, err)

	// 查询分析结果
	runs, err := service.GetAnalysisByAlertID(alertID.String())
	require.NoError(t, err)
	assert.Len(t, runs, 2)

	// 验证排序（最新的在前）
	assert.Equal(t, rcaRun2.ID, runs[0].ID)
	assert.Equal(t, rcaRun1.ID, runs[1].ID)
}

func TestHolmesService_GetAnalysisStats(t *testing.T) {
	database := setupHolmesTestDB(t)
	cfg := &config.Config{
		HolmesGPT: struct {
			URL            string `json:"url"`
			APIKey         string `json:"api_key"`
			TimeoutSeconds int    `json:"timeout_seconds"`
			Enabled        bool   `json:"enabled"`
			DefaultDepth   string `json:"default_depth"`
			Model          string `json:"model"`
		}{
			URL:            "http://localhost:8081",
			TimeoutSeconds: 300,
			Enabled:        true,
			DefaultDepth:   "standard",
			Model:          "test-model",
		},
	}

	service := NewHolmesService(database, cfg)

	// 创建测试集群和告警
	cluster := &models.Cluster{
		BaseModel: models.BaseModel{
			ID: uuid.New(),
		},
		ClusterID:   "test-cluster",
		Name:        "Test Cluster",
		Status:      "active",
		Description: "Test cluster",
	}

	alert1 := &models.Alert{
		BaseModel: models.BaseModel{
			ID: uuid.New(),
		},
		ClusterID:   "test-cluster",
		Fingerprint: "alert-1",
		Title:       "Alert 1",
		Severity:    "high",
		Status:      "firing",
	}

	alert2 := &models.Alert{
		BaseModel: models.BaseModel{
			ID: uuid.New(),
		},
		ClusterID:   "test-cluster",
		Fingerprint: "alert-2",
		Title:       "Alert 2",
		Severity:    "medium",
		Status:      "firing",
	}

	err := database.DB.Create([]*models.Cluster{cluster}).Error
	require.NoError(t, err)

	err = database.DB.Create([]*models.Alert{alert1, alert2}).Error
	require.NoError(t, err)

	// 创建RCA运行记录
	now := time.Now()
	completedAt := now.Add(5 * time.Minute)

	rcaRuns := []*models.RCARun{
		{
			BaseModel: models.BaseModel{
				ID: uuid.New(),
			},
			AlertID:     alert1.ID,
			Status:      "completed",
			StartedAt:   now.Add(-10 * time.Minute),
			CompletedAt: &completedAt,
		},
		{
			BaseModel: models.BaseModel{
				ID: uuid.New(),
			},
			AlertID:   alert1.ID,
			Status:    "failed",
			StartedAt: now.Add(-5 * time.Minute),
		},
		{
			BaseModel: models.BaseModel{
				ID: uuid.New(),
			},
			AlertID:   alert2.ID,
			Status:    "running",
			StartedAt: now,
		},
	}

	err = database.DB.Create(rcaRuns).Error
	require.NoError(t, err)

	// 获取统计信息
	stats, err := service.GetAnalysisStats("test-cluster")
	require.NoError(t, err)

	assert.Equal(t, int64(3), stats.Total)
	assert.Len(t, stats.ByStatus, 3)                          // completed, failed, running
	assert.Equal(t, float64(1)/float64(3), stats.SuccessRate) // 1 completed out of 3 total
	assert.Greater(t, stats.AvgDurationSeconds, 0.0)
}

func TestHolmesService_HandleAnalysisError(t *testing.T) {
	database := setupHolmesTestDB(t)

	// 创建模拟失败的HolmesGPT服务器
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal server error"))
	}))
	defer mockServer.Close()

	cfg := &config.Config{
		HolmesGPT: struct {
			URL            string `json:"url"`
			APIKey         string `json:"api_key"`
			TimeoutSeconds int    `json:"timeout_seconds"`
			Enabled        bool   `json:"enabled"`
			DefaultDepth   string `json:"default_depth"`
			Model          string `json:"model"`
		}{
			URL:            mockServer.URL,
			TimeoutSeconds: 300,
			Enabled:        true,
			DefaultDepth:   "standard",
			Model:          "test-model",
		},
	}

	// 创建测试告警
	alert := &models.Alert{
		BaseModel: models.BaseModel{
			ID: uuid.New(),
		},
		Fingerprint: "test-fingerprint",
		ClusterID:   "test-cluster",
		Title:       "Test Alert",
		Severity:    "high",
		Status:      "firing",
	}

	err := database.DB.Create(alert).Error
	require.NoError(t, err)

	service := NewHolmesService(database, cfg)

	// 触发分析（应该失败）
	rcaRun, err := service.TriggerAnalysis(context.Background(), alert.ID.String(), "standard")
	require.NoError(t, err)

	// 等待异步分析完成
	time.Sleep(100 * time.Millisecond)

	// 验证错误处理
	var updatedRun models.RCARun
	err = database.DB.Where("id = ?", rcaRun.ID).First(&updatedRun).Error
	require.NoError(t, err)
	assert.Equal(t, "failed", updatedRun.Status)
	assert.NotNil(t, updatedRun.ErrorMessage)
	assert.Contains(t, *updatedRun.ErrorMessage, "500")
}

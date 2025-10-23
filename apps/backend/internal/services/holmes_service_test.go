package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"robusta-web/backend/internal/config"
	"robusta-web/backend/internal/db"
	"robusta-web/backend/internal/models"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
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

type mockPayloadStorage struct {
	data    map[string][]byte
	lastKey string
}

func newMockPayloadStorage() *mockPayloadStorage {
	return &mockPayloadStorage{
		data: make(map[string][]byte),
	}
}

func (m *mockPayloadStorage) Save(ctx context.Context, prefix string, data []byte, contentType string) (string, error) {
	cleanPrefix := strings.Trim(prefix, "/")
	if cleanPrefix == "" {
		cleanPrefix = "rca-results"
	}
	key := fmt.Sprintf("%s/%s", cleanPrefix, uuid.NewString())
	m.data[key] = append([]byte(nil), data...)
	m.lastKey = key
	return key, nil
}

func (m *mockPayloadStorage) Get(ctx context.Context, key string) ([]byte, error) {
	if data, ok := m.data[key]; ok {
		return append([]byte(nil), data...), nil
	}
	return nil, fmt.Errorf("从MinIO获取对象失败: %w", minio.ErrorResponse{
		Code:       "NoSuchKey",
		StatusCode: http.StatusNotFound,
	})
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
	storage := newMockPayloadStorage()
	service := NewHolmesService(database, cfg, storage)

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
	assert.NotEmpty(t, updatedRun.RawPayloadKey)
	_, exists := storage.data[updatedRun.RawPayloadKey]
	assert.True(t, exists)
	var cached RCACachedResult
	err = json.Unmarshal(storage.data[updatedRun.RawPayloadKey], &cached)
	require.NoError(t, err)
	assert.Equal(t, updatedRun.ID.String(), cached.RunID)
	assert.Equal(t, alert.ID.String(), cached.AlertID)
	assert.NotNil(t, cached.Analysis)
	assert.Equal(t, "completed", cached.Analysis.Status)
	assert.Equal(t, updatedRun.RawPayloadKey, cached.StorageKey)
}

func TestHolmesService_GetAnalysisByAlertID(t *testing.T) {
	database := setupHolmesTestDB(t)
	cfg := &config.Config{
		HolmesGPT: config.HolmesGPTConfig{
			URL:            "http://localhost:8081",
			TimeoutSeconds: 300,
			Enabled:        true,
			DefaultDepth:   "standard",
			Model:          "test-model",
			ProxyAuthToken: "",
		},
	}

	storage := newMockPayloadStorage()
	service := NewHolmesService(database, cfg, storage)

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
		HolmesGPT: config.HolmesGPTConfig{
			URL:            "http://localhost:8081",
			TimeoutSeconds: 300,
			Enabled:        true,
			DefaultDepth:   "standard",
			Model:          "test-model",
			ProxyAuthToken: "",
		},
	}

	storage := newMockPayloadStorage()
	service := NewHolmesService(database, cfg, storage)

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
		HolmesGPT: config.HolmesGPTConfig{
			URL:            mockServer.URL,
			TimeoutSeconds: 300,
			Enabled:        true,
			DefaultDepth:   "standard",
			Model:          "test-model",
			ProxyAuthToken: "",
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

	storage := newMockPayloadStorage()
	service := NewHolmesService(database, cfg, storage)

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

func TestHolmesService_GetCachedResult(t *testing.T) {
	database := setupHolmesTestDB(t)
	storage := newMockPayloadStorage()

	cfg := &config.Config{
		HolmesGPT: config.HolmesGPTConfig{
			URL:            "http://localhost:8081",
			TimeoutSeconds: 300,
			Enabled:        true,
			DefaultDepth:   "standard",
			Model:          "test-model",
			ProxyAuthToken: "",
		},
	}

	service := NewHolmesService(database, cfg, storage)

	alertID := uuid.New()
	runID := uuid.New()
	now := time.Now()

	alert := &models.Alert{
		BaseModel: models.BaseModel{
			ID: alertID,
		},
		Fingerprint: "cached-alert",
		ClusterID:   "cluster-1",
		Title:       "Cached Alert",
		Severity:    "high",
		Status:      "firing",
	}
	require.NoError(t, database.DB.Create(alert).Error)

	cached := RCACachedResult{
		Version:  "v1",
		RunID:    runID.String(),
		AlertID:  alertID.String(),
		CachedAt: now,
		Depth:    "standard",
		Analysis: &HolmesAnalysisResponse{
			ID:          "analysis-cached",
			Status:      "completed",
			Summary:     "Cached summary",
			StartedAt:   now.Add(-5 * time.Minute),
			CompletedAt: &now,
		},
	}

	payload, err := json.Marshal(cached)
	require.NoError(t, err)
	key, err := storage.Save(context.Background(), fmt.Sprintf("rca-results/%s", alertID.String()), payload, "application/json")
	require.NoError(t, err)

	run := &models.RCARun{
		BaseModel: models.BaseModel{
			ID:        runID,
			CreatedAt: now,
			UpdatedAt: now,
		},
		AlertID:       alertID,
		Status:        "completed",
		StartedAt:     now.Add(-10 * time.Minute),
		CompletedAt:   &now,
		RawPayloadKey: key,
	}
	require.NoError(t, database.DB.Create(run).Error)

	result, err := service.GetCachedResult(context.Background(), alertID.String())
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, cached.RunID, result.RunID)
	if assert.NotNil(t, result.Analysis) {
		assert.Equal(t, cached.Analysis.ID, result.Analysis.ID)
		assert.Equal(t, cached.Analysis.Summary, result.Analysis.Summary)
	}
	assert.Equal(t, run.RawPayloadKey, result.StorageKey)

	// 通过运行ID获取缓存
	resultByRun, err := service.GetCachedResultByRunID(context.Background(), runID.String())
	require.NoError(t, err)
	require.NotNil(t, resultByRun)
	assert.Equal(t, cached.RunID, resultByRun.RunID)
	assert.Equal(t, run.RawPayloadKey, resultByRun.StorageKey)
}

func TestHolmesService_FinalizeStreamRun_Success(t *testing.T) {
	database := setupHolmesTestDB(t)
	storage := newMockPayloadStorage()

	cfg := &config.Config{
		HolmesGPT: config.HolmesGPTConfig{
			URL:            "http://localhost:8081",
			TimeoutSeconds: 300,
			Enabled:        true,
			DefaultDepth:   "standard",
			Model:          "test-model",
			ProxyAuthToken: "",
		},
	}

	service := NewHolmesService(database, cfg, storage)

	alert := &models.Alert{
		BaseModel: models.BaseModel{
			ID: uuid.New(),
		},
		Fingerprint: "stream-alert",
		ClusterID:   "cluster-stream",
		Title:       "Stream Alert",
		Severity:    "high",
		Status:      "firing",
	}
	require.NoError(t, database.DB.Create(alert).Error)

	run, err := service.StartStreamRun(alert.ID.String())
	require.NoError(t, err)
	require.NotNil(t, run)

	chunks := []string{
		"data: {\"type\":\"analysis\",\"data\":{\"step\":1}}\n\n",
		"data: {\"type\":\"complete\",\"data\":{}}\n\n",
	}

	metadata := map[string]interface{}{
		"summary": "最终结论",
	}

	service.FinalizeStreamRun(context.Background(), run, "standard", chunks, metadata, nil)

	var storedRun models.RCARun
	err = database.DB.Where("id = ?", run.ID).First(&storedRun).Error
	require.NoError(t, err)
	assert.Equal(t, "completed", storedRun.Status)
	assert.NotNil(t, storedRun.CompletedAt)
	assert.Equal(t, "最终结论", func() string {
		if storedRun.Summary != nil {
			return *storedRun.Summary
		}
		return ""
	}())
	assert.NotEmpty(t, storedRun.RawPayloadKey)

	data, ok := storage.data[storedRun.RawPayloadKey]
	assert.True(t, ok)

	var cached RCACachedResult
	err = json.Unmarshal(data, &cached)
	require.NoError(t, err)
	assert.Equal(t, run.ID.String(), cached.RunID)
	assert.Equal(t, alert.ID.String(), cached.AlertID)
	assert.ElementsMatch(t, chunks, cached.StreamChunks)
	assert.Equal(t, "最终结论", cached.Metadata["summary"])
}

func TestHolmesService_FinalizeStreamRun_Error(t *testing.T) {
	database := setupHolmesTestDB(t)
	storage := newMockPayloadStorage()

	cfg := &config.Config{
		HolmesGPT: config.HolmesGPTConfig{
			URL:            "http://localhost:8081",
			TimeoutSeconds: 300,
			Enabled:        true,
			DefaultDepth:   "standard",
			Model:          "test-model",
			ProxyAuthToken: "",
		},
	}

	service := NewHolmesService(database, cfg, storage)

	alert := &models.Alert{
		BaseModel: models.BaseModel{
			ID: uuid.New(),
		},
		Fingerprint: "stream-alert-err",
		ClusterID:   "cluster-stream",
		Title:       "Stream Alert Err",
		Severity:    "high",
		Status:      "firing",
	}
	require.NoError(t, database.DB.Create(alert).Error)

	run, err := service.StartStreamRun(alert.ID.String())
	require.NoError(t, err)

	streamErr := fmt.Errorf("stream interrupted")
	service.FinalizeStreamRun(context.Background(), run, "standard", nil, nil, streamErr)

	var storedRun models.RCARun
	err = database.DB.Where("id = ?", run.ID).First(&storedRun).Error
	require.NoError(t, err)
	assert.Equal(t, "failed", storedRun.Status)
	assert.NotNil(t, storedRun.CompletedAt)
	assert.NotNil(t, storedRun.ErrorMessage)
	assert.Contains(t, *storedRun.ErrorMessage, "stream interrupted")
	_, ok := storage.data[storage.lastKey]
	assert.False(t, ok)
}

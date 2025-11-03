package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"robusta-web/backend/internal/db"
	"robusta-web/backend/internal/middleware"
	"robusta-web/backend/internal/models"
	"robusta-web/backend/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// 简化的测试模型，兼容SQLite
type TestCluster struct {
	ID            string `gorm:"primaryKey"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
	DeletedAt     gorm.DeletedAt `gorm:"index"`
	ClusterID     string         `gorm:"uniqueIndex;not null"`
	Name          string         `gorm:"not null"`
	Description   string
	Status        string `gorm:"default:active"`
	LastHeartbeat *time.Time
}

func (TestCluster) TableName() string { return "clusters" }

type TestAlert struct {
	ID            string `gorm:"primaryKey"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
	DeletedAt     gorm.DeletedAt `gorm:"index"`
	Fingerprint   string         `gorm:"not null"`
	ClusterID     string         `gorm:"not null"`
	Title         string         `gorm:"not null"`
	Description   string
	Severity      string `gorm:"not null"`
	Status        string `gorm:"default:firing"`
	Labels        []byte
	Annotations   []byte
	StartsAt      *time.Time
	EndsAt        *time.Time
	RawPayloadKey string
}

func (TestAlert) TableName() string { return "alerts" }

type TestRCARun struct {
	ID              string `gorm:"primaryKey"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
	DeletedAt       gorm.DeletedAt `gorm:"index"`
	AlertID         string         `gorm:"not null"`
	Status          string         `gorm:"not null"`
	Summary         string
	Suspects        []byte
	Recommendations []byte
	Attachments     []byte
	StartedAt       time.Time
	CompletedAt     *time.Time
	ErrorMessage    string
	RawPayloadKey   string
}

func (TestRCARun) TableName() string { return "rca_runs" }

type fakeStorage struct {
	data map[string][]byte
}

func newFakeStorage() *fakeStorage {
	return &fakeStorage{data: make(map[string][]byte)}
}

func (f *fakeStorage) Save(ctx context.Context, prefix string, data []byte, contentType string) (string, error) {
	key := "test-object-key"
	f.data[key] = append([]byte(nil), data...)
	return key, nil
}

func (f *fakeStorage) Get(ctx context.Context, key string) ([]byte, error) {
	return append([]byte(nil), f.data[key]...), nil
}

func (f *fakeStorage) PutAt(ctx context.Context, key string, data []byte, contentType string) error {
	f.data[key] = append([]byte(nil), data...)
	return nil
}

func (f *fakeStorage) PresignPut(ctx context.Context, key string, ttl time.Duration, contentType string) (string, error) {
	return "", nil
}

func setupTestDB() *db.Database {
	// 使用内存SQLite数据库进行测试
	gormDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	database := &db.Database{DB: gormDB}

	// 使用简化的测试模型
	err = gormDB.AutoMigrate(
		&TestCluster{},
		&TestAlert{},
		&TestRCARun{},
	)
	if err != nil {
		panic("failed to migrate database")
	}

	return database
}

func setupTestHandler() (*IngestHandler, *gin.Engine, *db.Database) {
	database := setupTestDB()

	alertService := services.NewAlertService(database)
	rcaService := services.NewRCAService(database)
	storage := newFakeStorage()
	handler := NewIngestHandler(&IngestHandlerConfig{
		AlertService:   alertService,
		RCAService:     rcaService,
		ClusterService: nil, // 测试中不需要
		AuditService:   nil, // 测试中不需要
		StorageService: storage,
	})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.ErrorHandler())

	return handler, router, database
}

func TestIngestAlert(t *testing.T) {
	handler, router, _ := setupTestHandler()
	router.POST("/ingest/alert", handler.IngestAlert)

	tests := []struct {
		name           string
		payload        interface{}
		expectedStatus int
		expectedError  string
	}{
		{
			name: "valid alert",
			payload: IngestAlertRequest{
				Fingerprint: "test-fingerprint-1",
				ClusterID:   "test-cluster",
				Title:       "Test Alert",
				Description: "This is a test alert",
				Severity:    "high",
				Status:      "firing",
				Labels: map[string]interface{}{
					"app": "nginx",
					"env": "production",
				},
				Annotations: map[string]interface{}{
					"summary": "High CPU usage detected",
				},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "missing required fields",
			payload:        gin.H{"title": "Test Alert"},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "请求参数校验失败",
		},
		{
			name: "invalid severity",
			payload: IngestAlertRequest{
				Fingerprint: "test-fingerprint-2",
				ClusterID:   "test-cluster",
				Title:       "Test Alert",
				Severity:    "invalid-severity",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "无效的严重级别",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonPayload, err := json.Marshal(tt.payload)
			require.NoError(t, err)

			req, err := http.NewRequest("POST", "/ingest/alert", bytes.NewBuffer(jsonPayload))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Contains(t, response["error"], tt.expectedError)
			}
		})
	}
}

func TestIngestAlertmanagerWebhook(t *testing.T) {
	handler, router, database := setupTestHandler()
	router.POST("/ingest/alertmanager", handler.IngestAlertmanagerWebhook)

	now := time.Now().UTC().Truncate(time.Second)

	payload := AlertmanagerWebhookRequest{
		Receiver: "team-test",
		Status:   "firing",
		Alerts: []AlertmanagerAlert{
			{
				Status: "firing",
				Labels: map[string]string{
					"alertname":  "HighCPUUsage",
					"severity":   "critical",
					"cluster_id": "cluster-alpha",
					"instance":   "node-1",
				},
				Annotations: map[string]string{
					"summary": "CPU 使用率持续飙升",
				},
				StartsAt:     now,
				GeneratorURL: "http://prometheus.example/highcpu",
				Fingerprint:  "existing-fingerprint",
			},
			{
				Status: "resolved",
				Labels: map[string]string{
					"alertname": "PodNotReady",
					"severity":  "warning",
					"cluster":   "cluster-alpha",
					"namespace": "default",
					"pod":       "web-0",
				},
				Annotations: map[string]string{
					"description": "Pod default/web-0 仍未就绪",
				},
				StartsAt:     now.Add(-30 * time.Minute),
				EndsAt:       now,
				GeneratorURL: "http://prometheus.example/podnotready",
			},
		},
	}

	body, err := json.Marshal(payload)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", "/ingest/alertmanager", bytes.NewBuffer(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Message string `json:"message"`
		Data    struct {
			Alerts []map[string]interface{} `json:"alerts"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Len(t, resp.Data.Alerts, 2)

	firstFingerprint, ok := resp.Data.Alerts[0]["fingerprint"].(string)
	require.True(t, ok)
	assert.Equal(t, "existing-fingerprint", firstFingerprint)

	secondFingerprint, ok := resp.Data.Alerts[1]["fingerprint"].(string)
	require.True(t, ok)
	expectedFingerprint := generateAlertmanagerFingerprint("cluster-alpha", payload.Alerts[1].Labels, payload.Alerts[1].Annotations)
	assert.Equal(t, expectedFingerprint, secondFingerprint)

	secondStatus, ok := resp.Data.Alerts[1]["status"].(string)
	require.True(t, ok)
	assert.Equal(t, "resolved", secondStatus)

	var storedAlerts []TestAlert
	require.NoError(t, database.DB.Find(&storedAlerts).Error)
	require.Len(t, storedAlerts, 2)
	for _, item := range storedAlerts {
		assert.Empty(t, item.RawPayloadKey)
	}
}

func TestIngestRCA(t *testing.T) {
	handler, router, database := setupTestHandler()
	router.POST("/ingest/rca", handler.IngestRCA)

	// 首先创建一个告警
	alertService := services.NewAlertService(database)

	alert := &models.Alert{
		Fingerprint: "test-fingerprint",
		ClusterID:   "test-cluster",
		Title:       "Test Alert",
		Severity:    "high",
		Status:      "firing",
	}

	err := alertService.CreateOrUpdateAlert(context.Background(), alert)
	require.NoError(t, err)

	tests := []struct {
		name           string
		payload        IngestRCARequest
		expectedStatus int
		expectedError  string
	}{
		{
			name: "valid rca",
			payload: IngestRCARequest{
				AlertID: alert.ID.String(),
				Status:  "completed",
				Summary: "RCA analysis completed successfully",
				Suspects: map[string]interface{}{
					"primary": "High CPU usage",
				},
				Recommendations: map[string]interface{}{
					"action": "Scale up the deployment",
				},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "invalid alert id",
			payload: IngestRCARequest{
				AlertID: "invalid-uuid",
				Status:  "completed",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "无效的告警ID",
		},
		{
			name: "invalid status",
			payload: IngestRCARequest{
				AlertID: alert.ID.String(),
				Status:  "invalid-status",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "无效的RCA状态",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonPayload, err := json.Marshal(tt.payload)
			require.NoError(t, err)

			req, err := http.NewRequest("POST", "/ingest/rca", bytes.NewBuffer(jsonPayload))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Contains(t, response["error"], tt.expectedError)
			}
		})
	}
}

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

func setupTestHandler() (*IngestHandler, *gin.Engine) {
	database := setupTestDB()

	alertService := services.NewAlertService(database)
	rcaService := services.NewRCAService(database)
	clusterService := services.NewClusterService(database)
	auditService := services.NewAuditService(database)

	storage := newFakeStorage()
	handler := NewIngestHandler(alertService, rcaService, clusterService, auditService, storage)

	gin.SetMode(gin.TestMode)
	router := gin.New()

	return handler, router
}

func TestIngestAlert(t *testing.T) {
	handler, router := setupTestHandler()
	router.POST("/ingest/alert", handler.IngestAlert)

	tests := []struct {
		name           string
		payload        IngestAlertRequest
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
			name: "missing required fields",
			payload: IngestAlertRequest{
				Title: "Test Alert",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "无效的请求数据",
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

func TestIngestRCA(t *testing.T) {
	handler, router := setupTestHandler()
	router.POST("/ingest/rca", handler.IngestRCA)

	// 首先创建一个告警
	database := setupTestDB()
	alertService := services.NewAlertService(database)

	alert := &models.Alert{
		Fingerprint: "test-fingerprint",
		ClusterID:   "test-cluster",
		Title:       "Test Alert",
		Severity:    "high",
		Status:      "firing",
	}

	err := alertService.CreateOrUpdateAlert(alert)
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

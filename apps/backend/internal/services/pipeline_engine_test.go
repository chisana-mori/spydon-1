package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"robusta-web/backend/internal/config"
	"robusta-web/backend/internal/db"
	"robusta-web/backend/internal/models"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// ============================================================================
// Mock Implementations
// ============================================================================

// MockJobRuntime 模拟任务运行时
type MockJobRuntime struct {
	mu              sync.Mutex
	LaunchCount     int
	LaunchError     error
	WaitError       error
	JobStatus       string
	JobSuccess      bool
	Templates       []JobTemplateInfo
	LaunchHistory   []JobConfig
	CancelCount     int
	CancelledJobIDs []int
}

func NewMockJobRuntime() *MockJobRuntime {
	return &MockJobRuntime{
		JobStatus:  "successful",
		JobSuccess: true,
		Templates: []JobTemplateInfo{
			{ID: 1, Name: "Test Template", Description: "Test"},
		},
	}
}

func (m *MockJobRuntime) Name() string { return "mock" }

func (m *MockJobRuntime) LaunchJob(ctx context.Context, config JobConfig) (*JobHandle, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.LaunchCount++
	m.LaunchHistory = append(m.LaunchHistory, config)

	if m.LaunchError != nil {
		return nil, m.LaunchError
	}

	return &JobHandle{
		RuntimeName: m.Name(),
		JobID:       100 + m.LaunchCount,
	}, nil
}

func (m *MockJobRuntime) WaitForJob(ctx context.Context, handle *JobHandle, pollInterval time.Duration) (*JobResult, error) {
	if m.WaitError != nil {
		return nil, m.WaitError
	}

	return &JobResult{
		Status:  m.JobStatus,
		Success: m.JobSuccess,
	}, nil
}

func (m *MockJobRuntime) GetJobStatus(ctx context.Context, handle *JobHandle) (string, error) {
	return m.JobStatus, nil
}

func (m *MockJobRuntime) CancelJob(ctx context.Context, handle *JobHandle) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.CancelCount++
	m.CancelledJobIDs = append(m.CancelledJobIDs, handle.JobID)
	return nil
}

func (m *MockJobRuntime) ListTemplates(ctx context.Context) ([]JobTemplateInfo, error) {
	return m.Templates, nil
}

func (m *MockJobRuntime) GetTemplate(ctx context.Context, id int) (*JobTemplateInfo, error) {
	for _, t := range m.Templates {
		if t.ID == id {
			return &t, nil
		}
	}
	return nil, errors.New("template not found")
}

func (m *MockJobRuntime) GetInventoryVariables(ctx context.Context, clusterName string) (string, error) {
	return "---\nkey: value", nil
}

func (m *MockJobRuntime) UpdateInventoryVariables(ctx context.Context, clusterName string, variables string) error {
	return nil
}

func (m *MockJobRuntime) PrepareClonedTemplate(ctx context.Context, config CloneTemplateConfig) (int, error) {
	// 返回一个模拟的克隆模板 ID
	return config.TemplateID + 10000, nil
}

func (m *MockJobRuntime) CleanupClonedTemplate(ctx context.Context, clonedTemplateID int) error {
	return nil
}

// Reset 重置计数器
func (m *MockJobRuntime) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.LaunchCount = 0
	m.LaunchHistory = nil
	m.CancelCount = 0
	m.CancelledJobIDs = nil
}

// MockMetricsRuntime 模拟指标运行时
type MockMetricsRuntime struct {
	QueryValue float64
	QueryError error
	QueryCount int
}

func NewMockMetricsRuntime() *MockMetricsRuntime {
	return &MockMetricsRuntime{
		QueryValue: 1.0,
	}
}

func (m *MockMetricsRuntime) Name() string { return "mock_metrics" }

func (m *MockMetricsRuntime) Query(ctx context.Context, endpoint string, query string) (float64, error) {
	m.QueryCount++
	if m.QueryError != nil {
		return 0, m.QueryError
	}
	return m.QueryValue, nil
}

func (m *MockMetricsRuntime) QueryRaw(ctx context.Context, endpoint string, query string) (interface{}, error) {
	return m.QueryValue, m.QueryError
}

// ============================================================================
// Test Helpers
// ============================================================================

// setupPipelineTestDB 初始化测试用的内存数据库（专用于 Pipeline 测试）
// 每个测试使用独立的数据库实例以避免冲突
func setupPipelineTestDB(t *testing.T) *db.Database {
	// 使用唯一名称的共享内存数据库，确保在同一测试中多次访问时数据持久
	dbName := fmt.Sprintf("file:memdb_%s?mode=memory&cache=shared", t.Name())
	gormDB, err := gorm.Open(sqlite.Open(dbName), &gorm.Config{})
	require.NoError(t, err, "failed to connect database")

	err = gormDB.AutoMigrate(
		&models.Cluster{},
		&models.PipelineTemplate{},
		&models.PipelineExecution{},
		&models.StageRun{},
	)
	require.NoError(t, err, "failed to migrate")

	return &db.Database{DB: gormDB}
}

// createPipelineTestCluster 创建测试集群，返回 ID (uint64)
func createPipelineTestCluster(t *testing.T, database *db.Database, name string) uint64 {
	cluster := &models.Cluster{
		Name:          name,
		PrometheusURL: "http://prometheus.local:9090",
	}
	err := database.Create(cluster).Error
	require.NoError(t, err)
	return uint64(cluster.ID)
}

// createPipelineTestTemplate 创建测试模板
func createPipelineTestTemplate(t *testing.T, database *db.Database, name string, stages []models.StageDefinition) *models.PipelineTemplate {
	stagesJSON, err := json.Marshal(stages)
	require.NoError(t, err)

	template := &models.PipelineTemplate{
		Name:   name,
		Stages: datatypes.JSON(stagesJSON),
	}
	err = database.Create(template).Error
	require.NoError(t, err)
	return template
}

// waitForPipelineExecutionStatus 等待执行状态变更
func waitForPipelineExecutionStatus(t *testing.T, database *db.Database, execID uint64, expectedStatus models.ExecutionStatus, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		var exec models.PipelineExecution
		database.First(&exec, execID)
		if exec.Status == expectedStatus {
			return true
		}
		time.Sleep(50 * time.Millisecond)
	}
	return false
}

// ============================================================================
// Test Cases
// ============================================================================

func TestPipelineEngineNew(t *testing.T) {
	database := setupPipelineTestDB(t)
	mockJob := NewMockJobRuntime()
	mockMetrics := NewMockMetricsRuntime()

	t.Run("创建带运行时的引擎", func(t *testing.T) {
		engine := NewPipelineEngine(
			database,
			&config.Config{},
			WithJobRuntime(mockJob),
			WithMetricsRuntime(mockMetrics),
		)

		assert.NotNil(t, engine)
		assert.Equal(t, mockJob, engine.GetJobRuntime())
	})

	t.Run("创建不带运行时的引擎", func(t *testing.T) {
		engine := NewPipelineEngine(database, &config.Config{})
		assert.NotNil(t, engine)
		assert.Nil(t, engine.GetJobRuntime())
	})

	t.Run("动态设置运行时", func(t *testing.T) {
		engine := NewPipelineEngine(database, &config.Config{})
		engine.SetJobRuntime(mockJob)
		engine.SetMetricsRuntime(mockMetrics)

		assert.Equal(t, mockJob, engine.GetJobRuntime())
	})
}

func TestPipelineEngineTemplateManagement(t *testing.T) {
	database := setupPipelineTestDB(t)
	engine := NewPipelineEngine(database, &config.Config{})

	t.Run("创建模板", func(t *testing.T) {
		stages := []models.StageDefinition{
			{ID: "s1", Name: "Stage 1", Type: models.StageTypeAWXJob},
		}

		template, err := engine.CreateTemplate("Test Template", "Description", stages, 1)
		require.NoError(t, err)
		assert.NotZero(t, template.ID)
		assert.Equal(t, "Test Template", template.Name)
	})

	t.Run("获取模板", func(t *testing.T) {
		stages := []models.StageDefinition{
			{ID: "s1", Name: "Stage 1", Type: models.StageTypeAWXJob},
		}
		created, _ := engine.CreateTemplate("Get Template", "", stages, 1)

		template, err := engine.GetTemplate(created.ID)
		require.NoError(t, err)
		assert.Equal(t, created.ID, template.ID)
	})

	t.Run("获取不存在的模板", func(t *testing.T) {
		_, err := engine.GetTemplate(99999)
		assert.Error(t, err)
	})

	t.Run("更新模板", func(t *testing.T) {
		stages := []models.StageDefinition{
			{ID: "s1", Name: "Stage 1", Type: models.StageTypeAWXJob},
		}
		created, _ := engine.CreateTemplate("Update Template", "", stages, 1)

		newStages := []models.StageDefinition{
			{ID: "s1", Name: "Updated Stage", Type: models.StageTypeDelay},
		}
		updated, err := engine.UpdateTemplate(created.ID, "Updated Name", "New Desc", newStages)
		require.NoError(t, err)
		assert.Equal(t, "Updated Name", updated.Name)
	})

	t.Run("列出模板", func(t *testing.T) {
		templates, _, err := engine.ListTemplates(1, 10, "")
		require.NoError(t, err)
		assert.NotEmpty(t, templates)
	})

	t.Run("删除模板", func(t *testing.T) {
		stages := []models.StageDefinition{
			{ID: "s1", Name: "Stage 1", Type: models.StageTypeAWXJob},
		}
		created, _ := engine.CreateTemplate("Delete Template", "", stages, 1)

		err := engine.DeleteTemplate(created.ID)
		require.NoError(t, err)

		_, err = engine.GetTemplate(created.ID)
		assert.Error(t, err)
	})
}

func TestPipelineEngineExecuteAWXJob(t *testing.T) {
	t.Run("成功执行AWX任务", func(t *testing.T) {
		database := setupPipelineTestDB(t)
		mockJob := NewMockJobRuntime()
		mockJob.JobStatus = "successful"
		mockJob.JobSuccess = true

		engine := NewPipelineEngine(
			database, &config.Config{},
			WithJobRuntime(mockJob),
		)

		clusterID := createPipelineTestCluster(t, database, "test-cluster")
		stages := []models.StageDefinition{
			{
				ID:   "job-1",
				Name: "AWX Job",
				Type: models.StageTypeAWXJob,
				Config: models.StageConfig{
					AWXTemplateID:   1,
					AWXTemplateName: "Test Job",
				},
			},
		}
		template := createPipelineTestTemplate(t, database, "AWX Pipeline", stages)

		exec, err := engine.StartExecution(context.Background(), template.ID, clusterID, nil, 1, nil, 0, nil, false, true)
		require.NoError(t, err)

		// 等待完成
		assert.True(t, waitForPipelineExecutionStatus(t, database, exec.ID, models.ExecutionStatusSuccessful, 5*time.Second))
		assert.Equal(t, 1, mockJob.LaunchCount)
	})

	t.Run("AWX任务失败导致执行失败", func(t *testing.T) {
		database := setupPipelineTestDB(t)
		mockJob := NewMockJobRuntime()
		mockJob.JobStatus = "failed"
		mockJob.JobSuccess = false

		engine := NewPipelineEngine(
			database, &config.Config{},
			WithJobRuntime(mockJob),
		)

		clusterID := createPipelineTestCluster(t, database, "test-cluster")
		stages := []models.StageDefinition{
			{
				ID:        "job-1",
				Name:      "Failing Job",
				Type:      models.StageTypeAWXJob,
				Config:    models.StageConfig{AWXTemplateID: 1},
				OnFailure: models.FailureStrategyAbort,
			},
		}
		template := createPipelineTestTemplate(t, database, "Failing Pipeline", stages)

		exec, err := engine.StartExecution(context.Background(), template.ID, clusterID, nil, 1, nil, 0, nil, false, true)
		require.NoError(t, err)

		assert.True(t, waitForPipelineExecutionStatus(t, database, exec.ID, models.ExecutionStatusFailed, 5*time.Second))
	})

	t.Run("任务运行时未配置时报错", func(t *testing.T) {
		database := setupPipelineTestDB(t)
		engine := NewPipelineEngine(database, &config.Config{}) // 没有配置 JobRuntime

		clusterID := createPipelineTestCluster(t, database, "test-cluster")
		stages := []models.StageDefinition{
			{ID: "job-1", Name: "Job", Type: models.StageTypeAWXJob, OnFailure: models.FailureStrategyAbort},
		}
		template := createPipelineTestTemplate(t, database, "No Runtime Pipeline", stages)

		exec, err := engine.StartExecution(context.Background(), template.ID, clusterID, nil, 1, nil, 0, nil, false, true)
		require.NoError(t, err)

		// 没有 JobRuntime 时，AWX Job 阶段会失败
		assert.True(t, waitForPipelineExecutionStatus(t, database, exec.ID, models.ExecutionStatusFailed, 10*time.Second),
			"执行应该因为缺少 JobRuntime 而失败")
	})
}

func TestPipelineEngineExecuteCheck(t *testing.T) {
	t.Run("检查成功 - 值匹配", func(t *testing.T) {
		database := setupPipelineTestDB(t)
		mockMetrics := NewMockMetricsRuntime()
		mockMetrics.QueryValue = 1.0

		engine := NewPipelineEngine(
			database, &config.Config{},
			WithMetricsRuntime(mockMetrics),
		)

		clusterID := createPipelineTestCluster(t, database, "test-cluster")
		stages := []models.StageDefinition{
			{
				ID:   "check-1",
				Name: "Pre Check",
				Type: models.StageTypePreCheck,
				Config: models.StageConfig{
					PromQuery:     "up",
					ExpectedValue: "1",
					Operator:      "eq",
				},
			},
		}
		template := createPipelineTestTemplate(t, database, "Check Pipeline", stages)

		exec, err := engine.StartExecution(context.Background(), template.ID, clusterID, nil, 1, nil, 0, nil, false, true)
		require.NoError(t, err)

		assert.True(t, waitForPipelineExecutionStatus(t, database, exec.ID, models.ExecutionStatusSuccessful, 5*time.Second))
		assert.Equal(t, 1, mockMetrics.QueryCount)
	})

	t.Run("检查失败 - 值不匹配", func(t *testing.T) {
		database := setupPipelineTestDB(t)
		mockMetrics := NewMockMetricsRuntime()
		mockMetrics.QueryValue = 0.0 // 不等于期望的 1

		engine := NewPipelineEngine(
			database, &config.Config{},
			WithMetricsRuntime(mockMetrics),
		)

		clusterID := createPipelineTestCluster(t, database, "test-cluster")
		stages := []models.StageDefinition{
			{
				ID:        "check-1",
				Name:      "Failing Check",
				Type:      models.StageTypePreCheck,
				Config:    models.StageConfig{PromQuery: "up", ExpectedValue: "1", Operator: "eq"},
				OnFailure: models.FailureStrategyAbort,
			},
		}
		template := createPipelineTestTemplate(t, database, "Failing Check Pipeline", stages)

		exec, err := engine.StartExecution(context.Background(), template.ID, clusterID, nil, 1, nil, 0, nil, false, true)
		require.NoError(t, err)

		assert.True(t, waitForPipelineExecutionStatus(t, database, exec.ID, models.ExecutionStatusFailed, 5*time.Second))
	})

	t.Run("检查 - 各种操作符", func(t *testing.T) {
		testCases := []struct {
			name       string
			operator   string
			value      float64
			expected   string
			shouldPass bool
		}{
			{"gt success", "gt", 10, "5", true},
			{"gt fail", "gt", 3, "5", false},
			{"lt success", "lt", 3, "5", true},
			{"lt fail", "lt", 10, "5", false},
			{"ge success", "ge", 5, "5", true},
			{"le success", "le", 5, "5", true},
			{"ne success", "ne", 3, "5", true},
			{"ne fail", "ne", 5, "5", false},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				database := setupPipelineTestDB(t)
				mockMetrics := NewMockMetricsRuntime()
				mockMetrics.QueryValue = tc.value

				engine := NewPipelineEngine(
					database, &config.Config{},
					WithMetricsRuntime(mockMetrics),
				)

				clusterID := createPipelineTestCluster(t, database, "test-cluster")
				stages := []models.StageDefinition{
					{
						ID:        "check-1",
						Name:      "Check",
						Type:      models.StageTypePreCheck,
						Config:    models.StageConfig{PromQuery: "metric", ExpectedValue: tc.expected, Operator: tc.operator},
						OnFailure: models.FailureStrategyAbort,
					},
				}
				template := createPipelineTestTemplate(t, database, "Op Check", stages)

				exec, err := engine.StartExecution(context.Background(), template.ID, clusterID, nil, 1, nil, 0, nil, false, true)
				require.NoError(t, err)
				if tc.shouldPass {
					assert.True(t, waitForPipelineExecutionStatus(t, database, exec.ID, models.ExecutionStatusSuccessful, 5*time.Second))
				} else {
					assert.True(t, waitForPipelineExecutionStatus(t, database, exec.ID, models.ExecutionStatusFailed, 5*time.Second))
				}
			})
		}
	})
}

func TestPipelineEngineManualGate(t *testing.T) {
	t.Run("人工审批门暂停执行", func(t *testing.T) {
		database := setupPipelineTestDB(t)
		mockJob := NewMockJobRuntime()

		engine := NewPipelineEngine(
			database, &config.Config{},
			WithJobRuntime(mockJob),
		)

		clusterID := createPipelineTestCluster(t, database, "test-cluster")
		stages := []models.StageDefinition{
			{ID: "gate-1", Name: "Approval Gate", Type: models.StageTypeManualGate},
		}
		template := createPipelineTestTemplate(t, database, "Gate Pipeline", stages)

		exec, err := engine.StartExecution(context.Background(), template.ID, clusterID, nil, 1, nil, 0, nil, false, true)
		require.NoError(t, err)

		// 应该进入暂停状态
		assert.True(t, waitForPipelineExecutionStatus(t, database, exec.ID, models.ExecutionStatusPaused, 5*time.Second))
	})

	t.Run("审批后继续执行", func(t *testing.T) {
		database := setupPipelineTestDB(t)
		mockJob := NewMockJobRuntime()

		engine := NewPipelineEngine(
			database, &config.Config{},
			WithJobRuntime(mockJob),
		)

		clusterID := createPipelineTestCluster(t, database, "test-cluster")
		stages := []models.StageDefinition{
			{ID: "gate-1", Name: "Approval Gate", Type: models.StageTypeManualGate},
			{ID: "job-1", Name: "After Gate Job", Type: models.StageTypeAWXJob, Config: models.StageConfig{AWXTemplateID: 1}},
		}
		template := createPipelineTestTemplate(t, database, "Gate Pipeline", stages)

		exec, err := engine.StartExecution(context.Background(), template.ID, clusterID, nil, 1, nil, 0, nil, false, true)
		require.NoError(t, err)

		// 等待暂停
		assert.True(t, waitForPipelineExecutionStatus(t, database, exec.ID, models.ExecutionStatusPaused, 5*time.Second))

		// 恢复执行
		err = engine.ResumeExecution(context.Background(), exec.ID, 1, "Approved")
		require.NoError(t, err)

		// 应该完成
		assert.True(t, waitForPipelineExecutionStatus(t, database, exec.ID, models.ExecutionStatusSuccessful, 5*time.Second))
		assert.Equal(t, 1, mockJob.LaunchCount)
	})
}

func TestPipelineEngineDelay(t *testing.T) {
	t.Run("延时阶段", func(t *testing.T) {
		database := setupPipelineTestDB(t)

		engine := NewPipelineEngine(database, &config.Config{})

		clusterID := createPipelineTestCluster(t, database, "test-cluster")
		stages := []models.StageDefinition{
			{
				ID:     "delay-1",
				Name:   "Short Delay",
				Type:   models.StageTypeDelay,
				Config: models.StageConfig{DelaySeconds: 1}, // 1秒延时
			},
		}
		template := createPipelineTestTemplate(t, database, "Delay Pipeline", stages)

		start := time.Now()
		exec, err := engine.StartExecution(context.Background(), template.ID, clusterID, nil, 1, nil, 0, nil, false, true)
		require.NoError(t, err)

		assert.True(t, waitForPipelineExecutionStatus(t, database, exec.ID, models.ExecutionStatusSuccessful, 5*time.Second))
		elapsed := time.Since(start)
		assert.GreaterOrEqual(t, elapsed, 1*time.Second)
	})
}

func TestPipelineEngineFailureStrategies(t *testing.T) {
	t.Run("Abort策略 - 失败后终止", func(t *testing.T) {
		database := setupPipelineTestDB(t)
		mockJob := NewMockJobRuntime()
		mockJob.JobSuccess = false
		mockJob.JobStatus = "failed"

		engine := NewPipelineEngine(
			database, &config.Config{},
			WithJobRuntime(mockJob),
		)

		clusterID := createPipelineTestCluster(t, database, "test-cluster")
		stages := []models.StageDefinition{
			{ID: "job-1", Name: "Job 1", Type: models.StageTypeAWXJob, OnFailure: models.FailureStrategyAbort},
			{ID: "job-2", Name: "Job 2", Type: models.StageTypeAWXJob}, // 不应该执行
		}
		template := createPipelineTestTemplate(t, database, "Abort Pipeline", stages)

		exec, err := engine.StartExecution(context.Background(), template.ID, clusterID, nil, 1, nil, 0, nil, false, true)
		require.NoError(t, err)

		assert.True(t, waitForPipelineExecutionStatus(t, database, exec.ID, models.ExecutionStatusFailed, 5*time.Second))
		assert.Equal(t, 1, mockJob.LaunchCount) // 只执行了第一个
	})

	t.Run("Continue策略 - 失败后继续", func(t *testing.T) {
		database := setupPipelineTestDB(t)
		mockJob := NewMockJobRuntime()
		mockJob.JobSuccess = false
		mockJob.JobStatus = "failed"

		engine := NewPipelineEngine(
			database, &config.Config{},
			WithJobRuntime(mockJob),
		)

		clusterID := createPipelineTestCluster(t, database, "test-cluster")
		stages := []models.StageDefinition{
			{ID: "job-1", Name: "Failing Job", Type: models.StageTypeAWXJob, OnFailure: models.FailureStrategyContinue},
			{ID: "delay-1", Name: "Delay", Type: models.StageTypeDelay, Config: models.StageConfig{DelaySeconds: 1}},
		}
		template := createPipelineTestTemplate(t, database, "Continue Pipeline", stages)

		exec, err := engine.StartExecution(context.Background(), template.ID, clusterID, nil, 1, nil, 0, nil, false, true)
		require.NoError(t, err)

		// 应该完成（尽管第一个失败了）
		assert.True(t, waitForPipelineExecutionStatus(t, database, exec.ID, models.ExecutionStatusSuccessful, 5*time.Second))
		assert.Equal(t, 1, mockJob.LaunchCount)
	})

	t.Run("Pause策略 - 失败后暂停", func(t *testing.T) {
		database := setupPipelineTestDB(t)
		mockJob := NewMockJobRuntime()
		mockJob.JobSuccess = false
		mockJob.JobStatus = "failed"

		engine := NewPipelineEngine(
			database, &config.Config{},
			WithJobRuntime(mockJob),
		)

		clusterID := createPipelineTestCluster(t, database, "test-cluster")
		stages := []models.StageDefinition{
			{ID: "job-1", Name: "Job 1", Type: models.StageTypeAWXJob, OnFailure: models.FailureStrategyPause},
		}
		template := createPipelineTestTemplate(t, database, "Pause Pipeline", stages)

		exec, err := engine.StartExecution(context.Background(), template.ID, clusterID, nil, 1, nil, 0, nil, false, true)
		require.NoError(t, err)

		assert.True(t, waitForPipelineExecutionStatus(t, database, exec.ID, models.ExecutionStatusPaused, 5*time.Second))
	})
}

func TestPipelineEngineCancelExecution(t *testing.T) {
	t.Run("取消运行中的执行", func(t *testing.T) {
		database := setupPipelineTestDB(t)

		engine := NewPipelineEngine(database, &config.Config{})

		clusterID := createPipelineTestCluster(t, database, "test-cluster")
		stages := []models.StageDefinition{
			{ID: "delay-1", Name: "Long Delay", Type: models.StageTypeDelay, Config: models.StageConfig{DelaySeconds: 60}},
		}
		template := createPipelineTestTemplate(t, database, "Long Pipeline", stages)

		exec, err := engine.StartExecution(context.Background(), template.ID, clusterID, nil, 1, nil, 0, nil, false, true)
		require.NoError(t, err)

		// 等待开始运行
		time.Sleep(100 * time.Millisecond)

		// 取消
		err = engine.CancelExecution(exec.ID)
		require.NoError(t, err)

		assert.True(t, waitForPipelineExecutionStatus(t, database, exec.ID, models.ExecutionStatusCanceled, 5*time.Second))
	})
}

func TestPipelineEngineBatching(t *testing.T) {
	t.Run("批次执行 - 自动生成阶段", func(t *testing.T) {
		database := setupPipelineTestDB(t)
		mockJob := NewMockJobRuntime()

		engine := NewPipelineEngine(
			database, &config.Config{},
			WithJobRuntime(mockJob),
		)

		clusterID := createPipelineTestCluster(t, database, "test-cluster")
		stages := []models.StageDefinition{
			{ID: "job-1", Name: "Batch Job", Type: models.StageTypeAWXJob, Config: models.StageConfig{AWXTemplateID: 1}},
		}
		template := createPipelineTestTemplate(t, database, "Batch Pipeline", stages)

		targetNodes := []string{"node1", "node2", "node3", "node4", "node5"}
		batchSize := 2
		pause := true

		exec, err := engine.StartExecution(context.Background(), template.ID, clusterID, nil, 1, targetNodes, batchSize, nil, pause, true)
		require.NoError(t, err)

		// 验证 EffectiveStages
		var effectiveStages []models.StageDefinition
		err = json.Unmarshal(exec.EffectiveStages, &effectiveStages)
		require.NoError(t, err)

		// 5个节点，每批2个 = 3批
		// 每批后有 ManualGate（除了最后一批）= 2个Gate
		// 总共 3 + 2 = 5 个阶段
		assert.Len(t, effectiveStages, 5)
	})

	t.Run("批次执行 - 无暂停", func(t *testing.T) {
		database := setupPipelineTestDB(t)
		mockJob := NewMockJobRuntime()

		engine := NewPipelineEngine(
			database, &config.Config{},
			WithJobRuntime(mockJob),
		)

		clusterID := createPipelineTestCluster(t, database, "test-cluster")
		stages := []models.StageDefinition{
			{ID: "job-1", Name: "Batch Job", Type: models.StageTypeAWXJob, Config: models.StageConfig{AWXTemplateID: 1}},
		}
		template := createPipelineTestTemplate(t, database, "Batch No Pause Pipeline", stages)

		targetNodes := []string{"node1", "node2", "node3"}
		batchSize := 2
		pause := false

		exec, err := engine.StartExecution(context.Background(), template.ID, clusterID, nil, 1, targetNodes, batchSize, nil, pause, true)
		require.NoError(t, err)

		// 验证 EffectiveStages - 无 Gate
		var effectiveStages []models.StageDefinition
		err = json.Unmarshal(exec.EffectiveStages, &effectiveStages)
		require.NoError(t, err)

		// 3个节点，每批2个 = 2批，无暂停 = 只有2个Job阶段
		assert.Len(t, effectiveStages, 2)
		for _, stage := range effectiveStages {
			assert.Equal(t, models.StageTypeAWXJob, stage.Type)
		}
	})
}

func TestPipelineEngineResumeSkipsCompletedStages(t *testing.T) {
	t.Run("恢复执行时跳过已完成的阶段", func(t *testing.T) {
		database := setupPipelineTestDB(t)
		mockJob := NewMockJobRuntime()

		engine := NewPipelineEngine(
			database, &config.Config{},
			WithJobRuntime(mockJob),
		)

		clusterID := createPipelineTestCluster(t, database, "test-cluster")

		// 创建模板
		stages := []models.StageDefinition{
			{ID: "job-1", Name: "Job 1", Type: models.StageTypeAWXJob, Config: models.StageConfig{AWXTemplateID: 1}},
			{ID: "gate-1", Name: "Gate", Type: models.StageTypeManualGate},
			{ID: "job-2", Name: "Job 2", Type: models.StageTypeAWXJob, Config: models.StageConfig{AWXTemplateID: 2}},
		}
		stagesJSON, _ := json.Marshal(stages)

		// 手动创建模板和已暂停的执行记录
		template := &models.PipelineTemplate{
			Name:   "Resume Test",
			Stages: datatypes.JSON(stagesJSON),
		}
		database.Create(template)

		execution := &models.PipelineExecution{
			PipelineTemplateID: template.ID,
			ClusterID:          clusterID,
			ClusterName:        "test-cluster",
			Status:             models.ExecutionStatusPaused,
			CurrentStageID:     "gate-1",
			Parameters:         datatypes.JSON(`{}`),
		}
		database.Create(execution)

		// 创建阶段记录 - Job 1 已完成，Gate 等待审批
		database.Create(&models.StageRun{
			ExecutionID: execution.ID, StageID: "job-1", StageName: "Job 1",
			StageType: models.StageTypeAWXJob, Status: models.StageRunStatusSuccessful,
		})
		database.Create(&models.StageRun{
			ExecutionID: execution.ID, StageID: "gate-1", StageName: "Gate",
			StageType: models.StageTypeManualGate, Status: models.StageRunStatusWaitingApproval,
		})
		database.Create(&models.StageRun{
			ExecutionID: execution.ID, StageID: "job-2", StageName: "Job 2",
			StageType: models.StageTypeAWXJob, Status: models.StageRunStatusPending,
		})

		// 恢复执行
		err := engine.ResumeExecution(context.Background(), execution.ID, 1, "Approved")
		require.NoError(t, err)

		// 等待完成
		assert.True(t, waitForPipelineExecutionStatus(t, database, execution.ID, models.ExecutionStatusSuccessful, 5*time.Second))

		// 验证只有 Job 2 被执行（Job 1 被跳过）
		assert.Equal(t, 1, mockJob.LaunchCount)
	})
}

func TestPipelineEngineParameterValidation(t *testing.T) {
	t.Run("必填参数缺失", func(t *testing.T) {
		database := setupPipelineTestDB(t)
		mockJob := NewMockJobRuntime()

		engine := NewPipelineEngine(
			database, &config.Config{},
			WithJobRuntime(mockJob),
		)

		clusterID := createPipelineTestCluster(t, database, "test-cluster")
		stages := []models.StageDefinition{
			{
				ID:   "job-1",
				Name: "Job with Params",
				Type: models.StageTypeAWXJob,
				Config: models.StageConfig{
					AWXTemplateID: 1,
					Parameters: []models.ParameterBinding{
						{Name: "required_param", Label: "Required Param", Required: true},
					},
				},
			},
		}
		template := createPipelineTestTemplate(t, database, "Param Pipeline", stages)

		// 不提供必填参数
		_, err := engine.StartExecution(context.Background(), template.ID, clusterID, nil, 1, nil, 0, nil, false, true)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "缺少必填参数")
	})

	t.Run("有默认值的必填参数", func(t *testing.T) {
		database := setupPipelineTestDB(t)
		mockJob := NewMockJobRuntime()

		engine := NewPipelineEngine(
			database, &config.Config{},
			WithJobRuntime(mockJob),
		)

		clusterID := createPipelineTestCluster(t, database, "test-cluster")
		stages := []models.StageDefinition{
			{
				ID:   "job-1",
				Name: "Job with Default Param",
				Type: models.StageTypeAWXJob,
				Config: models.StageConfig{
					AWXTemplateID: 1,
					Parameters: []models.ParameterBinding{
						{Name: "param", Label: "Param", Required: true, DefaultValue: "default"},
					},
				},
			},
		}
		template := createPipelineTestTemplate(t, database, "Default Param Pipeline", stages)

		// 不提供参数，但有默认值，应该成功
		exec, err := engine.StartExecution(context.Background(), template.ID, clusterID, nil, 1, nil, 0, nil, false, true)
		require.NoError(t, err)
		assert.NotNil(t, exec)
	})
}

func TestPipelineEngineListJobTemplates(t *testing.T) {
	t.Run("列出任务模板", func(t *testing.T) {
		database := setupPipelineTestDB(t)
		mockJob := NewMockJobRuntime()
		mockJob.Templates = []JobTemplateInfo{
			{ID: 1, Name: "Template 1", Description: "Desc 1"},
			{ID: 2, Name: "Template 2", Description: "Desc 2"},
		}

		engine := NewPipelineEngine(
			database, &config.Config{},
			WithJobRuntime(mockJob),
		)

		templates, err := engine.ListJobTemplates(context.Background())
		require.NoError(t, err)
		assert.Len(t, templates, 2)
	})

	t.Run("未配置运行时", func(t *testing.T) {
		database := setupPipelineTestDB(t)
		engine := NewPipelineEngine(database, &config.Config{})

		_, err := engine.ListJobTemplates(context.Background())
		assert.Error(t, err)
	})
}

func TestPipelineEngineGetExecution(t *testing.T) {
	t.Run("获取执行详情", func(t *testing.T) {
		database := setupPipelineTestDB(t)
		mockJob := NewMockJobRuntime()

		engine := NewPipelineEngine(
			database, &config.Config{},
			WithJobRuntime(mockJob),
		)

		clusterID := createPipelineTestCluster(t, database, "test-cluster")
		stages := []models.StageDefinition{
			{ID: "job-1", Name: "Job", Type: models.StageTypeAWXJob, Config: models.StageConfig{AWXTemplateID: 1}},
		}
		template := createPipelineTestTemplate(t, database, "Test Pipeline", stages)

		exec, err := engine.StartExecution(context.Background(), template.ID, clusterID, nil, 1, nil, 0, nil, false, true)
		require.NoError(t, err)

		// 获取详情
		detail, err := engine.GetExecution(exec.ID)
		require.NoError(t, err)
		assert.Equal(t, exec.ID, detail.ID)
		assert.NotNil(t, detail.Template)
	})

	t.Run("获取不存在的执行", func(t *testing.T) {
		database := setupPipelineTestDB(t)
		engine := NewPipelineEngine(database, &config.Config{})

		_, err := engine.GetExecution(99999)
		assert.Error(t, err)
	})
}

func TestPipelineEngineGetExecutionHistory(t *testing.T) {
	t.Run("获取集群执行历史", func(t *testing.T) {
		database := setupPipelineTestDB(t)
		mockJob := NewMockJobRuntime()

		engine := NewPipelineEngine(
			database, &config.Config{},
			WithJobRuntime(mockJob),
		)

		clusterID := createPipelineTestCluster(t, database, "test-cluster")
		stages := []models.StageDefinition{
			{ID: "job-1", Name: "Job", Type: models.StageTypeAWXJob, Config: models.StageConfig{AWXTemplateID: 1}},
		}
		template := createPipelineTestTemplate(t, database, "Test Pipeline", stages)

		// 创建多个执行
		for i := 0; i < 3; i++ {
			_, err := engine.StartExecution(context.Background(), template.ID, clusterID, nil, 1, nil, 0, nil, false, true)
			require.NoError(t, err)
		}
		// 等待完成
		time.Sleep(500 * time.Millisecond)

		history, err := engine.GetExecutionHistory(clusterID, 10)
		require.NoError(t, err)
		assert.Len(t, history, 3)
	})
}

func TestPipelineEngineGetActiveExecutions(t *testing.T) {
	t.Run("获取活跃执行", func(t *testing.T) {
		database := setupPipelineTestDB(t)

		engine := NewPipelineEngine(database, &config.Config{})

		clusterID := createPipelineTestCluster(t, database, "test-cluster")
		stages := []models.StageDefinition{
			{ID: "delay-1", Name: "Delay", Type: models.StageTypeDelay, Config: models.StageConfig{DelaySeconds: 30}},
		}
		template := createPipelineTestTemplate(t, database, "Long Pipeline", stages)

		// 启动执行
		_, err := engine.StartExecution(context.Background(), template.ID, clusterID, nil, 1, nil, 0, nil, false, true)
		require.NoError(t, err)

		time.Sleep(100 * time.Millisecond)

		active, err := engine.GetActiveExecutions(10)
		require.NoError(t, err)
		assert.NotEmpty(t, active)
	})
}

func TestPipelineEngineMetricsQueryError(t *testing.T) {
	t.Run("指标查询出错", func(t *testing.T) {
		database := setupPipelineTestDB(t)
		mockMetrics := NewMockMetricsRuntime()
		mockMetrics.QueryError = errors.New("Prometheus connection failed")

		engine := NewPipelineEngine(
			database, &config.Config{},
			WithMetricsRuntime(mockMetrics),
		)

		clusterID := createPipelineTestCluster(t, database, "test-cluster")
		stages := []models.StageDefinition{
			{
				ID:        "check-1",
				Name:      "Failing Check",
				Type:      models.StageTypePreCheck,
				Config:    models.StageConfig{PromQuery: "up"},
				OnFailure: models.FailureStrategyAbort,
			},
		}
		template := createPipelineTestTemplate(t, database, "Error Check Pipeline", stages)

		exec, err := engine.StartExecution(context.Background(), template.ID, clusterID, nil, 1, nil, 0, nil, false, true)
		require.NoError(t, err)

		assert.True(t, waitForPipelineExecutionStatus(t, database, exec.ID, models.ExecutionStatusFailed, 5*time.Second))
	})
}

func TestPipelineEngineMultipleStages(t *testing.T) {
	t.Run("多阶段顺序执行", func(t *testing.T) {
		database := setupPipelineTestDB(t)
		mockJob := NewMockJobRuntime()
		mockMetrics := NewMockMetricsRuntime()

		engine := NewPipelineEngine(
			database, &config.Config{},
			WithJobRuntime(mockJob),
			WithMetricsRuntime(mockMetrics),
		)

		clusterID := createPipelineTestCluster(t, database, "test-cluster")
		stages := []models.StageDefinition{
			{ID: "check-1", Name: "Pre Check", Type: models.StageTypePreCheck, Config: models.StageConfig{PromQuery: "up"}},
			{ID: "job-1", Name: "Deploy", Type: models.StageTypeAWXJob, Config: models.StageConfig{AWXTemplateID: 1}},
			{ID: "delay-1", Name: "Wait", Type: models.StageTypeDelay, Config: models.StageConfig{DelaySeconds: 1}},
			{ID: "check-2", Name: "Post Check", Type: models.StageTypePostCheck, Config: models.StageConfig{PromQuery: "up"}},
		}
		template := createPipelineTestTemplate(t, database, "Multi Stage Pipeline", stages)

		exec, err := engine.StartExecution(context.Background(), template.ID, clusterID, nil, 1, nil, 0, nil, false, true)
		require.NoError(t, err)

		assert.True(t, waitForPipelineExecutionStatus(t, database, exec.ID, models.ExecutionStatusSuccessful, 10*time.Second))
		assert.Equal(t, 1, mockJob.LaunchCount)
		assert.Equal(t, 2, mockMetrics.QueryCount) // Pre + Post check
	})
}

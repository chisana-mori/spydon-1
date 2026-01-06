package services

import (
	"context"
	"fmt"
	"testing"
	"time"

	"robusta-web/backend/internal/config"
	"robusta-web/backend/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockHook 实现 PipelineHook 接口
type MockHook struct {
	CapturedCtx []PipelineHookContext
	BeforeError error
	AfterError  error
	BeforeCount int
	AfterCount  int
}

func (m *MockHook) Name() string { return "MockHook" }

func (m *MockHook) BeforeStage(ctx context.Context, hookCtx PipelineHookContext) error {
	m.BeforeCount++
	m.CapturedCtx = append(m.CapturedCtx, hookCtx)
	return m.BeforeError
}

func (m *MockHook) AfterStage(ctx context.Context, hookCtx PipelineHookContext, result *StageResult) error {
	m.AfterCount++
	return m.AfterError
}

func TestPipelineHooks(t *testing.T) {
	database := setupPipelineTestDB(t)
	mockJob := NewMockJobRuntime()

	t.Run("Hook正常执行", func(t *testing.T) {
		mockHook := &MockHook{}
		engine := NewPipelineEngine(
			database,
			&config.Config{},
			WithJobRuntime(mockJob),
			WithHooks(mockHook),
		)

		clusterID := createPipelineTestCluster(t, database, "hook-cluster")
		stages := []models.StageDefinition{
			{ID: "stage-1", Name: "Stage 1", Type: models.StageTypeAWXJob, Config: models.StageConfig{AWXTemplateID: 1}},
			{ID: "stage-2", Name: "Stage 2", Type: models.StageTypeAWXJob, Config: models.StageConfig{AWXTemplateID: 2}},
		}
		template := createPipelineTestTemplate(t, database, "Hook Pipeline", stages)

		exec, err := engine.StartExecution(context.Background(), template.ID, clusterID, nil, 1, nil, 0, nil, false, true)
		require.NoError(t, err)

		assert.True(t, waitForPipelineExecutionStatus(t, database, exec.ID, models.ExecutionStatusSuccessful, 5*time.Second))

		// 验证 Hook 调用次数
		assert.Equal(t, 2, mockHook.BeforeCount)
		assert.Equal(t, 2, mockHook.AfterCount)

		// 验证 Hook 上下文传递
		require.Len(t, mockHook.CapturedCtx, 2)
		assert.Nil(t, mockHook.CapturedCtx[0].PreviousResult)    // 第一阶段没有 PreviousResult
		assert.NotNil(t, mockHook.CapturedCtx[1].PreviousResult) // 第二阶段有
		assert.True(t, mockHook.CapturedCtx[1].PreviousResult.Success)
	})

	t.Run("BeforeHook失败阻止执行", func(t *testing.T) {
		mockHook := &MockHook{BeforeError: fmt.Errorf("pre-check failed")}
		engine := NewPipelineEngine(
			database,
			&config.Config{},
			WithJobRuntime(mockJob),
			WithHooks(mockHook),
		)

		clusterID := createPipelineTestCluster(t, database, "hook-fail-cluster")
		stages := []models.StageDefinition{
			{ID: "stage-1", Name: "Stage 1", Type: models.StageTypeAWXJob, Config: models.StageConfig{AWXTemplateID: 1}},
		}
		template := createPipelineTestTemplate(t, database, "Hook Fail Pipeline", stages)

		exec, err := engine.StartExecution(context.Background(), template.ID, clusterID, nil, 1, nil, 0, nil, false, true)
		require.NoError(t, err)

		assert.True(t, waitForPipelineExecutionStatus(t, database, exec.ID, models.ExecutionStatusFailed, 5*time.Second))
		assert.Equal(t, 1, mockHook.BeforeCount)
		assert.Equal(t, 0, mockHook.AfterCount) // BeforeFailed, After not called

		// 验证阶段状态
		var stageRun models.StageRun
		database.Where("execution_id = ? AND stage_id = ?", exec.ID, "stage-1").First(&stageRun)
		assert.Equal(t, models.StageRunStatusFailed, stageRun.Status)
	})

	t.Run("AfterHook失败导致任务失败", func(t *testing.T) {
		mockHook := &MockHook{AfterError: fmt.Errorf("post-check failed")}
		engine := NewPipelineEngine(
			database,
			&config.Config{},
			WithJobRuntime(mockJob),
			// 注意：这里需要配置 JobRuntime 否则 AWX Job 会因为无 Runtime 而直接失败，但我们测试的是 Hook 失败
			WithJobRuntime(mockJob),
			WithHooks(mockHook),
		)

		clusterID := createPipelineTestCluster(t, database, "hook-after-fail-cluster")
		stages := []models.StageDefinition{
			{ID: "stage-1", Name: "Stage 1", Type: models.StageTypeAWXJob, Config: models.StageConfig{AWXTemplateID: 1}},
		}
		template := createPipelineTestTemplate(t, database, "Hook After Fail Pipeline", stages)

		exec, err := engine.StartExecution(context.Background(), template.ID, clusterID, nil, 1, nil, 0, nil, false, true)
		require.NoError(t, err)

		assert.True(t, waitForPipelineExecutionStatus(t, database, exec.ID, models.ExecutionStatusFailed, 5*time.Second))
		assert.Equal(t, 1, mockHook.BeforeCount)
		assert.Equal(t, 1, mockHook.AfterCount)

		var stageRun models.StageRun
		database.Where("execution_id = ? AND stage_id = ?", exec.ID, "stage-1").First(&stageRun)
		assert.Equal(t, models.StageRunStatusFailed, stageRun.Status)
	})
}

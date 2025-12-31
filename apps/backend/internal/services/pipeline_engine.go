package services

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"robusta-web/backend/internal/config"
	"robusta-web/backend/internal/db"
	"robusta-web/backend/internal/logger"
	"robusta-web/backend/internal/models"
	"robusta-web/backend/internal/pkg/awx"

	"go.uber.org/zap"
	"gorm.io/datatypes"
)

// PipelineEngine 流水线执行引擎
type PipelineEngine struct {
	db        *db.Database
	config    *config.Config
	awxClient *awx.Client
	mu        sync.RWMutex
	running   map[uint64]context.CancelFunc // 执行中的流水线
}

// NewPipelineEngine 创建流水线引擎
func NewPipelineEngine(database *db.Database, cfg *config.Config, awxClient *awx.Client) *PipelineEngine {
	return &PipelineEngine{
		db:        database,
		config:    cfg,
		awxClient: awxClient,
		running:   make(map[uint64]context.CancelFunc),
	}
}

// StartExecution 启动流水线执行
func (e *PipelineEngine) StartExecution(ctx context.Context, templateID uint64, clusterID uint64, params map[string]interface{}, triggeredBy uint64) (*models.PipelineExecution, error) {
	// 获取模板
	var template models.PipelineTemplate
	if err := e.db.First(&template, templateID).Error; err != nil {
		return nil, fmt.Errorf("获取流水线模板失败: %w", err)
	}

	// 获取集群信息
	var cluster models.Cluster
	if err := e.db.First(&cluster, clusterID).Error; err != nil {
		return nil, fmt.Errorf("获取集群信息失败: %w", err)
	}

	// 序列化参数
	paramsJSON, _ := json.Marshal(params)

	// 创建执行记录
	now := time.Now()
	execution := &models.PipelineExecution{
		PipelineTemplateID: templateID,
		ClusterID:          clusterID,
		ClusterName:        cluster.Name,
		Status:             models.ExecutionStatusPending,
		Parameters:         datatypes.JSON(paramsJSON),
		StartedAt:          &now,
		TriggeredBy:        triggeredBy,
	}

	if err := e.db.Create(execution).Error; err != nil {
		return nil, fmt.Errorf("创建执行记录失败: %w", err)
	}

	// 解析stages
	var stages []models.StageDefinition
	if err := json.Unmarshal(template.Stages, &stages); err != nil {
		return nil, fmt.Errorf("解析模板阶段失败: %w", err)
	}

	// 创建各阶段运行记录
	for _, stage := range stages {
		stageRun := &models.StageRun{
			ExecutionID: execution.ID,
			StageID:     stage.ID,
			StageName:   stage.Name,
			StageType:   stage.Type,
			Status:      models.StageRunStatusPending,
		}
		if err := e.db.Create(stageRun).Error; err != nil {
			return nil, fmt.Errorf("创建阶段记录失败: %w", err)
		}
	}

	// 异步执行流水线
	execCtx, cancel := context.WithCancel(context.Background())
	e.mu.Lock()
	e.running[execution.ID] = cancel
	e.mu.Unlock()

	go e.runPipeline(execCtx, execution.ID)

	logger.L().Info("流水线已启动",
		zap.Uint64("execution_id", execution.ID),
		zap.Uint64("template_id", templateID),
		zap.String("cluster", cluster.Name))

	return execution, nil
}

// runPipeline 执行流水线主循环
func (e *PipelineEngine) runPipeline(ctx context.Context, executionID uint64) {
	defer e.cleanupExecution(executionID)

	// 更新状态为运行中
	e.updateExecutionStatus(executionID, models.ExecutionStatusRunning, "")

	// 获取执行记录和模板
	var execution models.PipelineExecution
	if err := e.db.Preload("Template").First(&execution, executionID).Error; err != nil {
		e.failExecution(executionID, fmt.Sprintf("获取执行记录失败: %v", err))
		return
	}

	// 解析stages
	var stages []models.StageDefinition
	if err := json.Unmarshal(execution.Template.Stages, &stages); err != nil {
		e.failExecution(executionID, fmt.Sprintf("解析阶段失败: %v", err))
		return
	}

	// 解析运行时参数
	var params map[string]interface{}
	_ = json.Unmarshal(execution.Parameters, &params)

	// 按顺序执行各阶段
	for _, stage := range stages {
		select {
		case <-ctx.Done():
			e.updateExecutionStatus(executionID, models.ExecutionStatusCanceled, "用户取消")
			return
		default:
		}

		// 更新当前阶段
		e.db.Model(&models.PipelineExecution{}).Where("id = ?", executionID).
			Update("current_stage_id", stage.ID)

		// 执行阶段
		result := e.executeStage(ctx, executionID, stage, params, execution.ClusterName)

		if !result.Success {
			// 根据失败策略处理
			switch stage.OnFailure {
			case models.FailureStrategyAbort:
				e.failExecution(executionID, result.Error)
				return
			case models.FailureStrategyRollback:
				e.triggerRollback(ctx, executionID, stages)
				return
			case models.FailureStrategyPause:
				e.pauseExecution(executionID, result.Error)
				return
			case models.FailureStrategyContinue:
				// 继续下一阶段
				continue
			}
		}

		// 如果是暂停状态，等待恢复
		if result.NeedsPause {
			e.pauseExecution(executionID, "等待人工审批")
			// 等待恢复信号
			if !e.waitForResume(ctx, executionID) {
				return
			}
		}
	}

	// 全部完成
	e.completeExecution(executionID)
}

// StageResult 阶段执行结果
type StageResult struct {
	Success    bool
	NeedsPause bool
	Error      string
	Output     interface{}
}

// executeStage 执行单个阶段
func (e *PipelineEngine) executeStage(ctx context.Context, executionID uint64, stage models.StageDefinition, params map[string]interface{}, clusterName string) StageResult {
	// 获取对应的StageRun记录
	var stageRun models.StageRun
	e.db.Where("execution_id = ? AND stage_id = ?", executionID, stage.ID).First(&stageRun)

	// 更新为运行中
	now := time.Now()
	stageRun.Status = models.StageRunStatusRunning
	stageRun.StartedAt = &now
	e.db.Save(&stageRun)

	logger.L().Info("开始执行阶段",
		zap.Uint64("execution_id", executionID),
		zap.String("stage_id", stage.ID),
		zap.String("stage_name", stage.Name),
		zap.String("stage_type", string(stage.Type)))

	var result StageResult

	switch stage.Type {
	case models.StageTypeAWXJob:
		result = e.executeAWXJob(ctx, &stageRun, stage.Config, params, clusterName)
	case models.StageTypeManualGate:
		result = e.executeManualGate(ctx, &stageRun, stage.Config)
	case models.StageTypeDelay:
		result = e.executeDelay(ctx, &stageRun, stage.Config)
	case models.StageTypePreCheck, models.StageTypePostCheck:
		result = e.executeCheck(ctx, &stageRun, stage.Config)
	case models.StageTypeCondition:
		// TODO: implement condition
		result = StageResult{Success: true}
	case models.StageTypeRollback:
		// Rollback stages are skipped during normal execution
		result = StageResult{Success: true}
	default:
		result = StageResult{Success: true}
	}

	// 更新阶段状态
	completed := time.Now()
	stageRun.CompletedAt = &completed
	if result.Success {
		stageRun.Status = models.StageRunStatusSuccessful
	} else if result.NeedsPause {
		stageRun.Status = models.StageRunStatusWaitingApproval
	} else {
		stageRun.Status = models.StageRunStatusFailed
		stageRun.ErrorMessage = &result.Error
	}
	e.db.Save(&stageRun)

	return result
}

// executeAWXJob 执行AWX Job
func (e *PipelineEngine) executeAWXJob(ctx context.Context, stageRun *models.StageRun, config models.StageConfig, params map[string]interface{}, clusterName string) StageResult {
	// 合并参数
	extraVars := make(map[string]interface{})
	for k, v := range config.ExtraVars {
		extraVars[k] = v
	}
	for k, v := range params {
		extraVars[k] = v
	}
	extraVars["cluster_name"] = clusterName

	// 准备请求
	req := awx.JobLaunchRequest{
		ExtraVars: extraVars,
		Limit:     config.Limit,
	}

	// Dry Run模式
	if config.DryRun {
		req.JobType = "check"
		req.Diff = true
	}

	// 启动Job
	launchResp, err := e.awxClient.LaunchJob(ctx, config.AWXTemplateID, req)
	if err != nil {
		return StageResult{Success: false, Error: fmt.Sprintf("启动AWX Job失败: %v", err)}
	}

	// 记录Job ID
	stageRun.AWXJobID = &launchResp.Job
	e.db.Save(stageRun)

	// 等待Job完成
	job, err := e.awxClient.WaitForJob(ctx, launchResp.Job, 5*time.Second)
	if err != nil {
		return StageResult{Success: false, Error: fmt.Sprintf("等待AWX Job失败: %v", err)}
	}

	stageRun.AWXJobStatus = job.Status
	e.db.Save(stageRun)

	if !awx.IsJobSuccessful(job.Status) {
		return StageResult{Success: false, Error: fmt.Sprintf("AWX Job失败: %s", job.Status)}
	}

	return StageResult{Success: true}
}

// executeManualGate 执行人工审批门
func (e *PipelineEngine) executeManualGate(ctx context.Context, stageRun *models.StageRun, config models.StageConfig) StageResult {
	stageRun.Status = models.StageRunStatusWaitingApproval
	e.db.Save(stageRun)

	return StageResult{Success: true, NeedsPause: true}
}

// executeDelay 执行延时
func (e *PipelineEngine) executeDelay(ctx context.Context, stageRun *models.StageRun, config models.StageConfig) StageResult {
	delay := time.Duration(config.DelaySeconds) * time.Second

	select {
	case <-ctx.Done():
		return StageResult{Success: false, Error: "延时被取消"}
	case <-time.After(delay):
		return StageResult{Success: true}
	}
}

// executeCheck 执行检查 (Pre/Post)
func (e *PipelineEngine) executeCheck(ctx context.Context, stageRun *models.StageRun, config models.StageConfig) StageResult {
	// TODO: 实现Prometheus查询检查
	// 目前简单通过
	return StageResult{Success: true}
}

// PauseExecution 暂停执行 (供外部API调用)
func (e *PipelineEngine) PauseExecution(executionID uint64) error {
	e.mu.RLock()
	cancel, exists := e.running[executionID]
	e.mu.RUnlock()

	if !exists {
		return fmt.Errorf("执行 %d 不存在或已结束", executionID)
	}

	cancel()
	e.pauseExecution(executionID, "用户手动暂停")
	return nil
}

// ResumeExecution 恢复执行
func (e *PipelineEngine) ResumeExecution(ctx context.Context, executionID uint64, approvedBy uint64, notes string) error {
	var execution models.PipelineExecution
	if err := e.db.First(&execution, executionID).Error; err != nil {
		return fmt.Errorf("获取执行记录失败: %w", err)
	}

	if execution.Status != models.ExecutionStatusPaused {
		return fmt.Errorf("执行 %d 不在暂停状态", executionID)
	}

	// 更新当前等待审批的阶段
	var stageRun models.StageRun
	if err := e.db.Where("execution_id = ? AND status = ?", executionID, models.StageRunStatusWaitingApproval).First(&stageRun).Error; err == nil {
		stageRun.ApprovedBy = &approvedBy
		stageRun.ApprovalNotes = notes
		stageRun.Status = models.StageRunStatusSuccessful
		e.db.Save(&stageRun)
	}

	// 重新启动流水线 (从当前阶段继续)
	execCtx, cancel := context.WithCancel(context.Background())
	e.mu.Lock()
	e.running[executionID] = cancel
	e.mu.Unlock()

	go e.resumePipeline(execCtx, executionID)

	logger.L().Info("流水线已恢复", zap.Uint64("execution_id", executionID), zap.Uint64("approved_by", approvedBy))
	return nil
}

// CancelExecution 取消执行
func (e *PipelineEngine) CancelExecution(executionID uint64) error {
	e.mu.Lock()
	cancel, exists := e.running[executionID]
	if exists {
		cancel()
		delete(e.running, executionID)
	}
	e.mu.Unlock()

	e.updateExecutionStatus(executionID, models.ExecutionStatusCanceled, "用户取消")
	return nil
}

// RollbackExecution 触发回滚
func (e *PipelineEngine) RollbackExecution(ctx context.Context, executionID uint64) error {
	var execution models.PipelineExecution
	if err := e.db.Preload("Template").First(&execution, executionID).Error; err != nil {
		return fmt.Errorf("获取执行记录失败: %w", err)
	}

	var stages []models.StageDefinition
	if err := json.Unmarshal(execution.Template.Stages, &stages); err != nil {
		return fmt.Errorf("解析阶段失败: %w", err)
	}

	go e.triggerRollback(ctx, executionID, stages)
	return nil
}

// === 内部辅助方法 ===

func (e *PipelineEngine) updateExecutionStatus(executionID uint64, status models.ExecutionStatus, errMsg string) {
	updates := map[string]interface{}{"status": status}
	if errMsg != "" {
		updates["error_message"] = errMsg
	}
	if status == models.ExecutionStatusSuccessful || status == models.ExecutionStatusFailed ||
		status == models.ExecutionStatusCanceled || status == models.ExecutionStatusRolledBack {
		now := time.Now()
		updates["completed_at"] = now
	}
	e.db.Model(&models.PipelineExecution{}).Where("id = ?", executionID).Updates(updates)
}

func (e *PipelineEngine) failExecution(executionID uint64, errMsg string) {
	e.updateExecutionStatus(executionID, models.ExecutionStatusFailed, errMsg)
	logger.L().Error("流水线执行失败", zap.Uint64("execution_id", executionID), zap.String("error", errMsg))
}

func (e *PipelineEngine) pauseExecution(executionID uint64, reason string) {
	e.updateExecutionStatus(executionID, models.ExecutionStatusPaused, reason)
	logger.L().Info("流水线已暂停", zap.Uint64("execution_id", executionID), zap.String("reason", reason))
}

func (e *PipelineEngine) completeExecution(executionID uint64) {
	e.updateExecutionStatus(executionID, models.ExecutionStatusSuccessful, "")
	logger.L().Info("流水线执行完成", zap.Uint64("execution_id", executionID))
}

func (e *PipelineEngine) cleanupExecution(executionID uint64) {
	e.mu.Lock()
	delete(e.running, executionID)
	e.mu.Unlock()
}

func (e *PipelineEngine) waitForResume(ctx context.Context, executionID uint64) bool {
	// 简单实现：轮询检查状态
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return false
		case <-ticker.C:
			var execution models.PipelineExecution
			e.db.First(&execution, executionID)
			if execution.Status == models.ExecutionStatusRunning {
				return true
			}
			if execution.Status == models.ExecutionStatusCanceled {
				return false
			}
		}
	}
}

func (e *PipelineEngine) resumePipeline(ctx context.Context, executionID uint64) {
	defer e.cleanupExecution(executionID)

	// 更新状态
	e.updateExecutionStatus(executionID, models.ExecutionStatusRunning, "")

	// 获取执行记录
	var execution models.PipelineExecution
	if err := e.db.Preload("Template").First(&execution, executionID).Error; err != nil {
		e.failExecution(executionID, fmt.Sprintf("获取执行记录失败: %v", err))
		return
	}

	// 解析stages并找到当前位置
	var stages []models.StageDefinition
	if err := json.Unmarshal(execution.Template.Stages, &stages); err != nil {
		e.failExecution(executionID, fmt.Sprintf("解析阶段失败: %v", err))
		return
	}

	var params map[string]interface{}
	_ = json.Unmarshal(execution.Parameters, &params)

	// 找到当前阶段的下一个
	startIdx := 0
	for i, stage := range stages {
		if stage.ID == execution.CurrentStageID {
			startIdx = i + 1
			break
		}
	}

	// 从当前阶段继续执行
	for i := startIdx; i < len(stages); i++ {
		stage := stages[i]

		select {
		case <-ctx.Done():
			e.updateExecutionStatus(executionID, models.ExecutionStatusCanceled, "用户取消")
			return
		default:
		}

		e.db.Model(&models.PipelineExecution{}).Where("id = ?", executionID).
			Update("current_stage_id", stage.ID)

		result := e.executeStage(ctx, executionID, stage, params, execution.ClusterName)

		if !result.Success {
			switch stage.OnFailure {
			case models.FailureStrategyAbort:
				e.failExecution(executionID, result.Error)
				return
			case models.FailureStrategyRollback:
				e.triggerRollback(ctx, executionID, stages)
				return
			case models.FailureStrategyPause:
				e.pauseExecution(executionID, result.Error)
				return
			case models.FailureStrategyContinue:
				// Continue to next stage despite failure
				logger.L().Warn("Stage failed but strategy is continue",
					zap.String("stage", stage.Name),
					zap.String("error", result.Error))
			}
		}

		if result.NeedsPause {
			e.pauseExecution(executionID, "等待人工审批")
			if !e.waitForResume(ctx, executionID) {
				return
			}
		}
	}

	e.completeExecution(executionID)
}

func (e *PipelineEngine) triggerRollback(ctx context.Context, executionID uint64, stages []models.StageDefinition) {
	e.updateExecutionStatus(executionID, models.ExecutionStatusRollingBack, "")

	// 查找回滚阶段
	for _, stage := range stages {
		if stage.Type == models.StageTypeRollback {
			var params map[string]interface{}
			var execution models.PipelineExecution
			e.db.First(&execution, executionID)
			_ = json.Unmarshal(execution.Parameters, &params)

			result := e.executeStage(ctx, executionID, stage, params, execution.ClusterName)
			if !result.Success {
				e.failExecution(executionID, fmt.Sprintf("回滚失败: %s", result.Error))
				return
			}
		}
	}

	e.updateExecutionStatus(executionID, models.ExecutionStatusRolledBack, "")
	logger.L().Info("流水线已回滚", zap.Uint64("execution_id", executionID))
}

// GetExecution 获取执行详情
func (e *PipelineEngine) GetExecution(executionID uint64) (*models.PipelineExecution, error) {
	var execution models.PipelineExecution
	if err := e.db.Preload("Template").Preload("StageRuns").First(&execution, executionID).Error; err != nil {
		return nil, fmt.Errorf("获取执行记录失败: %w", err)
	}
	return &execution, nil
}

// GetExecutionHistory 获取集群的执行历史
func (e *PipelineEngine) GetExecutionHistory(clusterID uint64, limit int) ([]models.PipelineExecution, error) {
	var executions []models.PipelineExecution
	if err := e.db.Where("cluster_id = ?", clusterID).
		Preload("Template").
		Order("created_at DESC").
		Limit(limit).
		Find(&executions).Error; err != nil {
		return nil, fmt.Errorf("获取执行历史失败: %w", err)
	}
	return executions, nil
}

package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"robusta-web/backend/internal/config"
	"robusta-web/backend/internal/db"
	"robusta-web/backend/internal/logger"
	"robusta-web/backend/internal/models"

	"go.uber.org/zap"
	"gorm.io/datatypes"
)

// PipelineEngine 流水线执行引擎
// 通过 JobRuntime 和 MetricsRuntime 接口实现可插拔的运行时支持
type PipelineEngine struct {
	db             *db.Database
	config         *config.Config
	jobRuntime     JobRuntime     // 任务运行时（AWX、SSH、K8s Job 等）
	metricsRuntime MetricsRuntime // 指标运行时（Prometheus 等）
	mu             sync.RWMutex
	running        map[uint64]context.CancelFunc // 执行中的流水线
	hooks          []PipelineHook                // 注册的钩子
}

// PipelineEngineOption 引擎配置选项
type PipelineEngineOption func(*PipelineEngine)

// WithJobRuntime 设置任务运行时
func WithJobRuntime(runtime JobRuntime) PipelineEngineOption {
	return func(e *PipelineEngine) {
		e.jobRuntime = runtime
	}
}

// WithMetricsRuntime 设置指标运行时
func WithMetricsRuntime(runtime MetricsRuntime) PipelineEngineOption {
	return func(e *PipelineEngine) {
		e.metricsRuntime = runtime
	}
}

// WithHooks 添加钩子
func WithHooks(hooks ...PipelineHook) PipelineEngineOption {
	return func(e *PipelineEngine) {
		e.hooks = append(e.hooks, hooks...)
	}
}

// NewPipelineEngine 创建流水线引擎
func NewPipelineEngine(database *db.Database, cfg *config.Config, opts ...PipelineEngineOption) *PipelineEngine {
	engine := &PipelineEngine{
		db:      database,
		config:  cfg,
		running: make(map[uint64]context.CancelFunc),
	}

	for _, opt := range opts {
		opt(engine)
	}

	return engine
}

// SetJobRuntime 设置任务运行时（运行时可更换）
func (e *PipelineEngine) SetJobRuntime(runtime JobRuntime) {
	e.jobRuntime = runtime
}

// SetMetricsRuntime 设置指标运行时（运行时可更换）
func (e *PipelineEngine) SetMetricsRuntime(runtime MetricsRuntime) {
	e.metricsRuntime = runtime
}

// GetJobRuntime 获取当前任务运行时
func (e *PipelineEngine) GetJobRuntime() JobRuntime {
	return e.jobRuntime
}

// RegisterHook 注册钩子
func (e *PipelineEngine) RegisterHook(hook PipelineHook) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.hooks = append(e.hooks, hook)
}

// ensureDB 检查数据库是否已初始化
func (e *PipelineEngine) ensureDB() error {
	if e == nil || e.db == nil {
		return fmt.Errorf("database not initialized")
	}
	return nil
}

// validateParameters 验证执行参数
func (e *PipelineEngine) validateParameters(stages []models.StageDefinition, params map[string]interface{}) error {
	for _, stage := range stages {
		if stage.Type != models.StageTypeAWXJob {
			continue
		}

		for _, param := range stage.Config.Parameters {
			value, exists := params[param.Name]

			// 检查必填参数
			if param.Required && !exists {
				// 如果有默认值且未提供，使用默认值
				if param.DefaultValue != "" {
					continue
				}
				return fmt.Errorf("阶段 [%s] 缺少必填参数: %s", stage.Name, param.Label)
			}

			if !exists {
				continue
			}

			// 检查单选/多选的值是否在可选范围内
			if param.InputType == "select" && len(param.Options) > 0 {
				strValue, ok := value.(string)
				if !ok {
					return fmt.Errorf("参数 [%s] 类型错误，应为字符串", param.Label)
				}
				valid := false
				for _, opt := range param.Options {
					if opt == strValue {
						valid = true
						break
					}
				}
				if !valid {
					return fmt.Errorf("参数 [%s] 的值 [%s] 不在可选范围内", param.Label, strValue)
				}
			}

			if param.InputType == "multi_select" && len(param.Options) > 0 {
				// 多选值可能是字符串数组
				var values []string
				switch v := value.(type) {
				case []string:
					values = v
				case []interface{}:
					for _, item := range v {
						if s, ok := item.(string); ok {
							values = append(values, s)
						}
					}
				}

				for _, val := range values {
					valid := false
					for _, opt := range param.Options {
						if opt == val {
							valid = true
							break
						}
					}
					if !valid {
						return fmt.Errorf("参数 [%s] 的值 [%s] 不在可选范围内", param.Label, val)
					}
				}
			}
		}
	}
	return nil
}

// StartExecution 启动流水线执行
func (e *PipelineEngine) StartExecution(ctx context.Context, templateID uint64, clusterID uint64, params map[string]interface{}, triggeredBy uint64, targetNodes []string, batchSize int, explicitBatches [][]string, pauseBetweenBatches bool, autoStart bool) (*models.PipelineExecution, error) {
	if err := e.ensureDB(); err != nil {
		return nil, err
	}
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

	// 解析stages用于验证参数
	var stages []models.StageDefinition
	if err := json.Unmarshal(template.Stages, &stages); err != nil {
		return nil, fmt.Errorf("解析模板阶段失败: %w", err)
	}

	// 验证参数绑定
	if err := e.validateParameters(stages, params); err != nil {
		return nil, err
	}

	// 处理批次生成 EffectiveStages
	var effectiveStages []models.StageDefinition

	// 优先使用显式批次配置
	if len(explicitBatches) > 0 {
		for _, stage := range stages {
			if stage.Type == models.StageTypeAWXJob {
				for i, batchNodes := range explicitBatches {
					if len(batchNodes) == 0 {
						continue
					}
					isLastBatch := i == len(explicitBatches)-1

					// 创建新的 StageDefinition
					newStage := stage
					newStage.ID = fmt.Sprintf("%s_batch_%d", stage.ID, i+1)
					newStage.Name = fmt.Sprintf("%s (Batch %d)", stage.Name, i+1)

					// 设置 limit 参数
					if newStage.Config.ExtraVars == nil {
						newStage.Config.ExtraVars = make(map[string]string)
					}
					newStage.Config.ExtraVars["limit"] = strings.Join(batchNodes, ",")

					effectiveStages = append(effectiveStages, newStage)

					// 如果需要批次间暂停，且不是最后一批，插入 Manual Gate
					if pauseBetweenBatches && !isLastBatch {
						gateStage := models.StageDefinition{
							ID:   fmt.Sprintf("%s_gate_%d", stage.ID, i+1),
							Name: fmt.Sprintf("Pause after Batch %d", i+1),
							Type: models.StageTypeManualGate,
							Config: models.StageConfig{
								TimeoutMinutes: 60 * 24, // 默认一天
							},
							OnFailure: models.FailureStrategyPause,
						}
						effectiveStages = append(effectiveStages, gateStage)
					}
				}
			} else {
				effectiveStages = append(effectiveStages, stage)
			}
		}
	} else if batchSize > 0 && len(targetNodes) > 0 {
		for _, stage := range stages {
			// 只对 AWX Job 进行批次拆分
			if stage.Type == models.StageTypeAWXJob {
				// 分批
				for i := 0; i < len(targetNodes); i += batchSize {
					end := i + batchSize
					if end > len(targetNodes) {
						end = len(targetNodes)
					}
					batchNodes := targetNodes[i:end]
					isLastBatch := end == len(targetNodes)

					// 创建新的 StageDefinition
					newStage := stage
					newStage.ID = fmt.Sprintf("%s_batch_%d", stage.ID, i/batchSize+1)
					newStage.Name = fmt.Sprintf("%s (Batch %d)", stage.Name, i/batchSize+1)

					// 设置 limit 参数
					if newStage.Config.ExtraVars == nil {
						newStage.Config.ExtraVars = make(map[string]string)
					}
					newStage.Config.ExtraVars["limit"] = strings.Join(batchNodes, ",")

					effectiveStages = append(effectiveStages, newStage)

					// 如果需要批次间暂停，且不是最后一批，插入 Manual Gate
					if pauseBetweenBatches && !isLastBatch {
						gateStage := models.StageDefinition{
							ID:   fmt.Sprintf("%s_gate_%d", stage.ID, i/batchSize+1),
							Name: fmt.Sprintf("Pause after Batch %d", i/batchSize+1),
							Type: models.StageTypeManualGate,
							Config: models.StageConfig{
								TimeoutMinutes: 60 * 24, // 默认一天
							},
							OnFailure: models.FailureStrategyPause, // 审批拒绝默认暂停? 其实Gate没有Failure
						}
						effectiveStages = append(effectiveStages, gateStage)
					}
				}
			} else {
				effectiveStages = append(effectiveStages, stage)
			}
		}
	}

	// 序列化参数
	paramsJSON, _ := json.Marshal(params)

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

	// 如果有 EffectiveStages，保存
	if len(effectiveStages) > 0 {
		effectiveStagesJSON, _ := json.Marshal(effectiveStages)
		execution.EffectiveStages = datatypes.JSON(effectiveStagesJSON)

		// 使用 EffectiveStages 替换 template stages 进行后续 StageRun 创建
		stages = effectiveStages
	}

	if err := e.db.Create(execution).Error; err != nil {
		return nil, fmt.Errorf("创建执行记录失败: %w", err)
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

		// 对于 AWX Job 阶段，在提交任务时就克隆模板并绑定 Inventory 和其他配置
		if stage.Type == models.StageTypeAWXJob && e.jobRuntime != nil && stage.Config.AWXTemplateID > 0 {
			// 构建克隆配置
			cloneConfig := CloneTemplateConfig{
				TemplateID:   stage.Config.AWXTemplateID,
				TemplateName: stage.Config.AWXTemplateName,
				ClusterName:  cluster.Name,
			}

			// 处理 extra_vars，添加 cluster_name
			if len(stage.Config.ExtraVars) > 0 {
				cloneConfig.ExtraVars = make(map[string]interface{})
				for k, v := range stage.Config.ExtraVars {
					cloneConfig.ExtraVars[k] = v
				}
			}

			clonedID, cloneErr := e.jobRuntime.PrepareClonedTemplate(ctx, cloneConfig)
			if cloneErr != nil {
				logger.L().Warn("克隆模板失败，将在执行时使用原模板",
					zap.Int("template_id", stage.Config.AWXTemplateID),
					zap.Error(cloneErr))
			} else {
				stageRun.ClonedTemplateID = &clonedID
				logger.L().Info("AWX 模板已克隆",
					zap.Uint64("execution_id", execution.ID),
					zap.String("stage_id", stage.ID),
					zap.Int("original_template_id", stage.Config.AWXTemplateID),
					zap.Int("cloned_template_id", clonedID))
			}
		}

		if err := e.db.Create(stageRun).Error; err != nil {
			return nil, fmt.Errorf("创建阶段记录失败: %w", err)
		}
	}

	// 如果设置了自动启动，则异步执行流水线
	if autoStart {
		execCtx, cancel := context.WithCancel(context.Background())
		e.mu.Lock()
		e.running[execution.ID] = cancel
		e.mu.Unlock()

		go e.runPipeline(execCtx, execution.ID)

		logger.L().Info("流水线已启动",
			zap.Uint64("execution_id", execution.ID),
			zap.Uint64("template_id", templateID),
			zap.String("cluster", cluster.Name))
	} else {
		logger.L().Info("流水线任务已创建（待执行）",
			zap.Uint64("execution_id", execution.ID),
			zap.Uint64("template_id", templateID),
			zap.String("cluster", cluster.Name))
	}

	return execution, nil
}

// RunPendingExecution 启动处于等待状态的流水线
func (e *PipelineEngine) RunPendingExecution(ctx context.Context, executionID uint64) error {
	if err := e.ensureDB(); err != nil {
		return err
	}
	var execution models.PipelineExecution
	if err := e.db.First(&execution, executionID).Error; err != nil {
		return fmt.Errorf("获取执行记录失败: %w", err)
	}

	if execution.Status != models.ExecutionStatusPending {
		return fmt.Errorf("执行 %d 不在等待状态 (当前状态: %s)", executionID, execution.Status)
	}

	// 启动流水线
	execCtx, cancel := context.WithCancel(context.Background())
	e.mu.Lock()
	e.running[execution.ID] = cancel
	e.mu.Unlock()

	go e.runPipeline(execCtx, execution.ID)

	logger.L().Info("流水线已手动触发启动", zap.Uint64("execution_id", executionID))
	return nil
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
	// 优先使用 EffectiveStages
	if len(execution.EffectiveStages) > 0 {
		if err := json.Unmarshal(execution.EffectiveStages, &stages); err != nil {
			e.failExecution(executionID, fmt.Sprintf("解析EffectiveStages失败: %v", err))
			return
		}
	} else {
		// 回退到 Template Stages
		if err := json.Unmarshal(execution.Template.Stages, &stages); err != nil {
			e.failExecution(executionID, fmt.Sprintf("解析阶段失败: %v", err))
			return
		}
	}

	// 解析运行时参数
	var params map[string]interface{}
	_ = json.Unmarshal(execution.Parameters, &params)

	// 按顺序执行各阶段
	var lastResult *StageResult
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

		// 执行 Pre-Hooks
		hookCtx := PipelineHookContext{
			ExecutionID:    executionID,
			Stage:          stage,
			PreviousResult: lastResult,
		}

		preHookFailed := false
		for _, hook := range e.hooks {
			if err := hook.BeforeStage(ctx, hookCtx); err != nil {
				errMsg := fmt.Sprintf("Hook [%s] pre-check failed: %v", hook.Name(), err)
				logger.L().Error(errMsg, zap.Uint64("execution_id", executionID), zap.String("stage", stage.Name))

				// 更新阶段状态为失败
				now := time.Now()
				e.db.Model(&models.StageRun{}).Where("execution_id = ? AND stage_id = ?", executionID, stage.ID).
					Updates(map[string]interface{}{
						"status":        models.StageRunStatusFailed,
						"error_message": errMsg,
						"completed_at":  now,
					})

				e.failExecution(executionID, errMsg)
				preHookFailed = true
				break
			}
		}
		if preHookFailed {
			return
		}

		// 执行阶段
		result := e.executeStage(ctx, executionID, stage, params, execution.ClusterName)

		// 执行 Post-Hooks
		for _, hook := range e.hooks {
			if err := hook.AfterStage(ctx, hookCtx, &result); err != nil {
				result.Success = false
				result.Error = fmt.Sprintf("Hook [%s] post-check failed: %v", hook.Name(), err)

				// 更新阶段状态为失败
				now := time.Now()
				e.db.Model(&models.StageRun{}).Where("execution_id = ? AND stage_id = ?", executionID, stage.ID).
					Updates(map[string]interface{}{
						"status":        models.StageRunStatusFailed,
						"error_message": result.Error,
						"completed_at":  now,
					})
			}
		}

		lastResult = &result

		if !result.Success {
			// 如果是因为上下文取消导致的失败，确保状态为已取消
			if ctx.Err() == context.Canceled {
				e.updateExecutionStatus(executionID, models.ExecutionStatusCanceled, "用户取消")
				return
			}

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
			default:
				// 默认为终止
				e.failExecution(executionID, result.Error)
				return
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

	// 如果已经成功或跳过，直接返回
	if stageRun.Status == models.StageRunStatusSuccessful || stageRun.Status == models.StageRunStatusSkipped {
		logger.L().Info("阶段已完成，跳过",
			zap.Uint64("execution_id", executionID),
			zap.String("stage_name", stage.Name))
		return StageResult{Success: true}
	}

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

// executeAWXJob 执行任务 (通过 JobRuntime 接口)
func (e *PipelineEngine) executeAWXJob(ctx context.Context, stageRun *models.StageRun, config models.StageConfig, params map[string]interface{}, clusterName string) StageResult {
	if e.jobRuntime == nil {
		return StageResult{Success: false, Error: "任务运行时未配置"}
	}

	// 合并参数
	extraVars := make(map[string]interface{})
	for k, v := range config.ExtraVars {
		extraVars[k] = v
	}
	for k, v := range params {
		extraVars[k] = v
	}
	extraVars["cluster_name"] = clusterName

	// 确定使用的模板 ID（优先使用克隆模板）
	templateID := config.AWXTemplateID
	if stageRun.ClonedTemplateID != nil && *stageRun.ClonedTemplateID > 0 {
		templateID = *stageRun.ClonedTemplateID
		logger.L().Debug("使用克隆模板执行",
			zap.Int("cloned_template_id", templateID),
			zap.Int("original_template_id", config.AWXTemplateID))
	}

	// 构建任务配置
	jobConfig := JobConfig{
		TemplateID:   templateID,
		TemplateName: config.AWXTemplateName,
		ExtraVars:    extraVars,
		DryRun:       config.DryRun,
		ClusterName:  clusterName,
	}

	// 处理 limit 参数
	if limit, ok := extraVars["limit"].(string); ok {
		jobConfig.Limit = limit
	}

	// 启动任务
	handle, err := e.jobRuntime.LaunchJob(ctx, jobConfig)
	if err != nil {
		// 启动失败也要清理克隆模板
		e.cleanupClonedTemplate(stageRun)
		return StageResult{Success: false, Error: fmt.Sprintf("启动任务失败: %v", err)}
	}

	// 记录 Job ID
	stageRun.AWXJobID = &handle.JobID
	e.db.Save(stageRun)

	// 等待任务完成
	result, err := e.jobRuntime.WaitForJob(ctx, handle, 5*time.Second)
	if err != nil {
		e.cleanupClonedTemplate(stageRun)
		return StageResult{Success: false, Error: fmt.Sprintf("等待任务失败: %v", err)}
	}

	stageRun.AWXJobStatus = result.Status
	// 尝试将输出解析为 JSON，如果失败则作为普通字符串包装
	var outputJSON interface{}
	if err := json.Unmarshal([]byte(result.Output), &outputJSON); err != nil {
		outputJSON = map[string]string{"log": result.Output}
	}
	outputBytes, _ := json.Marshal(outputJSON)
	stageRun.Output = datatypes.JSON(outputBytes)
	e.db.Save(stageRun)

	// 任务完成后清理克隆模板
	e.cleanupClonedTemplate(stageRun)

	if !result.Success {
		return StageResult{Success: false, Error: fmt.Sprintf("任务执行失败: %s", result.Status), Output: outputJSON}
	}

	return StageResult{Success: true, Output: outputJSON}
}

// cleanupClonedTemplate 清理克隆模板（在任务完成后调用）
func (e *PipelineEngine) cleanupClonedTemplate(stageRun *models.StageRun) {
	if stageRun.ClonedTemplateID == nil || *stageRun.ClonedTemplateID <= 0 {
		return
	}

	if e.jobRuntime == nil {
		return
	}

	// 使用独立的 context 清理，避免被父 context 取消
	cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := e.jobRuntime.CleanupClonedTemplate(cleanupCtx, *stageRun.ClonedTemplateID); err != nil {
		logger.L().Warn("清理克隆模板失败",
			zap.Uint64("stage_run_id", stageRun.ID),
			zap.Int("cloned_template_id", *stageRun.ClonedTemplateID),
			zap.Error(err))
	} else {
		logger.L().Debug("克隆模板已清理",
			zap.Uint64("stage_run_id", stageRun.ID),
			zap.Int("cloned_template_id", *stageRun.ClonedTemplateID))
	}
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

// executeCheck 执行检查 (Pre/Post) - 通过 MetricsRuntime 接口
func (e *PipelineEngine) executeCheck(ctx context.Context, stageRun *models.StageRun, config models.StageConfig) StageResult {
	if config.PromQuery == "" {
		return StageResult{Success: true}
	}

	// 1. 获取执行记录
	var execution models.PipelineExecution
	if err := e.db.First(&execution, stageRun.ExecutionID).Error; err != nil {
		return StageResult{Success: false, Error: fmt.Sprintf("获取执行记录失败: %v", err)}
	}

	// 2. 获取集群信息
	var cluster models.Cluster
	if err := e.db.First(&cluster, execution.ClusterID).Error; err != nil {
		return StageResult{Success: false, Error: fmt.Sprintf("获取集群信息失败: %v", err)}
	}

	if cluster.PrometheusURL == "" {
		return StageResult{Success: false, Error: "集群未配置 Prometheus URL"}
	}

	// 3. 查询指标
	var currentVal float64
	var err error

	if e.metricsRuntime != nil {
		// 使用配置的 MetricsRuntime
		currentVal, err = e.metricsRuntime.Query(ctx, cluster.PrometheusURL, config.PromQuery)
	} else {
		// 回退到内置 Prometheus 查询（向后兼容）
		currentVal, err = e.queryPrometheusDefault(ctx, cluster.PrometheusURL, config.PromQuery)
	}

	if err != nil {
		return StageResult{Success: false, Error: fmt.Sprintf("指标查询失败: %v", err)}
	}

	// 4. 如果没有配置 expected value，只要有数据就算成功
	if config.ExpectedValue == "" {
		return StageResult{
			Success: true,
			Output:  map[string]interface{}{"current": currentVal},
		}
	}

	expectedVal, err := strconv.ParseFloat(config.ExpectedValue, 64)
	if err != nil {
		return StageResult{Success: false, Error: fmt.Sprintf("无法解析期望值: %s", config.ExpectedValue)}
	}

	// 5. 比较结果
	success := e.compareValues(currentVal, expectedVal, config.Operator)

	if !success {
		return StageResult{
			Success: false,
			Error:   fmt.Sprintf("检查失败: 当前值 %v (期望 %s %v)", currentVal, config.Operator, expectedVal),
		}
	}

	return StageResult{
		Success: true,
		Output:  map[string]interface{}{"current": currentVal},
	}
}

// compareValues 比较数值
func (e *PipelineEngine) compareValues(current, expected float64, operator string) bool {
	switch operator {
	case "eq", "==":
		return current == expected
	case "ne", "!=":
		return current != expected
	case "gt", ">":
		return current > expected
	case "lt", "<":
		return current < expected
	case "ge", ">=":
		return current >= expected
	case "le", "<=":
		return current <= expected
	default:
		return current == expected
	}
}

// queryPrometheusDefault 内置的 Prometheus 查询实现（向后兼容）
func (e *PipelineEngine) queryPrometheusDefault(ctx context.Context, endpoint, query string) (float64, error) {
	runtime := NewPrometheusRuntime(logger.L())
	return runtime.Query(ctx, endpoint, query)
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
	if err := e.ensureDB(); err != nil {
		return err
	}
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
	if err := e.ensureDB(); err != nil {
		return err
	}
	var execution models.PipelineExecution
	if err := e.db.Preload("Template").First(&execution, executionID).Error; err != nil {
		return fmt.Errorf("获取执行记录失败: %w", err)
	}

	var stages []models.StageDefinition

	// 优先使用 EffectiveStages
	if len(execution.EffectiveStages) > 0 {
		if err := json.Unmarshal(execution.EffectiveStages, &stages); err != nil {
			return fmt.Errorf("解析EffectiveStages失败: %w", err)
		}
	} else {
		if err := json.Unmarshal(execution.Template.Stages, &stages); err != nil {
			return fmt.Errorf("解析阶段失败: %w", err)
		}
	}

	go e.triggerRollback(ctx, executionID, stages)
	return nil
}

// === 内部辅助方法 ===

func (e *PipelineEngine) updateExecutionStatus(executionID uint64, status models.ExecutionStatus, errMsg string) {
	if e == nil || e.db == nil {
		logger.L().Error("数据库未初始化，无法更新执行状态", zap.Uint64("execution_id", executionID))
		return
	}
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
	// 优先使用 EffectiveStages
	if len(execution.EffectiveStages) > 0 {
		if err := json.Unmarshal(execution.EffectiveStages, &stages); err != nil {
			e.failExecution(executionID, fmt.Sprintf("解析EffectiveStages失败: %v", err))
			return
		}
	} else {
		if err := json.Unmarshal(execution.Template.Stages, &stages); err != nil {
			e.failExecution(executionID, fmt.Sprintf("解析阶段失败: %v", err))
			return
		}
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
	var lastResult *StageResult
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

		// 执行 Pre-Hooks
		hookCtx := PipelineHookContext{
			ExecutionID:    executionID,
			Stage:          stage,
			PreviousResult: lastResult,
		}

		preHookFailed := false
		for _, hook := range e.hooks {
			if err := hook.BeforeStage(ctx, hookCtx); err != nil {
				errMsg := fmt.Sprintf("Hook [%s] pre-check failed: %v", hook.Name(), err)
				logger.L().Error(errMsg, zap.Uint64("execution_id", executionID), zap.String("stage", stage.Name))

				// 更新阶段状态为失败
				now := time.Now()
				e.db.Model(&models.StageRun{}).Where("execution_id = ? AND stage_id = ?", executionID, stage.ID).
					Updates(map[string]interface{}{
						"status":        models.StageRunStatusFailed,
						"error_message": errMsg,
						"completed_at":  now,
					})

				e.failExecution(executionID, errMsg)
				preHookFailed = true
				break
			}
		}
		if preHookFailed {
			return
		}

		result := e.executeStage(ctx, executionID, stage, params, execution.ClusterName)

		// 执行 Post-Hooks
		for _, hook := range e.hooks {
			if err := hook.AfterStage(ctx, hookCtx, &result); err != nil {
				result.Success = false
				result.Error = fmt.Sprintf("Hook [%s] post-check failed: %v", hook.Name(), err)

				// 更新阶段状态为失败
				now := time.Now()
				e.db.Model(&models.StageRun{}).Where("execution_id = ? AND stage_id = ?", executionID, stage.ID).
					Updates(map[string]interface{}{
						"status":        models.StageRunStatusFailed,
						"error_message": result.Error,
						"completed_at":  now,
					})
			}
		}

		lastResult = &result

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
			default:
				e.failExecution(executionID, result.Error)
				return
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
	if err := e.ensureDB(); err != nil {
		return nil, err
	}
	var execution models.PipelineExecution
	if err := e.db.Preload("Template").Preload("StageRuns").First(&execution, executionID).Error; err != nil {
		return nil, fmt.Errorf("获取执行记录失败: %w", err)
	}
	return &execution, nil
}

// GetExecutionHistory 获取集群的执行历史
func (e *PipelineEngine) GetExecutionHistory(clusterID uint64, limit int) ([]models.PipelineExecution, error) {
	if err := e.ensureDB(); err != nil {
		return nil, err
	}
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

// ListTemplates 获取所有流水线模板
// ListTemplates 获取流水线模板列表 (支持分页和搜索)
func (e *PipelineEngine) ListTemplates(page, pageSize int, keyword string) ([]models.PipelineTemplate, int64, error) {
	if err := e.ensureDB(); err != nil {
		return nil, 0, err
	}

	var templates []models.PipelineTemplate
	var total int64

	query := e.db.Model(&models.PipelineTemplate{})

	if keyword != "" {
		query = query.Where("name LIKE ? OR description LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("获取模板数量失败: %w", err)
	}

	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Limit(pageSize).Offset(offset).Find(&templates).Error; err != nil {
		return nil, 0, fmt.Errorf("获取模板列表失败: %w", err)
	}

	return templates, total, nil
}

// GetTemplate 获取单个模板
func (e *PipelineEngine) GetTemplate(templateID uint64) (*models.PipelineTemplate, error) {
	if err := e.ensureDB(); err != nil {
		return nil, err
	}
	var template models.PipelineTemplate
	if err := e.db.First(&template, templateID).Error; err != nil {
		return nil, fmt.Errorf("获取模板失败: %w", err)
	}
	return &template, nil
}

// CreateTemplate 创建流水线模板
func (e *PipelineEngine) CreateTemplate(name, description string, stages []models.StageDefinition, createdBy uint64) (*models.PipelineTemplate, error) {
	if err := e.ensureDB(); err != nil {
		return nil, err
	}
	stagesJSON, err := json.Marshal(stages)
	if err != nil {
		return nil, fmt.Errorf("序列化阶段失败: %w", err)
	}

	template := &models.PipelineTemplate{
		Name:        name,
		Description: description,
		Stages:      datatypes.JSON(stagesJSON),
		CreatedBy:   createdBy,
	}

	if err := e.db.Create(template).Error; err != nil {
		return nil, fmt.Errorf("创建模板失败: %w", err)
	}

	return template, nil
}

// UpdateTemplate 更新流水线模板
func (e *PipelineEngine) UpdateTemplate(templateID uint64, name, description string, stages []models.StageDefinition) (*models.PipelineTemplate, error) {
	if err := e.ensureDB(); err != nil {
		return nil, err
	}

	var template models.PipelineTemplate
	if err := e.db.First(&template, templateID).Error; err != nil {
		return nil, fmt.Errorf("模板不存在: %w", err)
	}

	stagesJSON, err := json.Marshal(stages)
	if err != nil {
		return nil, fmt.Errorf("序列化阶段失败: %w", err)
	}

	template.Name = name
	template.Description = description
	template.Stages = datatypes.JSON(stagesJSON)

	if err := e.db.Save(&template).Error; err != nil {
		return nil, fmt.Errorf("更新模板失败: %w", err)
	}

	return &template, nil
}

// DeleteTemplate 删除流水线模板
func (e *PipelineEngine) DeleteTemplate(templateID uint64) error {
	if err := e.ensureDB(); err != nil {
		return err
	}

	// 检查是否有正在运行的执行
	var count int64
	activeStatuses := []models.ExecutionStatus{
		models.ExecutionStatusPending,
		models.ExecutionStatusRunning,
		models.ExecutionStatusPaused,
		models.ExecutionStatusRollingBack,
	}
	if err := e.db.Model(&models.PipelineExecution{}).
		Where("pipeline_template_id = ? AND status IN ?", templateID, activeStatuses).
		Count(&count).Error; err != nil {
		return fmt.Errorf("检查执行状态失败: %w", err)
	}

	if count > 0 {
		return fmt.Errorf("无法删除：该模板有 %d 个正在执行的流水线", count)
	}

	if err := e.db.Delete(&models.PipelineTemplate{}, templateID).Error; err != nil {
		return fmt.Errorf("删除模板失败: %w", err)
	}

	return nil
}

// GetActiveExecutions 获取所有活跃的执行任务 (pending, running, paused状态)
func (e *PipelineEngine) GetActiveExecutions(limit int) ([]models.PipelineExecution, error) {
	if err := e.ensureDB(); err != nil {
		return nil, err
	}
	var executions []models.PipelineExecution
	activeStatuses := []models.ExecutionStatus{
		models.ExecutionStatusPending,
		models.ExecutionStatusRunning,
		models.ExecutionStatusPaused,
		models.ExecutionStatusRollingBack,
	}
	if err := e.db.Where("status IN ?", activeStatuses).
		Preload("Template").
		Preload("StageRuns").
		Order("created_at DESC").
		Limit(limit).
		Find(&executions).Error; err != nil {
		return nil, fmt.Errorf("获取活跃执行失败: %w", err)
	}
	return executions, nil
}

// GetAllExecutions 获取所有执行记录 (含历史)
func (e *PipelineEngine) GetAllExecutions(limit int) ([]models.PipelineExecution, error) {
	if err := e.ensureDB(); err != nil {
		return nil, err
	}
	var executions []models.PipelineExecution
	if err := e.db.
		Preload("Template").
		Order("created_at DESC").
		Limit(limit).
		Find(&executions).Error; err != nil {
		return nil, fmt.Errorf("获取执行记录失败: %w", err)
	}
	return executions, nil
}

// ListJobTemplates 获取任务模板列表（通过 JobRuntime 接口）
func (e *PipelineEngine) ListJobTemplates(ctx context.Context) ([]JobTemplateInfo, error) {
	if e.jobRuntime == nil {
		return nil, fmt.Errorf("任务运行时未配置")
	}
	return e.jobRuntime.ListTemplates(ctx)
}

// GetJobTemplate 获取任务模板详情
func (e *PipelineEngine) GetJobTemplate(ctx context.Context, id int) (*JobTemplateInfo, error) {
	if e.jobRuntime == nil {
		return nil, fmt.Errorf("任务运行时未配置")
	}
	return e.jobRuntime.GetTemplate(ctx, id)
}

// GetInventoryVariables 获取集群对应的Inventory变量
func (e *PipelineEngine) GetInventoryVariables(ctx context.Context, clusterName string) (string, error) {
	if e.jobRuntime == nil {
		return "", errors.New("job runtime not configured")
	}
	return e.jobRuntime.GetInventoryVariables(ctx, clusterName)
}

// GenerateSOPFlow 生成 AI-SOP 所需的完整执行流定义
func (e *PipelineEngine) GenerateSOPFlow(ctx context.Context, executionID uint64) (*models.SOPFlow, error) {
	if err := e.ensureDB(); err != nil {
		return nil, err
	}

	// 1. 获取执行记录
	var execution models.PipelineExecution
	if err := e.db.Preload("Template").First(&execution, executionID).Error; err != nil {
		return nil, fmt.Errorf("Execution not found: %w", err)
	}

	// 2. 解析 Stage 配置
	var stages []models.StageDefinition
	// 优先使用 EffectiveStages (实际执行的阶段)，如果为空则回退到 Template 的 Stages
	if len(execution.EffectiveStages) > 0 {
		// EffectiveStages 是 datatypes.JSON ([]byte)
		// 需要特定的结构体来解析，或者 []StageDefinition
		// 但 EffectiveStages 存储是 JSON，这里假设结构兼容
		if err := json.Unmarshal(execution.EffectiveStages, &stages); err != nil {
			logger.L().Warn("Parsed effective_stages failed, fallback to template", zap.Error(err))
		}
	}

	if len(stages) == 0 && execution.Template != nil {
		if err := json.Unmarshal(execution.Template.Stages, &stages); err != nil {
			return nil, fmt.Errorf("解析模板 Stages 失败: %w", err)
		}
	}

	// 3. 构建 SOP Flow
	sopFlow := &models.SOPFlow{
		ExecutionID:  execution.ID,
		PipelineName: execution.Template.Name, // 假设 Template 已加载
		ClusterName:  execution.ClusterName,
		GlobalParams: execution.Parameters,
		Stages:       make([]models.SOPStage, 0, len(stages)),
	}

	// 4. 遍历阶段并填充详情
	for _, stage := range stages {
		sopStage := models.SOPStage{
			StageID:   stage.ID,
			StageName: stage.Name,
			Type:      stage.Type,
		}

		switch stage.Type {
		case models.StageTypeAWXJob:
			// 获取 AWX 模板详情 (Playbook 等)
			// 注意：这里我们获取的是**原始**模板的详情，因为 SOP 分析的是这一类任务的逻辑
			// 用户在任务中可能配置了 override 参数，也应该包含进去
			templateInfo, err := e.GetJobTemplate(ctx, stage.Config.AWXTemplateID)
			if err != nil {
				logger.L().Warn("获取 AWX Template 详情失败",
					zap.Int("id", stage.Config.AWXTemplateID), zap.Error(err))
				// 继续执行，只是缺少部分信息
			}

			// 获取 Inventory 变量 (如果 ClusterName 存在)
			inventoryVars := ""
			if execution.ClusterName != "" {
				vars, err := e.GetInventoryVariables(ctx, execution.ClusterName)
				if err != nil {
					logger.L().Warn("获取 Inventory 变量失败",
						zap.String("cluster", execution.ClusterName), zap.Error(err))
				} else {
					inventoryVars = vars
				}
			}

			limit := ""
			if val, ok := stage.Config.ExtraVars["limit"]; ok {
				limit = val
			}

			awxDetail := &models.SOPAWXJobDetail{
				TemplateID:    stage.Config.AWXTemplateID,
				TemplateName:  stage.Config.AWXTemplateName,
				Limit:         limit,
				ExtraVars:     make(map[string]interface{}),
				Inventory:     execution.ClusterName, // 默认使用 ClusterName 作为 Inventory 名
				InventoryVars: inventoryVars,
			}

			if templateInfo != nil {
				awxDetail.Playbook = templateInfo.Playbook
				// 合并 Template 默认的 extra_vars? 暂时不需要，AI 可以自己分析
				// 但如果有 override 的 extra_vars，需要加上
			}

			// 合并 Stage 级别的 ExtraVars
			if stage.Config.ExtraVars != nil {
				for k, v := range stage.Config.ExtraVars {
					awxDetail.ExtraVars[k] = v
				}
			}

			// 合并运行时参数 (Global execution parameters) - 如果有参数绑定逻辑，这里应该处理
			// 简单起见，暂时把 GlobalParams 也包含进去或者由前端/AI处理
			// 这里我们只处理明确配置在 Stage 上的

			sopStage.AWXJob = awxDetail

		case models.StageTypeManualGate:
			sopStage.ManualGate = &models.SOPManualGateDetail{
				ApproverRoles: stage.Config.ApproverRoles,
				Timeout:       stage.Config.TimeoutMinutes,
			}
		}

		sopFlow.Stages = append(sopFlow.Stages, sopStage)
	}

	return sopFlow, nil
}

// UpdateInventoryVariables 更新集群对应的Inventory变量
func (e *PipelineEngine) UpdateInventoryVariables(ctx context.Context, clusterName string, variables string) error {
	if e.jobRuntime == nil {
		return fmt.Errorf("任务运行时未配置")
	}
	return e.jobRuntime.UpdateInventoryVariables(ctx, clusterName, variables)
}

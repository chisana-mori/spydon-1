package services

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"robusta-web/backend/internal/pkg/awx"

	"go.uber.org/zap"
)

// AWXRuntime AWX 任务运行时实现
type AWXRuntime struct {
	client *awx.Client
	logger *zap.Logger
}

func NewAWXRuntime(client *awx.Client, logger *zap.Logger) *AWXRuntime {
	return &AWXRuntime{
		client: client,
		logger: logger,
	}
}

func (r *AWXRuntime) Name() string {
	return "awx"
}

func (r *AWXRuntime) LaunchJob(ctx context.Context, config JobConfig) (*JobHandle, error) {
	req := awx.JobLaunchRequest{
		ExtraVars: config.ExtraVars,
		Limit:     config.Limit,
	}

	// Dry Run 模式
	if config.DryRun {
		req.JobType = "check"
		req.Diff = true
	}

	resp, err := r.client.LaunchJob(ctx, config.TemplateID, req)
	if err != nil {
		return nil, fmt.Errorf("启动 Job 失败: %w", err)
	}

	r.logger.Info("AWX Job 已启动",
		zap.Int("job_id", resp.Job),
		zap.Int("template_id", config.TemplateID),
		zap.String("cluster_name", config.ClusterName))

	return &JobHandle{
		RuntimeName: r.Name(),
		JobID:       resp.Job,
	}, nil
}

func (r *AWXRuntime) PrepareClonedTemplate(ctx context.Context, config CloneTemplateConfig) (int, error) {
	inventoryID := 0
	if config.ClusterName != "" {
		inventory, err := r.client.GetInventoryByName(ctx, config.ClusterName)
		if err != nil {
			r.logger.Info("Inventory 不存在, 正在创建",
				zap.String("cluster_name", config.ClusterName))

			createReq := awx.InventoryCreateRequest{
				Name:         config.ClusterName,
				Description:  "Auto-created for cluster: " + config.ClusterName,
				Organization: 1, // 默认使用 Organization ID 1
			}

			newInventory, createErr := r.client.CreateInventory(ctx, createReq)
			if createErr != nil {
				return 0, fmt.Errorf("创建 Inventory 失败: %w", createErr)
			}
			inventoryID = newInventory.ID
			r.logger.Info("Inventory 已创建", zap.Int("inventory_id", inventoryID))
		} else {
			inventoryID = inventory.ID
		}
	}

	clonedName := r.generateClonedTemplateName(config.TemplateName)
	clonedTemplate, err := r.client.CopyJobTemplate(ctx, config.TemplateID, clonedName)
	if err != nil {
		return 0, fmt.Errorf("克隆模板失败: %w", err)
	}
	clonedTemplateID := clonedTemplate.ID

	r.logger.Info("模板已克隆",
		zap.Int("source_id", config.TemplateID),
		zap.Int("cloned_id", clonedTemplateID),
		zap.String("cloned_name", clonedName))

	updateReq := awx.JobTemplateUpdateRequest{}
	needUpdate := false

	if inventoryID > 0 {
		updateReq.Inventory = inventoryID
		needUpdate = true
	}

	if config.Limit != "" {
		updateReq.Limit = config.Limit
		needUpdate = true
	}

	if len(config.ExtraVars) > 0 {
		extraVarsBytes, _ := json.Marshal(config.ExtraVars)
		updateReq.ExtraVars = string(extraVarsBytes)
		needUpdate = true
	}

	if needUpdate {
		if err := r.client.UpdateJobTemplate(ctx, clonedTemplateID, updateReq); err != nil {
			_ = r.client.DeleteJobTemplate(ctx, clonedTemplateID)
			return 0, fmt.Errorf("更新克隆模板配置失败: %w", err)
		}
		r.logger.Debug("克隆模板配置已更新",
			zap.Int("template_id", clonedTemplateID),
			zap.Int("inventory_id", inventoryID),
			zap.String("limit", config.Limit))
	}

	return clonedTemplateID, nil
}

func (r *AWXRuntime) CleanupClonedTemplate(ctx context.Context, clonedTemplateID int) error {
	if clonedTemplateID <= 0 {
		return nil
	}

	if err := r.client.DeleteJobTemplate(ctx, clonedTemplateID); err != nil {
		r.logger.Warn("删除克隆模板失败",
			zap.Int("template_id", clonedTemplateID),
			zap.Error(err))
		return err
	}

	r.logger.Debug("克隆模板已清理", zap.Int("template_id", clonedTemplateID))
	return nil
}

func (r *AWXRuntime) generateClonedTemplateName(originalName string) string {
	timestamp := time.Now().Format("20060102_150405")
	randomNum := rand.Intn(900000) + 100000 // 6位随机数 100000-999999
	return fmt.Sprintf("%s_clone_%s_%d", originalName, timestamp, randomNum)
}

func (r *AWXRuntime) WaitForJob(ctx context.Context, handle *JobHandle, pollInterval time.Duration) (*JobResult, error) {
	job, err := r.client.WaitForJob(ctx, handle.JobID, pollInterval)
	if err != nil {
		return nil, err
	}

	return &JobResult{
		Status:  job.Status,
		Success: awx.IsJobSuccessful(job.Status),
	}, nil
}

func (r *AWXRuntime) GetJobStatus(ctx context.Context, handle *JobHandle) (string, error) {
	job, err := r.client.GetJob(ctx, handle.JobID)
	if err != nil {
		return "", err
	}
	return job.Status, nil
}

func (r *AWXRuntime) CancelJob(ctx context.Context, handle *JobHandle) error {
	return r.client.CancelJob(ctx, handle.JobID)
}

func (r *AWXRuntime) ListTemplates(ctx context.Context) ([]JobTemplateInfo, error) {
	templates, err := r.client.ListJobTemplates(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]JobTemplateInfo, len(templates))
	for i, t := range templates {
		result[i] = JobTemplateInfo{
			ID:          t.ID,
			Name:        t.Name,
			Description: t.Description,
		}
	}
	return result, nil
}

func (r *AWXRuntime) GetTemplate(ctx context.Context, id int) (*JobTemplateInfo, error) {
	t, err := r.client.GetJobTemplate(ctx, id)
	if err != nil {
		return nil, err
	}

	return &JobTemplateInfo{
		ID:          t.ID,
		Name:        t.Name,
		Description: t.Description,
		Playbook:    t.Playbook,
		ExtraVars:   t.ExtraVars,
	}, nil
}

func (r *AWXRuntime) GetInventoryVariables(ctx context.Context, clusterName string) (string, error) {
	inventory, err := r.client.GetInventoryByName(ctx, clusterName)
	if err != nil {
		return "", err
	}

	fullInventory, err := r.client.GetInventory(ctx, inventory.ID)
	if err != nil {
		return "", err
	}

	r.logger.Debug("获取 Inventory 变量成功",
		zap.String("cluster_name", clusterName),
		zap.Int("inventory_id", inventory.ID))

	return fullInventory.Variables, nil
}

func (r *AWXRuntime) UpdateInventoryVariables(ctx context.Context, clusterName string, variables string) error {
	inventory, err := r.client.GetInventoryByName(ctx, clusterName)
	if err != nil {
		r.logger.Info("Inventory 不存在, 正在创建",
			zap.String("cluster_name", clusterName))

		createReq := awx.InventoryCreateRequest{
			Name:         clusterName,
			Description:  "Auto-created for cluster: " + clusterName,
			Organization: 1, // 默认使用 Organization ID 1
			Variables:    variables,
		}

		newInventory, createErr := r.client.CreateInventory(ctx, createReq)
		if createErr != nil {
			return createErr
		}

		r.logger.Info("Inventory 已创建并设置变量",
			zap.String("cluster_name", clusterName),
			zap.Int("inventory_id", newInventory.ID))
		return nil
	}

	if err := r.client.UpdateInventoryVariables(ctx, inventory.ID, variables); err != nil {
		return err
	}

	r.logger.Info("Inventory 变量已更新",
		zap.String("cluster_name", clusterName),
		zap.Int("inventory_id", inventory.ID))

	return nil
}

func (r *AWXRuntime) GetClient() *awx.Client {
	return r.client
}

// Ensure AWXRuntime implements JobRuntime
var _ JobRuntime = (*AWXRuntime)(nil)

package services

import (
	"context"
	"time"

	"robusta-web/backend/internal/pkg/awx"

	"go.uber.org/zap"
)

// AWXRuntime AWX 任务运行时实现
// 实现 JobRuntime 接口，封装 AWX API 调用
type AWXRuntime struct {
	client *awx.Client
	logger *zap.Logger
}

// NewAWXRuntime 创建 AWX 运行时
func NewAWXRuntime(client *awx.Client, logger *zap.Logger) *AWXRuntime {
	return &AWXRuntime{
		client: client,
		logger: logger,
	}
}

// Name 返回运行时名称
func (r *AWXRuntime) Name() string {
	return "awx"
}

// LaunchJob 启动 AWX Job
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
		return nil, err
	}

	r.logger.Info("AWX Job 已启动",
		zap.Int("job_id", resp.Job),
		zap.Int("template_id", config.TemplateID),
		zap.String("template_name", config.TemplateName))

	return &JobHandle{
		RuntimeName: r.Name(),
		JobID:       resp.Job,
	}, nil
}

// WaitForJob 等待 AWX Job 完成
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

// GetJobStatus 获取 AWX Job 状态
func (r *AWXRuntime) GetJobStatus(ctx context.Context, handle *JobHandle) (string, error) {
	job, err := r.client.GetJob(ctx, handle.JobID)
	if err != nil {
		return "", err
	}
	return job.Status, nil
}

// CancelJob 取消 AWX Job
func (r *AWXRuntime) CancelJob(ctx context.Context, handle *JobHandle) error {
	return r.client.CancelJob(ctx, handle.JobID)
}

// ListTemplates 列出 AWX Job Templates
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

// GetTemplate 获取单个 AWX Job Template
func (r *AWXRuntime) GetTemplate(ctx context.Context, id int) (*JobTemplateInfo, error) {
	t, err := r.client.GetJobTemplate(ctx, id)
	if err != nil {
		return nil, err
	}

	return &JobTemplateInfo{
		ID:          t.ID,
		Name:        t.Name,
		Description: t.Description,
		ExtraVars:   t.ExtraVars,
	}, nil
}

// GetClient 返回底层 AWX Client（用于需要直接访问的场景）
func (r *AWXRuntime) GetClient() *awx.Client {
	return r.client
}

// Ensure AWXRuntime implements JobRuntime
var _ JobRuntime = (*AWXRuntime)(nil)

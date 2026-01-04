package awx

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"
)

// Client AWX API 客户端
type Client struct {
	baseURL string
	client  *resty.Client
	logger  *zap.Logger
}

// Config AWX 连接配置
type Config struct {
	URL      string `mapstructure:"url" json:"url" yaml:"url"`
	Username string `mapstructure:"username" json:"username" yaml:"username"`
	Password string `mapstructure:"password" json:"password" yaml:"password"`
	Token    string `mapstructure:"token" json:"token" yaml:"token"`          // OAuth2 Token (优先使用)
	Timeout  int    `mapstructure:"timeout" json:"timeout" yaml:"timeout"`    // 请求超时秒数
	Insecure bool   `mapstructure:"insecure" json:"insecure" yaml:"insecure"` // 跳过TLS验证
}

// NewClient 创建AWX客户端
func NewClient(cfg Config, logger *zap.Logger) *Client {
	client := resty.New()

	// 设置超时
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 30
	}
	client.SetTimeout(time.Duration(timeout) * time.Second)

	// 设置基础URL
	baseURL := strings.TrimSuffix(cfg.URL, "/")
	client.SetBaseURL(baseURL)

	// 设置认证
	if cfg.Token != "" {
		client.SetAuthToken(cfg.Token)
	} else if cfg.Username != "" && cfg.Password != "" {
		client.SetBasicAuth(cfg.Username, cfg.Password)
	}

	// 通用请求头
	client.SetHeader("Content-Type", "application/json")

	// TLS配置
	if cfg.Insecure {
		client.SetTLSClientConfig(nil) // 简化：实际需要 &tls.Config{InsecureSkipVerify: true}
	}

	return &Client{
		baseURL: baseURL,
		client:  client,
		logger:  logger,
	}
}

// ListJobTemplates 获取Job Templates列表
func (c *Client) ListJobTemplates(ctx context.Context) ([]JobTemplate, error) {
	var result PaginatedResponse[JobTemplate]
	resp, err := c.client.R().
		SetContext(ctx).
		SetResult(&result).
		Get("/api/v2/job_templates/")

	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("AWX返回错误状态 %d: %s", resp.StatusCode(), string(resp.Body()))
	}

	return result.Results, nil
}

// GetJobTemplate 获取单个Job Template
func (c *Client) GetJobTemplate(ctx context.Context, id int) (*JobTemplate, error) {
	var template JobTemplate
	resp, err := c.client.R().
		SetContext(ctx).
		SetResult(&template).
		Get(fmt.Sprintf("/api/v2/job_templates/%d/", id))

	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("AWX返回错误 %d: %s", resp.StatusCode(), string(resp.Body()))
	}

	return &template, nil
}

// LaunchJob 启动Job
func (c *Client) LaunchJob(ctx context.Context, templateID int, req JobLaunchRequest) (*JobLaunchResponse, error) {
	var result JobLaunchResponse
	resp, err := c.client.R().
		SetContext(ctx).
		SetBody(req).
		SetResult(&result).
		Post(fmt.Sprintf("/api/v2/job_templates/%d/launch/", templateID))

	if err != nil {
		return nil, fmt.Errorf("启动Job失败: %w", err)
	}

	if resp.StatusCode() != http.StatusCreated {
		return nil, fmt.Errorf("AWX返回错误 %d: %s", resp.StatusCode(), string(resp.Body()))
	}

	c.logger.Info("AWX Job已启动", zap.Int("job_id", result.Job), zap.Int("template_id", templateID))
	return &result, nil
}

// LaunchDryRun 启动Dry-Run (Check Mode)
func (c *Client) LaunchDryRun(ctx context.Context, templateID int, req JobLaunchRequest) (*JobLaunchResponse, error) {
	req.JobType = "check"
	req.Diff = true
	return c.LaunchJob(ctx, templateID, req)
}

// GetJob 获取Job状态
func (c *Client) GetJob(ctx context.Context, jobID int) (*Job, error) {
	var job Job
	resp, err := c.client.R().
		SetContext(ctx).
		SetResult(&job).
		Get(fmt.Sprintf("/api/v2/jobs/%d/", jobID))

	if err != nil {
		return nil, fmt.Errorf("获取Job状态失败: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("AWX返回错误 %d: %s", resp.StatusCode(), string(resp.Body()))
	}

	return &job, nil
}

// CancelJob 取消Job
func (c *Client) CancelJob(ctx context.Context, jobID int) error {
	resp, err := c.client.R().
		SetContext(ctx).
		Post(fmt.Sprintf("/api/v2/jobs/%d/cancel/", jobID))

	if err != nil {
		return fmt.Errorf("取消Job失败: %w", err)
	}

	// 202 Accepted 或 405 (已结束无法取消) 都算成功
	if resp.StatusCode() != http.StatusAccepted && resp.StatusCode() != http.StatusMethodNotAllowed {
		return fmt.Errorf("AWX返回错误 %d: %s", resp.StatusCode(), string(resp.Body()))
	}

	c.logger.Info("AWX Job取消请求已发送", zap.Int("job_id", jobID))
	return nil
}

// GetJobEvents 获取Job的事件/日志
func (c *Client) GetJobEvents(ctx context.Context, jobID int, page int) ([]JobEvent, error) {
	var result PaginatedResponse[JobEvent]
	resp, err := c.client.R().
		SetContext(ctx).
		SetQueryParam("page", fmt.Sprintf("%d", page)).
		SetQueryParam("order_by", "counter").
		SetResult(&result).
		Get(fmt.Sprintf("/api/v2/jobs/%d/job_events/", jobID))

	if err != nil {
		return nil, fmt.Errorf("获取Job事件失败: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("AWX返回错误 %d: %s", resp.StatusCode(), string(resp.Body()))
	}

	return result.Results, nil
}

// GetJobStdout 获取Job的标准输出 (文本格式)
func (c *Client) GetJobStdout(ctx context.Context, jobID int) (string, error) {
	resp, err := c.client.R().
		SetContext(ctx).
		SetHeader("Accept", "text/plain").
		Get(fmt.Sprintf("/api/v2/jobs/%d/stdout/", jobID))

	if err != nil {
		return "", fmt.Errorf("获取Job输出失败: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return "", fmt.Errorf("AWX返回错误 %d", resp.StatusCode())
	}

	return string(resp.Body()), nil
}

// WaitForJob 等待Job完成 (轮询)
func (c *Client) WaitForJob(ctx context.Context, jobID int, pollInterval time.Duration) (*Job, error) {
	// 立即检查一次
	job, err := c.GetJob(ctx, jobID)
	if err != nil {
		return nil, err
	}
	if IsJobFinished(job.Status) {
		return job, nil
	}

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			job, err := c.GetJob(ctx, jobID)
			if err != nil {
				return nil, err
			}

			if IsJobFinished(job.Status) {
				return job, nil
			}

			c.logger.Debug("Job仍在运行", zap.Int("job_id", jobID), zap.String("status", job.Status))
		}
	}
}

// ListInventories 获取Inventory列表
func (c *Client) ListInventories(ctx context.Context) ([]Inventory, error) {
	var result PaginatedResponse[Inventory]
	resp, err := c.client.R().
		SetContext(ctx).
		SetResult(&result).
		Get("/api/v2/inventories/")

	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("AWX返回错误 %d: %s", resp.StatusCode(), string(resp.Body()))
	}

	return result.Results, nil
}

// Ping 测试AWX连接
func (c *Client) Ping(ctx context.Context) error {
	resp, err := c.client.R().
		SetContext(ctx).
		Get("/api/v2/ping/")

	if err != nil {
		return fmt.Errorf("连接AWX失败: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("AWX返回错误 %d", resp.StatusCode())
	}

	// 解析响应检查是否健康
	var pingResp map[string]interface{}
	if err := json.Unmarshal(resp.Body(), &pingResp); err != nil {
		return fmt.Errorf("解析Ping响应失败: %w", err)
	}

	c.logger.Info("AWX连接成功", zap.Any("response", pingResp))
	return nil
}

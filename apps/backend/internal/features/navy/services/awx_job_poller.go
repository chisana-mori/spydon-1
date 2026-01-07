package services

import (
	"context"
	"sync"
	"time"

	pipelineservice "robusta-web/backend/internal/features/pipeline/services"

	"go.uber.org/zap"
)

// AWXJobPoller 后台轮询器，监控 AWX Job 完成状态并关闭变更单
type AWXJobPoller struct {
	changeManager *ChangeManager
	awxRuntime    *pipelineservice.AWXRuntime
	interval      time.Duration
	logger        *zap.Logger
	stopCh        chan struct{}
	wg            sync.WaitGroup
}

// NewAWXJobPoller 创建 AWX Job Poller
func NewAWXJobPoller(
	changeManager *ChangeManager,
	awxRuntime *pipelineservice.AWXRuntime,
	logger *zap.Logger,
) *AWXJobPoller {
	return &AWXJobPoller{
		changeManager: changeManager,
		awxRuntime:    awxRuntime,
		interval:      30 * time.Second,
		logger:        logger,
		stopCh:        make(chan struct{}),
	}
}

// Start 启动轮询
func (p *AWXJobPoller) Start(ctx context.Context) {
	if !p.changeManager.IsEnabled() {
		p.logger.Info("变更管理已禁用，AWX Job Poller 不启动")
		return
	}

	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		p.run(ctx)
	}()

	p.logger.Info("AWX Job Poller 已启动", zap.Duration("interval", p.interval))
}

// Stop 停止轮询
func (p *AWXJobPoller) Stop() {
	close(p.stopCh)
	p.wg.Wait()
	p.logger.Info("AWX Job Poller 已停止")
}

func (p *AWXJobPoller) run(ctx context.Context) {
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			p.pollPendingJobs(ctx)
		case <-p.stopCh:
			return
		case <-ctx.Done():
			return
		}
	}
}

func (p *AWXJobPoller) pollPendingJobs(ctx context.Context) {
	if p.awxRuntime == nil {
		return
	}

	tickets := p.changeManager.GetPendingAWXTickets()
	if len(tickets) == 0 {
		return
	}

	p.logger.Debug("轮询 AWX Job 状态", zap.Int("pending_count", len(tickets)))

	for _, ticket := range tickets {
		// 构建 JobHandle
		handle := &pipelineservice.JobHandle{
			JobID: ticket.AWXJobID,
		}

		status, err := p.awxRuntime.GetJobStatus(ctx, handle)
		if err != nil {
			p.logger.Warn("获取 AWX Job 状态失败",
				zap.Int("job_id", ticket.AWXJobID),
				zap.String("ticket_id", ticket.ID),
				zap.Error(err))
			continue
		}

		// 检查是否为终态
		if p.isTerminalStatus(status) {
			success := status == "successful"
			errorMsg := ""
			if !success {
				errorMsg = "AWX Job 状态: " + status
			}

			p.logger.Info("AWX Job 已完成，关闭变更单",
				zap.Int("job_id", ticket.AWXJobID),
				zap.String("ticket_id", ticket.ID),
				zap.String("status", status),
				zap.Bool("success", success))

			// 关闭变更单 (无论成功失败都视为完成)
			if err := p.changeManager.CompleteTicket(ticket.ID, true, errorMsg); err != nil {
				p.logger.Error("完成变更单失败", zap.Error(err), zap.String("ticket_id", ticket.ID))
			}
		}
	}
}

// isTerminalStatus 判断是否为 AWX Job 终态
func (p *AWXJobPoller) isTerminalStatus(status string) bool {
	switch status {
	case "successful", "failed", "error", "canceled":
		return true
	default:
		return false
	}
}

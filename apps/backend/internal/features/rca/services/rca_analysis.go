package services

import (
	"context"
	"fmt"

	holmesservice "robusta-web/backend/internal/features/holmes/services"
	"robusta-web/backend/internal/models"
	"robusta-web/backend/pkg/logger"

	"go.uber.org/zap"
)

func (s *RCAService) runAnalysis(rcaRun *models.RCARun, alert *models.Alert) {
	if s.analysisExecutor == nil {
		s.executeRCAAnalysis(rcaRun, alert)
		return
	}

	s.analysisExecutor(rcaRun, alert)
}

// SetAnalysisExecutor 允许自定义分析执行逻辑（主要用于测试）
func (s *RCAService) SetAnalysisExecutor(fn func(*models.RCARun, *models.Alert)) {
	s.analysisExecutor = fn
}

// executeRCAAnalysis 执行RCA分析（异步）
func (s *RCAService) executeRCAAnalysis(rcaRun *models.RCARun, alert *models.Alert) {
	logger.L().Info("开始执行RCA分析",
		zap.Uint64("alert_id", alert.ID),
		zap.Uint64("run_id", rcaRun.ID),
	)

	// 更新状态为运行中
	_ = s.UpdateRCARunStatus(rcaRun.ID, string(models.RCAStatusRunning), "")
	s.broadcastStatus(models.FormatID(alert.ID), models.RCAStatusRunning, models.FormatID(rcaRun.ID))

	logger.L().Info("RCA状态已更新为运行中",
		zap.Uint64("alert_id", alert.ID),
		zap.Uint64("run_id", rcaRun.ID),
	)

	includeTools := true
	opts := holmesservice.InvestigateOptions{
		Language:         "zh-CN",
		IncludeToolCalls: &includeTools,
		Source:           "auto-rca",
	}

	// 尝试获取关联的知识库文章
	if s.knowledgeService != nil {
		if kb, err := s.knowledgeService.BuildKnowledgeBaseForRule(context.Background(), alert.Title); err == nil && kb != "" {
			opts.KnowledgeBase = kb
			logger.L().Info("已加载知识库文章作为提示词", zap.String("alert_rule", alert.Title))
		} else if err != nil {
			logger.L().Warn("获取知识库文章内容失败", zap.Error(err), zap.String("alert_rule", alert.Title))
		}
	}
	chunks, err := s.holmesService.StreamInvestigateChunks(context.Background(), alert, opts, nil)
	if err != nil {
		logger.L().Error("HolmesGPT流式分析失败", zap.Error(err), zap.Uint64("alert_id", alert.ID))
		s.holmesService.FinalizeStreamRun(context.Background(), rcaRun, chunks, nil, err)
		s.handleRCAFailure(rcaRun.ID, models.FormatID(alert.ID), fmt.Errorf("HolmesGPT stream failed: %w", err))
		return
	}

	logger.L().Info("RCA分析完成",
		zap.Uint64("alert_id", alert.ID),
		zap.Uint64("run_id", rcaRun.ID),
	)

	s.holmesService.FinalizeStreamRun(context.Background(), rcaRun, chunks, nil, nil)
	s.broadcastStatus(models.FormatID(alert.ID), models.RCAStatusCompleted, models.FormatID(rcaRun.ID))
}

func (s *RCAService) handleRCAFailure(runID uint64, alertID string, err error) {
	if err == nil {
		return
	}
	if s.holmesService == nil {
		_ = s.UpdateRCARunStatus(runID, string(models.RCAStatusFailed), err.Error())
	}
	s.Broadcast(alertID, fmt.Sprintf(`{"type":"status","status":"failed","error":"%s","run_id":"%s"}`, err.Error(), models.FormatID(runID)))
}

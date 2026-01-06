package services

import (
	"errors"
	"fmt"
	"time"

	systemsettingservice "robusta-web/backend/internal/features/systemsetting/services"
	"robusta-web/backend/internal/logger"
	"robusta-web/backend/internal/models"

	"go.uber.org/zap"
	"golang.org/x/time/rate"
	"gorm.io/gorm"
)

func (s *RCAService) fetchAutoRCAConfig() (*systemsettingservice.AutoRCAConfig, time.Time, error) {
	config, updatedAt, err := s.settingService.GetAutoRCAConfig()
	if err != nil {
		return nil, time.Time{}, err
	}
	if len(config.AllowedSeverities) == 0 {
		config.AllowedSeverities = systemsettingservice.DefaultAllowedSeverities()
	}
	return config, updatedAt, nil
}

func (s *RCAService) loadAutoRCAConfig() *systemsettingservice.AutoRCAConfig {
	config, _, err := s.fetchAutoRCAConfig()
	if err != nil {
		logger.L().Warn("获取Auto-RCA配置失败，使用降级配置（禁用Auto-RCA）", zap.Error(err))
		return fallbackAutoRCAConfig()
	}
	return config
}

func (s *RCAService) isSeverityAllowed(config *systemsettingservice.AutoRCAConfig, severity string) bool {
	if len(config.AllowedSeverities) == 0 {
		return true
	}

	normalized := normalizeSeverity(severity)
	for _, allowed := range config.AllowedSeverities {
		if normalizeSeverity(string(allowed)) == normalized {
			return true
		}
	}
	return false
}

// checkRateLimit 检查是否允许执行新的RCA（使用令牌桶）
func (s *RCAService) checkRateLimit(config *systemsettingservice.AutoRCAConfig) (bool, error) {
	if config == nil {
		return true, nil
	}

	if config.RateLimit <= 0 || config.Period <= 0 {
		return true, nil
	}

	// 动态更新令牌桶配置（如果配置变化）
	s.updateRateLimiter(config)

	// 尝试获取令牌（非阻塞）
	s.limiterMux.RLock()
	defer s.limiterMux.RUnlock()
	return s.rateLimiter.Allow(), nil
}

// updateRateLimiter 根据配置更新令牌桶参数
func (s *RCAService) updateRateLimiter(config *systemsettingservice.AutoRCAConfig) {
	if config == nil || config.Period <= 0 {
		return
	}

	// 计算每秒速率：rateLimit / period
	ratePerSecond := float64(config.RateLimit) / float64(config.Period)
	newLimit := rate.Limit(ratePerSecond)

	// 突发容量设为速率限制值本身，允许短时间内的突发请求
	burst := config.RateLimit
	if burst < 1 {
		burst = 1
	}

	s.limiterMux.Lock()
	defer s.limiterMux.Unlock()

	// 只在配置变化时更新，并重置令牌桶以避免旧的令牌配额影响新配置
	if s.rateLimiter.Limit() != newLimit || s.rateLimiter.Burst() != burst {
		s.rateLimiter = rate.NewLimiter(newLimit, burst)
		logger.L().Info("更新RCA速率限制配置",
			zap.Float64("rate_per_second", ratePerSecond),
			zap.Int("burst", burst),
			zap.Int("rate_limit", config.RateLimit),
			zap.Int("period", config.Period),
		)
	}
}

// processQueue 处理RCA队列
func (s *RCAService) processQueue() {
	logger.L().Info("RCA队列处理器已启动")
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.queueChan:
			logger.L().Debug("收到队列信号，处理下一个任务")
			s.processNextInQueue()
		case <-ticker.C:
			logger.L().Debug("定时检查队列")
			s.processNextInQueue()
		}
	}
}

func (s *RCAService) watchAutoRCAConfig() {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		s.reloadConfigIfChanged()
	}
}

func (s *RCAService) reloadConfigIfChanged() {
	config, updatedAt, err := s.fetchAutoRCAConfig()
	if err != nil {
		logger.L().Error("自动刷新Auto-RCA配置失败", zap.Error(err))
		return
	}

	s.configVersionMux.RLock()
	lastVersion := s.configVersion
	s.configVersionMux.RUnlock()

	if updatedAt.After(lastVersion) {
		s.updateRateLimiter(config)
		s.setConfigVersion(updatedAt)
		logger.L().Info("检测到Auto-RCA配置更新，已自动应用",
			zap.Int("rate_limit", config.RateLimit),
			zap.Int("period", config.Period),
			zap.Any("allowed_severities", config.AllowedSeverities),
		)
	}
}

func (s *RCAService) setConfigVersion(version time.Time) {
	s.configVersionMux.Lock()
	s.configVersion = version
	s.configVersionMux.Unlock()
}

// processNextInQueue 处理队列中的下一个任务
func (s *RCAService) processNextInQueue() {
	// 1. 获取配置
	config := s.loadAutoRCAConfig()

	logger.L().Debug("处理队列任务",
		zap.Bool("enabled", config.Enabled),
		zap.Int("rate_limit", config.RateLimit),
		zap.Int("period", config.Period),
	)

	if !config.Enabled {
		logger.L().Debug("Auto-RCA 未开启，跳过队列任务")
		return
	}

	// 2. 检查速率限制
	allowed, err := s.checkRateLimit(config)
	if err != nil {
		logger.L().Error("处理队列时检查速率限制失败", zap.Error(err))
		return
	}

	if !allowed {
		logger.L().Debug("速率限制中，跳过队列处理")
		return // 仍然受限，等待下一次检查
	}

	// 3. 获取优先级最高的排队任务（严重级别优先，其次按创建时间）
	var queuedRun models.RCARun
	if err := s.db.Preload("Alert").
		Joins("JOIN spydon_alerts ON spydon_alerts.id = spydon_rca_runs.alert_id").
		Where("spydon_rca_runs.status = ?", models.RCAStatusQueued).
		Order(severityPriorityCaseExpr + " DESC").
		Order("spydon_rca_runs.created_at ASC").
		First(&queuedRun).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			logger.L().Error("获取排队任务失败", zap.Error(err))
		} else {
			logger.L().Debug("队列中没有待处理任务")
		}
		return
	}

	logger.L().Info("找到排队的RCA任务",
		zap.Uint64("run_id", queuedRun.ID),
		zap.Uint64("alert_id", queuedRun.AlertID),
	)

	alertID := models.FormatID(queuedRun.AlertID)
	if queuedRun.Alert == nil {
		logger.L().Warn("排队任务缺少告警信息，标记为失败", zap.Uint64("rca_run_id", queuedRun.ID))
		errMsg := "missing alert reference for queued RCA run"
		now := time.Now()
		queuedRun.Status = string(models.RCAStatusFailed)
		queuedRun.ErrorMessage = &errMsg
		queuedRun.CompletedAt = &now
		if err := s.db.Save(&queuedRun).Error; err != nil {
			logger.L().Error("更新排队任务失败", zap.Error(err))
		}
		s.Broadcast(alertID, fmt.Sprintf(`{"type":"status","status":"failed","error":"%s"}`, errMsg))
		return
	}

	if !s.isSeverityAllowed(config, queuedRun.Alert.Severity) {
		logger.L().Info("跳过不符合级别配置的排队RCA任务",
			zap.Uint64("alert_id", queuedRun.Alert.ID),
			zap.String("severity", queuedRun.Alert.Severity),
		)
		errMsg := "alert severity no longer allowed by Auto-RCA configuration"
		now := time.Now()
		queuedRun.Status = string(models.RCAStatusFailed)
		queuedRun.ErrorMessage = &errMsg
		queuedRun.CompletedAt = &now
		if err := s.db.Save(&queuedRun).Error; err != nil {
			logger.L().Error("更新排队任务失败", zap.Error(err))
		}
		s.Broadcast(alertID, fmt.Sprintf(`{"type":"status","status":"failed","error":"%s"}`, errMsg))
		return
	}

	// 4. 更新状态并执行
	queuedRun.Status = string(models.RCAStatusPending)
	queuedRun.StartedAt = time.Now()
	if err := s.db.Save(&queuedRun).Error; err != nil {
		logger.L().Error("更新排队任务状态失败", zap.Error(err))
		return
	}

	logger.L().Info("从队列中取出RCA任务开始执行",
		zap.Uint64("alert_id", queuedRun.Alert.ID),
		zap.Uint64("run_id", queuedRun.ID),
		zap.String("severity", queuedRun.Alert.Severity),
	)

	s.broadcastStatus(models.FormatID(queuedRun.Alert.ID), models.RCAStatusPending, models.FormatID(queuedRun.ID))

	go s.runAnalysis(&queuedRun, queuedRun.Alert)
}

// RefreshAutoRCAConfig 立即重新加载Auto-RCA配置并应用速率限制
func (s *RCAService) RefreshAutoRCAConfig() {
	config, updatedAt, err := s.fetchAutoRCAConfig()
	if err != nil {
		logger.L().Error("刷新Auto-RCA配置失败", zap.Error(err))
		return
	}
	s.updateRateLimiter(config)
	s.setConfigVersion(updatedAt)
	logger.L().Info("Auto-RCA配置已刷新",
		zap.Int("rate_limit", config.RateLimit),
		zap.Int("period", config.Period),
		zap.Any("allowed_severities", config.AllowedSeverities),
	)
}

// TriggerRCAAnalysis 触发自动RCA分析（受级别配置限制）
func (s *RCAService) TriggerRCAAnalysis(alert *models.Alert) (*models.RCARun, error) {
	config := s.loadAutoRCAConfig()
	return s.triggerRCA(alert, config, true)
}

// TriggerRCAManual 手动触发RCA分析（绕过自动级别限制，但仍受速率和开关控制）
func (s *RCAService) TriggerRCAManual(alert *models.Alert) (*models.RCARun, error) {
	config := s.loadAutoRCAConfig()
	return s.triggerRCA(alert, config, false)
}

func (s *RCAService) triggerRCA(alert *models.Alert, config *systemsettingservice.AutoRCAConfig, enforceSeverity bool) (*models.RCARun, error) {
	if !config.Enabled {
		return nil, fmt.Errorf("Auto-RCA功能未开启")
	}

	if enforceSeverity && !s.isSeverityAllowed(config, alert.Severity) {
		logger.L().Debug("Auto-RCA因告警级别被跳过",
			zap.Uint64("alert_id", alert.ID),
			zap.String("severity", alert.Severity),
		)
		return nil, fmt.Errorf("告警级别 %s 不在自动RCA配置范围内", alert.Severity)
	}

	// 1. 检查是否已有正在运行的RCA
	var existingRun models.RCARun
	result := s.db.Where("alert_id = ? AND status IN (?)", alert.ID, []string{string(models.RCAStatusPending), string(models.RCAStatusRunning), string(models.RCAStatusQueued)}).First(&existingRun)

	if result.Error == nil {
		return &existingRun, nil // 直接返回现有运行记录
	} else if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("检查现有RCA运行失败: %w", result.Error)
	}

	// 3. 创建RCA运行记录
	rcaRun := &models.RCARun{
		AlertID:   alert.ID,
		Status:    string(models.RCAStatusPending),
		StartedAt: time.Now(),
	}

	// 4. 检查速率限制
	allowed, err := s.checkRateLimit(config)
	if err != nil {
		logger.L().Error("检查速率限制失败", zap.Error(err))
		// 降级处理：允许运行
		allowed = true
	}

	if !allowed {
		// 超过限制，入队
		rcaRun.Status = string(models.RCAStatusQueued)
		logger.L().Info("RCA请求超过速率限制，已入队", zap.Uint64("alert_id", alert.ID))
	}

	if err := s.CreateRCARun(rcaRun); err != nil {
		return nil, fmt.Errorf("创建RCA运行记录失败: %w", err)
	}

	if rcaRun.Status == string(models.RCAStatusQueued) {
		// 入队后广播状态，此时 rcaRun.ID 已生成
		logger.L().Info("RCA任务已入队",
			zap.Uint64("alert_id", alert.ID),
			zap.Uint64("run_id", rcaRun.ID),
			zap.String("severity", alert.Severity),
		)
		s.broadcastStatus(models.FormatID(alert.ID), models.RCAStatusQueued, models.FormatID(rcaRun.ID))
	} else if rcaRun.Status == string(models.RCAStatusPending) {
		logger.L().Info("RCA任务立即执行",
			zap.Uint64("alert_id", alert.ID),
			zap.Uint64("run_id", rcaRun.ID),
			zap.String("severity", alert.Severity),
		)
		s.broadcastStatus(models.FormatID(alert.ID), models.RCAStatusPending, models.FormatID(rcaRun.ID))
	}

	// 5. 如果未被限流，直接执行；否则通知队列处理器
	if rcaRun.Status != string(models.RCAStatusQueued) {
		go s.runAnalysis(rcaRun, alert)
	} else {
		// 尝试通知队列处理器有新任务
		select {
		case s.queueChan <- struct{}{}:
		default:
		}
	}

	return rcaRun, nil
}

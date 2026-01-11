package services

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"robusta-web/backend/internal/db"
	holmesservice "robusta-web/backend/internal/features/holmes/services"
	knowledgeservice "robusta-web/backend/internal/features/knowledge/services"
	sharedservices "robusta-web/backend/internal/features/shared/services"
	systemsettingservice "robusta-web/backend/internal/features/systemsetting/services"
	"robusta-web/backend/internal/models"
	"robusta-web/backend/pkg/logger"

	"go.uber.org/zap"
	"golang.org/x/time/rate"
	"gorm.io/gorm"
)

// RCAService RCA服务
type RCAService struct {
	db               *db.Database
	auditService     *sharedservices.AuditService
	payloadStorage   sharedservices.PayloadStorage
	settingService   *systemsettingservice.SystemSettingService
	holmesService    *holmesservice.HolmesService
	knowledgeService *knowledgeservice.KnowledgeService

	// 广播相关
	clients    map[string][]chan string // alertID -> []chan chunk
	clientsMux sync.RWMutex

	// 队列相关
	queueChan chan struct{}

	// 速率限制（令牌桶）
	rateLimiter *rate.Limiter
	limiterMux  sync.RWMutex

	analysisExecutor func(*models.RCARun, *models.Alert)
	configVersion    time.Time
	configVersionMux sync.RWMutex
}

// NewRCAService 创建新的RCA服务
func NewRCAService(
	database *db.Database,
	auditService *sharedservices.AuditService,
	payloadStorage sharedservices.PayloadStorage,
	settingService *systemsettingservice.SystemSettingService,
	holmesService *holmesservice.HolmesService,
	knowledgeService *knowledgeservice.KnowledgeService,
) *RCAService {
	// 从配置中获取速率限制参数
	config, version, err := settingService.GetAutoRCAConfig()
	if err != nil {
		logger.L().Warn("初始化RCA服务时获取配置失败，使用降级配置（Auto-RCA将被禁用）", zap.Error(err))
		config = fallbackAutoRCAConfig()
		version = time.Now()
	} else if len(config.AllowedSeverities) == 0 {
		config.AllowedSeverities = systemsettingservice.DefaultAllowedSeverities()
	}

	ratePerSecond := float64(config.RateLimit) / float64(config.Period)
	burst := config.RateLimit
	if burst < 1 {
		burst = 1
	}
	limiter := rate.NewLimiter(rate.Limit(ratePerSecond), burst)

	logger.L().Info("初始化RCA速率限制",
		zap.Int("rate_limit", config.RateLimit),
		zap.Int("period", config.Period),
		zap.Float64("rate_per_second", ratePerSecond),
		zap.Int("burst", burst),
	)

	s := &RCAService{
		db:               database,
		auditService:     auditService,
		payloadStorage:   payloadStorage,
		settingService:   settingService,
		holmesService:    holmesService,
		knowledgeService: knowledgeService,
		clients:          make(map[string][]chan string),
		queueChan:        make(chan struct{}, 1), // 简单的信号通道，实际队列在数据库
		rateLimiter:      limiter,
		configVersion:    version,
	}
	s.analysisExecutor = s.executeRCAAnalysis

	go s.processQueue()
	go s.watchAutoRCAConfig()

	return s
}

// ProcessRCA 处理RCA接收逻辑
func (s *RCAService) ProcessRCA(ctx context.Context, req ProcessRCARequest) (*uint64, error) {
	if !isValidRCAStatus(req.Status) {
		return nil, fmt.Errorf("无效的RCA状态: %s", req.Status)
	}

	var payloadKey string
	if s.payloadStorage != nil && len(req.RawBody) > 0 {
		key, err := s.savePayload(ctx, fmt.Sprintf("rca/%d", req.AlertID), req.RawBody, "application/json")
		if err != nil {
			logger.L().Warn("保存RCA原始数据失败", zap.Error(err), zap.Uint64("alert_id", req.AlertID))
		} else {
			payloadKey = key
		}
	}

	rcaRun := &models.RCARun{
		AlertID:         req.AlertID,
		Status:          req.Status,
		Summary:         &req.Summary,
		Suspects:        toJSON(req.Suspects),
		Recommendations: toJSON(req.Recommendations),
		Attachments:     toJSON(req.Attachments),
		ErrorMessage:    &req.ErrorMessage,
		RawPayloadKey:   payloadKey,
	}

	if req.Status == string(models.RCAStatusCompleted) || req.Status == string(models.RCAStatusFailed) || req.Status == string(models.RCAStatusTimeout) {
		now := time.Now()
		rcaRun.CompletedAt = &now
	}

	if err := s.CreateRCARun(rcaRun); err != nil {
		return nil, fmt.Errorf("保存RCA记录失败: %w", err)
	}

	if s.auditService != nil {
		_ = s.auditService.LogAction(0, "rca_ingested", "rca_run", &rcaRun.ID, map[string]interface{}{
			"alert_id": req.AlertID,
			"status":   req.Status,
		}, req.ClientIP, req.UserAgent)
	}

	return &rcaRun.ID, nil
}

// CreateRCARun 创建RCA运行记录
func (s *RCAService) CreateRCARun(rcaRun *models.RCARun) error {
	return s.db.Create(rcaRun).Error
}

// GetRCARunsByAlertID 根据告警ID获取RCA运行记录
func (s *RCAService) GetRCARunsByAlertID(alertID uint64) ([]models.RCARun, error) {
	var rcaRuns []models.RCARun
	if err := s.db.Where("alert_id = ?", alertID).
		Order("created_at DESC").
		Find(&rcaRuns).Error; err != nil {
		return nil, fmt.Errorf("获取RCA运行记录失败: %w", err)
	}
	return rcaRuns, nil
}

// GetRCARunByID 根据ID获取RCA运行记录
func (s *RCAService) GetRCARunByID(id uint64) (*models.RCARun, error) {
	var rcaRun models.RCARun
	if err := s.db.Preload("Alert").First(&rcaRun, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("RCA运行记录不存在")
		}
		return nil, fmt.Errorf("获取RCA运行记录失败: %w", err)
	}
	return &rcaRun, nil
}

// UpdateRCARunStatus 更新RCA运行状态
func (s *RCAService) UpdateRCARunStatus(id uint64, status string, errorMessage string) error {
	updates := map[string]interface{}{
		"status": status,
	}

	if errorMessage != "" {
		updates["error_message"] = errorMessage
	}

	if status == string(models.RCAStatusCompleted) || status == string(models.RCAStatusFailed) || status == string(models.RCAStatusTimeout) {
		updates["completed_at"] = time.Now()
	}

	result := s.db.Model(&models.RCARun{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("更新RCA运行状态失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("RCA运行记录不存在")
	}
	return nil
}

// GetPendingRCARuns 获取待处理的RCA运行记录
func (s *RCAService) GetPendingRCARuns() ([]models.RCARun, error) {
	var rcaRuns []models.RCARun
	if err := s.db.Preload("Alert").
		Where("status = ?", models.RCAStatusPending).
		Order("created_at ASC").
		Find(&rcaRuns).Error; err != nil {
		return nil, fmt.Errorf("获取待处理RCA记录失败: %w", err)
	}
	return rcaRuns, nil
}

// CleanupOldRCARuns 清理旧的RCA运行记录
func (s *RCAService) CleanupOldRCARuns(olderThan time.Duration) error {
	cutoff := time.Now().Add(-olderThan)
	result := s.db.Where("created_at < ?", cutoff).Delete(&models.RCARun{})
	if result.Error != nil {
		return fmt.Errorf("清理旧RCA记录失败: %w", result.Error)
	}
	return nil
}

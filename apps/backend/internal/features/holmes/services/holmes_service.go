package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"robusta-web/backend/internal/config"
	"robusta-web/backend/internal/db"
	knowledgeservice "robusta-web/backend/internal/features/knowledge/services"
	sharedservices "robusta-web/backend/internal/features/shared/services"
	"robusta-web/backend/internal/models"
	"robusta-web/backend/pkg/logger"

	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"
	"gorm.io/datatypes"
)

// HolmesService 处理与HolmesGPT的集成
type HolmesService struct {
	db           *db.Database
	config       *config.Config
	client       *resty.Client
	streamClient *resty.Client
	storage      sharedservices.PayloadStorage
	knowledge    *knowledgeservice.KnowledgeService
}

// InvestigateOptions 控制流式分析请求
type InvestigateOptions struct {
	ForceRefresh     bool
	PreferCache      bool
	Language         string
	IncludeToolCalls *bool
	Source           string
	KnowledgeBase    string
}

// HolmesAnalysisRequest HolmesGPT分析请求
type HolmesAnalysisRequest struct {
	AlertFingerprint string                 `json:"alert_fingerprint"`
	ClusterName      string                 `json:"cluster_name"`
	Context          map[string]interface{} `json:"context"`
	TimeoutSeconds   int                    `json:"timeout_seconds,omitempty"`
}

// HolmesAnalysisResponse HolmesGPT分析响应
type HolmesAnalysisResponse struct {
	ID              string                 `json:"id"`
	Status          string                 `json:"status"`
	Summary         string                 `json:"summary,omitempty"`
	Suspects        map[string]interface{} `json:"suspects,omitempty"`
	Recommendations map[string]interface{} `json:"recommendations,omitempty"`
	Attachments     map[string]interface{} `json:"attachments,omitempty"`
	StartedAt       time.Time              `json:"started_at"`
	CompletedAt     *time.Time             `json:"completed_at,omitempty"`
	ErrorMessage    string                 `json:"error_message,omitempty"`
}

// NewHolmesService 创建HolmesService实例
func NewHolmesService(database *db.Database, cfg *config.Config, storage sharedservices.PayloadStorage, knowledge *knowledgeservice.KnowledgeService) *HolmesService {
	client := resty.New()
	client.SetTimeout(time.Duration(cfg.HolmesGPT.TimeoutSeconds) * time.Second)
	if cfg.HolmesGPT.URL != "" {
		client.SetBaseURL(NormalizeURL(cfg.HolmesGPT.URL))
	}
	if cfg.HolmesGPT.APIKey != "" {
		client.SetAuthToken(cfg.HolmesGPT.APIKey)
	}

	streamClient := resty.New()
	streamClient.SetTimeout(0)
	if cfg.HolmesGPT.URL != "" {
		streamClient.SetBaseURL(NormalizeURL(cfg.HolmesGPT.URL))
	}
	if cfg.HolmesGPT.APIKey != "" {
		streamClient.SetAuthToken(cfg.HolmesGPT.APIKey)
	}

	return &HolmesService{
		db:           database,
		config:       cfg,
		client:       client,
		streamClient: streamClient,
		storage:      storage,
		knowledge:    knowledge,
	}
}

// TriggerAnalysis 触发HolmesGPT分析
func (s *HolmesService) TriggerAnalysis(ctx context.Context, alertID uint64) (*models.RCARun, error) {
	// 获取告警信息
	alert, err := s.getAlertByID(alertID)
	if err != nil {
		return nil, fmt.Errorf("获取告警失败: %w", err)
	}

	// 检查是否已有进行中的分析
	existingRun, err := s.getRunningAnalysis(alertID)
	if err != nil {
		return nil, fmt.Errorf("检查现有分析失败: %w", err)
	}
	if existingRun != nil {
		return existingRun, nil
	}

	// 创建RCA运行记录
	rcaRun := &models.RCARun{
		AlertID:         alert.ID,
		Status:          string(models.RCAStatusPending),
		StartedAt:       time.Now(),
		Suspects:        datatypes.JSON([]byte(`{}`)),
		Recommendations: datatypes.JSON([]byte(`{}`)),
		Attachments:     datatypes.JSON([]byte(`{}`)),
	}

	if err := s.db.Create(rcaRun).Error; err != nil {
		return nil, fmt.Errorf("创建RCA运行记录失败: %w", err)
	}

	// 异步执行分析
	go s.executeAnalysis(context.Background(), rcaRun, alert)

	return rcaRun, nil
}

// executeAnalysis 执行HolmesGPT分析
func (s *HolmesService) executeAnalysis(ctx context.Context, rcaRun *models.RCARun, alert *models.Alert) {
	// 更新状态为运行中
	rcaRun.Status = string(models.RCAStatusRunning)
	s.db.Save(rcaRun)

	// 准备分析请求
	request := HolmesAnalysisRequest{
		AlertFingerprint: alert.Fingerprint,
		ClusterName:      alert.ClusterName,
		Context: map[string]interface{}{
			"title":       alert.Title,
			"description": alert.Description,
			"severity":    alert.Severity,
			"labels":      alert.Labels,
			"annotations": alert.Annotations,
			"starts_at":   alert.StartsAt,
		},
		TimeoutSeconds: s.config.HolmesGPT.TimeoutSeconds,
	}

	// 发送分析请求
	response, err := s.sendAnalysisRequest(ctx, request)
	if err != nil {
		s.handleAnalysisError(rcaRun, err)
		return
	}

	// 更新分析结果
	s.updateAnalysisResult(ctx, rcaRun, response)
}

// sendAnalysisRequest 发送分析请求到HolmesGPT
func (s *HolmesService) sendAnalysisRequest(ctx context.Context, request HolmesAnalysisRequest) (*HolmesAnalysisResponse, error) {
	url := "/api/v1/analyze"
	resp, err := s.client.R().
		SetContext(ctx).
		SetHeader("Content-Type", "application/json").
		SetBody(request).
		Post(url)
	if err != nil {
		return nil, fmt.Errorf("发送HTTP请求失败: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("HolmesGPT返回错误状态 %d: %s", resp.StatusCode(), string(resp.Body()))
	}

	var response HolmesAnalysisResponse
	if err := json.Unmarshal(resp.Body(), &response); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}
	return &response, nil
}

// handleAnalysisError 处理分析错误
func (s *HolmesService) handleAnalysisError(rcaRun *models.RCARun, err error) {
	now := time.Now()
	errorMsg := err.Error()

	rcaRun.Status = string(models.RCAStatusFailed)
	rcaRun.ErrorMessage = &errorMsg
	rcaRun.CompletedAt = &now

	if dbErr := s.db.Save(rcaRun).Error; dbErr != nil {
		logger.L().Error("保存RCA错误状态失败", zap.Error(dbErr), zap.Uint64("rca_run_id", rcaRun.ID))
	}

	logger.L().Error("RCA分析失败", zap.Uint64("rca_run_id", rcaRun.ID), zap.Error(err))
}

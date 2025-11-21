package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"robusta-web/backend/internal/logger"
	"robusta-web/backend/internal/models"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"go.uber.org/zap"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// RCACachedResult 存储到对象存储中的RCA分析结果
type RCACachedResult struct {
	Version      string                  `json:"version"`
	RunID        string                  `json:"run_id"`
	AlertID      string                  `json:"alert_id"`
	CachedAt     time.Time               `json:"cached_at"`
	Analysis     *HolmesAnalysisResponse `json:"analysis"`
	StorageKey   string                  `json:"storage_key,omitempty"`
	StreamChunks []string                `json:"stream_chunks,omitempty"`
	Metadata     map[string]interface{}  `json:"metadata,omitempty"`
}

// updateAnalysisResult 更新分析结果
func (s *HolmesService) updateAnalysisResult(ctx context.Context, rcaRun *models.RCARun, response *HolmesAnalysisResponse) {
	now := time.Now()

	rcaRun.Status = response.Status
	rcaRun.Summary = &response.Summary
	if b, err := json.Marshal(response.Suspects); err == nil {
		rcaRun.Suspects = datatypes.JSON(b)
	}
	if b, err := json.Marshal(response.Recommendations); err == nil {
		rcaRun.Recommendations = datatypes.JSON(b)
	}
	if b, err := json.Marshal(response.Attachments); err == nil {
		rcaRun.Attachments = datatypes.JSON(b)
	}
	rcaRun.CompletedAt = &now

	if response.Status == string(models.RCAStatusFailed) && response.ErrorMessage != "" {
		rcaRun.ErrorMessage = &response.ErrorMessage
	}

	s.persistAnalysisResult(ctx, rcaRun, response, now)

	if err := s.db.Save(rcaRun).Error; err != nil {
		logger.L().Error("更新RCA运行结果失败", zap.Error(err), zap.String("rca_run_id", rcaRun.ID.String()))
	}

	logger.L().Info("RCA分析完成", zap.String("rca_run_id", rcaRun.ID.String()), zap.String("status", rcaRun.Status))
}

func (s *HolmesService) persistAnalysisResult(ctx context.Context, rcaRun *models.RCARun, response *HolmesAnalysisResponse, snapshot time.Time) {
	if s.storage == nil || response == nil {
		return
	}

	cached := RCACachedResult{
		Version:  "v1",
		RunID:    rcaRun.ID.String(),
		AlertID:  rcaRun.AlertID.String(),
		CachedAt: snapshot,
		Analysis: response,
	}

	payload, err := json.Marshal(cached)
	if err != nil {
		logger.L().Error("序列化RCA分析缓存失败", zap.Error(err), zap.String("rca_run_id", rcaRun.ID.String()))
		return
	}

	prefix := fmt.Sprintf("rca-results/%s", rcaRun.AlertID.String())
	key, err := s.storage.Save(ctx, prefix, payload, "application/json")
	if err != nil {
		logger.L().Error("写入RCA分析缓存失败", zap.Error(err), zap.String("rca_run_id", rcaRun.ID.String()))
		return
	}

	rcaRun.RawPayloadKey = key
}

// GetCachedResult 获取存储在对象存储中的最新RCA缓存结果
func (s *HolmesService) GetCachedResult(ctx context.Context, alertID string) (*RCACachedResult, error) {
	if s.storage == nil {
		return nil, nil
	}

	rcaRuns, err := s.GetAnalysisByAlertID(alertID)
	if err != nil {
		return nil, err
	}

	for _, run := range rcaRuns {
		cached, loadErr := s.loadCachedResultForRun(ctx, run)
		if loadErr != nil {
			return nil, loadErr
		}
		if cached != nil {
			return cached, nil
		}
	}

	return nil, nil
}

func (s *HolmesService) loadCachedResultForRun(ctx context.Context, run *models.RCARun) (*RCACachedResult, error) {
	if run == nil || run.RawPayloadKey == "" {
		return nil, nil
	}

	data, err := s.storage.Get(ctx, run.RawPayloadKey)
	if err != nil {
		var minioErr minio.ErrorResponse
		if errors.As(err, &minioErr) && (minioErr.Code == "NoSuchKey" || minioErr.StatusCode == http.StatusNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("获取RCA缓存失败: %w", err)
	}

	var cached RCACachedResult
	if err := json.Unmarshal(data, &cached); err != nil {
		return nil, fmt.Errorf("解析RCA缓存失败: %w", err)
	}

	if cached.StorageKey == "" {
		cached.StorageKey = run.RawPayloadKey
	}
	if cached.AlertID == "" {
		cached.AlertID = run.AlertID.String()
	}
	if cached.RunID == "" {
		cached.RunID = run.ID.String()
	}

	return &cached, nil
}

func (s *HolmesService) GetCachedResultByRunID(ctx context.Context, runID string) (*RCACachedResult, error) {
	if s.storage == nil {
		return nil, nil
	}

	if runID == "" {
		return nil, fmt.Errorf("运行ID不能为空")
	}

	parsedID, err := uuid.Parse(runID)
	if err != nil {
		return nil, fmt.Errorf("无效的运行ID: %w", err)
	}

	var run models.RCARun
	if err := s.db.Where("id = ?", parsedID).First(&run).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, gorm.ErrRecordNotFound
		}
		return nil, fmt.Errorf("查询RCA运行记录失败: %w", err)
	}

	return s.loadCachedResultForRun(ctx, &run)
}

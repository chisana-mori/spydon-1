package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"robusta-web/backend/internal/config"
	"robusta-web/backend/internal/db"
	"robusta-web/backend/internal/logger"
	"robusta-web/backend/internal/models"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"go.uber.org/zap"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// HolmesService 处理与HolmesGPT的集成
type HolmesService struct {
	db      *db.Database
	config  *config.Config
	client  *http.Client
	storage PayloadStorage
}

// HolmesAnalysisRequest HolmesGPT分析请求
type HolmesAnalysisRequest struct {
	AlertFingerprint string                 `json:"alert_fingerprint"`
	ClusterID        string                 `json:"cluster_id"`
	Context          map[string]interface{} `json:"context"`
	Depth            string                 `json:"depth,omitempty"`
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

// RCACachedResult 存储到对象存储中的RCA分析结果
type RCACachedResult struct {
	Version      string                  `json:"version"`
	RunID        string                  `json:"run_id"`
	AlertID      string                  `json:"alert_id"`
	CachedAt     time.Time               `json:"cached_at"`
	Depth        string                  `json:"depth,omitempty"`
	Analysis     *HolmesAnalysisResponse `json:"analysis"`
	StorageKey   string                  `json:"storage_key,omitempty"`
	StreamChunks []string                `json:"stream_chunks,omitempty"`
	Metadata     map[string]interface{}  `json:"metadata,omitempty"`
}

// NewHolmesService 创建HolmesService实例
func NewHolmesService(database *db.Database, cfg *config.Config, storage PayloadStorage) *HolmesService {
	client := &http.Client{
		Timeout: time.Duration(cfg.HolmesGPT.TimeoutSeconds) * time.Second,
	}

	return &HolmesService{
		db:      database,
		config:  cfg,
		client:  client,
		storage: storage,
	}
}

// TriggerAnalysis 触发HolmesGPT分析
func (s *HolmesService) TriggerAnalysis(ctx context.Context, alertID string, depth string) (*models.RCARun, error) {
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
		BaseModel: models.BaseModel{
			ID: uuid.New(),
		},
		AlertID:         uuid.MustParse(alertID),
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
	go s.executeAnalysis(context.Background(), rcaRun, alert, depth)

	return rcaRun, nil
}

// executeAnalysis 执行HolmesGPT分析
func (s *HolmesService) executeAnalysis(ctx context.Context, rcaRun *models.RCARun, alert *models.Alert, depth string) {
	// 更新状态为运行中
	rcaRun.Status = string(models.RCAStatusRunning)
	s.db.Save(rcaRun)

	// 准备分析请求
	request := HolmesAnalysisRequest{
		AlertFingerprint: alert.Fingerprint,
		ClusterID:        alert.ClusterID,
		Context: map[string]interface{}{
			"title":       alert.Title,
			"description": alert.Description,
			"severity":    alert.Severity,
			"labels":      alert.Labels,
			"annotations": alert.Annotations,
			"starts_at":   alert.StartsAt,
		},
		Depth:          depth,
		TimeoutSeconds: s.config.HolmesGPT.TimeoutSeconds,
	}

	// 发送分析请求
	response, err := s.sendAnalysisRequest(ctx, request)
	if err != nil {
		s.handleAnalysisError(rcaRun, err)
		return
	}

	// 更新分析结果
	s.updateAnalysisResult(ctx, rcaRun, response, depth)
}

// sendAnalysisRequest 发送分析请求到HolmesGPT
func (s *HolmesService) sendAnalysisRequest(ctx context.Context, request HolmesAnalysisRequest) (*HolmesAnalysisResponse, error) {
	// 序列化请求
	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	// 创建HTTP请求
	url := fmt.Sprintf("%s/api/v1/analyze", s.config.HolmesGPT.URL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	if s.config.HolmesGPT.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+s.config.HolmesGPT.APIKey)
	}

	// 发送请求
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送HTTP请求失败: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// 读取响应
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	// 检查状态码
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HolmesGPT返回错误状态 %d: %s", resp.StatusCode, string(responseBody))
	}

	// 解析响应
	var response HolmesAnalysisResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return &response, nil
}

// updateAnalysisResult 更新分析结果
func (s *HolmesService) updateAnalysisResult(ctx context.Context, rcaRun *models.RCARun, response *HolmesAnalysisResponse, depth string) {
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

	s.persistAnalysisResult(ctx, rcaRun, response, depth, now)

	if err := s.db.Save(rcaRun).Error; err != nil {
		logger.L().Error("更新RCA运行结果失败", zap.Error(err), zap.String("rca_run_id", rcaRun.ID.String()))
	}

	logger.L().Info("RCA分析完成", zap.String("rca_run_id", rcaRun.ID.String()), zap.String("status", rcaRun.Status))
}

func (s *HolmesService) persistAnalysisResult(ctx context.Context, rcaRun *models.RCARun, response *HolmesAnalysisResponse, depth string, snapshot time.Time) {
	if s.storage == nil || response == nil {
		return
	}

	cached := RCACachedResult{
		Version:  "v1",
		RunID:    rcaRun.ID.String(),
		AlertID:  rcaRun.AlertID.String(),
		CachedAt: snapshot,
		Depth:    depth,
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

// StartStreamRun 为流式RCA创建运行记录
func (s *HolmesService) StartStreamRun(alertID string) (*models.RCARun, error) {
	alert, err := s.getAlertByID(alertID)
	if err != nil {
		return nil, fmt.Errorf("获取告警失败: %w", err)
	}

	run := &models.RCARun{
		BaseModel: models.BaseModel{
			ID: uuid.New(),
		},
		AlertID:         alert.ID,
		Status:          string(models.RCAStatusRunning),
		StartedAt:       time.Now(),
		Suspects:        datatypes.JSON([]byte(`{}`)),
		Recommendations: datatypes.JSON([]byte(`{}`)),
		Attachments:     datatypes.JSON([]byte(`{}`)),
	}

	if err := s.db.Create(run).Error; err != nil {
		return nil, fmt.Errorf("创建流式RCA记录失败: %w", err)
	}

	return run, nil
}

// FinalizeStreamRun 在流式RCA结束后更新状态并缓存结果
func (s *HolmesService) FinalizeStreamRun(ctx context.Context, run *models.RCARun, depth string, streamChunks []string, metadata map[string]interface{}, streamErr error) {
	if run == nil {
		return
	}

	now := time.Now()

	updates := map[string]interface{}{
		"updated_at": now,
	}

	if streamErr != nil {
		errorMessage := streamErr.Error()
		updates["status"] = string(models.RCAStatusFailed)
		updates["error_message"] = errorMessage
		updates["completed_at"] = now

		run.Status = string(models.RCAStatusFailed)
		run.CompletedAt = &now
		run.ErrorMessage = &errorMessage

		if err := s.db.Model(&models.RCARun{}).Where("id = ?", run.ID).Updates(updates).Error; err != nil {
			logger.L().Error("更新流式RCA失败状态时出错", zap.Error(err), zap.String("rca_run_id", run.ID.String()))
		}
		return
	}

	updates["status"] = string(models.RCAStatusCompleted)
	updates["completed_at"] = now

	// 提取summary和完整文本内容
	var summary string
	var fullText string

	if metadata != nil {
		if s, ok := metadata["summary"].(string); ok {
			summary = s
			updates["summary"] = summary
		}
		if ft, ok := metadata["full_text"].(string); ok {
			fullText = ft
		}
	}

	// 如果metadata中没有full_text，从streamChunks中提取
	if fullText == "" && len(streamChunks) > 0 {
		fullText = s.extractTextFromStreamChunks(streamChunks)
	}

	// 保存到MinIO：将完整的分析结果合并到一个JSON文件
	if s.storage != nil {
		cached := RCACachedResult{
			Version:      "v1",
			RunID:        run.ID.String(),
			AlertID:      run.AlertID.String(),
			CachedAt:     now,
			Depth:        depth,
			StreamChunks: streamChunks,
			Metadata: map[string]interface{}{
				"summary":   summary,
				"full_text": fullText,
				"depth":     depth,
				"duration":  now.Sub(run.StartedAt).Seconds(),
			},
		}

		payload, err := json.Marshal(cached)
		if err != nil {
			logger.L().Error("序列化RCA流缓存失败", zap.Error(err), zap.String("rca_run_id", run.ID.String()))
		} else {
			prefix := fmt.Sprintf("rca-results/%s", run.AlertID.String())
			key, saveErr := s.storage.Save(ctx, prefix, payload, "application/json")
			if saveErr != nil {
				logger.L().Error("保存RCA流缓存失败", zap.Error(saveErr), zap.String("rca_run_id", run.ID.String()))
			} else {
				updates["raw_payload_key"] = key
				run.RawPayloadKey = key
				logger.L().Info("RCA分析结果已缓存到MinIO", zap.String("rca_run_id", run.ID.String()), zap.String("object_key", key), zap.Int("payload_size", len(payload)))
			}
		}
	}

	run.Status = string(models.RCAStatusCompleted)
	run.CompletedAt = &now

	if err := s.db.Model(&models.RCARun{}).Where("id = ?", run.ID).Updates(updates).Error; err != nil {
		logger.L().Error("更新流式RCA运行记录失败", zap.Error(err), zap.String("rca_run_id", run.ID.String()))
	}
}

// extractTextFromStreamChunks 从SSE流chunks中提取纯文本内容
func (s *HolmesService) extractTextFromStreamChunks(chunks []string) string {
	var textParts []string

	for _, chunk := range chunks {
		// 解析SSE格式: data: {...}
		lines := strings.Split(chunk, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if !strings.HasPrefix(line, "data: ") {
				continue
			}

			dataStr := strings.TrimPrefix(line, "data: ")
			if dataStr == "[DONE]" || dataStr == "" {
				continue
			}

			// 尝试解析JSON
			var data map[string]interface{}
			if err := json.Unmarshal([]byte(dataStr), &data); err != nil {
				continue
			}

			// 提取content字段
			if content, ok := data["content"].(string); ok && content != "" {
				textParts = append(textParts, content)
			}
		}
	}

	return strings.Join(textParts, "")
}

// handleAnalysisError 处理分析错误
func (s *HolmesService) handleAnalysisError(rcaRun *models.RCARun, err error) {
	now := time.Now()
	errorMsg := err.Error()

	rcaRun.Status = string(models.RCAStatusFailed)
	rcaRun.ErrorMessage = &errorMsg
	rcaRun.CompletedAt = &now

	if dbErr := s.db.Save(rcaRun).Error; dbErr != nil {
		logger.L().Error("保存RCA错误状态失败", zap.Error(dbErr), zap.String("rca_run_id", rcaRun.ID.String()))
	}

	logger.L().Error("RCA分析失败", zap.String("rca_run_id", rcaRun.ID.String()), zap.Error(err))
}

// GetAnalysisByAlertID 根据告警ID获取分析结果
func (s *HolmesService) GetAnalysisByAlertID(alertID string) ([]*models.RCARun, error) {
	var rcaRuns []*models.RCARun

	err := s.db.Where("alert_id = ?", alertID).
		Order("created_at DESC").
		Find(&rcaRuns).Error
	if err != nil {
		return nil, fmt.Errorf("查询RCA运行记录失败: %w", err)
	}

	return rcaRuns, nil
}

// GetAnalysisStats 获取分析统计信息
func (s *HolmesService) GetAnalysisStats(clusterID string) (*models.RCAStats, error) {
	var stats models.RCAStats

	// 基础查询
	query := s.db.Model(&models.RCARun{})
	if clusterID != "" {
		query = query.Joins("JOIN alerts ON rca_runs.alert_id = alerts.id").
			Where("alerts.cluster_id = ?", clusterID)
	}

	// 1. 获取总数
	if err := query.Count(&stats.Total).Error; err != nil {
		return nil, fmt.Errorf("查询总数失败: %w", err)
	}

	if stats.Total == 0 {
		return &stats, nil
	}

	// 2. 按状态分组统计
	// SELECT status, COUNT(*) as count FROM rca_runs ... GROUP BY status
	var statusCounts []struct {
		Status string `json:"status"`
		Count  int64  `json:"count"`
	}

	// 注意：这里需要重新构建查询，因为 Count() 可能会修改 query 对象或者我们想复用 query
	// 最好是重新构建或者 clone
	statusQuery := s.db.Model(&models.RCARun{})
	if clusterID != "" {
		statusQuery = statusQuery.Joins("JOIN alerts ON rca_runs.alert_id = alerts.id").
			Where("alerts.cluster_id = ?", clusterID)
	}

	if err := statusQuery.Select("rca_runs.status, COUNT(*) as count").
		Group("rca_runs.status").
		Scan(&statusCounts).Error; err != nil {
		return nil, fmt.Errorf("查询状态统计失败: %w", err)
	}

	stats.ByStatus = make([]map[string]interface{}, len(statusCounts))
	var successCount int64
	for i, item := range statusCounts {
		stats.ByStatus[i] = map[string]interface{}{
			"status": item.Status,
			"count":  item.Count,
		}
		if item.Status == string(models.RCAStatusCompleted) {
			successCount = item.Count
		}
	}

	// 3. 计算成功率
	if stats.Total > 0 {
		stats.SuccessRate = float64(successCount) / float64(stats.Total)
	}

	// 4. 计算平均耗时 (只计算已完成的任务)
	// SELECT AVG(EXTRACT(EPOCH FROM (completed_at - started_at))) FROM rca_runs WHERE status = 'completed'
	// 注意：不同数据库的日期计算函数不同，这里使用 Go 层计算以保持兼容性，或者使用通用 SQL
	// 为了简单和兼容性，我们查询 completed 的记录的起止时间

	durationQuery := s.db.Model(&models.RCARun{})
	if clusterID != "" {
		durationQuery = durationQuery.Joins("JOIN alerts ON rca_runs.alert_id = alerts.id").
			Where("alerts.cluster_id = ?", clusterID)
	}

	var avgDuration float64
	// Postgres/MySQL 兼容的写法可能比较复杂，这里先用 Go 计算，如果数据量大建议改为 SQL
	// 考虑到性能，只取最近的 1000 条记录来计算平均值作为估算
	var times []struct {
		StartedAt   time.Time
		CompletedAt *time.Time
	}

	if err := durationQuery.Where("rca_runs.status = ? AND rca_runs.completed_at IS NOT NULL", models.RCAStatusCompleted).
		Select("rca_runs.started_at, rca_runs.completed_at").
		Limit(1000).
		Find(&times).Error; err != nil {
		return nil, fmt.Errorf("查询耗时统计失败: %w", err)
	}

	if len(times) > 0 {
		var totalDuration float64
		for _, t := range times {
			if t.CompletedAt != nil {
				totalDuration += t.CompletedAt.Sub(t.StartedAt).Seconds()
			}
		}
		avgDuration = totalDuration / float64(len(times))
	}
	stats.AvgDurationSeconds = avgDuration

	return &stats, nil
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

// 辅助方法

func (s *HolmesService) getAlertByID(alertID string) (*models.Alert, error) {
	var alert models.Alert
	err := s.db.Where("id = ?", alertID).First(&alert).Error
	if err != nil {
		return nil, err
	}
	return &alert, nil
}

func (s *HolmesService) getRunningAnalysis(alertID string) (*models.RCARun, error) {
	var rcaRun models.RCARun
	err := s.db.Where("alert_id = ? AND status IN (?)", alertID, []string{string(models.RCAStatusPending), string(models.RCAStatusRunning)}).
		First(&rcaRun).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &rcaRun, nil
}

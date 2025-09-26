package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"robusta-web/backend/internal/config"
	"robusta-web/backend/internal/db"
	"robusta-web/backend/internal/models"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// HolmesService 处理与HolmesGPT的集成
type HolmesService struct {
	db     *db.Database
	config *config.Config
	client *http.Client
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

// NewHolmesService 创建HolmesService实例
func NewHolmesService(database *db.Database, cfg *config.Config) *HolmesService {
	client := &http.Client{
		Timeout: time.Duration(cfg.HolmesGPT.TimeoutSeconds) * time.Second,
	}

	return &HolmesService{
		db:     database,
		config: cfg,
		client: client,
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
		Status:          "pending",
		StartedAt:       time.Now(),
		Suspects:        datatypes.JSON([]byte(`{}`)),
		Recommendations: datatypes.JSON([]byte(`{}`)),
		Attachments:     datatypes.JSON([]byte(`{}`)),
	}

	if err := s.db.DB.Create(rcaRun).Error; err != nil {
		return nil, fmt.Errorf("创建RCA运行记录失败: %w", err)
	}

	// 异步执行分析
	go s.executeAnalysis(context.Background(), rcaRun, alert, depth)

	return rcaRun, nil
}

// executeAnalysis 执行HolmesGPT分析
func (s *HolmesService) executeAnalysis(ctx context.Context, rcaRun *models.RCARun, alert *models.Alert, depth string) {
	// 更新状态为运行中
	rcaRun.Status = "running"
	s.db.DB.Save(rcaRun)

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
	s.updateAnalysisResult(rcaRun, response)
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
	defer resp.Body.Close()

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
func (s *HolmesService) updateAnalysisResult(rcaRun *models.RCARun, response *HolmesAnalysisResponse) {
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

	if response.Status == "failed" && response.ErrorMessage != "" {
		rcaRun.ErrorMessage = &response.ErrorMessage
	}

	if err := s.db.DB.Save(rcaRun).Error; err != nil {
		log.Printf("更新RCA运行结果失败: %v", err)
	}

	log.Printf("RCA分析完成: %s, 状态: %s", rcaRun.ID, rcaRun.Status)
}

// handleAnalysisError 处理分析错误
func (s *HolmesService) handleAnalysisError(rcaRun *models.RCARun, err error) {
	now := time.Now()
	errorMsg := err.Error()

	rcaRun.Status = "failed"
	rcaRun.ErrorMessage = &errorMsg
	rcaRun.CompletedAt = &now

	if dbErr := s.db.DB.Save(rcaRun).Error; dbErr != nil {
		log.Printf("保存RCA错误状态失败: %v", dbErr)
	}

	log.Printf("RCA分析失败: %s, 错误: %v", rcaRun.ID, err)
}

// GetAnalysisByAlertID 根据告警ID获取分析结果
func (s *HolmesService) GetAnalysisByAlertID(alertID string) ([]*models.RCARun, error) {
	var rcaRuns []*models.RCARun

	err := s.db.DB.Where("alert_id = ?", alertID).
		Order("created_at DESC").
		Find(&rcaRuns).Error

	if err != nil {
		return nil, fmt.Errorf("查询RCA运行记录失败: %w", err)
	}

	return rcaRuns, nil
}

// GetAnalysisStats 获取分析统计信息
func (s *HolmesService) GetAnalysisStats(clusterID string) (*models.RCAStats, error) {
	buildQuery := func() *gorm.DB {
		q := s.db.DB.Model(&models.RCARun{})
		if clusterID != "" {
			q = q.Joins("JOIN alerts ON rca_runs.alert_id = alerts.id").
				Where("alerts.cluster_id = ?", clusterID)
		}
		return q
	}

	var stats models.RCAStats

	// 总数
	if err := buildQuery().Count(&stats.Total).Error; err != nil {
		return nil, fmt.Errorf("查询总数失败: %w", err)
	}

	// 按状态统计
	var statusStats []struct {
		Status string `json:"status"`
		Count  int64  `json:"count"`
	}

	err := buildQuery().
		Select("rca_runs.status, COUNT(*) as count").
		Group("rca_runs.status").
		Scan(&statusStats).Error
	if err != nil {
		return nil, fmt.Errorf("查询状态统计失败: %w", err)
	}

	// 转换为map格式
	byStatus := make([]map[string]interface{}, len(statusStats))
	for i, stat := range statusStats {
		byStatus[i] = map[string]interface{}{
			"status": stat.Status,
			"count":  stat.Count,
		}
	}
	stats.ByStatus = byStatus

	// 成功率
	var successCount int64
	err = buildQuery().
		Where("rca_runs.status = ?", "completed").
		Count(&successCount).Error
	if err != nil {
		return nil, fmt.Errorf("查询成功数量失败: %w", err)
	}

	if stats.Total > 0 {
		stats.SuccessRate = float64(successCount) / float64(stats.Total)
	}

	// 平均持续时间
	type durationRecord struct {
		StartedAt   time.Time
		CompletedAt *time.Time
	}

	var durationRecords []durationRecord
	err = buildQuery().
		Where("rca_runs.completed_at IS NOT NULL").
		Select("rca_runs.started_at, rca_runs.completed_at").
		Find(&durationRecords).Error
	if err != nil {
		return nil, fmt.Errorf("查询平均持续时间失败: %w", err)
	}

	if len(durationRecords) > 0 {
		var total float64
		for _, record := range durationRecords {
			if record.CompletedAt != nil {
				total += record.CompletedAt.Sub(record.StartedAt).Seconds()
			}
		}
		stats.AvgDurationSeconds = total / float64(len(durationRecords))
	}

	return &stats, nil
}

// 辅助方法

func (s *HolmesService) getAlertByID(alertID string) (*models.Alert, error) {
	var alert models.Alert
	err := s.db.DB.Where("id = ?", alertID).First(&alert).Error
	if err != nil {
		return nil, err
	}
	return &alert, nil
}

func (s *HolmesService) getRunningAnalysis(alertID string) (*models.RCARun, error) {
	var rcaRun models.RCARun
	err := s.db.DB.Where("alert_id = ? AND status IN (?)", alertID, []string{"pending", "running"}).
		First(&rcaRun).Error
	if err != nil {
		if err.Error() == "record not found" {
			return nil, nil
		}
		return nil, err
	}
	return &rcaRun, nil
}

// CreateStreamRun 创建用于流式分析的 RCA 运行记录
func (s *HolmesService) CreateStreamRun(alertID uuid.UUID) (*models.RCARun, error) {
	run := &models.RCARun{
		BaseModel:       models.BaseModel{ID: uuid.New()},
		AlertID:         alertID,
		Status:          string(models.RCAStatusRunning),
		StartedAt:       time.Now(),
		Suspects:        datatypes.JSON([]byte(`{}`)),
		Recommendations: datatypes.JSON([]byte(`{}`)),
		Attachments:     datatypes.JSON([]byte(`{}`)),
	}

	if err := s.db.DB.Create(run).Error; err != nil {
		return nil, fmt.Errorf("创建RCA流式运行记录失败: %w", err)
	}

	return run, nil
}

// MarkStreamRunCompleted 将流式分析标记为完成并保存结果
func (s *HolmesService) MarkStreamRunCompleted(runID uuid.UUID, analysisKey string) error {
	updates := map[string]interface{}{
		"status":               string(models.RCAStatusCompleted),
		"analysis_payload_key": analysisKey,
		"completed_at":         time.Now(),
	}

	result := s.db.DB.Model(&models.RCARun{}).Where("id = ?", runID).Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("更新RCA运行状态失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("RCA运行记录不存在")
	}
	return nil
}

// MarkStreamRunFailed 将流式分析标记为失败
func (s *HolmesService) MarkStreamRunFailed(runID uuid.UUID, errMsg string) error {
	updates := map[string]interface{}{
		"status":        string(models.RCAStatusFailed),
		"error_message": errMsg,
		"completed_at":  time.Now(),
	}
	result := s.db.DB.Model(&models.RCARun{}).Where("id = ?", runID).Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("更新失败状态时出错: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("RCA运行记录不存在")
	}
	return nil
}

// GetRunByID 根据ID获取运行记录
func (s *HolmesService) GetRunByID(runID uuid.UUID) (*models.RCARun, error) {
	var run models.RCARun
	if err := s.db.DB.Where("id = ?", runID).First(&run).Error; err != nil {
		return nil, err
	}
	return &run, nil
}

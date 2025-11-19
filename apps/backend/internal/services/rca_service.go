package services

import (
	"errors"
	"fmt"
	"time"

	"robusta-web/backend/internal/db"
	"robusta-web/backend/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// RCAService RCA服务
type RCAService struct {
	db *db.Database
}

// NewRCAService 创建新的RCA服务
func NewRCAService(database *db.Database) *RCAService {
	return &RCAService{
		db: database,
	}
}

// CreateRCARun 创建RCA运行记录
func (s *RCAService) CreateRCARun(rcaRun *models.RCARun) error {
	return s.db.Create(rcaRun).Error
}

// GetRCARunsByAlertID 根据告警ID获取RCA运行记录
func (s *RCAService) GetRCARunsByAlertID(alertID uuid.UUID) ([]models.RCARun, error) {
	var rcaRuns []models.RCARun
	if err := s.db.Where("alert_id = ?", alertID).
		Order("created_at DESC").
		Find(&rcaRuns).Error; err != nil {
		return nil, fmt.Errorf("获取RCA运行记录失败: %w", err)
	}
	return rcaRuns, nil
}

// GetRCARunByID 根据ID获取RCA运行记录
func (s *RCAService) GetRCARunByID(id uuid.UUID) (*models.RCARun, error) {
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
func (s *RCAService) UpdateRCARunStatus(id uuid.UUID, status string, errorMessage string) error {
	updates := map[string]interface{}{
		"status": status,
	}

	if errorMessage != "" {
		updates["error_message"] = errorMessage
	}

	// 如果状态是完成或失败，设置完成时间
	if status == "completed" || status == "failed" || status == "timeout" {
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

// TriggerRCAAnalysis 触发RCA分析
func (s *RCAService) TriggerRCAAnalysis(alert *models.Alert) (*models.RCARun, error) {
	// 检查是否已有正在运行的RCA
	var existingRun models.RCARun
	result := s.db.Where("alert_id = ? AND status IN (?)", alert.ID, []string{"pending", "running"}).First(&existingRun)

	if result.Error == nil {
		return nil, fmt.Errorf("该告警已有正在运行的RCA分析")
	} else if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("检查现有RCA运行失败: %w", result.Error)
	}

	// 创建新的RCA运行记录
	rcaRun := &models.RCARun{
		AlertID:   alert.ID,
		Status:    "pending",
		StartedAt: time.Now(),
	}

	if err := s.CreateRCARun(rcaRun); err != nil {
		return nil, fmt.Errorf("创建RCA运行记录失败: %w", err)
	}

	// 这里应该异步调用HolmesGPT服务
	go s.executeRCAAnalysis(rcaRun, alert)

	return rcaRun, nil
}

// executeRCAAnalysis 执行RCA分析（异步）
func (s *RCAService) executeRCAAnalysis(rcaRun *models.RCARun, alert *models.Alert) {
	// 更新状态为运行中
	_ = s.UpdateRCARunStatus(rcaRun.ID, "running", "")

	// 模拟RCA分析过程（实际应该调用HolmesGPT API）
	time.Sleep(30 * time.Second) // 模拟分析时间

	// 模拟分析结果
	suspects := map[string]interface{}{
		"primary_suspect": "High CPU usage in pod nginx-deployment-xxx",
		"secondary_suspects": []string{
			"Memory pressure on node worker-1",
			"Network latency to external service",
		},
		"confidence_score": 0.85,
	}

	recommendations := map[string]interface{}{
		"immediate_actions": []string{
			"Scale up the nginx deployment",
			"Check node resource allocation",
		},
		"long_term_actions": []string{
			"Implement horizontal pod autoscaling",
			"Review resource requests and limits",
		},
		"priority": "high",
	}

	// 更新RCA结果
	updates := map[string]interface{}{
		"status":          "completed",
		"summary":         "分析完成：发现CPU使用率过高导致的性能问题",
		"suspects":        suspects,
		"recommendations": recommendations,
		"completed_at":    time.Now(),
	}

	if err := s.db.Model(&models.RCARun{}).Where("id = ?", rcaRun.ID).Updates(updates).Error; err != nil {
		// 如果更新失败，标记为失败状态
		_ = s.UpdateRCARunStatus(rcaRun.ID, "failed", fmt.Sprintf("更新RCA结果失败: %v", err))
	}
}

// GetRCAStats 获取RCA统计信息
func (s *RCAService) GetRCAStats(clusterID string) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// 构建基础查询
	baseQuery := s.db.Model(&models.RCARun{})
	if clusterID != "" {
		baseQuery = baseQuery.Joins("JOIN alerts ON rca_runs.alert_id = alerts.id").
			Where("alerts.cluster_id = ?", clusterID)
	}

	// 按状态统计
	var statusStats []struct {
		Status string `json:"status"`
		Count  int64  `json:"count"`
	}

	query := baseQuery
	if clusterID != "" {
		query = s.db.Table("rca_runs").
			Select("rca_runs.status, count(*) as count").
			Joins("JOIN alerts ON rca_runs.alert_id = alerts.id").
			Where("alerts.cluster_id = ?", clusterID).
			Group("rca_runs.status")
	} else {
		query = s.db.Model(&models.RCARun{}).
			Select("status, count(*) as count").
			Group("status")
	}

	if err := query.Find(&statusStats).Error; err != nil {
		return nil, fmt.Errorf("获取RCA状态统计失败: %w", err)
	}
	stats["by_status"] = statusStats

	// 总数统计
	var totalCount int64
	if err := baseQuery.Count(&totalCount).Error; err != nil {
		return nil, fmt.Errorf("获取RCA总数失败: %w", err)
	}
	stats["total"] = totalCount

	// 成功率统计
	var successCount int64
	successQuery := baseQuery
	if err := successQuery.Where("status = ?", "completed").Count(&successCount).Error; err != nil {
		return nil, fmt.Errorf("获取RCA成功数失败: %w", err)
	}

	successRate := float64(0)
	if totalCount > 0 {
		successRate = float64(successCount) / float64(totalCount) * 100
	}
	stats["success_rate"] = successRate

	// 平均执行时间（已完成的RCA）
	var avgDuration float64
	durationQuery := `
		SELECT AVG(EXTRACT(EPOCH FROM (completed_at - started_at))) as avg_duration
		FROM rca_runs
		WHERE status = 'completed' AND completed_at IS NOT NULL
	`

	if clusterID != "" {
		durationQuery = `
			SELECT AVG(EXTRACT(EPOCH FROM (rca_runs.completed_at - rca_runs.started_at))) as avg_duration
			FROM rca_runs
			JOIN alerts ON rca_runs.alert_id = alerts.id
			WHERE rca_runs.status = 'completed'
				AND rca_runs.completed_at IS NOT NULL
				AND alerts.cluster_id = ?
		`
		if err := s.db.Raw(durationQuery, clusterID).Scan(&avgDuration).Error; err != nil {
			return nil, fmt.Errorf("获取平均执行时间失败: %w", err)
		}
	} else {
		if err := s.db.Raw(durationQuery).Scan(&avgDuration).Error; err != nil {
			return nil, fmt.Errorf("获取平均执行时间失败: %w", err)
		}
	}

	stats["avg_duration_seconds"] = avgDuration

	return stats, nil
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

// GetPendingRCARuns 获取待处理的RCA运行记录
func (s *RCAService) GetPendingRCARuns() ([]models.RCARun, error) {
	var rcaRuns []models.RCARun
	if err := s.db.Preload("Alert").
		Where("status = ?", "pending").
		Order("created_at ASC").
		Find(&rcaRuns).Error; err != nil {
		return nil, fmt.Errorf("获取待处理RCA记录失败: %w", err)
	}
	return rcaRuns, nil
}

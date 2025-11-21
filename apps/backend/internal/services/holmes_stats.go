package services

import (
	"fmt"
	"time"

	"robusta-web/backend/internal/models"
)

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
	durationQuery := s.db.Model(&models.RCARun{})
	if clusterID != "" {
		durationQuery = durationQuery.Joins("JOIN alerts ON rca_runs.alert_id = alerts.id").
			Where("alerts.cluster_id = ?", clusterID)
	}

	var avgDuration float64
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

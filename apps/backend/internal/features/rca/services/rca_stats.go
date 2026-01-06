package services

import (
	"fmt"

	"robusta-web/backend/internal/models"
)

// GetRCAStats 获取RCA统计信息
func (s *RCAService) GetRCAStats(clusterName string) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// 构建基础查询
	baseQuery := s.db.Model(&models.RCARun{})
	if clusterName != "" {
		baseQuery = baseQuery.Joins("JOIN spydon_alerts ON spydon_rca_runs.alert_id = spydon_alerts.id").
			Where("spydon_alerts.cluster_name = ?", clusterName)
	}

	// 按状态统计
	var statusStats []struct {
		Status string `json:"status"`
		Count  int64  `json:"count"`
	}

	query := baseQuery
	if clusterName != "" {
		query = s.db.Table("spydon_rca_runs").
			Select("spydon_rca_runs.status, count(*) as count").
			Joins("JOIN spydon_alerts ON spydon_rca_runs.alert_id = spydon_alerts.id").
			Where("spydon_alerts.cluster_name = ?", clusterName).
			Group("spydon_rca_runs.status")
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
	if err := successQuery.Where("status = ?", models.RCAStatusCompleted).Count(&successCount).Error; err != nil {
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
		FROM spydon_rca_runs
		WHERE status = 'completed' AND completed_at IS NOT NULL
	`

	if clusterName != "" {
		durationQuery = `
			SELECT AVG(EXTRACT(EPOCH FROM (spydon_rca_runs.completed_at - spydon_rca_runs.started_at))) as avg_duration
			FROM spydon_rca_runs
			JOIN spydon_alerts ON spydon_rca_runs.alert_id = spydon_alerts.id
			WHERE spydon_rca_runs.status = 'completed'
				AND spydon_rca_runs.completed_at IS NOT NULL
				AND spydon_alerts.cluster_name = ?
		`
		if err := s.db.Raw(durationQuery, clusterName).Scan(&avgDuration).Error; err != nil {
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

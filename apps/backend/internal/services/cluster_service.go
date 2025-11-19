package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"robusta-web/backend/internal/db"
	"robusta-web/backend/internal/models"

	"gorm.io/gorm"
)

// ClusterService 集群服务
type ClusterService struct {
	db *db.Database
}

// NewClusterService 创建新的集群服务
func NewClusterService(database *db.Database) *ClusterService {
	return &ClusterService{
		db: database,
	}
}

// ClusterSummary 集群概览信息
type ClusterSummary struct {
	TotalClusters  int64            `json:"total_clusters"`
	ActiveClusters int64            `json:"active_clusters"`
	TotalAlerts    int64            `json:"total_alerts"`
	CriticalAlerts int64            `json:"critical_alerts"`
	ClusterStats   []ClusterStats   `json:"cluster_stats"`
	AlertTrends    []AlertTrendData `json:"alert_trends"`
}

// ClusterStats 单个集群统计信息
type ClusterStats struct {
	ClusterID     string     `json:"cluster_id"`
	Name          string     `json:"name"`
	Status        string     `json:"status"`
	AlertCount    int64      `json:"alert_count"`
	CriticalCount int64      `json:"critical_count"`
	LastHeartbeat *time.Time `json:"last_heartbeat"`
}

// AlertTrendData 告警趋势数据
type AlertTrendData struct {
	Date  string `json:"date"`
	Count int64  `json:"count"`
}

// UpdateHeartbeat 更新集群心跳
func (s *ClusterService) UpdateHeartbeat(ctx context.Context, cluster *models.Cluster) error {
	now := time.Now()
	cluster.LastHeartbeat = &now

	// 使用cluster_id作为唯一标识
	var existingCluster models.Cluster
	db := s.dbWithContext(ctx)
	result := db.Where("cluster_id = ?", cluster.ClusterID).First(&existingCluster)

	if result.Error == nil {
		// 集群已存在，更新信息
		cluster.ID = existingCluster.ID
		cluster.CreatedAt = existingCluster.CreatedAt
		return db.Save(cluster).Error
	} else if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		// 集群不存在，创建新的
		return db.Create(cluster).Error
	} else {
		// 其他错误
		return result.Error
	}
}

// GetClustersSummary 获取集群概览
func (s *ClusterService) GetClustersSummary() (*ClusterSummary, error) {
	summary := &ClusterSummary{}

	// 获取集群总数
	if err := s.db.Model(&models.Cluster{}).Count(&summary.TotalClusters).Error; err != nil {
		return nil, fmt.Errorf("获取集群总数失败: %w", err)
	}

	// 获取活跃集群数（最近5分钟有心跳）
	fiveMinutesAgo := time.Now().Add(-5 * time.Minute)
	if err := s.db.Model(&models.Cluster{}).
		Where("last_heartbeat > ? AND status = ?", fiveMinutesAgo, "active").
		Count(&summary.ActiveClusters).Error; err != nil {
		return nil, fmt.Errorf("获取活跃集群数失败: %w", err)
	}

	// 获取告警总数
	if err := s.db.Model(&models.Alert{}).Count(&summary.TotalAlerts).Error; err != nil {
		return nil, fmt.Errorf("获取告警总数失败: %w", err)
	}

	// 获取严重告警数
	if err := s.db.Model(&models.Alert{}).
		Where("severity = ? AND status = ?", "critical", "firing").
		Count(&summary.CriticalAlerts).Error; err != nil {
		return nil, fmt.Errorf("获取严重告警数失败: %w", err)
	}

	// 获取各集群统计信息
	clusterStats, err := s.getClusterStats()
	if err != nil {
		return nil, fmt.Errorf("获取集群统计失败: %w", err)
	}
	summary.ClusterStats = clusterStats

	// 获取告警趋势数据（最近7天）
	alertTrends, err := s.getAlertTrends(7)
	if err != nil {
		return nil, fmt.Errorf("获取告警趋势失败: %w", err)
	}
	summary.AlertTrends = alertTrends

	return summary, nil
}

// getClusterStats 获取各集群统计信息
func (s *ClusterService) getClusterStats() ([]ClusterStats, error) {
	var stats []ClusterStats

	// 使用原生SQL查询获取集群统计信息
	query := `
		SELECT
			c.cluster_id,
			c.name,
			c.status,
			c.last_heartbeat,
			COALESCE(alert_counts.total_alerts, 0) as alert_count,
			COALESCE(alert_counts.critical_alerts, 0) as critical_count
		FROM clusters c
		LEFT JOIN (
			SELECT
				cluster_id,
				COUNT(*) as total_alerts,
				COUNT(CASE WHEN severity = 'critical' AND status = 'firing' THEN 1 END) as critical_alerts
			FROM alerts
			WHERE deleted_at IS NULL
			GROUP BY cluster_id
		) alert_counts ON c.cluster_id = alert_counts.cluster_id
		WHERE c.deleted_at IS NULL
		ORDER BY c.name
	`

	if err := s.db.Raw(query).Scan(&stats).Error; err != nil {
		return nil, err
	}

	return stats, nil
}

// getAlertTrends 获取告警趋势数据
func (s *ClusterService) getAlertTrends(days int) ([]AlertTrendData, error) {
	var trends []AlertTrendData

	// 使用原生SQL查询获取每日告警数量
	query := `
		SELECT
			DATE(created_at) as date,
			COUNT(*) as count
		FROM alerts
		WHERE created_at >= CURRENT_DATE - INTERVAL '%d days'
			AND deleted_at IS NULL
		GROUP BY DATE(created_at)
		ORDER BY date
	`

	if err := s.db.Raw(fmt.Sprintf(query, days)).Scan(&trends).Error; err != nil {
		return nil, err
	}

	return trends, nil
}

// GetClusters 获取集群列表
func (s *ClusterService) GetClusters(page, limit int, status string) ([]models.Cluster, int64, error) {
	var clusters []models.Cluster
	var total int64

	query := s.db.Model(&models.Cluster{})

	// 应用状态过滤
	if status != "" {
		query = query.Where("status = ?", status)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("获取集群总数失败: %w", err)
	}

	// 分页查询
	offset := (page - 1) * limit
	if err := query.Order("name").Offset(offset).Limit(limit).Find(&clusters).Error; err != nil {
		return nil, 0, fmt.Errorf("获取集群列表失败: %w", err)
	}

	return clusters, total, nil
}

func (s *ClusterService) dbWithContext(ctx context.Context) *gorm.DB {
	if ctx == nil {
		return s.db.DB
	}
	return s.db.WithContext(ctx)
}

// GetClusterByID 根据ID获取集群
func (s *ClusterService) GetClusterByID(clusterID string) (*models.Cluster, error) {
	var cluster models.Cluster
	if err := s.db.Where("cluster_id = ?", clusterID).First(&cluster).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("集群不存在")
		}
		return nil, fmt.Errorf("获取集群失败: %w", err)
	}
	return &cluster, nil
}

// DeleteCluster 删除集群
func (s *ClusterService) DeleteCluster(clusterID string) error {
	// 软删除集群
	result := s.db.Where("cluster_id = ?", clusterID).Delete(&models.Cluster{})
	if result.Error != nil {
		return fmt.Errorf("删除集群失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("集群不存在")
	}

	// 同时软删除该集群的所有告警
	if err := s.db.Where("cluster_id = ?", clusterID).Delete(&models.Alert{}).Error; err != nil {
		return fmt.Errorf("删除集群告警失败: %w", err)
	}

	return nil
}

// UpdateClusterStatus 更新集群状态
func (s *ClusterService) UpdateClusterStatus(clusterID, status string) error {
	result := s.db.Model(&models.Cluster{}).Where("cluster_id = ?", clusterID).Update("status", status)
	if result.Error != nil {
		return fmt.Errorf("更新集群状态失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("集群不存在")
	}
	return nil
}

// CheckInactiveClusters 检查不活跃的集群
func (s *ClusterService) CheckInactiveClusters(threshold time.Duration) ([]models.Cluster, error) {
	var inactiveClusters []models.Cluster
	cutoff := time.Now().Add(-threshold)

	if err := s.db.Where("last_heartbeat < ? OR last_heartbeat IS NULL", cutoff).
		Find(&inactiveClusters).Error; err != nil {
		return nil, fmt.Errorf("检查不活跃集群失败: %w", err)
	}

	return inactiveClusters, nil
}

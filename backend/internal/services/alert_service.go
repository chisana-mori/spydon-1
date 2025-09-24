package services

import (
	"fmt"
	"time"

	"robusta-web/backend/internal/db"
	"robusta-web/backend/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AlertService 告警服务
type AlertService struct {
	db *db.Database
}

// NewAlertService 创建新的告警服务
func NewAlertService(database *db.Database) *AlertService {
	return &AlertService{
		db: database,
	}
}

// AlertFilters 告警过滤条件
type AlertFilters struct {
	ClusterID string
	Severity  string
	Status    string
	Keyword   string
	Since     *time.Time
}

// CreateOrUpdateAlert 创建或更新告警
func (s *AlertService) CreateOrUpdateAlert(alert *models.Alert) error {
	// 使用fingerprint和cluster_id作为唯一标识
	var existingAlert models.Alert
	result := s.db.Where("fingerprint = ? AND cluster_id = ?", alert.Fingerprint, alert.ClusterID).First(&existingAlert)
	
	if result.Error == nil {
		// 告警已存在，更新
		alert.ID = existingAlert.ID
		alert.CreatedAt = existingAlert.CreatedAt
		return s.db.Save(alert).Error
	} else if result.Error == gorm.ErrRecordNotFound {
		// 告警不存在，创建新的
		return s.db.Create(alert).Error
	} else {
		// 其他错误
		return result.Error
	}
}

// GetAlerts 获取告警列表
func (s *AlertService) GetAlerts(page, limit int, filters AlertFilters) ([]models.Alert, int64, error) {
	var alerts []models.Alert
	var total int64

	// 构建查询
	query := s.db.Model(&models.Alert{}).Preload("Cluster")

	// 应用过滤条件
	if filters.ClusterID != "" {
		query = query.Where("cluster_id = ?", filters.ClusterID)
	}
	if filters.Severity != "" {
		query = query.Where("severity = ?", filters.Severity)
	}
	if filters.Status != "" {
		query = query.Where("status = ?", filters.Status)
	}
	if filters.Keyword != "" {
		query = query.Where("title ILIKE ? OR description ILIKE ?", "%"+filters.Keyword+"%", "%"+filters.Keyword+"%")
	}
	if filters.Since != nil {
		query = query.Where("created_at >= ?", filters.Since)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("获取告警总数失败: %w", err)
	}

	// 分页查询
	offset := (page - 1) * limit
	if err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&alerts).Error; err != nil {
		return nil, 0, fmt.Errorf("获取告警列表失败: %w", err)
	}

	return alerts, total, nil
}

// GetAlertByID 根据ID获取告警
func (s *AlertService) GetAlertByID(id uuid.UUID) (*models.Alert, error) {
	var alert models.Alert
	if err := s.db.Preload("Cluster").Preload("RCARuns").First(&alert, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("告警不存在")
		}
		return nil, fmt.Errorf("获取告警失败: %w", err)
	}
	return &alert, nil
}

// GetAlertByFingerprint 根据指纹获取告警
func (s *AlertService) GetAlertByFingerprint(fingerprint, clusterID string) (*models.Alert, error) {
	var alert models.Alert
	if err := s.db.Where("fingerprint = ? AND cluster_id = ?", fingerprint, clusterID).First(&alert).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("告警不存在")
		}
		return nil, fmt.Errorf("获取告警失败: %w", err)
	}
	return &alert, nil
}

// UpdateAlertStatus 更新告警状态
func (s *AlertService) UpdateAlertStatus(id uuid.UUID, status string) error {
	result := s.db.Model(&models.Alert{}).Where("id = ?", id).Update("status", status)
	if result.Error != nil {
		return fmt.Errorf("更新告警状态失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("告警不存在")
	}
	return nil
}

// GetAlertStats 获取告警统计信息
func (s *AlertService) GetAlertStats(clusterID string) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// 按严重级别统计
	var severityStats []struct {
		Severity string `json:"severity"`
		Count    int64  `json:"count"`
	}

	query := s.db.Model(&models.Alert{}).Select("severity, count(*) as count").Group("severity")
	if clusterID != "" {
		query = query.Where("cluster_id = ?", clusterID)
	}

	if err := query.Find(&severityStats).Error; err != nil {
		return nil, fmt.Errorf("获取严重级别统计失败: %w", err)
	}
	stats["by_severity"] = severityStats

	// 按状态统计
	var statusStats []struct {
		Status string `json:"status"`
		Count  int64  `json:"count"`
	}

	query = s.db.Model(&models.Alert{}).Select("status, count(*) as count").Group("status")
	if clusterID != "" {
		query = query.Where("cluster_id = ?", clusterID)
	}

	if err := query.Find(&statusStats).Error; err != nil {
		return nil, fmt.Errorf("获取状态统计失败: %w", err)
	}
	stats["by_status"] = statusStats

	// 总数统计
	var totalCount int64
	query = s.db.Model(&models.Alert{})
	if clusterID != "" {
		query = query.Where("cluster_id = ?", clusterID)
	}

	if err := query.Count(&totalCount).Error; err != nil {
		return nil, fmt.Errorf("获取总数统计失败: %w", err)
	}
	stats["total"] = totalCount

	// 最近24小时新增告警数
	var recentCount int64
	since := time.Now().Add(-24 * time.Hour)
	query = s.db.Model(&models.Alert{}).Where("created_at >= ?", since)
	if clusterID != "" {
		query = query.Where("cluster_id = ?", clusterID)
	}

	if err := query.Count(&recentCount).Error; err != nil {
		return nil, fmt.Errorf("获取最近告警统计失败: %w", err)
	}
	stats["recent_24h"] = recentCount

	return stats, nil
}

// DeleteAlert 删除告警
func (s *AlertService) DeleteAlert(id uuid.UUID) error {
	result := s.db.Delete(&models.Alert{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("删除告警失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("告警不存在")
	}
	return nil
}

// CleanupOldAlerts 清理旧告警（定期任务）
func (s *AlertService) CleanupOldAlerts(olderThan time.Duration) error {
	cutoff := time.Now().Add(-olderThan)
	result := s.db.Where("created_at < ? AND status = ?", cutoff, "resolved").Delete(&models.Alert{})
	if result.Error != nil {
		return fmt.Errorf("清理旧告警失败: %w", result.Error)
	}
	return nil
}

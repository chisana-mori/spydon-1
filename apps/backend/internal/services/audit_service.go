package services

import (
    "fmt"
    "time"
    "encoding/json"

    "robusta-web/backend/internal/db"
    "robusta-web/backend/internal/models"

    "github.com/google/uuid"
    "gorm.io/datatypes"
)

// AuditService 审计服务
type AuditService struct {
	db *db.Database
}

// NewAuditService 创建新的审计服务
func NewAuditService(database *db.Database) *AuditService {
	return &AuditService{
		db: database,
	}
}

// LogAction 记录操作日志
func (s *AuditService) LogAction(
    userID, action, resourceType, resourceID string,
    details map[string]interface{},
    ipAddress, userAgent string,
) error {
    var detailsJSON datatypes.JSON
    if details != nil {
        if b, err := json.Marshal(details); err == nil {
            detailsJSON = datatypes.JSON(b)
        }
    }
    auditLog := &models.AuditLog{
        UserID:       userID,
        Action:       action,
        ResourceType: resourceType,
        ResourceID:   resourceID,
        Details:      detailsJSON,
        IPAddress:    ipAddress,
        UserAgent:    userAgent,
    }

	if err := s.db.Create(auditLog).Error; err != nil {
		return fmt.Errorf("记录审计日志失败: %w", err)
	}

	return nil
}

// AuditFilters 审计日志过滤条件
type AuditFilters struct {
	UserID       string
	Action       string
	ResourceType string
	ResourceID   string
	Since        *time.Time
	Until        *time.Time
}

// GetAuditLogs 获取审计日志列表
func (s *AuditService) GetAuditLogs(page, limit int, filters AuditFilters) ([]models.AuditLog, int64, error) {
	var auditLogs []models.AuditLog
	var total int64

	// 构建查询
	query := s.db.Model(&models.AuditLog{})

	// 应用过滤条件
	if filters.UserID != "" {
		query = query.Where("user_id = ?", filters.UserID)
	}
	if filters.Action != "" {
		query = query.Where("action = ?", filters.Action)
	}
	if filters.ResourceType != "" {
		query = query.Where("resource_type = ?", filters.ResourceType)
	}
	if filters.ResourceID != "" {
		query = query.Where("resource_id = ?", filters.ResourceID)
	}
	if filters.Since != nil {
		query = query.Where("created_at >= ?", filters.Since)
	}
	if filters.Until != nil {
		query = query.Where("created_at <= ?", filters.Until)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("获取审计日志总数失败: %w", err)
	}

	// 分页查询
	offset := (page - 1) * limit
	if err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&auditLogs).Error; err != nil {
		return nil, 0, fmt.Errorf("获取审计日志列表失败: %w", err)
	}

	return auditLogs, total, nil
}

// GetAuditLogByID 根据ID获取审计日志
func (s *AuditService) GetAuditLogByID(id uuid.UUID) (*models.AuditLog, error) {
	var auditLog models.AuditLog
	if err := s.db.First(&auditLog, "id = ?", id).Error; err != nil {
		return nil, fmt.Errorf("获取审计日志失败: %w", err)
	}
	return &auditLog, nil
}

// GetAuditStats 获取审计统计信息
func (s *AuditService) GetAuditStats(days int) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// 时间范围
	since := time.Now().AddDate(0, 0, -days)

	// 按操作类型统计
	var actionStats []struct {
		Action string `json:"action"`
		Count  int64  `json:"count"`
	}

	if err := s.db.Model(&models.AuditLog{}).
		Select("action, count(*) as count").
		Where("created_at >= ?", since).
		Group("action").
		Order("count DESC").
		Find(&actionStats).Error; err != nil {
		return nil, fmt.Errorf("获取操作统计失败: %w", err)
	}
	stats["by_action"] = actionStats

	// 按用户统计
	var userStats []struct {
		UserID string `json:"user_id"`
		Count  int64  `json:"count"`
	}

	if err := s.db.Model(&models.AuditLog{}).
		Select("user_id, count(*) as count").
		Where("created_at >= ? AND user_id != ''", since).
		Group("user_id").
		Order("count DESC").
		Limit(10).
		Find(&userStats).Error; err != nil {
		return nil, fmt.Errorf("获取用户统计失败: %w", err)
	}
	stats["by_user"] = userStats

	// 按资源类型统计
	var resourceStats []struct {
		ResourceType string `json:"resource_type"`
		Count        int64  `json:"count"`
	}

	if err := s.db.Model(&models.AuditLog{}).
		Select("resource_type, count(*) as count").
		Where("created_at >= ? AND resource_type != ''", since).
		Group("resource_type").
		Order("count DESC").
		Find(&resourceStats).Error; err != nil {
		return nil, fmt.Errorf("获取资源类型统计失败: %w", err)
	}
	stats["by_resource_type"] = resourceStats

	// 每日活动统计
	var dailyStats []struct {
		Date  string `json:"date"`
		Count int64  `json:"count"`
	}

	if err := s.db.Model(&models.AuditLog{}).
		Select("DATE(created_at) as date, count(*) as count").
		Where("created_at >= ?", since).
		Group("DATE(created_at)").
		Order("date").
		Find(&dailyStats).Error; err != nil {
		return nil, fmt.Errorf("获取每日统计失败: %w", err)
	}
	stats["daily_activity"] = dailyStats

	// 总数统计
	var totalCount int64
	if err := s.db.Model(&models.AuditLog{}).
		Where("created_at >= ?", since).
		Count(&totalCount).Error; err != nil {
		return nil, fmt.Errorf("获取总数统计失败: %w", err)
	}
	stats["total"] = totalCount

	return stats, nil
}

// CleanupOldAuditLogs 清理旧的审计日志
func (s *AuditService) CleanupOldAuditLogs(olderThan time.Duration) error {
	cutoff := time.Now().Add(-olderThan)
	result := s.db.Where("created_at < ?", cutoff).Delete(&models.AuditLog{})
	if result.Error != nil {
		return fmt.Errorf("清理旧审计日志失败: %w", result.Error)
	}
	return nil
}

// GetUserActivity 获取用户活动记录
func (s *AuditService) GetUserActivity(userID string, limit int) ([]models.AuditLog, error) {
	var auditLogs []models.AuditLog
	if err := s.db.Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Find(&auditLogs).Error; err != nil {
		return nil, fmt.Errorf("获取用户活动记录失败: %w", err)
	}
	return auditLogs, nil
}

// GetResourceActivity 获取资源活动记录
func (s *AuditService) GetResourceActivity(resourceType, resourceID string, limit int) ([]models.AuditLog, error) {
	var auditLogs []models.AuditLog
	if err := s.db.Where("resource_type = ? AND resource_id = ?", resourceType, resourceID).
		Order("created_at DESC").
		Limit(limit).
		Find(&auditLogs).Error; err != nil {
		return nil, fmt.Errorf("获取资源活动记录失败: %w", err)
	}
	return auditLogs, nil
}

// GetSecurityEvents 获取安全相关事件
func (s *AuditService) GetSecurityEvents(limit int) ([]models.AuditLog, error) {
	securityActions := []string{
		"login_failed",
		"unauthorized_access",
		"permission_denied",
		"token_expired",
		"suspicious_activity",
	}

	var auditLogs []models.AuditLog
	if err := s.db.Where("action IN (?)", securityActions).
		Order("created_at DESC").
		Limit(limit).
		Find(&auditLogs).Error; err != nil {
		return nil, fmt.Errorf("获取安全事件失败: %w", err)
	}
	return auditLogs, nil
}

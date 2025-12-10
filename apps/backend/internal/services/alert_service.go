package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"robusta-web/backend/internal/db"
	"robusta-web/backend/internal/logger"
	"robusta-web/backend/internal/models"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// AlertService 告警服务
type AlertService struct {
	db             *db.Database
	clusterService *ClusterService
	auditService   *AuditService
	rcaService     *RCAService
	payloadStorage PayloadStorage
}

// NewAlertService 创建新的告警服务
func NewAlertService(
	database *db.Database,
	clusterService *ClusterService,
	auditService *AuditService,
	rcaService *RCAService,
	payloadStorage PayloadStorage,
) *AlertService {
	return &AlertService{
		db:             database,
		clusterService: clusterService,
		auditService:   auditService,
		rcaService:     rcaService,
		payloadStorage: payloadStorage,
	}
}

// ProcessAlertRequest 处理告警请求参数
type ProcessAlertRequest struct {
	Fingerprint string
	ClusterID   string
	Title       string
	Description string
	Severity    string
	Status      string
	Labels      map[string]interface{}
	Annotations map[string]interface{}
	StartsAt    *time.Time
	EndsAt      *time.Time
	RawBody     []byte
	ClientIP    string
	UserAgent   string
}

// ProcessAlert 处理告警接收逻辑
func (s *AlertService) ProcessAlert(ctx context.Context, req ProcessAlertRequest) (*uint64, error) {
	// 验证严重级别
	if !isValidSeverity(req.Severity) {
		return nil, fmt.Errorf("无效的严重级别: %s", req.Severity)
	}

	// 保存原始payload（可选）
	var payloadKey string
	if s.payloadStorage != nil && len(req.RawBody) > 0 {
		key, err := s.savePayload(ctx, fmt.Sprintf("alerts/%s", req.ClusterID), req.RawBody, "application/json")
		if err != nil {
			// 记录错误但不中断流程
			logger.L().Warn("保存原始告警数据失败", zap.Error(err), zap.String("cluster_id", req.ClusterID))
		} else {
			payloadKey = key
		}
	}

	alert := &models.Alert{
		Fingerprint:   req.Fingerprint,
		ClusterID:     req.ClusterID,
		Title:         req.Title,
		Description:   req.Description,
		Severity:      req.Severity,
		Status:        req.Status,
		Labels:        toJSON(req.Labels),
		Annotations:   toJSON(req.Annotations),
		StartsAt:      req.StartsAt,
		EndsAt:        req.EndsAt,
		RawPayloadKey: payloadKey,
	}

	// 使用 IngestConvertedAlert 处理后续逻辑
	if err := s.IngestConvertedAlert(ctx, alert, req.ClientIP, req.UserAgent); err != nil {
		return nil, err
	}

	return &alert.ID, nil
}

// IngestConvertedAlert 处理已转换的告警（保存、心跳、审计）
func (s *AlertService) IngestConvertedAlert(ctx context.Context, alert *models.Alert, clientIP, userAgent string) error {
	// 确保集群存在
	if s.clusterService != nil {
		_ = s.clusterService.UpdateHeartbeat(ctx, &models.Cluster{
			ClusterID: alert.ClusterID,
			Name:      alert.ClusterID,
			Status:    string(models.ClusterStatusActive),
		})
	}

	// 保存告警
	if err := s.CreateOrUpdateAlert(ctx, alert); err != nil {
		return fmt.Errorf("保存告警失败: %w", err)
	}

	// 记录审计日志
	if s.auditService != nil {
		_ = s.auditService.LogAction(0, "alert_ingested", "alert", &alert.ID, map[string]interface{}{
			"cluster_id":  alert.ClusterID,
			"fingerprint": alert.Fingerprint,
			"severity":    alert.Severity,
			"source":      "api",
		}, clientIP, userAgent)
	}

	// 触发Auto-RCA（仅在开启时）
	if s.rcaService != nil && alert.Status == string(models.AlertStatusFiring) {
		go func() {
			rcaRun, err := s.rcaService.TriggerRCAAnalysis(alert)
			if err != nil {
				logger.L().Debug("Auto-RCA未触发",
					zap.Uint64("alert_id", alert.ID),
					zap.String("reason", err.Error()),
				)
			} else if rcaRun != nil {
				logger.L().Info("Auto-RCA已触发",
					zap.Uint64("alert_id", alert.ID),
					zap.Uint64("run_id", rcaRun.ID),
					zap.String("status", rcaRun.Status),
				)
			}
		}()
	}

	return nil
}

// savePayload 保存原始数据
func (s *AlertService) savePayload(ctx context.Context, keyPrefix string, data []byte, contentType string) (string, error) {
	if s.payloadStorage == nil {
		return "", nil
	}

	// 使用 Save 方法，它会自动生成 ID
	key, err := s.payloadStorage.Save(ctx, keyPrefix, data, contentType)
	if err != nil {
		return "", err
	}

	return key, nil
}

// isValidSeverity 验证严重级别
func isValidSeverity(severity string) bool {
	switch models.AlertSeverity(severity) {
	case models.AlertSeverityInfo,
		models.AlertSeverityWarning,
		models.AlertSeverityError,
		models.AlertSeverityLow,
		models.AlertSeverityMedium,
		models.AlertSeverityHigh,
		models.AlertSeverityCritical:
		return true
	default:
		return false
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

// AlertTrendPoint 告警趋势点
type AlertTrendPoint struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

// CreateOrUpdateAlert 创建或更新告警
func (s *AlertService) CreateOrUpdateAlert(ctx context.Context, alert *models.Alert) error {
	db := s.dbWithContext(ctx)

	// 使用fingerprint和cluster_id作为唯一标识
	var existingAlert models.Alert
	result := db.Where("fingerprint = ? AND cluster_id = ?", alert.Fingerprint, alert.ClusterID).First(&existingAlert)

	if result.Error == nil {
		// 告警已存在，更新
		alert.ID = existingAlert.ID
		alert.CreatedAt = existingAlert.CreatedAt
		return db.Save(alert).Error
	} else if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		// 告警不存在，创建新的
		return db.Create(alert).Error
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
func (s *AlertService) GetAlertByID(id uint64) (*models.Alert, error) {
	var alert models.Alert
	if err := s.db.Preload("Cluster").Preload("RCARuns").First(&alert, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
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
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("告警不存在")
		}
		return nil, fmt.Errorf("获取告警失败: %w", err)
	}
	return &alert, nil
}

// UpdateAlertStatus 更新告警状态
func (s *AlertService) UpdateAlertStatus(id uint64, status string) error {
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

// GetAlertTrend 获取指定天数的告警趋势（按日统计）
func (s *AlertService) GetAlertTrend(days int) ([]AlertTrendPoint, error) {
	if days <= 0 {
		days = 30
	}
	if days > 365 {
		days = 365
	}

	today := time.Now().Truncate(24 * time.Hour)
	startDate := today.AddDate(0, 0, -(days - 1))

	var rawResults []struct {
		Date  time.Time
		Count int
	}

	if err := s.db.Model(&models.Alert{}).
		Select("DATE(created_at) as date, COUNT(*) as count").
		Where("created_at >= ?", startDate).
		Group("DATE(created_at)").
		Order("DATE(created_at)").
		Scan(&rawResults).Error; err != nil {
		return nil, fmt.Errorf("获取告警趋势失败: %w", err)
	}

	countMap := make(map[string]int)
	for _, result := range rawResults {
		key := result.Date.Format("2006-01-02")
		countMap[key] = result.Count
	}

	trend := make([]AlertTrendPoint, 0, days)
	for i := 0; i < days; i++ {
		date := startDate.AddDate(0, 0, i)
		key := date.Format("2006-01-02")
		trend = append(trend, AlertTrendPoint{
			Date:  key,
			Count: countMap[key],
		})
	}

	return trend, nil
}

// DeleteAlert 删除告警
func (s *AlertService) DeleteAlert(id uint64) error {
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
	result := s.db.Where("created_at < ? AND status = ?", cutoff, models.AlertStatusResolved).Delete(&models.Alert{})
	if result.Error != nil {
		return fmt.Errorf("清理旧告警失败: %w", result.Error)
	}
	return nil
}

func (s *AlertService) dbWithContext(ctx context.Context) *gorm.DB {
	if ctx == nil {
		return s.db.DB
	}
	return s.db.WithContext(ctx)
}

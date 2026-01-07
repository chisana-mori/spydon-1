package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"robusta-web/backend/internal/db"
	"robusta-web/backend/internal/models"

	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	SettingKeyAutoRCA = "auto_rca"
)

// AutoRCAConfig Auto-RCA配置
type AutoRCAConfig struct {
	Enabled           bool                   `json:"enabled"`
	RateLimit         int                    `json:"rate_limit"`
	Period            int                    `json:"period"`
	AllowedSeverities []models.AlertSeverity `json:"allowed_severities"`
}

// DefaultAllowedSeverities 返回默认允许的告警级别
func DefaultAllowedSeverities() []models.AlertSeverity {
	return []models.AlertSeverity{
		models.AlertSeverityCritical,
		models.AlertSeverityHigh,
		models.AlertSeverityError,
	}
}

// GetDefaultAutoRCAConfig 获取默认配置（用于初始化）
func GetDefaultAutoRCAConfig() *AutoRCAConfig {
	return &AutoRCAConfig{
		Enabled:           false, // 默认关闭，需要手动开启
		RateLimit:         10,
		Period:            60,
		AllowedSeverities: DefaultAllowedSeverities(),
	}
}

// SystemSettingService 系统设置服务
type SystemSettingService struct {
	db *db.Database
}

// NewSystemSettingService 创建新的系统设置服务
func NewSystemSettingService(database *db.Database) *SystemSettingService {
	return &SystemSettingService{
		db: database,
	}
}

// GetSetting 获取指定Key的设置
func (s *SystemSettingService) GetSetting(key string) (*models.SystemSetting, error) {
	var setting models.SystemSetting
	if err := s.db.Where("setting_key = ?", key).First(&setting).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("获取设置失败: %w", err)
	}
	return &setting, nil
}

// SetSetting 设置指定Key的值
func (s *SystemSettingService) SetSetting(key string, value interface{}, description string) (*models.SystemSetting, error) {
	jsonBytes, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("序列化设置值失败: %w", err)
	}

	setting := models.SystemSetting{
		SettingKey:  key,
		Value:       datatypes.JSON(jsonBytes),
		Description: description,
	}

	if err := s.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "setting_key"}},
		DoUpdates: clause.AssignmentColumns([]string{"value", "description", "updated_at"}),
	}).Create(&setting).Error; err != nil {
		return nil, fmt.Errorf("更新设置失败: %w", err)
	}

	return &setting, nil
}

// GetAutoRCAConfig 获取Auto-RCA配置
func (s *SystemSettingService) GetAutoRCAConfig() (*AutoRCAConfig, time.Time, error) {
	setting, err := s.GetSetting(SettingKeyAutoRCA)
	if err != nil {
		return nil, time.Time{}, err
	}

	if setting == nil {
		// 数据库中尚未初始化时返回默认配置
		defaultConfig := GetDefaultAutoRCAConfig()
		// 初始化到数据库，避免后续重复报错
		_, _ = s.SetSetting(SettingKeyAutoRCA, defaultConfig, "System Default Auto-RCA Configuration")
		return defaultConfig, time.Time{}, nil
	}

	var config AutoRCAConfig
	if err := json.Unmarshal(setting.Value, &config); err != nil {
		return nil, time.Time{}, fmt.Errorf("解析Auto-RCA配置失败: %w", err)
	}

	if len(config.AllowedSeverities) == 0 {
		config.AllowedSeverities = DefaultAllowedSeverities()
	}

	return &config, setting.UpdatedAt, nil
}

// ListSettings 列出所有设置
func (s *SystemSettingService) ListSettings() ([]models.SystemSetting, error) {
	var settings []models.SystemSetting
	if err := s.db.Find(&settings).Error; err != nil {
		return nil, fmt.Errorf("获取设置列表失败: %w", err)
	}
	return settings, nil
}

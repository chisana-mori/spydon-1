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
)

const (
	SettingKeyAutoRCA = "auto_rca"
)

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
	if err := s.db.Where("key = ?", key).First(&setting).Error; err != nil {
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

	var setting models.SystemSetting
	err = s.db.Where("key = ?", key).First(&setting).Error

	switch {
	case err == nil:
		// 更新
		setting.Value = datatypes.JSON(jsonBytes)
		if description != "" {
			setting.Description = description
		}
		if saveErr := s.db.Save(&setting).Error; saveErr != nil {
			return nil, fmt.Errorf("更新设置失败: %w", saveErr)
		}
	case errors.Is(err, gorm.ErrRecordNotFound):
		// 创建
		setting = models.SystemSetting{
			Key:         key,
			Value:       datatypes.JSON(jsonBytes),
			Description: description,
		}
		if createErr := s.db.Create(&setting).Error; createErr != nil {
			return nil, fmt.Errorf("创建设置失败: %w", createErr)
		}
	default:
		return nil, fmt.Errorf("查询设置失败: %w", err)
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
		return defaultConfig, time.Time{}, nil
	}

	var config AutoRCAConfig
	if err := json.Unmarshal(setting.Value, &config); err != nil {
		return nil, time.Time{}, fmt.Errorf("解析Auto-RCA配置失败: %w", err)
	}

	if len(config.AllowedSeverities) == 0 {
		config.AllowedSeverities = defaultAllowedSeverities()
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

package services

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"robusta-web/backend/internal/db"
	"robusta-web/backend/internal/models"

	"gorm.io/gorm"
)

// APIKeyService API Key服务
type APIKeyService struct {
	db *db.Database
}

// NewAPIKeyService 创建API Key服务实例
func NewAPIKeyService(database *db.Database) *APIKeyService {
	return &APIKeyService{db: database}
}

// GenerateAPIKey 生成新的API Key
func (s *APIKeyService) GenerateAPIKey(userID uint64, name string, expiresAt *time.Time, permissions string) (*models.APIKey, string, error) {
	keyBytes := make([]byte, 32)
	if _, err := rand.Read(keyBytes); err != nil {
		return nil, "", fmt.Errorf("生成随机密钥失败: %w", err)
	}

	rawKey := base64.URLEncoding.EncodeToString(keyBytes)

	fullKey := fmt.Sprintf("rsk_%s", rawKey)

	keyPrefix := fullKey[:12]

	hash := sha256.Sum256([]byte(fullKey))
	hashedKey := base64.URLEncoding.EncodeToString(hash[:])

	apiKey := &models.APIKey{
		UserID:      userID,
		Name:        name,
		Key:         hashedKey,
		KeyPrefix:   keyPrefix,
		ExpiresAt:   expiresAt,
		IsActive:    true,
		Permissions: permissions,
	}

	if err := s.db.Create(apiKey).Error; err != nil {
		return nil, "", fmt.Errorf("创建API Key失败: %w", err)
	}

	return apiKey, fullKey, nil
}

// ListAPIKeys 获取用户的API Key列表
func (s *APIKeyService) ListAPIKeys(userID uint64) ([]models.APIKey, error) {
	var apiKeys []models.APIKey

	err := s.db.Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&apiKeys).Error
	if err != nil {
		return nil, fmt.Errorf("查询API Key列表失败: %w", err)
	}

	return apiKeys, nil
}

// GetAPIKey 根据ID获取API Key
func (s *APIKeyService) GetAPIKey(id uint64, userID uint64) (*models.APIKey, error) {
	var apiKey models.APIKey

	err := s.db.Where("id = ? AND user_id = ?", id, userID).
		First(&apiKey).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("API Key不存在")
		}
		return nil, fmt.Errorf("查询API Key失败: %w", err)
	}

	return &apiKey, nil
}

// DeleteAPIKey 删除API Key
func (s *APIKeyService) DeleteAPIKey(id uint64, userID uint64) error {
	result := s.db.Where("id = ? AND user_id = ?", id, userID).
		Delete(&models.APIKey{})

	if result.Error != nil {
		return fmt.Errorf("删除API Key失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("API Key不存在或无权限删除")
	}

	return nil
}

// ValidateAPIKey 验证API Key是否有效
func (s *APIKeyService) ValidateAPIKey(rawKey string) (*models.APIKey, error) {
	hash := sha256.Sum256([]byte(rawKey))
	hashedKey := base64.URLEncoding.EncodeToString(hash[:])

	var apiKey models.APIKey
	err := s.db.Where("`key` = ? AND is_active = ?", hashedKey, true).
		Preload("User").
		First(&apiKey).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("无效的API Key")
		}
		return nil, fmt.Errorf("验证API Key失败: %w", err)
	}

	if apiKey.ExpiresAt != nil && apiKey.ExpiresAt.Before(time.Now()) {
		return nil, fmt.Errorf("API Key已过期")
	}

	now := time.Now()
	apiKey.LastUsedAt = &now
	s.db.Model(&apiKey).Update("last_used_at", now)

	return &apiKey, nil
}

// UpdateAPIKeyStatus 更新API Key状态
func (s *APIKeyService) UpdateAPIKeyStatus(id uint64, userID uint64, isActive bool) error {
	result := s.db.Model(&models.APIKey{}).
		Where("id = ? AND user_id = ?", id, userID).
		Update("is_active", isActive)

	if result.Error != nil {
		return fmt.Errorf("更新API Key状态失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("API Key不存在或无权限更新")
	}

	return nil
}

// ListAllAPIKeys 管理员获取所有API Key列表（带分页）
func (s *APIKeyService) ListAllAPIKeys(page, limit int) ([]models.APIKey, int64, error) {
	var apiKeys []models.APIKey
	var total int64

	// 计算总数
	if err := s.db.Model(&models.APIKey{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计API Key总数失败: %w", err)
	}

	// 分页查询
	offset := (page - 1) * limit
	err := s.db.Preload("User").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&apiKeys).Error
	if err != nil {
		return nil, 0, fmt.Errorf("查询API Key列表失败: %w", err)
	}

	return apiKeys, total, nil
}

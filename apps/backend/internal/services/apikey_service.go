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

	"github.com/google/uuid"
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
func (s *APIKeyService) GenerateAPIKey(userID uuid.UUID, name string, expiresAt *time.Time, permissions string) (*models.APIKey, string, error) {
	// 生成随机的API Key（32字节 = 256位）
	keyBytes := make([]byte, 32)
	if _, err := rand.Read(keyBytes); err != nil {
		return nil, "", fmt.Errorf("生成随机密钥失败: %w", err)
	}

	// 将密钥编码为base64字符串
	rawKey := base64.URLEncoding.EncodeToString(keyBytes)

	// 添加前缀以便识别
	fullKey := fmt.Sprintf("rsk_%s", rawKey)

	// 提取前缀用于显示（前12个字符）
	keyPrefix := fullKey[:12]

	// 对完整密钥进行SHA256哈希存储
	hash := sha256.Sum256([]byte(fullKey))
	hashedKey := base64.URLEncoding.EncodeToString(hash[:])

	// 创建API Key记录
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

	// 返回API Key记录和原始密钥（仅此一次）
	return apiKey, fullKey, nil
}

// ListAPIKeys 获取用户的API Key列表
func (s *APIKeyService) ListAPIKeys(userID uuid.UUID) ([]models.APIKey, error) {
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
func (s *APIKeyService) GetAPIKey(id uuid.UUID, userID uuid.UUID) (*models.APIKey, error) {
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
func (s *APIKeyService) DeleteAPIKey(id uuid.UUID, userID uuid.UUID) error {
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
	// 对输入的密钥进行哈希
	hash := sha256.Sum256([]byte(rawKey))
	hashedKey := base64.URLEncoding.EncodeToString(hash[:])

	var apiKey models.APIKey
	err := s.db.Where("key = ? AND is_active = ?", hashedKey, true).
		Preload("User").
		First(&apiKey).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("无效的API Key")
		}
		return nil, fmt.Errorf("验证API Key失败: %w", err)
	}

	// 检查是否过期
	if apiKey.ExpiresAt != nil && apiKey.ExpiresAt.Before(time.Now()) {
		return nil, fmt.Errorf("API Key已过期")
	}

	// 更新最后使用时间
	now := time.Now()
	apiKey.LastUsedAt = &now
	s.db.Model(&apiKey).Update("last_used_at", now)

	return &apiKey, nil
}

// UpdateAPIKeyStatus 更新API Key状态
func (s *APIKeyService) UpdateAPIKeyStatus(id uuid.UUID, userID uuid.UUID, isActive bool) error {
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

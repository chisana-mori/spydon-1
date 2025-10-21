package services

import (
	"errors"
	"fmt"

	"robusta-web/backend/internal/db"
	"robusta-web/backend/internal/models"
	"robusta-web/backend/internal/utils"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UserService 用户管理服务
type UserService struct {
	db *db.Database
}

// NewUserService 创建用户服务实例
func NewUserService(database *db.Database) *UserService {
	return &UserService{
		db: database,
	}
}

// UserListItem 用户列表项
type UserListItem struct {
	ID            string `json:"id"`
	Username      string `json:"username"`
	Email         string `json:"email"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
	IsAdmin       bool   `json:"is_admin"`
	EmailVerified bool   `json:"email_verified"`
	Provider      string `json:"provider"`
	LastLoginAt   string `json:"last_login_at,omitempty"`
	CreatedAt     string `json:"created_at"`
}

// GetUsers 获取用户列表
func (s *UserService) GetUsers(page, limit int, keyword string) ([]UserListItem, int64, error) {
	var users []models.User
	var total int64

	query := s.db.DB.Model(&models.User{})

	// 关键词搜索
	if keyword != "" {
		query = query.Where("username ILIKE ? OR email ILIKE ? OR name ILIKE ?",
			"%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计用户数量失败: %w", err)
	}

	// 分页查询
	offset := (page - 1) * limit
	if err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&users).Error; err != nil {
		return nil, 0, fmt.Errorf("查询用户列表失败: %w", err)
	}

	// 转换为列表项（应用掩码保护敏感信息）
	items := make([]UserListItem, len(users))
	for i, user := range users {
		// 对敏感信息进行掩码处理
		maskedUsername, maskedEmail, maskedName := utils.MaskUserInfo(user.Username, user.Email, user.Name)

		items[i] = UserListItem{
			ID:            user.ID.String(),
			Username:      maskedUsername,
			Email:         maskedEmail,
			Name:          maskedName,
			Picture:       user.Picture,
			IsAdmin:       user.IsAdmin,
			EmailVerified: user.EmailVerified,
			Provider:      user.Provider,
			CreatedAt:     user.CreatedAt.Format("2006-01-02 15:04:05"),
		}
		if user.LastLoginAt != nil {
			items[i].LastLoginAt = user.LastLoginAt.Format("2006-01-02 15:04:05")
		}
	}

	return items, total, nil
}

// GetUserByID 根据ID获取用户
func (s *UserService) GetUserByID(userID string) (*models.User, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("无效的用户ID: %w", err)
	}

	var user models.User
	if err := s.db.DB.Where("id = ?", uid).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("用户不存在")
		}
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}

	return &user, nil
}

// SetUserAdmin 设置用户管理员权限
func (s *UserService) SetUserAdmin(userID string, isAdmin bool) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("无效的用户ID: %w", err)
	}

	var user models.User
	if err := s.db.DB.Where("id = ?", uid).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("用户不存在")
		}
		return fmt.Errorf("查询用户失败: %w", err)
	}

	// 更新管理员状态
	user.IsAdmin = isAdmin
	if err := s.db.DB.Save(&user).Error; err != nil {
		return fmt.Errorf("更新用户权限失败: %w", err)
	}

	return nil
}

// DeleteUser 删除用户（软删除）
func (s *UserService) DeleteUser(userID string) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("无效的用户ID: %w", err)
	}

	var user models.User
	if err := s.db.DB.Where("id = ?", uid).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("用户不存在")
		}
		return fmt.Errorf("查询用户失败: %w", err)
	}

	// 软删除
	if err := s.db.DB.Delete(&user).Error; err != nil {
		return fmt.Errorf("删除用户失败: %w", err)
	}

	return nil
}

// GetAdminCount 获取管理员数量
func (s *UserService) GetAdminCount() (int64, error) {
	var count int64
	if err := s.db.DB.Model(&models.User{}).Where("is_admin = ?", true).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("统计管理员数量失败: %w", err)
	}
	return count, nil
}

// UpdateUserProfile 更新用户资料
func (s *UserService) UpdateUserProfile(userID string, name, picture string) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("无效的用户ID: %w", err)
	}

	var user models.User
	if err := s.db.DB.Where("id = ?", uid).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("用户不存在")
		}
		return fmt.Errorf("查询用户失败: %w", err)
	}

	// 更新资料
	if name != "" {
		user.Name = name
	}
	if picture != "" {
		user.Picture = picture
	}

	if err := s.db.DB.Save(&user).Error; err != nil {
		return fmt.Errorf("更新用户资料失败: %w", err)
	}

	return nil
}

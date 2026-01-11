package services

import (
	"errors"
	"fmt"

	"robusta-web/backend/internal/db"
	"robusta-web/backend/internal/models"
	"robusta-web/backend/internal/utils"

	"gorm.io/gorm"
)

// UserService 用户管理服务
type UserService struct {
	db *db.Database
}

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

	query := s.db.Model(&models.User{})

	if keyword != "" {
		query = query.Where("username ILIKE ? OR email ILIKE ? OR name ILIKE ?",
			"%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计用户数量失败: %w", err)
	}

	offset := (page - 1) * limit
	if err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&users).Error; err != nil {
		return nil, 0, fmt.Errorf("查询用户列表失败: %w", err)
	}

	items := make([]UserListItem, len(users))
	for i, user := range users {
		maskedUsername, maskedEmail, maskedName := utils.MaskUserInfo(user.Username, user.Email, user.Name)

		items[i] = UserListItem{
			ID:            models.FormatID(user.ID),
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

func (s *UserService) GetUserByID(userID uint64) (*models.User, error) {
	var user models.User
	if err := s.db.Where("id = ?", userID).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("用户不存在")
		}
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}

	return &user, nil
}

func (s *UserService) SetUserAdmin(userID uint64, isAdmin bool) error {
	var user models.User
	if err := s.db.Where("id = ?", userID).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("用户不存在")
		}
		return fmt.Errorf("查询用户失败: %w", err)
	}

	user.IsAdmin = isAdmin
	if err := s.db.Save(&user).Error; err != nil {
		return fmt.Errorf("更新用户权限失败: %w", err)
	}

	return nil
}

func (s *UserService) DeleteUser(userID uint64) error {
	var user models.User
	if err := s.db.Where("id = ?", userID).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("用户不存在")
		}
		return fmt.Errorf("查询用户失败: %w", err)
	}

	if err := s.db.Delete(&user).Error; err != nil {
		return fmt.Errorf("删除用户失败: %w", err)
	}

	return nil
}

func (s *UserService) GetAdminCount() (int64, error) {
	var count int64
	if err := s.db.Model(&models.User{}).Where("is_admin = ?", true).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("统计管理员数量失败: %w", err)
	}
	return count, nil
}

func (s *UserService) UpdateUserProfile(userID uint64, name, picture string) error {
	var user models.User
	if err := s.db.Where("id = ?", userID).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("用户不存在")
		}
		return fmt.Errorf("查询用户失败: %w", err)
	}

	if name != "" {
		user.Name = name
	}
	if picture != "" {
		user.Picture = picture
	}

	if err := s.db.Save(&user).Error; err != nil {
		return fmt.Errorf("更新用户资料失败: %w", err)
	}

	return nil
}

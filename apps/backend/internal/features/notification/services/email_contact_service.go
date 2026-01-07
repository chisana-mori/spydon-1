package services

import (
	"errors"
	"fmt"
	"strings"

	"robusta-web/backend/internal/db"
	"robusta-web/backend/internal/models"

	"gorm.io/gorm"
)

// EmailContactService 邮件联系人服务
type EmailContactService struct {
	db *db.Database
}

// NewEmailContactService 创建邮件联系人服务
func NewEmailContactService(database *db.Database) *EmailContactService {
	return &EmailContactService{db: database}
}

// ListContacts 获取联系人列表
func (s *EmailContactService) ListContacts(query EmailContactListQuery) ([]EmailContactResponse, int64, error) {
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.Size <= 0 || query.Size > 100 {
		query.Size = 20
	}

	var contacts []models.EmailContact
	var total int64

	tx := s.db.DB.Model(&models.EmailContact{})

	// 关键词搜索
	if query.Keyword != "" {
		keyword := "%" + query.Keyword + "%"
		tx = tx.Where("name LIKE ? OR address LIKE ?", keyword, keyword)
	}

	// 获取总数
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("查询联系人总数失败: %w", err)
	}

	// 分页查询
	offset := (query.Page - 1) * query.Size
	if err := tx.Order("created_at DESC").Offset(offset).Limit(query.Size).Find(&contacts).Error; err != nil {
		return nil, 0, fmt.Errorf("查询联系人列表失败: %w", err)
	}

	// 转换响应
	result := make([]EmailContactResponse, len(contacts))
	for i, c := range contacts {
		result[i] = s.toResponse(c)
	}

	return result, total, nil
}

// GetContact 获取单个联系人详情
func (s *EmailContactService) GetContact(id uint64) (*EmailContactResponse, error) {
	var contact models.EmailContact
	if err := s.db.DB.First(&contact, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("联系人不存在")
		}
		return nil, fmt.Errorf("查询联系人失败: %w", err)
	}
	resp := s.toResponse(contact)
	return &resp, nil
}

// CreateContact 创建联系人
func (s *EmailContactService) CreateContact(req CreateEmailContactRequest) (*models.EmailContact, error) {
	// 规范化地址格式
	address := normalizeAddresses(req.Address)

	contact := &models.EmailContact{
		Name:    req.Name,
		Address: address,
	}

	if err := s.db.DB.Create(contact).Error; err != nil {
		return nil, fmt.Errorf("创建联系人失败: %w", err)
	}

	return contact, nil
}

// UpdateContact 更新联系人
func (s *EmailContactService) UpdateContact(id uint64, req UpdateEmailContactRequest) (*models.EmailContact, error) {
	var contact models.EmailContact
	if err := s.db.DB.First(&contact, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("联系人不存在")
		}
		return nil, fmt.Errorf("查询联系人失败: %w", err)
	}

	if req.Name != nil {
		contact.Name = *req.Name
	}
	if req.Address != nil {
		contact.Address = normalizeAddresses(*req.Address)
	}

	if err := s.db.DB.Save(&contact).Error; err != nil {
		return nil, fmt.Errorf("更新联系人失败: %w", err)
	}

	return &contact, nil
}

// DeleteContact 删除联系人
func (s *EmailContactService) DeleteContact(id uint64) error {
	result := s.db.DB.Delete(&models.EmailContact{}, id)
	if result.Error != nil {
		return fmt.Errorf("删除联系人失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("联系人不存在")
	}
	return nil
}

// ParseAddresses 解析逗号分隔的邮箱地址
func ParseAddresses(addressStr string) []string {
	if addressStr == "" {
		return nil
	}
	parts := strings.Split(addressStr, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		addr := strings.TrimSpace(p)
		if addr != "" {
			result = append(result, addr)
		}
	}
	return result
}

// normalizeAddresses 规范化地址格式（去除多余空格）
func normalizeAddresses(addressStr string) string {
	addrs := ParseAddresses(addressStr)
	return strings.Join(addrs, ",")
}

// toResponse 转换为响应格式
func (s *EmailContactService) toResponse(c models.EmailContact) EmailContactResponse {
	return EmailContactResponse{
		ID:        c.ID,
		Name:      c.Name,
		Address:   c.Address,
		CreatedAt: c.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt: c.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
}

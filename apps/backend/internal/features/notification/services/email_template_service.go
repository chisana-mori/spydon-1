package services

import (
	"encoding/json"
	"errors"
	"fmt"

	"robusta-web/backend/internal/db"
	"robusta-web/backend/internal/models"

	"gorm.io/gorm"
)

// EmailTemplateService 邮件模板服务
type EmailTemplateService struct {
	db *db.Database
}

// NewEmailTemplateService 创建邮件模板服务
func NewEmailTemplateService(database *db.Database) *EmailTemplateService {
	return &EmailTemplateService{db: database}
}

// ListTemplates 获取模板列表
func (s *EmailTemplateService) ListTemplates(query EmailTemplateListQuery) ([]EmailTemplateResponse, int64, error) {
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.Size <= 0 || query.Size > 100 {
		query.Size = 20
	}

	var templates []models.EmailTemplate
	var total int64

	tx := s.db.DB.Model(&models.EmailTemplate{})

	// 关键词搜索
	if query.Keyword != "" {
		keyword := "%" + query.Keyword + "%"
		tx = tx.Where("name LIKE ? OR title LIKE ?", keyword, keyword)
	}

	// 启用状态过滤
	if query.IsEnabled != nil {
		tx = tx.Where("is_enabled = ?", *query.IsEnabled)
	}

	// 获取总数
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("查询模板总数失败: %w", err)
	}

	// 分页查询
	offset := (query.Page - 1) * query.Size
	if err := tx.Order("created_at DESC").Offset(offset).Limit(query.Size).Find(&templates).Error; err != nil {
		return nil, 0, fmt.Errorf("查询模板列表失败: %w", err)
	}

	// 转换响应
	result := make([]EmailTemplateResponse, len(templates))
	for i, t := range templates {
		result[i] = s.toResponse(t)
	}

	return result, total, nil
}

// GetTemplate 获取单个模板详情
func (s *EmailTemplateService) GetTemplate(id uint64) (*EmailTemplateResponse, error) {
	var template models.EmailTemplate
	if err := s.db.DB.First(&template, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("模板不存在")
		}
		return nil, fmt.Errorf("查询模板失败: %w", err)
	}
	resp := s.toResponse(template)
	return &resp, nil
}

// CreateTemplate 创建模板
func (s *EmailTemplateService) CreateTemplate(req CreateEmailTemplateRequest) (*models.EmailTemplate, error) {
	// 检查名称唯一性
	var count int64
	if err := s.db.DB.Model(&models.EmailTemplate{}).Where("name = ?", req.Name).Count(&count).Error; err != nil {
		return nil, fmt.Errorf("检查模板名称失败: %w", err)
	}
	if count > 0 {
		return nil, fmt.Errorf("模板名称已存在")
	}

	// 序列化参数
	paramsJSON := ""
	// 序列化参数
	data, err := json.Marshal(req.Params)
	if err != nil {
		return nil, fmt.Errorf("序列化参数失败: %w", err)
	}
	paramsJSON = string(data)

	isEnabled := true
	if req.IsEnabled != nil {
		isEnabled = *req.IsEnabled
	}

	template := &models.EmailTemplate{
		Name:      req.Name,
		Title:     req.Title,
		Body:      req.Body,
		Params:    paramsJSON,
		IsEnabled: isEnabled,
	}

	if err := s.db.DB.Create(template).Error; err != nil {
		return nil, fmt.Errorf("创建模板失败: %w", err)
	}

	return template, nil
}

// UpdateTemplate 更新模板
func (s *EmailTemplateService) UpdateTemplate(id uint64, req UpdateEmailTemplateRequest) (*models.EmailTemplate, error) {
	var template models.EmailTemplate
	if err := s.db.DB.First(&template, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("模板不存在")
		}
		return nil, fmt.Errorf("查询模板失败: %w", err)
	}

	// 检查名称唯一性（如果更新了名称）
	if req.Name != nil && *req.Name != template.Name {
		var count int64
		if err := s.db.DB.Model(&models.EmailTemplate{}).Where("name = ? AND id != ?", *req.Name, id).Count(&count).Error; err != nil {
			return nil, fmt.Errorf("检查模板名称失败: %w", err)
		}
		if count > 0 {
			return nil, fmt.Errorf("模板名称已存在")
		}
		template.Name = *req.Name
	}

	if req.Title != nil {
		template.Title = *req.Title
	}
	if req.Body != nil {
		template.Body = *req.Body
	}
	if req.Params != nil {
		data, err := json.Marshal(req.Params)
		if err != nil {
			return nil, fmt.Errorf("序列化参数失败: %w", err)
		}
		template.Params = string(data)
	}
	if req.IsEnabled != nil {
		template.IsEnabled = *req.IsEnabled
	}

	if err := s.db.DB.Save(&template).Error; err != nil {
		return nil, fmt.Errorf("更新模板失败: %w", err)
	}

	return &template, nil
}

// DeleteTemplate 删除模板
func (s *EmailTemplateService) DeleteTemplate(id uint64) error {
	result := s.db.DB.Delete(&models.EmailTemplate{}, id)
	if result.Error != nil {
		return fmt.Errorf("删除模板失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("模板不存在")
	}
	return nil
}

// toResponse 转换为响应格式
func (s *EmailTemplateService) toResponse(t models.EmailTemplate) EmailTemplateResponse {
	params, _ := ParseParams(t.Params)
	return EmailTemplateResponse{
		ID:        t.ID,
		Name:      t.Name,
		Title:     t.Title,
		Body:      t.Body,
		Params:    params,
		IsEnabled: t.IsEnabled,
		CreatedAt: t.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt: t.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
}

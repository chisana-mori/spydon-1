package services

import (
	"context"
	"errors"
	"fmt"

	"robusta-web/backend/internal/db"
	"robusta-web/backend/internal/models/navy"

	"gorm.io/gorm"
)

// TaintManagementService 污点管理服务
type TaintManagementService struct {
	db *db.Database
}

// NewTaintManagementService 创建污点管理服务
func NewTaintManagementService(database *db.Database) *TaintManagementService {
	return &TaintManagementService{db: database}
}

// ListTaints 获取污点列表
func (s *TaintManagementService) ListTaints(ctx context.Context, query TaintListQuery) (*ListResponse, error) {
	var taints []navy.TaintManagement
	var total int64

	tx := s.db.WithContext(ctx).Model(&navy.TaintManagement{})

	// 关键字搜索
	if query.Keyword != "" {
		keyword := "%" + query.Keyword + "%"
		tx = tx.Where("`key` LIKE ? OR value LIKE ? OR description LIKE ?", keyword, keyword, keyword)
	}

	// Effect筛选
	if query.Effect != "" {
		tx = tx.Where("effect = ?", query.Effect)
	}

	// 状态筛选
	if query.Status != nil {
		tx = tx.Where("status = ?", *query.Status)
	}

	// 计数
	if err := tx.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("统计污点数量失败: %w", err)
	}

	// 分页
	page := query.Page
	if page <= 0 {
		page = 1
	}
	size := query.Size
	if size <= 0 {
		size = 10
	}
	offset := (page - 1) * size

	if err := tx.Offset(offset).Limit(size).Find(&taints).Error; err != nil {
		return nil, fmt.Errorf("查询污点列表失败: %w", err)
	}

	// 转换为DTO
	dtos := make([]TaintManagementDTO, len(taints))
	for i, taint := range taints {
		dtos[i] = TaintManagementDTO{
			ID:          taint.ID,
			Key:         taint.Key,
			Value:       taint.Value,
			Effect:      taint.Effect,
			Description: taint.Description,
			Type:        taint.Type,
			Status:      taint.Status,
		}
	}

	return &ListResponse{
		List:  dtos,
		Total: total,
		Page:  page,
		Size:  size,
	}, nil
}

// GetTaint 获取单个污点
func (s *TaintManagementService) GetTaint(ctx context.Context, id int) (*TaintManagementDTO, error) {
	var taint navy.TaintManagement
	if err := s.db.WithContext(ctx).First(&taint, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("污点不存在")
		}
		return nil, fmt.Errorf("查询污点失败: %w", err)
	}

	return &TaintManagementDTO{
		ID:          taint.ID,
		Key:         taint.Key,
		Value:       taint.Value,
		Effect:      taint.Effect,
		Description: taint.Description,
		Type:        taint.Type,
		Status:      taint.Status,
	}, nil
}

// CreateTaint 创建污点
func (s *TaintManagementService) CreateTaint(ctx context.Context, req CreateTaintRequest) (*TaintManagementDTO, error) {
	taint := &navy.TaintManagement{
		Key:         req.Key,
		Value:       req.Value,
		Effect:      req.Effect,
		Description: req.Description,
		Type:        req.Type,
		Status:      req.Status,
	}

	if err := s.db.WithContext(ctx).Create(taint).Error; err != nil {
		return nil, fmt.Errorf("创建污点失败: %w", err)
	}

	return &TaintManagementDTO{
		ID:          taint.ID,
		Key:         taint.Key,
		Value:       taint.Value,
		Effect:      taint.Effect,
		Description: taint.Description,
		Type:        taint.Type,
		Status:      taint.Status,
	}, nil
}

// UpdateTaint 更新污点
func (s *TaintManagementService) UpdateTaint(ctx context.Context, id int, req UpdateTaintRequest) (*TaintManagementDTO, error) {
	var taint navy.TaintManagement
	if err := s.db.WithContext(ctx).First(&taint, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("污点不存在")
		}
		return nil, fmt.Errorf("查询污点失败: %w", err)
	}

	updates := make(map[string]interface{})
	if req.Key != nil {
		updates["key"] = *req.Key
	}
	if req.Value != nil {
		updates["value"] = *req.Value
	}
	if req.Effect != nil {
		updates["effect"] = *req.Effect
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.Type != nil {
		updates["type"] = *req.Type
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}

	if len(updates) > 0 {
		if err := s.db.WithContext(ctx).Model(&taint).Updates(updates).Error; err != nil {
			return nil, fmt.Errorf("更新污点失败: %w", err)
		}
	}

	return &TaintManagementDTO{
		ID:          taint.ID,
		Key:         taint.Key,
		Value:       taint.Value,
		Effect:      taint.Effect,
		Description: taint.Description,
		Type:        taint.Type,
		Status:      taint.Status,
	}, nil
}

// DeleteTaint 删除污点
func (s *TaintManagementService) DeleteTaint(ctx context.Context, id int) error {
	result := s.db.WithContext(ctx).Delete(&navy.TaintManagement{}, id)
	if result.Error != nil {
		return fmt.Errorf("删除污点失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("污点不存在")
	}
	return nil
}

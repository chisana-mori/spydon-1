package services

import (
	"context"
	"fmt"

	"robusta-web/backend/internal/db"
	"robusta-web/backend/internal/models/navy"

	"gorm.io/gorm"
)

// LabelManagementService 标签管理服务
type LabelManagementService struct {
	db *db.NavyDatabase
}

// NewLabelManagementService 创建标签管理服务
func NewLabelManagementService(database *db.NavyDatabase) *LabelManagementService {
	return &LabelManagementService{db: database}
}

// ListLabels 获取标签列表
func (s *LabelManagementService) ListLabels(ctx context.Context, query LabelListQuery) (*ListResponse, error) {
	var labels []navy.LabelManagement
	var total int64

	tx := s.db.WithContext(ctx).Model(&navy.LabelManagement{})

	// 关键字搜索
	if query.Keyword != "" {
		keyword := "%" + query.Keyword + "%"
		tx = tx.Where("name LIKE ? OR `key` LIKE ?", keyword, keyword)
	}

	// 来源筛选
	if query.Source != nil {
		tx = tx.Where("source = ?", *query.Source)
	}

	// 状态筛选
	if query.Status != nil {
		tx = tx.Where("status = ?", *query.Status)
	}

	// 计数
	if err := tx.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("统计标签数量失败: %w", err)
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

	// Preload LabelValues
	if err := tx.Preload("LabelValues").Offset(offset).Limit(size).Find(&labels).Error; err != nil {
		return nil, fmt.Errorf("查询标签列表失败: %w", err)
	}

	// 转换为DTO
	dtos := make([]LabelManagementDTO, len(labels))
	for i, label := range labels {
		values := make([]string, len(label.LabelValues))
		for j, val := range label.LabelValues {
			values[j] = val.Value
		}
		dtos[i] = LabelManagementDTO{
			ID:     label.ID,
			Name:   label.Name,
			Key:    label.Key,
			Source: label.Source,
			Status: label.Status,
			Values: values,
		}
	}

	return &ListResponse{
		List:  dtos,
		Total: total,
		Page:  page,
		Size:  size,
	}, nil
}

// GetLabel 获取单个标签
func (s *LabelManagementService) GetLabel(ctx context.Context, id int) (*LabelManagementDTO, error) {
	var label navy.LabelManagement
	if err := s.db.WithContext(ctx).Preload("LabelValues").First(&label, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("标签不存在")
		}
		return nil, fmt.Errorf("查询标签失败: %w", err)
	}

	values := make([]string, len(label.LabelValues))
	for i, val := range label.LabelValues {
		values[i] = val.Value
	}

	return &LabelManagementDTO{
		ID:     label.ID,
		Name:   label.Name,
		Key:    label.Key,
		Source: label.Source,
		Status: label.Status,
		Values: values,
	}, nil
}

// CreateLabel 创建标签
func (s *LabelManagementService) CreateLabel(ctx context.Context, req CreateLabelRequest) (*LabelManagementDTO, error) {
	return s.DoCreateLabel(ctx, req)
}

func (s *LabelManagementService) DoCreateLabel(ctx context.Context, req CreateLabelRequest) (*LabelManagementDTO, error) {
	// 检查 key 是否已存在
	var count int64
	if err := s.db.WithContext(ctx).Model(&navy.LabelManagement{}).Where("`key` = ?", req.Key).Count(&count).Error; err != nil {
		return nil, fmt.Errorf("检查标签Key失败: %w", err)
	}
	if count > 0 {
		return nil, fmt.Errorf("标签Key '%s' 已存在", req.Key)
	}

	label := &navy.LabelManagement{
		Name:   req.Name,
		Key:    req.Key,
		Source: req.Source,
		Status: req.Status,
	}

	// 开启事务
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(label).Error; err != nil {
			return err
		}

		if len(req.Values) > 0 {
			labelValues := make([]navy.LabelValue, len(req.Values))
			for i, v := range req.Values {
				labelValues[i] = navy.LabelValue{
					LabelID: label.ID,
					Value:   v,
				}
			}
			if err := tx.Create(&labelValues).Error; err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("创建标签失败: %w", err)
	}

	return &LabelManagementDTO{
		ID:     label.ID,
		Name:   label.Name,
		Key:    label.Key,
		Source: label.Source,
		Status: label.Status,
		Values: req.Values,
	}, nil
}

// UpdateLabel 更新标签
func (s *LabelManagementService) UpdateLabel(ctx context.Context, id int, req UpdateLabelRequest) (*LabelManagementDTO, error) {
	var label navy.LabelManagement
	if err := s.db.WithContext(ctx).Preload("LabelValues").First(&label, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("标签不存在")
		}
		return nil, fmt.Errorf("查询标签失败: %w", err)
	}

	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Key != nil {
		updates["key"] = *req.Key
	}
	if req.Source != nil {
		updates["source"] = *req.Source
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if len(updates) > 0 {
			if err := tx.Model(&label).Updates(updates).Error; err != nil {
				return err
			}
		}

		// 如果提供了Values，则全量替换
		if req.Values != nil {
			// 删除旧值
			if err := tx.Where("label_id = ?", label.ID).Delete(&navy.LabelValue{}).Error; err != nil {
				return err
			}
			// 插入新值
			if len(*req.Values) > 0 {
				newValues := make([]navy.LabelValue, len(*req.Values))
				for i, v := range *req.Values {
					newValues[i] = navy.LabelValue{
						LabelID: label.ID,
						Value:   v,
					}
				}
				if err := tx.Create(&newValues).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("更新标签失败: %w", err)
	}

	// 重新获取包含最新Values的Label
	// 或者直接构造返回
	finalValues := []string{}
	if req.Values != nil {
		finalValues = *req.Values
	} else {
		// 如果没更新values，返回旧的
		for _, v := range label.LabelValues {
			finalValues = append(finalValues, v.Value)
		}
	}

	return &LabelManagementDTO{
		ID:     label.ID,
		Name:   label.Name,
		Key:    label.Key,
		Source: label.Source,
		Status: label.Status,
		Values: finalValues,
	}, nil
}

// DeleteLabel 删除标签
func (s *LabelManagementService) DeleteLabel(ctx context.Context, id int) error {
	result := s.db.WithContext(ctx).Delete(&navy.LabelManagement{}, id)
	if result.Error != nil {
		return fmt.Errorf("删除标签失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("标签不存在")
	}
	return nil
}

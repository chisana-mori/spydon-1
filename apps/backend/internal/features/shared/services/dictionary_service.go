package services

import (
	"errors"
	"fmt"

	"robusta-web/backend/internal/db"
	"robusta-web/backend/internal/models"

	"gorm.io/gorm"
)

// DictionaryService 字典管理服务
type DictionaryService struct {
	db *db.Database
}

// NewDictionaryService 创建字典服务
func NewDictionaryService(database *db.Database) *DictionaryService {
	return &DictionaryService{db: database}
}

// =====================================================
// 字典 CRUD
// =====================================================

// ListDictionaries 获取字典列表
func (s *DictionaryService) ListDictionaries(query DictionaryListQuery) ([]DictionaryResponse, error) {
	var dictionaries []models.Dictionary

	tx := s.db.Model(&models.Dictionary{})

	// 模块筛选
	if query.Module != "" {
		tx = tx.Where("module = ?", query.Module)
	}

	// 启用状态筛选
	if query.IsEnabled != nil {
		tx = tx.Where("is_enabled = ?", *query.IsEnabled)
	}

	// 关键字搜索
	if query.Keyword != "" {
		keyword := "%" + query.Keyword + "%"
		tx = tx.Where("name LIKE ? OR code LIKE ? OR description LIKE ?", keyword, keyword, keyword)
	}

	// 排序
	tx = tx.Order("sort_order ASC, id ASC")

	if err := tx.Find(&dictionaries).Error; err != nil {
		return nil, fmt.Errorf("查询字典列表失败: %w", err)
	}

	// 转换为响应并统计字典项数量
	result := make([]DictionaryResponse, 0, len(dictionaries))
	for _, dict := range dictionaries {
		var itemCount int64
		s.db.Model(&models.DictionaryItem{}).Where("dictionary_id = ? AND is_enabled = true", dict.ID).Count(&itemCount)

		result = append(result, DictionaryResponse{
			ID:             dict.ID,
			Code:           dict.Code,
			Name:           dict.Name,
			Module:         dict.Module,
			Description:    dict.Description,
			IsEnabled:      dict.IsEnabled,
			KeySameAsValue: dict.KeySameAsValue,
			SortOrder:      dict.SortOrder,
			ItemCount:      int(itemCount),
		})
	}

	return result, nil
}

// GetDictionary 获取单个字典详情（含字典项）
func (s *DictionaryService) GetDictionary(id uint64) (*DictionaryDetailResponse, error) {
	var dict models.Dictionary
	if err := s.db.First(&dict, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("查询字典失败: %w", err)
	}

	// 获取字典项
	var items []models.DictionaryItem
	if err := s.db.Where("dictionary_id = ?", id).Order("sort_order ASC, id ASC").Find(&items).Error; err != nil {
		return nil, fmt.Errorf("查询字典项失败: %w", err)
	}

	// 转换响应
	itemResponses := make([]DictionaryItemResponse, 0, len(items))
	for _, item := range items {
		itemResponses = append(itemResponses, DictionaryItemResponse{
			ID:          item.ID,
			Key:         item.Key,
			Value:       item.Value,
			Description: item.Description,
			IsDefault:   item.IsDefault,
			IsEnabled:   item.IsEnabled,
			SortOrder:   item.SortOrder,
			Extra:       item.Extra,
		})
	}

	return &DictionaryDetailResponse{
		DictionaryResponse: DictionaryResponse{
			ID:             dict.ID,
			Code:           dict.Code,
			Name:           dict.Name,
			Module:         dict.Module,
			Description:    dict.Description,
			IsEnabled:      dict.IsEnabled,
			KeySameAsValue: dict.KeySameAsValue,
			SortOrder:      dict.SortOrder,
			ItemCount:      len(itemResponses),
		},
		Items: itemResponses,
	}, nil
}

// CreateDictionary 创建字典
func (s *DictionaryService) CreateDictionary(req CreateDictionaryRequest) (*models.Dictionary, error) {
	// 检查 code 是否已存在
	var existCount int64
	if err := s.db.Model(&models.Dictionary{}).Where("code = ?", req.Code).Count(&existCount).Error; err != nil {
		return nil, fmt.Errorf("检查字典编码失败: %w", err)
	}
	if existCount > 0 {
		return nil, fmt.Errorf("字典编码 '%s' 已存在", req.Code)
	}

	keySameAsValue := true
	if req.KeySameAsValue != nil {
		keySameAsValue = *req.KeySameAsValue
	}

	dict := &models.Dictionary{
		Code:           req.Code,
		Name:           req.Name,
		Module:         req.Module,
		Description:    req.Description,
		IsEnabled:      true,
		KeySameAsValue: keySameAsValue,
		SortOrder:      req.SortOrder,
	}

	// 强制插入 KeySameAsValue (即使是 false)
	if err := s.db.Select("Code", "Name", "Module", "Description", "IsEnabled", "KeySameAsValue", "SortOrder").Create(dict).Error; err != nil {
		return nil, fmt.Errorf("创建字典失败: %w", err)
	}

	return dict, nil
}

// UpdateDictionary 更新字典
func (s *DictionaryService) UpdateDictionary(id uint64, req UpdateDictionaryRequest) (*models.Dictionary, error) {
	var dict models.Dictionary
	if err := s.db.First(&dict, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("字典不存在")
		}
		return nil, fmt.Errorf("查询字典失败: %w", err)
	}

	// 更新字段
	updates := make(map[string]interface{})
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Module != "" {
		updates["module"] = req.Module
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	if req.IsEnabled != nil {
		updates["is_enabled"] = *req.IsEnabled
	}
	if req.KeySameAsValue != nil {
		updates["key_same_as_value"] = *req.KeySameAsValue
	}
	if req.SortOrder != nil {
		updates["sort_order"] = *req.SortOrder
	}

	if len(updates) > 0 {
		if err := s.db.Model(&dict).Updates(updates).Error; err != nil {
			return nil, fmt.Errorf("更新字典失败: %w", err)
		}
	}

	// 重新加载
	if err := s.db.First(&dict, id).Error; err != nil {
		return nil, fmt.Errorf("重新加载字典失败: %w", err)
	}

	return &dict, nil
}

// DeleteDictionary 删除字典（级联删除字典项）
func (s *DictionaryService) DeleteDictionary(id uint64) error {
	result := s.db.Delete(&models.Dictionary{}, id)
	if result.Error != nil {
		return fmt.Errorf("删除字典失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("字典不存在")
	}
	return nil
}

// =====================================================
// 字典项 CRUD
// =====================================================

// CreateDictionaryItem 创建字典项
func (s *DictionaryService) CreateDictionaryItem(dictID uint64, req CreateDictionaryItemRequest) (*models.DictionaryItem, error) {
	// 检查字典是否存在
	var dict models.Dictionary
	if err := s.db.First(&dict, dictID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("字典不存在")
		}
		return nil, fmt.Errorf("查询字典失败: %w", err)
	}

	// 检查 key 是否已存在
	var existCount int64
	if err := s.db.Model(&models.DictionaryItem{}).Where("dictionary_id = ? AND `key` = ?", dictID, req.Key).Count(&existCount).Error; err != nil {
		return nil, fmt.Errorf("检查字典项Key失败: %w", err)
	}
	if existCount > 0 {
		return nil, fmt.Errorf("字典项 Key '%s' 已存在", req.Key)
	}

	item := &models.DictionaryItem{
		DictionaryID: dictID,
		Key:          req.Key,
		Value:        req.Value,
		Description:  req.Description,
		IsDefault:    req.IsDefault,
		IsEnabled:    true,
		SortOrder:    req.SortOrder,
		Extra:        req.Extra,
	}

	if err := s.db.Create(item).Error; err != nil {
		return nil, fmt.Errorf("创建字典项失败: %w", err)
	}

	return item, nil
}

// UpdateDictionaryItem 更新字典项
func (s *DictionaryService) UpdateDictionaryItem(itemID uint64, req UpdateDictionaryItemRequest) (*models.DictionaryItem, error) {
	var item models.DictionaryItem
	if err := s.db.First(&item, itemID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("字典项不存在")
		}
		return nil, fmt.Errorf("查询字典项失败: %w", err)
	}

	updates := make(map[string]interface{})
	if req.Key != "" {
		updates["key"] = req.Key
	}
	if req.Value != "" {
		updates["value"] = req.Value
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	if req.IsDefault != nil {
		updates["is_default"] = *req.IsDefault
	}
	if req.IsEnabled != nil {
		updates["is_enabled"] = *req.IsEnabled
	}
	if req.SortOrder != nil {
		updates["sort_order"] = *req.SortOrder
	}
	if req.Extra != "" {
		updates["extra"] = req.Extra
	}

	if len(updates) > 0 {
		if err := s.db.Model(&item).Updates(updates).Error; err != nil {
			return nil, fmt.Errorf("更新字典项失败: %w", err)
		}
	}

	// 重新加载
	if err := s.db.First(&item, itemID).Error; err != nil {
		return nil, fmt.Errorf("重新加载字典项失败: %w", err)
	}

	return &item, nil
}

// DeleteDictionaryItem 删除字典项
func (s *DictionaryService) DeleteDictionaryItem(itemID uint64) error {
	result := s.db.Delete(&models.DictionaryItem{}, itemID)
	if result.Error != nil {
		return fmt.Errorf("删除字典项失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("字典项不存在")
	}
	return nil
}

// BatchUpdateItems 批量更新字典项
func (s *DictionaryService) BatchUpdateItems(dictID uint64, req BatchUpdateDictionaryItemsRequest) error {
	// 检查字典是否存在
	var dict models.Dictionary
	if err := s.db.First(&dict, dictID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("字典不存在")
		}
		return fmt.Errorf("查询字典失败: %w", err)
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		for _, item := range req.Items {
			if item.Delete && item.ID != nil {
				// 删除
				if err := tx.Delete(&models.DictionaryItem{}, *item.ID).Error; err != nil {
					return fmt.Errorf("删除字典项失败: %w", err)
				}
				continue
			}

			if item.ID != nil {
				// 更新
				if err := tx.Model(&models.DictionaryItem{}).Where("id = ?", *item.ID).Updates(map[string]interface{}{
					"key":         item.Key,
					"value":       item.Value,
					"description": item.Description,
					"is_default":  item.IsDefault,
					"is_enabled":  item.IsEnabled,
					"sort_order":  item.SortOrder,
					"extra":       item.Extra,
				}).Error; err != nil {
					return fmt.Errorf("更新字典项失败: %w", err)
				}
			} else {
				// 创建
				newItem := models.DictionaryItem{
					DictionaryID: dictID,
					Key:          item.Key,
					Value:        item.Value,
					Description:  item.Description,
					IsDefault:    item.IsDefault,
					IsEnabled:    item.IsEnabled,
					SortOrder:    item.SortOrder,
					Extra:        item.Extra,
				}
				if err := tx.Create(&newItem).Error; err != nil {
					return fmt.Errorf("创建字典项失败: %w", err)
				}
			}
		}
		return nil
	})
}

// =====================================================
// 通用接口 - 前端模版使用
// =====================================================

// GetItemsByCode 通过字典编码获取启用的字典项列表
// 这是所有前端模版的通用接口
func (s *DictionaryService) GetItemsByCode(code string) ([]DictionaryItemResponse, error) {
	// 查找字典
	var dict models.Dictionary
	if err := s.db.Where("code = ? AND is_enabled = true", code).First(&dict).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return []DictionaryItemResponse{}, nil // 返回空数组而非错误
		}
		return nil, fmt.Errorf("查询字典失败: %w", err)
	}

	// 查询启用的字典项
	var items []models.DictionaryItem
	if err := s.db.Where("dictionary_id = ? AND is_enabled = true", dict.ID).Order("sort_order ASC, id ASC").Find(&items).Error; err != nil {
		return nil, fmt.Errorf("查询字典项失败: %w", err)
	}

	result := make([]DictionaryItemResponse, 0, len(items))
	for _, item := range items {
		result = append(result, DictionaryItemResponse{
			ID:          item.ID,
			Key:         item.Key,
			Value:       item.Value,
			Description: item.Description,
			IsDefault:   item.IsDefault,
			IsEnabled:   item.IsEnabled,
			SortOrder:   item.SortOrder,
			Extra:       item.Extra,
		})
	}

	return result, nil
}

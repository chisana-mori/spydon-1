package services

import (
	"context"
	"errors"
	"fmt"

	"robusta-web/backend/internal/db"
	"robusta-web/backend/internal/models/navy"

	"gorm.io/gorm"
)

// DeviceAppService 设备应用管理服务
type DeviceAppService struct {
	db *db.Database
}

// NewDeviceAppService 创建设备应用管理服务
func NewDeviceAppService(database *db.Database) *DeviceAppService {
	return &DeviceAppService{db: database}
}

// ListDeviceApps 获取设备应用列表
func (s *DeviceAppService) ListDeviceApps(ctx context.Context, query DeviceAppListQuery) (*ListResponse, error) {
	var apps []navy.DeviceApp
	var total int64

	tx := s.db.WithContext(ctx).Model(&navy.DeviceApp{})

	// 关键字搜索
	if query.Keyword != "" {
		keyword := "%" + query.Keyword + "%"
		tx = tx.Where("app_id LIKE ? OR name LIKE ? OR owner LIKE ?", keyword, keyword, keyword)
	}

	if query.Type != nil {
		tx = tx.Where("type = ?", *query.Type)
	}

	// 状态筛选
	if query.Status != nil {
		tx = tx.Where("status = ?", *query.Status)
	}

	if err := tx.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("统计设备应用数量失败: %w", err)
	}

	page := query.Page
	if page <= 0 {
		page = 1
	}
	size := query.Size
	if size <= 0 {
		size = 10
	}
	offset := (page - 1) * size

	if err := tx.Offset(offset).Limit(size).Find(&apps).Error; err != nil {
		return nil, fmt.Errorf("查询设备应用列表失败: %w", err)
	}

	dtos := make([]DeviceAppDTO, len(apps))
	for i, app := range apps {
		dtos[i] = DeviceAppDTO{
			ID:          app.ID,
			AppId:       app.AppId,
			Type:        app.Type,
			Name:        app.Name,
			Owner:       app.Owner,
			Feature:     app.Feature,
			Description: app.Description,
			Status:      app.Status,
		}
	}

	return &ListResponse{
		List:  dtos,
		Total: total,
		Page:  page,
		Size:  size,
	}, nil
}

// GetDeviceApp 获取单个设备应用
func (s *DeviceAppService) GetDeviceApp(ctx context.Context, id int) (*DeviceAppDTO, error) {
	var app navy.DeviceApp
	if err := s.db.WithContext(ctx).First(&app, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("设备应用不存在")
		}
		return nil, fmt.Errorf("查询设备应用失败: %w", err)
	}

	return &DeviceAppDTO{
		ID:          app.ID,
		AppId:       app.AppId,
		Type:        app.Type,
		Name:        app.Name,
		Owner:       app.Owner,
		Feature:     app.Feature,
		Description: app.Description,
		Status:      app.Status,
	}, nil
}

// CreateDeviceApp 创建设备应用
func (s *DeviceAppService) CreateDeviceApp(ctx context.Context, req CreateDeviceAppRequest) (*DeviceAppDTO, error) {
	var count int64
	if err := s.db.WithContext(ctx).Model(&navy.DeviceApp{}).Where("app_id = ?", req.AppId).Count(&count).Error; err != nil {
		return nil, fmt.Errorf("检查AppId失败: %w", err)
	}
	if count > 0 {
		return nil, fmt.Errorf("AppId '%s' 已存在", req.AppId)
	}

	app := &navy.DeviceApp{
		AppId:       req.AppId,
		Type:        req.Type,
		Name:        req.Name,
		Owner:       req.Owner,
		Feature:     req.Feature,
		Description: req.Description,
		Status:      req.Status,
	}

	if err := s.db.WithContext(ctx).Create(app).Error; err != nil {
		return nil, fmt.Errorf("创建设备应用失败: %w", err)
	}

	return &DeviceAppDTO{
		ID:          app.ID,
		AppId:       app.AppId,
		Type:        app.Type,
		Name:        app.Name,
		Owner:       app.Owner,
		Feature:     app.Feature,
		Description: app.Description,
		Status:      app.Status,
	}, nil
}

// UpdateDeviceApp 更新设备应用
func (s *DeviceAppService) UpdateDeviceApp(ctx context.Context, id int, req UpdateDeviceAppRequest) (*DeviceAppDTO, error) {
	var app navy.DeviceApp
	if err := s.db.WithContext(ctx).First(&app, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("设备应用不存在")
		}
		return nil, fmt.Errorf("查询设备应用失败: %w", err)
	}

	updates := make(map[string]interface{})
	if req.AppId != nil {
		updates["app_id"] = *req.AppId
	}
	if req.Type != nil {
		updates["type"] = *req.Type
	}
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Owner != nil {
		updates["owner"] = *req.Owner
	}
	if req.Feature != nil {
		updates["feature"] = *req.Feature
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}

	if len(updates) > 0 {
		if err := s.db.WithContext(ctx).Model(&app).Updates(updates).Error; err != nil {
			return nil, fmt.Errorf("更新设备应用失败: %w", err)
		}
	}

	return &DeviceAppDTO{
		ID:          app.ID,
		AppId:       app.AppId,
		Type:        app.Type,
		Name:        app.Name,
		Owner:       app.Owner,
		Feature:     app.Feature,
		Description: app.Description,
		Status:      app.Status,
	}, nil
}

// DeleteDeviceApp 删除设备应用
func (s *DeviceAppService) DeleteDeviceApp(ctx context.Context, id int) error {
	result := s.db.WithContext(ctx).Delete(&navy.DeviceApp{}, id)
	if result.Error != nil {
		return fmt.Errorf("删除设备应用失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("设备应用不存在")
	}
	return nil
}

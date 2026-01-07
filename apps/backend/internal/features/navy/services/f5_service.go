package services

import (
	"context"
	"errors"
	"fmt"
	"robusta-web/backend/internal/models/navy"

	"gorm.io/gorm"
)

// F5InfoService provides business logic for F5 load balancer management.
type F5InfoService struct {
	db *gorm.DB
}

// NewF5InfoService creates a new F5InfoService.
func NewF5InfoService(db *gorm.DB) *F5InfoService {
	return &F5InfoService{db: db}
}

const (
	emptyString = ""
	defaultPage = 1
	defaultSize = 10
	maxSize     = 100
)

// GetF5Info retrieves a single F5 info record by ID.
func (s *F5InfoService) GetF5Info(ctx context.Context, id int) (*F5InfoResponse, error) {
	var model navy.F5Info
	err := s.db.WithContext(ctx).Preload("Cluster").
		Where("id = ? AND deleted = ?", id, emptyString).First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("F5 info with id %d not found", id)
		}
		return nil, fmt.Errorf("database error: %w", err)
	}
	return toF5InfoResponse(&model), nil
}

// ListF5Infos retrieves a paginated list of F5 info records with filtering.
func (s *F5InfoService) ListF5Infos(ctx context.Context, query *F5InfoQuery) (*F5InfoListResponse, error) {
	var models []navy.F5Info
	var total int64

	db := s.db.WithContext(ctx).Model(&navy.F5Info{}).Where("deleted = ?", emptyString)

	// Apply filters
	if query.Name != emptyString {
		db = db.Where("name LIKE ?", "%"+query.Name+"%")
	}
	if query.VIP != emptyString {
		db = db.Where("vip LIKE ?", "%"+query.VIP+"%")
	}
	if query.Port != emptyString {
		db = db.Where("port = ?", query.Port)
	}
	if query.AppID != emptyString {
		db = db.Where("appid LIKE ?", "%"+query.AppID+"%")
	}
	if query.InstanceGroup != emptyString {
		db = db.Where("instance_group LIKE ?", "%"+query.InstanceGroup+"%")
	}
	if query.Status != emptyString {
		db = db.Where("status = ?", query.Status)
	}
	if query.PoolName != emptyString {
		db = db.Where("pool_name LIKE ?", "%"+query.PoolName+"%")
	}
	if query.ClusterName != emptyString {
		db = db.Joins("LEFT JOIN robusta_hub.clusters ON clusters.id = f5_info.k8s_cluster_id").
			Where("clusters.name LIKE ?", "%"+query.ClusterName+"%")
	}

	// Count total records
	if err := db.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count F5 infos: %w", err)
	}

	// Adjust pagination
	if query.Page <= 0 {
		query.Page = defaultPage
	}
	if query.Size <= 0 || query.Size > maxSize {
		query.Size = defaultSize
	}

	// Fetch data with pagination and preloading
	err := db.Preload("Cluster").
		Offset((query.Page - 1) * query.Size).Limit(query.Size).
		Find(&models).Error
	if err != nil {
		return nil, fmt.Errorf("failed to list F5 infos: %w", err)
	}

	list := make([]*F5InfoResponse, 0, len(models))
	for i := range models {
		list = append(list, toF5InfoResponse(&models[i]))
	}

	return &F5InfoListResponse{
		List:  list,
		Page:  query.Page,
		Size:  query.Size,
		Total: total,
	}, nil
}

// UpdateF5Info updates an existing F5 info record.
func (s *F5InfoService) UpdateF5Info(ctx context.Context, id int, dto *F5InfoUpdateDTO) error {
	// Check if the record exists before updating
	var existing navy.F5Info
	if err := s.db.WithContext(ctx).Where("id = ? AND deleted = ?", id, emptyString).First(&existing).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("F5 info with id %d not found", id)
		}
		return fmt.Errorf("database error when checking existence: %w", err)
	}

	// Build update map
	updates := map[string]interface{}{
		"name":           dto.Name,
		"vip":            dto.VIP,
		"port":           dto.Port,
		"appid":          dto.AppID,
		"instance_group": dto.InstanceGroup,
		"status":         dto.Status,
		"pool_name":      dto.PoolName,
		"pool_status":    dto.PoolStatus,
		"pool_members":   dto.PoolMembers,
		"k8s_cluster_id": dto.ClusterID, // 使用现有列名
		"domains":        dto.Domains,
		"grafana_params": dto.GrafanaParams,
		"ignored":        dto.Ignored,
	}

	result := s.db.WithContext(ctx).Model(&navy.F5Info{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("failed to update F5 info: %w", result.Error)
	}

	return nil
}

// DeleteF5Info soft deletes an F5 info record.
func (s *F5InfoService) DeleteF5Info(ctx context.Context, id int) error {
	result := s.db.WithContext(ctx).Model(&navy.F5Info{}).
		Where("id = ? AND deleted = ?", id, emptyString).
		Update("deleted", "1")

	if result.Error != nil {
		return fmt.Errorf("failed to delete F5 info: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("F5 info with id %d not found", id)
	}
	return nil
}

// toF5InfoResponse converts navy.F5Info model to response DTO.
func toF5InfoResponse(m *navy.F5Info) *F5InfoResponse {
	if m == nil {
		return nil
	}

	resp := &F5InfoResponse{
		ID:            m.ID,
		Name:          m.Name,
		VIP:           m.VIP,
		Port:          m.Port,
		AppID:         m.AppID,
		InstanceGroup: m.InstanceGroup,
		Status:        m.Status,
		PoolName:      m.PoolName,
		PoolStatus:    m.PoolStatus,
		PoolMembers:   m.PoolMembers,
		Domains:       m.Domains,
		GrafanaParams: m.GrafanaParams,
		Ignored:       m.Ignored,
		CreatedAt:     m.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:     m.UpdatedAt.Format("2006-01-02 15:04:05"),
	}

	// Set cluster fields if cluster relation exists
	if m.ClusterID != nil {
		resp.ClusterID = m.ClusterID
		if m.Cluster != nil {
			resp.ClusterName = m.Cluster.Name
		}
	}

	return resp
}

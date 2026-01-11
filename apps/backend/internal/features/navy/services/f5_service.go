package services

import (
	"context"
	"errors"
	"fmt"
	"robusta-web/backend/internal/models"
	"robusta-web/backend/internal/models/navy"
	"sort"
	"strings"

	"gorm.io/gorm"
)

const (
	defaultPage = 1
	defaultSize = 10
	maxSize     = 100
)

// F5InfoService provides business logic for F5 load balancer management.
type F5InfoService struct {
	db *gorm.DB
}

// NewF5InfoService creates a new F5InfoService.
func NewF5InfoService(db *gorm.DB) *F5InfoService {
	return &F5InfoService{db: db}
}

// GetF5Info retrieves a single F5 info record by ID.
func (s *F5InfoService) GetF5Info(ctx context.Context, id int) (*F5InfoResponse, error) {
	var model navy.F5Info
	err := s.db.WithContext(ctx).
		Preload("Cluster", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, clustername")
		}).
		Where("id = ? AND deleted = ?", id, 0).
		First(&model).Error

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
	db := s.db.WithContext(ctx).Model(&navy.F5Info{}).Where("deleted = ?", 0)

	db = s.applyFilters(db, query)
	db = s.applyKeywordSearch(db, query.Keyword)

	sortBy, sortOrder, needsClusterSort := s.parseSortParams(query.SortBy, query.SortOrder)

	if !needsClusterSort {
		db = db.Order(fmt.Sprintf("%s %s", sortBy, sortOrder))
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count F5 infos: %w", err)
	}

	query = s.normalizePagination(query)

	var f5Infos []navy.F5Info
	err := db.Preload("Cluster").
		Offset((query.Page - 1) * query.Size).
		Limit(query.Size).
		Find(&f5Infos).Error
	if err != nil {
		return nil, fmt.Errorf("failed to list F5 infos: %w", err)
	}

	s.loadMissingClusters(ctx, f5Infos)

	list := s.convertToResponseList(f5Infos)

	if needsClusterSort {
		s.sortByClusterName(list, sortOrder)
	}

	return &F5InfoListResponse{
		List:  list,
		Page:  query.Page,
		Size:  query.Size,
		Total: total,
	}, nil
}

func (s *F5InfoService) applyFilters(db *gorm.DB, query *F5InfoQuery) *gorm.DB {
	if query.Name != "" {
		db = db.Where("name LIKE ?", "%"+query.Name+"%")
	}
	if query.VIP != "" {
		db = db.Where("vip LIKE ?", "%"+query.VIP+"%")
	}
	if query.Port != "" {
		db = db.Where("port = ?", query.Port)
	}
	if query.AppID != "" {
		db = db.Where("appid LIKE ?", "%"+query.AppID+"%")
	}
	if query.InstanceGroup != "" {
		db = db.Where("instance_group LIKE ?", "%"+query.InstanceGroup+"%")
	}
	if query.Status != "" {
		db = db.Where("status = ?", query.Status)
	}
	if query.PoolName != "" {
		db = db.Where("pool_name LIKE ?", "%"+query.PoolName+"%")
	}
	if query.ClusterName != "" {
		db = db.Where("k8s_cluster_id IN (SELECT id FROM k8s_clusters WHERE clustername LIKE ?)", "%"+query.ClusterName+"%")
	}

	return db
}

func (s *F5InfoService) applyKeywordSearch(db *gorm.DB, keyword string) *gorm.DB {
	if keyword == "" {
		return db
	}

	tokens := s.tokenizeKeyword(keyword)
	if len(tokens) == 0 {
		return db
	}

	var keywordQuery *gorm.DB
	for _, token := range tokens {
		if token == "" {
			continue
		}

		likeToken := "%" + token + "%"
		subQuery := s.db.Where("name LIKE ?", likeToken).
			Or("vip LIKE ?", likeToken).
			Or("appid LIKE ?", likeToken).
			Or("pool_members LIKE ?", likeToken).
			Or("k8s_cluster_id IN (SELECT id FROM k8s_clusters WHERE clustername LIKE ?)", likeToken)

		if keywordQuery == nil {
			keywordQuery = subQuery
		} else {
			keywordQuery = keywordQuery.Or(subQuery)
		}
	}

	if keywordQuery != nil {
		db = db.Where(keywordQuery)
	}

	return db
}

func (s *F5InfoService) tokenizeKeyword(keyword string) []string {
	tokens := strings.FieldsFunc(keyword, func(r rune) bool {
		return r == '\n' || r == ',' || r == ';' || r == ' '
	})

	result := make([]string, 0, len(tokens))
	for _, token := range tokens {
		trimmed := strings.TrimSpace(token)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result
}

// parseSortParams extracts and validates sort parameters.
func (s *F5InfoService) parseSortParams(sortBy, sortOrder string) (string, string, bool) {
	defaultSortBy := "id"
	defaultSortOrder := "DESC"

	if sortOrder == "ascend" || sortOrder == "asc" {
		defaultSortOrder = "ASC"
	}

	needsClusterSort := false

	if sortBy != "" {
		switch sortBy {
		case "name", "vip", "appid", "instance_group", "status", "pool_status", "ignored":
			defaultSortBy = sortBy
		case "cluster_name":
			needsClusterSort = true
		}
	}

	return defaultSortBy, defaultSortOrder, needsClusterSort
}

func (s *F5InfoService) normalizePagination(query *F5InfoQuery) *F5InfoQuery {
	if query.Page <= 0 {
		query.Page = defaultPage
	}
	if query.Size <= 0 || query.Size > maxSize {
		query.Size = defaultSize
	}
	return query
}

func (s *F5InfoService) loadMissingClusters(ctx context.Context, f5Infos []navy.F5Info) {
	var clusterIDs []uint
	for i := range f5Infos {
		if f5Infos[i].ClusterID != nil && f5Infos[i].Cluster == nil {
			clusterIDs = append(clusterIDs, *f5Infos[i].ClusterID)
		}
	}

	if len(clusterIDs) == 0 {
		return
	}

	var clusters []models.Cluster
	err := s.db.WithContext(ctx).
		Table("k8s_clusters").
		Select("id, clustername").
		Where("id IN ?", clusterIDs).
		Find(&clusters).Error
	if err != nil {
		return
	}

	clusterMap := make(map[uint]*models.Cluster, len(clusters))
	for i := range clusters {
		clusterMap[clusters[i].ID] = &clusters[i]
	}

	for i := range f5Infos {
		if f5Infos[i].ClusterID != nil && f5Infos[i].Cluster == nil {
			if cluster, ok := clusterMap[*f5Infos[i].ClusterID]; ok {
				f5Infos[i].Cluster = cluster
			}
		}
	}
}

func (s *F5InfoService) convertToResponseList(f5Infos []navy.F5Info) []*F5InfoResponse {
	list := make([]*F5InfoResponse, 0, len(f5Infos))
	for i := range f5Infos {
		list = append(list, toF5InfoResponse(&f5Infos[i]))
	}
	return list
}

// sortByClusterName sorts the response list by cluster name.
func (s *F5InfoService) sortByClusterName(list []*F5InfoResponse, sortOrder string) {
	sort.Slice(list, func(i, j int) bool {
		nameI := list[i].ClusterName
		nameJ := list[j].ClusterName
		if sortOrder == "ASC" {
			return nameI < nameJ
		}
		return nameI > nameJ
	})
}

// UpdateF5Info updates an existing F5 info record.
func (s *F5InfoService) UpdateF5Info(ctx context.Context, id int, dto *F5InfoUpdateDTO) error {
	var existing navy.F5Info
	err := s.db.WithContext(ctx).
		Where("id = ? AND deleted = ?", id, 0).
		First(&existing).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("F5 info with id %d not found", id)
		}
		return fmt.Errorf("database error when checking existence: %w", err)
	}

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
		"k8s_cluster_id": dto.ClusterID,
		"domains":        dto.Domains,
		"grafana_params": dto.GrafanaParams,
		"ignored":        dto.Ignored,
	}

	result := s.db.WithContext(ctx).
		Model(&navy.F5Info{}).
		Where("id = ?", id).
		Updates(updates)

	if result.Error != nil {
		return fmt.Errorf("failed to update F5 info: %w", result.Error)
	}

	return nil
}

// DeleteF5Info soft deletes an F5 info record.
func (s *F5InfoService) DeleteF5Info(ctx context.Context, id int) error {
	result := s.db.WithContext(ctx).
		Model(&navy.F5Info{}).
		Where("id = ? AND deleted = ?", id, 0).
		Update("deleted", 1)

	if result.Error != nil {
		return fmt.Errorf("failed to delete F5 info: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("F5 info with id %d not found", id)
	}

	return nil
}

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

	if m.ClusterID != nil {
		resp.ClusterID = m.ClusterID
	}
	if m.Cluster != nil {
		resp.ClusterName = m.Cluster.Name
	}

	return resp
}

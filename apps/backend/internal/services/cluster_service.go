package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"robusta-web/backend/internal/db"
	"robusta-web/backend/internal/models"

	"gorm.io/gorm"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// ClusterService 集群服务
type ClusterService struct {
	db *db.Database
}

// NewClusterService 创建新的集群服务
func NewClusterService(database *db.Database) *ClusterService {
	return &ClusterService{
		db: database,
	}
}

// ClusterSummary 集群概览信息
type ClusterSummary struct {
	TotalClusters  int64             `json:"total_clusters"`
	ActiveClusters int64             `json:"active_clusters"`
	TotalAlerts    int64             `json:"total_alerts"`
	CriticalAlerts int64             `json:"critical_alerts"`
	ClusterStats   []ClusterStats    `json:"cluster_stats"`
	AlertTrends    []AlertTrendPoint `json:"alert_trends"`
}

// ClusterStats 单个集群统计信息
type ClusterStats struct {
	Name          string     `json:"name"`
	ClusterID     string     `json:"cluster_id"`
	Status        string     `json:"status"`
	AlertCount    int64      `json:"alert_count"`
	CriticalCount int64      `json:"critical_count"`
	LastHeartbeat *time.Time `json:"last_heartbeat"`
}

// UpdateHeartbeat 更新集群心跳
func (s *ClusterService) UpdateHeartbeat(ctx context.Context, cluster *models.Cluster) error {
	return s.dbWithContext(ctx).Model(&models.Cluster{}).
		Where("name = ?", cluster.Name).
		Updates(map[string]interface{}{
			"last_heartbeat": time.Now(),
			"status":         models.ClusterStatusActive,
			"cluster_id":     cluster.ClusterID,
		}).Error
}

// GetClustersSummary 获取集群概览
func (s *ClusterService) GetClustersSummary() (*ClusterSummary, error) {
	summary := &ClusterSummary{}

	// 获取集群总数
	if err := s.db.Model(&models.Cluster{}).Count(&summary.TotalClusters).Error; err != nil {
		return nil, fmt.Errorf("获取集群总数失败: %w", err)
	}

	// 获取活跃集群数 (Status = active)
	if err := s.db.Model(&models.Cluster{}).
		Where("status = ?", models.ClusterStatusActive).
		Count(&summary.ActiveClusters).Error; err != nil {
		return nil, fmt.Errorf("获取活跃集群数失败: %w", err)
	}

	// 获取告警总数 (所有告警，包括已解决的，以匹配 cluster_stats 中的 total_alerts)
	if err := s.db.Model(&models.Alert{}).
		Count(&summary.TotalAlerts).Error; err != nil {
		return nil, fmt.Errorf("获取告警总数失败: %w", err)
	}

	// 获取严重告警数 (firing & critical)
	if err := s.db.Model(&models.Alert{}).
		Where("status = ? AND severity = ?", models.AlertStatusFiring, models.AlertSeverityCritical).
		Count(&summary.CriticalAlerts).Error; err != nil {
		return nil, fmt.Errorf("获取严重告警数失败: %w", err)
	}

	// 获取各集群统计信息
	clusterStats, err := s.getClusterStats()
	if err != nil {
		return nil, fmt.Errorf("获取集群统计失败: %w", err)
	}
	summary.ClusterStats = clusterStats

	// 趋势数据由前端单独获取，此处返回空数组
	summary.AlertTrends = []AlertTrendPoint{}

	return summary, nil
}

// getClusterStats 获取各集群统计信息
func (s *ClusterService) getClusterStats() ([]ClusterStats, error) {
	var stats []ClusterStats

	// 使用原生SQL查询获取集群统计信息
	query := `
		SELECT
			c.cluster_id,
			c.name,
			c.status,
			c.last_heartbeat,
			COALESCE(alert_counts.total_alerts, 0) as alert_count,
			COALESCE(alert_counts.critical_alerts, 0) as critical_count
		FROM clusters c
		LEFT JOIN (
			SELECT
				cluster_name,
				COUNT(*) as total_alerts,
				COUNT(CASE WHEN severity = 'critical' AND status = 'firing' THEN 1 END) as critical_alerts
			FROM spydon_alerts
			WHERE deleted_at IS NULL
			GROUP BY cluster_name
		) alert_counts ON c.name = alert_counts.cluster_name
		ORDER BY c.name
	`

	if err := s.db.Raw(query).Scan(&stats).Error; err != nil {
		return nil, err
	}

	return stats, nil
}

// ... (getAlertTrends omitted)

// GetClusters 获取集群列表
func (s *ClusterService) GetClusters(page, limit int, status string) ([]models.Cluster, int64, error) {
	var clusters []models.Cluster
	var total int64

	query := s.db.Model(&models.Cluster{})

	if status != "" {
		query = query.Where("status = ?", status)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("获取集群总数失败: %w", err)
	}

	// 分页查询
	offset := (page - 1) * limit
	if err := query.Order("name").Offset(offset).Limit(limit).Find(&clusters).Error; err != nil {
		return nil, 0, fmt.Errorf("获取集群列表失败: %w", err)
	}

	return clusters, total, nil
}

func (s *ClusterService) dbWithContext(ctx context.Context) *gorm.DB {
	if ctx == nil {
		return s.db.DB
	}
	return s.db.WithContext(ctx)
}

// GetClusterByID 根据ID获取集群
func (s *ClusterService) GetClusterByID(clusterName string) (*models.Cluster, error) {
	var cluster models.Cluster
	if err := s.db.Where("name = ?", clusterName).First(&cluster).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("集群不存在")
		}
		return nil, fmt.Errorf("获取集群失败: %w", err)
	}
	return &cluster, nil
}

// DeleteCluster 删除集群（硬删除）
func (s *ClusterService) DeleteCluster(clusterName string) error {
	// 硬删除集群
	result := s.db.Unscoped().Where("name = ?", clusterName).Delete(&models.Cluster{})
	if result.Error != nil {
		return fmt.Errorf("删除集群失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("集群不存在")
	}

	// 同时硬删除该集群的所有告警
	if err := s.db.Unscoped().Where("cluster_name = ?", clusterName).Delete(&models.Alert{}).Error; err != nil {
		return fmt.Errorf("删除集群告警失败: %w", err)
	}

	return nil
}

// UpdateClusterStatus 更新集群状态
// Deprecated: Status field is removed. This function should not be used or should be updated to do nothing.
func (s *ClusterService) UpdateClusterStatus(clusterName, status string) error {
	// Status field no longer exists.
	return nil
}

// CheckInactiveClusters 检查不活跃的集群
func (s *ClusterService) CheckInactiveClusters(threshold time.Duration) ([]models.Cluster, error) {
	var inactiveClusters []models.Cluster
	cutoff := time.Now().Add(-threshold)

	if err := s.db.Where("last_heartbeat < ? OR last_heartbeat IS NULL", cutoff).
		Find(&inactiveClusters).Error; err != nil {
		return nil, fmt.Errorf("检查不活跃集群失败: %w", err)
	}

	return inactiveClusters, nil
}

// validateKubeConfig 验证 KubeConfig 是否合法且可连接
func (s *ClusterService) validateKubeConfig(kubeConfig string) error {
	if kubeConfig == "" {
		return nil
	}

	config, err := clientcmd.RESTConfigFromKubeConfig([]byte(kubeConfig))
	if err != nil {
		return fmt.Errorf("invalid kubeconfig format: %w", err)
	}

	// 设置超时，避免连接卡死
	config.Timeout = 5 * time.Second

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	// 尝试获取 server version 以验证连接
	_, err = clientset.Discovery().ServerVersion()
	if err != nil {
		return fmt.Errorf("failed to connect to cluster: %w", err)
	}

	return nil
}

// CreateCluster 创建新集群
func (s *ClusterService) CreateCluster(cluster *models.Cluster) error {
	// 验证 KubeConfig
	if err := s.validateKubeConfig(string(cluster.Config)); err != nil {
		return fmt.Errorf("kubeconfig validation failed: %w", err)
	}

	// 检查是否存在同名集群
	var count int64
	if err := s.db.Model(&models.Cluster{}).Where("name = ?", cluster.Name).Count(&count).Error; err != nil {
		return fmt.Errorf("check cluster existence failed: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("cluster %s already exists", cluster.Name)
	}

	// 设置默认值
	cluster.Enable = true

	return s.db.Create(cluster).Error
}

// UpdateCluster 更新集群信息
func (s *ClusterService) UpdateCluster(cluster *models.Cluster) error {
	// 验证 KubeConfig
	if err := s.validateKubeConfig(string(cluster.Config)); err != nil {
		return fmt.Errorf("kubeconfig validation failed: %w", err)
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		var existingCluster models.Cluster
		if err := tx.Where("name = ?", cluster.Name).First(&existingCluster).Error; err != nil {
			return fmt.Errorf("cluster not found: %w", err)
		}

		// 只更新允许修改的字段
		existingCluster.Description = cluster.Description
		existingCluster.Config = cluster.Config
		existingCluster.PrometheusURL = cluster.PrometheusURL

		return tx.Save(&existingCluster).Error
	})
}

// GetClusterNodes 获取集群节点列表
func (s *ClusterService) GetClusterNodes(ctx context.Context, clusterName string) ([]string, error) {
	cluster, err := s.GetClusterByID(clusterName)
	if err != nil {
		return nil, err
	}

	if cluster.Config == "" {
		return nil, fmt.Errorf("cluster has no kubeconfig")
	}

	config, err := clientcmd.RESTConfigFromKubeConfig([]byte(cluster.Config))
	if err != nil {
		return nil, fmt.Errorf("invalid kubeconfig: %w", err)
	}
	// 设置超时
	config.Timeout = 10 * time.Second

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create k8s client: %w", err)
	}

	nodes, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}

	nodeNames := make([]string, 0, len(nodes.Items))
	for _, node := range nodes.Items {
		nodeNames = append(nodeNames, node.Name)
	}

	return nodeNames, nil
}

// GetClient 获取集群的 Kubernetes Client
func (s *ClusterService) GetClient(clusterName string) (kubernetes.Interface, error) {
	cluster, err := s.GetClusterByID(clusterName)
	if err != nil {
		return nil, err
	}

	if cluster.Config == "" {
		return nil, fmt.Errorf("cluster has no kubeconfig")
	}

	config, err := clientcmd.RESTConfigFromKubeConfig([]byte(cluster.Config))
	if err != nil {
		return nil, fmt.Errorf("invalid kubeconfig: %w", err)
	}
	// 设置超时
	config.Timeout = 10 * time.Second

	return kubernetes.NewForConfig(config)
}

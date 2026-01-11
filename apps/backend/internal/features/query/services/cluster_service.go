package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
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
	Name          string `json:"name"`
	ClusterID     string `json:"cluster_id"`
	Status        string `json:"status"`
	AlertCount    int64  `json:"alert_count"`
	CriticalCount int64  `json:"critical_count"`
}

type AlertTrendPoint struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

func (s *ClusterService) GetClustersSummary() (*ClusterSummary, error) {
	summary := &ClusterSummary{}

	if err := s.db.Model(&models.Cluster{}).Count(&summary.TotalClusters).Error; err != nil {
		return nil, fmt.Errorf("failed to count total clusters: %w", err)
	}

	if err := s.db.Model(&models.Cluster{}).
		Where("status = ?", models.ClusterStatusRunning).
		Count(&summary.ActiveClusters).Error; err != nil {
		return nil, fmt.Errorf("failed to count active clusters: %w", err)
	}

	if err := s.db.Model(&models.Alert{}).Count(&summary.TotalAlerts).Error; err != nil {
		return nil, fmt.Errorf("failed to count total alerts: %w", err)
	}

	if err := s.db.Model(&models.Alert{}).
		Where("status = ? AND severity = ?", models.AlertStatusFiring, models.AlertSeverityCritical).
		Count(&summary.CriticalAlerts).Error; err != nil {
		return nil, fmt.Errorf("failed to count critical alerts: %w", err)
	}

	clusterStats, err := s.getClusterStats()
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster stats: %w", err)
	}
	summary.ClusterStats = clusterStats

	summary.AlertTrends = []AlertTrendPoint{}

	return summary, nil
}

func (s *ClusterService) getClusterStats() ([]ClusterStats, error) {
	var stats []ClusterStats

	const query = `
		SELECT
			c.cluster_id,
			c.clustername as name,
			c.status,
			COALESCE(alert_counts.total_alerts, 0) as alert_count,
			COALESCE(alert_counts.critical_alerts, 0) as critical_count
		FROM k8s_clusters c
		LEFT JOIN (
			SELECT
				cluster_name,
				COUNT(*) as total_alerts,
				COUNT(CASE WHEN severity = 'critical' AND status = 'firing' THEN 1 END) as critical_alerts
			FROM spydon_alerts
			WHERE deleted_at IS NULL
			GROUP BY cluster_name
		) alert_counts ON c.clustername = alert_counts.cluster_name
		ORDER BY c.clustername`

	if err := s.db.Raw(query).Scan(&stats).Error; err != nil {
		return nil, err
	}

	return stats, nil
}

func (s *ClusterService) GetClusters(page, limit int, status, keyword string) ([]models.Cluster, int64, error) {
	var total int64

	baseQuery := s.applyClusterFilters(s.db.Model(&models.Cluster{}), status, keyword)

	if err := baseQuery.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count clusters: %w", err)
	}

	results, err := s.fetchClustersWithIPs(page, limit, status, keyword)
	if err != nil {
		return nil, 0, err
	}

	clusters := s.convertClusterResults(results)
	return clusters, total, nil
}

type clusterWithIPs struct {
	models.Cluster
	MasterIPsRaw    string `gorm:"column:master_ips_raw"`
	EtcdIPsRaw      string `gorm:"column:etcd_ips_raw"`
	EtcdEventIPsRaw string `gorm:"column:etcd_event_ips_raw"`
}

func (s *ClusterService) applyClusterFilters(query *gorm.DB, status, keyword string) *gorm.DB {
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if keyword != "" {
		pattern := "%" + keyword + "%"
		query = query.Where("clustername LIKE ? OR cluster_id LIKE ? OR idc LIKE ? OR zone LIKE ? OR purpose LIKE ?",
			pattern, pattern, pattern, pattern, pattern)
	}
	return query
}

func (s *ClusterService) fetchClustersWithIPs(page, limit int, status, keyword string) ([]clusterWithIPs, error) {
	var results []clusterWithIPs
	offset := (page - 1) * limit

	const selectClause = `k8s_clusters.*,
		GROUP_CONCAT(DISTINCT CASE WHEN LOWER(d.role) LIKE '%master%' THEN d.ip END ORDER BY d.ip SEPARATOR ',') AS master_ips_raw,
		GROUP_CONCAT(DISTINCT CASE WHEN LOWER(d.role) LIKE '%kube-etcd%' AND LOWER(d.role) NOT LIKE '%eventer%' THEN d.ip END ORDER BY d.ip SEPARATOR ',') AS etcd_ips_raw,
		GROUP_CONCAT(DISTINCT CASE WHEN LOWER(d.role) LIKE '%kube-etcd-eventer%' THEN d.ip END ORDER BY d.ip SEPARATOR ',') AS etcd_event_ips_raw`

	query := s.db.Model(&models.Cluster{}).
		Select(selectClause).
		Joins("LEFT JOIN device AS d ON k8s_clusters.clustername = d.cluster").
		Group("k8s_clusters.id")

	query = s.applyClusterFilters(query, status, keyword)

	if err := query.Order("k8s_clusters.clustername").Offset(offset).Limit(limit).Find(&results).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch clusters: %w", err)
	}

	return results, nil
}

func (s *ClusterService) convertClusterResults(results []clusterWithIPs) []models.Cluster {
	clusters := make([]models.Cluster, len(results))
	for i, r := range results {
		clusters[i] = r.Cluster
		clusters[i].MasterIPs = s.splitIPs(r.MasterIPsRaw)
		clusters[i].EtcdIPs = s.splitIPs(r.EtcdIPsRaw)
		clusters[i].EtcdEventIPs = s.splitIPs(r.EtcdEventIPsRaw)
		s.backfillKubeConfig(&clusters[i])
	}
	return clusters
}

func (s *ClusterService) splitIPs(ipsRaw string) []string {
	if ipsRaw == "" {
		return nil
	}
	return strings.Split(ipsRaw, ",")
}

func (s *ClusterService) backfillKubeConfig(cluster *models.Cluster) {
	if cluster.KubeConfig == "" && cluster.Config != "" {
		cluster.KubeConfig = string(cluster.Config)
	}
}

func (s *ClusterService) dbWithContext(ctx context.Context) *gorm.DB {
	if ctx == nil {
		return s.db.DB
	}
	return s.db.WithContext(ctx)
}

func (s *ClusterService) GetClusterByID(clusterName string) (*models.Cluster, error) {
	var cluster models.Cluster
	if err := s.db.Where("clustername = ?", clusterName).First(&cluster).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("cluster not found")
		}
		return nil, fmt.Errorf("failed to get cluster: %w", err)
	}
	s.backfillKubeConfig(&cluster)
	return &cluster, nil
}

func (s *ClusterService) DeleteCluster(clusterName string) error {
	result := s.db.Unscoped().Where("clustername = ?", clusterName).Delete(&models.Cluster{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete cluster: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("cluster not found")
	}

	if err := s.db.Unscoped().Where("cluster_name = ?", clusterName).Delete(&models.Alert{}).Error; err != nil {
		return fmt.Errorf("failed to delete cluster alerts: %w", err)
	}

	return nil
}

func (s *ClusterService) validateKubeConfig(kubeConfig string) error {
	if kubeConfig == "" {
		return nil
	}

	config, err := clientcmd.RESTConfigFromKubeConfig([]byte(kubeConfig))
	if err != nil {
		return fmt.Errorf("invalid kubeconfig format: %w", err)
	}

	config.Timeout = 5 * time.Second

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	_, err = clientset.Discovery().ServerVersion()
	if err != nil {
		return fmt.Errorf("failed to connect to cluster: %w", err)
	}

	return nil
}

func (s *ClusterService) CreateCluster(cluster *models.Cluster) error {
	if err := s.validateKubeConfig(string(cluster.Config)); err != nil {
		return fmt.Errorf("kubeconfig validation failed: %w", err)
	}

	var count int64
	if err := s.db.Model(&models.Cluster{}).Where("clustername = ?", cluster.Name).Count(&count).Error; err != nil {
		return fmt.Errorf("failed to check cluster existence: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("cluster %s already exists", cluster.Name)
	}

	cluster.Enable = true
	return s.db.Create(cluster).Error
}

func (s *ClusterService) UpdateCluster(cluster *models.Cluster) error {
	if err := s.validateKubeConfig(string(cluster.Config)); err != nil {
		return fmt.Errorf("kubeconfig validation failed: %w", err)
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		var existingCluster models.Cluster
		if err := tx.Where("clustername = ?", cluster.Name).First(&existingCluster).Error; err != nil {
			return fmt.Errorf("cluster not found: %w", err)
		}

		s.updateClusterFields(&existingCluster, cluster)
		return tx.Save(&existingCluster).Error
	})
}

func (s *ClusterService) updateClusterFields(existing, cluster *models.Cluster) {
	existing.Description = cluster.Description
	existing.PrometheusURL = cluster.PrometheusURL
	existing.Status = cluster.Status
	existing.ClusterVersion = cluster.ClusterVersion
	existing.Idc = cluster.Idc
	existing.Zone = cluster.Zone
	existing.FlowType = cluster.FlowType
	existing.Purpose = cluster.Purpose
	existing.Arch = cluster.Arch
	existing.Priority = cluster.Priority
	existing.ClusterGroup = cluster.ClusterGroup

	if cluster.Config != "" {
		existing.Config = cluster.Config
		if cluster.KubeConfig != "" {
			existing.KubeConfig = cluster.KubeConfig
		} else {
			existing.KubeConfig = string(cluster.Config)
		}
	}
}

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
	config.Timeout = 10 * time.Second

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create k8s client: %w", err)
	}

	nodes, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}

	nodeNames := make([]string, len(nodes.Items))
	for i, node := range nodes.Items {
		nodeNames[i] = node.Name
	}

	return nodeNames, nil
}

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
	config.Timeout = 10 * time.Second

	return kubernetes.NewForConfig(config)
}

func (s *ClusterService) SyncClusterConfig() (int64, error) {
	var clusters []models.Cluster
	if err := s.db.Find(&clusters).Error; err != nil {
		return 0, fmt.Errorf("failed to list clusters: %w", err)
	}

	var updatedCount int64
	for _, cluster := range clusters {
		if string(cluster.Config) == "" && cluster.KubeConfig != "" {
			cluster.Config = models.KiteSecretString(cluster.KubeConfig)

			if err := s.db.Model(&cluster).Update("config", cluster.Config).Error; err != nil {
				return updatedCount, fmt.Errorf("failed to update cluster %s: %w", cluster.Name, err)
			}
			updatedCount++
		}
	}
	return updatedCount, nil
}

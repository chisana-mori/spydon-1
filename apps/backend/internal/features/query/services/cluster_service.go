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
	Name          string     `json:"name"`
	ClusterID     string     `json:"cluster_id"`
	Status        string     `json:"status"`
	AlertCount    int64      `json:"alert_count"`
	CriticalCount int64      `json:"critical_count"`
	LastHeartbeat *time.Time `json:"last_heartbeat"`
}

// AlertTrendPoint 告警趋势点
type AlertTrendPoint struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
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
		FROM k8s_clusters c
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

// GetClusters 获取集群列表 - 使用 LEFT JOIN 优化查询效率
func (s *ClusterService) GetClusters(page, limit int, status, keyword string) ([]models.Cluster, int64, error) {
	var clusters []models.Cluster
	var total int64

	// 基础查询条件
	baseQuery := s.db.Model(&models.Cluster{})
	if status != "" {
		baseQuery = baseQuery.Where("status = ?", status)
	}
	if keyword != "" {
		likePattern := "%" + keyword + "%"
		baseQuery = baseQuery.Where("name LIKE ? OR cluster_id LIKE ? OR idc LIKE ? OR zone LIKE ? OR purpose LIKE ?", likePattern, likePattern, likePattern, likePattern, likePattern)
	}

	// 获取总数
	if err := baseQuery.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("获取集群总数失败: %w", err)
	}

	// 使用 LEFT JOIN 和 GROUP_CONCAT 进行聚合查询
	// 定义临时结构体接收聚合后的数据
	type ClusterWithIPs struct {
		models.Cluster
		MasterIPsRaw    string `gorm:"column:master_ips_raw"`
		EtcdIPsRaw      string `gorm:"column:etcd_ips_raw"`
		EtcdEventIPsRaw string `gorm:"column:etcd_event_ips_raw"`
	}

	var results []ClusterWithIPs
	offset := (page - 1) * limit

	// 构建 LEFT JOIN 查询
	// 使用子查询来聚合 device 的 IP，按角色分组
	joinQuery := s.db.Table("k8s_clusters AS c").
		Select(`c.*,
			GROUP_CONCAT(DISTINCT CASE WHEN LOWER(d.role) LIKE '%master%' THEN d.ip END ORDER BY d.ip SEPARATOR ',') AS master_ips_raw,
			GROUP_CONCAT(DISTINCT CASE WHEN LOWER(d.role) LIKE '%kube-etcd%' AND LOWER(d.role) NOT LIKE '%eventer%' THEN d.ip END ORDER BY d.ip SEPARATOR ',') AS etcd_ips_raw,
			GROUP_CONCAT(DISTINCT CASE WHEN LOWER(d.role) LIKE '%kube-etcd-eventer%' THEN d.ip END ORDER BY d.ip SEPARATOR ',') AS etcd_event_ips_raw`).
		Joins("LEFT JOIN device AS d ON c.name = d.cluster").
		Group("c.id")

	// 应用过滤条件
	if status != "" {
		joinQuery = joinQuery.Where("c.status = ?", status)
	}
	if keyword != "" {
		likePattern := "%" + keyword + "%"
		joinQuery = joinQuery.Where("c.name LIKE ? OR c.cluster_id LIKE ? OR c.idc LIKE ? OR c.zone LIKE ? OR c.purpose LIKE ?", likePattern, likePattern, likePattern, likePattern, likePattern)
	}

	// 分页
	if err := joinQuery.Order("c.name").Offset(offset).Limit(limit).Find(&results).Error; err != nil {
		return nil, 0, fmt.Errorf("获取集群列表失败: %w", err)
	}

	// 转换结果
	clusters = make([]models.Cluster, len(results))
	for i, r := range results {
		clusters[i] = r.Cluster
		// 解析逗号分隔的 IP 字符串
		if r.MasterIPsRaw != "" {
			clusters[i].MasterIPs = strings.Split(r.MasterIPsRaw, ",")
		}
		if r.EtcdIPsRaw != "" {
			clusters[i].EtcdIPs = strings.Split(r.EtcdIPsRaw, ",")
		}
		if r.EtcdEventIPsRaw != "" {
			clusters[i].EtcdEventIPs = strings.Split(r.EtcdEventIPsRaw, ",")
		}
		// 兼容：如果数据库里的 kube_config 为空，则用已解密的 config 值回填
		if clusters[i].KubeConfig == "" && clusters[i].Config != "" {
			clusters[i].KubeConfig = string(clusters[i].Config)
		}
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
	if cluster.KubeConfig == "" && cluster.Config != "" {
		cluster.KubeConfig = string(cluster.Config)
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
		// kubeconfig 兼容：只有在请求显式携带非空 kubeconfig 时才覆盖，避免前端未回填导致误清空
		if cluster.Config != "" {
			existingCluster.Config = cluster.Config
			if cluster.KubeConfig != "" {
				existingCluster.KubeConfig = cluster.KubeConfig
			} else {
				existingCluster.KubeConfig = string(cluster.Config)
			}
		}
		existingCluster.PrometheusURL = cluster.PrometheusURL
		existingCluster.Status = cluster.Status
		existingCluster.ClusterVersion = cluster.ClusterVersion
		existingCluster.Idc = cluster.Idc
		existingCluster.Zone = cluster.Zone
		existingCluster.FlowType = cluster.FlowType
		existingCluster.Purpose = cluster.Purpose
		existingCluster.Arch = cluster.Arch
		existingCluster.Priority = cluster.Priority
		existingCluster.ClusterGroup = cluster.ClusterGroup

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

// SyncClusterConfig 同步集群配置：扫描所有集群，将 config 为空但 kube_config 不为空的记录进行 config 加密回填
func (s *ClusterService) SyncClusterConfig() (int64, error) {
	var clusters []models.Cluster
	if err := s.db.Find(&clusters).Error; err != nil {
		return 0, fmt.Errorf("failed to list clusters: %w", err)
	}

	var updatedCount int64 = 0
	for _, cluster := range clusters {
		// 如果 Config 为空（解密后为空），且 KubeConfig（明文）不为空
		if string(cluster.Config) == "" && cluster.KubeConfig != "" {
			// 将明文 KubeConfig 赋值给 Config，GORM Value() hook 会自动加密
			cluster.Config = models.KiteSecretString(cluster.KubeConfig)

			// 只更新 config 字段
			if err := s.db.Model(&cluster).Update("config", cluster.Config).Error; err != nil {
				return updatedCount, fmt.Errorf("failed to update cluster %s: %w", cluster.Name, err)
			}
			updatedCount++
		}
	}
	return updatedCount, nil
}

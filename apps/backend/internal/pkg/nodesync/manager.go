package nodesync

import (
	"context"
	"fmt"
	"sync"
	"time"

	"robusta-web/backend/internal/db"
	"robusta-web/backend/internal/models"
	"robusta-web/backend/pkg/logger"

	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// isConnectableStatus 检查集群状态是否允许连接
// 只允许 Init 和 Running 状态的集群进行连接
func isConnectableStatus(status string) bool {
	return status == string(models.ClusterStatusInit) || status == string(models.ClusterStatusRunning)
}

// Manager 节点同步管理器，管理所有集群的节点同步
type Manager struct {
	mainDB      *db.Database
	controllers map[string]*Controller
	mu          sync.RWMutex
	stopCh      chan struct{}
	wg          sync.WaitGroup
}

// NewManager 创建节点同步管理器
func NewManager(mainDB *db.Database) *Manager {
	return &Manager{
		mainDB:      mainDB,
		controllers: make(map[string]*Controller),
		stopCh:      make(chan struct{}),
	}
}

// Start 启动节点同步管理器
func (m *Manager) Start(ctx context.Context) error {
	logger.S().Info("节点同步管理器启动中...")

	// 初始加载所有启用的集群
	if err := m.loadClusters(ctx); err != nil {
		logger.S().Errorw("加载集群失败", "error", err)
		return err
	}

	// 启动周期性刷新
	m.wg.Add(1)
	go m.refreshLoop(ctx)

	// 启动时进行一次全量同步
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		m.SyncAllClusters(ctx)
	}()

	logger.S().Infow("节点同步管理器已启动", "cluster_count", len(m.controllers))
	return nil
}

// Stop 停止节点同步管理器
func (m *Manager) Stop() {
	logger.S().Info("节点同步管理器停止中...")
	close(m.stopCh)

	m.mu.Lock()
	for id, ctrl := range m.controllers {
		ctrl.Stop()
		delete(m.controllers, id)
	}
	m.mu.Unlock()

	m.wg.Wait()
	logger.S().Info("节点同步管理器已停止")
}

// loadClusters 加载所有启用且状态为 Init 或 Running 的集群并启动控制器
func (m *Manager) loadClusters(ctx context.Context) error {
	var clusters []models.Cluster
	// 只加载启用且状态为 Init 或 Running 的集群
	if err := m.mainDB.Where("enable = ? AND status IN ?", true, []string{string(models.ClusterStatusInit), string(models.ClusterStatusRunning)}).Find(&clusters).Error; err != nil {
		return err
	}

	for _, cluster := range clusters {
		if cluster.Config == "" {
			logger.S().Warnw("集群无 KubeConfig，跳过", "cluster", cluster.Name)
			continue
		}

		if err := m.startController(ctx, &cluster); err != nil {
			logger.S().Errorw("启动集群控制器失败", "cluster", cluster.Name, "error", err)
			continue
		}
	}

	return nil
}

// startController 为单个集群启动控制器
func (m *Manager) startController(ctx context.Context, cluster *models.Cluster) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 如果已存在，先停止
	if existing, ok := m.controllers[cluster.Name]; ok {
		existing.Stop()
		delete(m.controllers, cluster.Name)
	}

	ctrl, err := NewController(cluster, m.mainDB)
	if err != nil {
		return err
	}

	ctrlCtx, cancel := context.WithCancel(ctx)
	ctrl.cancel = cancel

	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		if err := ctrl.Start(ctrlCtx); err != nil {
			logger.L().Error("集群控制器运行失败",
				zap.String("cluster", cluster.Name),
				zap.Error(err))
		}
	}()

	m.controllers[cluster.Name] = ctrl
	logger.S().Infow("集群控制器已启动", "cluster", cluster.Name)
	return nil
}

// refreshLoop 周期性刷新集群列表
func (m *Manager) refreshLoop(ctx context.Context) {
	defer m.wg.Done()
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopCh:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := m.refreshClusters(ctx); err != nil {
				logger.S().Errorw("刷新集群列表失败", "error", err)
			}
		}
	}
}

// refreshClusters 刷新集群列表，处理新增、删除和非可连接状态的集群
func (m *Manager) refreshClusters(ctx context.Context) error {
	var clusters []models.Cluster
	if err := m.mainDB.Where("enable = ?", true).Find(&clusters).Error; err != nil {
		return err
	}

	// 构建当前可连接集群名称集合（Init 或 Running 状态）
	connectableNames := make(map[string]bool)
	disconnectedNames := make(map[string]bool)

	for _, cluster := range clusters {
		// 非 Init 或 Running 状态的集群不可连接
		if !isConnectableStatus(cluster.Status) {
			disconnectedNames[cluster.Name] = true
			logger.S().Debugw("集群状态不可连接，跳过", "cluster", cluster.Name, "status", cluster.Status)
			continue
		}

		connectableNames[cluster.Name] = true

		m.mu.RLock()
		_, exists := m.controllers[cluster.Name]
		m.mu.RUnlock()

		// 新集群，启动控制器
		if !exists && cluster.Config != "" {
			if err := m.startController(ctx, &cluster); err != nil {
				logger.S().Errorw("启动新集群控制器失败", "cluster", cluster.Name, "error", err)
			}
		}
	}

	// 停止已删除、禁用或非可连接状态的集群控制器
	m.mu.Lock()
	for name, ctrl := range m.controllers {
		if !connectableNames[name] {
			ctrl.Stop()
			delete(m.controllers, name)
			if disconnectedNames[name] {
				logger.S().Infow("集群状态变更，断开连接", "cluster", name)
			} else {
				logger.S().Infow("集群控制器已停止", "cluster", name)
			}
		}
	}
	m.mu.Unlock()

	return nil
}

// GetNode 从缓存获取指定集群的节点
func (m *Manager) GetNode(clusterName string, nodeName string) (*corev1.Node, error) {
	m.mu.RLock()
	ctrl, ok := m.controllers[clusterName]
	m.mu.RUnlock()

	if !ok {
		return nil, nil // 集群未被管理或不存在
	}

	// 使用短暂的 context，因为是读缓存
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return ctrl.GetNode(ctx, nodeName)
}

// GetClient 获取指定集群的 K8s Client（用于实时操作）
func (m *Manager) GetClient(clusterName string) (interface{}, error) {
	m.mu.RLock()
	ctrl, ok := m.controllers[clusterName]
	m.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("集群 %s 未被管理或不存在", clusterName)
	}

	return ctrl.GetClient(), nil
}

// ListNodes 列出指定集群的所有节点
func (m *Manager) ListNodes(clusterName string) ([]corev1.Node, error) {
	m.mu.RLock()
	ctrl, ok := m.controllers[clusterName]
	m.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("集群 %s 未被管理或不存在", clusterName)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var nodeList corev1.NodeList
	if err := ctrl.GetClient().List(ctx, &nodeList); err != nil {
		return nil, err
	}

	return nodeList.Items, nil
}

// ListPodsOnNodes 按需查询指定节点上的 Pods（不使用缓存）
func (m *Manager) ListPodsOnNodes(clusterName string, nodeNames []string) ([]corev1.Pod, error) {
	m.mu.RLock()
	ctrl, ok := m.controllers[clusterName]
	m.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("集群 %s 未被管理或不存在", clusterName)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var allPods []corev1.Pod

	// 如果没有指定节点，获取所有 Pods
	if len(nodeNames) == 0 {
		var podList corev1.PodList
		if err := ctrl.GetClient().List(ctx, &podList); err != nil {
			return nil, fmt.Errorf("获取 Pods 失败: %w", err)
		}
		return podList.Items, nil
	}

	// 按节点过滤 Pods
	nodeSet := make(map[string]bool)
	for _, n := range nodeNames {
		nodeSet[n] = true
	}

	var podList corev1.PodList
	if err := ctrl.GetClient().List(ctx, &podList); err != nil {
		return nil, fmt.Errorf("获取 Pods 失败: %w", err)
	}

	for _, pod := range podList.Items {
		if nodeSet[pod.Spec.NodeName] {
			allPods = append(allPods, pod)
		}
	}

	return allPods, nil
}

// ListDeployments 按需查询指定集群的所有 Deployments（集群级别，不使用缓存）
func (m *Manager) ListDeployments(clusterName string, namespace string) ([]appsv1.Deployment, error) {
	m.mu.RLock()
	ctrl, ok := m.controllers[clusterName]
	m.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("集群 %s 未被管理或不存在", clusterName)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var deployList appsv1.DeploymentList
	opts := []client.ListOption{}
	if namespace != "" {
		opts = append(opts, client.InNamespace(namespace))
	}

	if err := ctrl.GetClient().List(ctx, &deployList, opts...); err != nil {
		return nil, fmt.Errorf("获取 Deployments 失败: %w", err)
	}

	return deployList.Items, nil
}

// ListDaemonSets 按需查询指定集群的所有 DaemonSets（集群级别，不使用缓存）
func (m *Manager) ListDaemonSets(clusterName string, namespace string) ([]appsv1.DaemonSet, error) {
	m.mu.RLock()
	ctrl, ok := m.controllers[clusterName]
	m.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("集群 %s 未被管理或不存在", clusterName)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var dsList appsv1.DaemonSetList
	opts := []client.ListOption{}
	if namespace != "" {
		opts = append(opts, client.InNamespace(namespace))
	}

	if err := ctrl.GetClient().List(ctx, &dsList, opts...); err != nil {
		return nil, fmt.Errorf("获取 DaemonSets 失败: %w", err)
	}

	return dsList.Items, nil
}

// GetManagedClusters 获取当前管理的所有活跃集群名称列表
func (m *Manager) GetManagedClusters() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	clusters := make([]string, 0, len(m.controllers))
	for name := range m.controllers {
		clusters = append(clusters, name)
	}
	return clusters
}

// GetRESTConfig 获取指定集群的 REST 配置（用于创建 dynamic client）
func (m *Manager) GetRESTConfig(clusterName string) (interface{}, error) {
	m.mu.RLock()
	ctrl, ok := m.controllers[clusterName]
	m.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("集群 %s 未被管理或不存在", clusterName)
	}

	return ctrl.GetRESTConfig(), nil
}

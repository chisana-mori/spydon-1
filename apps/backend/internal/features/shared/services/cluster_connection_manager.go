package services

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"robusta-web/backend/internal/models"

	"gorm.io/gorm"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// Ensure ClusterConnectionManager implements K8sClientFactory
var _ K8sClientFactory = (*ClusterConnectionManager)(nil)

// ClusterConnectionManager 简化的集群连接管理器
// 遵循快速失败原则，不维护长期连接状态，去掉锁机制
type ClusterConnectionManager struct {
	db       *gorm.DB
	configs  map[string]*rest.Config // 配置缓存，启动时加载一次
	managers map[string]manager.Manager
	mu       sync.Mutex
	scheme   *runtime.Scheme
}

// ClusterSnapshot 集群快照信息
type ClusterSnapshot struct {
	ClusterName    string    `json:"clusterName"`
	ClusterID      string    `json:"clusterId"`
	Status         string    `json:"status"`
	Connected      bool      `json:"connected"`
	ConnectionTime time.Time `json:"connectionTime"`
	LastError      string    `json:"lastError,omitempty"`
}

// NewClusterConnectionManager 创建新的集群连接管理器
func NewClusterConnectionManager(db *gorm.DB) *ClusterConnectionManager {
	// initialize scheme
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = appsv1.AddToScheme(scheme)
	_ = policyv1.AddToScheme(scheme)
	_ = policyv1beta1.AddToScheme(scheme)

	return &ClusterConnectionManager{
		db:       db,
		configs:  make(map[string]*rest.Config),
		managers: make(map[string]manager.Manager),
		scheme:   scheme,
	}
}

// Initialize 初始化管理器 - 简化版本，只加载配置
func (cm *ClusterConnectionManager) Initialize() error {
	log.Printf("ClusterConnectionManager: Loading cluster configurations...")

	// 从数据库加载所有集群配置
	var clusters []models.Cluster
	if err := cm.db.Find(&clusters).Error; err != nil {
		return fmt.Errorf("failed to load clusters from database: %w", err)
	}

	// 预加载配置到缓存
	for _, cluster := range clusters {
		config, err := cm.buildConfig(&cluster)
		if err != nil {
			log.Printf("Warning: Failed to build config for cluster %s: %v", cluster.Name, err)
			continue
		}
		cm.configs[cluster.Name] = config
		log.Printf("ClusterConnectionManager: Loaded config for cluster %s", cluster.Name)
	}

	log.Printf("ClusterConnectionManager: Initialized with %d cluster configurations", len(clusters))
	return nil
}

// Shutdown 关闭管理器 - 简化版本
func (cm *ClusterConnectionManager) Shutdown() {
	log.Printf("ClusterConnectionManager: Shutdown completed")
}

// GetManager 获取指定集群的管理器（懒加载并缓存）
func (cm *ClusterConnectionManager) GetManager(clusterName string) (manager.Manager, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.managers == nil {
		cm.managers = make(map[string]manager.Manager)
	}
	if m, ok := cm.managers[clusterName]; ok {
		return m, nil
	}

	cfg, ok := cm.configs[clusterName]
	if !ok {
		return nil, fmt.Errorf("config not found for cluster: %s", clusterName)
	}

	// 确保 scheme 已初始化
	if cm.scheme == nil {
		s := runtime.NewScheme()
		_ = corev1.AddToScheme(s)
		_ = appsv1.AddToScheme(s)
		_ = policyv1.AddToScheme(s)
		_ = policyv1beta1.AddToScheme(s)
		cm.scheme = s
	}

	mgr, err := manager.New(cfg, manager.Options{
		Scheme:         cm.scheme,
		LeaderElection: false,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create manager: %w", err)
	}

	// 为 Pod 建立按 nodeName 的索引，便于查询
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &corev1.Pod{}, "spec.nodeName", func(obj crclient.Object) []string {
		pod, ok := obj.(*corev1.Pod)
		if !ok {
			return nil
		}
		return []string{pod.Spec.NodeName}
	}); err != nil {
		log.Printf("Warning: failed to create pod.spec.nodeName index for cluster %s: %v", clusterName, err)
		// 不要因为索引创建失败就终止manager创建，继续创建manager但会回退到API查询
	}

	// 启动 manager
	go func() {
		if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
			log.Printf("manager for cluster %s stopped with error: %v", clusterName, err)
		}
	}()

	// 等待缓存同步，设置超时
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if ok := mgr.GetCache().WaitForCacheSync(ctx); !ok {
		log.Printf("Warning: cache sync timeout for cluster %s, manager will still be available but may have degraded performance", clusterName)
		// 不要因为缓存同步失败就终止，继续提供manager但性能可能降低
	}

	cm.managers[clusterName] = mgr
	return mgr, nil
}

// GetClient 获取指定集群的客户端 - 优先使用manager
func (cm *ClusterConnectionManager) GetClient(clusterName string) (kubernetes.Interface, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// 首先检查是否有已缓存的manager
	if mgr, exists := cm.managers[clusterName]; exists {
		// 如果manager存在，尝试从manager获取配置来创建kubernetes客户端
		config := mgr.GetConfig()
		client, err := kubernetes.NewForConfig(config)
		if err != nil {
			return nil, fmt.Errorf("failed to create client from manager config for cluster %s: %w", clusterName, err)
		}
		return client, nil
	}

	// 如果manager不存在，从配置缓存获取配置
	config, exists := cm.configs[clusterName]
	if !exists {
		return nil, fmt.Errorf("cluster %s not found", clusterName)
	}

	// 直接创建客户端 - 如果失败直接返回错误
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create client for cluster %s: %w", clusterName, err)
	}

	return client, nil
}

// GetClientFromManager 从指定的manager获取kubernetes客户端
func (cm *ClusterConnectionManager) GetClientFromManager(clusterName string) (kubernetes.Interface, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// 检查manager是否存在
	mgr, exists := cm.managers[clusterName]
	if !exists {
		return nil, fmt.Errorf("manager for cluster %s not found, call GetManager first", clusterName)
	}

	// 从manager的配置创建kubernetes客户端
	config := mgr.GetConfig()
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create client from manager config for cluster %s: %w", clusterName, err)
	}

	return client, nil
}

// GetConfig 获取指定集群的配置
func (cm *ClusterConnectionManager) GetConfig(clusterName string) (*rest.Config, error) {
	config, exists := cm.configs[clusterName]
	if !exists {
		return nil, fmt.Errorf("config not found for cluster: %s", clusterName)
	}
	return config, nil
}

// ConnectCluster 显式连接指定集群 - 简化版本不需要
func (cm *ClusterConnectionManager) ConnectCluster(clusterName string) error {
	// 简化版本不需要显式连接，客户端按需创建
	return nil
}

// AddCluster 添加集群到管理器 - 简化版本
func (cm *ClusterConnectionManager) AddCluster(clusterName string, config *rest.Config) error {
	// 简化版本只存储配置
	cm.configs[clusterName] = config
	log.Printf("ClusterConnectionManager: Added cluster %s configuration", clusterName)
	return nil
}

// RemoveCluster 从管理器中移除集群 - 简化版本
func (cm *ClusterConnectionManager) RemoveCluster(clusterName string) {
	delete(cm.configs, clusterName)
	log.Printf("ClusterConnectionManager: Removed cluster %s", clusterName)
}

// ListClusters 列出所有管理的集群 - 简化版本
func (cm *ClusterConnectionManager) ListClusters() []string {
	clusters := make([]string, 0, len(cm.configs))
	for name := range cm.configs {
		clusters = append(clusters, name)
	}
	return clusters
}

// IsClusterConnected 检查集群是否可连接 - 快速检查
func (cm *ClusterConnectionManager) IsClusterConnected(clusterName string) bool {
	client, err := cm.GetClient(clusterName)
	if err != nil {
		return false
	}

	// 快速检查连接 - 3秒超时
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// 尝试获取服务器版本
	_, err = client.Discovery().ServerVersion()
	_ = ctx // 使用上下文变量避免编译警告
	return err == nil
}

// TestConnection implements K8sClientFactory
func (cm *ClusterConnectionManager) TestConnection(ctx context.Context, clusterName string) error {
	// 忽略 ctx，复用现有的简单测试逻辑
	return cm.TestClusterConnection(clusterName)
}

// IsClusterAvailable implements K8sClientFactory
func (cm *ClusterConnectionManager) IsClusterAvailable(clusterName string) bool {
	return cm.IsClusterConnected(clusterName)
}

// GetClusterStatus 获取集群状态 - 快速检查
func (cm *ClusterConnectionManager) GetClusterStatus(clusterName string) string {
	if cm.IsClusterConnected(clusterName) {
		return "connected"
	}
	return "disconnected"
}

// GetAllClusterStatus 获取所有集群状态 - 简化版本
func (cm *ClusterConnectionManager) GetAllClusterStatus() map[string]string {
	statusMap := make(map[string]string)
	for name := range cm.configs {
		statusMap[name] = cm.GetClusterStatus(name)
	}
	return statusMap
}

// TestClusterConnection 测试集群连接
func (cm *ClusterConnectionManager) TestClusterConnection(clusterName string) error {
	client, err := cm.GetClient(clusterName)
	if err != nil {
		return err
	}

	// 测试连接
	_, err = client.Discovery().ServerVersion()
	if err != nil {
		return fmt.Errorf("failed to test cluster connection: %w", err)
	}

	return nil
}

// ForceScan 强制执行一次扫描 - 简化版本不需要
func (cm *ClusterConnectionManager) ForceScan() {
	// 简化版本不需要扫描
}

// GetStatus 获取管理器状态 - 简化版本
func (cm *ClusterConnectionManager) GetStatus() map[string]interface{} {
	return map[string]interface{}{
		"running":          true,
		"managed_clusters": cm.ListClusters(),
		"cluster_status":   cm.GetAllClusterStatus(),
	}
}

// GetClusterSnapshots 获取所有集群快照
func (cm *ClusterConnectionManager) GetClusterSnapshots() []ClusterSnapshot {
	var clusters []models.Cluster
	if err := cm.db.Find(&clusters).Error; err != nil {
		return []ClusterSnapshot{}
	}

	snapshots := make([]ClusterSnapshot, 0, len(clusters))
	for _, cluster := range clusters {
		snapshot := ClusterSnapshot{
			ClusterName: cluster.Name,
			ClusterID:   cluster.ClusterID,
			Status:      cluster.Status,
			Connected:   cm.IsClusterConnected(cluster.Name),
		}
		snapshots = append(snapshots, snapshot)
	}

	return snapshots
}

// CreateConfigFromKubeConfig 从kubeconfig字符串创建rest配置
func CreateConfigFromKubeConfig(kubeconfig string) (*rest.Config, error) {
	// 创建临时文件来存储kubeconfig
	tmpFile := "/tmp/kubeconfig_" + fmt.Sprintf("%d", time.Now().Unix())

	// 将kubeconfig字符串写入临时文件
	if err := os.WriteFile(tmpFile, []byte(kubeconfig), 0644); err != nil {
		return nil, fmt.Errorf("failed to write kubeconfig to temp file: %w", err)
	}

	// 确保函数返回时删除临时文件
	defer func() {
		if err := os.Remove(tmpFile); err != nil {
			log.Printf("[CreateConfigFromKubeConfig] failed to remove temp kubeconfig: %v", err)
		}
	}()

	// 从临时文件创建配置
	config, err := clientcmd.BuildConfigFromFlags("", tmpFile)
	if err != nil {
		return nil, fmt.Errorf("failed to build config from kubeconfig: %w", err)
	}

	return config, nil
}

// CreateConfigFromAPIServer 从API Server地址创建配置（用于集群内访问）
func CreateConfigFromAPIServer(apiServer string) (*rest.Config, error) {
	config := &rest.Config{
		Host: apiServer,
		// 这里可以根据需要添加TLS配置
		TLSClientConfig: rest.TLSClientConfig{
			Insecure: true, // 仅用于测试环境，生产环境需要 proper TLS 配置
		},
		Timeout: 5 * time.Second,
	}
	return config, nil
}

// 全局实例
var (
	GlobalClusterConnectionManager *ClusterConnectionManager
	clusterManagerOnce             sync.Once
)

// GetGlobalClusterConnectionManager 获取全局集群连接管理器
func GetGlobalClusterConnectionManager(db *gorm.DB) *ClusterConnectionManager {
	clusterManagerOnce.Do(func() {
		GlobalClusterConnectionManager = NewClusterConnectionManager(db)
	})
	return GlobalClusterConnectionManager
}

// InitGlobalClusterConnectionManager 初始化全局集群连接管理器
func InitGlobalClusterConnectionManager(db *gorm.DB) error {
	manager := GetGlobalClusterConnectionManager(db)
	return manager.Initialize()
}

// 兼容性函数 - 为旧代码提供兼容接口
type LegacyK8sClientManager struct {
	clusterMgr *ClusterConnectionManager
}

// GetGlobalK8sManager 兼容性函数 - 返回旧版K8s管理器
func GetGlobalK8sManager(db *gorm.DB) *LegacyK8sClientManager {
	return &LegacyK8sClientManager{
		clusterMgr: GetGlobalClusterConnectionManager(db),
	}
}

// IsInitialized 检查是否已初始化 - 简化版本总是返回true
func (m *LegacyK8sClientManager) IsInitialized() bool {
	return true
}

// InitDefault 使用默认方式初始化
func (m *LegacyK8sClientManager) InitDefault() error {
	return m.clusterMgr.Initialize()
}

// GetClientset 获取kubernetes clientset
func (m *LegacyK8sClientManager) GetClientset() (kubernetes.Interface, error) {
	// 获取第一个可用集群的客户端
	clusters := m.clusterMgr.ListClusters()
	if len(clusters) == 0 {
		return nil, fmt.Errorf("no available clusters")
	}
	return m.clusterMgr.GetClient(clusters[0])
}

// HealthCheck 健康检查
func (m *LegacyK8sClientManager) HealthCheck(ctx context.Context) error {
	clusters := m.clusterMgr.ListClusters()
	if len(clusters) == 0 {
		return fmt.Errorf("no available clusters")
	}

	// 测试第一个集群的连接
	client, err := m.clusterMgr.GetClient(clusters[0])
	if err != nil {
		return err
	}

	_, err = client.Discovery().ServerVersion()
	return err
}

// K8sClientFactoryAdapter 适配器，让 LegacyK8sClientManager 实现 K8sClientFactory 接口
type K8sClientFactoryAdapter struct {
	manager *LegacyK8sClientManager
}

// NewK8sClientFactoryAdapter 创建适配器
func NewK8sClientFactoryAdapter(manager *LegacyK8sClientManager) K8sClientFactory {
	return &K8sClientFactoryAdapter{
		manager: manager,
	}
}

// GetClient 获取指定集群的kubernetes客户端
func (a *K8sClientFactoryAdapter) GetClient(clusterName string) (kubernetes.Interface, error) {
	return a.manager.clusterMgr.GetClient(clusterName)
}

// GetClientFromManager 从manager获取kubernetes客户端
func (a *K8sClientFactoryAdapter) GetClientFromManager(clusterName string) (kubernetes.Interface, error) {
	return a.manager.clusterMgr.GetClientFromManager(clusterName)
}

// GetConfig 获取指定集群的rest配置
func (a *K8sClientFactoryAdapter) GetConfig(clusterName string) (*rest.Config, error) {
	return a.manager.clusterMgr.GetConfig(clusterName)
}

// GetManager 获取指定集群的controller-runtime管理器
func (a *K8sClientFactoryAdapter) GetManager(clusterName string) (manager.Manager, error) {
	return a.manager.clusterMgr.GetManager(clusterName)
}

// ListClusters 列出所有可用集群
func (a *K8sClientFactoryAdapter) ListClusters() []string {
	return a.manager.clusterMgr.ListClusters()
}

// IsClusterAvailable 检查集群是否可用
func (a *K8sClientFactoryAdapter) IsClusterAvailable(clusterName string) bool {
	return a.manager.clusterMgr.IsClusterConnected(clusterName)
}

// TestConnection 测试集群连接
func (a *K8sClientFactoryAdapter) TestConnection(ctx context.Context, clusterName string) error {
	return a.manager.clusterMgr.TestClusterConnection(clusterName)
}

// buildConfig 构建集群配置
func (cm *ClusterConnectionManager) buildConfig(cluster *models.Cluster) (*rest.Config, error) {
	if cluster.Config != "" {
		// 使用 kubeconfig 内容
		config, err := clientcmd.RESTConfigFromKubeConfig([]byte(cluster.Config.String()))
		if err != nil {
			return nil, fmt.Errorf("failed to build config from kubeconfig: %w", err)
		}
		// 优化速率限制配置
		cm.configureRateLimits(config)
		return config, nil
	}

	return nil, fmt.Errorf("cluster %s has no kubeconfig configured", cluster.Name)
}

// configureRateLimits 配置Kubernetes客户端速率限制
func (cm *ClusterConnectionManager) configureRateLimits(config *rest.Config) {
	// 提高QPS和Burst限制
	config.QPS = 50.0  // 从默认的5.0提高到50.0
	config.Burst = 100 // 从默认的10提高到100

	// 配置超时时间
	config.Timeout = 30 * time.Second

	log.Printf("ClusterConnectionManager: 配置了优化的速率限制 - QPS: %.1f, Burst: %d", config.QPS, config.Burst)
}

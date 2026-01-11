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

var _ K8sClientFactory = (*ClusterConnectionManager)(nil)

type ClusterConnectionManager struct {
	db       *gorm.DB
	configs  map[string]*rest.Config
	managers map[string]manager.Manager
	mu       sync.Mutex
	scheme   *runtime.Scheme
}

type ClusterSnapshot struct {
	ClusterName    string    `json:"clusterName"`
	ClusterID      string    `json:"clusterId"`
	Status         string    `json:"status"`
	Connected      bool      `json:"connected"`
	ConnectionTime time.Time `json:"connectionTime"`
	LastError      string    `json:"lastError,omitempty"`
}

func NewClusterConnectionManager(db *gorm.DB) *ClusterConnectionManager {
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

func (cm *ClusterConnectionManager) Initialize() error {
	log.Printf("ClusterConnectionManager: Loading cluster configurations...")

	var clusters []models.Cluster
	if err := cm.db.Find(&clusters).Error; err != nil {
		return fmt.Errorf("failed to load clusters from database: %w", err)
	}

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

func (cm *ClusterConnectionManager) Shutdown() {
	log.Printf("ClusterConnectionManager: Shutdown completed")
}

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

	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &corev1.Pod{}, "spec.nodeName", func(obj crclient.Object) []string {
		pod, ok := obj.(*corev1.Pod)
		if !ok {
			return nil
		}
		return []string{pod.Spec.NodeName}
	}); err != nil {
		log.Printf("Warning: failed to create pod.spec.nodeName index for cluster %s: %v", clusterName, err)
	}

	go func() {
		if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
			log.Printf("manager for cluster %s stopped with error: %v", clusterName, err)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if ok := mgr.GetCache().WaitForCacheSync(ctx); !ok {
		log.Printf("Warning: cache sync timeout for cluster %s, manager will still be available but may have degraded performance", clusterName)
	}

	cm.managers[clusterName] = mgr
	return mgr, nil
}

func (cm *ClusterConnectionManager) GetClient(clusterName string) (kubernetes.Interface, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if mgr, exists := cm.managers[clusterName]; exists {
		config := mgr.GetConfig()
		client, err := kubernetes.NewForConfig(config)
		if err != nil {
			return nil, fmt.Errorf("failed to create client from manager config for cluster %s: %w", clusterName, err)
		}
		return client, nil
	}

	config, exists := cm.configs[clusterName]
	if !exists {
		return nil, fmt.Errorf("cluster %s not found", clusterName)
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create client for cluster %s: %w", clusterName, err)
	}

	return client, nil
}

func (cm *ClusterConnectionManager) GetClientFromManager(clusterName string) (kubernetes.Interface, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	mgr, exists := cm.managers[clusterName]
	if !exists {
		return nil, fmt.Errorf("manager for cluster %s not found, call GetManager first", clusterName)
	}

	config := mgr.GetConfig()
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create client from manager config for cluster %s: %w", clusterName, err)
	}

	return client, nil
}

func (cm *ClusterConnectionManager) GetConfig(clusterName string) (*rest.Config, error) {
	config, exists := cm.configs[clusterName]
	if !exists {
		return nil, fmt.Errorf("config not found for cluster: %s", clusterName)
	}
	return config, nil
}

func (cm *ClusterConnectionManager) ConnectCluster(clusterName string) error {
	return nil
}

func (cm *ClusterConnectionManager) AddCluster(clusterName string, config *rest.Config) error {
	cm.configs[clusterName] = config
	log.Printf("ClusterConnectionManager: Added cluster %s configuration", clusterName)
	return nil
}

func (cm *ClusterConnectionManager) RemoveCluster(clusterName string) {
	delete(cm.configs, clusterName)
	log.Printf("ClusterConnectionManager: Removed cluster %s", clusterName)
}

func (cm *ClusterConnectionManager) ListClusters() []string {
	clusters := make([]string, 0, len(cm.configs))
	for name := range cm.configs {
		clusters = append(clusters, name)
	}
	return clusters
}

func (cm *ClusterConnectionManager) IsClusterConnected(clusterName string) bool {
	client, err := cm.GetClient(clusterName)
	if err != nil {
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err = client.Discovery().ServerVersion()
	_ = ctx
	return err == nil
}

func (cm *ClusterConnectionManager) TestConnection(ctx context.Context, clusterName string) error {
	return cm.TestClusterConnection(clusterName)
}

func (cm *ClusterConnectionManager) IsClusterAvailable(clusterName string) bool {
	return cm.IsClusterConnected(clusterName)
}

func (cm *ClusterConnectionManager) GetClusterStatus(clusterName string) string {
	if cm.IsClusterConnected(clusterName) {
		return "connected"
	}
	return "disconnected"
}

func (cm *ClusterConnectionManager) GetAllClusterStatus() map[string]string {
	statusMap := make(map[string]string)
	for name := range cm.configs {
		statusMap[name] = cm.GetClusterStatus(name)
	}
	return statusMap
}

func (cm *ClusterConnectionManager) TestClusterConnection(clusterName string) error {
	client, err := cm.GetClient(clusterName)
	if err != nil {
		return err
	}

	_, err = client.Discovery().ServerVersion()
	if err != nil {
		return fmt.Errorf("failed to test cluster connection: %w", err)
	}

	return nil
}

func (cm *ClusterConnectionManager) ForceScan() {}

func (cm *ClusterConnectionManager) GetStatus() map[string]interface{} {
	return map[string]interface{}{
		"running":          true,
		"managed_clusters": cm.ListClusters(),
		"cluster_status":   cm.GetAllClusterStatus(),
	}
}

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

func CreateConfigFromKubeConfig(kubeconfig string) (*rest.Config, error) {
	tmpFile := "/tmp/kubeconfig_" + fmt.Sprintf("%d", time.Now().Unix())

	if err := os.WriteFile(tmpFile, []byte(kubeconfig), 0644); err != nil {
		return nil, fmt.Errorf("failed to write kubeconfig to temp file: %w", err)
	}

	defer func() {
		if err := os.Remove(tmpFile); err != nil {
			log.Printf("[CreateConfigFromKubeConfig] failed to remove temp kubeconfig: %v", err)
		}
	}()

	config, err := clientcmd.BuildConfigFromFlags("", tmpFile)
	if err != nil {
		return nil, fmt.Errorf("failed to build config from kubeconfig: %w", err)
	}

	return config, nil
}

func CreateConfigFromAPIServer(apiServer string) (*rest.Config, error) {
	config := &rest.Config{
		Host: apiServer,
		TLSClientConfig: rest.TLSClientConfig{
			Insecure: true,
		},
		Timeout: 5 * time.Second,
	}
	return config, nil
}

var (
	GlobalClusterConnectionManager *ClusterConnectionManager
	clusterManagerOnce             sync.Once
)

func GetGlobalClusterConnectionManager(db *gorm.DB) *ClusterConnectionManager {
	clusterManagerOnce.Do(func() {
		GlobalClusterConnectionManager = NewClusterConnectionManager(db)
	})
	return GlobalClusterConnectionManager
}

func InitGlobalClusterConnectionManager(db *gorm.DB) error {
	manager := GetGlobalClusterConnectionManager(db)
	return manager.Initialize()
}

type LegacyK8sClientManager struct {
	clusterMgr *ClusterConnectionManager
}

func GetGlobalK8sManager(db *gorm.DB) *LegacyK8sClientManager {
	return &LegacyK8sClientManager{
		clusterMgr: GetGlobalClusterConnectionManager(db),
	}
}

func (m *LegacyK8sClientManager) IsInitialized() bool {
	return true
}

func (m *LegacyK8sClientManager) InitDefault() error {
	return m.clusterMgr.Initialize()
}

func (m *LegacyK8sClientManager) GetClientset() (kubernetes.Interface, error) {
	clusters := m.clusterMgr.ListClusters()
	if len(clusters) == 0 {
		return nil, fmt.Errorf("no available clusters")
	}
	return m.clusterMgr.GetClient(clusters[0])
}

func (m *LegacyK8sClientManager) HealthCheck(ctx context.Context) error {
	clusters := m.clusterMgr.ListClusters()
	if len(clusters) == 0 {
		return fmt.Errorf("no available clusters")
	}

	client, err := m.clusterMgr.GetClient(clusters[0])
	if err != nil {
		return err
	}

	_, err = client.Discovery().ServerVersion()
	return err
}

type K8sClientFactoryAdapter struct {
	manager *LegacyK8sClientManager
}

func NewK8sClientFactoryAdapter(manager *LegacyK8sClientManager) K8sClientFactory {
	return &K8sClientFactoryAdapter{
		manager: manager,
	}
}

func (a *K8sClientFactoryAdapter) GetClient(clusterName string) (kubernetes.Interface, error) {
	return a.manager.clusterMgr.GetClient(clusterName)
}

func (a *K8sClientFactoryAdapter) GetClientFromManager(clusterName string) (kubernetes.Interface, error) {
	return a.manager.clusterMgr.GetClientFromManager(clusterName)
}

func (a *K8sClientFactoryAdapter) GetConfig(clusterName string) (*rest.Config, error) {
	return a.manager.clusterMgr.GetConfig(clusterName)
}

func (a *K8sClientFactoryAdapter) GetManager(clusterName string) (manager.Manager, error) {
	return a.manager.clusterMgr.GetManager(clusterName)
}

func (a *K8sClientFactoryAdapter) ListClusters() []string {
	return a.manager.clusterMgr.ListClusters()
}

func (a *K8sClientFactoryAdapter) IsClusterAvailable(clusterName string) bool {
	return a.manager.clusterMgr.IsClusterConnected(clusterName)
}

func (a *K8sClientFactoryAdapter) TestConnection(ctx context.Context, clusterName string) error {
	return a.manager.clusterMgr.TestClusterConnection(clusterName)
}

func (cm *ClusterConnectionManager) buildConfig(cluster *models.Cluster) (*rest.Config, error) {
	if cluster.Config != "" {
		config, err := clientcmd.RESTConfigFromKubeConfig([]byte(cluster.Config.String()))
		if err != nil {
			return nil, fmt.Errorf("failed to build config from kubeconfig: %w", err)
		}
		cm.configureRateLimits(config)
		return config, nil
	}

	return nil, fmt.Errorf("cluster %s has no kubeconfig configured", cluster.Name)
}

func (cm *ClusterConnectionManager) configureRateLimits(config *rest.Config) {
	config.QPS = 50.0
	config.Burst = 100
	config.Timeout = 30 * time.Second

	log.Printf("ClusterConnectionManager: 配置了优化的速率限制 - QPS: %.1f, Burst: %d", config.QPS, config.Burst)
}

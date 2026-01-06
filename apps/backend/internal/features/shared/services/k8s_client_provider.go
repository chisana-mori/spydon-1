package services

import (
	"context"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// K8sClientProvider 按集群名提供 Kubernetes 客户端
type K8sClientProvider interface {
	GetClient(clusterName string) (kubernetes.Interface, error)
}

// K8sClientFactory K8s客户端工厂接口
type K8sClientFactory interface {
	// GetClient 获取指定集群的kubernetes客户端
	GetClient(clusterName string) (kubernetes.Interface, error)

	// GetClientFromManager 从manager获取kubernetes客户端
	GetClientFromManager(clusterName string) (kubernetes.Interface, error)

	// GetConfig 获取指定集群的rest配置
	GetConfig(clusterName string) (*rest.Config, error)

	// GetManager 获取指定集群的controller-runtime管理器
	GetManager(clusterName string) (manager.Manager, error)

	// ListClusters 列出所有可用集群
	ListClusters() []string

	// IsClusterAvailable 检查集群是否可用
	IsClusterAvailable(clusterName string) bool

	// TestConnection 测试集群连接
	TestConnection(ctx context.Context, clusterName string) error
}

// ClusterConnectionManagerFactory 集群连接管理器工厂实现
type ClusterConnectionManagerFactory struct {
	manager *ClusterConnectionManager
}

// NewClusterConnectionManagerFactory 创建集群连接管理器工厂
func NewClusterConnectionManagerFactory(manager *ClusterConnectionManager) K8sClientFactory {
	return &ClusterConnectionManagerFactory{
		manager: manager,
	}
}

// GetClient 获取指定集群的kubernetes客户端
func (f *ClusterConnectionManagerFactory) GetClient(clusterName string) (kubernetes.Interface, error) {
	return f.manager.GetClient(clusterName)
}

// GetClientFromManager 从manager获取kubernetes客户端
func (f *ClusterConnectionManagerFactory) GetClientFromManager(clusterName string) (kubernetes.Interface, error) {
	return f.manager.GetClientFromManager(clusterName)
}

// GetConfig 获取指定集群的rest配置
func (f *ClusterConnectionManagerFactory) GetConfig(clusterName string) (*rest.Config, error) {
	return f.manager.GetConfig(clusterName)
}

// GetManager 获取指定集群的controller-runtime管理器
func (f *ClusterConnectionManagerFactory) GetManager(clusterName string) (manager.Manager, error) {
	return f.manager.GetManager(clusterName)
}

// ListClusters 列出所有可用集群
func (f *ClusterConnectionManagerFactory) ListClusters() []string {
	return f.manager.ListClusters()
}

// IsClusterAvailable 检查集群是否可用
func (f *ClusterConnectionManagerFactory) IsClusterAvailable(clusterName string) bool {
	return f.manager.IsClusterConnected(clusterName)
}

// TestConnection 测试集群连接
func (f *ClusterConnectionManagerFactory) TestConnection(ctx context.Context, clusterName string) error {
	return f.manager.TestClusterConnection(clusterName)
}

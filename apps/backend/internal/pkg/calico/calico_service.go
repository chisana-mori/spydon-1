package calico

import (
	"context"
	"fmt"
	"time"

	"robusta-web/backend/internal/pkg/nodesync"

	calicov3 "github.com/projectcalico/api/pkg/apis/projectcalico/v3"
	"k8s.io/apimachinery/pkg/api/meta"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Service 提供 Calico CRD 资源查询能力
type Service struct {
	nodeSyncManager *nodesync.Manager
}

// NewService 创建 Calico 服务
func NewService(manager *nodesync.Manager) *Service {
	return &Service{
		nodeSyncManager: manager,
	}
}

// getClient 获取指定集群的 K8s client
func (s *Service) getClient(clusterName string) (client.Client, error) {
	c, err := s.nodeSyncManager.GetClient(clusterName)
	if err != nil {
		return nil, err
	}
	k8sClient, ok := c.(client.Client)
	if !ok {
		return nil, fmt.Errorf("无法转换为 client.Client 类型")
	}
	return k8sClient, nil
}

// defaultContext 创建默认超时 context
func defaultContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 30*time.Second)
}

// isNoMatchError 检查是否为 CRD 不存在错误
func isNoMatchError(err error) bool {
	return meta.IsNoMatchError(err)
}

// ========== IPPool 相关 ==========

// ListIPPools 列出指定集群的所有 IPPool
func (s *Service) ListIPPools(clusterName string) ([]calicov3.IPPool, error) {
	k8sClient, err := s.getClient(clusterName)
	if err != nil {
		return nil, err
	}

	ctx, cancel := defaultContext()
	defer cancel()

	var list calicov3.IPPoolList
	if err := k8sClient.List(ctx, &list); err != nil {
		if isNoMatchError(err) {
			return []calicov3.IPPool{}, nil
		}
		return nil, fmt.Errorf("获取 IPPool 列表失败: %w", err)
	}
	return list.Items, nil
}

// GetIPPool 获取指定 IPPool
func (s *Service) GetIPPool(clusterName, name string) (*calicov3.IPPool, error) {
	k8sClient, err := s.getClient(clusterName)
	if err != nil {
		return nil, err
	}

	ctx, cancel := defaultContext()
	defer cancel()

	var item calicov3.IPPool
	if err := k8sClient.Get(ctx, client.ObjectKey{Name: name}, &item); err != nil {
		return nil, fmt.Errorf("获取 IPPool 失败: %w", err)
	}
	return &item, nil
}

// ========== IPReservation 相关 ==========

// ListIPReservations 列出指定集群的所有 IPReservation
func (s *Service) ListIPReservations(clusterName string) ([]calicov3.IPReservation, error) {
	k8sClient, err := s.getClient(clusterName)
	if err != nil {
		return nil, err
	}

	ctx, cancel := defaultContext()
	defer cancel()

	var list calicov3.IPReservationList
	if err := k8sClient.List(ctx, &list); err != nil {
		if isNoMatchError(err) {
			return []calicov3.IPReservation{}, nil
		}
		return nil, fmt.Errorf("获取 IPReservation 列表失败: %w", err)
	}
	return list.Items, nil
}

// ========== BGPConfiguration 相关 ==========

// ListBGPConfigurations 列出指定集群的所有 BGP 配置
func (s *Service) ListBGPConfigurations(clusterName string) ([]calicov3.BGPConfiguration, error) {
	k8sClient, err := s.getClient(clusterName)
	if err != nil {
		return nil, err
	}

	ctx, cancel := defaultContext()
	defer cancel()

	var list calicov3.BGPConfigurationList
	if err := k8sClient.List(ctx, &list); err != nil {
		if isNoMatchError(err) {
			return []calicov3.BGPConfiguration{}, nil
		}
		return nil, fmt.Errorf("获取 BGPConfiguration 列表失败: %w", err)
	}
	return list.Items, nil
}

// GetBGPConfiguration 获取指定 BGP 配置
func (s *Service) GetBGPConfiguration(clusterName, name string) (*calicov3.BGPConfiguration, error) {
	k8sClient, err := s.getClient(clusterName)
	if err != nil {
		return nil, err
	}

	ctx, cancel := defaultContext()
	defer cancel()

	var item calicov3.BGPConfiguration
	if err := k8sClient.Get(ctx, client.ObjectKey{Name: name}, &item); err != nil {
		return nil, fmt.Errorf("获取 BGPConfiguration 失败: %w", err)
	}
	return &item, nil
}

// ========== BGPPeer 相关 ==========

// ListBGPPeers 列出指定集群的所有 BGP Peer
func (s *Service) ListBGPPeers(clusterName string) ([]calicov3.BGPPeer, error) {
	k8sClient, err := s.getClient(clusterName)
	if err != nil {
		return nil, err
	}

	ctx, cancel := defaultContext()
	defer cancel()

	var list calicov3.BGPPeerList
	if err := k8sClient.List(ctx, &list); err != nil {
		if isNoMatchError(err) {
			return []calicov3.BGPPeer{}, nil
		}
		return nil, fmt.Errorf("获取 BGPPeer 列表失败: %w", err)
	}
	return list.Items, nil
}

// GetBGPPeer 获取指定 BGP Peer
func (s *Service) GetBGPPeer(clusterName, name string) (*calicov3.BGPPeer, error) {
	k8sClient, err := s.getClient(clusterName)
	if err != nil {
		return nil, err
	}

	ctx, cancel := defaultContext()
	defer cancel()

	var item calicov3.BGPPeer
	if err := k8sClient.Get(ctx, client.ObjectKey{Name: name}, &item); err != nil {
		return nil, fmt.Errorf("获取 BGPPeer 失败: %w", err)
	}
	return &item, nil
}

// ========== GlobalNetworkPolicy 相关 ==========

// ListGlobalNetworkPolicies 列出指定集群的所有全局网络策略
func (s *Service) ListGlobalNetworkPolicies(clusterName string) ([]calicov3.GlobalNetworkPolicy, error) {
	k8sClient, err := s.getClient(clusterName)
	if err != nil {
		return nil, err
	}

	ctx, cancel := defaultContext()
	defer cancel()

	var list calicov3.GlobalNetworkPolicyList
	if err := k8sClient.List(ctx, &list); err != nil {
		if isNoMatchError(err) {
			return []calicov3.GlobalNetworkPolicy{}, nil
		}
		return nil, fmt.Errorf("获取 GlobalNetworkPolicy 列表失败: %w", err)
	}
	return list.Items, nil
}

// GetGlobalNetworkPolicy 获取指定全局网络策略
func (s *Service) GetGlobalNetworkPolicy(clusterName, name string) (*calicov3.GlobalNetworkPolicy, error) {
	k8sClient, err := s.getClient(clusterName)
	if err != nil {
		return nil, err
	}

	ctx, cancel := defaultContext()
	defer cancel()

	var item calicov3.GlobalNetworkPolicy
	if err := k8sClient.Get(ctx, client.ObjectKey{Name: name}, &item); err != nil {
		return nil, fmt.Errorf("获取 GlobalNetworkPolicy 失败: %w", err)
	}
	return &item, nil
}

// ========== NetworkPolicy 相关 ==========

// ListNetworkPolicies 列出指定集群指定命名空间的网络策略
func (s *Service) ListNetworkPolicies(clusterName, namespace string) ([]calicov3.NetworkPolicy, error) {
	k8sClient, err := s.getClient(clusterName)
	if err != nil {
		return nil, err
	}

	ctx, cancel := defaultContext()
	defer cancel()

	var list calicov3.NetworkPolicyList
	opts := []client.ListOption{}
	if namespace != "" {
		opts = append(opts, client.InNamespace(namespace))
	}

	if err := k8sClient.List(ctx, &list, opts...); err != nil {
		if isNoMatchError(err) {
			return []calicov3.NetworkPolicy{}, nil
		}
		return nil, fmt.Errorf("获取 NetworkPolicy 列表失败: %w", err)
	}
	return list.Items, nil
}

// GetNetworkPolicy 获取指定网络策略
func (s *Service) GetNetworkPolicy(clusterName, namespace, name string) (*calicov3.NetworkPolicy, error) {
	k8sClient, err := s.getClient(clusterName)
	if err != nil {
		return nil, err
	}

	ctx, cancel := defaultContext()
	defer cancel()

	var item calicov3.NetworkPolicy
	if err := k8sClient.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, &item); err != nil {
		return nil, fmt.Errorf("获取 NetworkPolicy 失败: %w", err)
	}
	return &item, nil
}

// ========== GlobalNetworkSet 相关 ==========

// ListGlobalNetworkSets 列出指定集群的所有全局网络集
func (s *Service) ListGlobalNetworkSets(clusterName string) ([]calicov3.GlobalNetworkSet, error) {
	k8sClient, err := s.getClient(clusterName)
	if err != nil {
		return nil, err
	}

	ctx, cancel := defaultContext()
	defer cancel()

	var list calicov3.GlobalNetworkSetList
	if err := k8sClient.List(ctx, &list); err != nil {
		if isNoMatchError(err) {
			return []calicov3.GlobalNetworkSet{}, nil
		}
		return nil, fmt.Errorf("获取 GlobalNetworkSet 列表失败: %w", err)
	}
	return list.Items, nil
}

// ========== NetworkSet 相关 ==========

// ListNetworkSets 列出指定集群指定命名空间的网络集
func (s *Service) ListNetworkSets(clusterName, namespace string) ([]calicov3.NetworkSet, error) {
	k8sClient, err := s.getClient(clusterName)
	if err != nil {
		return nil, err
	}

	ctx, cancel := defaultContext()
	defer cancel()

	var list calicov3.NetworkSetList
	opts := []client.ListOption{}
	if namespace != "" {
		opts = append(opts, client.InNamespace(namespace))
	}

	if err := k8sClient.List(ctx, &list, opts...); err != nil {
		if isNoMatchError(err) {
			return []calicov3.NetworkSet{}, nil
		}
		return nil, fmt.Errorf("获取 NetworkSet 列表失败: %w", err)
	}
	return list.Items, nil
}

// ========== HostEndpoint 相关 ==========

// ListHostEndpoints 列出指定集群的所有主机端点
func (s *Service) ListHostEndpoints(clusterName string) ([]calicov3.HostEndpoint, error) {
	k8sClient, err := s.getClient(clusterName)
	if err != nil {
		return nil, err
	}

	ctx, cancel := defaultContext()
	defer cancel()

	var list calicov3.HostEndpointList
	if err := k8sClient.List(ctx, &list); err != nil {
		if isNoMatchError(err) {
			return []calicov3.HostEndpoint{}, nil
		}
		return nil, fmt.Errorf("获取 HostEndpoint 列表失败: %w", err)
	}
	return list.Items, nil
}

// GetHostEndpoint 获取指定主机端点
func (s *Service) GetHostEndpoint(clusterName, name string) (*calicov3.HostEndpoint, error) {
	k8sClient, err := s.getClient(clusterName)
	if err != nil {
		return nil, err
	}

	ctx, cancel := defaultContext()
	defer cancel()

	var item calicov3.HostEndpoint
	if err := k8sClient.Get(ctx, client.ObjectKey{Name: name}, &item); err != nil {
		return nil, fmt.Errorf("获取 HostEndpoint 失败: %w", err)
	}
	return &item, nil
}

// ========== FelixConfiguration 相关 ==========

// ListFelixConfigurations 列出指定集群的所有 Felix 配置
func (s *Service) ListFelixConfigurations(clusterName string) ([]calicov3.FelixConfiguration, error) {
	k8sClient, err := s.getClient(clusterName)
	if err != nil {
		return nil, err
	}

	ctx, cancel := defaultContext()
	defer cancel()

	var list calicov3.FelixConfigurationList
	if err := k8sClient.List(ctx, &list); err != nil {
		if isNoMatchError(err) {
			return []calicov3.FelixConfiguration{}, nil
		}
		return nil, fmt.Errorf("获取 FelixConfiguration 列表失败: %w", err)
	}
	return list.Items, nil
}

// GetFelixConfiguration 获取指定 Felix 配置
func (s *Service) GetFelixConfiguration(clusterName, name string) (*calicov3.FelixConfiguration, error) {
	k8sClient, err := s.getClient(clusterName)
	if err != nil {
		return nil, err
	}

	ctx, cancel := defaultContext()
	defer cancel()

	var item calicov3.FelixConfiguration
	if err := k8sClient.Get(ctx, client.ObjectKey{Name: name}, &item); err != nil {
		return nil, fmt.Errorf("获取 FelixConfiguration 失败: %w", err)
	}
	return &item, nil
}

// ========== ClusterInformation 相关 ==========

// GetClusterInformation 获取 Calico 集群信息
func (s *Service) GetClusterInformation(clusterName string) (*calicov3.ClusterInformation, error) {
	k8sClient, err := s.getClient(clusterName)
	if err != nil {
		return nil, err
	}

	ctx, cancel := defaultContext()
	defer cancel()

	// ClusterInformation 通常只有一个名为 "default" 的资源
	var item calicov3.ClusterInformation
	if err := k8sClient.Get(ctx, client.ObjectKey{Name: "default"}, &item); err != nil {
		if isNoMatchError(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("获取 ClusterInformation 失败: %w", err)
	}
	return &item, nil
}

// ========== KubeControllersConfiguration 相关 ==========

// ListKubeControllersConfigurations 列出指定集群的所有 KubeControllers 配置
func (s *Service) ListKubeControllersConfigurations(clusterName string) ([]calicov3.KubeControllersConfiguration, error) {
	k8sClient, err := s.getClient(clusterName)
	if err != nil {
		return nil, err
	}

	ctx, cancel := defaultContext()
	defer cancel()

	var list calicov3.KubeControllersConfigurationList
	if err := k8sClient.List(ctx, &list); err != nil {
		if isNoMatchError(err) {
			return []calicov3.KubeControllersConfiguration{}, nil
		}
		return nil, fmt.Errorf("获取 KubeControllersConfiguration 列表失败: %w", err)
	}
	return list.Items, nil
}

// ========== Tier 相关 ==========

// ListTiers 列出指定集群的所有策略层级
func (s *Service) ListTiers(clusterName string) ([]calicov3.Tier, error) {
	k8sClient, err := s.getClient(clusterName)
	if err != nil {
		return nil, err
	}

	ctx, cancel := defaultContext()
	defer cancel()

	var list calicov3.TierList
	if err := k8sClient.List(ctx, &list); err != nil {
		if isNoMatchError(err) {
			return []calicov3.Tier{}, nil
		}
		return nil, fmt.Errorf("获取 Tier 列表失败: %w", err)
	}
	return list.Items, nil
}

// GetTier 获取指定策略层级
func (s *Service) GetTier(clusterName, name string) (*calicov3.Tier, error) {
	k8sClient, err := s.getClient(clusterName)
	if err != nil {
		return nil, err
	}

	ctx, cancel := defaultContext()
	defer cancel()

	var item calicov3.Tier
	if err := k8sClient.Get(ctx, client.ObjectKey{Name: name}, &item); err != nil {
		return nil, fmt.Errorf("获取 Tier 失败: %w", err)
	}
	return &item, nil
}

// ========== CalicoNodeStatus 相关 ==========

// ListCalicoNodeStatuses 列出指定集群的所有 Calico 节点状态
func (s *Service) ListCalicoNodeStatuses(clusterName string) ([]calicov3.CalicoNodeStatus, error) {
	k8sClient, err := s.getClient(clusterName)
	if err != nil {
		return nil, err
	}

	ctx, cancel := defaultContext()
	defer cancel()

	var list calicov3.CalicoNodeStatusList
	if err := k8sClient.List(ctx, &list); err != nil {
		if isNoMatchError(err) {
			return []calicov3.CalicoNodeStatus{}, nil
		}
		return nil, fmt.Errorf("获取 CalicoNodeStatus 列表失败: %w", err)
	}
	return list.Items, nil
}

// GetCalicoNodeStatus 获取指定 Calico 节点状态
func (s *Service) GetCalicoNodeStatus(clusterName, name string) (*calicov3.CalicoNodeStatus, error) {
	k8sClient, err := s.getClient(clusterName)
	if err != nil {
		return nil, err
	}

	ctx, cancel := defaultContext()
	defer cancel()

	var item calicov3.CalicoNodeStatus
	if err := k8sClient.Get(ctx, client.ObjectKey{Name: name}, &item); err != nil {
		return nil, fmt.Errorf("获取 CalicoNodeStatus 失败: %w", err)
	}
	return &item, nil
}

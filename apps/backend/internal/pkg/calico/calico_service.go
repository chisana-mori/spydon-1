package calico

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"robusta-web/backend/internal/config"
	"robusta-web/backend/internal/pkg/nodesync"
	"robusta-web/backend/pkg/logger"
	"robusta-web/backend/pkg/wayne_api"

	calicov3 "github.com/projectcalico/api/pkg/apis/projectcalico/v3"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Service provides Calico CRD resource query capabilities with automatic v3/v1 API adaptation
type Service struct {
	nodeSyncManager *nodesync.Manager
	config          *config.Config
	versionCache    map[string]APIVersion
	dynamicClients  map[string]dynamic.Interface
	cacheMu         sync.RWMutex
}

// NewService creates a new Calico service instance
func NewService(manager *nodesync.Manager, cfg *config.Config) *Service {
	return &Service{
		nodeSyncManager: manager,
		config:          cfg,
		versionCache:    make(map[string]APIVersion),
		dynamicClients:  make(map[string]dynamic.Interface),
	}
}

// ========== Client & Version Management ==========

// getClient gets the K8s client for the specified cluster (typed, for v3 API)
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

// getAPIVersion gets the Calico API version for a cluster with caching
func (s *Service) getAPIVersion(clusterName string) APIVersion {
	s.cacheMu.RLock()
	version, exists := s.versionCache[clusterName]
	s.cacheMu.RUnlock()

	if exists {
		return version
	}

	version = s.detectAPIVersion(clusterName)

	s.cacheMu.Lock()
	s.versionCache[clusterName] = version
	s.cacheMu.Unlock()

	return version
}

// detectAPIVersion detects the Calico API version supported by the cluster
func (s *Service) detectAPIVersion(clusterName string) APIVersion {
	k8sClient, err := s.getClient(clusterName)
	if err != nil {
		logger.S().Warnw("获取 k8s client 失败，使用 v1 API", "cluster", clusterName, "error", err)
		return APIVersionV1
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var list calicov3.IPPoolList
	if err := k8sClient.List(ctx, &list); err != nil {
		if isNoMatchError(err) {
			logger.S().Infow("集群不支持 v3 API，使用 v1 API", "cluster", clusterName)
			return APIVersionV1
		}
		logger.S().Warnw("v3 API 查询失败，使用 v1 API", "cluster", clusterName, "error", err)
		return APIVersionV1
	}

	logger.S().Infow("集群支持 v3 API", "cluster", clusterName)
	return APIVersionV3
}

// ========== Utility Functions ==========

const (
	// defaultTimeout is the default timeout for API calls
	defaultTimeout = 30 * time.Second
	// versionDetectTimeout is the timeout for API version detection
	versionDetectTimeout = 10 * time.Second
	// defaultClusterInfoName is the default name for ClusterInformation resource
	defaultClusterInfoName = "default"
)

// defaultContext creates a context with default timeout
func defaultContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), defaultTimeout)
}

// isNoMatchError checks if the error indicates CRD doesn't exist
func isNoMatchError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return containsAny(errStr, "no matches for kind", "the server could not find the requested resource")
}

// containsAny checks if the string contains any of the substrings
func containsAny(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if len(s) >= len(sub) {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
		}
	}
	return false
}

// formatError formats an error message with resource type
func formatError(action, resourceType string, err error) error {
	return fmt.Errorf("%s %s 失败: %w", action, resourceType, err)
}

// ========== Generic Query Functions ==========

// listOptions holds options for listing resources
type listOptions struct {
	namespace string // empty for cluster-scoped resources
}

// getOptions holds options for getting a single resource
type getOptions struct {
	namespace      string // empty for cluster-scoped resources
	name           string // resource name
	ignoreNotFound bool   // if true, return nil, nil on not found
}

// listWithFallback is a generic list function with v3/v1 fallback
func listWithFallback[T any](
	s *Service,
	clusterName string,
	resourceType string,
	v3ListFunc func(client.Client, context.Context, []client.ListOption) ([]T, error),
	gvr schema.GroupVersionResource,
	converter func(*unstructured.Unstructured) (T, error),
	opts listOptions,
) ([]T, error) {
	version := s.getAPIVersion(clusterName)

	if version == APIVersionV3 {
		return listV3(s, clusterName, resourceType, v3ListFunc, opts)
	}

	return listV1WithNamespace(s, clusterName, opts.namespace, gvr, converter)
}

// listV3 lists resources using the v3 typed API
func listV3[T any](
	s *Service,
	clusterName string,
	resourceType string,
	listFunc func(client.Client, context.Context, []client.ListOption) ([]T, error),
	opts listOptions,
) ([]T, error) {
	k8sClient, err := s.getClient(clusterName)
	if err != nil {
		return nil, err
	}

	ctx, cancel := defaultContext()
	defer cancel()

	listOpts := []client.ListOption{}
	if opts.namespace != "" {
		listOpts = append(listOpts, client.InNamespace(opts.namespace))
	}

	items, err := listFunc(k8sClient, ctx, listOpts)
	if err != nil {
		if isNoMatchError(err) {
			return []T{}, nil
		}
		return nil, formatError("获取", resourceType, err)
	}

	return items, nil
}

// getWithFallback is a generic get function with v3/v1 fallback
func getWithFallback[T any](
	s *Service,
	clusterName string,
	resourceType string,
	v3GetFunc func(client.Client, context.Context, client.ObjectKey) (T, error),
	gvr schema.GroupVersionResource,
	converter func(*unstructured.Unstructured) (T, error),
	opts getOptions,
) (*T, error) {
	version := s.getAPIVersion(clusterName)

	if version == APIVersionV3 {
		return getV3(s, clusterName, resourceType, v3GetFunc, opts)
	}

	result, err := getV1WithNamespace(s, clusterName, opts.namespace, opts.name, gvr, converter)
	if err != nil {
		if opts.ignoreNotFound && isNoMatchError(err) {
			return nil, nil
		}
		return nil, err
	}
	return &result, nil
}

// getV3 gets a single resource using the v3 typed API
func getV3[T any](
	s *Service,
	clusterName string,
	resourceType string,
	getFunc func(client.Client, context.Context, client.ObjectKey) (T, error),
	opts getOptions,
) (*T, error) {
	k8sClient, err := s.getClient(clusterName)
	if err != nil {
		return nil, err
	}

	ctx, cancel := defaultContext()
	defer cancel()

	key := client.ObjectKey{Name: opts.name}
	if opts.namespace != "" {
		key.Namespace = opts.namespace
	}

	item, err := getFunc(k8sClient, ctx, key)
	if err != nil {
		if opts.ignoreNotFound && isNoMatchError(err) {
			return nil, nil
		}
		return nil, formatError("获取", resourceType, err)
	}

	return &item, nil
}

// ========== IPPool Operations ==========

// ListIPPools lists all IPPools in the specified cluster
func (s *Service) ListIPPools(clusterName string) ([]calicov3.IPPool, error) {
	return listWithFallback(
		s, clusterName, "IPPool",
		func(c client.Client, ctx context.Context, opts []client.ListOption) ([]calicov3.IPPool, error) {
			var list calicov3.IPPoolList
			if err := c.List(ctx, &list, opts...); err != nil {
				return nil, err
			}
			return list.Items, nil
		},
		IPPoolGVR,
		convertToIPPool,
		listOptions{},
	)
}

// GetIPPool gets a specific IPPool by name
func (s *Service) GetIPPool(clusterName, name string) (*calicov3.IPPool, error) {
	return getWithFallback(
		s, clusterName, "IPPool",
		func(c client.Client, ctx context.Context, key client.ObjectKey) (calicov3.IPPool, error) {
			var item calicov3.IPPool
			if err := c.Get(ctx, key, &item); err != nil {
				return item, err
			}
			return item, nil
		},
		IPPoolGVR,
		convertToIPPool,
		getOptions{name: name},
	)
}

// ========== IPReservation Operations ==========

// ListIPReservations lists all IPReservations in the specified cluster
func (s *Service) ListIPReservations(clusterName string) ([]calicov3.IPReservation, error) {
	return listWithFallback(
		s, clusterName, "IPReservation",
		func(c client.Client, ctx context.Context, opts []client.ListOption) ([]calicov3.IPReservation, error) {
			var list calicov3.IPReservationList
			if err := c.List(ctx, &list, opts...); err != nil {
				return nil, err
			}
			return list.Items, nil
		},
		IPReservationGVR,
		convertToIPReservation,
		listOptions{},
	)
}

// ========== BGPConfiguration Operations ==========

// ListBGPConfigurations lists all BGP configurations in the specified cluster
func (s *Service) ListBGPConfigurations(clusterName string) ([]calicov3.BGPConfiguration, error) {
	return listWithFallback(
		s, clusterName, "BGPConfiguration",
		func(c client.Client, ctx context.Context, opts []client.ListOption) ([]calicov3.BGPConfiguration, error) {
			var list calicov3.BGPConfigurationList
			if err := c.List(ctx, &list, opts...); err != nil {
				return nil, err
			}
			return list.Items, nil
		},
		BGPConfigurationGVR,
		convertToBGPConfiguration,
		listOptions{},
	)
}

// GetBGPConfiguration gets a specific BGP configuration by name
func (s *Service) GetBGPConfiguration(clusterName, name string) (*calicov3.BGPConfiguration, error) {
	return getWithFallback(
		s, clusterName, "BGPConfiguration",
		func(c client.Client, ctx context.Context, key client.ObjectKey) (calicov3.BGPConfiguration, error) {
			var item calicov3.BGPConfiguration
			if err := c.Get(ctx, key, &item); err != nil {
				return item, err
			}
			return item, nil
		},
		BGPConfigurationGVR,
		convertToBGPConfiguration,
		getOptions{name: name},
	)
}

// ========== BGPPeer Operations ==========

// ListBGPPeers lists all BGP peers in the specified cluster
func (s *Service) ListBGPPeers(clusterName string) ([]calicov3.BGPPeer, error) {
	return listWithFallback(
		s, clusterName, "BGPPeer",
		func(c client.Client, ctx context.Context, opts []client.ListOption) ([]calicov3.BGPPeer, error) {
			var list calicov3.BGPPeerList
			if err := c.List(ctx, &list, opts...); err != nil {
				return nil, err
			}
			return list.Items, nil
		},
		BGPPeerGVR,
		convertToBGPPeer,
		listOptions{},
	)
}

// GetBGPPeer gets a specific BGP peer by name
func (s *Service) GetBGPPeer(clusterName, name string) (*calicov3.BGPPeer, error) {
	return getWithFallback(
		s, clusterName, "BGPPeer",
		func(c client.Client, ctx context.Context, key client.ObjectKey) (calicov3.BGPPeer, error) {
			var item calicov3.BGPPeer
			if err := c.Get(ctx, key, &item); err != nil {
				return item, err
			}
			return item, nil
		},
		BGPPeerGVR,
		convertToBGPPeer,
		getOptions{name: name},
	)
}

// ========== GlobalNetworkPolicy Operations ==========

// ListGlobalNetworkPolicies lists all global network policies in the specified cluster
func (s *Service) ListGlobalNetworkPolicies(clusterName string) ([]calicov3.GlobalNetworkPolicy, error) {
	return listWithFallback(
		s, clusterName, "GlobalNetworkPolicy",
		func(c client.Client, ctx context.Context, opts []client.ListOption) ([]calicov3.GlobalNetworkPolicy, error) {
			var list calicov3.GlobalNetworkPolicyList
			if err := c.List(ctx, &list, opts...); err != nil {
				return nil, err
			}
			return list.Items, nil
		},
		GlobalNetworkPolicyGVR,
		convertToGlobalNetworkPolicy,
		listOptions{},
	)
}

// GetGlobalNetworkPolicy gets a specific global network policy by name
func (s *Service) GetGlobalNetworkPolicy(clusterName, name string) (*calicov3.GlobalNetworkPolicy, error) {
	return getWithFallback(
		s, clusterName, "GlobalNetworkPolicy",
		func(c client.Client, ctx context.Context, key client.ObjectKey) (calicov3.GlobalNetworkPolicy, error) {
			var item calicov3.GlobalNetworkPolicy
			if err := c.Get(ctx, key, &item); err != nil {
				return item, err
			}
			return item, nil
		},
		GlobalNetworkPolicyGVR,
		convertToGlobalNetworkPolicy,
		getOptions{name: name},
	)
}

// ========== NetworkPolicy Operations ==========

// ListNetworkPolicies lists network policies in the specified cluster and namespace
func (s *Service) ListNetworkPolicies(clusterName, namespace string) ([]calicov3.NetworkPolicy, error) {
	return listWithFallback(
		s, clusterName, "NetworkPolicy",
		func(c client.Client, ctx context.Context, opts []client.ListOption) ([]calicov3.NetworkPolicy, error) {
			var list calicov3.NetworkPolicyList
			if err := c.List(ctx, &list, opts...); err != nil {
				return nil, err
			}
			return list.Items, nil
		},
		NetworkPolicyGVR,
		convertToNetworkPolicy,
		listOptions{namespace: namespace},
	)
}

// GetNetworkPolicy gets a specific network policy by name and namespace
func (s *Service) GetNetworkPolicy(clusterName, namespace, name string) (*calicov3.NetworkPolicy, error) {
	return getWithFallback(
		s, clusterName, "NetworkPolicy",
		func(c client.Client, ctx context.Context, key client.ObjectKey) (calicov3.NetworkPolicy, error) {
			var item calicov3.NetworkPolicy
			if err := c.Get(ctx, key, &item); err != nil {
				return item, err
			}
			return item, nil
		},
		NetworkPolicyGVR,
		convertToNetworkPolicy,
		getOptions{namespace: namespace, name: name},
	)
}

// ========== GlobalNetworkSet Operations ==========

// ListGlobalNetworkSets lists all global network sets in the specified cluster
func (s *Service) ListGlobalNetworkSets(clusterName string) ([]calicov3.GlobalNetworkSet, error) {
	return listWithFallback(
		s, clusterName, "GlobalNetworkSet",
		func(c client.Client, ctx context.Context, opts []client.ListOption) ([]calicov3.GlobalNetworkSet, error) {
			var list calicov3.GlobalNetworkSetList
			if err := c.List(ctx, &list, opts...); err != nil {
				return nil, err
			}
			return list.Items, nil
		},
		GlobalNetworkSetGVR,
		convertToGlobalNetworkSet,
		listOptions{},
	)
}

// ========== NetworkSet Operations ==========

// ListNetworkSets lists network sets in the specified cluster and namespace
func (s *Service) ListNetworkSets(clusterName, namespace string) ([]calicov3.NetworkSet, error) {
	return listWithFallback(
		s, clusterName, "NetworkSet",
		func(c client.Client, ctx context.Context, opts []client.ListOption) ([]calicov3.NetworkSet, error) {
			var list calicov3.NetworkSetList
			if err := c.List(ctx, &list, opts...); err != nil {
				return nil, err
			}
			return list.Items, nil
		},
		NetworkSetGVR,
		convertToNetworkSet,
		listOptions{namespace: namespace},
	)
}

// ========== HostEndpoint Operations ==========

// ListHostEndpoints lists all host endpoints in the specified cluster
func (s *Service) ListHostEndpoints(clusterName string) ([]calicov3.HostEndpoint, error) {
	return listWithFallback(
		s, clusterName, "HostEndpoint",
		func(c client.Client, ctx context.Context, opts []client.ListOption) ([]calicov3.HostEndpoint, error) {
			var list calicov3.HostEndpointList
			if err := c.List(ctx, &list, opts...); err != nil {
				return nil, err
			}
			return list.Items, nil
		},
		HostEndpointGVR,
		convertToHostEndpoint,
		listOptions{},
	)
}

// GetHostEndpoint gets a specific host endpoint by name
func (s *Service) GetHostEndpoint(clusterName, name string) (*calicov3.HostEndpoint, error) {
	return getWithFallback(
		s, clusterName, "HostEndpoint",
		func(c client.Client, ctx context.Context, key client.ObjectKey) (calicov3.HostEndpoint, error) {
			var item calicov3.HostEndpoint
			if err := c.Get(ctx, key, &item); err != nil {
				return item, err
			}
			return item, nil
		},
		HostEndpointGVR,
		convertToHostEndpoint,
		getOptions{name: name},
	)
}

// ========== FelixConfiguration Operations ==========

// ListFelixConfigurations lists all Felix configurations in the specified cluster
func (s *Service) ListFelixConfigurations(clusterName string) ([]calicov3.FelixConfiguration, error) {
	return listWithFallback(
		s, clusterName, "FelixConfiguration",
		func(c client.Client, ctx context.Context, opts []client.ListOption) ([]calicov3.FelixConfiguration, error) {
			var list calicov3.FelixConfigurationList
			if err := c.List(ctx, &list, opts...); err != nil {
				return nil, err
			}
			return list.Items, nil
		},
		FelixConfigurationGVR,
		convertToFelixConfiguration,
		listOptions{},
	)
}

// GetFelixConfiguration gets a specific Felix configuration by name
func (s *Service) GetFelixConfiguration(clusterName, name string) (*calicov3.FelixConfiguration, error) {
	return getWithFallback(
		s, clusterName, "FelixConfiguration",
		func(c client.Client, ctx context.Context, key client.ObjectKey) (calicov3.FelixConfiguration, error) {
			var item calicov3.FelixConfiguration
			if err := c.Get(ctx, key, &item); err != nil {
				return item, err
			}
			return item, nil
		},
		FelixConfigurationGVR,
		convertToFelixConfiguration,
		getOptions{name: name},
	)
}

// ========== ClusterInformation Operations ==========

// GetClusterInformation gets the Calico cluster information
func (s *Service) GetClusterInformation(clusterName string) (*calicov3.ClusterInformation, error) {
	return getWithFallback(
		s, clusterName, "ClusterInformation",
		func(c client.Client, ctx context.Context, key client.ObjectKey) (calicov3.ClusterInformation, error) {
			var item calicov3.ClusterInformation
			if err := c.Get(ctx, key, &item); err != nil {
				return item, err
			}
			return item, nil
		},
		ClusterInformationGVR,
		convertToClusterInformation,
		getOptions{
			name:           defaultClusterInfoName,
			ignoreNotFound: true,
		},
	)
}

// ========== KubeControllersConfiguration Operations ==========

// ListKubeControllersConfigurations lists all KubeControllers configurations in the specified cluster
func (s *Service) ListKubeControllersConfigurations(clusterName string) ([]calicov3.KubeControllersConfiguration, error) {
	return listWithFallback(
		s, clusterName, "KubeControllersConfiguration",
		func(c client.Client, ctx context.Context, opts []client.ListOption) ([]calicov3.KubeControllersConfiguration, error) {
			var list calicov3.KubeControllersConfigurationList
			if err := c.List(ctx, &list, opts...); err != nil {
				return nil, err
			}
			return list.Items, nil
		},
		KubeControllersConfigurationGVR,
		convertToKubeControllersConfiguration,
		listOptions{},
	)
}

// ========== Tier Operations ==========

// ListTiers lists all policy tiers in the specified cluster
func (s *Service) ListTiers(clusterName string) ([]calicov3.Tier, error) {
	return listWithFallback(
		s, clusterName, "Tier",
		func(c client.Client, ctx context.Context, opts []client.ListOption) ([]calicov3.Tier, error) {
			var list calicov3.TierList
			if err := c.List(ctx, &list, opts...); err != nil {
				return nil, err
			}
			return list.Items, nil
		},
		TierGVR,
		convertToTier,
		listOptions{},
	)
}

// GetTier gets a specific tier by name
func (s *Service) GetTier(clusterName, name string) (*calicov3.Tier, error) {
	return getWithFallback(
		s, clusterName, "Tier",
		func(c client.Client, ctx context.Context, key client.ObjectKey) (calicov3.Tier, error) {
			var item calicov3.Tier
			if err := c.Get(ctx, key, &item); err != nil {
				return item, err
			}
			return item, nil
		},
		TierGVR,
		convertToTier,
		getOptions{name: name},
	)
}

// ========== CalicoNodeStatus Operations ==========

// ListCalicoNodeStatuses lists all Calico node statuses in the specified cluster
func (s *Service) ListCalicoNodeStatuses(clusterName string) ([]calicov3.CalicoNodeStatus, error) {
	return listWithFallback(
		s, clusterName, "CalicoNodeStatus",
		func(c client.Client, ctx context.Context, opts []client.ListOption) ([]calicov3.CalicoNodeStatus, error) {
			var list calicov3.CalicoNodeStatusList
			if err := c.List(ctx, &list, opts...); err != nil {
				return nil, err
			}
			return list.Items, nil
		},
		CalicoNodeStatusGVR,
		convertToCalicoNodeStatus,
		listOptions{},
	)
}

// GetCalicoNodeStatus gets a specific Calico node status by name
func (s *Service) GetCalicoNodeStatus(clusterName, name string) (*calicov3.CalicoNodeStatus, error) {
	return getWithFallback(
		s, clusterName, "CalicoNodeStatus",
		func(c client.Client, ctx context.Context, key client.ObjectKey) (calicov3.CalicoNodeStatus, error) {
			var item calicov3.CalicoNodeStatus
			if err := c.Get(ctx, key, &item); err != nil {
				return item, err
			}
			return item, nil
		},
		CalicoNodeStatusGVR,
		convertToCalicoNodeStatus,
		getOptions{name: name},
	)
}

// ========== Wayne Integration ==========

// SyncIPPoolsToWayne syncs IPPools from a cluster to Wayne
// user: username of the user performing the sync operation
func (s *Service) SyncIPPoolsToWayne(clusterName, user string) (*SyncRet, error) {
	pools, err := s.ListIPPools(clusterName)
	if err != nil {
		return nil, err
	}

	converter := NewIPPoolConverter()
	waynePools, err := converter.ConvertBatchToModelIPpools(pools, clusterName)
	if err != nil {
		logger.S().Errorw("转换 IPPool 到 Wayne 模型失败", "error", err)
		return nil, fmt.Errorf("模型转换失败: %w", err)
	}

	needSync := filterIPPoolsForSync(waynePools, user)

	respData, err := syncIPPoolsToWayneAPI(s, clusterName, needSync)
	if err != nil {
		return nil, err
	}

	return &SyncRet{
		Created: respData.Created,
		Updated: respData.Updated,
		Deleted: respData.Deleted,
	}, nil
}

// filterIPPoolsForSync filters IPPools for sync, excluding K8SBASE pools
func filterIPPoolsForSync(pools []*WayneIpPool, user string) []*WayneIpPool {
	var filtered []*WayneIpPool
	for _, pool := range pools {
		if pool.DisplayName != "K8SBASE" {
			pool.User = user
			filtered = append(filtered, pool)
		}
	}
	return filtered
}

// wayneSyncRequest represents the request structure for syncing IP pools
type wayneSyncRequest struct {
	ClusterName string         `json:"clusterName"`
	IpPools     []*WayneIpPool `json:"ipPools"`
}

// wayneSyncResponse represents the response structure from Wayne sync API
type wayneSyncResponse struct {
	Data wayneSyncResponseData `json:"data"`
}

// wayneSyncResponseData represents the data portion of Wayne sync response
type wayneSyncResponseData struct {
	Created int64 `json:"created"`
	Updated int64 `json:"updated"`
	Deleted int64 `json:"deleted"`
}

// syncIPPoolsToWayneAPI sends IPPools to Wayne API
func syncIPPoolsToWayneAPI(s *Service, clusterName string, pools []*WayneIpPool) (*wayneSyncResponseData, error) {
	req := wayneSyncRequest{
		ClusterName: clusterName,
		IpPools:     pools,
	}

	body, err := json.Marshal(req)
	if err != nil {
		logger.S().Errorw("序列化 WayneSyncRequest 失败", "error", err)
		return nil, fmt.Errorf("JSON 序列化失败: %w", err)
	}

	client := wayne_api.GetClient()
	if client == nil || !wayne_api.IsConfigured() {
		return nil, fmt.Errorf("Wayne Host 未配置")
	}

	respBytes, err := client.SyncIPPool(body)
	if err != nil {
		logger.S().Errorw("同步 IPPool 到 Wayne 失败", "error", err)
		return nil, fmt.Errorf("Wayne 同步请求失败: %w", err)
	}

	var resp wayneSyncResponse
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		return nil, fmt.Errorf("解析 Wayne 响应失败: %w", err)
	}

	return &resp.Data, nil
}

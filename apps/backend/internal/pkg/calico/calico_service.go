package calico

import (
	"context"
	"encoding/json"
	"fmt"
	"strings" // Added strings
	"sync"
	"time"

	"robusta-web/backend/internal/config"
	"robusta-web/backend/internal/pkg/nodesync"
	"robusta-web/backend/pkg/logger"
	"robusta-web/backend/pkg/wayne_api"

	calicov3 "github.com/projectcalico/api/pkg/apis/projectcalico/v3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	calicov1 "robusta-web/backend/internal/pkg/calico/v1"
	calicov3custom "robusta-web/backend/internal/pkg/calico/v3"
)

func init() {
	_ = calicov1.AddToScheme(nodesync.Scheme)
	_ = calicov3custom.AddToScheme(nodesync.Scheme)
}

// Service provides Calico CRD resource query capabilities with automatic v3/v1 API adaptation
type Service struct {
	nodeSyncManager *nodesync.Manager
	config          *config.Config
	versionCache    map[string]APIVersion
	cacheMu         sync.RWMutex
}

// NewService creates a new Calico service instance
func NewService(manager *nodesync.Manager, cfg *config.Config) *Service {
	return &Service{
		nodeSyncManager: manager,
		config:          cfg,
		versionCache:    make(map[string]APIVersion),
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
	return containsAny(errStr, "no matches for kind", "the server could not find the requested resource", "could not find the requested resource")
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
	v1ListFunc func(client.Client, context.Context, []client.ListOption) ([]T, error),
	opts listOptions,
) ([]T, error) {
	version := s.getAPIVersion(clusterName)

	if version == APIVersionV3 {
		return listGeneric(s, clusterName, resourceType, v3ListFunc, opts)
	}

	return listGeneric(s, clusterName, resourceType, v1ListFunc, opts)
}

// listGeneric lists resources using the provided list function
func listGeneric[T any](
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
	v1GetFunc func(client.Client, context.Context, client.ObjectKey) (T, error),
	opts getOptions,
) (*T, error) {
	version := s.getAPIVersion(clusterName)

	if version == APIVersionV3 {
		return getGeneric(s, clusterName, resourceType, v3GetFunc, opts)
	}

	return getGeneric(s, clusterName, resourceType, v1GetFunc, opts)
}

// getGeneric gets a single resource using the provided get function
func getGeneric[T any](
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
			return nil, nil // Return nil pointer for not found if ignored
		}
		// If ignoreNotFound is true, we should also check for "not found" error specifically from Get
		if opts.ignoreNotFound && client.IgnoreNotFound(err) == nil {
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
		func(c client.Client, ctx context.Context, opts []client.ListOption) ([]calicov3.IPPool, error) {
			var list calicov1.IPPoolList
			if err := c.List(ctx, &list, opts...); err != nil {
				return nil, err
			}
			items := make([]calicov3.IPPool, len(list.Items))
			for i, item := range list.Items {
				items[i] = calicov3.IPPool{
					TypeMeta:   item.TypeMeta,
					ObjectMeta: item.ObjectMeta,
					Spec:       item.Spec,
				}
			}
			return items, nil
		},
		listOptions{},
	)
}

// GetIPPool gets a specific IPPool by name
func (s *Service) GetIPPool(clusterName, name string) (*calicov3.IPPool, error) {
	return getWithFallback(
		s, clusterName, "IPPool",
		func(c client.Client, ctx context.Context, key client.ObjectKey) (calicov3.IPPool, error) {
			var item calicov3.IPPool
			err := c.Get(ctx, key, &item)
			return item, err
		},
		func(c client.Client, ctx context.Context, key client.ObjectKey) (calicov3.IPPool, error) {
			var item calicov1.IPPool
			if err := c.Get(ctx, key, &item); err != nil {
				return calicov3.IPPool{}, err
			}
			return calicov3.IPPool{
				TypeMeta:   item.TypeMeta,
				ObjectMeta: item.ObjectMeta,
				Spec:       item.Spec,
			}, nil
		},
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
		func(c client.Client, ctx context.Context, opts []client.ListOption) ([]calicov3.IPReservation, error) {
			var list calicov1.IPReservationList
			if err := c.List(ctx, &list, opts...); err != nil {
				return nil, err
			}
			items := make([]calicov3.IPReservation, len(list.Items))
			for i, item := range list.Items {
				items[i] = calicov3.IPReservation{
					TypeMeta:   item.TypeMeta,
					ObjectMeta: item.ObjectMeta,
					Spec:       item.Spec,
				}
			}
			return items, nil
		},
		listOptions{},
	)
}

// ========== BGPConfiguration Operations ==========

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
		func(c client.Client, ctx context.Context, opts []client.ListOption) ([]calicov3.BGPConfiguration, error) {
			var list calicov1.BGPConfigurationList
			if err := c.List(ctx, &list, opts...); err != nil {
				return nil, err
			}
			items := make([]calicov3.BGPConfiguration, len(list.Items))
			for i, item := range list.Items {
				items[i] = calicov3.BGPConfiguration{
					TypeMeta:   item.TypeMeta,
					ObjectMeta: item.ObjectMeta,
					Spec:       item.Spec,
				}
			}
			return items, nil
		},
		listOptions{},
	)
}

// GetBGPConfiguration gets a specific BGP configuration by name
func (s *Service) GetBGPConfiguration(clusterName, name string) (*calicov3.BGPConfiguration, error) {
	return getWithFallback(
		s, clusterName, "BGPConfiguration",
		func(c client.Client, ctx context.Context, key client.ObjectKey) (calicov3.BGPConfiguration, error) {
			var item calicov3.BGPConfiguration
			err := c.Get(ctx, key, &item)
			return item, err
		},
		func(c client.Client, ctx context.Context, key client.ObjectKey) (calicov3.BGPConfiguration, error) {
			var item calicov1.BGPConfiguration
			if err := c.Get(ctx, key, &item); err != nil {
				return calicov3.BGPConfiguration{}, err
			}
			return calicov3.BGPConfiguration{
				TypeMeta:   item.TypeMeta,
				ObjectMeta: item.ObjectMeta,
				Spec:       item.Spec,
			}, nil
		},
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
		func(c client.Client, ctx context.Context, opts []client.ListOption) ([]calicov3.BGPPeer, error) {
			var list calicov1.BGPPeerList
			if err := c.List(ctx, &list, opts...); err != nil {
				return nil, err
			}
			items := make([]calicov3.BGPPeer, len(list.Items))
			for i, item := range list.Items {
				items[i] = calicov3.BGPPeer{
					TypeMeta:   item.TypeMeta,
					ObjectMeta: item.ObjectMeta,
					Spec:       item.Spec,
				}
			}
			return items, nil
		},
		listOptions{},
	)
}

// GetBGPPeer gets a specific BGP peer by name
func (s *Service) GetBGPPeer(clusterName, name string) (*calicov3.BGPPeer, error) {
	return getWithFallback(
		s, clusterName, "BGPPeer",
		func(c client.Client, ctx context.Context, key client.ObjectKey) (calicov3.BGPPeer, error) {
			var item calicov3.BGPPeer
			err := c.Get(ctx, key, &item)
			return item, err
		},
		func(c client.Client, ctx context.Context, key client.ObjectKey) (calicov3.BGPPeer, error) {
			var item calicov1.BGPPeer
			if err := c.Get(ctx, key, &item); err != nil {
				return calicov3.BGPPeer{}, err
			}
			return calicov3.BGPPeer{
				TypeMeta:   item.TypeMeta,
				ObjectMeta: item.ObjectMeta,
				Spec:       item.Spec,
			}, nil
		},
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
		func(c client.Client, ctx context.Context, opts []client.ListOption) ([]calicov3.GlobalNetworkPolicy, error) {
			var list calicov1.GlobalNetworkPolicyList
			if err := c.List(ctx, &list, opts...); err != nil {
				return nil, err
			}
			items := make([]calicov3.GlobalNetworkPolicy, len(list.Items))
			for i, item := range list.Items {
				items[i] = calicov3.GlobalNetworkPolicy{
					TypeMeta:   item.TypeMeta,
					ObjectMeta: item.ObjectMeta,
					Spec:       item.Spec,
				}
			}
			return items, nil
		},
		listOptions{},
	)
}

// GetGlobalNetworkPolicy gets a specific global network policy by name
func (s *Service) GetGlobalNetworkPolicy(clusterName, name string) (*calicov3.GlobalNetworkPolicy, error) {
	return getWithFallback(
		s, clusterName, "GlobalNetworkPolicy",
		func(c client.Client, ctx context.Context, key client.ObjectKey) (calicov3.GlobalNetworkPolicy, error) {
			var item calicov3.GlobalNetworkPolicy
			err := c.Get(ctx, key, &item)
			return item, err
		},
		func(c client.Client, ctx context.Context, key client.ObjectKey) (calicov3.GlobalNetworkPolicy, error) {
			var item calicov1.GlobalNetworkPolicy
			if err := c.Get(ctx, key, &item); err != nil {
				return calicov3.GlobalNetworkPolicy{}, err
			}
			return calicov3.GlobalNetworkPolicy{
				TypeMeta:   item.TypeMeta,
				ObjectMeta: item.ObjectMeta,
				Spec:       item.Spec,
			}, nil
		},
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
		func(c client.Client, ctx context.Context, opts []client.ListOption) ([]calicov3.NetworkPolicy, error) {
			var list calicov1.NetworkPolicyList
			if err := c.List(ctx, &list, opts...); err != nil {
				return nil, err
			}
			items := make([]calicov3.NetworkPolicy, len(list.Items))
			for i, item := range list.Items {
				items[i] = calicov3.NetworkPolicy{
					TypeMeta:   item.TypeMeta,
					ObjectMeta: item.ObjectMeta,
					Spec:       item.Spec,
				}
			}
			return items, nil
		},
		listOptions{namespace: namespace},
	)
}

// GetNetworkPolicy gets a specific network policy by name and namespace
func (s *Service) GetNetworkPolicy(clusterName, namespace, name string) (*calicov3.NetworkPolicy, error) {
	return getWithFallback(
		s, clusterName, "NetworkPolicy",
		func(c client.Client, ctx context.Context, key client.ObjectKey) (calicov3.NetworkPolicy, error) {
			var item calicov3.NetworkPolicy
			err := c.Get(ctx, key, &item)
			return item, err
		},
		func(c client.Client, ctx context.Context, key client.ObjectKey) (calicov3.NetworkPolicy, error) {
			var item calicov1.NetworkPolicy
			if err := c.Get(ctx, key, &item); err != nil {
				return calicov3.NetworkPolicy{}, err
			}
			return calicov3.NetworkPolicy{
				TypeMeta:   item.TypeMeta,
				ObjectMeta: item.ObjectMeta,
				Spec:       item.Spec,
			}, nil
		},
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
		func(c client.Client, ctx context.Context, opts []client.ListOption) ([]calicov3.GlobalNetworkSet, error) {
			var list calicov1.GlobalNetworkSetList
			if err := c.List(ctx, &list, opts...); err != nil {
				return nil, err
			}
			items := make([]calicov3.GlobalNetworkSet, len(list.Items))
			for i, item := range list.Items {
				items[i] = calicov3.GlobalNetworkSet{
					TypeMeta:   item.TypeMeta,
					ObjectMeta: item.ObjectMeta,
					Spec:       item.Spec,
				}
			}
			return items, nil
		},
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
		func(c client.Client, ctx context.Context, opts []client.ListOption) ([]calicov3.NetworkSet, error) {
			var list calicov1.NetworkSetList
			if err := c.List(ctx, &list, opts...); err != nil {
				return nil, err
			}
			items := make([]calicov3.NetworkSet, len(list.Items))
			for i, item := range list.Items {
				items[i] = calicov3.NetworkSet{
					TypeMeta:   item.TypeMeta,
					ObjectMeta: item.ObjectMeta,
					Spec:       item.Spec,
				}
			}
			return items, nil
		},
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
		func(c client.Client, ctx context.Context, opts []client.ListOption) ([]calicov3.HostEndpoint, error) {
			var list calicov1.HostEndpointList
			if err := c.List(ctx, &list, opts...); err != nil {
				return nil, err
			}
			items := make([]calicov3.HostEndpoint, len(list.Items))
			for i, item := range list.Items {
				items[i] = calicov3.HostEndpoint{
					TypeMeta:   item.TypeMeta,
					ObjectMeta: item.ObjectMeta,
					Spec:       item.Spec,
				}
			}
			return items, nil
		},
		listOptions{},
	)
}

// GetHostEndpoint gets a specific host endpoint by name
func (s *Service) GetHostEndpoint(clusterName, name string) (*calicov3.HostEndpoint, error) {
	return getWithFallback(
		s, clusterName, "HostEndpoint",
		func(c client.Client, ctx context.Context, key client.ObjectKey) (calicov3.HostEndpoint, error) {
			var item calicov3.HostEndpoint
			err := c.Get(ctx, key, &item)
			return item, err
		},
		func(c client.Client, ctx context.Context, key client.ObjectKey) (calicov3.HostEndpoint, error) {
			var item calicov1.HostEndpoint
			if err := c.Get(ctx, key, &item); err != nil {
				return calicov3.HostEndpoint{}, err
			}
			return calicov3.HostEndpoint{
				TypeMeta:   item.TypeMeta,
				ObjectMeta: item.ObjectMeta,
				Spec:       item.Spec,
			}, nil
		},
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
		func(c client.Client, ctx context.Context, opts []client.ListOption) ([]calicov3.FelixConfiguration, error) {
			var list calicov1.FelixConfigurationList
			if err := c.List(ctx, &list, opts...); err != nil {
				return nil, err
			}
			items := make([]calicov3.FelixConfiguration, len(list.Items))
			for i, item := range list.Items {
				items[i] = calicov3.FelixConfiguration{
					TypeMeta:   item.TypeMeta,
					ObjectMeta: item.ObjectMeta,
					Spec:       item.Spec,
				}
			}
			return items, nil
		},
		listOptions{},
	)
}

// GetFelixConfiguration gets a specific Felix configuration by name
func (s *Service) GetFelixConfiguration(clusterName, name string) (*calicov3.FelixConfiguration, error) {
	return getWithFallback(
		s, clusterName, "FelixConfiguration",
		func(c client.Client, ctx context.Context, key client.ObjectKey) (calicov3.FelixConfiguration, error) {
			var item calicov3.FelixConfiguration
			err := c.Get(ctx, key, &item)
			return item, err
		},
		func(c client.Client, ctx context.Context, key client.ObjectKey) (calicov3.FelixConfiguration, error) {
			var item calicov1.FelixConfiguration
			if err := c.Get(ctx, key, &item); err != nil {
				return calicov3.FelixConfiguration{}, err
			}
			return calicov3.FelixConfiguration{
				TypeMeta:   item.TypeMeta,
				ObjectMeta: item.ObjectMeta,
				Spec:       item.Spec,
			}, nil
		},
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
			err := c.Get(ctx, key, &item)
			return item, err
		},
		func(c client.Client, ctx context.Context, key client.ObjectKey) (calicov3.ClusterInformation, error) {
			var item calicov1.ClusterInformation
			if err := c.Get(ctx, key, &item); err != nil {
				return calicov3.ClusterInformation{}, err
			}
			return calicov3.ClusterInformation{
				TypeMeta:   item.TypeMeta,
				ObjectMeta: item.ObjectMeta,
				Spec:       item.Spec,
			}, nil
		},
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
		func(c client.Client, ctx context.Context, opts []client.ListOption) ([]calicov3.KubeControllersConfiguration, error) {
			var list calicov1.KubeControllersConfigurationList
			if err := c.List(ctx, &list, opts...); err != nil {
				return nil, err
			}
			items := make([]calicov3.KubeControllersConfiguration, len(list.Items))
			for i, item := range list.Items {
				items[i] = calicov3.KubeControllersConfiguration{
					TypeMeta:   item.TypeMeta,
					ObjectMeta: item.ObjectMeta,
					Spec:       item.Spec,
					Status:     item.Status,
				}
			}
			return items, nil
		},
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
		func(c client.Client, ctx context.Context, opts []client.ListOption) ([]calicov3.Tier, error) {
			var list calicov1.TierList
			if err := c.List(ctx, &list, opts...); err != nil {
				return nil, err
			}
			items := make([]calicov3.Tier, len(list.Items))
			for i, item := range list.Items {
				items[i] = calicov3.Tier{
					TypeMeta:   item.TypeMeta,
					ObjectMeta: item.ObjectMeta,
					Spec:       item.Spec,
				}
			}
			return items, nil
		},
		listOptions{},
	)
}

// GetTier gets a specific tier by name
func (s *Service) GetTier(clusterName, name string) (*calicov3.Tier, error) {
	return getWithFallback(
		s, clusterName, "Tier",
		func(c client.Client, ctx context.Context, key client.ObjectKey) (calicov3.Tier, error) {
			var item calicov3.Tier
			err := c.Get(ctx, key, &item)
			return item, err
		},
		func(c client.Client, ctx context.Context, key client.ObjectKey) (calicov3.Tier, error) {
			var item calicov1.Tier
			if err := c.Get(ctx, key, &item); err != nil {
				return calicov3.Tier{}, err
			}
			return calicov3.Tier{
				TypeMeta:   item.TypeMeta,
				ObjectMeta: item.ObjectMeta,
				Spec:       item.Spec,
			}, nil
		},
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
		func(c client.Client, ctx context.Context, opts []client.ListOption) ([]calicov3.CalicoNodeStatus, error) {
			var list calicov1.CalicoNodeStatusList
			if err := c.List(ctx, &list, opts...); err != nil {
				return nil, err
			}
			items := make([]calicov3.CalicoNodeStatus, len(list.Items))
			for i, item := range list.Items {
				items[i] = calicov3.CalicoNodeStatus{
					TypeMeta:   item.TypeMeta,
					ObjectMeta: item.ObjectMeta,
					Spec:       item.Spec,
					Status:     item.Status,
				}
			}
			return items, nil
		},
		listOptions{},
	)
}

// GetCalicoNodeStatus gets a specific Calico node status by name
func (s *Service) GetCalicoNodeStatus(clusterName, name string) (*calicov3.CalicoNodeStatus, error) {
	return getWithFallback(
		s, clusterName, "CalicoNodeStatus",
		func(c client.Client, ctx context.Context, key client.ObjectKey) (calicov3.CalicoNodeStatus, error) {
			var item calicov3.CalicoNodeStatus
			err := c.Get(ctx, key, &item)
			return item, err
		},
		func(c client.Client, ctx context.Context, key client.ObjectKey) (calicov3.CalicoNodeStatus, error) {
			var item calicov1.CalicoNodeStatus
			if err := c.Get(ctx, key, &item); err != nil {
				return calicov3.CalicoNodeStatus{}, err
			}
			return calicov3.CalicoNodeStatus{
				TypeMeta:   item.TypeMeta,
				ObjectMeta: item.ObjectMeta,
				Spec:       item.Spec,
				Status:     item.Status,
			}, nil
		},
		getOptions{name: name},
	)
}

// ========== Node Operations ==========

// ListNodes lists all Nodes in the specified cluster
// It attempts to list Calico Nodes first. If not found, it falls back to K8s Nodes.
// ListNodes lists all Nodes in the specified cluster
// It attempts to list Calico Nodes first. If not found, it falls back to K8s Nodes.
func (s *Service) ListNodes(clusterName string) ([]calicov1.Node, error) {
	// Try Calico Nodes
	nodes, err := listWithFallback(
		s, clusterName, "Node",
		func(c client.Client, ctx context.Context, opts []client.ListOption) ([]calicov1.Node, error) {
			var list calicov3custom.V3NodeList
			if err := c.List(ctx, &list, opts...); err != nil {
				return nil, err
			}
			// Convert calicov3custom.V3Node to calicov1.Node
			items := make([]calicov1.Node, len(list.Items))
			for i, item := range list.Items {
				items[i] = convertV3NodeToV1Node(item)
			}
			return items, nil
		},
		func(c client.Client, ctx context.Context, opts []client.ListOption) ([]calicov1.Node, error) {
			var list calicov1.NodeList
			if err := c.List(ctx, &list, opts...); err != nil {
				return nil, err
			}
			return list.Items, nil
		},
		listOptions{},
	)

	if err == nil {
		return nodes, nil
	}

	// If failed, check if it's because CRD is missing
	if isNoMatchError(err) || strings.Contains(err.Error(), "could not find the requested resource") {
		// Fallback to K8s Nodes
		logger.S().Infow("Calico Node CRD not found, falling back to K8s Nodes", "cluster", clusterName)
		return s.listK8sNodes(clusterName)
	}

	return nil, err
}

// GetNode gets a specific Node by name
func (s *Service) GetNode(clusterName, name string) (*calicov1.Node, error) {
	node, err := getWithFallback(
		s, clusterName, "Node",
		func(c client.Client, ctx context.Context, key client.ObjectKey) (calicov1.Node, error) {
			var item calicov3custom.V3Node
			if err := c.Get(ctx, key, &item); err != nil {
				return calicov1.Node{}, err
			}
			return convertV3NodeToV1Node(item), nil
		},
		func(c client.Client, ctx context.Context, key client.ObjectKey) (calicov1.Node, error) {
			var item calicov1.Node
			err := c.Get(ctx, key, &item)
			return item, err
		},
		getOptions{name: name},
	)

	if err == nil {
		return node, nil
	}

	if isNoMatchError(err) || strings.Contains(err.Error(), "could not find the requested resource") {
		return s.getK8sNode(clusterName, name)
	}

	return nil, err
}

func (s *Service) listK8sNodes(clusterName string) ([]calicov1.Node, error) {
	k8sClient, err := s.getClient(clusterName)
	if err != nil {
		return nil, err
	}

	ctx, cancel := defaultContext()
	defer cancel()

	var list corev1.NodeList
	if err := k8sClient.List(ctx, &list); err != nil {
		return nil, err
	}

	res := make([]calicov1.Node, len(list.Items))
	for i, n := range list.Items {
		res[i] = convertCoreNodeToNode(&n)
	}
	return res, nil
}

func (s *Service) getK8sNode(clusterName, name string) (*calicov1.Node, error) {
	k8sClient, err := s.getClient(clusterName)
	if err != nil {
		return nil, err
	}

	ctx, cancel := defaultContext()
	defer cancel()

	var node corev1.Node
	if err := k8sClient.Get(ctx, client.ObjectKey{Name: name}, &node); err != nil {
		return nil, err
	}

	res := convertCoreNodeToNode(&node)
	return &res, nil
}

func convertCoreNodeToNode(n *corev1.Node) calicov1.Node {
	// Map K8s Node to Calico Node struct (minimal fields)
	return calicov1.Node{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Node",
			APIVersion: "projectcalico.org/v3", // Fake it
		},
		ObjectMeta: n.ObjectMeta,
		// Spec is empty effectively, or we could try to map internalIP to BGP IPv4Address if needed?
		// For now leave empty to indicate "No Calico Config".
	}
}

// convertV3NodeToV1Node converts calicov3custom.V3Node to calicov1.Node
func convertV3NodeToV1Node(v3Node calicov3custom.V3Node) calicov1.Node {
	v1Node := calicov1.Node{
		TypeMeta:   v3Node.TypeMeta,
		ObjectMeta: v3Node.ObjectMeta,
	}

	// Map Spec
	if v3Node.Spec.BGP != nil {
		v1Node.Spec.BGP = &calicov1.NodeBGPSpec{}
		if v3Node.Spec.BGP.ASNumber != nil {
			v1Node.Spec.BGP.ASNumber = v3Node.Spec.BGP.ASNumber
		}
		v1Node.Spec.BGP.IPv4Address = v3Node.Spec.BGP.IPv4Address
		v1Node.Spec.BGP.IPv6Address = v3Node.Spec.BGP.IPv6Address
	}

	if len(v3Node.Spec.OrchRefs) > 0 {
		v1Node.Spec.OrchRefs = make([]calicov1.OrchRef, len(v3Node.Spec.OrchRefs))
		for i, ref := range v3Node.Spec.OrchRefs {
			v1Node.Spec.OrchRefs[i] = calicov1.OrchRef{
				NodeName:     ref.NodeName,
				Orchestrator: ref.Orchestrator,
			}
		}
	}

	return v1Node
}

// ========== BlockAffinity Operations ==========

// ListBlockAffinities lists all BlockAffinities in the specified cluster
func (s *Service) ListBlockAffinities(clusterName string) ([]calicov1.BlockAffinity, error) {
	// BlockAffinity is custom. We try V1 typed list.
	// We don't have V3 type support for this custom type usually.
	// Use listGeneric directly with V1 logic.
	return listGeneric(
		s, clusterName, "BlockAffinity",
		func(c client.Client, ctx context.Context, opts []client.ListOption) ([]calicov1.BlockAffinity, error) {
			var list calicov1.BlockAffinityList
			if err := c.List(ctx, &list, opts...); err != nil {
				return nil, err
			}
			return list.Items, nil
		},
		listOptions{},
	)
}

// ========== IPAMBlock Operations ==========

// ListIPAMBlocks lists all IPAMBlocks in the specified cluster
func (s *Service) ListIPAMBlocks(clusterName string) ([]calicov1.IPAMBlock, error) {
	return listGeneric(
		s, clusterName, "IPAMBlock",
		func(c client.Client, ctx context.Context, opts []client.ListOption) ([]calicov1.IPAMBlock, error) {
			var list calicov1.IPAMBlockList
			if err := c.List(ctx, &list, opts...); err != nil {
				return nil, err
			}
			return list.Items, nil
		},
		listOptions{},
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

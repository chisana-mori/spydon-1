package calico

import (
	"encoding/json"
	"fmt"

	"robusta-web/backend/pkg/logger"

	calicov3 "github.com/projectcalico/api/pkg/apis/projectcalico/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

// getDynamicClient 获取或创建指定集群的 dynamic client (for v1 API)
func (s *Service) getDynamicClient(clusterName string) (dynamic.Interface, error) {
	s.cacheMu.RLock()
	dc, exists := s.dynamicClients[clusterName]
	s.cacheMu.RUnlock()

	if exists {
		return dc, nil
	}

	// Get REST config from manager
	configInterface, err := s.nodeSyncManager.GetRESTConfig(clusterName)
	if err != nil {
		return nil, err
	}

	restConfig, ok := configInterface.(*rest.Config)
	if !ok {
		return nil, fmt.Errorf("无法转换为 *rest.Config 类型")
	}

	dc, err = dynamic.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("创建 dynamic client 失败: %w", err)
	}

	s.cacheMu.Lock()
	s.dynamicClients[clusterName] = dc
	s.cacheMu.Unlock()

	return dc, nil
}

// listV1 使用 dynamic client 查询 v1 API (cluster-scoped resources)
func listV1[T any](s *Service, clusterName string, gvr schema.GroupVersionResource, converter func(*unstructured.Unstructured) (T, error)) ([]T, error) {
	return listV1WithNamespace(s, clusterName, "", gvr, converter)
}

// listV1WithNamespace 使用 dynamic client 查询 v1 API (支持 namespace)
func listV1WithNamespace[T any](s *Service, clusterName, namespace string, gvr schema.GroupVersionResource, converter func(*unstructured.Unstructured) (T, error)) ([]T, error) {
	dc, err := s.getDynamicClient(clusterName)
	if err != nil {
		return nil, err
	}

	ctx, cancel := defaultContext()
	defer cancel()

	var list *unstructured.UnstructuredList
	if namespace != "" {
		list, err = dc.Resource(gvr).Namespace(namespace).List(ctx, metav1.ListOptions{})
	} else {
		list, err = dc.Resource(gvr).List(ctx, metav1.ListOptions{})
	}

	if err != nil {
		return nil, fmt.Errorf("获取资源列表失败 (v1 API): %w", err)
	}

	var results []T
	for _, item := range list.Items {
		converted, err := converter(&item)
		if err != nil {
			logger.S().Warnw("转换资源失败", "name", item.GetName(), "error", err)
			continue
		}
		results = append(results, converted)
	}

	return results, nil
}

// listV1Namespaced 使用 dynamic client 查询 v1 API (namespaced resources)
func listV1Namespaced[T any](s *Service, clusterName, namespace string, gvr schema.GroupVersionResource, converter func(*unstructured.Unstructured) (T, error)) ([]T, error) {
	dc, err := s.getDynamicClient(clusterName)
	if err != nil {
		return nil, err
	}

	ctx, cancel := defaultContext()
	defer cancel()

	var list *unstructured.UnstructuredList
	if namespace != "" {
		list, err = dc.Resource(gvr).Namespace(namespace).List(ctx, metav1.ListOptions{})
	} else {
		list, err = dc.Resource(gvr).List(ctx, metav1.ListOptions{})
	}

	if err != nil {
		return nil, fmt.Errorf("获取资源列表失败 (v1 API): %w", err)
	}

	var results []T
	for _, item := range list.Items {
		converted, err := converter(&item)
		if err != nil {
			logger.S().Warnw("转换资源失败", "name", item.GetName(), "error", err)
			continue
		}
		results = append(results, converted)
	}

	return results, nil
}

// getV1 使用 dynamic client 获取单个资源 (cluster-scoped)
func getV1[T any](s *Service, clusterName, name string, gvr schema.GroupVersionResource, converter func(*unstructured.Unstructured) (T, error)) (T, error) {
	return getV1WithNamespace(s, clusterName, "", name, gvr, converter)
}

// getV1WithNamespace 使用 dynamic client 获取单个资源 (支持 namespace)
func getV1WithNamespace[T any](s *Service, clusterName, namespace, name string, gvr schema.GroupVersionResource, converter func(*unstructured.Unstructured) (T, error)) (T, error) {
	var zero T
	dc, err := s.getDynamicClient(clusterName)
	if err != nil {
		return zero, err
	}

	ctx, cancel := defaultContext()
	defer cancel()

	var u *unstructured.Unstructured
	if namespace != "" {
		u, err = dc.Resource(gvr).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	} else {
		u, err = dc.Resource(gvr).Get(ctx, name, metav1.GetOptions{})
	}

	if err != nil {
		return zero, fmt.Errorf("获取资源失败 (v1 API): %w", err)
	}

	return converter(u)
}

// getV1Namespaced 使用 dynamic client 获取单个资源 (namespaced)
func getV1Namespaced[T any](s *Service, clusterName, namespace, name string, gvr schema.GroupVersionResource, converter func(*unstructured.Unstructured) (T, error)) (T, error) {
	var zero T
	dc, err := s.getDynamicClient(clusterName)
	if err != nil {
		return zero, err
	}

	ctx, cancel := defaultContext()
	defer cancel()

	u, err := dc.Resource(gvr).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return zero, fmt.Errorf("获取资源失败 (v1 API): %w", err)
	}

	return converter(u)
}

// ========== 转换函数 ==========

func convertUnstructured(u *unstructured.Unstructured, target interface{}) error {
	data, err := json.Marshal(u.Object)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}

func convertToIPPool(u *unstructured.Unstructured) (calicov3.IPPool, error) {
	var pool calicov3.IPPool
	if err := convertUnstructured(u, &pool); err != nil {
		return pool, err
	}
	return pool, nil
}

func convertToBGPPeer(u *unstructured.Unstructured) (calicov3.BGPPeer, error) {
	var peer calicov3.BGPPeer
	if err := convertUnstructured(u, &peer); err != nil {
		return peer, err
	}
	return peer, nil
}

func convertToBGPConfiguration(u *unstructured.Unstructured) (calicov3.BGPConfiguration, error) {
	var cfg calicov3.BGPConfiguration
	if err := convertUnstructured(u, &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func convertToGlobalNetworkPolicy(u *unstructured.Unstructured) (calicov3.GlobalNetworkPolicy, error) {
	var policy calicov3.GlobalNetworkPolicy
	if err := convertUnstructured(u, &policy); err != nil {
		return policy, err
	}
	return policy, nil
}

func convertToNetworkPolicy(u *unstructured.Unstructured) (calicov3.NetworkPolicy, error) {
	var policy calicov3.NetworkPolicy
	if err := convertUnstructured(u, &policy); err != nil {
		return policy, err
	}
	return policy, nil
}

func convertToHostEndpoint(u *unstructured.Unstructured) (calicov3.HostEndpoint, error) {
	var hep calicov3.HostEndpoint
	if err := convertUnstructured(u, &hep); err != nil {
		return hep, err
	}
	return hep, nil
}

func convertToFelixConfiguration(u *unstructured.Unstructured) (calicov3.FelixConfiguration, error) {
	var cfg calicov3.FelixConfiguration
	if err := convertUnstructured(u, &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func convertToGlobalNetworkSet(u *unstructured.Unstructured) (calicov3.GlobalNetworkSet, error) {
	var set calicov3.GlobalNetworkSet
	if err := convertUnstructured(u, &set); err != nil {
		return set, err
	}
	return set, nil
}

func convertToNetworkSet(u *unstructured.Unstructured) (calicov3.NetworkSet, error) {
	var set calicov3.NetworkSet
	if err := convertUnstructured(u, &set); err != nil {
		return set, err
	}
	return set, nil
}

func convertToClusterInformation(u *unstructured.Unstructured) (calicov3.ClusterInformation, error) {
	var info calicov3.ClusterInformation
	if err := convertUnstructured(u, &info); err != nil {
		return info, err
	}
	return info, nil
}

func convertToKubeControllersConfiguration(u *unstructured.Unstructured) (calicov3.KubeControllersConfiguration, error) {
	var cfg calicov3.KubeControllersConfiguration
	if err := convertUnstructured(u, &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func convertToTier(u *unstructured.Unstructured) (calicov3.Tier, error) {
	var tier calicov3.Tier
	if err := convertUnstructured(u, &tier); err != nil {
		return tier, err
	}
	return tier, nil
}

func convertToCalicoNodeStatus(u *unstructured.Unstructured) (calicov3.CalicoNodeStatus, error) {
	var status calicov3.CalicoNodeStatus
	if err := convertUnstructured(u, &status); err != nil {
		return status, err
	}
	return status, nil
}

func convertToIPReservation(u *unstructured.Unstructured) (calicov3.IPReservation, error) {
	var res calicov3.IPReservation
	if err := convertUnstructured(u, &res); err != nil {
		return res, err
	}
	return res, nil
}

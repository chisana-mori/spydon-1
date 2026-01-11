package calico

import "k8s.io/apimachinery/pkg/runtime/schema"

// APIVersion 表示 Calico API 版本
type APIVersion string

const (
	// APIVersionV3 represents projectcalico.org/v3 (requires Calico API Server)
	APIVersionV3 APIVersion = "v3"
	// APIVersionV1 represents crd.projectcalico.org/v1 (direct CRD access)
	APIVersionV1 APIVersion = "v1"
	// APIVersionUnknown 表示尚未检测
	APIVersionUnknown APIVersion = ""
)

// GVR definitions for crd.projectcalico.org/v1 resources
var (
	IPPoolGVR = schema.GroupVersionResource{
		Group:    "crd.projectcalico.org",
		Version:  "v1",
		Resource: "ippools",
	}

	IPReservationGVR = schema.GroupVersionResource{
		Group:    "crd.projectcalico.org",
		Version:  "v1",
		Resource: "ipreservations",
	}

	BGPConfigurationGVR = schema.GroupVersionResource{
		Group:    "crd.projectcalico.org",
		Version:  "v1",
		Resource: "bgpconfigurations",
	}

	BGPPeerGVR = schema.GroupVersionResource{
		Group:    "crd.projectcalico.org",
		Version:  "v1",
		Resource: "bgppeers",
	}

	GlobalNetworkPolicyGVR = schema.GroupVersionResource{
		Group:    "crd.projectcalico.org",
		Version:  "v1",
		Resource: "globalnetworkpolicies",
	}

	NetworkPolicyGVR = schema.GroupVersionResource{
		Group:    "crd.projectcalico.org",
		Version:  "v1",
		Resource: "networkpolicies",
	}

	GlobalNetworkSetGVR = schema.GroupVersionResource{
		Group:    "crd.projectcalico.org",
		Version:  "v1",
		Resource: "globalnetworksets",
	}

	NetworkSetGVR = schema.GroupVersionResource{
		Group:    "crd.projectcalico.org",
		Version:  "v1",
		Resource: "networksets",
	}

	HostEndpointGVR = schema.GroupVersionResource{
		Group:    "crd.projectcalico.org",
		Version:  "v1",
		Resource: "hostendpoints",
	}

	FelixConfigurationGVR = schema.GroupVersionResource{
		Group:    "crd.projectcalico.org",
		Version:  "v1",
		Resource: "felixconfigurations",
	}

	ClusterInformationGVR = schema.GroupVersionResource{
		Group:    "crd.projectcalico.org",
		Version:  "v1",
		Resource: "clusterinformations",
	}

	KubeControllersConfigurationGVR = schema.GroupVersionResource{
		Group:    "crd.projectcalico.org",
		Version:  "v1",
		Resource: "kubecontrollersconfigurations",
	}

	TierGVR = schema.GroupVersionResource{
		Group:    "crd.projectcalico.org",
		Version:  "v1",
		Resource: "tiers",
	}

	CalicoNodeStatusGVR = schema.GroupVersionResource{
		Group:    "crd.projectcalico.org",
		Version:  "v1",
		Resource: "caliconodestatuses",
	}
)

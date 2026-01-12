package http

import (
	"robusta-web/backend/internal/pkg/calico"

	calicov3 "github.com/projectcalico/api/pkg/apis/projectcalico/v3"
)

// OverviewStats represents aggregated statistics across all Calico clusters.
type OverviewStats struct {
	TotalIPPools    int `json:"total_ip_pools"`
	ActiveBGPPeers  int `json:"active_bgp_peers"`
	TotalBGPPeers   int `json:"total_bgp_peers"`
	TotalPolicies   int `json:"total_policies"`
	HealthyClusters int `json:"healthy_clusters"`
	TotalClusters   int `json:"total_clusters"`
	TotalIPv4Pools  int `json:"total_ipv4_pools"`
	TotalIPv6Pools  int `json:"total_ipv6_pools"`
}

// ClusterOverviewItem represents summary information for a single Calico cluster.
type ClusterOverviewItem struct {
	ClusterName    string   `json:"cluster_name"`
	IPv4PoolCount  int      `json:"ipv4_pool_count"`
	IPv6PoolCount  int      `json:"ipv6_pool_count"`
	ActiveBGPPeers int      `json:"active_bgp_peers"`
	TotalBGPPeers  int      `json:"total_bgp_peers"`
	PolicyCount    int      `json:"policy_count"`
	IsHealthy      bool     `json:"is_healthy"`
	HealthScore    int      `json:"health_score"`
	Deductions     []string `json:"deductions,omitempty"`
	ErrMsg         string   `json:"err_msg,omitempty"`
}

// OverviewResponse contains statistics and cluster list for the overview endpoint.
type OverviewResponse struct {
	Stats    OverviewStats         `json:"stats"`
	Clusters []ClusterOverviewItem `json:"clusters"`
}

// IPPoolDetail represents detailed information about a Calico IP pool.
type IPPoolDetail struct {
	Name           string  `json:"name"`
	CIDR           string  `json:"cidr"`
	IPVersion      int     `json:"ip_version"`
	BlockSize      int     `json:"block_size"`
	Allocated      int64   `json:"allocated"`
	Capacity       int64   `json:"capacity"`
	AllocationRate float64 `json:"allocation_rate"`
	NATOutgoing    bool    `json:"nat_outgoing"`
	IPIPMode       string  `json:"ipip_mode"`
	VXLANMode      string  `json:"vxlan_mode"`
	Disabled       bool    `json:"disabled"`
	WayneEnabled   string  `json:"wayne_enabled"` // true/false/-
	Subfunction    string  `json:"subfunction"`   // Translated subfunction name
}

// BGPPeerDetail represents detailed information about a BGP peer.
type BGPPeerDetail struct {
	Name         string `json:"name"`
	PeerIP       string `json:"peer_ip"`
	ASNumber     string `json:"as_number"`
	State        string `json:"state"`
	NodeSelector string `json:"node_selector"`
	Scope        string `json:"scope"`
	Uptime       string `json:"uptime"`
}

// PolicySummary represents summary information for a network policy.
type PolicySummary struct {
	Name        string   `json:"name"`
	Namespace   string   `json:"namespace"`
	Types       []string `json:"types"`
	IngressRule int      `json:"ingress_rule_count"`
	EgressRule  int      `json:"egress_rule_count"`
}

// ClusterDetailResponse contains detailed information about a single Calico cluster.
type ClusterDetailResponse struct {
	ClusterName            string                       `json:"cluster_name"`
	HealthStatus           string                       `json:"health_status"`
	Issues                 []string                     `json:"issues"`
	ScoreDetail            *calico.HealthScoreDetail    `json:"score_detail,omitempty"`
	IPPoolsV4              []IPPoolDetail               `json:"ip_pools_v4"`
	IPPoolsV6              []IPPoolDetail               `json:"ip_pools_v6"`
	BGPConfiguration       *calicov3.BGPConfiguration   `json:"bgp_configuration"`
	BGPPeers               []BGPPeerDetail              `json:"bgp_peers"`
	GlobalNetworkPolicies  []PolicySummary              `json:"global_network_policies"`
	NamespacedPolicyCounts map[string]int               `json:"namespaced_policy_counts"`
	FelixConfiguration     *calicov3.FelixConfiguration `json:"felix_configuration"`
	HostEndpointCount      int                          `json:"host_endpoint_count"`
	NetworkSetCount        int                          `json:"network_set_count"`
	GlobalNetworkSetCount  int                          `json:"global_network_set_count"`
}

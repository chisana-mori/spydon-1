
// Calico Types

export interface OverviewStats {
    total_ip_pools: number
    active_bgp_peers: number
    total_bgp_peers: number
    total_policies: number
    healthy_clusters: number
    total_clusters: number
    total_ipv4_pools: number
    total_ipv6_pools: number
}

export interface ClusterOverviewItem {
    cluster_name: string
    ipv4_pool_count: number
    ipv6_pool_count: number
    active_bgp_peers: number
    total_bgp_peers: number
    policy_count: number
    is_healthy: boolean
    health_score: number
    err_msg?: string
}

export interface OverviewResponse {
    stats: OverviewStats
    clusters: ClusterOverviewItem[]
}

export interface IPPoolDetail {
    name: string
    cidr: string
    ip_version: number
    block_size: number
    allocated: number
    capacity: number
    allocation_rate: number
    nat_outgoing: boolean
    ipip_mode: string
    vxlan_mode: string
    disabled: boolean
    wayne_enabled: string
    subfunction: string
}

export interface BGPPeerDetail {
    name: string
    peer_ip: string
    as_number: string
    state: string
    node_selector: string
    scope: string
    uptime: string
}

export interface PolicySummary {
    name: string
    namespace: string
    types: string[]
    ingress_rule_count: number
    egress_rule_count: number
}

// Partial mapping from Calico API structure
export interface BGPConfiguration {
    metadata: {
        name: string
        creationTimestamp?: string
    }
    spec: {
        logSeverityScreen?: string
        nodeToNodeMeshEnabled?: boolean
        asNumber?: number
        serviceClusterIPs?: { cidr: string }[]
        serviceExternalIPs?: { cidr: string }[]
        // Add other fields as needed
    }
}

export interface FelixConfiguration {
    metadata: {
        name: string
    }
    spec: {
        ipipEnabled?: boolean
        vxlanEnabled?: boolean
        wireguardEnabled?: boolean
        logSeverityScreen?: string
        prometheusMetricsEnabled?: boolean
        // Add others
    }
}

export interface ClusterDetailResponse {
    cluster_name: string
    health_status: 'healthy' | 'warning' | 'critical'
    issues: string[]
    ip_pools_v4: IPPoolDetail[]
    ip_pools_v6: IPPoolDetail[]
    bgp_configuration: BGPConfiguration | null
    bgp_peers: BGPPeerDetail[]
    global_network_policies: PolicySummary[]
    namespaced_policy_counts: Record<string, number>
    felix_configuration: FelixConfiguration | null
    host_endpoint_count: number
    network_set_count: number
    global_network_set_count: number
}

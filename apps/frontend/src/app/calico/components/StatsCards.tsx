
import { Card, CardContent } from "@/components/ui/card"
import { OverviewStats } from "@/types/calico"
import { Activity, Network, Server, Shield, AlertCircle, CheckCircle } from "lucide-react"

interface StatsCardsProps {
    stats?: OverviewStats
    isLoading?: boolean
}

export function StatsCards({ stats, isLoading }: StatsCardsProps) {
    if (isLoading || !stats) {
        return (
            <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
                {[1, 2, 3, 4].map((i) => (
                    <Card key={i} className="bg-card/50">
                        <CardContent className="p-6">
                            <div className="h-4 w-24 bg-muted animate-pulse rounded mb-2" />
                            <div className="h-8 w-16 bg-muted animate-pulse rounded" />
                        </CardContent>
                    </Card>
                ))}
            </div>
        )
    }

    const {
        total_ip_pools,
        total_ipv4_pools,
        total_ipv6_pools,
        active_bgp_peers,
        total_bgp_peers,
        total_policies,
        healthy_clusters,
        total_clusters
    } = stats

    // BGP Health
    const bgpHealthy = active_bgp_peers === total_bgp_peers && total_bgp_peers > 0
    const bgpWarning = active_bgp_peers < total_bgp_peers

    // Cluster Health
    const clusterAllHealthy = healthy_clusters === total_clusters && total_clusters > 0

    return (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
            {/* IP Pools Card */}
            <Card className="bg-gradient-to-br from-blue-500/10 to-transparent border-blue-500/20 hover:shadow-lg hover:from-blue-500/20 transition-all duration-300">
                <CardContent className="p-6 relative overflow-hidden">
                    <div className="absolute right-4 top-4 bg-blue-500/10 p-3 rounded-full">
                        <Network className="h-6 w-6 text-blue-500" />
                    </div>
                    <div className="text-sm font-medium text-blue-500 mb-1">IP 地址池总数</div>
                    <div className="text-3xl font-bold text-foreground mt-2">
                        {total_ip_pools}
                        <span className="text-base font-normal text-muted-foreground ml-2">个</span>
                    </div>
                    <div className="text-xs text-muted-foreground mt-3 flex items-center gap-2">
                        <span className="bg-blue-500/10 text-blue-500 px-2 py-0.5 rounded-full font-medium">IPv4: {total_ipv4_pools}</span>
                        <span className="bg-purple-500/10 text-purple-500 px-2 py-0.5 rounded-full font-medium">IPv6: {total_ipv6_pools}</span>
                    </div>
                </CardContent>
            </Card>

            {/* BGP Peers Card */}
            <Card className={`hover:shadow-lg transition-all duration-300 ${bgpHealthy ? 'bg-gradient-to-br from-green-500/10 to-transparent border-green-500/20 hover:from-green-500/20' : 'bg-gradient-to-br from-yellow-500/10 to-transparent border-yellow-500/20 hover:from-yellow-500/20'}`}>
                <CardContent className="p-6 relative overflow-hidden">
                    <div className={`absolute right-4 top-4 p-3 rounded-full ${bgpHealthy ? 'bg-green-500/10' : 'bg-yellow-500/10'}`}>
                        <Activity className={`h-6 w-6 ${bgpHealthy ? 'text-green-500' : 'text-yellow-500'}`} />
                    </div>
                    <div className={`text-sm font-medium ${bgpHealthy ? 'text-green-500' : 'text-yellow-500'} mb-1`}>BGP 对等体</div>

                    {bgpHealthy ? (
                        <div className="mt-2">
                            <div className="text-3xl font-bold text-foreground">
                                {active_bgp_peers}/{total_bgp_peers}
                            </div>
                            <div className="text-xs text-green-600 mt-3 font-medium flex items-center gap-1">
                                <CheckCircle className="h-3 w-3" />
                                所有对等体连接正常
                            </div>
                        </div>
                    ) : (
                        <div className="mt-2">
                            <div className="flex items-baseline gap-2">
                                <span className="text-3xl font-bold text-foreground">{active_bgp_peers}</span>
                                <span className="text-sm text-yellow-500">/ {total_bgp_peers}</span>
                            </div>
                            <div className="text-xs text-red-500 mt-3 font-medium flex items-center gap-1">
                                <AlertCircle className="h-3 w-3" />
                                {total_bgp_peers - active_bgp_peers} 个对等体异常
                            </div>
                        </div>
                    )}
                </CardContent>
            </Card>

            {/* Policies Card */}
            <Card className="bg-gradient-to-br from-purple-500/10 to-transparent border-purple-500/20 hover:shadow-lg hover:from-purple-500/20 transition-all duration-300">
                <CardContent className="p-6 relative overflow-hidden">
                    <div className="absolute right-4 top-4 bg-purple-500/10 p-3 rounded-full">
                        <Shield className="h-6 w-6 text-purple-500" />
                    </div>
                    <div className="text-sm font-medium text-purple-500 mb-1">网络策略</div>
                    <div className="text-3xl font-bold text-foreground mt-2">
                        {total_policies}
                        <span className="text-base font-normal text-muted-foreground ml-2">生效中</span>
                    </div>
                    <div className="text-xs text-muted-foreground mt-3">
                        覆盖所有命名空间
                    </div>
                </CardContent>
            </Card>

            {/* Cluster Health Card */}
            <Card className={`hover:shadow-lg transition-all duration-300 ${clusterAllHealthy ? 'bg-gradient-to-br from-teal-500/10 to-transparent border-teal-500/20 hover:from-teal-500/20' : 'bg-gradient-to-br from-red-500/10 to-transparent border-red-500/20 hover:from-red-500/20'}`}>
                <CardContent className="p-6 relative overflow-hidden">
                    <div className={`absolute right-4 top-4 p-3 rounded-full ${clusterAllHealthy ? 'bg-teal-500/10' : 'bg-red-500/10'}`}>
                        <Server className={`h-6 w-6 ${clusterAllHealthy ? 'text-teal-500' : 'text-red-500'}`} />
                    </div>
                    <div className={`text-sm font-medium ${clusterAllHealthy ? 'text-teal-500' : 'text-red-500'} mb-1`}>集群健康状态</div>

                    {clusterAllHealthy ? (
                        <div className="mt-2">
                            <div className="text-3xl font-bold text-foreground">
                                {healthy_clusters}/{total_clusters}
                            </div>
                            <div className="text-xs text-teal-600 mt-3 font-medium flex items-center gap-1">
                                <CheckCircle className="h-3 w-3" />
                                所有集群运行正常
                            </div>
                        </div>
                    ) : (
                        <div className="mt-2">
                            <div className="flex items-baseline gap-2">
                                <span className="text-3xl font-bold text-foreground">{healthy_clusters}</span>
                                <span className="text-sm text-muted-foreground">/ {total_clusters} 正常</span>
                            </div>
                            <div className="text-xs text-red-500 mt-3 font-medium flex items-center gap-1">
                                <AlertCircle className="h-3 w-3" />
                                {total_clusters - healthy_clusters} 个集群存在异常
                            </div>
                        </div>
                    )}
                </CardContent>
            </Card>
        </div>
    )
}

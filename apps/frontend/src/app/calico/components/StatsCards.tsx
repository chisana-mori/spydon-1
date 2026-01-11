
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
                    <Card key={i} className="relative overflow-hidden border-border/50 bg-card/40 backdrop-blur-sm">
                        <div className="absolute inset-0 bg-gradient-to-br from-muted/20 to-transparent opacity-50" />
                        <CardContent className="p-6 relative">
                            <div className="h-4 w-28 bg-muted/60 animate-pulse rounded mb-3" />
                            <div className="h-9 w-20 bg-muted/60 animate-pulse rounded mb-2" />
                            <div className="h-3 w-16 bg-muted/40 animate-pulse rounded" />
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
            <Card className="group relative overflow-hidden border-blue-500/20 bg-gradient-to-br from-blue-500/5 via-blue-500/[0.02] to-transparent hover:shadow-lg hover:shadow-blue-500/10 hover:from-blue-500/10 transition-all duration-300">
                {/* Decorative glow */}
                <div className="absolute -right-8 -top-8 h-24 w-24 rounded-full bg-blue-500/10 blur-2xl group-hover:bg-blue-500/20 transition-all duration-500" />
                <CardContent className="p-6 relative">
                    <div className="flex items-start justify-between mb-4">
                        <div className="space-y-1">
                            <div className="text-xs font-medium uppercase tracking-wider text-blue-500/80">IP 地址池</div>
                            <div className="h-px w-8 bg-gradient-to-r from-blue-500/50 to-transparent" />
                        </div>
                        <div className="inline-flex items-center justify-center rounded-xl bg-blue-500/10 p-2.5 ring-1 ring-blue-500/20">
                            <Network className="h-5 w-5 text-blue-500" />
                        </div>
                    </div>
                    <div className="space-y-3">
                        <div className="flex items-baseline gap-2">
                            <span className="text-4xl font-bold tracking-tight text-foreground tabular-nums">{total_ip_pools}</span>
                            <span className="text-sm text-muted-foreground">总池数</span>
                        </div>
                        <div className="flex items-center gap-2">
                            <span className="inline-flex items-center rounded-full bg-blue-500/10 px-2.5 py-1 text-xs font-medium text-blue-600 dark:text-blue-400 ring-1 ring-blue-500/20">
                                IPv4: {total_ipv4_pools}
                            </span>
                            <span className="inline-flex items-center rounded-full bg-purple-500/10 px-2.5 py-1 text-xs font-medium text-purple-600 dark:text-purple-400 ring-1 ring-purple-500/20">
                                IPv6: {total_ipv6_pools}
                            </span>
                        </div>
                    </div>
                </CardContent>
            </Card>

            {/* BGP Peers Card */}
            <Card className={`group relative overflow-hidden hover:shadow-lg transition-all duration-300 ${
                bgpHealthy
                    ? 'border-green-500/20 bg-gradient-to-br from-green-500/5 via-green-500/[0.02] to-transparent hover:shadow-green-500/10 hover:from-green-500/10'
                    : 'border-amber-500/20 bg-gradient-to-br from-amber-500/5 via-amber-500/[0.02] to-transparent hover:shadow-amber-500/10 hover:from-amber-500/10'
            }`}>
                <div className={`absolute -right-8 -top-8 h-24 w-24 rounded-full blur-2xl transition-all duration-500 ${
                    bgpHealthy ? 'bg-green-500/10 group-hover:bg-green-500/20' : 'bg-amber-500/10 group-hover:bg-amber-500/20'
                }`} />
                <CardContent className="p-6 relative">
                    <div className="flex items-start justify-between mb-4">
                        <div className="space-y-1">
                            <div className={`text-xs font-medium uppercase tracking-wider ${bgpHealthy ? 'text-green-500/80' : 'text-amber-500/80'}`}>BGP 对等体</div>
                            <div className={`h-px w-8 bg-gradient-to-r ${bgpHealthy ? 'from-green-500/50' : 'from-amber-500/50'} to-transparent`} />
                        </div>
                        <div className={`inline-flex items-center justify-center rounded-xl p-2.5 ring-1 ${
                            bgpHealthy ? 'bg-green-500/10 ring-green-500/20' : 'bg-amber-500/10 ring-amber-500/20'
                        }`}>
                            <Activity className={`h-5 w-5 ${bgpHealthy ? 'text-green-500' : 'text-amber-500'}`} />
                        </div>
                    </div>
                    <div className="space-y-3">
                        <div className="flex items-baseline gap-2">
                            <span className="text-4xl font-bold tracking-tight text-foreground tabular-nums">{active_bgp_peers}</span>
                            <span className={`text-sm ${bgpHealthy ? 'text-green-500' : 'text-amber-500'}`}>/ {total_bgp_peers}</span>
                        </div>
                        {bgpHealthy ? (
                            <div className="inline-flex items-center gap-1.5 rounded-full bg-green-500/10 px-3 py-1.5 text-xs font-medium text-green-600 dark:text-green-400 ring-1 ring-green-500/20">
                                <CheckCircle className="h-3.5 w-3.5" />
                                <span>所有对等体连接正常</span>
                            </div>
                        ) : (
                            <div className="inline-flex items-center gap-1.5 rounded-full bg-red-500/10 px-3 py-1.5 text-xs font-medium text-red-600 dark:text-red-400 ring-1 ring-red-500/20">
                                <AlertCircle className="h-3.5 w-3.5" />
                                <span>{total_bgp_peers - active_bgp_peers} 个对等体异常</span>
                            </div>
                        )}
                    </div>
                </CardContent>
            </Card>

            {/* Policies Card */}
            <Card className="group relative overflow-hidden border-purple-500/20 bg-gradient-to-br from-purple-500/5 via-purple-500/[0.02] to-transparent hover:shadow-lg hover:shadow-purple-500/10 hover:from-purple-500/10 transition-all duration-300">
                <div className="absolute -right-8 -top-8 h-24 w-24 rounded-full bg-purple-500/10 blur-2xl group-hover:bg-purple-500/20 transition-all duration-500" />
                <CardContent className="p-6 relative">
                    <div className="flex items-start justify-between mb-4">
                        <div className="space-y-1">
                            <div className="text-xs font-medium uppercase tracking-wider text-purple-500/80">网络策略</div>
                            <div className="h-px w-8 bg-gradient-to-r from-purple-500/50 to-transparent" />
                        </div>
                        <div className="inline-flex items-center justify-center rounded-xl bg-purple-500/10 p-2.5 ring-1 ring-purple-500/20">
                            <Shield className="h-5 w-5 text-purple-500" />
                        </div>
                    </div>
                    <div className="space-y-3">
                        <div className="flex items-baseline gap-2">
                            <span className="text-4xl font-bold tracking-tight text-foreground tabular-nums">{total_policies}</span>
                            <span className="text-sm text-muted-foreground">生效中</span>
                        </div>
                        <div className="text-xs text-muted-foreground">
                            覆盖所有命名空间
                        </div>
                    </div>
                </CardContent>
            </Card>

            {/* Cluster Health Card */}
            <Card className={`group relative overflow-hidden hover:shadow-lg transition-all duration-300 ${
                clusterAllHealthy
                    ? 'border-teal-500/20 bg-gradient-to-br from-teal-500/5 via-teal-500/[0.02] to-transparent hover:shadow-teal-500/10 hover:from-teal-500/10'
                    : 'border-red-500/20 bg-gradient-to-br from-red-500/5 via-red-500/[0.02] to-transparent hover:shadow-red-500/10 hover:from-red-500/10'
            }`}>
                <div className={`absolute -right-8 -top-8 h-24 w-24 rounded-full blur-2xl transition-all duration-500 ${
                    clusterAllHealthy ? 'bg-teal-500/10 group-hover:bg-teal-500/20' : 'bg-red-500/10 group-hover:bg-red-500/20'
                }`} />
                <CardContent className="p-6 relative">
                    <div className="flex items-start justify-between mb-4">
                        <div className="space-y-1">
                            <div className={`text-xs font-medium uppercase tracking-wider ${clusterAllHealthy ? 'text-teal-500/80' : 'text-red-500/80'}`}>集群健康</div>
                            <div className={`h-px w-8 bg-gradient-to-r ${clusterAllHealthy ? 'from-teal-500/50' : 'from-red-500/50'} to-transparent`} />
                        </div>
                        <div className={`inline-flex items-center justify-center rounded-xl p-2.5 ring-1 ${
                            clusterAllHealthy ? 'bg-teal-500/10 ring-teal-500/20' : 'bg-red-500/10 ring-red-500/20'
                        }`}>
                            <Server className={`h-5 w-5 ${clusterAllHealthy ? 'text-teal-500' : 'text-red-500'}`} />
                        </div>
                    </div>
                    <div className="space-y-3">
                        <div className="flex items-baseline gap-2">
                            <span className="text-4xl font-bold tracking-tight text-foreground tabular-nums">{healthy_clusters}</span>
                            <span className="text-sm text-muted-foreground">/ {total_clusters}</span>
                        </div>
                        {clusterAllHealthy ? (
                            <div className="inline-flex items-center gap-1.5 rounded-full bg-teal-500/10 px-3 py-1.5 text-xs font-medium text-teal-600 dark:text-teal-400 ring-1 ring-teal-500/20">
                                <CheckCircle className="h-3.5 w-3.5" />
                                <span>所有集群运行正常</span>
                            </div>
                        ) : (
                            <div className="inline-flex items-center gap-1.5 rounded-full bg-red-500/10 px-3 py-1.5 text-xs font-medium text-red-600 dark:text-red-400 ring-1 ring-red-500/20">
                                <AlertCircle className="h-3.5 w-3.5" />
                                <span>{total_clusters - healthy_clusters} 个集群存在异常</span>
                            </div>
                        )}
                    </div>
                </CardContent>
            </Card>
        </div>
    )
}

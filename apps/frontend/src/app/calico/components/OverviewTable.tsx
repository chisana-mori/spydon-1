
import {
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableHeader,
    TableRow,
} from "@/components/ui/table"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Progress } from "@/components/ui/progress"
import { ClusterOverviewItem } from "@/types/calico"
import { ArrowRight, CheckCircle, AlertTriangle, SearchX, RefreshCw } from "lucide-react"
import Link from 'next/link'

interface OverviewTableProps {
    clusters: ClusterOverviewItem[]
    isLoading?: boolean
}

export function OverviewTable({ clusters, isLoading }: OverviewTableProps) {
    if (isLoading) {
        return (
            <div className="rounded-md border bg-card">
                <div className="p-8 space-y-4">
                    {[1, 2, 3].map(i => (
                        <div key={i} className="flex items-center space-x-4">
                            <div className="h-12 w-12 rounded-full bg-muted animate-pulse" />
                            <div className="space-y-2 flex-1">
                                <div className="h-4 w-1/3 bg-muted animate-pulse rounded" />
                                <div className="h-3 w-1/4 bg-muted animate-pulse rounded" />
                            </div>
                        </div>
                    ))}
                </div>
            </div>
        )
    }

    return (
        <div className="rounded-xl border bg-card text-card-foreground shadow-sm overflow-hidden">
            <Table>
                <TableHeader className="bg-muted/30">
                    <TableRow>
                        <TableHead className="font-semibold">集群名称</TableHead>
                        <TableHead className="font-semibold">IP 地址池</TableHead>
                        <TableHead className="font-semibold">BGP 对等体 (活跃/总数)</TableHead>
                        <TableHead className="font-semibold">网络策略</TableHead>
                        <TableHead className="w-[200px] font-semibold">健康评分</TableHead>
                        <TableHead className="text-right font-semibold">操作</TableHead>
                    </TableRow>
                </TableHeader>
                <TableBody>
                    {clusters.length === 0 ? (
                        <TableRow>
                            <TableCell colSpan={6} className="h-[300px] text-center">
                                <div className="flex flex-col items-center justify-center space-y-3 py-10">
                                    <div className="flex h-16 w-16 items-center justify-center rounded-full bg-muted/30 ring-1 ring-border">
                                        <SearchX className="h-8 w-8 text-muted-foreground" />
                                    </div>
                                    <h3 className="text-lg font-medium text-foreground">未发现集群</h3>
                                    <p className="text-sm text-muted-foreground max-w-sm mx-auto text-center">
                                        暂无受管纳的 Calico 集群。请检查 NodeSync 状态或确认集群是否已接入。
                                    </p>
                                    <Button variant="outline" size="sm" className="mt-2" onClick={() => window.location.reload()}>
                                        <RefreshCw className="mr-2 h-4 w-4" />
                                        重试
                                    </Button>
                                </div>
                            </TableCell>
                        </TableRow>
                    ) : (
                        clusters.map((cluster) => (
                            <TableRow key={cluster.cluster_name} className="hover:bg-muted/5 transition-colors">
                                <TableCell className="font-medium">
                                    <div className="flex flex-col">
                                        <span className="text-base">{cluster.cluster_name}</span>
                                        {cluster.err_msg && (
                                            <span className="text-xs text-destructive truncate max-w-[150px]" title={cluster.err_msg}>
                                                {cluster.err_msg}
                                            </span>
                                        )}
                                    </div>
                                </TableCell>
                                <TableCell>
                                    <div className="flex items-center gap-2">
                                        <Badge variant="secondary" className="bg-blue-500/10 text-blue-600 dark:text-blue-400 hover:bg-blue-500/20">
                                            Total: {cluster.ipv4_pool_count + cluster.ipv6_pool_count}
                                        </Badge>
                                        <span className="text-xs text-muted-foreground tabular-nums">
                                            (v4: {cluster.ipv4_pool_count}, v6: {cluster.ipv6_pool_count})
                                        </span>
                                    </div>
                                </TableCell>
                                <TableCell>
                                    <div className="flex items-center gap-2">
                                        {cluster.active_bgp_peers === cluster.total_bgp_peers && cluster.total_bgp_peers > 0 ? (
                                            <Badge variant="outline" className="bg-green-500/10 text-green-600 border-green-500/20 gap-1 pr-3">
                                                <CheckCircle className="h-3 w-3" />
                                                {cluster.active_bgp_peers}/{cluster.total_bgp_peers}
                                            </Badge>
                                        ) : (
                                            <Badge variant="destructive" className="bg-red-500/10 text-red-600 border-red-500/20 gap-1 pr-3">
                                                <AlertTriangle className="h-3 w-3" />
                                                {cluster.active_bgp_peers}/{cluster.total_bgp_peers}
                                            </Badge>
                                        )}
                                    </div>
                                </TableCell>
                                <TableCell>
                                    <span className="font-mono text-sm">{cluster.policy_count}</span>
                                </TableCell>
                                <TableCell>
                                    <div className="space-y-1.5 pr-4">
                                        <div className="flex justify-between text-xs font-medium">
                                            <span className={cluster.health_score > 90 ? "text-green-600" : cluster.health_score > 70 ? "text-yellow-600" : "text-red-600"}>
                                                {cluster.health_score > 90 ? '极佳' : cluster.health_score > 70 ? '警告' : '危险'}
                                            </span>
                                            <span>{cluster.health_score}%</span>
                                        </div>
                                        <Progress
                                            value={cluster.health_score}
                                            className={`h-2 bg-muted/50 ${cluster.health_score > 90 ? "[&>div]:bg-green-500" :
                                                    cluster.health_score > 70 ? "[&>div]:bg-yellow-500" :
                                                        "[&>div]:bg-red-500"
                                                }`}
                                        />
                                    </div>
                                </TableCell>
                                <TableCell className="text-right">
                                    <Button asChild variant="ghost" size="sm" className="hover:bg-primary/5 hover:text-primary">
                                        <Link href={`/calico/${cluster.cluster_name}`}>
                                            详情 <ArrowRight className="ml-2 h-4 w-4" />
                                        </Link>
                                    </Button>
                                </TableCell>
                            </TableRow>
                        )))}
                </TableBody>
            </Table>
        </div>
    )
}

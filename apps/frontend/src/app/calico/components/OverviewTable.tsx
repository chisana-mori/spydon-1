
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
import {
    Tooltip,
    TooltipContent,
    TooltipProvider,
    TooltipTrigger,
} from "@/components/ui/tooltip"

interface OverviewTableProps {
    clusters: ClusterOverviewItem[]
    isLoading?: boolean
}

export function OverviewTable({ clusters, isLoading }: OverviewTableProps) {
    if (isLoading) {
        return (
            <div className="relative overflow-hidden rounded-xl border border-border/50 bg-card/40 backdrop-blur-sm">
                <div className="absolute inset-0 bg-gradient-to-br from-muted/10 to-transparent opacity-30" />
                <div className="p-8 space-y-4 relative">
                    {[1, 2, 3].map(i => (
                        <div key={i} className="flex items-center space-x-4">
                            <div className="h-12 w-12 rounded-xl bg-muted/40 animate-pulse" />
                            <div className="space-y-2 flex-1">
                                <div className="h-4 w-1/3 bg-muted/40 animate-pulse rounded" />
                                <div className="h-3 w-1/4 bg-muted/30 animate-pulse rounded" />
                            </div>
                        </div>
                    ))}
                </div>
            </div>
        )
    }

    return (
        <div className="relative overflow-hidden rounded-xl border border-border/50 bg-card/60 backdrop-blur-sm shadow-sm">
            {/* Subtle gradient overlay */}
            <div className="absolute inset-0 bg-gradient-to-br from-muted/[0.02] to-transparent pointer-events-none" />

            <Table>
                <TableHeader className="bg-muted/20">
                    <TableRow className="hover:bg-transparent border-border/50">
                        <TableHead className="font-semibold text-xs uppercase tracking-wider text-muted-foreground">集群名称</TableHead>
                        <TableHead className="font-semibold text-xs uppercase tracking-wider text-muted-foreground">IP 地址池</TableHead>
                        <TableHead className="font-semibold text-xs uppercase tracking-wider text-muted-foreground">BGP 对等体 (活跃/总数)</TableHead>
                        <TableHead className="font-semibold text-xs uppercase tracking-wider text-muted-foreground">网络策略</TableHead>
                        <TableHead className="w-[200px] font-semibold text-xs uppercase tracking-wider text-muted-foreground">健康评分</TableHead>
                        <TableHead className="text-right font-semibold text-xs uppercase tracking-wider text-muted-foreground">操作</TableHead>
                    </TableRow>
                </TableHeader>
                <TableBody>
                    {clusters.length === 0 ? (
                        <TableRow>
                            <TableCell colSpan={6} className="h-[400px]">
                                <div className="flex flex-col items-center justify-center space-y-4 py-16">
                                    <div className="relative">
                                        <div className="flex h-20 w-20 items-center justify-center rounded-2xl bg-muted/10 ring-1 ring-border/50">
                                            <SearchX className="h-10 w-10 text-muted-foreground/50" />
                                        </div>
                                        {/* Subtle glow effect */}
                                        <div className="absolute inset-0 h-20 w-20 rounded-2xl bg-muted/5 blur-xl" />
                                    </div>
                                    <div className="space-y-2 text-center">
                                        <h3 className="text-lg font-semibold text-foreground">未发现集群</h3>
                                        <p className="text-sm text-muted-foreground max-w-md mx-auto leading-relaxed">
                                            暂无受管理的 Calico 集群。请检查 NodeSync 状态或确认集群是否已正确接入。
                                        </p>
                                    </div>
                                    <Button
                                        variant="outline"
                                        size="default"
                                        className="mt-4 shadow-sm hover:shadow hover:bg-primary/5 transition-all duration-200"
                                        onClick={() => window.location.reload()}
                                    >
                                        <RefreshCw className="mr-2 h-4 w-4" />
                                        重试
                                    </Button>
                                </div>
                            </TableCell>
                        </TableRow>
                    ) : (
                        clusters.map((cluster) => (
                            <TableRow
                                key={cluster.cluster_name}
                                className="hover:bg-muted/[0.03] transition-colors border-border/30"
                            >
                                <TableCell className="font-medium">
                                    <div className="flex flex-col">
                                        <span className="text-sm font-medium">{cluster.cluster_name}</span>
                                        {cluster.err_msg && (
                                            <span className="text-xs text-destructive truncate max-w-[180px] mt-1" title={cluster.err_msg}>
                                                {cluster.err_msg}
                                            </span>
                                        )}
                                    </div>
                                </TableCell>
                                <TableCell>
                                    <div className="flex items-center gap-2">
                                        <Badge
                                            variant="secondary"
                                            className="bg-blue-500/10 text-blue-600 dark:text-blue-400 hover:bg-blue-500/20 border-blue-500/20 font-normal"
                                        >
                                            Total: {cluster.ipv4_pool_count + cluster.ipv6_pool_count}
                                        </Badge>
                                        <span className="text-[11px] text-muted-foreground tabular-nums">
                                            (v4: {cluster.ipv4_pool_count}, v6: {cluster.ipv6_pool_count})
                                        </span>
                                    </div>
                                </TableCell>
                                <TableCell>
                                    <div className="flex items-center gap-2">
                                        {cluster.active_bgp_peers === cluster.total_bgp_peers && cluster.total_bgp_peers > 0 ? (
                                            <Badge className="bg-green-500/10 text-green-600 dark:text-green-400 border-green-500/20 gap-1.5 pr-3 font-normal">
                                                <CheckCircle className="h-3.5 w-3.5" />
                                                {cluster.active_bgp_peers}/{cluster.total_bgp_peers}
                                            </Badge>
                                        ) : (
                                            <Badge className="bg-red-500/10 text-red-600 dark:text-red-400 border-red-500/20 gap-1.5 pr-3 font-normal">
                                                <AlertTriangle className="h-3.5 w-3.5" />
                                                {cluster.active_bgp_peers}/{cluster.total_bgp_peers}
                                            </Badge>
                                        )}
                                    </div>
                                </TableCell>
                                <TableCell>
                                    <span className="font-mono text-sm font-medium">{cluster.policy_count}</span>
                                </TableCell>
                                <TableCell>
                                    <div className="space-y-2 pr-4">
                                        <TooltipProvider>
                                            <Tooltip delayDuration={0}>
                                                <TooltipTrigger asChild>
                                                    <div className="cursor-help">
                                                        <div className="flex justify-between text-xs font-medium mb-1.5">
                                                            <span className={
                                                                cluster.health_score > 90
                                                                    ? "text-green-600 dark:text-green-400"
                                                                    : cluster.health_score > 70
                                                                        ? "text-yellow-600 dark:text-yellow-400"
                                                                        : "text-red-600 dark:text-red-400"
                                                            }>
                                                                {cluster.health_score > 90 ? '极佳' : cluster.health_score > 70 ? '警告' : '危险'}
                                                            </span>
                                                            <span className="tabular-nums">{cluster.health_score}%</span>
                                                        </div>
                                                        <Progress
                                                            value={cluster.health_score}
                                                            className={`h-2 bg-muted/50 ${cluster.health_score > 90 ? "[&>div]:bg-green-500" :
                                                                cluster.health_score > 70 ? "[&>div]:bg-yellow-500" :
                                                                    "[&>div]:bg-red-500"
                                                                }`}
                                                        />
                                                    </div>
                                                </TooltipTrigger>
                                                {cluster.deductions && cluster.deductions.length > 0 ? (
                                                    <TooltipContent side="left" className="max-w-[320px] p-4 bg-popover/95 backdrop-blur-sm border-border/50 shadow-xl supports-[backdrop-filter]:bg-popover/80">
                                                        <div className="space-y-2.5">
                                                            <div className="flex items-center justify-between pb-2 border-b border-border/50">
                                                                <span className="font-semibold text-xs text-foreground">健康评分详情</span>
                                                                <span className="text-[10px] text-muted-foreground bg-muted/50 px-1.5 py-0.5 rounded">
                                                                    -{100 - cluster.health_score}分
                                                                </span>
                                                            </div>
                                                            <ul className="space-y-2">
                                                                {cluster.deductions.map((d, i) => (
                                                                    <li key={i} className="text-xs text-muted-foreground flex items-start gap-2 leading-relaxed">
                                                                        <span className="text-red-500 shrink-0 mt-0.5">•</span>
                                                                        <span>{d}</span>
                                                                    </li>
                                                                ))}
                                                            </ul>
                                                        </div>
                                                    </TooltipContent>
                                                ) : (
                                                    <TooltipContent>
                                                        <p className="text-xs">无扣分项，状态完美</p>
                                                    </TooltipContent>
                                                )}
                                            </Tooltip>
                                        </TooltipProvider>
                                    </div>
                                </TableCell>
                                <TableCell className="text-right">
                                    <Button
                                        asChild
                                        variant="ghost"
                                        size="sm"
                                        className="hover:bg-primary/5 hover:text-primary group transition-all duration-200"
                                    >
                                        <Link href={`/calico/${cluster.cluster_name}`}>
                                            详情
                                            <ArrowRight className="ml-2 h-4 w-4 group-hover:translate-x-0.5 transition-transform duration-200" />
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

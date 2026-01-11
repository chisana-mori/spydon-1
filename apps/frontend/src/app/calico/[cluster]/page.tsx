
"use client"

import { useEffect, useState } from "react"
import { RobustaAPI } from "@/lib/api"
import { ClusterDetailResponse } from "@/types/calico"
import { useParams, useRouter } from "next/navigation"
import { Button } from "@/components/ui/button"
import {
    ArrowLeft,
    CheckCircle2,
    AlertTriangle,
    RotateCw,
    Globe,
    Layers,
    Monitor,
    Share2,
    Network,
    Shield
} from "lucide-react"
import { IPPoolTable } from "../components/IPPoolTable"
import { BGPPeerList } from "../components/BGPPeerList"
import { BGPConfigCard, FelixConfigCard, NetworkPolicySummary } from "../components/InfoCards"
import { Card, CardContent } from "@/components/ui/card"
import { cn } from "@/lib/utils"

export default function ClusterCalicoDetailPage() {
    const params = useParams()
    const router = useRouter()
    const clusterName = params.cluster as string

    const [data, setData] = useState<ClusterDetailResponse | null>(null)
    const [isLoading, setIsLoading] = useState(true)
    const [isRefreshing, setIsRefreshing] = useState(false)
    const [isSyncing, setIsSyncing] = useState(false)
    const [syncResult, setSyncResult] = useState<{ created: number; updated: number; deleted: number } | null>(null)

    const fetchData = async () => {
        setIsRefreshing(true)
        try {
            const res = await RobustaAPI.getCalicoClusterDetail(clusterName)
            setData(res)
        } catch (error) {
            console.error("Failed to fetch cluster detail", error)
        } finally {
            setIsLoading(false)
            setIsRefreshing(false)
        }
    }

    const handleSyncWayne = async () => {
        setIsSyncing(true)
        setSyncResult(null)
        try {
            const result = await RobustaAPI.syncIPPoolsToWayne(clusterName)
            setSyncResult(result)
        } catch (error) {
            console.error("Failed to sync to Wayne", error)
        } finally {
            setIsSyncing(false)
        }
    }

    useEffect(() => {
        if (clusterName) {
            fetchData()
        }
    }, [clusterName])

    if (isLoading) {
        return (
            <div className="space-y-6">
                <div className="rounded-xl border border-border/50 bg-card p-6">
                    <div className="flex items-center justify-between">
                        <div className="space-y-2">
                            <div className="h-8 w-64 bg-muted animate-pulse rounded" />
                            <div className="h-4 w-96 bg-muted/60 animate-pulse rounded" />
                        </div>
                        <div className="h-10 w-24 bg-muted animate-pulse rounded-lg" />
                    </div>
                </div>
                <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
                    {[1, 2, 3, 4].map(i => (
                        <Card key={i} className="border-border/50">
                            <CardContent className="p-6">
                                <div className="h-4 w-24 bg-muted animate-pulse rounded mb-4" />
                                <div className="h-8 w-16 bg-muted animate-pulse rounded" />
                            </CardContent>
                        </Card>
                    ))}
                </div>
            </div>
        )
    }

    if (!data) {
        return (
            <div className="flex flex-col items-center justify-center min-h-[400px] space-y-4">
                <div className="p-4 rounded-2xl bg-muted/10 ring-1 ring-border/50">
                    <AlertTriangle className="h-12 w-12 text-muted-foreground/50" />
                </div>
                <div className="text-center space-y-2">
                    <h3 className="text-lg font-semibold">集群加载失败</h3>
                    <p className="text-sm text-muted-foreground">无法找到集群或加载出错</p>
                </div>
                <Button variant="outline" onClick={() => router.push('/calico')}>
                    返回概览
                </Button>
            </div>
        )
    }

    const isHealthy = data.health_status === 'healthy'

    return (
        <div className="space-y-6">
            {/* Header */}
            <Card className="border-border/50">
                <CardContent className="p-6">
                    <div className="flex items-center justify-between">
                        <div className="flex items-center gap-4">
                            <Button
                                variant="ghost"
                                size="icon"
                                onClick={() => router.push('/calico')}
                            >
                                <ArrowLeft className="h-5 w-5" />
                            </Button>
                            <div className="inline-flex items-center justify-center rounded-xl bg-primary/10 p-2.5">
                                <Network className="h-6 w-6 text-primary" />
                            </div>
                            <div>
                                <div className="flex items-center gap-3">
                                    <h1 className="text-2xl font-bold">{clusterName}</h1>
                                    <span className={cn(
                                        "inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium border",
                                        isHealthy
                                            ? "bg-green-500/10 border-green-500/20 text-green-600 dark:text-green-400"
                                            : "bg-red-500/10 border-red-500/20 text-red-600 dark:text-red-400"
                                    )}>
                                        <div className={cn(
                                            "h-2 w-2 rounded-full",
                                            isHealthy ? "bg-green-500" : "bg-red-500"
                                        )} />
                                        {isHealthy ? '运行正常' : '存在异常'}
                                    </span>
                                </div>
                                <p className="text-sm text-muted-foreground mt-0.5">
                                    Calico 网络详情与配置管理
                                </p>
                            </div>
                        </div>
                        <Button
                            variant="outline"
                            size="sm"
                            onClick={fetchData}
                            disabled={isRefreshing}
                        >
                            <RotateCw className={cn("mr-2 h-4 w-4", isRefreshing && 'animate-spin')} />
                            刷新
                        </Button>
                    </div>
                </CardContent>
            </Card>

            {/* Health Banner */}
            <div className={cn(
                "rounded-xl border-l-4 p-4",
                isHealthy
                    ? "border-l-green-500 bg-green-500/5 border border-green-500/20"
                    : "border-l-red-500 bg-red-500/5 border border-red-500/20"
            )}>
                <div className="flex items-start gap-3">
                    {isHealthy ? (
                        <CheckCircle2 className="h-5 w-5 text-green-500 mt-0.5" />
                    ) : (
                        <AlertTriangle className="h-5 w-5 text-red-500 mt-0.5" />
                    )}
                    <div className="flex-1">
                        <p className={cn(
                            "font-medium text-sm mb-1",
                            isHealthy ? "text-green-600 dark:text-green-400" : "text-red-600 dark:text-red-400"
                        )}>
                            {isHealthy ? '网络状态正常' : '检测到网络异常'}
                        </p>
                        {isHealthy ? (
                            <p className="text-sm text-muted-foreground">
                                所有 Calico 组件运行正常，BGP 对等体连接已建立，IP 地址池容量充足。
                            </p>
                        ) : (
                            <ul className="space-y-1">
                                {(data.issues ?? []).map((issue, i) => (
                                    <li key={i} className="text-sm text-muted-foreground flex items-start gap-2">
                                        <span className="h-1 w-1 rounded-full bg-red-500 mt-1.5 flex-shrink-0" />
                                        {issue}
                                    </li>
                                ))}
                                {(data.issues?.length ?? 0) === 0 && (
                                    <li className="text-sm text-muted-foreground">检测到未知异常，请检查日志。</li>
                                )}
                            </ul>
                        )}
                    </div>
                </div>
            </div>

            {/* Quick Stats - with colored left borders */}
            <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
                <Card className="border-l-4 border-l-blue-500 border-border/50">
                    <CardContent className="p-5">
                        <div className="flex items-center justify-between">
                            <div>
                                <p className="text-2xl font-bold tabular-nums">{data.global_network_policies?.length ?? 0}</p>
                                <p className="text-xs text-muted-foreground mt-1">全局策略</p>
                            </div>
                            <div className="h-10 w-10 rounded-lg bg-blue-500/10 flex items-center justify-center">
                                <Globe className="h-5 w-5 text-blue-500" />
                            </div>
                        </div>
                    </CardContent>
                </Card>

                <Card className="border-l-4 border-l-purple-500 border-border/50">
                    <CardContent className="p-5">
                        <div className="flex items-center justify-between">
                            <div>
                                <p className="text-2xl font-bold tabular-nums">{data.network_set_count + data.global_network_set_count}</p>
                                <p className="text-xs text-muted-foreground mt-1">网络集合</p>
                            </div>
                            <div className="h-10 w-10 rounded-lg bg-purple-500/10 flex items-center justify-center">
                                <Layers className="h-5 w-5 text-purple-500" />
                            </div>
                        </div>
                    </CardContent>
                </Card>

                <Card className="border-l-4 border-l-orange-500 border-border/50">
                    <CardContent className="p-5">
                        <div className="flex items-center justify-between">
                            <div>
                                <p className="text-2xl font-bold tabular-nums">{data.host_endpoint_count}</p>
                                <p className="text-xs text-muted-foreground mt-1">主机端点</p>
                            </div>
                            <div className="h-10 w-10 rounded-lg bg-orange-500/10 flex items-center justify-center">
                                <Monitor className="h-5 w-5 text-orange-500" />
                            </div>
                        </div>
                    </CardContent>
                </Card>

                <Card className="border-l-4 border-l-teal-500 border-border/50">
                    <CardContent className="p-5">
                        <div className="flex items-center justify-between">
                            <div>
                                <p className="text-2xl font-bold tabular-nums">{data.bgp_peers?.length ?? 0}</p>
                                <p className="text-xs text-muted-foreground mt-1">BGP 对等体</p>
                            </div>
                            <div className="h-10 w-10 rounded-lg bg-teal-500/10 flex items-center justify-center">
                                <Share2 className="h-5 w-5 text-teal-500" />
                            </div>
                        </div>
                    </CardContent>
                </Card>
            </div>

            {/* Content Sections */}
            <div className="space-y-6">
                {/* Section Header */}
                <div className="flex items-center gap-2 text-sm text-muted-foreground">
                    <Shield className="h-4 w-4" />
                    <span>网络配置详情</span>
                </div>

                {/* IP Pool Section */}
                <IPPoolTable
                    ipv4Pools={data.ip_pools_v4 ?? []}
                    ipv6Pools={data.ip_pools_v6 ?? []}
                    clusterName={clusterName}
                    onSyncWayne={handleSyncWayne}
                    isSyncing={isSyncing}
                    syncResult={syncResult}
                />

                {/* Grid Layout for Configs */}
                <div className="grid gap-6 md:grid-cols-2">
                    <BGPConfigCard config={data.bgp_configuration} />
                    <FelixConfigCard config={data.felix_configuration} />
                </div>

                {/* BGP Peers */}
                <BGPPeerList peers={data.bgp_peers ?? []} />

                {/* Policies */}
                <NetworkPolicySummary
                    globalPolicies={data.global_network_policies ?? []}
                    namespacedCounts={data.namespaced_policy_counts ?? {}}
                />
            </div>
        </div>
    )
}


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
    Share2
} from "lucide-react"
import { IPPoolTable } from "../components/IPPoolTable"
import { BGPPeerList } from "../components/BGPPeerList"
import { BGPConfigCard, FelixConfigCard, NetworkPolicySummary } from "../components/InfoCards"
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Card, CardContent } from "@/components/ui/card"

export default function ClusterCalicoDetailPage() {
    const params = useParams()
    const router = useRouter()
    const clusterName = params.cluster as string

    const [data, setData] = useState<ClusterDetailResponse | null>(null)
    const [isLoading, setIsLoading] = useState(true)

    const fetchData = async () => {
        setIsLoading(true)
        try {
            const res = await RobustaAPI.getCalicoClusterDetail(clusterName)
            setData(res)
        } catch (error) {
            console.error("Failed to fetch cluster detail", error)
        } finally {
            setIsLoading(false)
        }
    }

    useEffect(() => {
        if (clusterName) {
            fetchData()
        }
    }, [clusterName])

    if (isLoading) {
        return (
            <div className="space-y-6 animate-pulse">
                <div className="h-8 w-1/3 bg-muted rounded" />
                <div className="h-24 w-full bg-muted rounded" />
                <div className="grid gap-6 md:grid-cols-4">
                    {[1, 2, 3, 4].map(i => <div key={i} className="h-32 bg-muted rounded" />)}
                </div>
            </div>
        )
    }

    if (!data) {
        return <div className="py-10">Cluster not found or failed to load.</div>
    }

    return (
        <div className="space-y-6">
            {/* Header */}
            <div className="flex items-center justify-between">
                <div className="flex items-center gap-4">
                    <Button variant="ghost" size="icon" onClick={() => router.push('/calico')}>
                        <ArrowLeft className="h-5 w-5" />
                    </Button>
                    <div>
                        <div className="text-sm text-muted-foreground mb-1">Calico 网络 / {clusterName}</div>
                        <h1 className="text-2xl font-bold tracking-tight">集群网络概览</h1>
                    </div>
                </div>
                <Button variant="outline" size="sm" onClick={fetchData}>
                    <RotateCw className="mr-2 h-4 w-4" /> 刷新
                </Button>
            </div>

            {/* Health Banner */}
            {data.health_status === 'healthy' ? (
                <div className="rounded-lg border bg-green-500/10 border-green-500/20 p-6 flex items-start gap-4">
                    <CheckCircle2 className="h-8 w-8 text-green-500 mt-1" />
                    <div>
                        <h3 className="text-lg font-semibold text-green-600">网络状态正常</h3>
                        <p className="text-muted-foreground text-sm mt-1">
                            所有 Calico 组件运行正常。BGP 对等体连接已建立，IP 地址池容量充足，网络策略已生效。
                        </p>
                    </div>
                </div>
            ) : (
                <Alert variant="destructive" className="bg-red-500/10 border-red-500/20">
                    <AlertTriangle className="h-5 w-5" />
                    <AlertTitle className="ml-2 text-lg font-semibold">检测到网络异常</AlertTitle>
                    <AlertDescription className="mt-2 ml-7">
                        <ul className="list-disc space-y-1">
                            {data.issues.map((issue, i) => (
                                <li key={i}>{issue}</li>
                            ))}
                            {data.issues.length === 0 && <li>检测到未知异常，请检查日志。</li>}
                        </ul>
                    </AlertDescription>
                </Alert>
            )}

            {/* Quick Stats Row with Icons */}
            <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
                <Card className="bg-card/50 hover:bg-card/80 transition-colors">
                    <CardContent className="p-6 flex items-center gap-4">
                        <div className="p-3 bg-blue-500/10 rounded-full">
                            <Globe className="h-6 w-6 text-blue-500" />
                        </div>
                        <div>
                            <p className="text-sm font-medium text-muted-foreground">全局策略</p>
                            <h3 className="text-2xl font-bold">{data.global_network_policies.length}</h3>
                        </div>
                    </CardContent>
                </Card>
                <Card className="bg-card/50 hover:bg-card/80 transition-colors">
                    <CardContent className="p-6 flex items-center gap-4">
                        <div className="p-3 bg-purple-500/10 rounded-full">
                            <Layers className="h-6 w-6 text-purple-500" />
                        </div>
                        <div>
                            <p className="text-sm font-medium text-muted-foreground">网络集合</p>
                            <h3 className="text-2xl font-bold">{data.network_set_count + data.global_network_set_count}</h3>
                        </div>
                    </CardContent>
                </Card>
                <Card className="bg-card/50 hover:bg-card/80 transition-colors">
                    <CardContent className="p-6 flex items-center gap-4">
                        <div className="p-3 bg-orange-500/10 rounded-full">
                            <Monitor className="h-6 w-6 text-orange-500" />
                        </div>
                        <div>
                            <p className="text-sm font-medium text-muted-foreground">主机端点</p>
                            <h3 className="text-2xl font-bold">{data.host_endpoint_count}</h3>
                        </div>
                    </CardContent>
                </Card>
                <Card className="bg-card/50 hover:bg-card/80 transition-colors">
                    <CardContent className="p-6 flex items-center gap-4">
                        <div className="p-3 bg-teal-500/10 rounded-full">
                            <Share2 className="h-6 w-6 text-teal-500" />
                        </div>
                        <div>
                            <p className="text-sm font-medium text-muted-foreground">BGP 对等体</p>
                            <h3 className="text-2xl font-bold">{data.bgp_peers.length}</h3>
                        </div>
                    </CardContent>
                </Card>
            </div>

            {/* IP Pool Section */}
            <IPPoolTable ipv4Pools={data.ip_pools_v4} ipv6Pools={data.ip_pools_v6} />

            {/* Grid Layout for Configs */}
            <div className="grid gap-6 md:grid-cols-2">
                {/* BGP Config */}
                <BGPConfigCard config={data.bgp_configuration} />

                {/* Felix Config */}
                <FelixConfigCard config={data.felix_configuration} />
            </div>

            {/* BGP Peers */}
            <BGPPeerList peers={data.bgp_peers} />

            {/* Policies */}
            <div className="grid gap-6 md:grid-cols-1">
                <NetworkPolicySummary
                    globalPolicies={data.global_network_policies}
                    namespacedCounts={data.namespaced_policy_counts}
                />
            </div>
        </div>
    )
}

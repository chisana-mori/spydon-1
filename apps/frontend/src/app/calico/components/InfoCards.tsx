import { BGPConfiguration, FelixConfiguration, PolicySummary } from "@/types/calico"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Check, X, Shield, Activity, Settings2, Zap } from "lucide-react"
import { cn } from "@/lib/utils"
import { Badge } from "@/components/ui/badge"

export function BGPConfigCard({ config }: { config: BGPConfiguration | null }) {
    if (!config) return null

    const items = [
        { label: "Node-to-Node Mesh", value: config.spec.nodeToNodeMeshEnabled, type: 'bool' },
        { label: "AS Number", value: config.spec.asNumber, type: 'text' },
        { label: "Log Severity", value: config.spec.logSeverityScreen, type: 'text' },
        { label: "Service Cluster IPs", value: !!config.spec.serviceClusterIPs, type: 'bool' },
        { label: "External IPs", value: !!config.spec.serviceExternalIPs, type: 'bool' },
    ]

    return (
        <Card className="border-l-4 border-l-amber-500 border-border/50">
            <CardHeader className="pb-4">
                <CardTitle className="flex items-center gap-2 text-base">
                    <div className="h-8 w-8 rounded-lg bg-amber-500/10 flex items-center justify-center">
                        <Activity className="h-4 w-4 text-amber-500" />
                    </div>
                    BGP 配置
                </CardTitle>
            </CardHeader>
            <CardContent>
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-y-4 gap-x-6">
                    {items.map((item, i) => (
                        <div key={i} className="flex flex-col">
                            <span className="text-xs text-muted-foreground mb-1">{item.label}</span>
                            <div className="flex items-center gap-2">
                                {item.type === 'bool' ? (
                                    item.value !== false ? (
                                        <>
                                            <Check className="h-4 w-4 text-green-500" />
                                            <span className="text-sm font-medium text-green-600 dark:text-green-400">已启用</span>
                                        </>
                                    ) : (
                                        <>
                                            <X className="h-4 w-4 text-muted-foreground/50" />
                                            <span className="text-sm text-muted-foreground">未启用</span>
                                        </>
                                    )
                                ) : (
                                    <span className="text-sm font-mono bg-muted/50 px-2 py-0.5 rounded">{item.value?.toString() || '-'}</span>
                                )}
                            </div>
                        </div>
                    ))}
                </div>
            </CardContent>
        </Card>
    )
}

export function FelixConfigCard({ config }: { config: FelixConfiguration | null }) {
    if (!config) return null

    const items = [
        { label: "IPIP", value: config.spec.ipipEnabled, type: 'bool' },
        { label: "VXLAN", value: config.spec.vxlanEnabled, type: 'bool' },
        { label: "Wireguard", value: config.spec.wireguardEnabled, type: 'bool' },
        { label: "Log Level", value: config.spec.logSeverityScreen || 'Info', type: 'text' },
        { label: "Prometheus", value: config.spec.prometheusMetricsEnabled, type: 'bool' },
    ]

    return (
        <Card className="border-l-4 border-l-indigo-500 border-border/50">
            <CardHeader className="pb-4">
                <CardTitle className="flex items-center gap-2 text-base">
                    <div className="h-8 w-8 rounded-lg bg-indigo-500/10 flex items-center justify-center">
                        <Settings2 className="h-4 w-4 text-indigo-500" />
                    </div>
                    Felix 配置
                </CardTitle>
            </CardHeader>
            <CardContent>
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-y-4 gap-x-6">
                    {items.map((item, i) => (
                        <div key={i} className="flex flex-col">
                            <span className="text-xs text-muted-foreground mb-1">{item.label}</span>
                            <div className="flex items-center gap-2">
                                {item.type === 'bool' ? (
                                    item.value === true ? (
                                        <>
                                            <Check className="h-4 w-4 text-green-500" />
                                            <span className="text-sm font-medium text-green-600 dark:text-green-400">开启</span>
                                        </>
                                    ) : (
                                        <>
                                            <X className="h-4 w-4 text-muted-foreground/50" />
                                            <span className="text-sm text-muted-foreground">关闭</span>
                                        </>
                                    )
                                ) : (
                                    <Badge variant="secondary" className="text-xs">
                                        {item.value?.toString() || '-'}
                                    </Badge>
                                )}
                            </div>
                        </div>
                    ))}
                </div>
            </CardContent>
        </Card>
    )
}

export function NetworkPolicySummary({ globalPolicies, namespacedCounts }: { globalPolicies: PolicySummary[], namespacedCounts: Record<string, number> }) {
    const totalNamespaced = Object.values(namespacedCounts).reduce((a, b) => a + b, 0)
    const topNamespaces = Object.entries(namespacedCounts)
        .sort(([, a], [, b]) => b - a)
        .slice(0, 5)

    return (
        <Card className="border-l-4 border-l-violet-500 border-border/50">
            <CardHeader className="pb-4">
                <CardTitle className="flex items-center gap-2 text-base">
                    <div className="h-8 w-8 rounded-lg bg-violet-500/10 flex items-center justify-center">
                        <Shield className="h-4 w-4 text-violet-500" />
                    </div>
                    网络策略
                </CardTitle>
            </CardHeader>
            <CardContent>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-8">
                    {/* Global Policies */}
                    <div className="space-y-4">
                        <div className="flex items-center justify-between">
                            <span className="text-sm font-medium">全局策略</span>
                            <Badge variant="outline" className="text-xs border-violet-500/20 text-violet-600 dark:text-violet-400">
                                {globalPolicies.length} 条
                            </Badge>
                        </div>
                        <div className="space-y-2">
                            {globalPolicies.length > 0 ? (
                                globalPolicies.slice(0, 6).map(p => (
                                    <div key={p.name} className="flex items-center gap-2 text-sm">
                                        <div className="h-1 w-1 rounded-full bg-violet-500" />
                                        <span className="font-mono text-xs truncate max-w-[200px]" title={p.name}>{p.name}</span>
                                    </div>
                                ))
                            ) : (
                                <div className="flex items-center gap-2 text-sm text-muted-foreground">
                                    <Zap className="h-3.5 w-3.5 opacity-50" />
                                    <span>暂无全局策略</span>
                                </div>
                            )}
                        </div>
                    </div>

                    {/* Namespaced Policies */}
                    <div className="space-y-4">
                        <div className="flex items-center justify-between">
                            <span className="text-sm font-medium">命名空间策略</span>
                            <Badge variant="outline" className="text-xs border-violet-500/20 text-violet-600 dark:text-violet-400">
                                {totalNamespaced} 条
                            </Badge>
                        </div>

                        {/* Bar Chart */}
                        {topNamespaces.length > 0 && (
                            <div className="space-y-2">
                                {topNamespaces.map(([ns, count], idx) => {
                                    const percentage = totalNamespaced > 0 ? (count / totalNamespaced) * 100 : 0
                                    const colors = [
                                        'bg-violet-500',
                                        'bg-purple-500',
                                        'bg-fuchsia-500',
                                        'bg-pink-500',
                                        'bg-rose-500'
                                    ]
                                    return (
                                        <div key={ns} className="space-y-1">
                                            <div className="flex justify-between text-xs">
                                                <span className="font-mono truncate max-w-[100px]" title={ns}>{ns}</span>
                                                <span className="tabular-nums">{count}</span>
                                            </div>
                                            <div className="h-1.5 w-full bg-muted rounded-full overflow-hidden">
                                                <div
                                                    className={cn("h-full rounded-full", colors[idx % colors.length])}
                                                    style={{ width: `${percentage}%` }}
                                                />
                                            </div>
                                        </div>
                                    )
                                })}
                            </div>
                        )}

                        {topNamespaces.length === 0 && (
                            <div className="flex items-center gap-2 text-sm text-muted-foreground">
                                <Zap className="h-3.5 w-3.5 opacity-50" />
                                <span>暂无命名空间策略</span>
                            </div>
                        )}
                    </div>
                </div>
            </CardContent>
        </Card>
    )
}

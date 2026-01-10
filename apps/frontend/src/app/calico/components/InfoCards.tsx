
import { BGPConfiguration, FelixConfiguration, PolicySummary } from "@/types/calico"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Check, X, Shield, Activity, Settings2 } from "lucide-react"

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
        <Card className="h-full">
            <CardHeader>
                <CardTitle className="flex items-center gap-2">
                    <Activity className="h-5 w-5 text-primary" />
                    BGP 配置
                </CardTitle>
            </CardHeader>
            <CardContent>
                <div className="grid grid-cols-2 gap-y-4 gap-x-8">
                    {items.map((item, i) => (
                        <div key={i} className="flex flex-col">
                            <span className="text-sm text-muted-foreground">{item.label}</span>
                            <div className="flex items-center gap-2 font-medium mt-1">
                                {item.type === 'bool' ? (
                                    item.value !== false ? (
                                        <><Check className="h-4 w-4 text-green-500" /> 已启用</>
                                    ) : (
                                        <><X className="h-4 w-4 text-muted-foreground" /> 未启用</>
                                    )
                                ) : (
                                    <span>{item.value?.toString() || '-'}</span>
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
        { label: "IPIP Enabled", value: config.spec.ipipEnabled, type: 'bool' },
        { label: "VXLAN Enabled", value: config.spec.vxlanEnabled, type: 'bool' },
        { label: "Wireguard Enabled", value: config.spec.wireguardEnabled, type: 'bool' },
        { label: "Log Level", value: config.spec.logSeverityScreen || 'Info', type: 'text' },
        { label: "Prometheus Metrics", value: config.spec.prometheusMetricsEnabled, type: 'bool' },
    ]

    return (
        <Card className="h-full">
            <CardHeader>
                <CardTitle className="flex items-center gap-2">
                    <Settings2 className="h-5 w-5 text-primary" />
                    Felix 配置
                </CardTitle>
            </CardHeader>
            <CardContent>
                <div className="grid grid-cols-2 md:grid-cols-3 gap-y-4 gap-x-8">
                    {items.map((item, i) => (
                        <div key={i} className="flex flex-col">
                            <span className="text-sm text-muted-foreground">{item.label}</span>
                            <div className="flex items-center gap-2 font-medium mt-1">
                                {item.type === 'bool' ? (
                                    item.value === true ? (
                                        <><Check className="h-4 w-4 text-green-500" /> 开启</>
                                    ) : (
                                        <><X className="h-4 w-4 text-muted-foreground" /> 关闭</>
                                    )
                                ) : (
                                    <span>{item.value?.toString() || '-'}</span>
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
        <Card className="h-full">
            <CardHeader>
                <CardTitle className="flex items-center gap-2">
                    <Shield className="h-5 w-5 text-primary" />
                    网络策略
                </CardTitle>
            </CardHeader>
            <CardContent>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-8">
                    <div>
                        <div className="text-sm text-muted-foreground mb-2">全局策略</div>
                        <div className="text-2xl font-bold mb-4">{globalPolicies.length} <span className="text-sm font-normal text-muted-foreground">生效中</span></div>
                        <ul className="space-y-1 text-sm">
                            {globalPolicies.slice(0, 5).map(p => (
                                <li key={p.name} className="flex items-center gap-2">
                                    <span className="h-1.5 w-1.5 rounded-full bg-blue-500" />
                                    {p.name}
                                </li>
                            ))}
                            {globalPolicies.length > 5 && (
                                <li className="text-muted-foreground pl-3.5 pt-1 text-xs">+ {globalPolicies.length - 5} more</li>
                            )}
                            {globalPolicies.length === 0 && <li className="text-muted-foreground italic">无全局策略</li>}
                        </ul>
                    </div>
                    <div>
                        <div className="text-sm text-muted-foreground mb-2">命名空间策略</div>
                        <div className="text-2xl font-bold mb-4">{totalNamespaced} <span className="text-sm font-normal text-muted-foreground">分布在 {Object.keys(namespacedCounts).length} 个命名空间</span></div>

                        {/* Simple Bar Chart Visualization using CSS */}
                        <div className="space-y-2 mt-4">
                            {topNamespaces.map(([ns, count]) => {
                                const percentage = totalNamespaced > 0 ? (count / totalNamespaced) * 100 : 0
                                return (
                                    <div key={ns} className="space-y-1">
                                        <div className="flex justify-between text-xs">
                                            <span>{ns}</span>
                                            <span>{count}</span>
                                        </div>
                                        <div className="h-1.5 w-full bg-muted rounded-full overflow-hidden">
                                            <div className="h-full bg-indigo-500" style={{ width: `${percentage}%` }} />
                                        </div>
                                    </div>
                                )
                            })}
                        </div>
                    </div>
                </div>
            </CardContent>
        </Card>
    )
}

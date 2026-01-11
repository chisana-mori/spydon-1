import { useState, useEffect } from 'react'
import {
    Sheet,
    SheetContent,
    SheetDescription,
    SheetHeader,
    SheetTitle,
} from '@/components/ui/sheet'
import { Cluster } from '@/types/api'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Copy, Server, HardDrive, Activity, Network } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { toast } from 'sonner'
import { cn } from '@/lib/utils'

interface ClusterNodesSheetProps {
    open: boolean
    onOpenChange: (open: boolean) => void
    cluster: Cluster | null
}

export function ClusterNodesSheet({ open, onOpenChange, cluster }: ClusterNodesSheetProps) {
    if (!cluster) return null

    const [isReady, setIsReady] = useState(false)

    useEffect(() => {
        if (open) {
            const timer = setTimeout(() => {
                requestAnimationFrame(() => setIsReady(true))
            }, 300) // Wait for slide-in animation to complete
            return () => clearTimeout(timer)
        } else {
            setIsReady(false)
        }
    }, [open])

    const copyToClipboard = (text: string) => {
        navigator.clipboard.writeText(text)
        toast.success('IP 已复制')
    }

    // Design System Mappings - Infusing Low-Saturation Semantic Colors
    const SECTION_STYLES = {
        master: {
            color: 'text-sky-600/80 dark:text-sky-400/80',
            bg: 'bg-sky-500/10',
            border: 'group-hover:border-sky-500/30',
            icon: Server,
            label: 'Master Control Plane',
            desc: '负责集群的控制与调度'
        },
        etcd: {
            color: 'text-slate-600/80 dark:text-slate-400/80',
            bg: 'bg-slate-500/10',
            border: 'group-hover:border-slate-500/30',
            icon: HardDrive,
            label: 'Etcd Storage',
            desc: '一致性键值存储数据库'
        },
        event: {
            color: 'text-amber-600/70 dark:text-amber-400/70',
            bg: 'bg-amber-500/10',
            border: 'group-hover:border-amber-500/30',
            icon: Activity,
            label: 'Etcd Events',
            desc: '专门存储 Kubernetes 事件'
        }
    }

    const renderNodeSection = (title: string, ips: string[] | undefined, type: 'master' | 'etcd' | 'event') => {
        if (!ips || ips.length === 0) return null
        const style = SECTION_STYLES[type]
        const Icon = style.icon

        // Low-saturation hover states
        const hoverStyles = {
            master: "hover:bg-sky-500/[0.03] hover:border-sky-500/20 dark:hover:bg-sky-500/10",
            etcd: "hover:bg-slate-500/[0.03] hover:border-slate-500/20 dark:hover:bg-slate-500/10",
            event: "hover:bg-amber-500/[0.03] hover:border-amber-500/20 dark:hover:bg-amber-500/10"
        }

        return (
            <div className="space-y-4">
                <div className="flex items-center justify-between pb-2 border-b border-border/60">
                    <div className="space-y-1">
                        <h4 className="text-sm font-semibold text-foreground/90 flex items-center gap-2">
                            {title}
                        </h4>
                        <p className="text-[10px] text-muted-foreground/80">{style.desc}</p>
                    </div>
                    <Badge
                        variant="outline"
                        className={cn(
                            "h-5 px-2 text-[10px] font-mono border-transparent",
                            style.bg, style.color
                        )}
                    >
                        {ips.length} NODES
                    </Badge>
                </div>
                <div className="grid gap-3">
                    {!isReady ? (
                        Array.from({ length: Math.min(ips.length, 3) }).map((_, i) => (
                            <Skeleton key={i} className="h-[74px] w-full rounded-xl bg-muted/40" />
                        ))
                    ) : (
                        ips.map((ip, index) => (
                            <div
                                key={ip}
                                onClick={() => copyToClipboard(ip)}
                                className={cn(
                                    "group relative flex items-center justify-between p-3 rounded-xl border border-border/50 cursor-pointer",
                                    "bg-card/50 backdrop-blur-sm transition-all duration-300",
                                    hoverStyles[type]
                                )}
                                title="点击复制 IP"
                            >
                                <div className="flex items-center gap-4">
                                    {/* Semantic Icon Container */}
                                    <div className={cn(
                                        "h-10 w-10 rounded-lg flex items-center justify-center transition-all duration-500",
                                        style.bg,
                                        "group-hover:scale-105 group-hover:shadow-inner"
                                    )}>
                                        <Icon className={cn("h-5 w-5 opacity-70 group-hover:opacity-100 transition-opacity", style.color)} />
                                    </div>

                                    <div className="flex flex-col gap-0.5">
                                        <span className="font-mono text-sm font-medium tracking-tight text-foreground/80 group-hover:text-foreground transition-colors">{ip}</span>
                                        <span className="text-[10px] text-muted-foreground/60 font-mono">
                                            INDEX: {String(index + 1).padStart(2, '0')}
                                        </span>
                                    </div>
                                </div>

                                <div className="flex items-center gap-2">
                                    <span className={cn(
                                        "opacity-0 group-hover:opacity-100 text-[10px] font-medium transition-all transform translate-x-2 group-hover:translate-x-0",
                                        style.color
                                    )}>
                                        复制 IP
                                    </span>
                                    <Button
                                        variant="ghost"
                                        size="icon"
                                        className={cn(
                                            "h-8 w-8 text-muted-foreground/40 transition-all",
                                            "hover:bg-background hover:shadow-sm",
                                            style.color.replace('text-', 'hover:text-')
                                        )}
                                    >
                                        <Copy className="h-4 w-4" />
                                    </Button>
                                </div>
                            </div>
                        )))}
                </div>
            </div>
        )
    }

    const totalNodes = (cluster.master_ips?.length || 0) +
        (cluster.etcd_ips?.length || 0) +
        (cluster.etcd_event_ips?.length || 0)

    return (
        <Sheet open={open} onOpenChange={onOpenChange}>
            <SheetContent className="p-0 w-full sm:max-w-[1200px] sm:w-[1200px] flex flex-col h-full bg-background border-l shadow-2xl overflow-hidden">
                <div className="relative overflow-hidden border-b bg-muted/20">
                    {/* Background Decorative Elements - Subtle and Adaptive */}
                    {isReady && (
                        <div className="absolute top-0 right-0 w-[500px] h-[500px] bg-primary/5 blur-[120px] -mr-48 -mt-48 transition-all animate-in fade-in duration-700" />
                    )}

                    <div className="px-8 py-8 relative z-10">
                        <div className="flex flex-col md:flex-row md:items-end justify-between gap-6">
                            <div className="space-y-4">
                                <div className="flex items-center gap-4">
                                    <div className="relative group">
                                        <div className="absolute -inset-1 bg-gradient-to-r from-primary/80 to-primary/40 rounded-2xl blur opacity-25 group-hover:opacity-40 transition duration-1000 group-hover:duration-200"></div>
                                        <div className="relative p-3.5 bg-background border border-border rounded-2xl ring-1 ring-inset ring-primary/20 text-primary shadow-xl">
                                            <Server className="h-7 w-7" />
                                        </div>
                                    </div>
                                    <div className="space-y-1">
                                        <div className="flex items-center gap-2">
                                            <h2 className="text-2xl font-bold tracking-tight bg-clip-text text-transparent bg-gradient-to-br from-foreground to-foreground/70">
                                                控制节点详情
                                            </h2>
                                            <Badge variant="secondary" className="h-6 px-2 font-mono">
                                                {cluster.cluster_id || 'LOCAL-CLUSTER'}
                                            </Badge>
                                        </div>
                                        <p className="text-sm text-muted-foreground flex items-center gap-2">
                                            <span className="font-semibold text-foreground/80">{cluster.name}</span>
                                            <span className="w-1 h-1 rounded-full bg-muted-foreground/30" />
                                            查看集群核心组件拓扑与状态
                                        </p>
                                    </div>
                                </div>
                            </div>

                            {/* Stats Summary Panel - Shadcn Styled */}
                            <div className="flex items-center gap-3 p-1.5 bg-background/50 backdrop-blur-sm rounded-2xl border border-border shadow-sm">
                                <div className="px-4 py-2 bg-muted/50 rounded-xl border border-border flex items-center gap-3 transition-all hover:border-primary/20 group">
                                    <div className="h-8 w-8 rounded-lg bg-primary/10 flex items-center justify-center text-primary group-hover:bg-primary group-hover:text-primary-foreground transition-all">
                                        <Network className="h-4 w-4" />
                                    </div>
                                    <div className="flex flex-col">
                                        <span className="text-[10px] text-muted-foreground font-medium uppercase tracking-wider">Total Nodes</span>
                                        <span className="text-lg font-bold font-mono leading-none">{totalNodes}</span>
                                    </div>
                                </div>
                                <div className="h-8 w-px bg-border" />
                                <div className="flex items-center gap-0.5 px-2">
                                    <div className="flex flex-col items-center px-4 hover:bg-muted rounded-lg transition-colors py-1">
                                        <span className="text-sm font-bold text-primary">{cluster.master_ips?.length || 0}</span>
                                        <span className="text-[9px] text-muted-foreground font-medium uppercase">Master</span>
                                    </div>
                                    <div className="flex flex-col items-center px-4 hover:bg-muted rounded-lg transition-colors py-1">
                                        <span className="text-sm font-bold text-foreground">{cluster.etcd_ips?.length || 0}</span>
                                        <span className="text-[9px] text-muted-foreground font-medium uppercase">Etcd</span>
                                    </div>
                                    <div className="flex flex-col items-center px-4 hover:bg-muted rounded-lg transition-colors py-1">
                                        <span className="text-sm font-bold text-muted-foreground">{cluster.etcd_event_ips?.length || 0}</span>
                                        <span className="text-[9px] text-muted-foreground font-medium uppercase">Event</span>
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>

                <ScrollArea className="flex-1">
                    <div className="px-8 py-8">
                        {!cluster.master_ips?.length && !cluster.etcd_ips?.length && !cluster.etcd_event_ips?.length && (
                            <div className="text-center py-10 text-muted-foreground">
                                <Server className="h-10 w-10 mx-auto mb-3 opacity-20" />
                                <p>暂无节点信息</p>
                            </div>
                        )}

                        <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
                            {/* Master 节点列 */}
                            <div className="space-y-4">
                                {renderNodeSection('Master 节点', cluster.master_ips, 'master')}
                                {!cluster.master_ips?.length && (
                                    <div className="h-full border rounded-xl border-dashed flex flex-col items-center justify-center min-h-[120px] bg-sky-500/[0.02] border-sky-500/10 transition-colors hover:bg-sky-500/[0.05]">
                                        <Server className="h-8 w-8 text-sky-500/20 mb-2" />
                                        <span className="text-xs text-sky-600/40 font-medium">无 Master 节点数据</span>
                                    </div>
                                )}
                            </div>

                            {/* Etcd 节点列 */}
                            <div className="space-y-4">
                                {renderNodeSection('Etcd 节点', cluster.etcd_ips, 'etcd')}
                                {!cluster.etcd_ips?.length && (
                                    <div className="h-full border rounded-xl border-dashed flex flex-col items-center justify-center min-h-[120px] bg-slate-500/[0.02] border-slate-500/10 transition-colors hover:bg-slate-500/[0.05]">
                                        <HardDrive className="h-8 w-8 text-slate-500/20 mb-2" />
                                        <span className="text-xs text-slate-600/40 font-medium">无 Etcd 节点数据</span>
                                    </div>
                                )}
                            </div>

                            {/* Etcd Event 节点列 */}
                            <div className="space-y-4">
                                {renderNodeSection('Etcd Event 节点', cluster.etcd_event_ips, 'event')}
                                {!cluster.etcd_event_ips?.length && (
                                    <div className="h-full border rounded-xl border-dashed flex flex-col items-center justify-center min-h-[120px] bg-amber-500/[0.02] border-amber-500/10 transition-colors hover:bg-amber-500/[0.05]">
                                        <Activity className="h-8 w-8 text-amber-500/20 mb-2" />
                                        <span className="text-xs text-amber-600/40 font-medium">无 Event 节点数据</span>
                                    </div>
                                )}
                            </div>
                        </div>
                    </div>
                </ScrollArea>
            </SheetContent>
        </Sheet>
    )
}

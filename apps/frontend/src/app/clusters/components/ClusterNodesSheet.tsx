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
import { Button } from '@/components/ui/button'
import { toast } from 'sonner'
import { Separator } from '@/components/ui/separator'
import { cn } from '@/lib/utils'

interface ClusterNodesSheetProps {
    open: boolean
    onOpenChange: (open: boolean) => void
    cluster: Cluster | null
}

export function ClusterNodesSheet({ open, onOpenChange, cluster }: ClusterNodesSheetProps) {
    if (!cluster) return null

    const copyToClipboard = (text: string) => {
        navigator.clipboard.writeText(text)
        toast.success('IP 已复制')
    }

    // Design System Mappings
    const SECTION_STYLES = {
        master: {
            color: 'text-blue-500',
            bg: 'bg-blue-500/10',
            border: 'hover:border-blue-500/20',
            icon: Server,
            label: 'Master Control Plane',
            desc: '负责集群的控制与调度'
        },
        etcd: {
            color: 'text-purple-500',
            bg: 'bg-purple-500/10',
            border: 'hover:border-purple-500/20',
            icon: HardDrive,
            label: 'Etcd Storage',
            desc: '一致性键值存储数据库'
        },
        event: {
            color: 'text-orange-500',
            bg: 'bg-orange-500/10',
            border: 'hover:border-orange-500/20',
            icon: Activity,
            label: 'Etcd Events',
            desc: '专门存储 Kubernetes 事件'
        }
    }

    const renderNodeSection = (title: string, ips: string[] | undefined, type: 'master' | 'etcd' | 'event') => {
        if (!ips || ips.length === 0) return null
        const style = SECTION_STYLES[type]
        const Icon = style.icon

        return (
            <div className="space-y-4">
                <div className="flex items-center justify-between pb-2 border-b border-border/40">
                    <div className="space-y-1">
                        <h4 className="text-sm font-semibold text-foreground flex items-center gap-2">
                            {title}
                        </h4>
                        <p className="text-[10px] text-muted-foreground">{style.desc}</p>
                    </div>
                    <span className={cn(
                        "text-[10px] font-mono px-2.5 py-1 rounded-full font-medium ring-1 ring-inset",
                        style.color, style.bg, "ring-transparent" // using bg/color for tag style
                    )}>
                        {ips.length} NODES
                    </span>
                </div>
                <div className="grid gap-3">
                    {ips.map((ip, index) => (
                        <div
                            key={ip}
                            className={cn(
                                "group relative flex items-center justify-between p-3 rounded-xl border",
                                "bg-card/50 hover:bg-accent hover:text-accent-foreground transition-all duration-200"
                            )}
                        >
                            <div className="flex items-center gap-4">
                                {/* Semantic Icon Container */}
                                <div className={cn(
                                    "h-10 w-10 rounded-lg flex items-center justify-center transition-colors",
                                    style.bg
                                )}>
                                    <Icon className={cn("h-5 w-5", style.color)} />
                                </div>

                                <div className="flex flex-col gap-0.5">
                                    <span className="font-mono text-sm font-medium tracking-tight text-foreground/90">{ip}</span>
                                    <span className="text-[10px] text-muted-foreground font-mono">
                                        INDEX: {String(index + 1).padStart(2, '0')}
                                    </span>
                                </div>
                            </div>

                            <Button
                                variant="ghost"
                                size="icon"
                                className="h-8 w-8 text-muted-foreground hover:text-foreground hover:bg-muted transition-all opacity-0 group-hover:opacity-100 translate-x-2 group-hover:translate-x-0"
                                onClick={() => copyToClipboard(ip)}
                                title="复制 IP"
                            >
                                <Copy className="h-4 w-4" />
                            </Button>
                        </div>
                    ))}
                </div>
            </div>
        )
    }

    return (
        <Sheet open={open} onOpenChange={onOpenChange}>
            <SheetContent className="w-[400px] sm:w-[540px] flex flex-col h-full bg-background/95 backdrop-blur-sm border-l shadow-2xl">
                <SheetHeader className="pb-6 border-b z-10 bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
                    <SheetTitle className="flex items-center gap-3 text-xl">
                        <div className="p-2.5 bg-indigo-500/10 rounded-xl ring-1 ring-indigo-500/20 text-indigo-500">
                            <Server className="h-5 w-5" />
                        </div>
                        节点详情
                    </SheetTitle>
                    <SheetDescription>
                        查看 {cluster.name} 的 Master、Etcd 和 Etcd Event 节点列表。
                    </SheetDescription>
                </SheetHeader>

                <ScrollArea className="flex-1 -mx-6 px-6">
                    <div className="py-6 space-y-8">
                        {!cluster.master_ips?.length && !cluster.etcd_ips?.length && !cluster.etcd_event_ips?.length && (
                            <div className="text-center py-10 text-muted-foreground">
                                <Server className="h-10 w-10 mx-auto mb-3 opacity-20" />
                                <p>暂无节点信息</p>
                            </div>
                        )}

                        {renderNodeSection('Master 节点', cluster.master_ips, 'master')}
                        {(cluster.master_ips?.length && (cluster.etcd_ips?.length || cluster.etcd_event_ips?.length)) ? <Separator /> : null}

                        {renderNodeSection('Etcd 节点', cluster.etcd_ips, 'etcd')}
                        {((cluster.etcd_ips?.length || cluster.master_ips?.length) && cluster.etcd_event_ips?.length) ? <Separator /> : null}

                        {renderNodeSection('Etcd Event 节点', cluster.etcd_event_ips, 'event')}
                    </div>
                </ScrollArea>
            </SheetContent>
        </Sheet>
    )
}

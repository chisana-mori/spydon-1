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
                            onClick={() => copyToClipboard(ip)}
                            className={cn(
                                "group relative flex items-center justify-between p-3 rounded-xl border cursor-pointer",
                                "bg-card/50 hover:bg-blue-50/50 dark:hover:bg-blue-900/10 hover:border-blue-200 dark:hover:border-blue-800 transition-all duration-200"
                            )}
                            title="点击复制 IP"
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

                            <div className="flex items-center">
                                <span className="opacity-0 group-hover:opacity-100 text-[10px] text-blue-500 font-medium mr-2 transition-opacity">
                                    点击复制
                                </span>
                                <Button
                                    variant="ghost"
                                    size="icon"
                                    className="h-8 w-8 text-muted-foreground hover:text-blue-500 hover:bg-blue-100/50 dark:hover:bg-blue-900/50 transition-all"
                                >
                                    <Copy className="h-4 w-4" />
                                </Button>
                            </div>
                        </div>
                    ))}
                </div>
            </div>
        )
    }

    return (
        <Sheet open={open} onOpenChange={onOpenChange}>
            <SheetContent className="w-full sm:max-w-[1200px] sm:w-[1200px] flex flex-col h-full bg-background/95 backdrop-blur-sm border-l shadow-2xl">
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
                    <div className="py-6">
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
                                    <div className="h-full border rounded-xl border-dashed flex items-center justify-center min-h-[100px] bg-muted/30">
                                        <span className="text-xs text-muted-foreground">无 Master 节点数据</span>
                                    </div>
                                )}
                            </div>

                            {/* Etcd 节点列 */}
                            <div className="space-y-4">
                                {renderNodeSection('Etcd 节点', cluster.etcd_ips, 'etcd')}
                                {!cluster.etcd_ips?.length && (
                                    <div className="h-full border rounded-xl border-dashed flex items-center justify-center min-h-[100px] bg-muted/30">
                                        <span className="text-xs text-muted-foreground">无 Etcd 节点数据</span>
                                    </div>
                                )}
                            </div>

                            {/* Etcd Event 节点列 */}
                            <div className="space-y-4">
                                {renderNodeSection('Etcd Event 节点', cluster.etcd_event_ips, 'event')}
                                {!cluster.etcd_event_ips?.length && (
                                    <div className="h-full border rounded-xl border-dashed flex items-center justify-center min-h-[100px] bg-muted/30">
                                        <span className="text-xs text-muted-foreground">无 Event 节点数据</span>
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

import {
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableHeader,
    TableRow,
} from "@/components/ui/table"
import { Badge } from "@/components/ui/badge"
import { BGPPeerDetail } from "@/types/calico"
import { Card } from "@/components/ui/card"
import { useState } from "react"
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible"
import { ChevronDown, ChevronRight, CheckCircle2, AlertCircle, Share2 } from "lucide-react"
import { cn } from "@/lib/utils"

interface BGPPeerListProps {
    peers: BGPPeerDetail[]
}

export function BGPPeerList({ peers }: BGPPeerListProps) {
    const globalPeers = peers.filter(p => p.scope === 'global')
    const nodeSpecificPeers = peers.filter(p => p.scope === 'node-specific')

    const establishedCount = peers.filter(p => p.state === 'Established').length

    return (
        <Card className="border-l-4 border-l-teal-500 border-border/50">
            <div className="p-6">
                <div className="flex items-start justify-between mb-4">
                    <div className="space-y-1">
                        <div className="flex items-center gap-3">
                            <div className="h-8 w-8 rounded-lg bg-teal-500/10 flex items-center justify-center">
                                <Share2 className="h-4 w-4 text-teal-500" />
                            </div>
                            <h3 className="text-lg font-semibold">BGP 对等体</h3>
                            <Badge variant="outline" className="text-xs border-teal-500/20 text-teal-600 dark:text-teal-400">
                                {establishedCount}/{peers.length} 已建立
                            </Badge>
                        </div>
                        <p className="text-sm text-muted-foreground ml-11">边界网关协议对等连接状态监控</p>
                    </div>
                </div>

                <div className="space-y-3">
                    <PeerGroup title="全局对等体" peers={globalPeers} defaultOpen={true} />
                    <PeerGroup title="节点特定对等体" peers={nodeSpecificPeers} />
                </div>

                {peers.length === 0 && (
                    <div className="flex flex-col items-center justify-center py-16 text-muted-foreground">
                        <Share2 className="h-10 w-10 opacity-50 mb-4" />
                        <p className="text-sm">未找到 BGP 对等体配置</p>
                    </div>
                )}
            </div>
        </Card>
    )
}

function PeerGroup({ title, peers, defaultOpen = false }: { title: string, peers: BGPPeerDetail[], defaultOpen?: boolean }) {
    const [isOpen, setIsOpen] = useState(defaultOpen)

    if (peers.length === 0) return null

    const establishedCount = peers.filter(p => p.state === 'Established').length

    return (
        <Collapsible open={isOpen} onOpenChange={setIsOpen} className="rounded-lg border border-border/30 overflow-hidden">
            <CollapsibleTrigger className="flex items-center justify-between w-full px-4 py-3 hover:bg-muted/50 transition-colors">
                <div className="flex items-center gap-3">
                    {isOpen ? <ChevronDown className="h-4 w-4 text-muted-foreground" /> : <ChevronRight className="h-4 w-4 text-muted-foreground" />}
                    <span className="font-medium text-sm">{title}</span>
                    <Badge variant="secondary" className="text-xs">{peers.length}</Badge>
                    {establishedCount === peers.length && peers.length > 0 && (
                        <Badge className="bg-green-500/10 text-green-600 dark:text-green-400 border-green-500/20 text-xs gap-1">
                            <CheckCircle2 className="h-3 w-3" />
                            全部在线
                        </Badge>
                    )}
                </div>
            </CollapsibleTrigger>
            <CollapsibleContent>
                <div className="px-2 pb-2">
                    <div className="rounded-lg border border-border/30 overflow-hidden">
                        <Table>
                            <TableHeader className="bg-muted/50">
                                <TableRow className="hover:bg-transparent border-border/50">
                                    <TableHead className="font-semibold text-xs uppercase tracking-wider text-muted-foreground">对等体 IP</TableHead>
                                    <TableHead className="font-semibold text-xs uppercase tracking-wider text-muted-foreground">ASN</TableHead>
                                    <TableHead className="font-semibold text-xs uppercase tracking-wider text-muted-foreground">状态</TableHead>
                                    <TableHead className="font-semibold text-xs uppercase tracking-wider text-muted-foreground">节点选择器</TableHead>
                                    <TableHead className="font-semibold text-xs uppercase tracking-wider text-muted-foreground">运行时长</TableHead>
                                </TableRow>
                            </TableHeader>
                            <TableBody>
                                {peers.map((peer, idx) => {
                                    const isEstablished = peer.state === 'Established'
                                    return (
                                        <TableRow key={idx} className="hover:bg-muted/50 transition-colors border-border/30">
                                            <TableCell className="font-mono text-sm">
                                                {peer.peer_ip}
                                            </TableCell>
                                            <TableCell className="text-sm">
                                                <Badge variant="outline" className="font-mono text-xs">
                                                    {peer.as_number}
                                                </Badge>
                                            </TableCell>
                                            <TableCell>
                                                {isEstablished ? (
                                                    <Badge className="bg-green-500/10 text-green-600 dark:text-green-400 border-green-500/20 gap-1.5">
                                                        <CheckCircle2 className="h-3.5 w-3.5" />
                                                        已建立
                                                    </Badge>
                                                ) : (
                                                    <Badge className="bg-red-500/10 text-red-600 dark:text-red-400 border-red-500/20 gap-1.5">
                                                        <AlertCircle className="h-3.5 w-3.5" />
                                                        {peer.state || 'Down'}
                                                    </Badge>
                                                )}
                                            </TableCell>
                                            <TableCell className="font-mono text-xs text-muted-foreground max-w-[200px] truncate" title={peer.node_selector}>
                                                {peer.node_selector || '-'}
                                            </TableCell>
                                            <TableCell className="text-sm text-muted-foreground">
                                                {peer.uptime || '-'}
                                            </TableCell>
                                        </TableRow>
                                    )
                                })}
                            </TableBody>
                        </Table>
                    </div>
                </div>
            </CollapsibleContent>
        </Collapsible>
    )
}

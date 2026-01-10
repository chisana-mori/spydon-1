
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
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { useState } from "react"
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible"
import { ChevronDown, ChevronRight, CheckCircle2, AlertCircle } from "lucide-react"

interface BGPPeerListProps {
    peers: BGPPeerDetail[]
}

export function BGPPeerList({ peers }: BGPPeerListProps) {
    const globalPeers = peers.filter(p => p.scope === 'global')
    const nodeSpecificPeers = peers.filter(p => p.scope === 'node-specific')

    return (
        <Card>
            <CardHeader>
                <CardTitle>BGP 对等体</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
                <PeerGroup title="全局对等体" peers={globalPeers} defaultOpen={true} />
                <PeerGroup title="节点特定对等体" peers={nodeSpecificPeers} />
            </CardContent>
        </Card>
    )
}

function PeerGroup({ title, peers, defaultOpen = false }: { title: string, peers: BGPPeerDetail[], defaultOpen?: boolean }) {
    const [isOpen, setIsOpen] = useState(defaultOpen)

    if (peers.length === 0) return null

    return (
        <Collapsible open={isOpen} onOpenChange={setIsOpen} className="rounded-lg border bg-muted/30">
            <CollapsibleTrigger className="flex items-center justify-between w-full px-4 py-3 font-medium hover:bg-muted/50 rounded-t-lg">
                <div className="flex items-center gap-2">
                    {isOpen ? <ChevronDown className="h-4 w-4" /> : <ChevronRight className="h-4 w-4" />}
                    {title} <span className="text-muted-foreground ml-2">({peers.length})</span>
                </div>
            </CollapsibleTrigger>
            <CollapsibleContent>
                <div className="px-4 pb-4">
                    <Table>
                        <TableHeader>
                            <TableRow>
                                <TableHead>对等体 IP</TableHead>
                                <TableHead>ASN</TableHead>
                                <TableHead>状态</TableHead>
                                <TableHead>节点选择器</TableHead>
                                <TableHead>运行时长</TableHead>
                            </TableRow>
                        </TableHeader>
                        <TableBody>
                            {peers.map((peer, idx) => (
                                <TableRow key={idx}>
                                    <TableCell className="font-mono">{peer.peer_ip}</TableCell>
                                    <TableCell>{peer.as_number}</TableCell>
                                    <TableCell>
                                        <div className="flex items-center gap-2">
                                            {peer.state === 'Established' ? (
                                                <>
                                                    <span className="h-2 w-2 rounded-full bg-green-500" />
                                                    <span>已建立连接</span>
                                                </>
                                            ) : (
                                                <>
                                                    <span className="h-2 w-2 rounded-full bg-red-500 animate-pulse" />
                                                    <span className="text-red-500">{peer.state || 'Down'}</span>
                                                </>
                                            )}
                                        </div>
                                    </TableCell>
                                    <TableCell className="font-mono text-xs text-muted-foreground">
                                        {peer.node_selector}
                                    </TableCell>
                                    <TableCell>{peer.uptime}</TableCell>
                                </TableRow>
                            ))}
                        </TableBody>
                    </Table>
                </div>
            </CollapsibleContent>
        </Collapsible>
    )
}

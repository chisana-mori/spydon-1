
import {
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableHeader,
    TableRow,
} from "@/components/ui/table"
import { Badge } from "@/components/ui/badge"
import { Progress } from "@/components/ui/progress"
import { IPPoolDetail } from "@/types/calico"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { ChevronDown, ChevronUp } from "lucide-react"
import { useState } from "react"
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible"
import { Button } from "@/components/ui/button"

interface IPPoolTableProps {
    ipv4Pools: IPPoolDetail[]
    ipv6Pools: IPPoolDetail[]
}

export function IPPoolTable({ ipv4Pools, ipv6Pools }: IPPoolTableProps) {
    return (
        <div className="rounded-xl border bg-card text-card-foreground shadow">
            <div className="p-6 pb-3">
                <h3 className="font-semibold leading-none tracking-tight">IP 地址池</h3>
                <p className="text-sm text-muted-foreground mt-1">已分配 IP 网段的状态与使用情况</p>
            </div>
            <div className="p-6 pt-0">
                <Tabs defaultValue="ipv4" className="w-full">
                    <TabsList className="grid w-full grid-cols-2 mb-4">
                        <TabsTrigger value="ipv4">IPv4 地址池 ({ipv4Pools.length})</TabsTrigger>
                        <TabsTrigger value="ipv6">IPv6 地址池 ({ipv6Pools.length})</TabsTrigger>
                    </TabsList>

                    <TabsContent value="ipv4">
                        <PoolList pools={ipv4Pools} />
                    </TabsContent>
                    <TabsContent value="ipv6">
                        <PoolList pools={ipv6Pools} />
                    </TabsContent>
                </Tabs>
            </div>
        </div>
    )
}

function PoolList({ pools }: { pools: IPPoolDetail[] }) {
    if (pools.length === 0) {
        return <div className="text-center py-8 text-muted-foreground">未找到该版本的 IP 地址池。</div>
    }

    return (
        <Table>
            <TableHeader>
                <TableRow>
                    <TableHead>名称</TableHead>
                    <TableHead>CIDR</TableHead>
                    <TableHead>块大小</TableHead>
                    <TableHead>已分配 / 容量</TableHead>
                    <TableHead>使用率</TableHead>
                </TableRow>
            </TableHeader>
            <TableBody>
                {pools.map(pool => (
                    <PoolRow key={pool.name} pool={pool} />
                ))}
            </TableBody>
        </Table>
    )
}

function PoolRow({ pool }: { pool: IPPoolDetail }) {
    const [isOpen, setIsOpen] = useState(false)

    return (
        <>
            <TableRow className="cursor-pointer hover:bg-muted/50" onClick={() => setIsOpen(!isOpen)}>
                <TableCell className="font-medium">
                    <div className="flex items-center gap-2">
                        {isOpen ? <ChevronUp className="h-4 w-4 text-muted-foreground" /> : <ChevronDown className="h-4 w-4 text-muted-foreground" />}
                        {pool.name}
                        {pool.disabled && <Badge variant="secondary" className="ml-2 text-xs">已禁用</Badge>}
                    </div>
                </TableCell>
                <TableCell><Badge variant="outline" className="font-mono">{pool.cidr}</Badge></TableCell>
                <TableCell>/{pool.block_size}</TableCell>
                <TableCell className="text-sm">
                    {pool.allocated.toLocaleString()} / {pool.capacity.toLocaleString()}
                </TableCell>
                <TableCell className="w-[200px]">
                    <div className="space-y-1">
                        <div className="flex justify-between text-xs">
                            <span className={pool.allocation_rate > 90 ? "text-red-500" : "text-muted-foreground"}>
                                {pool.allocation_rate.toFixed(1)}%
                            </span>
                        </div>
                        <Progress
                            value={pool.allocation_rate}
                            className={`h-2 ${pool.allocation_rate > 90 ? "bg-red-500/20 [&>div]:bg-red-500" : "bg-blue-500/20 [&>div]:bg-blue-500"}`}
                        />
                    </div>
                </TableCell>
            </TableRow>
            {isOpen && (
                <TableRow className="bg-muted/30 hover:bg-muted/30">
                    <TableCell colSpan={5} className="p-4">
                        <div className="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
                            <div className="space-y-1">
                                <span className="text-xs text-muted-foreground block">封装模式</span>
                                <div className="flex gap-2">
                                    <Badge variant="secondary">{pool.ipip_mode}</Badge>
                                    <Badge variant="secondary">{pool.vxlan_mode}</Badge>
                                </div>
                            </div>
                            <div className="space-y-1">
                                <span className="text-xs text-muted-foreground block">NAT 出网</span>
                                <div>{pool.nat_outgoing ? "已启用" : "未启用"}</div>
                            </div>
                            <div className="space-y-1">
                                <span className="text-xs text-muted-foreground block">块大小</span>
                                <div>/{pool.block_size}</div>
                            </div>
                            <div className="space-y-1">
                                <span className="text-xs text-muted-foreground block">CIDR</span>
                                <div>{pool.cidr}</div>
                            </div>
                        </div>
                    </TableCell>
                </TableRow>
            )}
        </>
    )
}

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
import { ChevronDown, ChevronUp, Database, Globe2, CloudUpload, CheckCircle, XCircle } from "lucide-react"
import { useState } from "react"
import { cn } from "@/lib/utils"
import { Card } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Alert, AlertDescription } from "@/components/ui/alert"

interface IPPoolTableProps {
    ipv4Pools: IPPoolDetail[]
    ipv6Pools: IPPoolDetail[]
    clusterName: string
    onSyncWayne: () => void
    isSyncing: boolean
    syncResult: { created: number; updated: number; deleted: number } | null
}

export function IPPoolTable({ ipv4Pools, ipv6Pools, clusterName, onSyncWayne, isSyncing, syncResult }: IPPoolTableProps) {
    const totalPools = ipv4Pools.length + ipv6Pools.length
    const totalCapacity = [...ipv4Pools, ...ipv6Pools].reduce((sum, p) => sum + p.capacity, 0)
    const totalAllocated = [...ipv4Pools, ...ipv6Pools].reduce((sum, p) => sum + p.allocated, 0)
    const avgUsageRate = totalCapacity > 0 ? (totalAllocated / totalCapacity) * 100 : 0

    return (
        <Card className="border-l-4 border-l-blue-500 border-border/50">
            <div className="p-6 pb-4">
                <div className="flex items-start justify-between mb-6">
                    <div className="space-y-1">
                        <div className="flex items-center gap-3">
                            <div className="h-8 w-8 rounded-lg bg-blue-500/10 flex items-center justify-center">
                                <Database className="h-4 w-4 text-blue-500" />
                            </div>
                            <h3 className="text-lg font-semibold">IP 地址池</h3>
                        </div>
                        <p className="text-sm text-muted-foreground ml-11">已分配 IP 网段的状态与使用情况</p>
                    </div>

                    {/* Quick Stats + Sync Button */}
                    <div className="flex items-center gap-6 text-sm">
                        <div className="text-right">
                            <div className="text-xs text-muted-foreground">总池数</div>
                            <div className="text-lg font-semibold tabular-nums">{totalPools}</div>
                        </div>
                        <div className="text-right">
                            <div className="text-xs text-muted-foreground">总容量</div>
                            <div className="text-lg font-semibold tabular-nums">{totalCapacity.toLocaleString()}</div>
                        </div>
                        <div className="text-right">
                            <div className="text-xs text-muted-foreground">使用率</div>
                            <div className={cn(
                                "text-lg font-semibold tabular-nums",
                                avgUsageRate > 90 ? "text-red-500" : avgUsageRate > 70 ? "text-yellow-500" : "text-green-500"
                            )}>
                                {avgUsageRate.toFixed(1)}%
                            </div>
                        </div>
                        <div className="h-8 w-px bg-border" />
                        <Button
                            variant="outline"
                            size="sm"
                            onClick={onSyncWayne}
                            disabled={isSyncing}
                            className="gap-2"
                        >
                            <CloudUpload className={cn("h-4 w-4", isSyncing && "animate-pulse")} />
                            {isSyncing ? "同步中..." : "同步 Wayne"}
                        </Button>
                    </div>
                </div>

                {/* Sync Result Alert */}
                {syncResult && (
                    <Alert className="mb-4 border-green-500/50 bg-green-500/5">
                        <CheckCircle className="h-4 w-4 text-green-500" />
                        <AlertDescription className="flex items-center gap-6 text-sm">
                            <div className="flex items-center gap-1.5">
                                <span className="text-muted-foreground">创建:</span>
                                <span className="font-semibold text-green-600 dark:text-green-400">{syncResult.created}</span>
                            </div>
                            <div className="flex items-center gap-1.5">
                                <span className="text-muted-foreground">更新:</span>
                                <span className="font-semibold text-blue-600 dark:text-blue-400">{syncResult.updated}</span>
                            </div>
                            <div className="flex items-center gap-1.5">
                                <span className="text-muted-foreground">删除:</span>
                                <span className="font-semibold text-orange-600 dark:text-orange-400">{syncResult.deleted}</span>
                            </div>
                        </AlertDescription>
                    </Alert>
                )}

                <Tabs defaultValue="ipv4" className="w-full">
                    <TabsList className="grid w-full grid-cols-2 h-9 bg-muted/50 p-1">
                        <TabsTrigger value="ipv4" className="gap-2">
                            <Globe2 className="h-4 w-4" />
                            IPv4 ({ipv4Pools.length})
                        </TabsTrigger>
                        <TabsTrigger value="ipv6" className="gap-2">
                            <Globe2 className="h-4 w-4" />
                            IPv6 ({ipv6Pools.length})
                        </TabsTrigger>
                    </TabsList>

                    <TabsContent value="ipv4" className="mt-4">
                        <PoolList pools={ipv4Pools} />
                    </TabsContent>
                    <TabsContent value="ipv6" className="mt-4">
                        <PoolList pools={ipv6Pools} />
                    </TabsContent>
                </Tabs>
            </div>
        </Card>
    )
}

function PoolList({ pools }: { pools: IPPoolDetail[] }) {
    if (pools.length === 0) {
        return (
            <div className="flex flex-col items-center justify-center py-16 text-muted-foreground">
                <Database className="h-10 w-10 opacity-50 mb-4" />
                <p className="text-sm">未找到该版本的 IP 地址池</p>
            </div>
        )
    }

    return (
        <div className="rounded-lg border border-border/30 overflow-hidden">
            <Table>
                <TableHeader className="bg-muted/50">
                    <TableRow className="hover:bg-transparent border-border/50">
                        <TableHead className="font-semibold text-xs uppercase tracking-wider text-muted-foreground w-10"></TableHead>
                        <TableHead className="font-semibold text-xs uppercase tracking-wider text-muted-foreground">名称</TableHead>
                        <TableHead className="font-semibold text-xs uppercase tracking-wider text-muted-foreground">CIDR</TableHead>
                        <TableHead className="font-semibold text-xs uppercase tracking-wider text-muted-foreground">Wayne</TableHead>
                        <TableHead className="font-semibold text-xs uppercase tracking-wider text-muted-foreground">小类</TableHead>
                        <TableHead className="font-semibold text-xs uppercase tracking-wider text-muted-foreground">块大小</TableHead>
                        <TableHead className="font-semibold text-xs uppercase tracking-wider text-muted-foreground">已分配 / 容量</TableHead>
                        <TableHead className="font-semibold text-xs uppercase tracking-wider text-muted-foreground w-[200px]">使用率</TableHead>
                    </TableRow>
                </TableHeader>
                <TableBody>
                    {pools.map(pool => (
                        <PoolRow key={pool.name} pool={pool} />
                    ))}
                </TableBody>
            </Table>
        </div>
    )
}

function PoolRow({ pool }: { pool: IPPoolDetail }) {
    const [isOpen, setIsOpen] = useState(false)
    const isCritical = pool.allocation_rate > 90
    const isWarning = pool.allocation_rate > 70 && pool.allocation_rate <= 90

    return (
        <>
            <TableRow className="cursor-pointer hover:bg-muted/50 transition-colors border-border/30" onClick={() => setIsOpen(!isOpen)}>
                <TableCell className="w-10">
                    {isOpen ? <ChevronUp className="h-4 w-4 text-muted-foreground" /> : <ChevronDown className="h-4 w-4 text-muted-foreground" />}
                </TableCell>
                <TableCell className="font-medium">
                    <div className="flex items-center gap-2">
                        <span className="text-sm">{pool.name}</span>
                        {pool.disabled && (
                            <Badge variant="secondary" className="ml-2 text-xs">已禁用</Badge>
                        )}
                    </div>
                </TableCell>
                <TableCell>
                    <Badge variant="outline" className={cn(
                        "font-mono text-xs",
                        isCritical && "border-red-500/30 text-red-500",
                        isWarning && "border-yellow-500/30 text-yellow-500"
                    )}>
                        {pool.cidr}
                    </Badge>
                </TableCell>
                <TableCell>
                    {pool.wayne_enabled === "true" ? (
                        <Badge className="text-xs bg-green-500/10 text-green-600 dark:text-green-400 hover:bg-green-500/20 shadow-none border-green-500/20">已启用</Badge>
                    ) : pool.wayne_enabled === "false" ? (
                        <Badge variant="secondary" className="text-xs">未启用</Badge>
                    ) : (
                        <span className="text-xs text-muted-foreground">-</span>
                    )}
                </TableCell>
                <TableCell>
                    <span className="text-sm">{pool.subfunction || "-"}</span>
                </TableCell>
                <TableCell className="text-sm text-muted-foreground">/{pool.block_size}</TableCell>
                <TableCell className="text-sm font-medium tabular-nums">
                    {pool.allocated.toLocaleString()} / {pool.capacity.toLocaleString()}
                </TableCell>
                <TableCell className="w-[200px]">
                    <div className="space-y-1.5">
                        <div className="flex justify-between text-xs">
                            <span className={cn(
                                isCritical ? "text-red-500" : isWarning ? "text-yellow-500" : "text-muted-foreground"
                            )}>
                                {pool.allocation_rate.toFixed(1)}%
                            </span>
                        </div>
                        <Progress
                            value={pool.allocation_rate}
                            className={cn(
                                "h-2",
                                isCritical && "[&>div]:bg-red-500",
                                isWarning && "[&>div]:bg-yellow-500",
                                !isCritical && !isWarning && "[&>div]:bg-blue-500"
                            )}
                        />
                    </div>
                </TableCell>
            </TableRow>
            {isOpen && (
                <TableRow className="bg-muted/20 hover:bg-muted/20 border-border/20">
                    <TableCell colSpan={5} className="p-4">
                        <div className="grid grid-cols-2 md:grid-cols-4 gap-6 text-sm">
                            <div className="space-y-2">
                                <span className="text-xs text-muted-foreground">封装模式</span>
                                <div className="flex gap-2">
                                    <Badge variant="secondary" className="text-xs">{pool.ipip_mode}</Badge>
                                    <Badge variant="secondary" className="text-xs">{pool.vxlan_mode}</Badge>
                                </div>
                            </div>
                            <div className="space-y-2">
                                <span className="text-xs text-muted-foreground">NAT 出网</span>
                                <div>
                                    {pool.nat_outgoing ? (
                                        <Badge className="text-xs bg-green-500/10 text-green-600 dark:text-green-400">已启用</Badge>
                                    ) : (
                                        <Badge variant="secondary" className="text-xs">未启用</Badge>
                                    )}
                                </div>
                            </div>
                            <div className="space-y-2">
                                <span className="text-xs text-muted-foreground">块大小</span>
                                <span className="font-mono text-xs">/{pool.block_size}</span>
                            </div>
                            <div className="space-y-2">
                                <span className="text-xs text-muted-foreground">CIDR 范围</span>
                                <span className="font-mono text-xs">{pool.cidr}</span>
                            </div>
                        </div>
                    </TableCell>
                </TableRow>
            )}
        </>
    )
}

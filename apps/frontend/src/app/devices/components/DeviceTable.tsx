'use client'

import {
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableHeader,
    TableRow,
} from "@/components/ui/table"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Eye, Pencil, Server, Laptop, Cpu, Globe, Database, Star, Network } from 'lucide-react'
import { NavyDevice } from "@/types/navy"
import { formatDistanceToNow } from 'date-fns'
import { zhCN } from 'date-fns/locale'
import { cn } from "@/lib/utils"

interface DeviceTableProps {
    devices: NavyDevice[]
    isLoading: boolean
    onEdit: (device: NavyDevice) => void
    onView: (device: NavyDevice) => void
}

export function DeviceTable({ devices, isLoading, onEdit, onView }: DeviceTableProps) {
    const getStatusBadge = (status: string) => {
        switch (status?.toLowerCase()) {
            case 'online':
            case 'running':
            case '活跃':
                return <Badge className="bg-emerald-50 text-emerald-700 border-emerald-200 hover:bg-emerald-100">运行中</Badge>
            case 'offline':
            case '离线':
                return <Badge variant="outline" className="bg-rose-50 text-rose-700 border-rose-200 hover:bg-rose-100">离线</Badge>
            default:
                return <Badge variant="secondary">{status || '未知'}</Badge>
        }
    }

    const getArchIcon = (arch: string) => {
        if (arch?.toLowerCase().includes('arm') || arch?.toLowerCase().includes('aarch64')) {
            return <Cpu className="h-4 w-4 mr-1 text-orange-500" />
        }
        return <Cpu className="h-4 w-4 mr-1 text-blue-500" />
    }

    const getRowClassName = (device: NavyDevice) => {
        // Special device = yellow background
        if (device.is_special) {
            return "bg-amber-50/70 hover:bg-amber-100/70 border-l-2 border-l-amber-400"
        }
        // Has cluster = green background
        if (device.cluster && device.cluster.trim() !== '') {
            return "bg-emerald-50/50 hover:bg-emerald-100/50 border-l-2 border-l-emerald-400"
        }
        return "hover:bg-muted/50"
    }

    return (
        <div className="rounded-lg border bg-card shadow-sm overflow-hidden">
            <Table>
                <TableHeader>
                    <TableRow className="bg-muted/30">
                        <TableHead className="w-[180px] font-semibold">设备信息</TableHead>
                        <TableHead className="font-semibold">IP 地址</TableHead>
                        <TableHead className="font-semibold">架构/位置</TableHead>
                        <TableHead className="font-semibold">所属集群</TableHead>
                        <TableHead className="font-semibold">网络区域</TableHead>
                        <TableHead className="font-semibold">状态</TableHead>
                        <TableHead className="font-semibold">创建时间</TableHead>
                        <TableHead className="text-right font-semibold">操作</TableHead>
                    </TableRow>
                </TableHeader>
                <TableBody>
                    {isLoading ? (
                        Array.from({ length: 5 }).map((_, i) => (
                            <TableRow key={i}>
                                <TableCell><div className="h-4 w-24 bg-muted animate-pulse rounded" /></TableCell>
                                <TableCell><div className="h-4 w-32 bg-muted animate-pulse rounded" /></TableCell>
                                <TableCell><div className="h-4 w-16 bg-muted animate-pulse rounded" /></TableCell>
                                <TableCell><div className="h-4 w-20 bg-muted animate-pulse rounded" /></TableCell>
                                <TableCell><div className="h-4 w-16 bg-muted animate-pulse rounded" /></TableCell>
                                <TableCell><div className="h-4 w-16 bg-muted animate-pulse rounded" /></TableCell>
                                <TableCell><div className="h-4 w-24 bg-muted animate-pulse rounded" /></TableCell>
                                <TableCell className="text-right"><div className="h-8 w-16 bg-muted animate-pulse rounded ml-auto" /></TableCell>
                            </TableRow>
                        ))
                    ) : devices.length === 0 ? (
                        <TableRow>
                            <TableCell colSpan={8} className="h-32 text-center text-muted-foreground">
                                <div className="flex flex-col items-center justify-center space-y-3">
                                    <Server className="h-10 w-10 opacity-20" />
                                    <span className="text-sm">未发现匹配的设备</span>
                                </div>
                            </TableCell>
                        </TableRow>
                    ) : (
                        devices.map((device) => (
                            <TableRow key={device.id} className={cn("transition-colors", getRowClassName(device))}>
                                <TableCell className="font-medium">
                                    <div className="flex flex-col">
                                        <span className="flex items-center">
                                            {device.is_special ? (
                                                <Star className="h-4 w-4 mr-2 text-amber-500 fill-amber-500" />
                                            ) : (
                                                <Laptop className="h-4 w-4 mr-2 text-muted-foreground" />
                                            )}
                                            <span className="font-mono text-sm">{device.ci_code}</span>
                                        </span>
                                        {device.group && (
                                            <span className="text-xs text-muted-foreground mt-1 ml-6 bg-slate-100 dark:bg-slate-800 px-1.5 py-0.5 rounded w-fit">
                                                {device.group}
                                            </span>
                                        )}
                                    </div>
                                </TableCell>
                                <TableCell>
                                    <span className="font-mono text-sm bg-slate-100 dark:bg-slate-800 px-2 py-1 rounded">
                                        {device.ip}
                                    </span>
                                </TableCell>
                                <TableCell>
                                    <div className="flex flex-col space-y-1 text-sm text-muted-foreground">
                                        <span className="flex items-center">
                                            {getArchIcon(device.arch_type)}
                                            <span className="font-medium text-foreground">{device.arch_type || '-'}</span>
                                        </span>
                                        <span className="flex items-center text-xs">
                                            <Globe className="h-3 w-3 mr-1" />
                                            {device.idc || '-'} {device.room && `/ ${device.room}`}
                                        </span>
                                    </div>
                                </TableCell>
                                <TableCell>
                                    {device.cluster ? (
                                        <div className="flex flex-col">
                                            <span className="flex items-center text-sm font-medium">
                                                <Database className="h-4 w-4 mr-2 text-emerald-500" />
                                                {device.cluster}
                                            </span>
                                            {device.role && (
                                                <span className="text-xs text-muted-foreground ml-6">
                                                    {device.role}
                                                </span>
                                            )}
                                        </div>
                                    ) : (
                                        <span className="text-muted-foreground">-</span>
                                    )}
                                </TableCell>
                                <TableCell>
                                    {device.net_zone ? (
                                        <span className="flex items-center text-sm">
                                            <Network className="h-3.5 w-3.5 mr-1.5 text-muted-foreground" />
                                            {device.net_zone}
                                        </span>
                                    ) : (
                                        <span className="text-muted-foreground">-</span>
                                    )}
                                </TableCell>
                                <TableCell>{getStatusBadge(device.status)}</TableCell>
                                <TableCell className="text-sm text-muted-foreground">
                                    {device.created_at ? formatDistanceToNow(new Date(device.created_at), { addSuffix: true, locale: zhCN }) : '-'}
                                </TableCell>
                                <TableCell className="text-right">
                                    <div className="flex justify-end gap-1">
                                        <Button
                                            variant="ghost"
                                            size="sm"
                                            onClick={() => onView(device)}
                                            className="h-8 px-2 text-xs"
                                        >
                                            <Eye className="h-3.5 w-3.5 mr-1" />
                                            详情
                                        </Button>
                                        <Button
                                            variant="ghost"
                                            size="sm"
                                            onClick={() => onEdit(device)}
                                            className="h-8 px-2 text-xs"
                                        >
                                            <Pencil className="h-3.5 w-3.5 mr-1" />
                                            编辑
                                        </Button>
                                    </div>
                                </TableCell>
                            </TableRow>
                        ))
                    )}
                </TableBody>
            </Table>
        </div>
    )
}

'use client'

import { useState, useCallback } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import {
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableHeader,
    TableRow,
} from '@/components/ui/table'
import {
    Pagination,
    PaginationContent,
    PaginationEllipsis,
    PaginationItem,
    PaginationLink,
    PaginationNext,
    PaginationPrevious,
} from '@/components/ui/pagination'
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from '@/components/ui/select'
import { Database, RefreshCw, Eye, AlertCircle, CheckCircle, Search, ArrowUpDown, Copy, Check } from 'lucide-react'
import RobustaAPI from '@/lib/api'
import type { F5Info } from '@/types/f5'
import { toast } from 'sonner'
import { cn } from '@/lib/utils'
import { F5DetailSheet } from './components/F5DetailSheet'
import { SimpleQueryPanel } from './components/SimpleQueryPanel'

export default function F5ManagementPage() {
    // State
    const [page, setPage] = useState(1)
    const [pageSize, setPageSize] = useState(20)
    const [keyword, setKeyword] = useState('')

    // Sorting
    const [sortConfig, setSortConfig] = useState<{ key: string; direction: 'asc' | 'desc' }>({
        key: 'id',
        direction: 'desc'
    })

    // Selected F5 for detail view
    const [selectedF5, setSelectedF5] = useState<F5Info | null>(null)
    const [sheetOpen, setSheetOpen] = useState(false)
    const [copiedId, setCopiedId] = useState<string | null>(null)

    // Fetch F5 data
    const { data: f5Response, isLoading, refetch } = useQuery({
        queryKey: ['f5-list', page, pageSize, keyword, sortConfig],
        queryFn: () => RobustaAPI.listF5Infos({
            page,
            size: pageSize,
            keyword: keyword,
            sort_by: sortConfig.key,
            sort_order: sortConfig.direction === 'asc' ? 'ascend' : 'descend'
        }),
    })

    const handleSort = (key: string) => {
        setSortConfig(current => ({
            key,
            direction: current.key === key && current.direction === 'asc' ? 'desc' : 'asc'
        }))
    }

    const copyToClipboard = async (text: string, id: string) => {
        if (!text) return
        try {
            await navigator.clipboard.writeText(text)
            setCopiedId(id)
            toast.success('已复制到剪贴板')
            setTimeout(() => setCopiedId(null), 2000)
        } catch (err) {
            toast.error('复制失败')
        }
    }

    const totalPages = Math.ceil((f5Response?.total || 0) / pageSize)

    const handleRefresh = useCallback(() => {
        refetch()
    }, [refetch])

    const handleRowClick = useCallback((f5: F5Info) => {
        setSelectedF5(f5)
        setSheetOpen(true)
    }, [])

    const handleSimpleSearch = useCallback((newKeyword: string) => {
        setKeyword(newKeyword)
        setPage(1)
    }, [])

    const getStatusBadge = (status: string) => {
        const lower = status?.toLowerCase() || ''
        if (lower === 'active' || lower === 'running' || lower === 'online') {
            return <Badge variant="outline" className="gap-1.5 bg-emerald-50/80 text-emerald-700 border-emerald-200/50 dark:bg-emerald-950/30 dark:text-emerald-400 dark:border-emerald-800/50 font-medium shadow-sm"><CheckCircle className="h-3 w-3" />{status}</Badge>
        }
        if (lower === 'inactive' || lower === 'offline' || lower === 'stopped') {
            return <Badge variant="outline" className="gap-1.5 bg-rose-50/80 text-rose-700 border-rose-200/50 dark:bg-rose-950/30 dark:text-rose-400 dark:border-rose-800/50 font-medium shadow-sm"><AlertCircle className="h-3 w-3" />{status}</Badge>
        }
        if (lower === 'maintenance') {
            return <Badge variant="outline" className="gap-1.5 bg-amber-50/80 text-amber-700 border-amber-200/50 dark:bg-amber-950/30 dark:text-amber-400 dark:border-amber-800/50 font-medium shadow-sm">{status}</Badge>
        }
        return <Badge variant="outline" className="bg-slate-50/80 text-slate-600 border-slate-200/50 dark:bg-slate-900/50 dark:text-slate-400 dark:border-slate-700/50 font-medium">{status}</Badge>
    }

    const SortableHeader = ({ label, sortKey, className }: { label: string, sortKey: string, className?: string }) => (
        <TableHead
            className={cn("cursor-pointer hover:bg-slate-100/50 dark:hover:bg-slate-800/30 transition-all duration-200 select-none group", className)}
            onClick={() => handleSort(sortKey)}
        >
            <div className="flex items-center gap-1.5">
                {label}
                <ArrowUpDown className={cn(
                    "h-3 w-3 transition-all duration-200",
                    sortConfig.key === sortKey ? "opacity-100 text-blue-600 dark:text-blue-400" : "opacity-0 group-hover:opacity-40 text-muted-foreground"
                )} />
            </div>
        </TableHead>
    )

    return (
        <div className="space-y-6 animate-in fade-in duration-500">
            {/* Header */}
            <div className="flex items-center justify-between">
                <div className="flex items-center gap-3">
                    <div className="p-2.5 rounded-xl bg-gradient-to-br from-blue-50 to-indigo-50 text-blue-600 ring-1 ring-blue-200/50 shadow-sm dark:from-blue-950/40 dark:to-indigo-950/40 dark:ring-blue-800/50">
                        <Database className="h-6 w-6" />
                    </div>
                    <div>
                        <h1 className="text-2xl font-bold tracking-tight bg-clip-text text-transparent bg-gradient-to-r from-slate-900 via-slate-800 to-slate-700 dark:from-slate-100 dark:via-slate-200 dark:to-slate-300">
                            F5 负载均衡器管理
                        </h1>
                        <p className="text-sm text-muted-foreground mt-0.5">
                            管理和监控 F5 负载均衡器资产
                        </p>
                    </div>
                </div>
                <div className="flex items-center gap-2">
                    <Button variant="outline" size="sm" onClick={handleRefresh} disabled={isLoading} className="shadow-sm hover:shadow-md hover:bg-muted/50 transition-all border-dashed">
                        <RefreshCw className={cn("h-4 w-4 mr-2", isLoading && "animate-spin")} />
                        刷新列表
                    </Button>
                </div>
            </div>

            {/* Query Panel with Stats */}
            <Card className="border-none shadow-sm bg-gradient-to-b from-slate-50/50 to-slate-50/30 ring-1 ring-slate-200/50 dark:from-slate-900/20 dark:to-slate-900/10 dark:ring-slate-700/50">
                <CardContent className="p-6">
                    <div className="flex items-start justify-between mb-4">
                        <div className="flex items-center gap-2">
                            <div className="p-1.5 rounded-lg bg-slate-100/80 text-slate-600 dark:bg-slate-800/50 dark:text-slate-400">
                                <Search className="h-3.5 w-3.5" />
                            </div>
                            <h2 className="text-sm font-medium">查询筛选</h2>
                        </div>
                        <div className="flex items-center gap-4 text-sm">
                            <div className="flex items-center gap-2">
                                <span className="text-muted-foreground">共</span>
                                <Badge variant="secondary" className="font-mono bg-blue-50/80 text-blue-700 border-blue-200/50 dark:bg-blue-950/30 dark:text-blue-400 dark:border-blue-800/50">
                                    {(f5Response?.total || 0).toLocaleString()}
                                </Badge>
                                <span className="text-muted-foreground">条记录</span>
                            </div>
                        </div>
                    </div>
                    <SimpleQueryPanel
                        onSearch={handleSimpleSearch}
                        isLoading={isLoading}
                    />
                </CardContent>
            </Card>

            {/* Results Table */}
            <Card className="border-none shadow-md ring-1 ring-slate-200/50 overflow-hidden dark:ring-slate-700/50">
                <CardContent className="p-0">
                    <div className="rounded-md">
                        <Table>
                            <TableHeader className="bg-slate-50/80 dark:bg-slate-900/30">
                                <TableRow className="hover:bg-transparent">
                                    <SortableHeader label="名称" sortKey="name" />
                                    <SortableHeader label="VIP" sortKey="vip" />
                                    <SortableHeader label="端口" sortKey="port" />
                                    <SortableHeader label="AppID" sortKey="appid" />
                                    <SortableHeader label="实例组" sortKey="instance_group" />
                                    <SortableHeader label="状态" sortKey="status" />
                                    <SortableHeader label="Pool状态" sortKey="pool_status" />
                                    <SortableHeader label="集群" sortKey="cluster_name" />
                                    <SortableHeader label="忽略" sortKey="ignored" />
                                    <TableHead className="text-right w-[100px]">操作</TableHead>
                                </TableRow>
                            </TableHeader>
                            <TableBody>
                                {isLoading ? (
                                    <TableRow>
                                        <TableCell colSpan={10} className="h-32 text-center text-muted-foreground">
                                            <div className="flex flex-col items-center gap-2">
                                                <RefreshCw className="h-6 w-6 animate-spin" />
                                                <span>加载中...</span>
                                            </div>
                                        </TableCell>
                                    </TableRow>
                                ) : !f5Response?.list || f5Response.list.length === 0 ? (
                                    <TableRow>
                                        <TableCell colSpan={10} className="h-32 text-center text-muted-foreground">
                                            暂无数据
                                        </TableCell>
                                    </TableRow>
                                ) : (
                                    f5Response.list.map((f5) => (
                                        <TableRow
                                            key={f5.id}
                                            className="cursor-pointer hover:bg-slate-50/80 dark:hover:bg-slate-800/20 transition-all duration-200 group"
                                            onClick={() => handleRowClick(f5)}
                                        >
                                            <TableCell className="font-medium">
                                                <div className="flex items-center gap-2 group/name">
                                                    <span className="truncate max-w-[180px]" title={f5.name}>{f5.name}</span>
                                                    <Button
                                                        variant="ghost"
                                                        size="icon"
                                                        className="h-6 w-6 opacity-0 group-hover/name:opacity-100 transition-opacity"
                                                        onClick={(e) => {
                                                            e.stopPropagation()
                                                            copyToClipboard(f5.name, `name-${f5.id}`)
                                                        }}
                                                    >
                                                        {copiedId === `name-${f5.id}` ? (
                                                            <Check className="h-3 w-3 text-green-500" />
                                                        ) : (
                                                            <Copy className="h-3 w-3 text-muted-foreground" />
                                                        )}
                                                    </Button>
                                                </div>
                                            </TableCell>
                                            <TableCell className="font-mono text-xs">
                                                <div className="flex items-center gap-2 group/vip">
                                                    <span>{f5.vip || '-'}</span>
                                                    {f5.vip && (
                                                        <Button
                                                            variant="ghost"
                                                            size="icon"
                                                            className="h-6 w-6 opacity-0 group-hover/vip:opacity-100 transition-opacity"
                                                            onClick={(e) => {
                                                                e.stopPropagation()
                                                                copyToClipboard(f5.vip, `vip-${f5.id}`)
                                                            }}
                                                        >
                                                            {copiedId === `vip-${f5.id}` ? (
                                                                <Check className="h-3 w-3 text-green-500" />
                                                            ) : (
                                                                <Copy className="h-3 w-3 text-muted-foreground" />
                                                            )}
                                                        </Button>
                                                    )}
                                                </div>
                                            </TableCell>
                                            <TableCell>{f5.port || '-'}</TableCell>
                                            <TableCell>{f5.appid || '-'}</TableCell>
                                            <TableCell>{f5.instance_group || '-'}</TableCell>
                                            <TableCell>{f5.status ? getStatusBadge(f5.status) : '-'}</TableCell>
                                            <TableCell>{f5.pool_status ? getStatusBadge(f5.pool_status) : '-'}</TableCell>
                                            <TableCell>
                                                {f5.cluster_name ? (
                                                    <Badge variant="secondary" className="font-normal border-slate-200/50 bg-slate-100/80 text-slate-700 dark:bg-slate-800/50 dark:text-slate-300 dark:border-slate-700/50 hover:bg-slate-200/80 dark:hover:bg-slate-700/50 transition-colors">
                                                        {f5.cluster_name}
                                                    </Badge>
                                                ) : '-'}
                                            </TableCell>
                                            <TableCell className="align-middle">
                                                <div className="flex items-center">
                                                    {f5.ignored ? (
                                                        <Badge variant="outline" className="text-amber-700 bg-amber-50/80 border-amber-200/50 dark:bg-amber-950/30 dark:text-amber-400 dark:border-amber-800/50 h-6 flex items-center justify-center px-2 font-medium shadow-sm">是</Badge>
                                                    ) : (
                                                        <Badge variant="outline" className="text-slate-500 bg-slate-50/80 border-slate-200/50 dark:bg-slate-900/50 dark:text-slate-400 dark:border-slate-700/50 h-6 flex items-center justify-center px-2 font-medium">否</Badge>
                                                    )}
                                                </div>
                                            </TableCell>
                                            <TableCell className="text-right">
                                                <Button
                                                    variant="ghost"
                                                    size="sm"
                                                    onClick={(e) => {
                                                        e.stopPropagation()
                                                        handleRowClick(f5)
                                                    }}
                                                    className="opacity-0 group-hover:opacity-100 transition-all duration-200 hover:bg-blue-50/80 hover:text-blue-700 dark:hover:bg-blue-950/30 dark:hover:text-blue-400"
                                                >
                                                    <Eye className="h-4 w-4 mr-1" />
                                                    详情
                                                </Button>
                                            </TableCell>
                                        </TableRow>
                                    ))
                                )}
                            </TableBody>
                        </Table>
                    </div>
                </CardContent>
            </Card>

            {/* Pagination */}
            {(f5Response?.total || 0) > 0 && (
                <div className="flex items-center justify-between py-4 px-2">
                    <div className="flex items-center gap-6">
                        <p className="text-sm text-muted-foreground">
                            共 {f5Response?.total || 0} 条记录
                        </p>
                        <div className="flex items-center gap-2 text-sm text-muted-foreground">
                            <span>每页</span>
                            <Select
                                value={pageSize.toString()}
                                onValueChange={(value) => {
                                    setPageSize(Number(value))
                                    setPage(1)
                                }}
                            >
                                <SelectTrigger className="h-8 w-[80px]">
                                    <SelectValue placeholder={pageSize.toString()} />
                                </SelectTrigger>
                                <SelectContent side="top">
                                    {[10, 20, 50, 100].map((size) => (
                                        <SelectItem key={size} value={size.toString()}>
                                            {size}
                                        </SelectItem>
                                    ))}
                                </SelectContent>
                            </Select>
                            <span>条</span>
                        </div>
                    </div>
                    <Pagination className="w-auto mx-0">
                        <PaginationContent>
                            <PaginationItem>
                                <PaginationPrevious
                                    onClick={() => setPage(p => Math.max(1, p - 1))}
                                    className={cn("cursor-pointer", page <= 1 && "pointer-events-none opacity-50")}
                                />
                            </PaginationItem>

                            {(() => {
                                const pages: (number | string)[] = [];
                                const maxVisible = 5;

                                if (totalPages <= maxVisible) {
                                    for (let i = 1; i <= totalPages; i++) pages.push(i);
                                } else {
                                    pages.push(1);
                                    if (page > 3) pages.push('ellipsis-start');

                                    const start = Math.max(2, page - 1);
                                    const end = Math.min(totalPages - 1, page + 1);

                                    for (let i = start; i <= end; i++) {
                                        if (i > 1 && i < totalPages) pages.push(i);
                                    }

                                    if (page < totalPages - 2) pages.push('ellipsis-end');
                                    if (totalPages > 1) pages.push(totalPages);
                                }

                                return pages.map((p, i) => (
                                    <PaginationItem key={i}>
                                        {typeof p === 'number' ? (
                                            <PaginationLink
                                                isActive={page === p}
                                                onClick={() => setPage(p)}
                                                className="cursor-pointer"
                                            >
                                                {p}
                                            </PaginationLink>
                                        ) : (
                                            <PaginationEllipsis />
                                        )}
                                    </PaginationItem>
                                ));
                            })()}

                            <PaginationItem>
                                <PaginationNext
                                    onClick={() => setPage(p => Math.min(totalPages, p + 1))}
                                    className={cn("cursor-pointer", page >= totalPages && "pointer-events-none opacity-50")}
                                />
                            </PaginationItem>
                        </PaginationContent>
                    </Pagination>
                </div>
            )}

            {/* Detail Sheet */}
            <F5DetailSheet
                f5={selectedF5}
                open={sheetOpen}
                onOpenChange={(open) => {
                    setSheetOpen(open)
                    if (!open) setTimeout(() => setSelectedF5(null), 300)
                }}
                onUpdate={handleRefresh}
            />
        </div>
    )
}

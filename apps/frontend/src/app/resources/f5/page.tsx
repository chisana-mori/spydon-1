'use client'

import { useState, useMemo, useCallback } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
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
import { Server, Search, RefreshCw, Database, Eye, AlertCircle, CheckCircle, Filter } from 'lucide-react'
import RobustaAPI from '@/lib/api'
import type { F5Info } from '@/types/f5'
import { toast } from 'sonner'
import { cn } from '@/lib/utils'
import { F5DetailSheet } from './components/F5DetailSheet'

export default function F5ManagementPage() {
    // State
    const [page, setPage] = useState(1)
    const [pageSize] = useState(20)
    const [searchMode, setSearchMode] = useState<'simple' | 'advanced'>('simple')
    const [keyword, setKeyword] = useState('')
    const [batchInput, setBatchInput] = useState('')

    // Advanced filters
    const [filters, setFilters] = useState({
        name: '',
        vip: '',
        port: '',
        appid: '',
        status: '',
        cluster_name: '',
    })

    // Selected F5 for detail view
    const [selectedF5, setSelectedF5] = useState<F5Info | null>(null)
    const [sheetOpen, setSheetOpen] = useState(false)

    // Build query filter
    const queryFilters = useMemo(() => {
        if (searchMode === 'simple') {
            // 简单模式：使用 keyword 搜索（可能是多行）
            const lines = keyword.split(/[\n,;]+/).map(l => l.trim()).filter(Boolean)
            if (lines.length === 1) {
                // 单行：模糊搜索 name/vip
                return { name: lines[0] }
            }
            // 多行：提取第一行作为name模糊搜索（后续会在前端做匹配排序）
            return lines.length > 0 ? { name: lines[0] } : {}
        } else {
            // 高级模式：使用 filters
            return Object.fromEntries(
                Object.entries(filters).filter(([_, v]) => v !== '')
            )
        }
    }, [searchMode, keyword, filters])

    // Fetch F5 data
    const { data: f5Response, isLoading, refetch } = useQuery({
        queryKey: ['f5-list', page, pageSize, queryFilters],
        queryFn: () => RobustaAPI.listF5Infos({
            page,
            size: pageSize,
            ...queryFilters,
        }),
    })

    // Process batch search results (multi-line matching with order)
    const processedF5List = useMemo(() => {
        const rawList = f5Response?.list || []

        if (searchMode !== 'simple' || !keyword) {
            return rawList
        }

        const lines = keyword.split(/[\n,;]+/).map(l => l.trim()).filter(Boolean)
        if (lines.length <= 1) {
            return rawList
        }

        // Multi-line mode: match and order
        const result: (F5Info & { isExactMatch?: boolean; isMissing?: boolean; originalKeyword?: string })[] = []
        const seenIds = new Set<number>()

        lines.forEach(line => {
            const lowerLine = line.toLowerCase()

            // Find matches in results
            const matches = rawList.filter(f5 => {
                const name = f5.name?.toLowerCase() || ''
                const vip = f5.vip?.toLowerCase() || ''
                const appid = f5.appid?.toLowerCase() || ''
                return name.includes(lowerLine) || vip.includes(lowerLine) || appid.includes(lowerLine)
            })

            if (matches.length > 0) {
                // Sort by exact match
                const sortedMatches = [...matches].sort((a, b) => {
                    const aExact = (a.name?.toLowerCase() === lowerLine || a.vip?.toLowerCase() === lowerLine) ? 1 : 0
                    const bExact = (b.name?.toLowerCase() === lowerLine || b.vip?.toLowerCase() === lowerLine) ? 1 : 0
                    return bExact - aExact
                })

                sortedMatches.forEach(match => {
                    const isExact = (match.name?.toLowerCase() === lowerLine || match.vip?.toLowerCase() === lowerLine)

                    if (!seenIds.has(match.id)) {
                        result.push({
                            ...match,
                            isExactMatch: isExact,
                            originalKeyword: line,
                        })
                        seenIds.add(match.id)
                    }
                })
            } else {
                // Missing entry - create placeholder
                result.push({
                    id: -Math.random(),
                    name: line,
                    vip: '',
                    port: '',
                    appid: '',
                    instance_group: '',
                    status: '',
                    pool_name: '',
                    pool_status: '',
                    pool_members: '',
                    domains: '',
                    grafana_params: '',
                    ignored: false,
                    created_at: '',
                    updated_at: '',
                    isMissing: true,
                    originalKeyword: line,
                } as any)
            }
        })

        return result
    }, [f5Response?.list, searchMode, keyword])

    const displayTotal = useMemo(() => {
        if (searchMode === 'simple' && processedF5List.some((f: any) => f.isMissing)) {
            return processedF5List.length
        }
        return f5Response?.total || 0
    }, [searchMode, processedF5List, f5Response?.total])

    const totalPages = Math.ceil(displayTotal / pageSize)

    // Handlers
    const handleSearch = useCallback(() => {
        setPage(1)
        refetch()
    }, [refetch])

    const handleRefresh = useCallback(() => {
        refetch()
    }, [refetch])

    const handleRowClick = useCallback((f5: F5Info) => {
        if ((f5 as any).isMissing) {
            toast.error('未找到该F5记录')
            return
        }
        setSelectedF5(f5)
        setSheetOpen(true)
    }, [])

    // Get status badge
    const getStatusBadge = (status: string) => {
        const lower = status?.toLowerCase() || ''
        if (lower === 'active' || lower === 'running' || lower === 'online') {
            return <Badge variant="outline" className="gap-1 bg-green-500/10 text-green-600 border-green-500/20"><CheckCircle className="h-3 w-3" />{status}</Badge>
        }
        if (lower === 'inactive' || lower === 'offline' || lower === 'stopped') {
            return <Badge variant="outline" className="gap-1 bg-red-500/10 text-red-500 border-red-500/20"><AlertCircle className="h-3 w-3" />{status}</Badge>
        }
        return <Badge variant="outline" className="bg-muted/50 text-muted-foreground">{status}</Badge>
    }

    return (
        <div className="space-y-6">
            {/* Header */}
            <div className="flex items-center justify-between">
                <div className="flex items-center gap-3">
                    <div className="p-2.5 rounded-xl bg-blue-500/10 text-blue-500 ring-1 ring-blue-500/20">
                        <Database className="h-6 w-6" />
                    </div>
                    <div>
                        <h1 className="text-2xl font-bold tracking-tight">F5 负载均衡器管理</h1>
                        <p className="text-sm text-muted-foreground mt-0.5">
                            管理和监控 F5 负载均衡器资产
                        </p>
                    </div>
                </div>
                <Button variant="outline" size="sm" onClick={handleRefresh} disabled={isLoading} className="shadow-sm hover:shadow transition-all">
                    <RefreshCw className={cn("h-4 w-4 mr-2", isLoading && "animate-spin")} />
                    刷新
                </Button>
            </div>

            {/* Search and Filter Tabs */}
            <Tabs value={searchMode} onValueChange={(v) => setSearchMode(v as 'simple' | 'advanced')}>
                <div className="flex items-center justify-between">
                    <TabsList>
                        <TabsTrigger value="simple" className="gap-2">
                            <Search className="h-4 w-4" />
                            快速搜索
                        </TabsTrigger>
                        <TabsTrigger value="advanced" className="gap-2">
                            <Filter className="h-4 w-4" />
                            高级筛选
                        </TabsTrigger>
                    </TabsList>

                    <div className="flex items-center gap-2 text-sm">
                        <span className="text-muted-foreground">共</span>
                        <Badge variant="secondary" className="font-mono">
                            {displayTotal.toLocaleString()}
                        </Badge>
                        <span className="text-muted-foreground">条记录</span>
                    </div>
                </div>

                {/* Simple Search */}
                <TabsContent value="simple" className="mt-4">
                    <Card>
                        <CardHeader>
                            <CardTitle>快速搜索</CardTitle>
                            <CardDescription>
                                支持单行或多行输入，多行时自动按顺序匹配结果
                            </CardDescription>
                        </CardHeader>
                        <CardContent className="space-y-4">
                            <Textarea
                                placeholder="输入名称、VIP 或 AppID 进行搜索（支持多行，用换行、逗号或分号分隔）&#10;例如：&#10;192.168.1.100&#10;test-f5&#10;app-001"
                                value={keyword}
                                onChange={(e) => setKeyword(e.target.value)}
                                className="min-h-[120px] font-mono"
                                onKeyDown={(e) => {
                                    if (e.key === 'Enter' && e.ctrlKey) {
                                        handleSearch()
                                    }
                                }}
                            />
                            <Button onClick={handleSearch} disabled={isLoading} className="w-full">
                                <Search className="h-4 w-4 mr-2" />
                                搜索 (Ctrl + Enter)
                            </Button>
                        </CardContent>
                    </Card>
                </TabsContent>

                {/* Advanced Filters */}
                <TabsContent value="advanced" className="mt-4">
                    <Card>
                        <CardHeader>
                            <CardTitle>高级筛选</CardTitle>
                            <CardDescription>使用多个条件精确筛选</CardDescription>
                        </CardHeader>
                        <CardContent className="space-y-4">
                            <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
                                <div className="space-y-2">
                                    <label className="text-sm font-medium">名称</label>
                                    <Input
                                        placeholder="F5名称"
                                        value={filters.name}
                                        onChange={(e) => setFilters({ ...filters, name: e.target.value })}
                                    />
                                </div>
                                <div className="space-y-2">
                                    <label className="text-sm font-medium">VIP</label>
                                    <Input
                                        placeholder="虚拟IP"
                                        value={filters.vip}
                                        onChange={(e) => setFilters({ ...filters, vip: e.target.value })}
                                    />
                                </div>
                                <div className="space-y-2">
                                    <label className="text-sm font-medium">端口</label>
                                    <Input
                                        placeholder="端口号"
                                        value={filters.port}
                                        onChange={(e) => setFilters({ ...filters, port: e.target.value })}
                                    />
                                </div>
                                <div className="space-y-2">
                                    <label className="text-sm font-medium">AppID</label>
                                    <Input
                                        placeholder="应用ID"
                                        value={filters.appid}
                                        onChange={(e) => setFilters({ ...filters, appid: e.target.value })}
                                    />
                                </div>
                                <div className="space-y-2">
                                    <label className="text-sm font-medium">状态</label>
                                    <Input
                                        placeholder="状态"
                                        value={filters.status}
                                        onChange={(e) => setFilters({ ...filters, status: e.target.value })}
                                    />
                                </div>
                                <div className="space-y-2">
                                    <label className="text-sm font-medium">集群名称</label>
                                    <Input
                                        placeholder="K8s集群"
                                        value={filters.cluster_name}
                                        onChange={(e) => setFilters({ ...filters, cluster_name: e.target.value })}
                                    />
                                </div>
                            </div>
                            <div className="flex gap-2">
                                <Button onClick={handleSearch} disabled={isLoading} className="flex-1">
                                    <Search className="h-4 w-4 mr-2" />
                                    应用筛选
                                </Button>
                                <Button
                                    variant="outline"
                                    onClick={() => setFilters({ name: '', vip: '', port: '', appid: '', status: '', cluster_name: '' })}
                                >
                                    重置
                                </Button>
                            </div>
                        </CardContent>
                    </Card>
                </TabsContent>
            </Tabs>

            {/* Results Table */}
            <Card>
                <CardHeader>
                    <CardTitle>F5 列表</CardTitle>
                </CardHeader>
                <CardContent>
                    {isLoading ? (
                        <div className="flex items-center justify-center py-12">
                            <RefreshCw className="h-8 w-8 animate-spin text-muted-foreground" />
                        </div>
                    ) : processedF5List.length === 0 ? (
                        <div className="text-center py-12 text-muted-foreground">
                            暂无数据
                        </div>
                    ) : (
                        <div className="rounded-md border">
                            <Table>
                                <TableHeader>
                                    <TableRow>
                                        <TableHead>名称</TableHead>
                                        <TableHead>VIP</TableHead>
                                        <TableHead>端口</TableHead>
                                        <TableHead>AppID</TableHead>
                                        <TableHead>实例组</TableHead>
                                        <TableHead>状态</TableHead>
                                        <TableHead>Pool状态</TableHead>
                                        <TableHead>集群</TableHead>
                                        <TableHead>忽略</TableHead>
                                        <TableHead className="text-right">操作</TableHead>
                                    </TableRow>
                                </TableHeader>
                                <TableBody>
                                    {processedF5List.map((f5: any) => (
                                        <TableRow
                                            key={f5.id}
                                            className={cn(
                                                "cursor-pointer hover:bg-muted/50 transition-colors",
                                                f5.isMissing && "bg-red-50 dark:bg-red-950/20",
                                                f5.isExactMatch && "bg-blue-50 dark:bg-blue-950 /20"
                                            )}
                                            onClick={() => handleRowClick(f5)}
                                        >
                                            <TableCell className="font-medium">
                                                {f5.isMissing ? (
                                                    <span className="text-red-600 font-mono">
                                                        {f5.originalKeyword} <Badge variant="destructive" className="ml-2">未找到</Badge>
                                                    </span>
                                                ) : (
                                                    <div className="flex items-center gap-2">
                                                        {f5.name}
                                                        {f5.isExactMatch && <Badge variant="outline" className="text-blue-600 border-blue-200">精确匹配</Badge>}
                                                    </div>
                                                )}
                                            </TableCell>
                                            <TableCell className="font-mono">{f5.vip || '-'}</TableCell>
                                            <TableCell>{f5.port || '-'}</TableCell>
                                            <TableCell>{f5.appid || '-'}</TableCell>
                                            <TableCell>{f5.instance_group || '-'}</TableCell>
                                            <TableCell>{f5.status ? getStatusBadge(f5.status) : '-'}</TableCell>
                                            <TableCell>{f5.pool_status ? getStatusBadge(f5.pool_status) : '-'}</TableCell>
                                            <TableCell>{f5.cluster_name || '-'}</TableCell>
                                            <TableCell>
                                                {f5.ignored ? (
                                                    <Badge variant="outline" className="text-yellow-600">是</Badge>
                                                ) : (
                                                    <span className="text-muted-foreground">否</span>
                                                )}
                                            </TableCell>
                                            <TableCell className="text-right">
                                                {!f5.isMissing && (
                                                    <Button
                                                        variant="ghost"
                                                        size="sm"
                                                        onClick={(e) => {
                                                            e.stopPropagation()
                                                            handleRowClick(f5)
                                                        }}
                                                    >
                                                        <Eye className="h-4 w-4 mr-1" />
                                                        详情
                                                    </Button>
                                                )}
                                            </TableCell>
                                        </TableRow>
                                    ))}
                                </TableBody>
                            </Table>
                        </div>
                    )}
                </CardContent>
            </Card>

            {/* Pagination */}
            {displayTotal > 0 && (
                <div className="flex items-center justify-between py-4">
                    <p className="text-sm text-muted-foreground">
                        共 {displayTotal} 条记录
                    </p>
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

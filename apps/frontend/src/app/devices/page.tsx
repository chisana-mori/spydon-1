'use client'

import { useState, useCallback, useEffect, useMemo } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useSearchParams } from 'next/navigation'
import { toast } from 'sonner'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import {
    Server,
    Search,
    Filter,
    FileText,
    Download,
    RefreshCw,
    Star,
} from 'lucide-react'
import RobustaAPI from '@/lib/api'
import { NavyDevice, FilterGroup, NavyDeviceQueryRequest, QueryTemplate, createDefaultFilterGroup } from '@/types/navy'
import { DeviceDataTable } from './components/DeviceDataTable'
import { DeviceSheet } from './components/DeviceSheet'
import { SimpleQueryPanel } from './components/SimpleQueryPanel'
import { AdvancedQueryPanel } from './components/AdvancedQueryPanel'
import { TemplatePanel } from './components/TemplatePanel'
import { DeviceBulkActions } from './components/DeviceBulkActions'
import { cn } from '@/lib/utils'
import {
    Pagination,
    PaginationContent,
    PaginationEllipsis,
    PaginationItem,
    PaginationLink,
    PaginationNext,
    PaginationPrevious,
} from "@/components/ui/pagination"

type QueryMode = 'simple' | 'advanced' | 'template'

interface QueryState {
    mode: QueryMode
    // 简单查询状态
    simpleKeyword: string
    simpleOnlySpecial: boolean
    // 高级查询状态
    advancedGroups: FilterGroup[]
    advancedSourceTemplateId?: number
    advancedSourceTemplateName?: string
    // 模板查询状态
    templateId: number | null
    templateName: string
}

import { SafeDrainProvider } from './context/SafeDrainContext'
import { SafeDrainDrawer } from './components/SafeDrainDrawer'

function DevicesContent() {
    const searchParams = useSearchParams()
    const queryClient = useQueryClient()

    // URL 参数
    const urlTab = searchParams.get('tab') as QueryMode | null
    const urlTemplateId = searchParams.get('templateId')

    // 查询状态
    const [queryState, setQueryState] = useState<QueryState>({
        mode: urlTab || 'simple',
        simpleKeyword: '',
        simpleOnlySpecial: false,
        advancedGroups: [createDefaultFilterGroup()],
        templateId: urlTemplateId ? parseInt(urlTemplateId) : null,
        templateName: '',
    })

    // 分页状态
    const [page, setPage] = useState(1)
    const [pageSize] = useState(20)

    // 选中设备（单个详情）
    const [selectedDevice, setSelectedDevice] = useState<NavyDevice | null>(null)
    const [sheetOpen, setSheetOpen] = useState(false)

    // 批量选择的设备
    const [selectedDevices, setSelectedDevices] = useState<Set<number>>(new Set())

    // 简单查询
    const simpleQuery = useQuery({
        queryKey: ['navy-devices-simple', queryState.simpleKeyword, queryState.simpleOnlySpecial, page, pageSize],
        queryFn: () => RobustaAPI.listNavyDevices({
            keyword: queryState.simpleKeyword,
            onlySpecial: queryState.simpleOnlySpecial,
            page,
            size: pageSize,
        }),
        enabled: queryState.mode === 'simple',
    })

    // 高级查询
    const advancedQueryMutation = useMutation({
        mutationFn: (req: NavyDeviceQueryRequest) => RobustaAPI.queryNavyDevices(req),
    })

    // 执行高级查询
    const executeAdvancedQuery = useCallback(() => {
        // 过滤激活的组和块
        const activeGroups = queryState.advancedGroups
            .map(group => ({
                ...group,
                blocks: group.blocks.filter(b => b.isActive !== false && b.key),
            }))
            .filter(group => group.blocks.length > 0)

        if (activeGroups.length === 0) {
            toast.error('请至少添加一个有效的筛选条件')
            return
        }

        advancedQueryMutation.mutate({
            groups: activeGroups,
            page,
            size: pageSize,
        })
    }, [queryState.advancedGroups, page, pageSize, advancedQueryMutation])

    // 加载模板并执行查询
    const loadAndExecuteTemplate = useCallback(async (templateId: number) => {
        try {
            const template = await RobustaAPI.getNavyTemplate(templateId)
            // 确保加载模板时所有条件都是激活的
            const activeGroups = template.groups.map(g => ({
                ...g,
                blocks: g.blocks.map(b => ({ ...b, isActive: true }))
            }))

            setQueryState(prev => ({
                ...prev,
                mode: 'advanced',
                advancedGroups: activeGroups,
                advancedSourceTemplateId: template.id,
                advancedSourceTemplateName: template.name,
            }))
            // 自动执行查询
            advancedQueryMutation.mutate({
                groups: activeGroups,
                page,
                size: pageSize,
            })
        } catch (err) {
            toast.error('加载模板失败')
        }
    }, [page, pageSize, advancedQueryMutation])

    // 处理简单搜索
    const handleSimpleSearch = useCallback((keyword: string) => {
        setQueryState(prev => ({ ...prev, simpleKeyword: keyword }))
        setPage(1)
    }, [])

    // 处理高级查询条件变更
    const handleAdvancedGroupsChange = useCallback((groups: FilterGroup[]) => {
        setQueryState(prev => ({ ...prev, advancedGroups: groups }))
    }, [])

    // 处理模式切换
    const handleModeChange = useCallback((mode: QueryMode) => {
        setQueryState(prev => ({ ...prev, mode }))
        setPage(1)
    }, [])

    // 处理设备选择
    const handleDeviceSelect = useCallback((device: NavyDevice) => {
        setSelectedDevice(device)
        setSheetOpen(true)
    }, [])

    // 刷新数据
    const handleRefresh = useCallback(() => {
        if (queryState.mode === 'simple') {
            simpleQuery.refetch()
        } else {
            executeAdvancedQuery()
        }
    }, [queryState.mode, simpleQuery, executeAdvancedQuery])

    // 导出数据
    const handleExport = useCallback(() => {
        window.open(RobustaAPI.getNavyExportUrl(), '_blank')
    }, [])

    // 根据模式获取当前数据
    const getCurrentData = () => {
        if (queryState.mode === 'simple') {
            return {
                data: simpleQuery.data?.data || [],
                total: simpleQuery.data?.pagination?.total || 0,
                isLoading: simpleQuery.isLoading,
            }
        } else {
            return {
                data: advancedQueryMutation.data?.data || [],
                total: advancedQueryMutation.data?.pagination?.total || 0,
                isLoading: advancedQueryMutation.isPending,
            }
        }
    }

    // 处理批量查询结果
    const processBatchResults = useCallback((devices: NavyDevice[], keyword: string): NavyDevice[] => {
        const inputLines = keyword.split(/[\n,;]+/)
            .map(line => line.trim())
            .filter(line => line.length > 0)

        // 如果不是多行输入，或者是单行输入且不含分隔符，不做特殊处理
        if (inputLines.length <= 1) {
            return devices.map(d => {
                const isExact = (inputLines[0] && (d.ip === inputLines[0] || d.ci_code === inputLines[0]))
                return isExact ? { ...d, isExactMatch: true } : d
            })
        }

        const result: NavyDevice[] = []
        const seenIds = new Set<number>()

        // 遍历每行输入，在 devices 中寻找匹配项
        inputLines.forEach(line => {
            const lowerLine = line.toLowerCase()

            // 在返回结果中查找所有匹配该行的设备 (模糊匹配)
            const matches = devices.filter(d => {
                const ip = d.ip?.toLowerCase() || ''
                const ci = d.ci_code?.toLowerCase() || ''
                return ip.includes(lowerLine) || ci.includes(lowerLine)
            })

            if (matches.length > 0) {
                // 将匹配到的设备按照是否精确匹配排序，确保精确匹配优先处理（如果当前行命中了多个设备）
                const sortedMatches = [...matches].sort((a, b) => {
                    const aExact = (a.ip?.toLowerCase() === lowerLine || a.ci_code?.toLowerCase() === lowerLine) ? 1 : 0
                    const bExact = (b.ip?.toLowerCase() === lowerLine || b.ci_code?.toLowerCase() === lowerLine) ? 1 : 0
                    return bExact - aExact
                })

                sortedMatches.forEach(match => {
                    const isExact = (match.ip?.toLowerCase() === lowerLine || match.ci_code?.toLowerCase() === lowerLine)

                    if (!seenIds.has(match.id)) {
                        // 新设备：直接添加
                        result.push({
                            ...match,
                            isExactMatch: isExact,
                            originalKeyword: line
                        })
                        seenIds.add(match.id)
                    } else if (isExact) {
                        // 已存在的设备，但当前行是它的“精确匹配”：升级该设备的显示状态
                        const existingIdx = result.findIndex(r => r.id === match.id)
                        if (existingIdx !== -1) {
                            result[existingIdx] = {
                                ...result[existingIdx],
                                isExactMatch: true,
                                originalKeyword: line
                            }
                        }
                    }
                })
            } else {
                // 未找到匹配项的处理：创建虚拟行
                result.push({
                    id: -Math.random(), // 临时负数 ID
                    ci_code: line,      // 借用字段显示输入值
                    ip: '',
                    status: 'missing',
                    isVirtual: true,
                    isMissing: true,
                    originalKeyword: line,
                    // 填充必填字段以防 TS 报错
                    arch_type: '',
                    idc: '',
                    room: '',
                    cabinet: '',
                    cabinet_no: '',
                    infra_type: '',
                    is_localization: false,
                    net_zone: '',
                    group: '',
                    appid: '',
                    app_name: '',
                    os_create_time: '',
                    cpu: 0,
                    memory: 0,
                    model: '',
                    kvm_ip: '',
                    os: '',
                    company: '',
                    os_name: '',
                    os_issue: '',
                    os_kernel: '',
                    role: '',
                    cluster: '',
                    cluster_id: 0,
                    acceptance_time: '',
                    disk_count: 0,
                    disk_detail: '',
                    network_speed: '',
                    is_special: false,
                    feature_count: 0,
                    created_at: '',
                    updated_at: '',
                } as NavyDevice)
            }
        })

        return result
    }, [])

    const { data: rawDevices, total, isLoading } = getCurrentData()

    // 应用批量处理逻辑
    const processedDevices = useMemo(() => {
        return (queryState.mode === 'simple' && queryState.simpleKeyword)
            ? processBatchResults(rawDevices, queryState.simpleKeyword)
            : rawDevices
    }, [queryState.mode, queryState.simpleKeyword, rawDevices, processBatchResults])

    // 在批量模式下，总数应该是输入的行数（包含虚拟行），或者是处理后的结果数
    // 如果是普通模式，使用后端返回的 total
    const displayTotal = useMemo(() => {
        return (queryState.mode === 'simple' && processedDevices.some(d => d.isVirtual))
            ? processedDevices.length
            : total
    }, [queryState.mode, processedDevices, total])

    const totalPages = Math.ceil(displayTotal / pageSize)

    // 初始加载模板
    useEffect(() => {
        if (urlTemplateId && queryState.mode === 'template') {
            loadAndExecuteTemplate(parseInt(urlTemplateId))
        }
    }, [urlTemplateId, queryState.mode, loadAndExecuteTemplate])

    return (
        <div className="space-y-6">
            {/* 页面标题 */}
            <div className="flex items-center justify-between">
                <div className="flex items-center gap-3">
                    <div className="p-2 rounded-lg bg-primary/10">
                        <Server className="h-6 w-6 text-primary" />
                    </div>
                    <div>
                        <h1 className="text-2xl font-bold">设备管理</h1>
                        <p className="text-sm text-muted-foreground">
                            管理和查询数据中心的物理设备资产
                        </p>
                    </div>
                </div>
                <div className="flex items-center gap-2">
                    <Button variant="outline" size="sm" onClick={handleRefresh}>
                        <RefreshCw className="h-4 w-4 mr-2" />
                        刷新
                    </Button>
                    <Button variant="outline" size="sm" onClick={handleExport}>
                        <Download className="h-4 w-4 mr-2" />
                        导出
                    </Button>
                </div>
            </div>

            {/* 查询模式切换 */}
            <Tabs value={queryState.mode} onValueChange={(v) => handleModeChange(v as QueryMode)}>
                <div className="flex items-center justify-between">
                    <TabsList className="grid w-auto grid-cols-3">
                        <TabsTrigger value="simple" className="gap-2">
                            <Search className="h-4 w-4" />
                            简单查询
                        </TabsTrigger>
                        <TabsTrigger value="advanced" className="gap-2">
                            <Filter className="h-4 w-4" />
                            高级查询
                        </TabsTrigger>
                        <TabsTrigger value="template" className="gap-2">
                            <FileText className="h-4 w-4" />
                            模板查询
                        </TabsTrigger>
                    </TabsList>

                    {/* 统计信息 */}
                    <div className="flex items-center gap-4 text-sm">
                        <div className="flex items-center gap-2">
                            <span className="text-muted-foreground">共</span>
                            <Badge variant="secondary" className="font-mono">
                                {total.toLocaleString()}
                            </Badge>
                            <span className="text-muted-foreground">台设备</span>
                        </div>
                        {queryState.simpleOnlySpecial && (
                            <Badge variant="outline" className="gap-1 text-amber-600 border-amber-200 bg-amber-50">
                                <Star className="h-3 w-3 fill-amber-600" />
                                仅特殊设备
                            </Badge>
                        )}
                    </div>
                </div>

                {/* 简单查询 */}
                <TabsContent value="simple" className="mt-4 space-y-4">
                    <SimpleQueryPanel
                        onSearch={handleSimpleSearch}
                        isLoading={isLoading}
                        showSpecial={queryState.simpleOnlySpecial}
                        onToggleSpecial={() => setQueryState(prev => ({
                            ...prev,
                            simpleOnlySpecial: !prev.simpleOnlySpecial
                        }))}
                    />
                </TabsContent>

                {/* 高级查询 */}
                <TabsContent value="advanced" className="mt-4">
                    <AdvancedQueryPanel
                        groups={queryState.advancedGroups}
                        onChange={handleAdvancedGroupsChange}
                        onExecute={executeAdvancedQuery}
                        isLoading={advancedQueryMutation.isPending}
                        sourceTemplateId={queryState.advancedSourceTemplateId}
                        sourceTemplateName={queryState.advancedSourceTemplateName}
                    />
                </TabsContent>

                {/* 模板查询 */}
                <TabsContent value="template" className="mt-4">
                    <TemplatePanel
                        onSelectTemplate={(template: QueryTemplate) => {
                            // 确保加载模板时所有条件都是激活的
                            const activeGroups = template.groups.map(g => ({
                                ...g,
                                blocks: g.blocks.map(b => ({ ...b, isActive: true }))
                            }))

                            setQueryState(prev => ({
                                ...prev,
                                templateId: template.id || null,
                                templateName: template.name,
                                advancedGroups: activeGroups,
                            }))
                            // 切换到高级模式并执行
                            handleModeChange('advanced')
                            advancedQueryMutation.mutate({
                                groups: activeGroups,
                                page: 1,
                                size: pageSize,
                            })
                        }}
                    />
                </TabsContent>
            </Tabs>

            {/* 批量操作工具栏 */}
            <DeviceBulkActions
                selectedDevices={selectedDevices}
                devices={processedDevices}
                onClearSelection={() => setSelectedDevices(new Set())}
                onRefresh={handleRefresh}
            />

            {/* 设备列表 */}
            <DeviceDataTable
                devices={processedDevices}
                isLoading={isLoading || advancedQueryMutation.isPending}
                onSelect={handleDeviceSelect}
                selectedId={selectedDevice?.id}
                onRefresh={handleRefresh}
                selectedDevices={selectedDevices}
                onSelectionChange={setSelectedDevices}
            />

            {/* 分页 */}
            {total > 0 && (
                <div className="flex items-center justify-between py-4">
                    <p className="text-sm text-muted-foreground">
                        共 {total} 条记录
                    </p>
                    <Pagination className="w-auto mx-0">
                        <PaginationContent>
                            <PaginationItem>
                                <PaginationPrevious
                                    onClick={() => setPage(p => Math.max(1, p - 1))}
                                    className={cn("cursor-pointer", page <= 1 && "pointer-events-none opacity-50")}
                                />
                            </PaginationItem>

                            {/* Logic to show reasonable page numbers */}
                            {(() => {
                                const pages = [];
                                const maxVisible = 5;

                                if (totalPages <= maxVisible) {
                                    for (let i = 1; i <= totalPages; i++) pages.push(i);
                                } else {
                                    // Always show 1
                                    pages.push(1);

                                    if (page > 3) pages.push('ellipsis-start');

                                    const start = Math.max(2, page - 1);
                                    const end = Math.min(totalPages - 1, page + 1);

                                    for (let i = start; i <= end; i++) {
                                        if (i > 1 && i < totalPages) pages.push(i);
                                    }

                                    if (page < totalPages - 2) pages.push('ellipsis-end');

                                    // Always show last
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

            {/* 设备详情抽屉 */}
            <DeviceSheet
                device={selectedDevice}
                open={sheetOpen}
                onOpenChange={(open) => {
                    setSheetOpen(open)
                    if (!open) setTimeout(() => setSelectedDevice(null), 300) // 延迟清除以避免UI闪烁
                }}
                onUpdate={handleRefresh}
            />
        </div>

    )
}

export default function DevicesPage() {
    return (
        <SafeDrainProvider>
            <DevicesContent />
            <SafeDrainDrawer />
        </SafeDrainProvider>
    )
}

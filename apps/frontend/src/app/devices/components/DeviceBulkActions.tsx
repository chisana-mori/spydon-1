'use client'

import { useState, useEffect, useMemo } from 'react'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuSeparator,
    DropdownMenuTrigger,
    DropdownMenuLabel,
} from '@/components/ui/dropdown-menu'
import {
    AlertDialog,
    AlertDialogAction,
    AlertDialogCancel,
    AlertDialogContent,
    AlertDialogDescription,
    AlertDialogFooter,
    AlertDialogHeader,
    AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import {
    Sheet,
    SheetContent,
    SheetHeader,
    SheetTitle,
} from '@/components/ui/sheet'
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Input } from '@/components/ui/input'
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from '@/components/ui/select'
import { ScrollArea } from '@/components/ui/scroll-area'
import {
    Settings2,
    Power,
    X,
    Loader2,
    ShieldOff,
    Shield,
    Trash2,
    Tag,
    RefreshCw,
    PowerOff,
    Hexagon,
    Ban,
    PlayCircle,
    DoorOpen,
    Zap,
    CheckCircle2,
    LayoutGrid,
    Plus,
    Check,
    AlertTriangle,
    Layers,
    Server,
    ShieldAlert,
    Info,
} from 'lucide-react'
import { toast } from 'sonner'
import RobustaAPI, { BatchOperationResult } from '@/lib/api'
import { DeviceFeatureDetails, NodeLabelTaintResponse } from '@/types/navy'
import { NavyDevice } from '@/types/navy'
import { cn } from '@/lib/utils'
import { ConfigCombobox } from '@/components/common/ConfigCombobox'
import { Separator } from "@/components/ui/separator"

interface DeviceBulkActionsProps {
    selectedDevices: Set<number>
    devices: NavyDevice[]
    onClearSelection: () => void
    onRefresh: () => void
}

type OperationType = 'cordon' | 'uncordon' | 'drain' | 'shutdown' | 'reboot'

import { useSafeDrain } from '../context/SafeDrainContext'

export function DeviceBulkActions({
    selectedDevices,
    devices,
    onClearSelection,
    onRefresh,
}: DeviceBulkActionsProps) {
    const { addDrain, activeDrains, isDrawerOpen, restoreDrawer } = useSafeDrain()
    const [isLoading, setIsLoading] = useState(false)
    const [confirmDialog, setConfirmDialog] = useState<{
        open: boolean
        operation: OperationType | null
        title: string
        description: string
    }>({
        open: false,
        operation: null,
        title: '',
        description: '',
    })
    const [taintLabelSheetOpen, setTaintLabelSheetOpen] = useState(false)
    const [k8sConfirmOpen, setK8sConfirmOpen] = useState(false)
    const [k8sOperation, setK8sOperation] = useState<'cordon' | 'uncordon' | 'drain'>('drain')

    const selectedCount = selectedDevices.size
    const runningDrains = activeDrains.filter(d => d.status === 'running')
    const completedOrFailedDrains = activeDrains.filter(d => d.status !== 'running')
    const hasActiveDrains = runningDrains.length > 0 && !isDrawerOpen

    // 获取选中设备的 ci_codes
    const getSelectedCICodes = (): string[] => {
        return devices
            .filter(d => selectedDevices.has(d.id))
            .map(d => d.ci_code)
            .filter(Boolean)
    }

    // 获取选中设备中有集群关联的数量
    const clusteredCount = devices.filter(
        d => selectedDevices.has(d.id) && d.cluster
    ).length

    const showConfirmDialog = (operation: OperationType) => {
        if (operation === 'drain' || operation === 'cordon' || operation === 'uncordon') {
            setK8sOperation(operation)
            setK8sConfirmOpen(true)
            return
        }

        const ciCodes = getSelectedCICodes()
        const titles: Record<string, string> = {
            shutdown: '关机',
            reboot: '重启',
        }
        const descriptions: Record<string, string> = {
            shutdown: `确定要关闭 ${ciCodes.length} 台设备吗？此操作将通过 AWX 执行。`,
            reboot: `确定要重启 ${ciCodes.length} 台设备吗？此操作将通过 AWX 执行。`,
        }

        setConfirmDialog({
            open: true,
            operation,
            title: titles[operation] || operation,
            description: descriptions[operation] || '',
        })
    }

    const handleK8sConfirm = async (confirmedCiCodes: string[]) => {
        setIsLoading(true)
        setK8sConfirmOpen(false)
        try {
            let result: BatchOperationResult | undefined
            switch (k8sOperation) {
                case 'cordon':
                    result = await RobustaAPI.cordonNodes(confirmedCiCodes)
                    break
                case 'uncordon':
                    result = await RobustaAPI.uncordonNodes(confirmedCiCodes)
                    break
                case 'drain':
                    result = await RobustaAPI.drainNodes(confirmedCiCodes, {
                        force: false,
                        ignore_daemonsets: true,
                        delete_local_data: false,
                        timeout: 300,
                    })
                    break
            }

            if (result) {
                const titleMap = { uncordon: 'Uncordon', cordon: 'Cordon', drain: 'Drain' }
                const title = titleMap[k8sOperation]

                if (result.failed === 0) {
                    toast.success(`${title} 节点成功`, {
                        description: `共 ${result.succeeded} 个节点操作成功`,
                    })
                } else if (result.succeeded === 0) {
                    const errorDetail = result.results?.find((r: any) => !r.success)?.error || '未知错误'
                    toast.error(`${title} 节点失败`, {
                        description: errorDetail,
                    })
                } else {
                    const failedItems = result.results?.filter((r: any) => !r.success)
                    const errorDetail = failedItems?.[0]?.error || '部分节点操作失败'
                    toast.warning(`${title} 节点部分成功`, {
                        description: `${result.succeeded} 成功, ${result.failed} 失败. 例如: ${errorDetail}`,
                    })
                }

                if (k8sOperation === 'drain' && result.results) {
                    result.results.forEach((r: any) => {
                        if (r.success && r.drain_id) {
                            const device = devices.find(d => d.ci_code === r.ci_code)
                            addDrain(r.drain_id, r.ci_code, device?.cluster || 'Unknown')
                        }
                    })
                }


                // 立即刷新
                onRefresh()
                // 延迟 3 秒再次刷新以确保状态更新
                setTimeout(() => {
                    onRefresh()
                }, 3000)

                onClearSelection()
            }
        } catch (error) {
            console.error(error)
            toast.error(`${k8sOperation} 请求失败`)
        } finally {
            setIsLoading(false)
        }
    }

    const executeOperation = async () => {
        const operation = confirmDialog.operation
        if (!operation) return

        const ciCodes = getSelectedCICodes()
        if (ciCodes.length === 0) {
            toast.error('没有可操作的设备')
            return
        }

        setIsLoading(true)
        setConfirmDialog({ ...confirmDialog, open: false })
        try {
            switch (operation) {
                // K8s operations moved to handleK8sConfirm
                case 'shutdown':
                    await RobustaAPI.shutdownNodes(ciCodes)
                    toast.success('关机任务已提交')
                    onClearSelection()
                    return
                case 'reboot':
                    await RobustaAPI.rebootNodes(ciCodes)
                    toast.success('重启任务已提交')
                    onClearSelection()
                    return
            }
        } catch (error: unknown) {
            // ...
        } finally {
            setIsLoading(false)
        }
    }

    if (selectedCount === 0 && !hasActiveDrains) return null

    return (
        <>
            <div className="fixed bottom-10 left-1/2 -translate-x-1/2 z-50 flex items-center gap-2 p-1.5 pl-4 bg-background/80 backdrop-blur-xl supports-[backdrop-filter]:bg-background/80 rounded-full border border-border/40 shadow-[0_8px_30px_rgb(0,0,0,0.12)] animate-in slide-in-from-bottom-8 duration-300 zoom-in-95 group">

                {selectedCount > 0 ? (
                    <>
                        {/* 选中信息 */}
                        <div className="flex items-center gap-3 mr-2">
                            <div className="flex items-center gap-2 text-foreground/80">
                                <CheckCircle2 className="h-4 w-4 text-primary" />
                                <span className="text-sm font-medium">
                                    已选 <span className="font-mono font-semibold">{selectedCount}</span>
                                </span>
                            </div>

                            {clusteredCount > 0 && selectedCount !== clusteredCount && (
                                <span className="text-[10px] font-medium text-muted-foreground bg-muted/80 px-2 py-0.5 rounded-full ring-1 ring-border/50">
                                    {clusteredCount} K8s
                                </span>
                            )}
                        </div>

                        <div className="w-px h-5 bg-border/60 mx-1" />

                        {/* 操作按钮组 */}
                        <div className="flex items-center gap-1.5">
                            {/* K8s 操作 */}
                            <DropdownMenu>
                                <DropdownMenuTrigger asChild>
                                    <Button
                                        variant="ghost"
                                        size="sm"
                                        disabled={isLoading || clusteredCount === 0}
                                        className={cn(
                                            "h-8 w-28 rounded-full border border-transparent font-medium",
                                            "text-blue-700 hover:text-blue-800 hover:bg-blue-50/80 dark:text-blue-400 dark:hover:bg-blue-900/30",
                                            "data-[state=open]:bg-blue-100/50 data-[state=open]:text-blue-700",
                                            "transition-all duration-200"
                                        )}
                                    >
                                        {isLoading ? (
                                            <Loader2 className="h-3.5 w-3.5 mr-2 animate-spin" />
                                        ) : (
                                            <LayoutGrid className="h-3.5 w-3.5 mr-2" />
                                        )}
                                        K8s 管理
                                    </Button>
                                </DropdownMenuTrigger>
                                <DropdownMenuContent align="center" side="top" className="w-56 p-1 mb-3 rounded-xl shadow-lg border-border/60 backdrop-blur-lg bg-background/95">
                                    {/* ... existing menu items ... */}
                                    <DropdownMenuLabel className="text-xs font-medium text-muted-foreground px-2 py-1.5 uppercase tracking-wider">调度控制</DropdownMenuLabel>

                                    <DropdownMenuItem onClick={() => showConfirmDialog('cordon')} className="flex items-center gap-2.5 px-2 py-2 cursor-pointer rounded-lg mb-0.5 focus:bg-amber-50 dark:focus:bg-amber-950/20">
                                        <div className="h-8 w-8 rounded-md bg-amber-100/50 flex items-center justify-center text-amber-600 dark:bg-amber-900/40 dark:text-amber-400">
                                            <Ban className="h-4 w-4" />
                                        </div>
                                        <div className="flex flex-col gap-0.5 flex-1">
                                            <span className="text-sm font-medium">Cordon</span>
                                            <span className="text-[10px] text-muted-foreground">禁止调度</span>
                                        </div>
                                    </DropdownMenuItem>

                                    <DropdownMenuItem onClick={() => showConfirmDialog('uncordon')} className="flex items-center gap-2.5 px-2 py-2 cursor-pointer rounded-lg mb-0.5 focus:bg-emerald-50 dark:focus:bg-emerald-950/20">
                                        <div className="h-8 w-8 rounded-md bg-emerald-100/50 flex items-center justify-center text-emerald-600 dark:bg-emerald-900/40 dark:text-emerald-400">
                                            <PlayCircle className="h-4 w-4" />
                                        </div>
                                        <div className="flex flex-col gap-0.5 flex-1">
                                            <span className="text-sm font-medium">Uncordon</span>
                                            <span className="text-[10px] text-muted-foreground">恢复调度</span>
                                        </div>
                                    </DropdownMenuItem>

                                    <DropdownMenuSeparator className="my-1 bg-border/50" />

                                    <DropdownMenuItem
                                        onClick={() => showConfirmDialog('drain')}
                                        className="flex items-center gap-2.5 px-2 py-2 cursor-pointer rounded-lg focus:bg-orange-50 dark:focus:bg-orange-950/20 group/item"
                                    >
                                        <div className="h-8 w-8 rounded-md bg-orange-100/50 flex items-center justify-center text-orange-600 dark:bg-orange-900/40 dark:text-orange-400 group-focus/item:text-orange-700">
                                            <DoorOpen className="h-4 w-4" />
                                        </div>
                                        <div className="flex flex-col gap-0.5 flex-1">
                                            <span className="text-sm font-medium group-focus/item:text-orange-700 dark:group-focus/item:text-orange-300">Drain</span>
                                            <span className="text-[10px] text-muted-foreground">驱逐 Pod</span>
                                        </div>
                                    </DropdownMenuItem>

                                    <DropdownMenuItem onClick={() => setTaintLabelSheetOpen(true)} className="flex items-center gap-2.5 px-2 py-2 cursor-pointer rounded-lg focus:bg-purple-50 dark:focus:bg-purple-950/20 group/item">
                                        <div className="h-8 w-8 rounded-md bg-purple-100/50 flex items-center justify-center text-purple-600 dark:bg-purple-900/40 dark:text-purple-400">
                                            <Tag className="h-4 w-4" />
                                        </div>
                                        <div className="flex flex-col gap-0.5 flex-1">
                                            <span className="text-sm font-medium">Taint / Label</span>
                                            <span className="text-[10px] text-muted-foreground">管理标签和污点</span>
                                        </div>
                                    </DropdownMenuItem>
                                </DropdownMenuContent>
                            </DropdownMenu>

                            {/* 电源操作 */}
                            <DropdownMenu>
                                <DropdownMenuTrigger asChild>
                                    <Button
                                        variant="ghost"
                                        size="sm"
                                        disabled={isLoading}
                                        className={cn(
                                            "h-8 w-28 rounded-full border border-transparent font-medium",
                                            "text-orange-700 hover:text-orange-800 hover:bg-orange-50/80 dark:text-orange-400 dark:hover:bg-orange-900/30",
                                            "data-[state=open]:bg-orange-100/50 data-[state=open]:text-orange-700",
                                            "transition-all duration-200"
                                        )}
                                    >
                                        <Zap className="h-3.5 w-3.5 mr-2" />
                                        电源
                                    </Button>
                                </DropdownMenuTrigger>
                                <DropdownMenuContent align="center" side="top" className="w-56 p-1 mb-3 rounded-xl shadow-lg border-border/60 backdrop-blur-lg bg-background/95">
                                    <DropdownMenuLabel className="text-xs font-medium text-muted-foreground px-2 py-1.5 uppercase tracking-wider">AWX 操作</DropdownMenuLabel>

                                    <DropdownMenuItem
                                        onClick={() => showConfirmDialog('reboot')}
                                        className="flex items-center gap-2.5 px-2 py-2 cursor-pointer rounded-lg mb-0.5 focus:text-orange-700 focus:bg-orange-50 dark:focus:bg-orange-950/20"
                                    >
                                        <div className="h-8 w-8 rounded-md bg-orange-100/50 flex items-center justify-center text-orange-600 dark:bg-orange-900/40 dark:text-orange-400">
                                            <RefreshCw className="h-4 w-4" />
                                        </div>
                                        <span className="font-medium text-sm">重启设备</span>
                                    </DropdownMenuItem>

                                    <DropdownMenuItem
                                        onClick={() => showConfirmDialog('shutdown')}
                                        className="flex items-center gap-2.5 px-2 py-2 cursor-pointer rounded-lg focus:text-red-700 focus:bg-red-50 dark:focus:bg-red-950/20"
                                    >
                                        <div className="h-8 w-8 rounded-md bg-red-100/50 flex items-center justify-center text-red-600 dark:bg-red-900/40 dark:text-red-400">
                                            <PowerOff className="h-4 w-4" />
                                        </div>
                                        <span className="font-medium text-sm">立即关机</span>
                                    </DropdownMenuItem>
                                </DropdownMenuContent>
                            </DropdownMenu>
                        </div>

                        <div className="pl-1">
                            <Button
                                variant="ghost"
                                size="icon"
                                onClick={onClearSelection}
                                className="h-8 w-8 rounded-full hover:bg-muted text-muted-foreground hover:text-foreground transition-all duration-200 hover:scale-105"
                                title="取消选择"
                            >
                                <X className="h-4 w-4" />
                            </Button>
                        </div>
                    </>
                ) : null}

                {/* Minimized Drain Indicator */}
                {/* Minimized Drain Indicator: Only show if there are RUNNING drains */}
                {hasActiveDrains && (
                    <>
                        {selectedCount > 0 && <div className="w-px h-5 bg-border/60 mx-1" />}
                        <Button
                            variant="ghost"
                            size="sm"
                            onClick={restoreDrawer}
                            className="h-8 rounded-full border border-orange-200/50 bg-orange-50/50 text-orange-700 hover:text-orange-800 hover:bg-orange-100/80 dark:border-orange-800/30 dark:bg-orange-900/20 dark:text-orange-300 dark:hover:bg-orange-900/40"
                        >
                            <Loader2 className="h-3.5 w-3.5 mr-2 animate-spin text-orange-600 dark:text-orange-400" />
                            <span className="font-medium">
                                Drain 任务 ({runningDrains.length})
                            </span>
                        </Button>
                    </>
                )
                }
            </div>

            {/* Confirm Dialog */}
            <AlertDialog
                open={confirmDialog.open}
                onOpenChange={(open) =>
                    setConfirmDialog({ ...confirmDialog, open })
                }
            >
                {/* ... existing dialog content ... */}
                <AlertDialogContent>
                    <AlertDialogHeader>
                        <AlertDialogTitle>{confirmDialog.title}</AlertDialogTitle>
                        <AlertDialogDescription>
                            {confirmDialog.description}
                        </AlertDialogDescription>
                    </AlertDialogHeader>
                    <AlertDialogFooter>
                        <AlertDialogCancel>取消</AlertDialogCancel>
                        <AlertDialogAction
                            onClick={executeOperation}
                            className={
                                confirmDialog.operation === 'shutdown'
                                    ? 'bg-red-600 hover:bg-red-700'
                                    : confirmDialog.operation === 'drain' ||
                                        confirmDialog.operation === 'reboot'
                                        ? 'bg-orange-600 hover:bg-orange-700'
                                        : ''
                            }
                        >
                            确认执行
                        </AlertDialogAction>
                    </AlertDialogFooter>
                </AlertDialogContent>
            </AlertDialog>

            {/* Taint Label Sheet */}
            {/* ... existing sheet ... */}
            <TaintLabelSheet
                open={taintLabelSheetOpen}
                onOpenChange={setTaintLabelSheetOpen}
                devices={devices.filter(d => selectedDevices.has(d.id))}
                onSuccess={() => {
                    onRefresh()
                }}
            />

            <BulkK8sConfirmDialog
                open={k8sConfirmOpen}
                onOpenChange={setK8sConfirmOpen}
                devices={devices.filter(d => selectedDevices.has(d.id))}
                onConfirm={handleK8sConfirm}
                operation={k8sOperation}
            />
        </>
    )
}

// TaintLabelSheet Component
function TaintLabelSheet({
    open,
    onOpenChange,
    devices,
    onSuccess,
}: {
    open: boolean
    onOpenChange: (open: boolean) => void
    devices: NavyDevice[]
    onSuccess: () => void
}) {
    const [mode, setMode] = useState<'label' | 'taint'>('label')
    const [action, setAction] = useState<'add' | 'remove'>('add')
    const [labelKey, setLabelKey] = useState('')
    const [labelValue, setLabelValue] = useState('')
    const [taintKey, setTaintKey] = useState('')
    const [taintValue, setTaintValue] = useState('')
    const [taintEffect, setTaintEffect] = useState<'NoSchedule' | 'PreferNoSchedule' | 'NoExecute'>('NoSchedule')
    const [isSubmitting, setIsSubmitting] = useState(false)
    const [isLoadingFeatures, setIsLoadingFeatures] = useState(false)
    const [nodeFeatures, setNodeFeatures] = useState<NodeLabelTaintResponse[]>([])

    // 获取 ciCodes 用于批量操作 API
    const ciCodes = devices.map(d => d.ci_code).filter(Boolean)

    useEffect(() => {
        if (open && devices.length > 0) {
            fetchFeatures()
        } else {
            setNodeFeatures([])
        }
    }, [open, devices])

    const fetchFeatures = async () => {
        setIsLoadingFeatures(true)
        try {
            // 对每个有集群的设备获取实时标签/污点数据
            const clusteredDevices = devices.filter(d => d.cluster)
            const results: NodeLabelTaintResponse[] = []

            for (const device of clusteredDevices) {
                try {
                    const data = await RobustaAPI.getNodeLabelsAndTaints(device.cluster, device.ci_code)
                    results.push(data)
                } catch (e) {
                    console.warn(`获取 ${device.ci_code} 节点信息失败`, e)
                }
            }
            setNodeFeatures(results)
        } catch (err) {
            console.error(err)
            toast.error('获取现有标签/污点失败')
        } finally {
            setIsLoadingFeatures(false)
        }
    }

    // 聚合所有节点的标签和污点
    // 分为两类：1. 共同的（所有节点都有）2. 不同的（按节点分组）
    const aggregatedFeatures = useMemo(() => {
        if (nodeFeatures.length === 0) return {
            commonLabels: [],
            commonTaints: [],
            nodeSpecificLabels: new Map<string, { key: string; value: string }[]>(),
            nodeSpecificTaints: new Map<string, { key: string; value: string; effect: string }[]>(),
        }

        const nodeCount = nodeFeatures.length
        const labelMap = new Map<string, { key: string; value: string; nodes: string[] }>()
        const taintMap = new Map<string, { key: string; value: string; effect: string; nodes: string[] }>()

        // 1. 收集所有标签和污点
        for (const node of nodeFeatures) {
            for (const label of node.labels) {
                const mapKey = `${label.key}=${label.value}`
                const existing = labelMap.get(mapKey)
                if (existing) {
                    existing.nodes.push(node.nodeName)
                } else {
                    labelMap.set(mapKey, { ...label, nodes: [node.nodeName] })
                }
            }
            for (const taint of node.taints) {
                const mapKey = `${taint.key}=${taint.value}:${taint.effect}`
                const existing = taintMap.get(mapKey)
                if (existing) {
                    existing.nodes.push(node.nodeName)
                } else {
                    taintMap.set(mapKey, { ...taint, nodes: [node.nodeName] })
                }
            }
        }

        // 2. 分离共同的和特定节点的
        const commonLabels: { key: string; value: string }[] = []
        const nodeSpecificLabels = new Map<string, { key: string; value: string }[]>()

        for (const item of labelMap.values()) {
            if (item.nodes.length === nodeCount) {
                // 所有节点都有这个标签
                commonLabels.push({ key: item.key, value: item.value })
            } else {
                // 只有部分节点有，按节点分组
                for (const nodeName of item.nodes) {
                    const existing = nodeSpecificLabels.get(nodeName) || []
                    existing.push({ key: item.key, value: item.value })
                    nodeSpecificLabels.set(nodeName, existing)
                }
            }
        }

        const commonTaints: { key: string; value: string; effect: string }[] = []
        const nodeSpecificTaints = new Map<string, { key: string; value: string; effect: string }[]>()

        for (const item of taintMap.values()) {
            if (item.nodes.length === nodeCount) {
                commonTaints.push({ key: item.key, value: item.value, effect: item.effect })
            } else {
                for (const nodeName of item.nodes) {
                    const existing = nodeSpecificTaints.get(nodeName) || []
                    existing.push({ key: item.key, value: item.value, effect: item.effect })
                    nodeSpecificTaints.set(nodeName, existing)
                }
            }
        }

        return { commonLabels, commonTaints, nodeSpecificLabels, nodeSpecificTaints }
    }, [nodeFeatures])

    const resetForm = () => {
        setLabelKey('')
        setLabelValue('')
        setTaintKey('')
        setTaintValue('')
        setTaintEffect('NoSchedule')
    }

    const handleSubmit = async () => {
        if (ciCodes.length === 0) {
            toast.error('没有选中的设备')
            return
        }

        if (mode === 'label') {
            if (!labelKey.trim()) {
                toast.error('请输入 Label Key')
                return
            }
            setIsSubmitting(true)
            try {
                const result = await RobustaAPI.labelNodes(ciCodes, { [labelKey]: labelValue }, action)
                if (result.failed === 0) {
                    toast.success(`Label ${action === 'add' ? '添加' : '删除'}成功`, {
                        description: `${result.succeeded} 个节点操作成功`
                    })
                } else {
                    toast.warning(`部分${action === 'add' ? '添加' : '删除'}失败`, {
                        description: `${result.succeeded} 成功, ${result.failed} 失败`
                    })
                }
                resetForm()
                onSuccess()
                fetchFeatures() // Refresh list
            } catch (err) {
                toast.error(`Label ${action === 'add' ? '添加' : '删除'}失败`)
                console.error(err)
            } finally {
                setIsSubmitting(false)
            }
        } else {
            if (!taintKey.trim()) {
                toast.error('请输入 Taint Key')
                return
            }
            setIsSubmitting(true)
            try {
                const result = await RobustaAPI.taintNodes(ciCodes, taintKey, taintValue, taintEffect, action)
                if (result.failed === 0) {
                    toast.success(`Taint ${action === 'add' ? '添加' : '删除'}成功`, {
                        description: `${result.succeeded} 个节点操作成功`
                    })
                } else {
                    toast.warning(`部分${action === 'add' ? '添加' : '删除'}失败`, {
                        description: `${result.succeeded} 成功, ${result.failed} 失败`
                    })
                }
                resetForm()
                onSuccess()
                fetchFeatures() // Refresh list
            } catch (err) {
                toast.error(`Taint ${action === 'add' ? '添加' : '删除'}失败`)
                console.error(err)
            } finally {
                setIsSubmitting(false)
            }
        }
    }

    const handleDeleteLabel = async (key: string) => {
        setIsSubmitting(true)
        try {
            await RobustaAPI.labelNodes(ciCodes, { [key]: '' }, 'remove')
            toast.success(`Label ${key} 删除成功`)
            fetchFeatures()
            onSuccess() // trigger refresh of main list too
        } catch (err) {
            toast.error('删除失败')
        } finally {
            setIsSubmitting(false)
        }
    }

    const handleDeleteTaint = async (key: string, value: string, effect: string) => {
        setIsSubmitting(true)
        try {
            await RobustaAPI.taintNodes(ciCodes, key, value, effect as any, 'remove')
            toast.success(`Taint ${key} 删除成功`)
            fetchFeatures()
            onSuccess()
        } catch (err) {
            toast.error('删除失败')
        } finally {
            setIsSubmitting(false)
        }
    }

    return (
        <Sheet open={open} onOpenChange={onOpenChange}>
            <SheetContent className="w-[800px] sm:max-w-[800px] overflow-y-auto">
                <SheetHeader className="pb-4">
                    <SheetTitle className="flex items-center gap-2">
                        <Tag className="h-5 w-5" />
                        管理 Taint / Label
                    </SheetTitle>
                </SheetHeader>

                <div className="space-y-6">
                    {/* 选中信息 */}
                    <div className="flex items-center gap-2 p-3 rounded-lg bg-muted/50 border">
                        <CheckCircle2 className="h-4 w-4 text-primary" />
                        <span className="text-sm">
                            已选择 <span className="font-semibold">{ciCodes.length}</span> 个节点
                        </span>
                    </div>

                    {/* Exisiting Features Display */}
                    {isLoadingFeatures ? (
                        <div className="flex flex-col items-center justify-center py-12 space-y-2 text-muted-foreground/60">
                            <Loader2 className="h-8 w-8 animate-spin text-primary/60" />
                            <span className="text-sm">正在加载节点信息...</span>
                        </div>
                    ) : (
                        <div className="space-y-6">
                            {/* Labels */}
                            {mode === 'label' && (
                                <div className="space-y-4 animate-in fade-in zoom-in-95 duration-200">
                                    <div className="flex items-center justify-between">
                                        <h4 className="text-sm font-medium text-foreground flex items-center gap-2">
                                            <Tag className="h-4 w-4 text-blue-500" />
                                            现有 Labels
                                        </h4>
                                        <span className="text-xs text-muted-foreground bg-muted px-2 py-0.5 rounded-full">
                                            {aggregatedFeatures.commonLabels.length + aggregatedFeatures.nodeSpecificLabels.size} 项
                                        </span>
                                    </div>

                                    {/* 共同标签 */}
                                    {aggregatedFeatures.commonLabels.length > 0 && (
                                        <div className="bg-slate-50 dark:bg-slate-900/50 border border-slate-100 dark:border-slate-800 rounded-lg p-3 space-y-2">
                                            <div className="flex items-center gap-1.5 text-xs text-muted-foreground font-medium mb-1">
                                                <Layers className="h-3.5 w-3.5" />
                                                共同标签 (所有节点)
                                            </div>
                                            <div className="flex flex-wrap gap-2">
                                                {aggregatedFeatures.commonLabels.map((item, idx) => (
                                                    <Badge key={idx} variant="secondary" className="pl-2 pr-1 py-1 flex items-center gap-1.5 group hover:bg-secondary/80 transition-colors border-0 shadow-sm">
                                                        <span className="font-mono text-xs">{item.key}</span>
                                                        {item.value && (
                                                            <>
                                                                <span className="text-muted-foreground/40">|</span>
                                                                <span className="text-muted-foreground font-normal">{item.value}</span>
                                                            </>
                                                        )}
                                                        <Button
                                                            variant="ghost"
                                                            size="icon"
                                                            className="h-4 w-4 ml-0.5 lg:opacity-0 group-hover:opacity-100 transition-opacity hover:bg-red-100 hover:text-red-600 dark:hover:bg-red-900/30 dark:hover:text-red-400 rounded-full"
                                                            onClick={() => handleDeleteLabel(item.key)}
                                                            disabled={isSubmitting}
                                                        >
                                                            <X className="h-2.5 w-2.5" />
                                                        </Button>
                                                    </Badge>
                                                ))}
                                            </div>
                                        </div>
                                    )}

                                    {/* 按节点分组的标签 */}
                                    {aggregatedFeatures.nodeSpecificLabels.size > 0 && (
                                        <div className="space-y-2">
                                            <div className="flex items-center gap-1.5 text-xs text-muted-foreground font-medium px-1">
                                                <Server className="h-3.5 w-3.5" />
                                                节点特定标签
                                            </div>
                                            <div className="grid gap-3">
                                                {Array.from(aggregatedFeatures.nodeSpecificLabels.entries()).map(([nodeName, labels]) => (
                                                    <div key={nodeName} className="bg-background border rounded-lg p-3 flex flex-col sm:flex-row sm:items-start gap-3 hover:shadow-sm transition-shadow">
                                                        <div className="flex items-center gap-2 min-w-[140px] pt-1">
                                                            <div className="h-2 w-2 rounded-full bg-blue-500/50" />
                                                            <span className="text-xs font-medium text-foreground truncate" title={nodeName}>{nodeName}</span>
                                                        </div>
                                                        <div className="flex flex-wrap gap-2 flex-1">
                                                            {labels.map((item, idx) => (
                                                                <Badge key={idx} variant="outline" className="pl-2 pr-1 py-0.5 flex items-center gap-1 text-xs border-dashed text-muted-foreground hover:text-foreground transition-colors">
                                                                    <span>{item.key}</span>
                                                                    {item.value && <span className="opacity-60">={item.value}</span>}
                                                                    <Button
                                                                        variant="ghost"
                                                                        size="icon"
                                                                        className="h-3.5 w-3.5 ml-1 hover:bg-destructive/10 hover:text-destructive rounded-full"
                                                                        onClick={() => handleDeleteLabel(item.key)}
                                                                        disabled={isSubmitting}
                                                                    >
                                                                        <X className="h-2.5 w-2.5" />
                                                                    </Button>
                                                                </Badge>
                                                            ))}
                                                        </div>
                                                    </div>
                                                ))}
                                            </div>
                                        </div>
                                    )}

                                    {aggregatedFeatures.commonLabels.length === 0 && aggregatedFeatures.nodeSpecificLabels.size === 0 && (
                                        <div className="flex flex-col items-center justify-center py-8 text-center bg-slate-50 dark:bg-slate-900/50 rounded-lg border border-dashed">
                                            <Info className="h-8 w-8 text-muted-foreground/30 mb-2" />
                                            <p className="text-sm text-muted-foreground">该组节点暂无 Label</p>
                                        </div>
                                    )}
                                </div>
                            )}

                            {/* Taints */}
                            {mode === 'taint' && (
                                <div className="space-y-4 animate-in fade-in zoom-in-95 duration-200">
                                    <div className="flex items-center justify-between">
                                        <h4 className="text-sm font-medium text-foreground flex items-center gap-2">
                                            <ShieldAlert className="h-4 w-4 text-orange-500" />
                                            现有 Taints
                                        </h4>
                                        <span className="text-xs text-muted-foreground bg-muted px-2 py-0.5 rounded-full">
                                            {aggregatedFeatures.commonTaints.length + aggregatedFeatures.nodeSpecificTaints.size} 项
                                        </span>
                                    </div>

                                    {/* 共同污点 */}
                                    {aggregatedFeatures.commonTaints.length > 0 && (
                                        <div className="bg-orange-50/50 dark:bg-orange-950/10 border border-orange-100 dark:border-orange-900/30 rounded-lg p-3 space-y-2">
                                            <div className="flex items-center gap-1.5 text-xs text-orange-600/80 dark:text-orange-400/80 font-medium mb-1">
                                                <Layers className="h-3.5 w-3.5" />
                                                共同污点 (所有节点)
                                            </div>
                                            <div className="flex flex-wrap gap-2">
                                                {aggregatedFeatures.commonTaints.map((item, idx) => (
                                                    <Badge key={idx} variant="outline" className="pl-2 pr-1 py-1 flex items-center gap-1.5 bg-background/50 border-orange-200 dark:border-orange-900/50 text-orange-700 dark:text-orange-300">
                                                        <span className="font-mono text-xs font-semibold">{item.key}</span>
                                                        {item.value && (
                                                            <>
                                                                <span className="opacity-40">|</span>
                                                                <span className="opacity-80">{item.value}</span>
                                                            </>
                                                        )}
                                                        <span className="text-[10px] px-1.5 py-0.5 rounded-sm bg-orange-100/50 dark:bg-orange-900/30 font-medium uppercase tracking-wider opacity-80">
                                                            {item.effect === 'NoSchedule' ? 'NoSched' : item.effect === 'NoExecute' ? 'NoExec' : 'PrefNoSched'}
                                                        </span>
                                                        <Button
                                                            variant="ghost"
                                                            size="icon"
                                                            className="h-4 w-4 ml-0.5 hover:bg-orange-200/50 dark:hover:bg-orange-900/50 hover:text-orange-700 dark:hover:text-orange-300 rounded-full"
                                                            onClick={() => handleDeleteTaint(item.key, item.value, item.effect)}
                                                            disabled={isSubmitting}
                                                        >
                                                            <X className="h-3 w-3" />
                                                        </Button>
                                                    </Badge>
                                                ))}
                                            </div>
                                        </div>
                                    )}

                                    {/* 按节点分组的污点 */}
                                    {aggregatedFeatures.nodeSpecificTaints.size > 0 && (
                                        <div className="space-y-2">
                                            <div className="flex items-center gap-1.5 text-xs text-muted-foreground font-medium px-1">
                                                <Server className="h-3.5 w-3.5" />
                                                节点特定污点
                                            </div>
                                            <div className="grid gap-3">
                                                {Array.from(aggregatedFeatures.nodeSpecificTaints.entries()).map(([nodeName, taints]) => (
                                                    <div key={nodeName} className="bg-background border border-orange-100 dark:border-orange-900/20 rounded-lg p-3 flex flex-col sm:flex-row sm:items-start gap-3 hover:shadow-sm transition-shadow">
                                                        <div className="flex items-center gap-2 min-w-[140px] pt-1">
                                                            <div className="h-2 w-2 rounded-full bg-orange-500/50" />
                                                            <span className="text-xs font-medium text-foreground truncate" title={nodeName}>{nodeName}</span>
                                                        </div>
                                                        <div className="flex flex-wrap gap-2 flex-1">
                                                            {taints.map((item, idx) => (
                                                                <Badge key={idx} variant="outline" className="pl-2 pr-1 py-0.5 flex items-center gap-1 text-xs border-dashed border-orange-200 dark:border-orange-900/40 text-muted-foreground hover:text-foreground transition-colors">
                                                                    <span>{item.key}</span>
                                                                    {item.value && <span className="opacity-60">={item.value}</span>}
                                                                    <span className="text-[9px] uppercase opacity-50 border px-0.5 rounded ml-0.5">
                                                                        {item.effect === 'NoSchedule' ? 'NS' : item.effect === 'NoExecute' ? 'NE' : 'PNS'}
                                                                    </span>
                                                                    <Button
                                                                        variant="ghost"
                                                                        size="icon"
                                                                        className="h-3.5 w-3.5 ml-1 hover:bg-destructive/10 hover:text-destructive rounded-full"
                                                                        onClick={() => handleDeleteTaint(item.key, item.value, item.effect)}
                                                                        disabled={isSubmitting}
                                                                    >
                                                                        <X className="h-2.5 w-2.5" />
                                                                    </Button>
                                                                </Badge>
                                                            ))}
                                                        </div>
                                                    </div>
                                                ))}
                                            </div>
                                        </div>
                                    )}

                                    {aggregatedFeatures.commonTaints.length === 0 && aggregatedFeatures.nodeSpecificTaints.size === 0 && (
                                        <div className="flex flex-col items-center justify-center py-8 text-center bg-slate-50 dark:bg-slate-900/50 rounded-lg border border-dashed border-orange-200/50 dark:border-orange-900/20">
                                            <Info className="h-8 w-8 text-orange-500/20 mb-2" />
                                            <p className="text-sm text-muted-foreground">该组节点暂无 Taint</p>
                                        </div>
                                    )}
                                </div>
                            )}
                        </div>
                    )}

                    <Separator />


                    <Tabs defaultValue="label" className="w-full">
                        <TabsList className="grid w-full grid-cols-2">
                            <TabsTrigger value="label" onClick={() => setMode('label')}>
                                <Tag className="h-4 w-4 mr-2" />
                                Label 管理
                            </TabsTrigger>
                            <TabsTrigger value="taint" onClick={() => setMode('taint')}>
                                <AlertTriangle className="h-4 w-4 mr-2" />
                                Taint 管理
                            </TabsTrigger>
                        </TabsList>

                        <div className="mt-4 bg-muted/30 p-4 rounded-xl border space-y-4">
                            <TabsContent value="label" className="mt-0 space-y-4">
                                <div className="flex items-center justify-between">
                                    <div className="inline-flex rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-900 p-0.5 shadow-sm">
                                        <button
                                            onClick={() => setAction('add')}
                                            className={cn(
                                                "relative px-3 py-1.5 text-xs font-medium rounded-md transition-all duration-200",
                                                action === 'add'
                                                    ? "bg-gradient-to-r from-emerald-500 to-teal-500 text-white shadow-md shadow-emerald-500/25"
                                                    : "text-slate-600 dark:text-slate-400 hover:text-emerald-600 hover:bg-emerald-50 dark:hover:bg-emerald-950/30"
                                            )}
                                        >
                                            添加 / 更新
                                        </button>
                                        <button
                                            onClick={() => setAction('remove')}
                                            className={cn(
                                                "relative px-3 py-1.5 text-xs font-medium rounded-md transition-all duration-200",
                                                action === 'remove'
                                                    ? "bg-gradient-to-r from-rose-500 to-red-500 text-white shadow-md shadow-rose-500/25"
                                                    : "text-slate-600 dark:text-slate-400 hover:text-rose-600 hover:bg-rose-50 dark:hover:bg-rose-950/30"
                                            )}
                                        >
                                            批量删除
                                        </button>
                                    </div>
                                    <span className="text-xs text-muted-foreground">
                                        {action === 'add' ? '添加或更新所选节点的 Label' : '移除所选节点的特定 Label'}
                                    </span>
                                </div>

                                <div className="flex items-center gap-2 animate-in fade-in duration-200">
                                    <ConfigCombobox
                                        type="label"
                                        value={labelKey}
                                        onChange={setLabelKey}
                                        placeholder="Label Key (select or type)"
                                        className="flex-1"
                                    />
                                    <span className="text-muted-foreground">=</span>
                                    <ConfigCombobox
                                        type="label-value"
                                        parentKey={labelKey}
                                        value={labelValue}
                                        onChange={setLabelValue}
                                        placeholder="Label Value"
                                        className="flex-1"
                                        disabled={action === 'remove'}
                                    />
                                    <Button onClick={handleSubmit} disabled={isSubmitting}>
                                        {isSubmitting ? <Loader2 className="h-4 w-4 animate-spin" /> : <Check className="h-4 w-4" />}
                                        执行
                                    </Button>
                                </div>
                            </TabsContent>

                            <TabsContent value="taint" className="mt-0 space-y-4">
                                <div className="flex items-center justify-between">
                                    <div className="inline-flex rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-900 p-0.5 shadow-sm">
                                        <button
                                            onClick={() => setAction('add')}
                                            className={cn(
                                                "relative px-3 py-1.5 text-xs font-medium rounded-md transition-all duration-200",
                                                action === 'add'
                                                    ? "bg-gradient-to-r from-emerald-500 to-teal-500 text-white shadow-md shadow-emerald-500/25"
                                                    : "text-slate-600 dark:text-slate-400 hover:text-emerald-600 hover:bg-emerald-50 dark:hover:bg-emerald-950/30"
                                            )}
                                        >
                                            添加 / 更新
                                        </button>
                                        <button
                                            onClick={() => setAction('remove')}
                                            className={cn(
                                                "relative px-3 py-1.5 text-xs font-medium rounded-md transition-all duration-200",
                                                action === 'remove'
                                                    ? "bg-gradient-to-r from-rose-500 to-red-500 text-white shadow-md shadow-rose-500/25"
                                                    : "text-slate-600 dark:text-slate-400 hover:text-rose-600 hover:bg-rose-50 dark:hover:bg-rose-950/30"
                                            )}
                                        >
                                            批量删除
                                        </button>
                                    </div>
                                    <span className="text-xs text-muted-foreground">
                                        {action === 'add' ? '设置或更新所选节点的 Taint' : '移除所选节点的特定 Taint'}
                                    </span>
                                </div>

                                <div className="space-y-4 animate-in fade-in duration-200">
                                    <div className="flex items-center gap-2">
                                        <ConfigCombobox
                                            type="taint"
                                            value={taintKey}
                                            onChange={setTaintKey}
                                            placeholder="Taint Key"
                                            className="flex-[2]"
                                        />
                                        <span className="text-muted-foreground font-light">=</span>
                                        <ConfigCombobox
                                            type="taint-value"
                                            parentKey={taintKey}
                                            value={taintValue}
                                            onChange={setTaintValue}
                                            placeholder="Value"
                                            className="flex-[2]"
                                            disabled={action === 'remove'}
                                        />
                                        <span className="text-muted-foreground font-light">:</span>
                                        <Select value={taintEffect} onValueChange={(v) => setTaintEffect(v as any)}>
                                            <SelectTrigger className="flex-[1.5] min-w-[140px]">
                                                <SelectValue />
                                            </SelectTrigger>
                                            <SelectContent>
                                                <SelectItem value="NoSchedule">NoSchedule</SelectItem>
                                                <SelectItem value="PreferNoSchedule">PreferNoSchedule</SelectItem>
                                                <SelectItem value="NoExecute">NoExecute</SelectItem>
                                            </SelectContent>
                                        </Select>
                                    </div>

                                    <div className="flex justify-end">
                                        <Button
                                            onClick={handleSubmit}
                                            disabled={isSubmitting}
                                            className={cn(
                                                "w-full sm:w-auto min-w-[120px]",
                                                action === 'remove' ? "bg-destructive hover:bg-destructive/90" : ""
                                            )}
                                        >
                                            {isSubmitting ? (
                                                <Loader2 className="h-4 w-4 animate-spin mr-2" />
                                            ) : (
                                                action === 'remove' ? <X className="h-4 w-4 mr-2" /> : <Check className="h-4 w-4 mr-2" />
                                            )}
                                            {action === 'remove' ? '确认移除' : '执行操作'}
                                        </Button>
                                    </div>
                                </div>
                            </TabsContent>
                        </div>
                    </Tabs>
                </div>
            </SheetContent>
        </Sheet>
    )
}

function BulkK8sConfirmDialog({
    open,
    onOpenChange,
    devices,
    onConfirm,
    operation,
}: {
    open: boolean
    onOpenChange: (open: boolean) => void
    devices: NavyDevice[]
    onConfirm: (ciCodes: string[]) => void
    operation: 'cordon' | 'uncordon' | 'drain'
}) {
    const [targetDevices, setTargetDevices] = useState(devices)
    const [nodeInfos, setNodeInfos] = useState<Map<string, NodeLabelTaintResponse>>(new Map())
    const [loadingInfos, setLoadingInfos] = useState(false)

    useEffect(() => {
        if (open) {
            setTargetDevices(devices)

            // 仅 Drain 操作需要查询 Cordon 时间
            if (operation === 'drain' && devices.length > 0) {
                fetchNodeInfos(devices)
            }
        } else {
            setNodeInfos(new Map())
        }
    }, [open, devices, operation])

    const fetchNodeInfos = async (targets: NavyDevice[]) => {
        setLoadingInfos(true)
        try {
            const results = new Map<string, NodeLabelTaintResponse>()
            // 按集群分组获取以减少并发（虽然目前 API 是单点的，但逻辑上清晰）
            // 这里简单并发获取
            await Promise.all(targets.map(async (d) => {
                if (!d.cluster) return
                try {
                    const info = await RobustaAPI.getNodeLabelsAndTaints(d.cluster, d.ci_code)
                    results.set(d.ci_code, info)
                } catch (e) {
                    // ignore individual errors
                }
            }))
            setNodeInfos(results)
        } finally {
            setLoadingInfos(false)
        }
    }

    const clusters = useMemo(() => {
        const map = new Map<string, NavyDevice[]>()
        targetDevices.forEach(d => {
            const c = d.cluster || 'Unknown'
            const list = map.get(c) || []
            list.push(d)
            map.set(c, list)
        })
        return map
    }, [targetDevices])

    const clusterKeys = Array.from(clusters.keys())
    const isValid = clusterKeys.length <= 1 && targetDevices.length > 0
    const hasConflict = clusterKeys.length > 1

    const handleRemoveDevice = (ciCode: string) => {
        setTargetDevices(prev => prev.filter(d => d.ci_code !== ciCode))
    }

    const handleRemoveCluster = (clusterName: string) => {
        setTargetDevices(prev => prev.filter(d => d.cluster !== clusterName))
    }

    const config = {
        drain: {
            title: 'Safe Drain 节点确认',
            icon: DoorOpen,
            color: 'text-orange-600',
            btnClass: 'bg-orange-600 hover:bg-orange-700',
            description: '驱逐',
        },
        cordon: {
            title: 'Cordon 节点确认',
            icon: Ban,
            color: 'text-amber-600',
            btnClass: 'bg-amber-600 hover:bg-amber-700',
            description: '禁止调度',
        },
        uncordon: {
            title: 'Uncordon 节点确认',
            icon: PlayCircle,
            color: 'text-emerald-600',
            btnClass: 'bg-emerald-600 hover:bg-emerald-700',
            description: '恢复调度',
        }
    }[operation]

    const activeConfig = config || { title: '操作确认', icon: AlertTriangle, color: 'text-primary', btnClass: '', description: '操作' }
    const Icon = activeConfig.icon

    return (
        <AlertDialog open={open} onOpenChange={onOpenChange}>
            <AlertDialogContent className="max-w-3xl">
                <AlertDialogHeader>
                    <AlertDialogTitle className="flex items-center gap-2">
                        <Icon className={`h-5 w-5 ${activeConfig.color}`} />
                        {activeConfig.title}
                    </AlertDialogTitle>
                    <AlertDialogDescription>
                        请确认要对节点执行 {activeConfig.description} 操作。
                        {hasConflict ? (
                            <div className="mt-3 p-3 bg-red-50 dark:bg-red-950/20 text-red-600 dark:text-red-400 rounded-lg text-sm flex items-start gap-2 border border-red-200 dark:border-red-900/30">
                                <AlertTriangle className="h-5 w-5 shrink-0" />
                                <div>
                                    <p className="font-semibold">检测到多个集群的节点</p>
                                    <p className="opacity-90">为保证安全，一次只能批量操作一个集群的节点。请移除多余集群的节点。</p>
                                </div>
                            </div>
                        ) : (
                            <div className="mt-1">
                                将对以下 <span className="font-mono font-medium text-foreground">{targetDevices.length}</span> 个节点执行 {activeConfig.description} 操作。
                            </div>
                        )}
                    </AlertDialogDescription>
                </AlertDialogHeader>

                <div className="my-4 max-h-[50vh] overflow-y-auto border rounded-lg bg-muted/20">
                    {clusterKeys.map(clusterName => (
                        <div key={clusterName} className="border-b last:border-b-0">
                            <div className="bg-muted/50 px-4 py-2 flex items-center justify-between sticky top-0 z-10 backdrop-blur-sm">
                                <div className="flex items-center gap-2">
                                    <Hexagon className="h-4 w-4 text-blue-500" />
                                    <span className="font-medium text-sm">{clusterName}</span>
                                    <Badge variant="outline" className="text-xs">{clusters.get(clusterName)?.length}</Badge>
                                </div>
                                {hasConflict && (
                                    <Button
                                        variant="ghost"
                                        size="sm"
                                        onClick={() => handleRemoveCluster(clusterName)}
                                        className="h-7 text-xs text-red-600 hover:text-red-700 hover:bg-red-50 dark:hover:bg-red-950/20"
                                    >
                                        <Trash2 className="h-3.5 w-3.5 mr-1.5" />
                                        移除此集群所有节点
                                    </Button>
                                )}
                            </div>
                            <div className="divide-y">
                                {clusters.get(clusterName)?.map((device, idx) => (
                                    <div key={device.ci_code} className="px-4 py-3 flex items-center justify-between hover:bg-muted/30 transition-colors group">
                                        <div className="flex items-center gap-4">
                                            <span className="text-muted-foreground font-mono text-xs w-6 text-center">{idx + 1}</span>
                                            <div>
                                                <div className="font-medium text-sm flex items-center gap-2">
                                                    {device.ci_code}
                                                    <Badge variant="secondary" className="text-[10px] h-4 px-1 rounded-sm font-normal text-muted-foreground">{device.arch_type}</Badge>
                                                </div>
                                                <div className="text-xs text-muted-foreground font-mono mt-0.5">{device.ip}</div>
                                            </div>
                                        </div>

                                        <div className="flex items-center gap-3">
                                            {operation === 'drain' && (
                                                <div className="flex justify-end">
                                                    {(() => {
                                                        const info = nodeInfos.get(device.ci_code)
                                                        if (loadingInfos) return <span className="text-[10px] text-muted-foreground animate-pulse">正在查询状态...</span>

                                                        const unschedulableTaint = info?.taints.find(t => t.key === 'node.kubernetes.io/unschedulable')
                                                        if (unschedulableTaint) {
                                                            const timeAdded = unschedulableTaint.timeAdded
                                                            return (
                                                                <div className="text-[10px] flex items-center gap-1.5 text-amber-600 dark:text-amber-500 bg-amber-50 dark:bg-amber-950/30 px-2 py-1 rounded">
                                                                    <Ban className="h-3 w-3" />
                                                                    <span>
                                                                        已 Cordon
                                                                        {timeAdded ? ` (${new Date(timeAdded).toLocaleString()})` : ''}
                                                                    </span>
                                                                </div>
                                                            )
                                                        } else if (info) {
                                                            return (
                                                                <div className="text-[10px] flex items-center gap-1.5 text-emerald-600 dark:text-emerald-500 bg-emerald-50 dark:bg-emerald-950/30 px-2 py-1 rounded">
                                                                    <CheckCircle2 className="h-3 w-3" />
                                                                    <span>可调度 (未Cordon)</span>
                                                                </div>
                                                            )
                                                        }
                                                        return null
                                                    })()}
                                                </div>
                                            )}

                                            <Button
                                                variant="ghost"
                                                size="icon"
                                                onClick={() => handleRemoveDevice(device.ci_code)}
                                                className="h-8 w-8 opacity-0 group-hover:opacity-100 hover:bg-destructive/10 hover:text-destructive rounded-full transition-all"
                                            >
                                                <X className="h-4 w-4" />
                                            </Button>
                                        </div>
                                    </div>
                                ))}
                            </div>
                        </div>
                    ))}
                </div>

                <AlertDialogFooter>
                    <AlertDialogCancel>取消</AlertDialogCancel>
                    <AlertDialogAction
                        disabled={!isValid}
                        onClick={(e) => {
                            if (!isValid) {
                                e.preventDefault()
                                return
                            }
                            onConfirm(targetDevices.map(d => d.ci_code))
                        }}
                        className={activeConfig.btnClass}
                    >
                        确认执行 ({targetDevices.length})
                    </AlertDialogAction>
                </AlertDialogFooter>
            </AlertDialogContent>
        </AlertDialog>
    )
}

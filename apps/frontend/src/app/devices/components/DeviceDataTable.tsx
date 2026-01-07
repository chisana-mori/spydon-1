'use client'

import { useState, useEffect } from 'react'
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
import {
    HoverCard,
    HoverCardContent,
    HoverCardTrigger,
} from "@/components/ui/hover-card"
import {
    Laptop,
    Globe,
    Database,
    Star,
    Copy,
    Check,
    Tag,
    AlertTriangle,
    Loader2,
    Edit,
    Search,
    Play,
    ExternalLink,
} from 'lucide-react'
import { NavyDevice, DeviceFeatureDetails } from "@/types/navy"
import { formatDistanceToNow } from 'date-fns'
import { zhCN } from 'date-fns/locale'
import { cn } from "@/lib/utils"
import { toast } from 'sonner'
import RobustaAPI from '@/lib/api'
import {
    Dialog,
    DialogContent,
    DialogHeader,
    DialogTitle,
    DialogFooter,
    DialogDescription,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Checkbox } from '@/components/ui/checkbox'


interface DeviceDataTableProps {
    devices: NavyDevice[]
    isLoading: boolean
    onSelect: (device: NavyDevice) => void
    onRefresh: () => void
    selectedId?: number
    selectedDevices: Set<number>
    onSelectionChange: (selected: Set<number>) => void
}

// 特殊设备悬浮卡片组件
function SpecialDeviceHoverCard({ device }: { device: NavyDevice }) {
    const [featureDetails, setFeatureDetails] = useState<DeviceFeatureDetails | null>(null)
    const [filterOptions, setFilterOptions] = useState<{ labelKeys: string[]; taintKeys: string[] } | null>(null)
    const [isLoading, setIsLoading] = useState(false)
    const [isOpen, setIsOpen] = useState(false)

    // 当卡片打开时加载特性详情和筛选选项
    useEffect(() => {
        if (isOpen && !featureDetails && !isLoading && device.ci_code) {
            setIsLoading(true)

            // 并行加载特性详情和筛选选项
            Promise.all([
                RobustaAPI.getBatchDeviceFeatures([device.ci_code]),
                RobustaAPI.getNavyFilterOptions()
            ])
                .then(([details, options]) => {
                    setFeatureDetails(details)
                    setFilterOptions(options)
                })
                .catch(err => {
                    console.error('加载设备特性失败:', err)
                })
                .finally(() => {
                    setIsLoading(false)
                })
        }
    }, [isOpen, device.ci_code, featureDetails, isLoading])

    // 过滤受管理的标签和污点
    const managedLabels = featureDetails?.labels?.filter(label =>
        filterOptions?.labelKeys?.includes(label.key)
    ) || []

    const managedTaints = featureDetails?.taints?.filter(taint =>
        filterOptions?.taintKeys?.includes(taint.key)
    ) || []

    const hasAnyFeatures = managedLabels.length > 0 || managedTaints.length > 0

    return (
        <HoverCard open={isOpen} onOpenChange={setIsOpen}>
            <HoverCardTrigger asChild>
                <div className="cursor-help">
                    <Star className="h-4 w-4 text-amber-500 fill-amber-500 mt-0.5 flex-shrink-0" />
                </div>
            </HoverCardTrigger>
            <HoverCardContent className="w-80 p-0 overflow-hidden" align="start">
                <div className="bg-amber-50 dark:bg-amber-950/30 p-3 border-b border-amber-100 dark:border-amber-900/50 flex items-start gap-3">
                    <div className="p-2 bg-amber-100/50 dark:bg-amber-900/50 rounded-lg shrink-0">
                        <Star className="h-5 w-5 text-amber-600 dark:text-amber-500 fill-amber-600 dark:fill-amber-500" />
                    </div>
                    <div>
                        <h4 className="text-sm font-semibold text-amber-900 dark:text-amber-100 mb-0.5">特殊设备</h4>
                        <p className="text-xs text-amber-700 dark:text-amber-300/80">
                            此设备已被标记为特殊资产
                        </p>
                    </div>
                </div>

                <div className="p-4 space-y-4">
                    {/* 基础配置 */}
                    <div className="grid grid-cols-2 gap-3">
                        <div className="space-y-1">
                            <span className="text-[10px] uppercase tracking-wider text-muted-foreground font-semibold">机器用途</span>
                            <p className="text-xs font-medium line-clamp-1">{device.group || '-'}</p>
                        </div>
                        <div className="space-y-1">
                            <span className="text-[10px] uppercase tracking-wider text-muted-foreground font-semibold">关联应用</span>
                            <p className="text-xs font-medium line-clamp-1">{device.app_name || '-'}</p>
                        </div>
                    </div>

                    {/* 节点特性信息 */}
                    <div className="space-y-3 pt-2 border-t border-border/50">
                        {isLoading ? (
                            <div className="flex items-center gap-2 py-4 justify-center">
                                <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
                                <span className="text-xs text-muted-foreground">加载节点特性...</span>
                            </div>
                        ) : hasAnyFeatures ? (
                            <>
                                {managedLabels.length > 0 && (
                                    <div className="space-y-1.5">
                                        <div className="flex items-center gap-1 text-[10px] font-semibold text-muted-foreground">
                                            <Tag className="h-3 w-3" />
                                            <span>受管理标签 ({managedLabels.length})</span>
                                        </div>
                                        <div className="flex flex-wrap gap-1.5">
                                            {managedLabels.map((l, i) => (
                                                <Badge
                                                    key={i}
                                                    variant="secondary"
                                                    className="px-1.5 h-5 text-[10px] bg-blue-500/10 text-blue-600 border border-blue-500/20"
                                                >
                                                    {l.key}: {l.value}
                                                </Badge>
                                            ))}
                                        </div>
                                    </div>
                                )}

                                {managedTaints.length > 0 && (
                                    <div className="space-y-1.5">
                                        <div className="flex items-center gap-1 text-[10px] font-semibold text-muted-foreground">
                                            <AlertTriangle className="h-3 w-3" />
                                            <span>受管理污点 ({managedTaints.length})</span>
                                        </div>
                                        <div className="flex flex-wrap gap-1.5">
                                            {managedTaints.map((t, i) => (
                                                <Badge
                                                    key={i}
                                                    variant="secondary"
                                                    className="px-1.5 h-5 text-[10px] bg-orange-500/10 text-orange-600 border border-orange-500/20"
                                                >
                                                    {t.key}={t.value}:{t.effect}
                                                </Badge>
                                            ))}
                                        </div>
                                    </div>
                                )}
                            </>
                        ) : (
                            <p className="text-[10px] text-center py-4 text-muted-foreground italic">暂无额外的受管理特性信息</p>
                        )}
                    </div>
                </div>

                <div className="bg-muted/30 p-3 border-t border-border/50 flex items-center justify-between text-[10px]">
                    <span className="text-muted-foreground">更新于 {device.updated_at ? formatDistanceToNow(new Date(device.updated_at), { addSuffix: true, locale: zhCN }) : '-'}</span>
                </div>
            </HoverCardContent>
        </HoverCard>
    )
}

/**
 * 编辑机器用途的弹窗组件
 */
function EditGroupDialog({
    device,
    open,
    onOpenChange,
    onSuccess,
}: {
    device: NavyDevice | null
    open: boolean
    onOpenChange: (open: boolean) => void
    onSuccess: () => void
}) {
    const [searchValue, setSearchValue] = useState('')
    const [options, setOptions] = useState<string[]>([])
    const [isLoadingOptions, setIsLoadingOptions] = useState(false)
    const [isSaving, setIsSaving] = useState(false)

    // 加载已有选项
    useEffect(() => {
        if (open) {
            const fetchOptions = async () => {
                setIsLoadingOptions(true)
                try {
                    const values = await RobustaAPI.getDeviceFieldValues('group')
                    setOptions(values.filter(v => v && v.trim() !== ''))
                } catch (err) {
                    console.error('加载用途选项失败:', err)
                } finally {
                    setIsLoadingOptions(false)
                }
            }
            fetchOptions()
            setSearchValue(device?.group || '')
        }
    }, [open, device])

    const handleSave = async (value: string) => {
        if (!device) return
        setIsSaving(true)
        try {
            await RobustaAPI.updateNavyDeviceGroup(device.id, value)
            toast.success('更新成功')
            onSuccess()
            onOpenChange(false)
        } catch (err) {
            toast.error('更新失败')
            console.error(err)
        } finally {
            setIsSaving(false)
        }
    }

    const filteredOptions = options.filter(opt =>
        opt.toLowerCase().includes(searchValue.toLowerCase())
    )

    return (
        <Dialog open={open} onOpenChange={onOpenChange}>
            <DialogContent className="sm:max-w-[425px]">
                <DialogHeader>
                    <DialogTitle>编辑机器用途</DialogTitle>
                    <DialogDescription>
                        设置设备的机器类别或主要业务用途
                    </DialogDescription>
                </DialogHeader>

                <div className="space-y-4 py-4">
                    <div className="space-y-2">
                        <Label htmlFor="purpose-input">手动输入用途</Label>
                        <div className="relative">
                            <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
                            <Input
                                id="purpose-input"
                                placeholder="输入新的用途描述..."
                                value={searchValue}
                                onChange={(e) => setSearchValue(e.target.value)}
                                className="pl-9"
                                onKeyDown={(e) => {
                                    if (e.key === 'Enter') handleSave(searchValue)
                                }}
                            />
                        </div>
                    </div>

                    <div className="space-y-2">
                        <Label>已有用途</Label>
                        <ScrollArea className="h-[200px] rounded-md border p-1">
                            <div className="grid grid-cols-1 gap-1">
                                {filteredOptions.length > 0 ? (
                                    filteredOptions.map((opt) => (
                                        <Button
                                            key={opt}
                                            variant={searchValue === opt ? "secondary" : "ghost"}
                                            className={cn(
                                                "w-full justify-between font-normal h-9",
                                                searchValue === opt && "bg-secondary"
                                            )}
                                            onClick={() => {
                                                setSearchValue(opt)
                                            }}
                                            onDoubleClick={() => handleSave(opt)}
                                        >
                                            <span className="truncate">{opt}</span>
                                            {searchValue === opt && <Check className="h-4 w-4 shrink-0 text-primary" />}
                                        </Button>
                                    ))
                                ) : (
                                    <div className="flex flex-col items-center justify-center py-8 text-muted-foreground">
                                        <Tag className="h-8 w-8 mb-2 opacity-20" />
                                        <p className="text-xs">无匹配选项</p>
                                    </div>
                                )}
                            </div>
                        </ScrollArea>
                    </div>
                </div>

                <DialogFooter>
                    <Button variant="outline" onClick={() => onOpenChange(false)}>
                        取消
                    </Button>
                    <Button
                        onClick={() => handleSave(searchValue)}
                        disabled={isSaving}
                    >
                        {isSaving && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                        确定
                    </Button>
                </DialogFooter>
            </DialogContent>
        </Dialog>
    )
}

export function DeviceDataTable({ devices, isLoading, onSelect, onRefresh, selectedId, selectedDevices, onSelectionChange }: DeviceDataTableProps) {
    const [editingDevice, setEditingDevice] = useState<NavyDevice | null>(null)
    const [isEditDialogOpen, setIsEditDialogOpen] = useState(false)

    // 可选中的设备（非虚拟设备）
    const selectableDevices = devices.filter(d => !d.isVirtual)
    const allSelected = selectableDevices.length > 0 && selectableDevices.every(d => selectedDevices.has(d.id))
    const someSelected = selectableDevices.some(d => selectedDevices.has(d.id))

    const handleSelectAll = (checked: boolean) => {
        if (checked) {
            const newSelected = new Set(selectedDevices)
            selectableDevices.forEach(d => newSelected.add(d.id))
            onSelectionChange(newSelected)
        } else {
            const newSelected = new Set(selectedDevices)
            selectableDevices.forEach(d => newSelected.delete(d.id))
            onSelectionChange(newSelected)
        }
    }

    const handleSelectOne = (deviceId: number, checked: boolean) => {
        const newSelected = new Set(selectedDevices)
        if (checked) {
            newSelected.add(deviceId)
        } else {
            newSelected.delete(deviceId)
        }
        onSelectionChange(newSelected)
    }

    const copyToClipboard = (text: string, label: string) => {
        navigator.clipboard.writeText(text)
        toast.success(`已复制 ${label}`)
    }

    const getArchBadge = (arch: string) => {
        if (!arch) return null
        const isArm = arch.toLowerCase().includes('arm') || arch.toLowerCase().includes('aarch64')
        return (
            <Badge
                variant="outline"
                className={cn(
                    "text-[10px] px-1.5 h-4 font-mono uppercase",
                    isArm ? "bg-amber-50 text-amber-700 border-amber-200" : "bg-blue-50 text-blue-700 border-blue-200"
                )}
            >
                {arch}
            </Badge>
        )
    }

    const getRowClassName = (device: NavyDevice) => {
        const isSelected = selectedId === device.id
        if (isSelected) return "bg-blue-100/60 dark:bg-blue-900/40 border-blue-300 dark:border-blue-700"
        if (device.isVirtual) return "opacity-60 bg-muted/30"

        // 1. 特殊设备或有应用名称：浅黄色 (Amber)
        if (device.is_special || (device.app_name && device.app_name.trim() !== '')) {
            return "bg-amber-100/40 dark:bg-amber-900/20 hover:bg-amber-100/60 dark:hover:bg-amber-800/30"
        }

        // 2. 有关联集群：浅绿色 (Emerald)
        if (device.cluster && device.cluster.trim() !== '') {
            return "bg-emerald-100/30 dark:bg-emerald-950/15 hover:bg-emerald-100/60 dark:hover:bg-emerald-900/30"
        }

        // 3. 其他：白色 (Default)
        return "hover:bg-muted/50"
    }

    if (isLoading && devices.length === 0) {
        return (
            <div className="h-64 flex flex-col items-center justify-center gap-2">
                <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
                <p className="text-sm text-muted-foreground">加载设备列表中...</p>
            </div>
        )
    }

    return (
        <div className="rounded-xl border border-border/50 bg-card overflow-hidden shadow-sm">
            <Table>
                <TableHeader className="bg-muted/30">
                    <TableRow className="hover:bg-transparent border-border/50">
                        <TableHead className="w-10 text-center">
                            <Checkbox
                                checked={allSelected}
                                onCheckedChange={handleSelectAll}
                                aria-label="全选"
                                className={cn(someSelected && !allSelected && "data-[state=checked]:bg-primary/50")}
                            />
                        </TableHead>
                        <TableHead className="text-xs uppercase tracking-wider font-medium text-muted-foreground min-w-[200px]">
                            <div className="flex items-center gap-1.5">
                                设备ID
                                <Button
                                    variant="ghost"
                                    size="icon"
                                    className="h-6 w-6 text-muted-foreground/50 hover:text-foreground"
                                    onClick={(e) => {
                                        e.stopPropagation()
                                        const ids = devices.filter(d => !d.isVirtual).map(d => d.ci_code).join('\n')
                                        copyToClipboard(ids, '本页所有设备ID')
                                    }}
                                >
                                    <Copy className="h-3 w-3" />
                                </Button>
                            </div>
                        </TableHead>
                        <TableHead className="text-xs uppercase tracking-wider font-medium text-muted-foreground w-40">
                            <div className="flex items-center gap-1.5">
                                IP
                                <Button
                                    variant="ghost"
                                    size="icon"
                                    className="h-6 w-6 text-muted-foreground/50 hover:text-foreground"
                                    onClick={() => copyToClipboard(devices.map(d => d.ip).join('\n'), '本页所有 IP')}
                                >
                                    <Copy className="h-3 w-3" />
                                </Button>
                            </div>
                        </TableHead>
                        <TableHead className="text-xs uppercase tracking-wider font-medium text-muted-foreground w-[100px]">K8s 状态</TableHead>
                        <TableHead className="text-xs uppercase tracking-wider font-medium text-muted-foreground w-[120px]">角色</TableHead>
                        <TableHead className="text-xs uppercase tracking-wider font-medium text-muted-foreground w-[120px]">关联集群</TableHead>
                        <TableHead className="text-xs uppercase tracking-wider font-medium text-muted-foreground w-[150px]">用途</TableHead>
                        <TableHead className="text-xs uppercase tracking-wider font-medium text-muted-foreground w-[140px]">IDC / 房间</TableHead>
                        <TableHead className="text-xs uppercase tracking-wider font-medium text-muted-foreground w-[80px]">Zone</TableHead>
                        <TableHead className="text-xs uppercase tracking-wider font-medium text-muted-foreground w-[100px]">AppID</TableHead>
                        <TableHead className="text-xs uppercase tracking-wider font-medium text-muted-foreground whitespace-nowrap">国产化</TableHead>
                        <TableHead className="text-xs uppercase tracking-wider font-medium text-muted-foreground w-[80px]">状态</TableHead>
                    </TableRow>
                </TableHeader>
                <TableBody>
                    {devices.map((device, index) => (
                        <TableRow
                            key={`${device.id}-${index}`}
                            className={cn("cursor-pointer transition-colors border-border/40", getRowClassName(device))}
                            onClick={() => onSelect(device)}
                        >
                            <TableCell className="text-center" onClick={(e) => e.stopPropagation()}>
                                {!device.isVirtual && (
                                    <Checkbox
                                        checked={selectedDevices.has(device.id)}
                                        onCheckedChange={(checked) => handleSelectOne(device.id, checked as boolean)}
                                        aria-label={`选择 ${device.ci_code}`}
                                        className="mt-0.5"
                                    />
                                )}
                            </TableCell>
                            <TableCell>
                                <div className="flex items-start gap-2 max-w-[220px]">
                                    {device.isVirtual ? (
                                        <div className="h-4 w-4 shrink-0" />
                                    ) : device.is_special ? (
                                        <SpecialDeviceHoverCard device={device} />
                                    ) : (
                                        <Laptop className="h-4 w-4 text-muted-foreground mt-0.5 flex-shrink-0" />
                                    )}
                                    <div className="min-w-0 flex-1">
                                        <div className="flex items-center gap-2">
                                            <p className={cn("font-mono text-sm truncate", device.isVirtual && "text-muted-foreground italic")}>
                                                {device.isVirtual ? device.originalKeyword : device.ci_code}
                                                {device.isExactMatch && !device.isVirtual && device.ci_code?.toLowerCase() === device.originalKeyword?.toLowerCase() && (
                                                    <Check className="inline-block w-3.5 h-3.5 ml-1.5 text-green-600 align-middle mb-0.5" strokeWidth={3} />
                                                )}
                                            </p>
                                            {!device.isVirtual && device.cluster && (
                                                <a
                                                    href={`/kite/nodes/${device.ci_code}?tab=overview&cluster=${device.cluster}`}
                                                    target="_blank"
                                                    rel="noopener noreferrer"
                                                    onClick={(e) => e.stopPropagation()}
                                                    className="inline-flex items-center justify-center h-5 w-5 rounded-md hover:bg-muted text-muted-foreground hover:text-primary transition-colors ml-1"
                                                    title="在新窗口查看节点详情"
                                                >
                                                    <ExternalLink className="h-3 w-3" />
                                                </a>
                                            )}
                                        </div>
                                        {device.isVirtual && (
                                            <p className="text-xs text-destructive truncate mt-0.5">未找到匹配项</p>
                                        )}
                                    </div>
                                </div>
                            </TableCell>
                            <TableCell>
                                {device.isVirtual ? (
                                    <span className="text-muted-foreground">-</span>
                                ) : (
                                    <div className="flex items-center gap-2">
                                        <code className="text-[13px] font-medium bg-muted/50 px-2 py-0.5 rounded-md border border-border/10">
                                            {device.ip}
                                        </code>
                                        {device.isExactMatch && !device.isVirtual && device.ip?.toLowerCase() === device.originalKeyword?.toLowerCase() && (
                                            <Check className="w-3.5 h-3.5 text-green-600" strokeWidth={3} />
                                        )}
                                    </div>
                                )}
                            </TableCell>
                            <TableCell>
                                {device.isVirtual ? (
                                    <span className="text-muted-foreground">-</span>
                                ) : device.k8s_status ? (
                                    <Badge
                                        variant="outline"
                                        className={cn(
                                            "text-[10px] h-5",
                                            device.k8s_status === 'Ready'
                                                ? "border-green-500/20 text-green-600 bg-green-500/10"
                                                : device.k8s_status === 'Unschedulable'
                                                    ? "border-amber-500/20 text-amber-600 bg-amber-500/10"
                                                    : device.k8s_status === 'NotReady'
                                                        ? "border-red-500/20 text-red-500 bg-red-500/10"
                                                        : "border-muted text-muted-foreground bg-muted/30"
                                        )}
                                    >
                                        {device.k8s_status}
                                    </Badge>
                                ) : (
                                    <span className="text-muted-foreground text-xs">-</span>
                                )}
                            </TableCell>
                            <TableCell>
                                <span className="text-xs font-medium text-slate-600 dark:text-slate-400">{device.isVirtual ? '-' : (device.role || '-')}</span>
                            </TableCell>
                            <TableCell>
                                {device.isVirtual ? (
                                    <span className="text-muted-foreground">-</span>
                                ) : device.cluster ? (
                                    <div className="flex items-center gap-1.5">
                                        <Database className="h-3.5 w-3.5 text-emerald-500" />
                                        <span className="text-xs font-semibold">{device.cluster}</span>
                                    </div>
                                ) : (
                                    <span className="text-muted-foreground text-xs font-mono">-</span>
                                )}
                            </TableCell>
                            <TableCell onClick={(e) => {
                                if (!device.isVirtual) {
                                    e.stopPropagation()
                                    setEditingDevice(device)
                                    setIsEditDialogOpen(true)
                                }
                            }}>
                                {device.isVirtual ? (
                                    <span className="text-muted-foreground">-</span>
                                ) : (
                                    <div className="flex items-center gap-2 group/group">
                                        {device.group ? (
                                            <>
                                                <span className="text-sm font-medium line-clamp-1 group-hover/group:text-blue-600 transition-colors">
                                                    {device.group}
                                                </span>
                                                <Edit className="h-3 w-3 text-muted-foreground opacity-0 group-hover/group:opacity-100 transition-opacity shrink-0" />
                                            </>
                                        ) : (
                                            <Button
                                                variant="ghost"
                                                size="sm"
                                                className="h-7 text-[10px] text-muted-foreground/60 border border-dashed border-border/50 hover:border-blue-300 hover:text-blue-600 hover:bg-blue-50 py-0 px-2"
                                            >
                                                <Edit className="h-3 w-3 mr-1" />
                                                设置用途
                                            </Button>
                                        )}
                                    </div>
                                )}
                            </TableCell>
                            <TableCell>
                                {device.isVirtual ? (
                                    <span className="text-muted-foreground">-</span>
                                ) : (
                                    <div className="flex items-center gap-1.5 text-xs text-foreground/80 font-medium">
                                        <Globe className="h-3.5 w-3.5 text-muted-foreground/50" />
                                        <span>{device.idc || '-'}</span>
                                        {device.room && <span className="text-muted-foreground/40 font-normal">/ {device.room}</span>}
                                    </div>
                                )}
                            </TableCell>
                            <TableCell>
                                <span className="text-xs text-muted-foreground">{device.isVirtual ? '-' : (device.net_zone || '-')}</span>
                            </TableCell>
                            <TableCell>
                                <span className="text-xs font-mono text-muted-foreground">{device.isVirtual ? '-' : (device.appid || '-')}</span>
                            </TableCell>
                            <TableCell>
                                {!device.isVirtual && (
                                    <Badge
                                        variant="outline"
                                        className={cn(
                                            "text-[10px] h-5 transition-colors min-w-[32px] justify-center",
                                            device.is_localization
                                                ? "border-green-500/20 bg-green-500/10 text-green-600"
                                                : "border-red-500/20 bg-red-500/10 text-red-500"
                                        )}
                                    >
                                        {device.is_localization ? '是' : '否'}
                                    </Badge>
                                )}
                            </TableCell>
                            <TableCell>
                                {device.isVirtual ? (
                                    <Badge variant="outline" className="border-dashed text-muted-foreground text-[10px]">
                                        MISSING
                                    </Badge>
                                ) : (
                                    <Badge
                                        variant="outline"
                                        className={cn(
                                            "text-[10px] h-5",
                                            device.status === 'online' || device.status === '活跃' || device.status === 'Running' || device.status === 'READY'
                                                ? "border-green-500/20 text-green-600 bg-green-500/10"
                                                : "border-muted text-muted-foreground bg-muted/30"
                                        )}
                                    >
                                        {device.status || '未知'}
                                    </Badge>
                                )}
                            </TableCell>
                        </TableRow>
                    ))}
                </TableBody>
            </Table >

            <EditGroupDialog
                device={editingDevice}
                open={isEditDialogOpen}
                onOpenChange={setIsEditDialogOpen}
                onSuccess={onRefresh}
            />
        </div >
    )
}

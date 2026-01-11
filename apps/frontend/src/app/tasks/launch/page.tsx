'use client'

import { useState, useEffect, useMemo, useRef, useCallback } from 'react'
import { useRouter, useSearchParams } from 'next/navigation'
import { Card, CardContent, CardHeader } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Separator } from '@/components/ui/separator'
import { Switch } from '@/components/ui/switch'
import { Checkbox } from '@/components/ui/checkbox'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Textarea } from '@/components/ui/textarea'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import {
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableHeader,
    TableRow,
} from "@/components/ui/table"
import {
    Popover,
    PopoverContent,
    PopoverTrigger,
} from "@/components/ui/popover"
import {
    HoverCard,
    HoverCardContent,
    HoverCardTrigger,
} from "@/components/ui/hover-card"
import { toast } from "sonner"
import {
    ArrowLeft,
    Play,
    Server,
    Settings2,
    Loader2,
    Layers,
    ListFilter,
    Check,
    Cpu,
    ArrowRight,
    Search,
    ChevronDown,
    X,
    GripVertical,
    Plus,
    Trash2,
    Save,
    RotateCcw,
    Laptop,
    Globe,
    Database,
    Tag,
    AlertTriangle,
    Copy,
    Star,
    ChevronLeft,
    ChevronRight,
    ChevronsLeft,
    ChevronsRight
} from 'lucide-react'
import React from 'react'
import type { Cluster } from '@/types/api'
import type { PipelineTemplate } from '@/types/pipeline'
import { NavyDevice, DeviceFeatureDetails } from "@/types/navy"
import { RobustaAPI } from '@/lib/api'
import { cn } from '@/lib/utils'
import { formatDistanceToNow } from 'date-fns'
import { zhCN } from 'date-fns/locale'

// --- Helper Functions ---
function parseMultilineInput(input: string): string[] {
    if (!input.trim()) return []
    let normalized = input.replace(/\r\n/g, '\n').replace(/\r/g, '\n')
    normalized = normalized.replace(/，/g, ',').replace(/；/g, ';')
    const lines = normalized.split(/[\n,;]+/)
    return lines.map(line => line.trim()).filter(line => line.length > 0)
}

// --- Special Device Hover Card ---
function SpecialDeviceHoverCard({ device }: { device: NavyDevice }) {
    const [featureDetails, setFeatureDetails] = useState<DeviceFeatureDetails | null>(null)
    const [filterOptions, setFilterOptions] = useState<{ labelKeys: string[]; taintKeys: string[] } | null>(null)
    const [isLoading, setIsLoading] = useState(false)
    const [isOpen, setIsOpen] = useState(false)

    useEffect(() => {
        if (isOpen && !featureDetails && !isLoading && device.ci_code) {
            setIsLoading(true)
            Promise.all([
                RobustaAPI.getNavyDeviceFeatures(device.ci_code),
                RobustaAPI.getNavyFilterOptions()
            ])
                .then(([details, options]) => {
                    setFeatureDetails(details)
                    setFilterOptions(options)
                })
                .catch(err => console.error('加载设备特性失败:', err))
                .finally(() => setIsLoading(false))
        }
    }, [isOpen, device.ci_code, featureDetails, isLoading])

    const managedLabels = featureDetails?.labels?.filter(label => filterOptions?.labelKeys?.includes(label.key)) || []
    const managedTaints = featureDetails?.taints?.filter(taint => filterOptions?.taintKeys?.includes(taint.key)) || []
    const hasAnyFeatures = managedLabels.length > 0 || managedTaints.length > 0

    return (
        <HoverCard open={isOpen} onOpenChange={setIsOpen}>
            <HoverCardTrigger asChild>
                <div className="cursor-help inline-flex">
                    <Star className="h-4 w-4 text-amber-500 fill-amber-500 flex-shrink-0" />
                </div>
            </HoverCardTrigger>
            <HoverCardContent className="w-80 p-0 overflow-hidden" align="start">
                <div className="bg-amber-50 dark:bg-amber-950/30 p-3 border-b border-amber-100 dark:border-amber-900/50 flex items-start gap-3">
                    <div className="p-2 bg-amber-100/50 dark:bg-amber-900/50 rounded-lg shrink-0">
                        <Star className="h-5 w-5 text-amber-600 dark:text-amber-500 fill-amber-600 dark:fill-amber-500" />
                    </div>
                    <div>
                        <h4 className="text-sm font-semibold text-amber-900 dark:text-amber-100 mb-0.5">特殊设备</h4>
                        <p className="text-xs text-amber-700 dark:text-amber-300/80">此设备已被标记为特殊资产</p>
                    </div>
                </div>
                <div className="p-4 space-y-4">
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
                                                <Badge key={i} variant="secondary" className="px-1.5 h-5 text-[10px] bg-blue-50 text-blue-700 border-none">
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
                                                <Badge key={i} variant="secondary" className="px-1.5 h-5 text-[10px] bg-orange-50 text-orange-700 border-none">
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
            </HoverCardContent>
        </HoverCard>
    )
}

// --- Searchable Select Component ---
interface SearchableSelectProps<T> {
    items: T[]
    value: T | null
    onChange: (value: T | null) => void
    label: string
    placeholder?: string
    renderItem: (item: T) => React.ReactNode
    getItemKey: (item: T) => string
    getItemText: (item: T) => string // For search
    icon?: React.ReactNode
}

function SearchableSelect<T>({ items, value, onChange, label, placeholder, renderItem, getItemKey, getItemText, icon }: SearchableSelectProps<T>) {
    const [open, setOpen] = useState(false)
    const [search, setSearch] = useState('')
    const inputRef = useRef<HTMLInputElement>(null)

    const filteredItems = useMemo(() => {
        if (!search) return items
        return items.filter(item => getItemText(item).toLowerCase().includes(search.toLowerCase()))
    }, [items, search, getItemText])

    useEffect(() => {
        if (open && inputRef.current) {
            setTimeout(() => inputRef.current?.focus(), 100)
        }
        if (!open) {
            setSearch('')
        }
    }, [open])

    return (
        <div className="space-y-2.5">
            <Label className="text-sm font-medium text-foreground/90 flex items-center gap-2">
                {icon && <span className="text-primary/70">{icon}</span>}
                {label}
            </Label>
            <Popover open={open} onOpenChange={setOpen}>
                <PopoverTrigger asChild>
                    <Button
                        variant="outline"
                        role="combobox"
                        aria-expanded={open}
                        className={cn(
                            "w-full justify-between h-auto min-h-[52px] px-4 py-3 text-left font-normal",
                            "bg-gradient-to-r from-background to-muted/20",
                            "border-muted-foreground/20 hover:border-primary/50",
                            "shadow-sm hover:shadow-md",
                            "transition-all duration-200 ease-out",
                            "group",
                            open && "ring-2 ring-primary/20 border-primary/50 shadow-md",
                            value && "border-primary/30 bg-gradient-to-r from-primary/5 to-background"
                        )}
                    >
                        {value ? (
                            <div className="flex items-center gap-3 flex-1 mr-2">
                                <div className="flex items-center justify-center w-8 h-8 rounded-lg bg-primary/10 text-primary shrink-0">
                                    <Layers className="w-4 h-4" />
                                </div>
                                <div className="flex-1 min-w-0">
                                    {renderItem(value)}
                                </div>
                            </div>
                        ) : (
                            <div className="flex items-center gap-3 flex-1">
                                <div className="flex items-center justify-center w-8 h-8 rounded-lg bg-muted/50 text-muted-foreground/50 shrink-0 group-hover:bg-muted group-hover:text-muted-foreground transition-colors">
                                    <Search className="w-4 h-4" />
                                </div>
                                <span className="text-muted-foreground/70 group-hover:text-muted-foreground transition-colors">
                                    {placeholder || "Select..."}
                                </span>
                            </div>
                        )}
                        <ChevronDown className={cn(
                            "h-5 w-5 shrink-0 text-muted-foreground/50 transition-transform duration-200",
                            open && "rotate-180 text-primary"
                        )} />
                    </Button>
                </PopoverTrigger>
                <PopoverContent
                    className={cn(
                        "w-[var(--radix-popover-trigger-width)] p-0",
                        "border-muted-foreground/20 shadow-xl shadow-black/5",
                        "bg-gradient-to-b from-background to-muted/10",
                        "backdrop-blur-xl",
                        "animate-in fade-in-0 zoom-in-95 duration-200"
                    )}
                    align="start"
                    sideOffset={8}
                >
                    <div className="relative border-b border-muted-foreground/10 bg-muted/30">
                        <div className="absolute left-4 top-1/2 -translate-y-1/2">
                            <Search className="h-4 w-4 text-muted-foreground/50" />
                        </div>
                        <Input
                            ref={inputRef}
                            placeholder="输入关键词搜索..."
                            value={search}
                            onChange={(e) => setSearch(e.target.value)}
                            className={cn(
                                "h-12 w-full rounded-none rounded-t-md",
                                "pl-11 pr-4 text-sm",
                                "bg-transparent border-none shadow-none",
                                "focus-visible:ring-0 focus-visible:ring-offset-0",
                                "placeholder:text-muted-foreground/40"
                            )}
                        />
                        {search && (
                            <button
                                onClick={() => setSearch('')}
                                className="absolute right-3 top-1/2 -translate-y-1/2 p-1 rounded-full hover:bg-muted transition-colors"
                            >
                                <X className="h-3 w-3 text-muted-foreground/50" />
                            </button>
                        )}
                    </div>

                    <ScrollArea className="h-[280px]">
                        <div className="p-2">
                            {filteredItems.length === 0 ? (
                                <div className="flex flex-col items-center justify-center py-10 text-muted-foreground">
                                    <Search className="h-8 w-8 mb-3 opacity-20" />
                                    <p className="text-sm font-medium">未找到匹配结果</p>
                                    <p className="text-xs text-muted-foreground/60 mt-1">尝试其他关键词</p>
                                </div>
                            ) : (
                                <div className="space-y-1">
                                    {filteredItems.map((item, index) => {
                                        const isSelected = value && getItemKey(value) === getItemKey(item)
                                        return (
                                            <div
                                                key={getItemKey(item)}
                                                onClick={() => {
                                                    onChange(item)
                                                    setOpen(false)
                                                }}
                                                className={cn(
                                                    "relative flex cursor-pointer select-none items-center",
                                                    "rounded-lg px-3 py-3 text-sm outline-none",
                                                    "transition-all duration-150 ease-out",
                                                    "hover:bg-primary/5 hover:shadow-sm",
                                                    "active:scale-[0.99]",
                                                    isSelected
                                                        ? "bg-primary/10 shadow-sm border border-primary/20"
                                                        : "hover:border hover:border-transparent"
                                                )}
                                                style={{
                                                    animationDelay: `${index * 30}ms`
                                                }}
                                            >
                                                <div className={cn(
                                                    "absolute left-0 top-1/2 -translate-y-1/2 w-1 h-6 rounded-r-full transition-all duration-200",
                                                    isSelected ? "bg-primary" : "bg-transparent"
                                                )} />

                                                <div className="flex items-center gap-3 flex-1 pl-2">
                                                    <div className={cn(
                                                        "flex items-center justify-center w-7 h-7 rounded-md shrink-0 transition-colors",
                                                        isSelected
                                                            ? "bg-primary/20 text-primary"
                                                            : "bg-muted/50 text-muted-foreground/70"
                                                    )}>
                                                        <Layers className="w-3.5 h-3.5" />
                                                    </div>
                                                    <div className="flex-1 min-w-0">
                                                        {renderItem(item)}
                                                    </div>
                                                </div>

                                                <div className={cn(
                                                    "flex items-center justify-center w-6 h-6 rounded-full transition-all duration-200",
                                                    isSelected
                                                        ? "bg-primary text-primary-foreground scale-100"
                                                        : "bg-transparent scale-75 opacity-0"
                                                )}>
                                                    <Check className="h-3.5 w-3.5" />
                                                </div>
                                            </div>
                                        )
                                    })}
                                </div>
                            )}
                        </div>
                    </ScrollArea>

                    {items.length > 0 && (
                        <div className="border-t border-muted-foreground/10 px-3 py-2 bg-muted/20">
                            <p className="text-[10px] text-muted-foreground/60 text-center">
                                共 {items.length} 个模板 · 显示 {filteredItems.length} 个结果
                            </p>
                        </div>
                    )}
                </PopoverContent>
            </Popover>
        </div>
    )
}

// --- Verified Timeline Batch Editor ---
interface BatchEditorProps {
    availableNodes: string[]
    batches: string[][]
    onChange: (batches: string[][]) => void
    pauseIndices: Set<number>
    onPauseIndicesChange: (indices: Set<number>) => void
}

function BatchEditor({ availableNodes, batches, onChange, pauseIndices, onPauseIndicesChange }: BatchEditorProps) {
    const [numBatches, setNumBatches] = useState<number>(batches.length || 1)

    // Helper to redistribute nodes into N batches
    const redistribute = (n: number) => {
        if (availableNodes.length === 0) {
            onChange([[]])
            return
        }
        const newBatches: string[][] = Array.from({ length: n }, () => [])
        availableNodes.forEach((node, i) => {
            newBatches[i % n].push(node)
        })
        onChange(newBatches)
    }

    const handleNumBatchesChange = (n: number) => {
        if (n < 1) return
        setNumBatches(n)
        redistribute(n)
    }

    const togglePause = (index: number) => {
        const newSet = new Set(pauseIndices)
        if (newSet.has(index)) {
            newSet.delete(index)
        } else {
            newSet.add(index)
        }
        onPauseIndicesChange(newSet)
    }

    // Drag and Drop Logic
    const [draggingNode, setDraggingNode] = useState<{ node: string, sourceBatchIndex: number } | null>(null)

    const onDragStart = (e: React.DragEvent, node: string, batchIndex: number) => {
        setDraggingNode({ node, sourceBatchIndex: batchIndex })
        e.dataTransfer.effectAllowed = 'move'
    }

    const onDrop = (e: React.DragEvent, targetBatchIndex: number) => {
        e.preventDefault()
        if (!draggingNode) return
        if (draggingNode.sourceBatchIndex === targetBatchIndex) return

        const newBatches = [...batches]
        newBatches[draggingNode.sourceBatchIndex] = newBatches[draggingNode.sourceBatchIndex].filter(n => n !== draggingNode.node)
        newBatches[targetBatchIndex] = [...newBatches[targetBatchIndex], draggingNode.node]

        onChange(newBatches)
        setDraggingNode(null)
    }

    const onDragOver = (e: React.DragEvent) => {
        e.preventDefault()
    }

    const removeBatch = (index: number) => {
        if (batches.length <= 1) return
        const nodesToMove = batches[index]
        const newBatches = batches.filter((_, i) => i !== index)
        if (newBatches.length > 0) {
            newBatches[newBatches.length - 1] = [...newBatches[newBatches.length - 1], ...nodesToMove]
        }
        setNumBatches(newBatches.length)
        onChange(newBatches)

        // Adjust pause indices
        const newPauseIndices = new Set<number>()
        Array.from(pauseIndices).forEach(idx => {
            if (idx < index) newPauseIndices.add(idx)
            else if (idx > index) newPauseIndices.add(idx - 1)
        })
        onPauseIndicesChange(newPauseIndices)
    }

    // Sync if availableNodes changes
    useEffect(() => {
        const currentTotal = batches.flat().length
        if (currentTotal !== availableNodes.length) {
            redistribute(numBatches)
        }
    }, [availableNodes.length])

    useEffect(() => {
        setNumBatches(batches.length)
    }, [batches.length])

    return (
        <div className="space-y-6 w-full max-w-4xl">
            {/* Controls */}
            <div className="flex items-center justify-between border-b pb-4 mb-8">
                <div className="flex items-center gap-3">
                    <div className="flex items-center justify-center w-7 h-7 bg-primary/10 text-primary rounded-md font-bold text-xs">
                        {batches.length}
                    </div>
                    <span className="text-sm font-medium">分批执行计划</span>
                </div>
                <div className="flex items-center gap-2">
                    <span className="text-xs text-muted-foreground mr-2">批次数量:</span>
                    <div className="flex items-center border rounded-md shadow-sm bg-background">
                        <Button variant="ghost" size="sm" className="h-7 w-8 rounded-none border-r px-0" onClick={() => handleNumBatchesChange(numBatches - 1)} disabled={numBatches <= 1}>
                            -
                        </Button>
                        <div className="w-10 text-center text-sm font-mono">{numBatches}</div>
                        <Button variant="ghost" size="sm" className="h-7 w-8 rounded-none border-l px-0" onClick={() => handleNumBatchesChange(numBatches + 1)}>
                            +
                        </Button>
                    </div>
                </div>
            </div>

            {/* Timeline */}
            <div className="relative pl-6">
                {/* Continuous Vertical Line */}
                <div className="absolute left-[35px] top-4 bottom-4 w-px bg-border -z-10" />

                <div className="space-y-0">
                    {batches.map((batch, batchIndex) => {
                        const isLast = batchIndex === batches.length - 1
                        const isPaused = pauseIndices.has(batchIndex)

                        return (
                            <React.Fragment key={batchIndex}>
                                <div className="flex gap-6 relative py-2">
                                    {/* Timeline Node */}
                                    <div className="flex flex-col items-center shrink-0 w-[40px]">
                                        <div className={cn(
                                            "flex items-center justify-center w-8 h-8 rounded-full border-2 bg-background z-10 transition-colors",
                                            "border-primary text-primary shadow-sm"
                                        )}>
                                            <span className="text-xs font-bold">{batchIndex + 1}</span>
                                        </div>
                                    </div>

                                    {/* Card */}
                                    <Card
                                        className={cn(
                                            "flex-1 transition-all duration-200 group border-l-2",
                                            draggingNode ? "border-dashed bg-muted/30" : "bg-card hover:shadow-md",
                                            isPaused && !isLast ? "border-l-amber-500" : "border-l-primary/30"
                                        )}
                                        onDragOver={onDragOver}
                                        onDrop={(e) => onDrop(e, batchIndex)}
                                    >
                                        <div className="py-2.5 px-3 border-b flex items-center justify-between bg-muted/5">
                                            <div className="flex items-center gap-2">
                                                <span className="text-xs font-semibold text-muted-foreground">BATCH {batchIndex + 1}</span>
                                                <Badge variant="secondary" className="text-[10px] px-1 h-4 min-w-[20px] justify-center">{batch.length}</Badge>
                                            </div>
                                            {batches.length > 1 && (
                                                <Button variant="ghost" size="icon" className="h-5 w-5 text-muted-foreground hover:text-destructive opacity-0 group-hover:opacity-100 transition-opacity" onClick={() => removeBatch(batchIndex)}>
                                                    <X className="w-3 h-3" />
                                                </Button>
                                            )}
                                        </div>
                                        <div className="p-3">
                                            <div className="flex flex-wrap gap-2 min-h-[32px]">
                                                {batch.map((node) => (
                                                    <div
                                                        key={node}
                                                        draggable
                                                        onDragStart={(e) => onDragStart(e, node, batchIndex)}
                                                        className="flex items-center gap-1.5 px-2 py-1 rounded bg-muted/30 border border-muted text-[11px] font-mono cursor-grab active:cursor-grabbing hover:bg-background hover:border-primary/50 transition-colors"
                                                    >
                                                        {node}
                                                    </div>
                                                ))}
                                                {batch.length === 0 && (
                                                    <div className="w-full flex items-center justify-center text-[10px] text-muted-foreground/50 border border-dashed border-muted/50 rounded h-8">
                                                        拖拽节点至此
                                                    </div>
                                                )}
                                            </div>
                                        </div>
                                    </Card>
                                </div>

                                {/* Connector Node (Pause Toggle) */}
                                {!isLast && (
                                    <div className="flex gap-6 py-1 h-14">
                                        <div className="flex flex-col items-center shrink-0 w-[40px] relative justify-center">
                                            <div
                                                onClick={() => togglePause(batchIndex)}
                                                className={cn(
                                                    "w-6 h-6 rounded-full flex items-center justify-center cursor-pointer transition-all duration-300 z-10 border shadow-sm hover:scale-110",
                                                    isPaused
                                                        ? "bg-amber-100 border-amber-300 text-amber-600 dark:bg-amber-900 dark:border-amber-700"
                                                        : "bg-background border-border text-muted-foreground hover:text-foreground hover:border-primary/50"
                                                )}
                                                title={isPaused ? "点击取消暂停" : "点击设置暂停"}
                                            >
                                                {isPaused ? <Play className="w-2.5 h-2.5 fill-current ml-0.5" /> : <ChevronDown className="w-3 h-3" />}
                                            </div>
                                        </div>

                                        {/* Optional status text next to toggle */}
                                        <div className="flex items-center">
                                            {isPaused && (
                                                <Badge variant="outline" className="border-amber-200 bg-amber-50 text-amber-700 text-[10px] px-2 py-0 h-5">
                                                    等待确认
                                                </Badge>
                                            )}
                                        </div>
                                    </div>
                                )}
                            </React.Fragment>
                        )
                    })}

                    {/* End Node */}
                    <div className="flex gap-6 py-2 pt-1 opacity-50">
                        <div className="flex flex-col items-center shrink-0 w-[40px]">
                            <div className="flex items-center justify-center w-8 h-8 rounded-full border border-dashed border-muted bg-muted/10 z-10">
                                <Check className="w-3.5 h-3.5" />
                            </div>
                        </div>
                        <div className="flex items-center text-xs font-medium text-muted-foreground">任务完成</div>
                    </div>
                </div>
            </div>
        </div>
    )
}

// --- Main Page ---
export default function PipelineLaunchWizard() {
    const router = useRouter()
    const searchParams = useSearchParams()
    const templateIdFromUrl = searchParams.get('templateId')

    const [loading, setLoading] = useState(false)
    const [submitting, setSubmitting] = useState(false)

    // Data state
    const [clusters, setClusters] = useState<Cluster[]>([])
    const [templates, setTemplates] = useState<PipelineTemplate[]>([])

    // Form state
    const [selectedTemplate, setSelectedTemplate] = useState<PipelineTemplate | null>(null)
    const [selectedCluster, setSelectedCluster] = useState<Cluster | null>(null)
    const [parameters, setParameters] = useState<Record<string, any>>({})

    // Device selection state
    const [clusterDevices, setClusterDevices] = useState<NavyDevice[]>([]) // All devices in cluster
    const [loadingDevices, setLoadingDevices] = useState(false)
    const [keyword, setKeyword] = useState('')
    const [inputText, setInputText] = useState('')
    const [selectedDeviceIds, setSelectedDeviceIds] = useState<Set<number>>(new Set())

    // Pagination state
    const [currentPage, setCurrentPage] = useState(1)
    const [pageSize, setPageSize] = useState(10)

    // Batching state
    const [batches, setBatches] = useState<string[][]>([])
    const [batchPauseIndices, setBatchPauseIndices] = useState<Set<number>>(new Set())

    // Reset selection and filter when cluster changes
    useEffect(() => {
        if (selectedCluster) {
            loadDevices()
        } else {
            setClusterDevices([])
            setSelectedDeviceIds(new Set())
        }
        setKeyword('')
        setInputText('')
        setCurrentPage(1)
    }, [selectedCluster])

    // Load data only on mount
    useEffect(() => {
        const loadInitialData = async () => {
            setLoading(true)
            try {
                const [clustersData, templatesData] = await Promise.all([
                    RobustaAPI.getClusters(1, 100, 'Running'),
                    RobustaAPI.listPipelineTemplates(1, 100)
                ])
                setClusters(clustersData.data)
                setTemplates(templatesData.data)

                if (templateIdFromUrl && templatesData.data) {
                    const t = templatesData.data.find((t: PipelineTemplate) => String(t.id) === templateIdFromUrl)
                    if (t) setSelectedTemplate(t)
                }
            } catch (error) {
                console.error('Failed to load data:', error)
                toast.error("加载失败", { description: "无法获取集群或模板列表" })
            } finally {
                setLoading(false)
            }
        }
        loadInitialData()
    }, [])

    const loadDevices = async () => {
        if (!selectedCluster) return
        setLoadingDevices(true)
        try {
            // Query devices by cluster name
            const response = await RobustaAPI.queryNavyDevices({
                groups: [{
                    id: 'g1',
                    operator: 'and',
                    blocks: [{
                        id: 'b1',
                        type: 'device',
                        conditionType: 'equal',
                        key: 'cluster',
                        value: selectedCluster.name,
                        operator: 'and',
                        isActive: true
                    }]
                }],
                page: 1,
                size: 2000
            })
            setClusterDevices(response.data || [])
            setSelectedDeviceIds(new Set())
        } catch (error) {
            console.error('Failed to load devices:', error)
            setClusterDevices([])
            toast.error("获取节点失败", { description: "无法获取集群节点列表" })
        } finally {
            setLoadingDevices(false)
        }
    }

    const handleSave = async () => {
        if (!selectedTemplate || !selectedCluster) {
            toast.error("请完善配置", { description: "必须选择模板和目标集群" })
            return
        }

        const validBatches = batches.filter(b => b.length > 0)
        if (validBatches.length === 0) {
            toast.error("执行列表为空", { description: "请至少在一个批次中包含节点" })
            return
        }

        setSubmitting(true)
        try {
            // Pack pause indices into parameters
            const pauseIndicesArray = Array.from(batchPauseIndices).sort((a, b) => a - b)
            const updatedParameters = {
                ...parameters,
                _batch_pause_indices: pauseIndicesArray
            }

            await RobustaAPI.startExecution({
                template_id: parseInt(selectedTemplate.id),
                cluster_id: parseInt(selectedCluster.id as any) || 0,
                parameters: updatedParameters,
                batches: validBatches,
                pause_between_batches: false,
                target_nodes: validBatches.flat(),
                auto_start: false,
            })

            toast.success("保存成功", { description: "任务已保存到待执行列表" })
            router.push('/tasks?tab=tasks' as any)
        } catch (error: any) {
            console.error('Save failed:', error)
            toast.error("保存失败", { description: error.response?.data?.message || "未知错误" })
        } finally {
            setSubmitting(false)
        }
    }

    const renderParameterInput = (param: any) => {
        const value = parameters[param.name] !== undefined ? parameters[param.name] : param.default_value || ''
        switch (param.input_type) {
            case 'select':
                return (
                    <Select value={value} onValueChange={(v) => setParameters({ ...parameters, [param.name]: v })}>
                        <SelectTrigger><SelectValue placeholder="请选择" /></SelectTrigger>
                        <SelectContent>{param.options?.map((opt: string) => <SelectItem key={opt} value={opt}>{opt}</SelectItem>)}</SelectContent>
                    </Select>
                )
            case 'text':
            default:
                return (
                    <Input value={value} onChange={(e) => setParameters({ ...parameters, [param.name]: e.target.value })} placeholder={param.description} />
                )
        }
    }

    // Keyword filtering logic
    const processedDevices = useMemo(() => {
        if (!keyword.trim()) return clusterDevices

        const lines = parseMultilineInput(keyword)
        if (lines.length === 0) return clusterDevices

        const result: NavyDevice[] = []
        const seenIds = new Set<number>()

        lines.forEach(line => {
            const lowerLine = line.toLowerCase()
            const matches = clusterDevices.filter(d => {
                const ip = d.ip?.toLowerCase() || ''
                const ci = d.ci_code?.toLowerCase() || ''
                return ip.includes(lowerLine) || ci.includes(lowerLine)
            })

            if (matches.length > 0) {
                const sortedMatches = [...matches].sort((a, b) => {
                    const aExact = (a.ip?.toLowerCase() === lowerLine || a.ci_code?.toLowerCase() === lowerLine) ? 1 : 0
                    const bExact = (b.ip?.toLowerCase() === lowerLine || b.ci_code?.toLowerCase() === lowerLine) ? 1 : 0
                    return bExact - aExact
                })

                sortedMatches.forEach(match => {
                    if (!seenIds.has(match.id)) {
                        result.push({ ...match, originalKeyword: line })
                        seenIds.add(match.id)
                    }
                })
            } else {
                result.push({
                    id: -Math.random(),
                    ci_code: line,
                    ip: '',
                    status: 'missing',
                    isVirtual: true,
                    isMissing: true,
                    originalKeyword: line,
                    // Mock fields
                    arch_type: '', idc: '', room: '', cabinet: '', cabinet_no: '', infra_type: '',
                    is_localization: false, net_zone: '', group: '', appid: '', app_name: '',
                    os_create_time: '', cpu: 0, memory: 0, model: '', kvm_ip: '', os: '',
                    company: '', os_name: '', os_issue: '', os_kernel: '', role: '',
                    cluster: '', cluster_id: 0, acceptance_time: '', disk_count: 0,
                    disk_detail: '', network_speed: '', is_special: false, feature_count: 0,
                    created_at: '', updated_at: ''
                } as NavyDevice)
            }
        })
        return result
    }, [clusterDevices, keyword])

    // Pagination logic
    const totalPages = Math.ceil(processedDevices.length / pageSize)
    const paginatedDevices = useMemo(() => {
        const start = (currentPage - 1) * pageSize
        return processedDevices.slice(start, start + pageSize)
    }, [processedDevices, currentPage, pageSize])

    // Reset page on filter
    useEffect(() => {
        setCurrentPage(1)
    }, [processedDevices.length])

    // Compute selected IPs for batching
    const targetIps = useMemo(() => {
        const ips: string[] = []
        clusterDevices.forEach(d => {
            if (selectedDeviceIds.has(d.id) && d.ip) {
                ips.push(d.ip)
            }
        })
        return ips
    }, [clusterDevices, selectedDeviceIds])

    // Robust Selection Handlers
    const toggleSelect = (id: number) => {
        const safeId = Number(id)
        if (isNaN(safeId)) return

        const newSet = new Set(selectedDeviceIds)
        if (newSet.has(safeId)) {
            newSet.delete(safeId)
        } else {
            newSet.add(safeId)
        }
        setSelectedDeviceIds(newSet)
    }

    const toggleSelectAll = () => {
        const validDevices = processedDevices.filter(d => !d.isVirtual)
        const allSelected = validDevices.length > 0 && validDevices.every(d => selectedDeviceIds.has(Number(d.id)))

        const newSet = new Set(selectedDeviceIds)
        if (allSelected) {
            validDevices.forEach(d => newSet.delete(Number(d.id)))
        } else {
            validDevices.forEach(d => newSet.add(Number(d.id)))
        }
        setSelectedDeviceIds(newSet)
    }

    if (loading) return <div className="flex justify-center items-center h-screen"><Loader2 className="w-8 h-8 animate-spin text-primary" /></div>

    const isReady = selectedTemplate && selectedCluster

    const SectionHeader = ({ index, title, description, isActive, isCompleted }: { index: number, title: string, description: string, isActive?: boolean, isCompleted?: boolean }) => (
        <div className="flex items-start gap-4 mb-6">
            <div className={cn(
                "flex items-center justify-center w-8 h-8 rounded-full border-2 text-sm font-semibold transition-colors shrink-0",
                isCompleted ? "bg-primary border-primary text-primary-foreground" :
                    isActive ? "border-primary text-primary bg-background" : "border-muted text-muted-foreground bg-muted/30"
            )}>
                {isCompleted ? <Check className="w-4 h-4" /> : index}
            </div>
            <div className="space-y-1">
                <h2 className={cn("text-lg font-semibold leading-none", isActive || isCompleted ? "text-foreground" : "text-muted-foreground")}>
                    {title}
                </h2>
                <p className="text-sm text-muted-foreground">{description}</p>
            </div>
        </div>
    )

    // Check if row is selected
    const isSelected = (id: number) => selectedDeviceIds.has(Number(id))

    // Row style helper
    const getRowClassName = (device: NavyDevice) => {
        if (!device.isVirtual && isSelected(device.id)) return "bg-blue-50/50 dark:bg-blue-900/10 border-blue-200 dark:border-blue-800"
        if (device.isVirtual) return "opacity-60 bg-muted/30"
        if (device.is_special || (device.app_name && device.app_name.trim() !== '')) {
            return "bg-amber-100/40 dark:bg-amber-900/20 hover:bg-amber-100/60 dark:hover:bg-amber-800/30"
        }
        if (device.cluster && device.cluster.trim() !== '') {
            return "bg-emerald-100/30 dark:bg-emerald-950/15 hover:bg-emerald-100/60 dark:hover:bg-emerald-900/30"
        }
        return "hover:bg-muted/50"
    }

    // Pagination Component
    const PaginationControls = () => {
        if (totalPages <= 1) return null
        return (
            <div className="flex items-center justify-between p-2 mt-2 border-t">
                <div className="text-sm text-muted-foreground">
                    第 {currentPage} 页 / 共 {totalPages} 页
                </div>
                <div className="flex items-center gap-1">
                    <Button variant="outline" size="sm" className="h-8 w-8 p-0" onClick={() => setCurrentPage(1)} disabled={currentPage === 1}><ChevronsLeft className="h-4 w-4" /></Button>
                    <Button variant="outline" size="sm" className="h-8 w-8 p-0" onClick={() => setCurrentPage(c => Math.max(1, c - 1))} disabled={currentPage === 1}><ChevronLeft className="h-4 w-4" /></Button>
                    <Button variant="outline" size="sm" className="h-8 w-8 p-0" onClick={() => setCurrentPage(c => Math.min(totalPages, c + 1))} disabled={currentPage === totalPages}><ChevronRight className="h-4 w-4" /></Button>
                    <Button variant="outline" size="sm" className="h-8 w-8 p-0" onClick={() => setCurrentPage(totalPages)} disabled={currentPage === totalPages}><ChevronsRight className="h-4 w-4" /></Button>
                </div>
            </div>
        )
    }

    return (
        <div className="min-h-screen bg-background pb-20">
            {/* Header */}
            <header className="w-full border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60 sticky top-0 z-10">
                <div className="px-8 flex h-14 items-center gap-4">
                    <Button variant="ghost" size="icon" onClick={() => router.back()} className="-ml-2">
                        <ArrowLeft className="w-4 h-4" />
                    </Button>
                    <Separator orientation="vertical" className="h-6" />
                    <div className="flex items-center gap-2">
                        <h1 className="font-semibold">新建任务</h1>
                        <Badge variant="outline" className="font-normal text-xs text-muted-foreground">New Task</Badge>
                    </div>
                    <div className="ml-auto flex items-center gap-2">
                        <Button variant="ghost" size="sm" onClick={() => router.back()}>取消</Button>
                        <Button size="sm" onClick={handleSave} disabled={submitting || !isReady}>
                            {submitting ? <Loader2 className="w-4 h-4 mr-2 animate-spin" /> : <Save className="w-4 h-4 mr-2" />}
                            {submitting ? '保存中...' : '保存'}
                        </Button>
                    </div>
                </div>
            </header>

            <div className="px-8 py-10 space-y-12 w-full">
                <section>
                    <SectionHeader index={1} title="基础配置" description="选择要执行的任务模板和目标集群" isActive={true} />
                    <div className="pl-12 grid grid-cols-1 md:grid-cols-2 gap-8 w-full">
                        <div className="space-y-4">
                            <SearchableSelect
                                label="任务模板"
                                placeholder="搜索并选择模板..."
                                items={templates}
                                value={selectedTemplate}
                                onChange={setSelectedTemplate}
                                getItemKey={(item) => item.id}
                                getItemText={(item) => item.name}
                                renderItem={(item) => <div className="font-medium">{item.name}</div>}
                            />
                            {selectedTemplate && (
                                <div className="rounded-lg border p-4 bg-muted/5 text-sm space-y-2">
                                    <div className="flex gap-2 text-muted-foreground"><Layers className="w-4 h-4" /><span>包含阶段:</span></div>
                                    <div className="flex flex-wrap gap-1">{selectedTemplate.stages.map((s, i) => <Badge key={i} variant="secondary" className="text-[10px] bg-background/80 border">{s.name}</Badge>)}</div>
                                </div>
                            )}
                        </div>
                        <div className="space-y-4">
                            <SearchableSelect
                                label="目标集群"
                                placeholder="搜索并选择集群..."
                                items={clusters}
                                value={selectedCluster}
                                onChange={setSelectedCluster}
                                getItemKey={(item) => item.id}
                                getItemText={(item) => item.name}
                                renderItem={(item) => (
                                    <div className="flex items-center justify-between w-full">
                                        <span className="font-medium">{item.name}</span>
                                        <Badge variant={item.status === 'Running' ? 'default' : 'secondary'} className="text-[10px] h-4 px-1">{item.status}</Badge>
                                    </div>
                                )}
                            />
                            {selectedCluster && (
                                <div className="rounded-lg border p-4 bg-muted/5 text-sm space-y-2">
                                    <div className="flex gap-2 text-muted-foreground"><Server className="w-4 h-4" /><span>集群信息:</span></div>
                                    <div className="text-muted-foreground text-xs leading-relaxed">{selectedCluster.description || "暂无描述"}</div>
                                </div>
                            )}
                        </div>
                    </div>
                </section>

                <Separator />

                {/* Node Selection & Filtering */}
                <section className={cn("transition-opacity duration-300", !selectedCluster ? "opacity-50 pointer-events-none" : "opacity-100")}>
                    <SectionHeader index={2} title="节点选择" description="筛选并选择需要执行任务的节点" isActive={!!selectedCluster} />
                    <div className="pl-12 w-full">
                        {loadingDevices ? (
                            <div className="flex justify-center py-12 text-muted-foreground border rounded-lg bg-muted/5">
                                <Loader2 className="w-6 h-6 animate-spin mr-2" />
                                <span className="text-sm">正在加载设备列表...</span>
                            </div>
                        ) : (
                            <div className="space-y-4">
                                <div className="space-y-2">
                                    <Label className="text-sm font-medium">节点筛选 (支持IP/CI_CODE多行输入)</Label>
                                    <div className="flex gap-4 items-start">
                                        <Textarea
                                            placeholder="输入IP或设备ID，支持多行批量筛选..."
                                            className="min-h-[80px] font-mono text-sm resize-y"
                                            value={inputText}
                                            onChange={(e) => {
                                                setInputText(e.target.value)
                                                setKeyword(e.target.value)
                                            }}
                                            onKeyDown={(e) => {
                                                if (e.key === 'Enter' && (e.metaKey || e.ctrlKey)) {
                                                    setKeyword(inputText)
                                                }
                                            }}
                                        />
                                        <div className="flex flex-col gap-2 shrink-0">
                                            <Button size="sm" onClick={() => setKeyword(inputText)}>
                                                <Search className="h-4 w-4 mr-1" /> 搜索
                                            </Button>
                                            <Button variant="outline" size="sm" onClick={() => { setKeyword(''); setInputText('') }}>
                                                <RotateCcw className="h-4 w-4 mr-1" /> 重置
                                            </Button>
                                        </div>
                                    </div>
                                </div>

                                <div className="rounded-md border bg-card">
                                    <div className="p-2 border-b bg-muted/20 flex items-center justify-between">
                                        <div className="text-sm text-muted-foreground ml-2">
                                            <span>显示 {processedDevices.length} 个节点</span>
                                            {selectedDeviceIds.size > 0 && <span className="ml-2 font-medium text-primary">(已选 {selectedDeviceIds.size} 个)</span>}
                                        </div>
                                        <div className="flex items-center gap-2">
                                            <span className="text-xs text-muted-foreground">每页显示</span>
                                            <Select value={String(pageSize)} onValueChange={v => setPageSize(Number(v))}>
                                                <SelectTrigger className="w-[70px] h-8 text-xs"><SelectValue /></SelectTrigger>
                                                <SelectContent>
                                                    <SelectItem value="10">10</SelectItem>
                                                    <SelectItem value="20">20</SelectItem>
                                                    <SelectItem value="50">50</SelectItem>
                                                    <SelectItem value="100">100</SelectItem>
                                                </SelectContent>
                                            </Select>
                                        </div>
                                    </div>
                                    <div className="max-h-[500px] overflow-auto">
                                        <Table>
                                            <TableHeader className="sticky top-0 bg-background z-10 shadow-sm">
                                                <TableRow>
                                                    <TableHead className="w-[40px] px-0 text-center">
                                                        <div
                                                            className="flex items-center justify-center w-full h-full cursor-pointer"
                                                            onClick={toggleSelectAll}
                                                        >
                                                            <Checkbox
                                                                checked={processedDevices.length > 0 && processedDevices.filter(d => !d.isVirtual).every(d => selectedDeviceIds.has(Number(d.id)))}
                                                                className="pointer-events-none"
                                                            />
                                                        </div>
                                                    </TableHead>
                                                    <TableHead className="min-w-[180px]">设备ID</TableHead>
                                                    <TableHead className="min-w-[150px]">IP</TableHead>
                                                    <TableHead>IDC/Room</TableHead>
                                                    <TableHead>App</TableHead>
                                                    <TableHead>状态</TableHead>
                                                </TableRow>
                                            </TableHeader>
                                            <TableBody>
                                                {processedDevices.length === 0 ? (
                                                    <TableRow>
                                                        <TableCell colSpan={6} className="h-24 text-center text-muted-foreground">暂无数据，请确认集群或筛选条件</TableCell>
                                                    </TableRow>
                                                ) : (
                                                    paginatedDevices.map((device, idx) => (
                                                        <TableRow
                                                            key={Number(device.id) || idx}
                                                            className={getRowClassName(device)}
                                                            onClick={(e) => {
                                                                if (!device.isVirtual) toggleSelect(Number(device.id))
                                                            }}
                                                        >
                                                            <TableCell className="w-[40px] p-0 text-center">
                                                                <div
                                                                    className="flex items-center justify-center w-full h-full cursor-pointer py-4"
                                                                    onClick={(e) => {
                                                                        e.stopPropagation()
                                                                        if (!device.isVirtual) toggleSelect(Number(device.id))
                                                                    }}
                                                                >
                                                                    {!device.isVirtual && (
                                                                        <Checkbox
                                                                            checked={isSelected(device.id)}
                                                                            className="pointer-events-none"
                                                                        />
                                                                    )}
                                                                </div>
                                                            </TableCell>
                                                            <TableCell>
                                                                <div className="flex items-center gap-2">
                                                                    {device.isVirtual ? <AlertTriangle className="h-4 w-4 text-destructive" /> : device.is_special ? <SpecialDeviceHoverCard device={device} /> : <Laptop className="h-4 w-4 text-muted-foreground shrink-0" />}
                                                                    <div className="flex flex-col">
                                                                        <div className="flex items-center gap-1">
                                                                            <span className={cn("font-mono text-sm", device.isVirtual && "text-muted-foreground")}>
                                                                                {device.isVirtual ? device.originalKeyword : device.ci_code}
                                                                            </span>
                                                                            {!device.isVirtual && (device as any).originalKeyword && device.ci_code?.toLowerCase() === (device as any).originalKeyword.toLowerCase() && (
                                                                                <Check className="h-3 w-3 text-emerald-500" />
                                                                            )}
                                                                        </div>
                                                                        {device.isVirtual && <span className="text-[10px] text-destructive">未找到设备</span>}
                                                                    </div>
                                                                </div>
                                                            </TableCell>
                                                            <TableCell>
                                                                <div className="flex items-center gap-1">
                                                                    <span>{device.isVirtual ? '-' : device.ip}</span>
                                                                    {!device.isVirtual && (device as any).originalKeyword && device.ip?.toLowerCase() === (device as any).originalKeyword.toLowerCase() && (
                                                                        <Check className="h-3 w-3 text-emerald-500" />
                                                                    )}
                                                                </div>
                                                            </TableCell>
                                                            <TableCell>
                                                                {device.isVirtual ? '-' : <div className="flex items-center gap-1 text-xs text-muted-foreground"><Globe className="h-3 w-3" />{device.idc} {device.room && `/ ${device.room}`}</div>}
                                                            </TableCell>
                                                            <TableCell>
                                                                {device.isVirtual ? '-' : <div className="flex flex-col"><span className="text-xs font-medium">{device.app_name || '-'}</span><span className="text-[10px] text-muted-foreground">{device.group}</span></div>}
                                                            </TableCell>
                                                            <TableCell>
                                                                {device.isVirtual ? <Badge variant="outline" className="border-dashed text-muted-foreground text-[10px]">MISSING</Badge> : (
                                                                    <Badge variant="outline" className={cn("text-[10px] h-5", ["online", "活跃", "Running", "READY"].includes(device.status) ? "border-emerald-300 text-emerald-800 bg-emerald-100/80 dark:bg-emerald-900/40 dark:text-emerald-400" : "border-muted text-muted-foreground")}>{device.status || '未知'}</Badge>
                                                                )}
                                                            </TableCell>
                                                        </TableRow>
                                                    ))
                                                )}
                                            </TableBody>
                                        </Table>
                                    </div>
                                    <PaginationControls />
                                </div>
                            </div>
                        )}
                    </div>
                </section>

                <Separator />

                {/* Batch Editor */}
                <section className={cn("transition-opacity duration-300", targetIps.length === 0 ? "opacity-50 pointer-events-none" : "opacity-100")}>
                    <SectionHeader index={3} title="分批策略" description="配置节点分批执行策略，支持拖拽调整" isActive={targetIps.length > 0} />
                    <div className="pl-12 w-full">
                        {targetIps.length === 0 ? (
                            <div className="flex flex-col items-center justify-center py-8 text-muted-foreground bg-muted/20 rounded-md border border-dashed"><Cpu className="w-8 h-8 mb-2 opacity-20" /><p className="text-sm">请先在上方列表选择节点</p></div>
                        ) : (
                            <BatchEditor availableNodes={targetIps} batches={batches} onChange={setBatches} pauseIndices={batchPauseIndices} onPauseIndicesChange={setBatchPauseIndices} />
                        )}
                    </div>
                </section>

                <Separator />

                {/* Parameters */}
                <section className={cn("transition-opacity duration-300", !selectedTemplate ? "opacity-50 pointer-events-none" : "opacity-100")}>
                    <SectionHeader index={4} title="任务参数" description="配置任务运行所需的运行时变量" isActive={!!selectedTemplate} />
                    <div className="pl-12 w-full">
                        {(selectedTemplate?.stages.flatMap(s => s.config.parameters || []) || []).length > 0 ? (
                            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                                {selectedTemplate?.stages.map(stage => (
                                    stage.config.parameters?.map((param: any, idx: number) => (
                                        <Card key={`${stage.id}-${idx}`} className="shadow-none border bg-card/50 hover:bg-card transition-colors">
                                            <CardContent className="p-4 space-y-3">
                                                {/* Parameter Header */}
                                                <div className="flex items-center justify-between">
                                                    <div className="flex items-center gap-2">
                                                        <Label className="text-sm font-medium">
                                                            {param.label}
                                                            {param.required && <span className="text-destructive ml-0.5">*</span>}
                                                        </Label>
                                                        {param.description && (
                                                            <HoverCard openDelay={200} closeDelay={100}>
                                                                <HoverCardTrigger asChild>
                                                                    <div className="flex items-center justify-center w-4 h-4 rounded-full bg-muted hover:bg-primary/10 cursor-help transition-colors">
                                                                        <span className="text-[10px] font-medium text-muted-foreground">?</span>
                                                                    </div>
                                                                </HoverCardTrigger>
                                                                <HoverCardContent side="top" className="w-72 p-0 shadow-lg" align="start">
                                                                    <div className="bg-gradient-to-br from-primary/5 to-primary/10 px-4 py-3 border-b">
                                                                        <div className="flex items-center gap-2">
                                                                            <div className="w-6 h-6 rounded-md bg-primary/10 flex items-center justify-center">
                                                                                <Settings2 className="w-3.5 h-3.5 text-primary" />
                                                                            </div>
                                                                            <span className="text-sm font-semibold">{param.label}</span>
                                                                        </div>
                                                                    </div>
                                                                    <div className="p-4 space-y-3">
                                                                        <p className="text-sm text-muted-foreground leading-relaxed">
                                                                            {param.description}
                                                                        </p>
                                                                        <div className="flex flex-wrap gap-2 pt-1">
                                                                            <Badge variant="secondary" className="text-[10px] h-5 px-1.5 font-normal">
                                                                                <code className="text-primary">{param.name}</code>
                                                                            </Badge>
                                                                            <Badge variant="outline" className="text-[10px] h-5 px-1.5 font-normal">
                                                                                {param.input_type === 'text' ? '文本' : param.input_type === 'select' ? '单选' : param.input_type === 'multi_select' ? '多选' : '固定值'}
                                                                            </Badge>
                                                                            {param.default_value && (
                                                                                <Badge variant="outline" className="text-[10px] h-5 px-1.5 font-normal text-muted-foreground">
                                                                                    默认: {param.default_value}
                                                                                </Badge>
                                                                            )}
                                                                        </div>
                                                                    </div>
                                                                </HoverCardContent>
                                                            </HoverCard>
                                                        )}
                                                    </div>
                                                    <Badge variant="outline" className="text-[10px] h-5 px-1.5 font-normal text-muted-foreground">
                                                        {param.input_type === 'text' ? '文本' : param.input_type === 'select' ? '单选' : param.input_type === 'multi_select' ? '多选' : '固定值'}
                                                    </Badge>
                                                </div>

                                                {/* Input Field */}
                                                {renderParameterInput(param)}

                                                {/* Inline Description (subtle hint) */}
                                                {param.description && (
                                                    <p className="text-[11px] text-muted-foreground/70 leading-relaxed line-clamp-2">
                                                        💡 {param.description}
                                                    </p>
                                                )}
                                            </CardContent>
                                        </Card>
                                    ))
                                ))}
                            </div>
                        ) : (
                            <div className="p-6 rounded-lg border border-dashed bg-muted/20 text-sm text-muted-foreground text-center">当前模板无需额外配置参数</div>
                        )}
                    </div>
                </section>
            </div>
        </div>
    )
}

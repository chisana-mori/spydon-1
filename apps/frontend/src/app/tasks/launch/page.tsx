'use client'

import { useState, useEffect, useMemo, useRef } from 'react'
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
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import {
    Popover,
    PopoverContent,
    PopoverTrigger,
} from "@/components/ui/popover"
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
    Save
} from 'lucide-react'
import type { Cluster } from '@/types/api'
import type { PipelineTemplate } from '@/types/pipeline'
import { RobustaAPI } from '@/lib/api'
import { cn } from '@/lib/utils'

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
}

function SearchableSelect<T>({ items, value, onChange, label, placeholder, renderItem, getItemKey, getItemText }: SearchableSelectProps<T>) {
    const [open, setOpen] = useState(false)
    const [search, setSearch] = useState('')

    const filteredItems = useMemo(() => {
        if (!search) return items
        return items.filter(item => getItemText(item).toLowerCase().includes(search.toLowerCase()))
    }, [items, search, getItemText])

    return (
        <div className="space-y-2">
            <Label>{label}</Label>
            <Popover open={open} onOpenChange={setOpen}>
                <PopoverTrigger asChild>
                    <Button
                        variant="outline"
                        role="combobox"
                        aria-expanded={open}
                        className="w-full justify-between h-auto min-h-[44px] px-3 py-2 text-left font-normal"
                    >
                        {value ? (
                            <div className="flex-1 mr-2">{renderItem(value)}</div>
                        ) : (
                            <span className="text-muted-foreground">{placeholder || "Select..."}</span>
                        )}
                        <ChevronDown className="ml-2 h-4 w-4 shrink-0 opacity-50" />
                    </Button>
                </PopoverTrigger>
                <PopoverContent className="w-[400px] p-0" align="start">
                    <div className="flex items-center border-b px-3">
                        <Search className="mr-2 h-4 w-4 shrink-0 opacity-50" />
                        <Input
                            placeholder="Type to search..."
                            value={search}
                            onChange={(e) => setSearch(e.target.value)}
                            className="flex h-11 w-full rounded-md bg-transparent py-3 text-sm outline-none placeholder:text-muted-foreground border-none shadow-none focus-visible:ring-0 px-0"
                        />
                    </div>
                    <ScrollArea className="h-[300px]">
                        <div className="p-1">
                            {filteredItems.length === 0 ? (
                                <div className="py-6 text-center text-sm text-muted-foreground">
                                    No results found.
                                </div>
                            ) : (
                                filteredItems.map((item) => (
                                    <div
                                        key={getItemKey(item)}
                                        onClick={() => {
                                            onChange(item)
                                            setOpen(false)
                                        }}
                                        className={cn(
                                            "relative flex cursor-pointer select-none items-center rounded-sm px-2 py-1.5 text-sm outline-none hover:bg-accent hover:text-accent-foreground data-[disabled]:pointer-events-none data-[disabled]:opacity-50",
                                            value && getItemKey(value) === getItemKey(item) ? "bg-accent/50 text-accent-foreground" : ""
                                        )}
                                    >
                                        <div className="flex-1">
                                            {renderItem(item)}
                                        </div>
                                        {value && getItemKey(value) === getItemKey(item) && (
                                            <Check className="ml-2 h-4 w-4 opacity-50" />
                                        )}
                                    </div>
                                ))
                            )}
                        </div>
                    </ScrollArea>
                </PopoverContent>
            </Popover>
        </div>
    )
}

// --- Batch Editor Component ---
interface BatchEditorProps {
    availableNodes: string[]
    batches: string[][]
    onChange: (batches: string[][]) => void
    pauseBetweenBatches: boolean
    setPauseBetweenBatches: (v: boolean) => void
}

function BatchEditor({ availableNodes, batches, onChange, pauseBetweenBatches, setPauseBetweenBatches }: BatchEditorProps) {
    const [numBatches, setNumBatches] = useState<number>(1)
    const [selectedNodeSource, setSelectedNodeSource] = useState<string[]>([]) // For potential manual selection logic from scratch if needed

    // Initialize batches when availableNodes change or just initially
    // Since we control batches externally, we use this only for reset logic if needed
    // But here we'll assume the parent manages initial state.

    // Helper to redistribute nodes into N batches
    const redistribute = (n: number) => {
        if (availableNodes.length === 0) return
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
        // Remove from source
        newBatches[draggingNode.sourceBatchIndex] = newBatches[draggingNode.sourceBatchIndex].filter(n => n !== draggingNode.node)
        // Add to target
        newBatches[targetBatchIndex] = [...newBatches[targetBatchIndex], draggingNode.node]

        onChange(newBatches)
        setDraggingNode(null)
    }

    const onDragOver = (e: React.DragEvent) => {
        e.preventDefault()
    }

    const addBatch = () => {
        const newBatches = [...batches, []]
        setNumBatches(newBatches.length)
        onChange(newBatches)
    }

    const removeBatch = (index: number) => {
        if (batches.length <= 1) return
        // Move nodes from removed batch to the first batch (or previous)
        const nodesToMove = batches[index]
        const newBatches = batches.filter((_, i) => i !== index)
        if (newBatches.length > 0) {
            newBatches[newBatches.length - 1] = [...newBatches[newBatches.length - 1], ...nodesToMove]
        }
        setNumBatches(newBatches.length)
        onChange(newBatches)
    }

    return (
        <div className="space-y-4">
            <div className="flex items-center justify-between">
                <div className="flex items-center gap-4">
                    <div className="flex items-center gap-2">
                        <Label>执行批次数:</Label>
                        <Input
                            type="number"
                            min="1"
                            className="w-24"
                            value={numBatches}
                            onChange={(e) => handleNumBatchesChange(parseInt(e.target.value) || 1)}
                        />
                    </div>
                    <div className="flex items-center gap-2 border-l pl-4">
                        <Switch
                            id="pause-switch"
                            checked={pauseBetweenBatches}
                            onCheckedChange={setPauseBetweenBatches}
                        />
                        <Label htmlFor="pause-switch" className="text-sm font-normal text-muted-foreground cursor-pointer">批次间暂停 (等待确认)</Label>
                    </div>
                </div>
                <Button variant="outline" size="sm" onClick={addBatch}>
                    <Plus className="w-4 h-4 mr-1" /> 添加批次
                </Button>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
                {batches.map((batch, batchIndex) => (
                    <Card
                        key={batchIndex}
                        className={cn("bg-muted/10 border-dashed transition-colors", draggingNode ? "border-primary/50 bg-primary/5" : "")}
                        onDragOver={onDragOver}
                        onDrop={(e) => onDrop(e, batchIndex)}
                    >
                        <CardHeader className="py-3 px-4 flex flex-row items-center justify-between space-y-0">
                            <span className="font-semibold text-sm">Batch {batchIndex + 1}</span>
                            <div className="flex items-center gap-2">
                                <Badge variant="secondary" className="text-xs">{batch.length}</Badge>
                                {batches.length > 1 && (
                                    <Button variant="ghost" size="icon" className="h-6 w-6 text-muted-foreground hover:text-destructive" onClick={() => removeBatch(batchIndex)}>
                                        <X className="w-3 h-3" />
                                    </Button>
                                )}
                            </div>
                        </CardHeader>
                        <Separator />
                        <CardContent className="p-2 min-h-[100px]">
                            <div className="space-y-1">
                                {batch.map((node) => (
                                    <div
                                        key={node}
                                        draggable
                                        onDragStart={(e) => onDragStart(e, node, batchIndex)}
                                        className="group flex items-center gap-2 p-2 rounded-md bg-background border shadow-sm cursor-grab active:cursor-grabbing hover:border-primary/50 transition-colors"
                                    >
                                        <GripVertical className="w-3 h-3 text-muted-foreground opacity-50 group-hover:opacity-100" />
                                        <Cpu className="w-3 h-3 text-muted-foreground" />
                                        <span className="text-xs truncate font-medium flex-1">{node}</span>
                                    </div>
                                ))}
                                {batch.length === 0 && (
                                    <div className="flex items-center justify-center h-20 text-xs text-muted-foreground border border-dashed rounded-md bg-muted/20">
                                        拖拽节点到此处
                                    </div>
                                )}
                            </div>
                        </CardContent>
                    </Card>
                ))}
            </div>

            <p className="text-[10px] text-muted-foreground mt-2">
                * 拖拽节点可在不同批次间移动。执行时将按 Batch 1, Batch 2... 的顺序串行执行。
            </p>
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

    // Batching state
    // We maintain 'batches' (list of lists) directly.
    const [batches, setBatches] = useState<string[][]>([])
    const [pauseBetweenBatches, setPauseBetweenBatches] = useState(false)
    const [loadingNodes, setLoadingNodes] = useState(false)
    const [availableNodes, setAvailableNodes] = useState<string[]>([])

    // Load initial data
    useEffect(() => {
        const loadData = async () => {
            setLoading(true)
            try {
                const [clustersData, templatesData] = await Promise.all([
                    RobustaAPI.getClusters(1, 100, 'active'),
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
                toast.error("加载失败", {
                    description: "无法获取集群或模板列表",
                })
            } finally {
                setLoading(false)
            }
        }
        loadData()
    }, [])

    // Load nodes and initialize batches when cluster changes
    useEffect(() => {
        if (selectedCluster) {
            const loadNodes = async () => {
                setLoadingNodes(true)
                try {
                    const nodes = await RobustaAPI.getClusterNodes(selectedCluster.id)
                    // Auto-initialize to 1 batch with all nodes
                    setAvailableNodes(nodes || [])
                    setBatches(nodes && nodes.length > 0 ? [nodes] : [[]])
                } catch (error) {
                    console.error('Failed to load nodes:', error)
                    setAvailableNodes([])
                    setBatches([[]])
                    toast.error("获取节点失败", {
                        description: "无法获取集群节点列表",
                    })
                } finally {
                    setLoadingNodes(false)
                }
            }
            loadNodes()
        } else {
            setAvailableNodes([])
            setBatches([])
        }
    }, [selectedCluster])


    const handleSave = async () => {
        if (!selectedTemplate || !selectedCluster) {
            toast.error("请完善配置", { description: "必须选择模板和目标集群" })
            return
        }

        // Filter out empty batches
        const validBatches = batches.filter(b => b.length > 0)
        if (validBatches.length === 0) {
            toast.error("节点配置为空", { description: "请至少在一个批次中包含节点" })
            return
        }

        setSubmitting(true)
        try {
            await RobustaAPI.startExecution({
                template_id: parseInt(selectedTemplate.id),
                cluster_id: parseInt(selectedCluster.id as any) || 0,
                parameters: parameters,
                batches: validBatches,
                pause_between_batches: pauseBetweenBatches,
                target_nodes: validBatches.flat(),
                auto_start: false, // 只保存，不立即执行
            })

            toast.success("保存成功", {
                description: "任务已保存到待执行列表",
            })
            router.push('/tasks?tab=tasks' as any)
        } catch (error: any) {
            console.error('Save failed:', error)
            toast.error("保存失败", {
                description: error.response?.data?.message || error.message || "未知错误",
            })
        } finally {
            setSubmitting(false)
        }
    }

    // Render parameter input based on type
    const renderParameterInput = (param: any) => {
        const value = parameters[param.name] !== undefined ? parameters[param.name] : param.default_value || ''

        switch (param.input_type) {
            case 'select':
                return (
                    <Select
                        value={value}
                        onValueChange={(v) => setParameters({ ...parameters, [param.name]: v })}
                    >
                        <SelectTrigger>
                            <SelectValue placeholder="请选择" />
                        </SelectTrigger>
                        <SelectContent>
                            {param.options?.map((opt: string) => (
                                <SelectItem key={opt} value={opt}>{opt}</SelectItem>
                            ))}
                        </SelectContent>
                    </Select>
                )
            case 'text':
            default:
                return (
                    <Input
                        value={value}
                        onChange={(e) => setParameters({ ...parameters, [param.name]: e.target.value })}
                        placeholder={param.description}
                    />
                )
        }
    }

    if (loading) {
        return <div className="flex justify-center items-center h-screen"><Loader2 className="w-8 h-8 animate-spin text-primary" /></div>
    }

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
                <p className="text-sm text-muted-foreground">
                    {description}
                </p>
            </div>
        </div>
    )

    return (
        <div className="min-h-screen bg-background">
            {/* Header */}
            <header className="w-full border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
                <div className="container flex h-14 max-w-5xl items-center gap-4">
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
                        <Button
                            size="sm"
                            onClick={handleSave}
                            disabled={submitting || !isReady}
                        >
                            {submitting ? <Loader2 className="w-4 h-4 mr-2 animate-spin" /> : <Save className="w-4 h-4 mr-2" />}
                            {submitting ? '保存中...' : '保存'}
                        </Button>
                    </div>
                </div>
            </header>

            <div className="container max-w-5xl py-10 space-y-12">
                <section>
                    <SectionHeader index={1} title="基础配置" description="选择要执行的任务模板和目标集群" isActive={true} />
                    <div className="pl-12 grid grid-cols-1 md:grid-cols-2 gap-8">
                        <div className="space-y-4">
                            <SearchableSelect
                                label="任务模板"
                                placeholder="搜索并选择模板..."
                                items={templates}
                                value={selectedTemplate}
                                onChange={setSelectedTemplate}
                                getItemKey={(item) => item.id}
                                getItemText={(item) => item.name}
                                renderItem={(item) => (
                                    <div className="font-medium">{item.name}</div>
                                )}
                            />
                            {selectedTemplate && (
                                <div className="rounded-lg border p-4 bg-muted/5 text-sm space-y-2">
                                    <div className="flex gap-2 text-muted-foreground">
                                        <Layers className="w-4 h-4" />
                                        <span>包含阶段:</span>
                                    </div>
                                    <div className="flex flex-wrap gap-1">
                                        {selectedTemplate.stages.map((s, i) => (
                                            <Badge key={i} variant="secondary" className="text-[10px] bg-background/80 border">{s.name}</Badge>
                                        ))}
                                    </div>
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
                                        <Badge variant={item.status === 'active' ? 'default' : 'secondary'} className="text-[10px] h-4 px-1">{item.status}</Badge>
                                    </div>
                                )}
                            />
                            {selectedCluster && (
                                <div className="rounded-lg border p-4 bg-muted/5 text-sm space-y-2">
                                    <div className="flex gap-2 text-muted-foreground">
                                        <Server className="w-4 h-4" />
                                        <span>集群信息:</span>
                                    </div>
                                    <div className="text-muted-foreground text-xs leading-relaxed">
                                        {selectedCluster.description || "暂无描述"}
                                    </div>
                                </div>
                            )}
                        </div>
                    </div>
                </section>

                <Separator />

                {/* Batch / Node Selection */}
                <section className={cn("transition-opacity duration-300", !selectedCluster ? "opacity-50 pointer-events-none" : "opacity-100")}>
                    <SectionHeader index={2} title="执行策略 & 节点分批" description="配置节点分批执行策略，支持拖拽调整" isActive={!!selectedCluster} />
                    <div className="pl-12">
                        {loadingNodes ? (
                            <div className="flex justify-center py-8 text-muted-foreground border rounded-lg bg-muted/5">
                                <Loader2 className="w-5 h-5 animate-spin mr-2" />
                                <span className="text-sm">正在加载集群节点...</span>
                            </div>
                        ) : availableNodes.length > 0 ? (
                            <BatchEditor
                                availableNodes={availableNodes}
                                batches={batches}
                                onChange={setBatches}
                                pauseBetweenBatches={pauseBetweenBatches}
                                setPauseBetweenBatches={setPauseBetweenBatches}
                            />
                        ) : (
                            <div className="flex flex-col items-center justify-center py-8 text-muted-foreground bg-muted/20 rounded-md border border-dashed">
                                <Server className="w-8 h-8 mb-2 opacity-20" />
                                <p className="text-sm">该集群未发现可用节点</p>
                            </div>
                        )}
                    </div>
                </section>

                <Separator />

                {/* Parameters */}
                <section className={cn("transition-opacity duration-300", !selectedTemplate ? "opacity-50 pointer-events-none" : "opacity-100")}>
                    <SectionHeader index={3} title="任务参数" description="配置任务运行所需的运行时变量" isActive={!!selectedTemplate} />
                    <div className="pl-12">
                        {(selectedTemplate?.stages.flatMap(s => s.config.parameters || []) || []).length > 0 ? (
                            <div className="grid grid-cols-1 md:grid-cols-2 gap-6 p-6 rounded-lg border bg-card/50">
                                {selectedTemplate?.stages.map(stage => (
                                    stage.config.parameters?.map((param: any, idx: number) => (
                                        <div key={`${stage.id}-${idx}`} className="grid gap-2">
                                            <Label className="text-sm font-medium">
                                                {param.label}
                                                {param.required && <span className="text-destructive ml-1">*</span>}
                                            </Label>
                                            {renderParameterInput(param)}
                                            {param.description && <p className="text-[10px] text-muted-foreground">{param.description}</p>}
                                        </div>
                                    ))
                                ))}
                            </div>
                        ) : (
                            <div className="p-6 rounded-lg border border-dashed bg-muted/20 text-sm text-muted-foreground text-center">
                                当前模板无需额外配置参数
                            </div>
                        )}
                    </div>
                </section>
            </div>


        </div>
    )
}

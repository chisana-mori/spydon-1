'use client'

import { useState, useEffect, useCallback, useMemo } from 'react'
import { useRouter, useSearchParams } from 'next/navigation'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle, DialogTrigger } from '@/components/ui/dialog'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Label } from '@/components/ui/label'
import { Input } from '@/components/ui/input'
import { Checkbox } from '@/components/ui/checkbox'
import { Textarea } from '@/components/ui/textarea'
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from '@/components/ui/alert-dialog'
import {
    Play,
    Pause,
    StopCircle,
    RotateCcw,
    Plus,
    ChevronRight,
    Clock,
    CheckCircle,
    XCircle,
    AlertCircle,
    Loader2,
    RefreshCw,
    GitBranch,
    Edit,
    Trash2,
    MoreVertical,
    Server,
    Layers,
    Search,
    Sparkles,
} from 'lucide-react'
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from '@/components/ui/dropdown-menu'
import { toast } from 'sonner'
import { RobustaAPI } from '@/lib/api'
import type { PipelineTemplate, PipelineExecution, ExecutionStatus, StageRun, StageDefinition, ParameterBinding } from '@/types/pipeline'
import type { Cluster } from '@/types/api'
import {
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableHeader,
    TableRow,
} from "@/components/ui/table"
import { cn } from '@/lib/utils'
import { formatDistanceToNow } from 'date-fns'
import { zhCN } from 'date-fns/locale'

// 状态颜色映射
const statusColors: Record<ExecutionStatus, string> = {
    pending: 'bg-yellow-500',
    running: 'bg-blue-500',
    paused: 'bg-orange-500',
    successful: 'bg-green-500',
    failed: 'bg-red-500',
    canceled: 'bg-gray-500',
    rolling_back: 'bg-purple-500',
    rolled_back: 'bg-purple-400',
}

// 状态中文映射
const statusLabels: Record<ExecutionStatus, string> = {
    pending: '等待中',
    running: '执行中',
    paused: '已暂停',
    successful: '成功',
    failed: '失败',
    canceled: '已取消',
    rolling_back: '回滚中',
    rolled_back: '已回滚',
}

const StatusIcon = ({ status }: { status: ExecutionStatus }) => {
    switch (status) {
        case 'pending':
            return <Clock className="w-4 h-4" />
        case 'running':
            return <Loader2 className="w-4 h-4 animate-spin" />
        case 'paused':
            return <Pause className="w-4 h-4" />
        case 'successful':
            return <CheckCircle className="w-4 h-4" />
        case 'failed':
            return <XCircle className="w-4 h-4" />
        case 'canceled':
            return <StopCircle className="w-4 h-4" />
        case 'rolling_back':
            return <RotateCcw className="w-4 h-4 animate-spin" />
        case 'rolled_back':
            return <RotateCcw className="w-4 h-4" />
        default:
            return <AlertCircle className="w-4 h-4" />
    }
}

// 解析 stages JSON
const parseStages = (stagesData: unknown): StageDefinition[] => {
    if (!stagesData) return []
    if (Array.isArray(stagesData)) return stagesData as StageDefinition[]
    if (typeof stagesData === 'string') {
        try {
            return JSON.parse(stagesData) as StageDefinition[]
        } catch {
            return []
        }
    }
    return []
}

// --- Independent Execution Card Component ---
interface ExecutionCardProps {
    exec: PipelineExecution
    showActions?: boolean
    onRun: (id: string) => void
    onPause: (id: string) => void
    onResume: (id: string) => void
    onCancel: (id: string) => void
    onRollback: (id: string) => void
    onGenerateSOP: (id: string) => void
}

const ExecutionCard = ({ exec, showActions = true, onRun, onPause, onResume, onCancel, onRollback, onGenerateSOP }: ExecutionCardProps) => {
    const templateName = exec.template?.name || `模板 #${exec.pipeline_template_id}`
    const stages = parseStages(exec.template?.stages)
    const stageRuns = exec.stage_runs || []

    // Time display logic
    const timeDisplay = useMemo(() => {
        if (exec.status === 'pending') {
            return {
                icon: <Clock className="w-3 h-3" />,
                text: exec.created_at ? `创建于 ${formatDistanceToNow(new Date(exec.created_at), { addSuffix: true, locale: zhCN })}` : '刚刚创建'
            }
        }
        if (exec.started_at) {
            return {
                icon: <Play className="w-3 h-3" />,
                text: `开始于 ${formatDistanceToNow(new Date(exec.started_at), { addSuffix: true, locale: zhCN })}`
            }
        }
        return { icon: <Clock className="w-3 h-3" />, text: '未开始' }
    }, [exec.status, exec.created_at, exec.started_at])

    // Visual styles helpers
    const getStatusStyle = (s: ExecutionStatus) => {
        switch (s) {
            case 'running': return "bg-blue-50 text-blue-700 border-blue-200 dark:bg-blue-900/30 dark:text-blue-400 dark:border-blue-800"
            case 'successful': return "bg-emerald-50 text-emerald-700 border-emerald-200 dark:bg-emerald-900/30 dark:text-emerald-400 dark:border-emerald-800"
            case 'failed': return "bg-red-50 text-red-700 border-red-200 dark:bg-red-900/30 dark:text-red-400 dark:border-red-800"
            case 'paused': return "bg-amber-50 text-amber-700 border-amber-200 dark:bg-amber-900/30 dark:text-amber-400 dark:border-amber-800"
            case 'pending': return "bg-gray-100 text-gray-700 border-gray-200 dark:bg-gray-800 dark:text-gray-400 dark:border-gray-700"
            default: return "bg-gray-50 text-gray-600 border-gray-200"
        }
    }

    const getStripeColor = (s: ExecutionStatus) => {
        switch (s) {
            case 'running': return "bg-blue-500"
            case 'successful': return "bg-emerald-500"
            case 'failed': return "bg-red-500"
            case 'paused': return "bg-amber-500"
            case 'pending': return "bg-gray-400"
            default: return "bg-gray-300"
        }
    }

    // Render Stage Progress (Internal Helper)
    const renderStageProgress = () => {
        if (stages.length === 0) {
            return <div className="text-sm text-muted-foreground/50 italic">此模板未定义执行阶段</div>
        }

        return (
            <div className="relative">
                <div className="flex items-center gap-3 overflow-x-auto pb-2 scrollbar-hide mask-fade-right">
                    {stages.map((stage: StageDefinition, idx: number) => {
                        const stageRun = stageRuns.find((sr: StageRun) => sr.stage_id === stage.id)
                        const isCurrentStage = exec.current_stage_id === stage.id
                        const status = stageRun?.status || 'pending'

                        let icon = <div className="w-1.5 h-1.5 rounded-full bg-current" />
                        let wrapperClass = "bg-muted/50 text-muted-foreground border-transparent"

                        if (status === 'successful') {
                            icon = <CheckCircle className="w-3.5 h-3.5" />
                            wrapperClass = "bg-emerald-50 text-emerald-600 border-emerald-200 dark:bg-emerald-900/20 dark:text-emerald-400 dark:border-emerald-800"
                        } else if (status === 'running') {
                            icon = <Loader2 className="w-3.5 h-3.5 animate-spin" />
                            wrapperClass = "bg-blue-50 text-blue-600 border-blue-200 shadow-sm ring-1 ring-blue-100 dark:bg-blue-900/20 dark:text-blue-400 dark:border-blue-800 dark:ring-blue-900"
                        } else if (status === 'waiting_approval') {
                            icon = <div className="w-2 h-2 rounded-full bg-orange-500 animate-pulse" />
                            wrapperClass = "bg-orange-50 text-orange-600 border-orange-200 dark:bg-orange-900/20 dark:text-orange-400 dark:border-orange-800"
                        } else if (status === 'failed') {
                            icon = <XCircle className="w-3.5 h-3.5" />
                            wrapperClass = "bg-red-50 text-red-600 border-red-200 dark:bg-red-900/20 dark:text-red-400 dark:border-red-800"
                        } else if (isCurrentStage) {
                            icon = <Loader2 className="w-3.5 h-3.5 animate-spin" />
                            wrapperClass = "bg-blue-50 text-blue-600 border-blue-200 dark:bg-blue-900/20 dark:text-blue-400 dark:border-blue-800"
                        }

                        return (
                            <div key={stage.id} className="flex items-center flex-shrink-0 group">
                                <div className={cn(
                                    "px-3 py-1.5 rounded-full text-xs font-medium border flex items-center gap-2 transition-all duration-200",
                                    wrapperClass,
                                    isCurrentStage && "scale-105 shadow-md"
                                )}>
                                    {icon}
                                    <span>{stage.name || `Stage ${idx + 1}`}</span>
                                </div>
                                {idx < stages.length - 1 && (
                                    <div className={cn(
                                        "w-4 h-[1px] mx-1 transition-colors",
                                        stageRuns.some(r => r.stage_id === stages[idx + 1].id) ? "bg-muted-foreground/30" : "bg-muted/40"
                                    )} />
                                )}
                            </div>
                        )
                    })}
                </div>
            </div>
        )
    }

    return (
        <div
            className={cn(
                "relative group overflow-hidden bg-card rounded-xl border transition-all duration-300",
                "hover:shadow-lg hover:border-primary/20",
                "dark:hover:shadow-primary/5"
            )}
        >
            {/* Status Stripe */}
            <div className={cn("absolute left-0 top-0 bottom-0 w-1", getStripeColor(exec.status))} />

            <div className="p-5 pl-7">
                {/* Header Row */}
                <div className="flex flex-col md:flex-row md:items-start justify-between gap-4 mb-5">
                    <div className="space-y-1.5">
                        <div className="flex items-center gap-3">
                            <h3 className="text-lg font-bold flex items-center gap-2 tracking-tight">
                                <Server className="w-4 h-4 text-muted-foreground/70" />
                                {exec.cluster_name || `集群 #${exec.cluster_id}`}
                            </h3>
                            <Badge variant="outline" className={cn("text-xs px-2 py-0.5 h-6 font-medium border", getStatusStyle(exec.status))}>
                                <StatusIcon status={exec.status} />
                                <span className="ml-1.5">{statusLabels[exec.status]}</span>
                            </Badge>
                        </div>
                        <div className="flex items-center gap-3 text-xs text-muted-foreground/80">
                            <span className="flex items-center gap-1.5 bg-muted/50 px-2 py-0.5 rounded border border-muted">
                                <Layers className="w-3 h-3" />
                                {templateName}
                            </span>
                            <span className="w-1 h-1 rounded-full bg-muted-foreground/30" />
                            <span className="flex items-center gap-1">
                                {timeDisplay.icon}
                                <span>{timeDisplay.text}</span>
                            </span>
                        </div>
                    </div>

                    {/* Actions Group */}
                    {showActions && (
                        <div className="flex items-center gap-2 opacity-90 md:opacity-0 md:group-hover:opacity-100 transition-opacity duration-200">
                            <Button size="sm" variant="outline" onClick={() => onGenerateSOP(exec.id)} className="h-8 border-violet-200 hover:bg-violet-50 text-violet-700">
                                <Sparkles className="w-3.5 h-3.5 mr-1.5" /> AI-SOP
                            </Button>
                            {exec.status === 'pending' && (
                                <Button size="sm" onClick={() => onRun(exec.id)} className="h-8 shadow-sm">
                                    <Play className="w-3.5 h-3.5 mr-1.5" /> 执行
                                </Button>
                            )}
                            {exec.status === 'paused' && (
                                <Button size="sm" onClick={() => onResume(exec.id)} className="h-8 shadow-sm bg-amber-500 hover:bg-amber-600 text-white">
                                    <Play className="w-3.5 h-3.5 mr-1.5" /> 继续
                                </Button>
                            )}
                            {exec.status === 'running' && (
                                <Button size="sm" variant="outline" onClick={() => onPause(exec.id)} className="h-8 border-amber-200 hover:bg-amber-50 text-amber-700">
                                    <Pause className="w-3.5 h-3.5 mr-1.5" /> 暂停
                                </Button>
                            )}
                            {(exec.status === 'running' || exec.status === 'paused' || exec.status === 'pending') && (
                                <Button size="sm" variant="ghost" onClick={() => onCancel(exec.id)} className="h-8 text-muted-foreground hover:text-destructive">
                                    <StopCircle className="w-3.5 h-3.5 mr-1.5" /> 取消
                                </Button>
                            )}
                            {(exec.status === 'successful' || exec.status === 'failed') && (
                                <Button size="sm" variant="secondary" onClick={() => onRollback(exec.id)} className="h-8">
                                    <RotateCcw className="w-3.5 h-3.5 mr-1.5" /> 回滚
                                </Button>
                            )}
                        </div>
                    )}
                </div>

                {/* Progress Area */}
                <div className="bg-muted/30 rounded-lg p-4 border border-border/40">
                    {stages.length > 0 ? (
                        <div className="flex flex-col gap-3">
                            {renderStageProgress()}
                            {exec.error_message && (
                                <div className="flex items-start gap-2 text-xs text-red-600 bg-red-50 dark:bg-red-900/10 dark:text-red-400 p-2.5 rounded border border-red-100 dark:border-red-900/30">
                                    <AlertCircle className="w-4 h-4 shrink-0 mt-0.5" />
                                    <span className="leading-relaxed">{exec.error_message}</span>
                                </div>
                            )}
                        </div>
                    ) : (
                        <div className="flex items-center justify-center py-4 text-xs text-muted-foreground/60 border border-dashed rounded bg-background/50">
                            暂无阶段配置
                        </div>
                    )}
                </div>
            </div>
        </div>
    )
}

export default function TasksPage() {
    const router = useRouter()
    const searchParams = useSearchParams()
    const tabFromUrl = searchParams.get('tab')

    const [templates, setTemplates] = useState<PipelineTemplate[]>([])
    const [activeExecutions, setActiveExecutions] = useState<PipelineExecution[]>([])
    const [historyExecutions, setHistoryExecutions] = useState<PipelineExecution[]>([])
    const [clusters, setClusters] = useState<Cluster[]>([])
    const [loading, setLoading] = useState(true)
    const [refreshing, setRefreshing] = useState(false)
    const [activeTab, setActiveTab] = useState(tabFromUrl || 'tasks')

    // Template Pagination & Search State
    const [page, setPage] = useState(1)
    const [pageSize, setPageSize] = useState(10)
    const [keyword, setKeyword] = useState('')
    const [total, setTotal] = useState(0)
    const [searchInputValue, setSearchInputValue] = useState('') // For input field before search trigger

    // 审批对话框状态
    const [approvalDialogOpen, setApprovalDialogOpen] = useState(false)
    const [approvalExecutionId, setApprovalExecutionId] = useState<string>('')
    const [approvalNotes, setApprovalNotes] = useState('')
    const [approving, setApproving] = useState(false)

    // 删除确认对话框状态
    const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)
    const [deletingTemplate, setDeletingTemplate] = useState<PipelineTemplate | null>(null)
    const [deleting, setDeleting] = useState(false)

    // 加载数据
    const fetchData = useCallback(async (showRefreshing = false) => {
        if (showRefreshing) setRefreshing(true)
        try {
            const [templatesRes, activeRes, historyRes, clustersRes] = await Promise.all([
                RobustaAPI.listPipelineTemplates(page, pageSize, keyword),
                RobustaAPI.getActiveExecutions(50),
                RobustaAPI.getExecutionHistory({ limit: 100 }),
                RobustaAPI.getClusters(1, 100),
            ])
            setTemplates(templatesRes.data || [])
            setTotal(templatesRes.pagination?.total || 0)

            setActiveExecutions(activeRes || [])
            // 历史记录排除活跃的
            const activeIds = new Set((activeRes || []).map(e => e.id))
            setHistoryExecutions((historyRes || []).filter(e => !activeIds.has(e.id)))
            setClusters(clustersRes?.data || [])
        } catch (error) {
            console.error('Failed to fetch pipeline data:', error)
            toast.error('获取任务数据失败')
        } finally {
            setLoading(false)
            setRefreshing(false)
        }
    }, [page, pageSize, keyword])

    useEffect(() => {
        fetchData()
        // 每30秒自动刷新活跃执行
        const interval = setInterval(() => {
            fetchData()
        }, 30000)
        return () => clearInterval(interval)
    }, [fetchData])

    // 同步 URL tab 参数
    useEffect(() => {
        if (tabFromUrl && tabFromUrl !== activeTab) {
            setActiveTab(tabFromUrl)
        }
    }, [tabFromUrl, activeTab])

    const handleTabChange = (value: string) => {
        setActiveTab(value)
        const params = new URLSearchParams(searchParams)
        params.set('tab', value)
        router.push(`/tasks?${params.toString()}`)
    }

    // 暂停执行
    const handlePause = async (id: string) => {
        try {
            await RobustaAPI.pauseExecution(id)
            toast.success('任务已暂停')
            fetchData(true)
        } catch (error) {
            console.error('Failed to pause execution:', error)
            toast.error('暂停失败')
        }
    }

    // 恢复执行
    const handleResume = async () => {
        if (!approvalExecutionId) return
        setApproving(true)
        try {
            await RobustaAPI.resumeExecution(approvalExecutionId, approvalNotes)
            toast.success('任务已恢复')
            setApprovalDialogOpen(false)
            setApprovalExecutionId('')
            setApprovalNotes('')
            fetchData(true)
        } catch (error) {
            console.error('Failed to resume execution:', error)
            toast.error('恢复失败')
        } finally {
            setApproving(false)
        }
    }

    // 取消执行
    const handleCancel = async (id: string) => {
        try {
            await RobustaAPI.cancelExecution(id)
            toast.success('任务已取消')
            fetchData(true)
        } catch (error) {
            console.error('Failed to cancel execution:', error)
            toast.error('取消失败')
        }
    }

    // 生成 AI-SOP
    const handleGenerateSOP = async (id: string) => {
        toast.promise(
            async () => {
                const data = await RobustaAPI.generateSOPFlow(id)
                await navigator.clipboard.writeText(JSON.stringify(data, null, 2))
                return 'AI-SOP 流程已生成并复制到剪贴板'
            },
            {
                loading: '正在生成 SOP 流程...',
                success: (msg) => msg,
                error: '生成 AI-SOP 流程失败'
            }
        )
    }

    // 回滚执行
    const handleRollback = async (id: string) => {
        try {
            await RobustaAPI.rollbackExecution(id)
            toast.success('回滚已触发')
            fetchData(true)
        } catch (error) {
            console.error('Failed to rollback execution:', error)
            toast.error('回滚失败')
        }
    }

    // 执行待执行任务
    const handleRunPending = async (id: string) => {
        try {
            await RobustaAPI.runPendingExecution(id)
            toast.success('任务已开始执行')
            fetchData(true)
        } catch (error) {
            console.error('Failed to run pending execution:', error)
            toast.error('启动失败')
        }
    }

    // 打开审批对话框
    const openApprovalDialog = (executionId: string) => {
        setApprovalExecutionId(executionId)
        setApprovalNotes('')
        setApprovalDialogOpen(true)
    }

    // 新建模板 - 导航到编辑页
    const handleNewTemplate = () => {
        router.push('/tasks/templates/edit')
    }

    // 编辑模板 - 导航到编辑页
    const handleEditTemplate = (template: PipelineTemplate) => {
        router.push(`/tasks/templates/edit?id=${template.id}`)
    }

    // 删除模板
    const handleDeleteTemplate = async () => {
        if (!deletingTemplate) return
        setDeleting(true)
        try {
            await RobustaAPI.deletePipelineTemplate(deletingTemplate.id)
            toast.success('任务模板已删除')
            setDeleteDialogOpen(false)
            setDeletingTemplate(null)
            fetchData(true)
        } catch (error) {
            console.error('Failed to delete template:', error)
            toast.error('删除任务模板失败')
        } finally {
            setDeleting(false)
        }
    }

    // 确认删除
    const confirmDeleteTemplate = (template: PipelineTemplate) => {
        setDeletingTemplate(template)
        setDeleteDialogOpen(true)
    }

    if (loading) {
        return (
            <div className="flex items-center justify-center min-h-screen">
                <Loader2 className="w-8 h-8 animate-spin text-primary" />
            </div>
        )
    }

    return (
        <div className="space-y-6">
            {/* Header */}
            <div className="flex items-center justify-between">
                <div>
                    <h1 className="text-3xl font-bold tracking-tight">变更管理</h1>
                    <p className="text-muted-foreground">管理和执行自动化运维流程</p>
                </div>
                <div className="flex items-center gap-2">
                    <Button variant="outline" size="sm" onClick={() => fetchData(true)} disabled={refreshing}>
                        <RefreshCw className={`w-4 h-4 mr-2 ${refreshing ? 'animate-spin' : ''}`} />
                        刷新
                    </Button>
                    <Button disabled={templates.length === 0} onClick={() => router.push('/tasks/launch')}>
                        <Plus className="w-4 h-4 mr-2" />
                        新建任务
                    </Button>
                </div>
            </div>

            {/* Tabs */}
            <Tabs value={activeTab} onValueChange={handleTabChange}>
                <TabsList>
                    <TabsTrigger value="tasks">
                        任务
                        {activeExecutions.length > 0 && (
                            <Badge variant="secondary" className="ml-2">{activeExecutions.length}</Badge>
                        )}
                    </TabsTrigger>
                    <TabsTrigger value="templates">模板库</TabsTrigger>
                    <TabsTrigger value="history">历史记录</TabsTrigger>
                </TabsList>

                {/* Tasks Tab */}
                <TabsContent value="tasks" className="space-y-4">
                    {activeExecutions.length === 0 ? (
                        <Card>
                            <CardContent className="flex flex-col items-center justify-center py-12">
                                <Play className="w-12 h-12 text-muted-foreground mb-4" />
                                <p className="text-lg font-medium">暂无待执行的任务</p>
                                <p className="text-sm text-muted-foreground">点击上方"新建任务"创建新任务</p>
                            </CardContent>
                        </Card>
                    ) : (
                        <div className="grid gap-4">
                            {activeExecutions.map((exec) => (
                                <ExecutionCard
                                    key={exec.id}
                                    exec={exec}
                                    showActions={true}
                                    onRun={handleRunPending}
                                    onPause={handlePause}
                                    onResume={openApprovalDialog}
                                    onCancel={handleCancel}
                                    onRollback={handleRollback}
                                    onGenerateSOP={handleGenerateSOP}
                                />
                            ))}
                        </div>
                    )}
                </TabsContent>

                {/* Templates Tab */}
                <TabsContent value="templates" className="space-y-4">
                    <div className="flex justify-between items-center">
                        <div className="flex items-center gap-2 max-w-sm w-full">
                            <div className="relative flex-1">
                                <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
                                <Input
                                    type="search"
                                    placeholder="搜索任务模板..."
                                    className="pl-8"
                                    value={searchInputValue}
                                    onChange={(e) => setSearchInputValue(e.target.value)}
                                    onKeyDown={(e) => {
                                        if (e.key === 'Enter') {
                                            setKeyword(searchInputValue)
                                            setPage(1)
                                        }
                                    }}
                                />
                            </div>
                            <Button variant="secondary" onClick={() => {
                                setKeyword(searchInputValue)
                                setPage(1)
                            }}>
                                搜索
                            </Button>
                        </div>
                        <Button onClick={handleNewTemplate}>
                            <Plus className="w-4 h-4 mr-2" />
                            新建任务模板
                        </Button>
                    </div>

                    <div className="rounded-md border">
                        <Table>
                            <TableHeader>
                                <TableRow>
                                    <TableHead className="w-[200px]">名称</TableHead>
                                    <TableHead className="w-[300px]">描述</TableHead>
                                    <TableHead>阶段数</TableHead>
                                    <TableHead className="text-right">操作</TableHead>
                                </TableRow>
                            </TableHeader>
                            <TableBody>
                                {templates.length === 0 ? (
                                    <TableRow>
                                        <TableCell colSpan={4} className="h-24 text-center">
                                            暂无数据
                                        </TableCell>
                                    </TableRow>
                                ) : (
                                    templates.map((template) => {
                                        const stages = parseStages(template.stages)
                                        return (
                                            <TableRow key={template.id}>
                                                <TableCell className="font-medium">
                                                    {template.name}
                                                </TableCell>
                                                <TableCell className="text-muted-foreground truncate max-w-[300px]">
                                                    {template.description || '-'}
                                                </TableCell>
                                                <TableCell>
                                                    <div className="flex flex-wrap gap-1">
                                                        {stages.slice(0, 3).map((stage: StageDefinition) => (
                                                            <Badge key={stage.id} variant="secondary" className="text-xs">
                                                                {stage.name || stage.type}
                                                            </Badge>
                                                        ))}
                                                        {stages.length > 3 && (
                                                            <Badge variant="outline" className="text-xs">
                                                                +{stages.length - 3}
                                                            </Badge>
                                                        )}
                                                    </div>
                                                </TableCell>
                                                <TableCell className="text-right">
                                                    <div className="flex justify-end gap-2">
                                                        <Button
                                                            size="sm"
                                                            variant="ghost"
                                                            onClick={() => router.push(`/tasks/launch?templateId=${template.id}`)}
                                                            disabled={clusters.length === 0}
                                                        >
                                                            <Play className="w-4 h-4" />
                                                        </Button>
                                                        <Button
                                                            size="sm"
                                                            variant="ghost"
                                                            onClick={() => handleEditTemplate(template)}
                                                        >
                                                            <Edit className="w-4 h-4" />
                                                        </Button>
                                                        <DropdownMenu>
                                                            <DropdownMenuTrigger asChild>
                                                                <Button variant="ghost" size="icon" className="h-8 w-8">
                                                                    <MoreVertical className="w-4 h-4" />
                                                                </Button>
                                                            </DropdownMenuTrigger>
                                                            <DropdownMenuContent align="end">
                                                                <DropdownMenuItem
                                                                    onClick={() => confirmDeleteTemplate(template)}
                                                                    className="text-red-600"
                                                                >
                                                                    <Trash2 className="w-4 h-4 mr-2" />
                                                                    删除
                                                                </DropdownMenuItem>
                                                            </DropdownMenuContent>
                                                        </DropdownMenu>
                                                    </div>
                                                </TableCell>
                                            </TableRow>
                                        )
                                    })
                                )}
                            </TableBody>
                        </Table>
                    </div>

                    {/* Pagination */}
                    <div className="flex items-center justify-end space-x-2 py-4">
                        <div className="text-xs text-muted-foreground">
                            共 {total} 条
                        </div>
                        <div className="space-x-2">
                            <Button
                                variant="outline"
                                size="sm"
                                onClick={() => setPage(p => Math.max(1, p - 1))}
                                disabled={page <= 1}
                            >
                                上一页
                            </Button>
                            <div className="inline-flex items-center text-sm font-medium">
                                第 {page} 页
                            </div>
                            <Button
                                variant="outline"
                                size="sm"
                                onClick={() => setPage(p => p + 1)}
                                disabled={total <= page * pageSize}
                            >
                                下一页
                            </Button>
                        </div>
                    </div>
                </TabsContent>

                {/* History Tab */}
                <TabsContent value="history" className="space-y-4">
                    {historyExecutions.length === 0 ? (
                        <Card>
                            <CardContent className="flex flex-col items-center justify-center py-12">
                                <Clock className="w-12 h-12 text-muted-foreground mb-4" />
                                <p className="text-lg font-medium">暂无执行历史</p>
                                <p className="text-sm text-muted-foreground">完成的任务执行将显示在这里</p>
                            </CardContent>
                        </Card>
                    ) : (
                        <div className="grid gap-4">
                            {historyExecutions.map((exec) => (
                                <ExecutionCard
                                    key={exec.id}
                                    exec={exec}
                                    showActions={false}
                                    onRun={handleRunPending}
                                    onPause={handlePause}
                                    onResume={openApprovalDialog}
                                    onCancel={handleCancel}
                                    onRollback={handleRollback}
                                    onGenerateSOP={handleGenerateSOP}
                                />
                            ))}
                        </div>
                    )}
                </TabsContent>
            </Tabs>

            {/* 审批对话框 */}
            <Dialog open={approvalDialogOpen} onOpenChange={setApprovalDialogOpen}>
                <DialogContent>
                    <DialogHeader>
                        <DialogTitle>审批确认</DialogTitle>
                        <DialogDescription>确认继续执行此任务</DialogDescription>
                    </DialogHeader>
                    <div className="space-y-4 py-4">
                        <div className="space-y-2">
                            <Label>审批备注（可选）</Label>
                            <Textarea
                                value={approvalNotes}
                                onChange={(e) => setApprovalNotes(e.target.value)}
                                placeholder="输入审批备注..."
                                rows={3}
                            />
                        </div>
                    </div>
                    <DialogFooter>
                        <Button variant="outline" onClick={() => setApprovalDialogOpen(false)}>取消</Button>
                        <Button onClick={handleResume} disabled={approving}>
                            {approving ? <Loader2 className="w-4 h-4 mr-2 animate-spin" /> : <CheckCircle className="w-4 h-4 mr-2" />}
                            确认继续
                        </Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>

            {/* 删除确认对话框 */}
            <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
                <AlertDialogContent>
                    <AlertDialogHeader>
                        <AlertDialogTitle>确认删除</AlertDialogTitle>
                        <AlertDialogDescription>
                            确定要删除模板 &quot;{deletingTemplate?.name}&quot; 吗？此操作不可撤销。
                            如果有正在执行的任务使用此模板，删除将会失败。
                        </AlertDialogDescription>
                    </AlertDialogHeader>
                    <AlertDialogFooter>
                        <AlertDialogCancel disabled={deleting}>取消</AlertDialogCancel>
                        <AlertDialogAction
                            onClick={handleDeleteTemplate}
                            disabled={deleting}
                            className="bg-red-600 hover:bg-red-700"
                        >
                            {deleting ? <Loader2 className="w-4 h-4 mr-2 animate-spin" /> : <Trash2 className="w-4 h-4 mr-2" />}
                            删除
                        </AlertDialogAction>
                    </AlertDialogFooter>
                </AlertDialogContent>
            </AlertDialog>
        </div>
    )
}

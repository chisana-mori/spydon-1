'use client'

import { useState, useEffect, useCallback } from 'react'
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
    MoreVertical
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
import { Search } from 'lucide-react'

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

    // 渲染阶段进度
    const renderStageProgress = (execution: PipelineExecution) => {
        const stages = parseStages(execution.template?.stages)
        const stageRuns = execution.stage_runs || []

        if (stages.length === 0) {
            return <div className="text-sm text-muted-foreground">无阶段信息</div>
        }

        return (
            <div className="flex items-center gap-2 overflow-x-auto py-2">
                {stages.map((stage: StageDefinition, idx: number) => {
                    const stageRun = stageRuns.find((sr: StageRun) => sr.stage_id === stage.id)
                    const isCurrentStage = execution.current_stage_id === stage.id
                    const status = stageRun?.status || 'pending'

                    let bgClass = 'bg-gray-100 text-gray-600 dark:bg-gray-800 dark:text-gray-400'
                    if (status === 'successful') {
                        bgClass = 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-300'
                    } else if (status === 'running') {
                        bgClass = 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-300'
                    } else if (status === 'waiting_approval') {
                        bgClass = 'bg-orange-100 text-orange-800 dark:bg-orange-900 dark:text-orange-300'
                    } else if (status === 'failed') {
                        bgClass = 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-300'
                    }

                    return (
                        <div key={stage.id} className="flex items-center">
                            <div className={`
                                px-3 py-1 rounded-full text-sm whitespace-nowrap transition-all
                                ${bgClass}
                                ${isCurrentStage ? 'ring-2 ring-blue-500 dark:ring-blue-400' : ''}
                            `}>
                                {stage.name || `阶段 ${idx + 1}`}
                            </div>
                            {idx < stages.length - 1 && (
                                <ChevronRight className="w-4 h-4 text-gray-400 mx-1 flex-shrink-0" />
                            )}
                        </div>
                    )
                })}
            </div>
        )
    }

    // 渲染执行卡片
    const renderExecutionCard = (exec: PipelineExecution, showActions = true) => {
        const templateName = exec.template?.name || `模板 #${exec.pipeline_template_id}`
        const stages = parseStages(exec.template?.stages)

        return (
            <Card key={exec.id} className="hover:shadow-md transition-shadow">
                <CardHeader className="pb-2">
                    <div className="flex items-center justify-between">
                        <div className="flex items-center gap-3 flex-wrap">
                            <Badge className={`${statusColors[exec.status]} text-white`}>
                                <StatusIcon status={exec.status} />
                                <span className="ml-1">{statusLabels[exec.status]}</span>
                            </Badge>
                            <CardTitle className="text-lg">{exec.cluster_name || `集群 #${exec.cluster_id}`}</CardTitle>
                            <span className="text-sm text-muted-foreground">· {templateName}</span>
                        </div>
                        {showActions && (
                            <div className="flex gap-2 flex-wrap">
                                {exec.status === 'pending' && (
                                    <Button size="sm" onClick={() => handleRunPending(exec.id)}>
                                        <Play className="w-4 h-4 mr-1" />
                                        执行
                                    </Button>
                                )}
                                {exec.status === 'paused' && (
                                    <Button size="sm" variant="outline" onClick={() => openApprovalDialog(exec.id)}>
                                        <Play className="w-4 h-4 mr-1" />
                                        继续
                                    </Button>
                                )}
                                {exec.status === 'running' && (
                                    <Button size="sm" variant="outline" onClick={() => handlePause(exec.id)}>
                                        <Pause className="w-4 h-4 mr-1" />
                                        暂停
                                    </Button>
                                )}
                                {(exec.status === 'running' || exec.status === 'paused' || exec.status === 'pending') && (
                                    <Button size="sm" variant="outline" onClick={() => handleCancel(exec.id)}>
                                        <StopCircle className="w-4 h-4 mr-1" />
                                        取消
                                    </Button>
                                )}
                                {(exec.status === 'successful' || exec.status === 'failed') && (
                                    <Button size="sm" variant="outline" onClick={() => handleRollback(exec.id)}>
                                        <RotateCcw className="w-4 h-4 mr-1" />
                                        回滚
                                    </Button>
                                )}
                            </div>
                        )}
                    </div>
                    <CardDescription>
                        开始于 {exec.started_at ? new Date(exec.started_at).toLocaleString() : '-'}
                        {exec.completed_at && ` · 完成于 ${new Date(exec.completed_at).toLocaleString()}`}
                    </CardDescription>
                </CardHeader>
                <CardContent>
                    {stages.length > 0 ? (
                        renderStageProgress(exec)
                    ) : (
                        <div className="text-sm text-muted-foreground py-2">无阶段配置</div>
                    )}
                    {exec.error_message && (
                        <div className="mt-2 text-sm text-red-600 dark:text-red-400 bg-red-50 dark:bg-red-900/20 p-2 rounded">
                            错误：{exec.error_message}
                        </div>
                    )}
                </CardContent>
            </Card>
        )
    }

    if (loading) {
        return (
            <div className="flex items-center justify-center min-h-screen">
                <Loader2 className="w-8 h-8 animate-spin text-primary" />
            </div>
        )
    }

    return (
        <div className="container mx-auto p-6 space-y-6">
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
            <Tabs value={activeTab} onValueChange={setActiveTab}>
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
                            {activeExecutions.map((exec) => renderExecutionCard(exec, true))}
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
                            {historyExecutions.map((exec) => renderExecutionCard(exec, false))}
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

'use client'

import { useState, useEffect } from 'react'
import { useParams, useRouter } from 'next/navigation'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Separator } from '@/components/ui/separator'
import { ScrollArea } from '@/components/ui/scroll-area'
import {
    ArrowLeft,
    Play,
    Pause,
    StopCircle,
    RotateCcw,
    CheckCircle,
    XCircle,
    Clock,
    Loader2,
    ChevronDown,
    ChevronRight,
    Terminal
} from 'lucide-react'
import type { PipelineExecution, StageRun, ExecutionStatus, StageRunStatus } from '@/types/pipeline'

// 状态颜色
const statusColors: Record<ExecutionStatus | StageRunStatus, string> = {
    pending: 'bg-yellow-500',
    running: 'bg-blue-500',
    paused: 'bg-orange-500',
    successful: 'bg-green-500',
    failed: 'bg-red-500',
    canceled: 'bg-gray-500',
    rolling_back: 'bg-purple-500',
    rolled_back: 'bg-purple-400',
    waiting_approval: 'bg-orange-400',
    skipped: 'bg-gray-400',
}

// 模拟日志数据
const mockLogs = [
    { time: '14:32:05', level: 'INFO', message: 'PLAY [drain nodes] ***' },
    { time: '14:32:06', level: 'INFO', message: 'TASK [Gathering Facts] ***' },
    { time: '14:32:08', level: 'OK', message: 'ok: [worker-01]' },
    { time: '14:32:09', level: 'OK', message: 'ok: [worker-02]' },
    { time: '14:32:10', level: 'INFO', message: 'TASK [drain node] ***' },
    { time: '14:32:15', level: 'CHANGED', message: 'changed: [worker-01]' },
    { time: '14:32:20', level: 'CHANGED', message: 'changed: [worker-02]' },
]

export default function ExecutionDetailPage() {
    const params = useParams()
    const router = useRouter()
    const [execution, setExecution] = useState<PipelineExecution | null>(null)
    const [loading, setLoading] = useState(true)
    const [expandedStage, setExpandedStage] = useState<string | null>(null)

    useEffect(() => {
        // TODO: Fetch from API
        setLoading(false)
        // Mock data
        setExecution({
            id: params.id as string,
            pipeline_template_id: '1',
            cluster_id: '1',
            cluster_name: 'prod-cluster-01',
            status: 'running',
            current_stage_id: '2',
            started_at: new Date(Date.now() - 5 * 60 * 1000).toISOString(),
            created_at: new Date().toISOString(),
            updated_at: new Date().toISOString(),
            template: {
                id: '1',
                name: 'K8s 集群升级',
                description: '标准的 Kubernetes 版本升级流程',
                stages: [
                    { id: '1', name: '预检查', type: 'pre_check', config: {}, on_failure: 'abort' },
                    { id: '2', name: '节点驱逐', type: 'awx_job', config: { awx_template_name: 'drain-nodes' }, on_failure: 'rollback' },
                    { id: '3', name: '人工确认', type: 'manual_gate', config: {}, on_failure: 'pause' },
                    { id: '4', name: '版本升级', type: 'awx_job', config: { awx_template_name: 'upgrade-k8s' }, on_failure: 'rollback' },
                ],
                created_at: new Date().toISOString(),
                updated_at: new Date().toISOString(),
            },
            stage_runs: [
                { id: '1', execution_id: '1', stage_id: '1', stage_name: '预检查', stage_type: 'pre_check', status: 'successful', started_at: new Date(Date.now() - 5 * 60 * 1000).toISOString(), completed_at: new Date(Date.now() - 4 * 60 * 1000).toISOString(), created_at: '', updated_at: '' },
                { id: '2', execution_id: '1', stage_id: '2', stage_name: '节点驱逐', stage_type: 'awx_job', status: 'running', awx_job_id: 123, started_at: new Date(Date.now() - 4 * 60 * 1000).toISOString(), created_at: '', updated_at: '' },
                { id: '3', execution_id: '1', stage_id: '3', stage_name: '人工确认', stage_type: 'manual_gate', status: 'pending', created_at: '', updated_at: '' },
                { id: '4', execution_id: '1', stage_id: '4', stage_name: '版本升级', stage_type: 'awx_job', status: 'pending', created_at: '', updated_at: '' },
            ],
        })
        setExpandedStage('2') // 默认展开运行中的阶段
    }, [params.id])

    if (loading || !execution) {
        return (
            <div className="flex items-center justify-center min-h-screen">
                <Loader2 className="w-8 h-8 animate-spin text-primary" />
            </div>
        )
    }

    const getStageIcon = (status: StageRunStatus) => {
        switch (status) {
            case 'successful':
                return <CheckCircle className="w-5 h-5 text-green-500" />
            case 'failed':
                return <XCircle className="w-5 h-5 text-red-500" />
            case 'running':
                return <Loader2 className="w-5 h-5 text-blue-500 animate-spin" />
            case 'waiting_approval':
                return <Pause className="w-5 h-5 text-orange-500" />
            default:
                return <Clock className="w-5 h-5 text-gray-400" />
        }
    }

    return (
        <div className="container mx-auto p-6 space-y-6">
            {/* Header */}
            <div className="flex items-center gap-4">
                <Button variant="ghost" size="icon" onClick={() => router.back()}>
                    <ArrowLeft className="w-5 h-5" />
                </Button>
                <div className="flex-1">
                    <div className="flex items-center gap-3">
                        <h1 className="text-2xl font-bold">{execution.template?.name}</h1>
                        <Badge className={`${statusColors[execution.status]} text-white`}>
                            {execution.status}
                        </Badge>
                    </div>
                    <p className="text-muted-foreground">
                        集群: {execution.cluster_name} • 开始于 {execution.started_at ? new Date(execution.started_at).toLocaleString() : '-'}
                    </p>
                </div>
                <div className="flex gap-2">
                    {execution.status === 'running' && (
                        <Button variant="outline">
                            <Pause className="w-4 h-4 mr-2" />
                            暂停
                        </Button>
                    )}
                    {execution.status === 'paused' && (
                        <Button>
                            <Play className="w-4 h-4 mr-2" />
                            继续
                        </Button>
                    )}
                    <Button variant="destructive">
                        <StopCircle className="w-4 h-4 mr-2" />
                        取消
                    </Button>
                </div>
            </div>

            {/* Main Content */}
            <div className="grid grid-cols-12 gap-6">
                {/* Timeline */}
                <div className="col-span-4">
                    <Card>
                        <CardHeader>
                            <CardTitle className="text-lg">执行阶段</CardTitle>
                            <CardDescription>
                                {execution.stage_runs?.filter(s => s.status === 'successful').length || 0} / {execution.stage_runs?.length || 0} 已完成
                            </CardDescription>
                        </CardHeader>
                        <CardContent className="space-y-1">
                            {execution.stage_runs?.map((stage, idx) => (
                                <div key={stage.id}>
                                    <button
                                        onClick={() => setExpandedStage(expandedStage === stage.stage_id ? null : stage.stage_id)}
                                        className={`
                      w-full flex items-center gap-3 p-3 rounded-lg transition-colors text-left
                      ${expandedStage === stage.stage_id ? 'bg-accent' : 'hover:bg-muted'}
                    `}
                                    >
                                        {getStageIcon(stage.status)}
                                        <div className="flex-1 min-w-0">
                                            <p className="font-medium truncate">{stage.stage_name}</p>
                                            <p className="text-xs text-muted-foreground">
                                                {stage.stage_type}
                                                {stage.awx_job_id && ` • AWX Job #${stage.awx_job_id}`}
                                            </p>
                                        </div>
                                        {expandedStage === stage.stage_id ? (
                                            <ChevronDown className="w-4 h-4" />
                                        ) : (
                                            <ChevronRight className="w-4 h-4" />
                                        )}
                                    </button>
                                    {idx < (execution.stage_runs?.length || 0) - 1 && (
                                        <div className="ml-6 h-4 border-l-2 border-dashed border-gray-300 dark:border-gray-700" />
                                    )}
                                </div>
                            ))}
                        </CardContent>
                    </Card>

                    {/* Actions for Manual Gate */}
                    {execution.stage_runs?.some(s => s.status === 'waiting_approval') && (
                        <Card className="mt-4 border-orange-500">
                            <CardHeader>
                                <CardTitle className="text-lg text-orange-600">等待审批</CardTitle>
                                <CardDescription>请确认是否继续执行下一阶段</CardDescription>
                            </CardHeader>
                            <CardContent className="space-y-4">
                                <textarea
                                    className="w-full p-3 border rounded-lg resize-none"
                                    rows={3}
                                    placeholder="审批备注（可选）"
                                />
                                <div className="flex gap-2">
                                    <Button className="flex-1">
                                        <CheckCircle className="w-4 h-4 mr-2" />
                                        批准继续
                                    </Button>
                                    <Button variant="destructive" className="flex-1">
                                        <XCircle className="w-4 h-4 mr-2" />
                                        拒绝
                                    </Button>
                                </div>
                            </CardContent>
                        </Card>
                    )}
                </div>

                {/* Log Viewer */}
                <div className="col-span-8">
                    <Card className="h-[600px] flex flex-col">
                        <CardHeader className="flex-shrink-0">
                            <div className="flex items-center justify-between">
                                <div className="flex items-center gap-2">
                                    <Terminal className="w-5 h-5" />
                                    <CardTitle className="text-lg">执行日志</CardTitle>
                                </div>
                                <Badge variant="outline">实时</Badge>
                            </div>
                        </CardHeader>
                        <Separator />
                        <ScrollArea className="flex-1">
                            <div className="p-4 font-mono text-sm space-y-1 bg-gray-950 text-gray-100 min-h-full">
                                {mockLogs.map((log, idx) => (
                                    <div key={idx} className="flex gap-4">
                                        <span className="text-gray-500 flex-shrink-0">{log.time}</span>
                                        <span className={`flex-shrink-0 w-16 ${log.level === 'OK' ? 'text-green-400' :
                                                log.level === 'CHANGED' ? 'text-yellow-400' :
                                                    log.level === 'ERROR' ? 'text-red-400' :
                                                        'text-blue-400'
                                            }`}>
                                            [{log.level}]
                                        </span>
                                        <span className="text-gray-200">{log.message}</span>
                                    </div>
                                ))}
                                <div className="flex items-center gap-2 text-gray-500 mt-4">
                                    <Loader2 className="w-4 h-4 animate-spin" />
                                    等待更多输出...
                                </div>
                            </div>
                        </ScrollArea>
                    </Card>
                </div>
            </div>

            {/* Rollback Section */}
            {(execution.status === 'successful' || execution.status === 'failed') && (
                <Card className="border-purple-500">
                    <CardHeader>
                        <CardTitle className="text-lg flex items-center gap-2">
                            <RotateCcw className="w-5 h-5" />
                            回滚选项
                        </CardTitle>
                        <CardDescription>
                            如果需要撤销本次变更，可以触发回滚操作
                        </CardDescription>
                    </CardHeader>
                    <CardContent>
                        <Button variant="outline" className="border-purple-500 text-purple-600 hover:bg-purple-50">
                            <RotateCcw className="w-4 h-4 mr-2" />
                            一键回滚
                        </Button>
                    </CardContent>
                </Card>
            )}
        </div>
    )
}

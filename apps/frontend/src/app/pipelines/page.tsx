'use client'

import { useState, useEffect } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
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
    Loader2
} from 'lucide-react'
import type { PipelineTemplate, PipelineExecution, ExecutionStatus } from '@/types/pipeline'

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
        default:
            return <AlertCircle className="w-4 h-4" />
    }
}

export default function PipelinesPage() {
    const [templates, setTemplates] = useState<PipelineTemplate[]>([])
    const [executions, setExecutions] = useState<PipelineExecution[]>([])
    const [loading, setLoading] = useState(true)
    const [activeTab, setActiveTab] = useState('executions')

    useEffect(() => {
        // TODO: Fetch from API
        setLoading(false)
        // Mock data for UI development
        setTemplates([
            {
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
        ])
        setExecutions([
            {
                id: '1',
                pipeline_template_id: '1',
                cluster_id: '1',
                cluster_name: 'prod-cluster-01',
                status: 'running',
                current_stage_id: '2',
                started_at: new Date(Date.now() - 5 * 60 * 1000).toISOString(),
                created_at: new Date().toISOString(),
                updated_at: new Date().toISOString(),
            },
            {
                id: '2',
                pipeline_template_id: '1',
                cluster_id: '2',
                cluster_name: 'staging-cluster-01',
                status: 'paused',
                current_stage_id: '3',
                started_at: new Date(Date.now() - 15 * 60 * 1000).toISOString(),
                created_at: new Date().toISOString(),
                updated_at: new Date().toISOString(),
            },
        ])
    }, [])

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
                    <h1 className="text-3xl font-bold tracking-tight">流水线</h1>
                    <p className="text-muted-foreground">管理和执行自动化运维流程</p>
                </div>
                <Button>
                    <Plus className="w-4 h-4 mr-2" />
                    新建流水线
                </Button>
            </div>

            {/* Tabs */}
            <Tabs value={activeTab} onValueChange={setActiveTab}>
                <TabsList>
                    <TabsTrigger value="executions">执行中</TabsTrigger>
                    <TabsTrigger value="templates">模板库</TabsTrigger>
                    <TabsTrigger value="history">历史记录</TabsTrigger>
                </TabsList>

                {/* Executions Tab */}
                <TabsContent value="executions" className="space-y-4">
                    {executions.length === 0 ? (
                        <Card>
                            <CardContent className="flex flex-col items-center justify-center py-12">
                                <Play className="w-12 h-12 text-muted-foreground mb-4" />
                                <p className="text-lg font-medium">暂无执行中的流水线</p>
                                <p className="text-sm text-muted-foreground">选择一个模板开始新的执行</p>
                            </CardContent>
                        </Card>
                    ) : (
                        <div className="grid gap-4">
                            {executions.map((exec) => (
                                <Card key={exec.id} className="hover:shadow-md transition-shadow cursor-pointer">
                                    <CardHeader className="pb-2">
                                        <div className="flex items-center justify-between">
                                            <div className="flex items-center gap-3">
                                                <Badge className={`${statusColors[exec.status]} text-white`}>
                                                    <StatusIcon status={exec.status} />
                                                    <span className="ml-1">{exec.status}</span>
                                                </Badge>
                                                <CardTitle className="text-lg">{exec.cluster_name}</CardTitle>
                                            </div>
                                            <div className="flex gap-2">
                                                {exec.status === 'paused' && (
                                                    <Button size="sm" variant="outline">
                                                        <Play className="w-4 h-4 mr-1" />
                                                        继续
                                                    </Button>
                                                )}
                                                {exec.status === 'running' && (
                                                    <Button size="sm" variant="outline">
                                                        <Pause className="w-4 h-4 mr-1" />
                                                        暂停
                                                    </Button>
                                                )}
                                                {(exec.status === 'successful' || exec.status === 'failed') && (
                                                    <Button size="sm" variant="outline">
                                                        <RotateCcw className="w-4 h-4 mr-1" />
                                                        回滚
                                                    </Button>
                                                )}
                                            </div>
                                        </div>
                                        <CardDescription>
                                            开始于 {exec.started_at ? new Date(exec.started_at).toLocaleString() : '-'}
                                        </CardDescription>
                                    </CardHeader>
                                    <CardContent>
                                        {/* Stage Progress */}
                                        <div className="flex items-center gap-2 overflow-x-auto py-2">
                                            {['预检查', '节点驱逐', '人工确认', '版本升级'].map((stage, idx) => (
                                                <div key={idx} className="flex items-center">
                                                    <div className={`
                            px-3 py-1 rounded-full text-sm whitespace-nowrap
                            ${idx < 2 ? 'bg-green-100 text-green-800' :
                                                            idx === 2 ? 'bg-blue-100 text-blue-800 ring-2 ring-blue-500' :
                                                                'bg-gray-100 text-gray-600'}
                          `}>
                                                        {stage}
                                                    </div>
                                                    {idx < 3 && <ChevronRight className="w-4 h-4 text-gray-400 mx-1" />}
                                                </div>
                                            ))}
                                        </div>
                                    </CardContent>
                                </Card>
                            ))}
                        </div>
                    )}
                </TabsContent>

                {/* Templates Tab */}
                <TabsContent value="templates" className="space-y-4">
                    <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
                        {templates.map((template) => (
                            <Card key={template.id} className="hover:shadow-md transition-shadow">
                                <CardHeader>
                                    <CardTitle>{template.name}</CardTitle>
                                    <CardDescription>{template.description}</CardDescription>
                                </CardHeader>
                                <CardContent>
                                    <div className="flex flex-wrap gap-1 mb-4">
                                        {template.stages.map((stage) => (
                                            <Badge key={stage.id} variant="secondary" className="text-xs">
                                                {stage.name}
                                            </Badge>
                                        ))}
                                    </div>
                                    <Button className="w-full">
                                        <Play className="w-4 h-4 mr-2" />
                                        启动执行
                                    </Button>
                                </CardContent>
                            </Card>
                        ))}
                    </div>
                </TabsContent>

                {/* History Tab */}
                <TabsContent value="history">
                    <Card>
                        <CardContent className="flex flex-col items-center justify-center py-12">
                            <Clock className="w-12 h-12 text-muted-foreground mb-4" />
                            <p className="text-lg font-medium">执行历史</p>
                            <p className="text-sm text-muted-foreground">查看过去的流水线执行记录</p>
                        </CardContent>
                    </Card>
                </TabsContent>
            </Tabs>
        </div>
    )
}

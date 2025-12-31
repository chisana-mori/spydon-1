'use client'

import { useState, useEffect } from 'react'
import { useRouter } from 'next/navigation'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Separator } from '@/components/ui/separator'
import {
    ArrowLeft,
    ArrowRight,
    Play,
    CheckCircle,
    AlertTriangle,
    Server,
    Settings,
    Eye,
    Loader2,
    ChevronRight
} from 'lucide-react'
import type { Cluster } from '@/types/api'
import type { PipelineTemplate, StageDefinition } from '@/types/pipeline'

// Wizard Steps
type WizardStep = 'cluster' | 'template' | 'params' | 'review'

const steps: { id: WizardStep; title: string; description: string }[] = [
    { id: 'cluster', title: '选择集群', description: '选择目标集群' },
    { id: 'template', title: '选择模板', description: '选择流水线模板' },
    { id: 'params', title: '配置参数', description: '设置执行参数' },
    { id: 'review', title: '确认执行', description: '检查并启动' },
]

export default function PipelineLaunchWizard() {
    const router = useRouter()
    const [currentStep, setCurrentStep] = useState<WizardStep>('cluster')
    const [loading, setLoading] = useState(false)

    // Form state
    const [selectedCluster, setSelectedCluster] = useState<Cluster | null>(null)
    const [selectedTemplate, setSelectedTemplate] = useState<PipelineTemplate | null>(null)
    const [parameters, setParameters] = useState<Record<string, string>>({})

    // Mock data
    const [clusters] = useState<Cluster[]>([
        { id: '1', name: 'prod-cluster-01', status: 'active', description: '生产环境主集群', created_at: '', updated_at: '' },
        { id: '2', name: 'prod-cluster-02', status: 'active', description: '生产环境备用集群', created_at: '', updated_at: '' },
        { id: '3', name: 'staging-cluster-01', status: 'active', description: '测试环境集群', created_at: '', updated_at: '' },
    ])

    const [templates] = useState<PipelineTemplate[]>([
        {
            id: '1',
            name: 'K8s 集群升级',
            description: '标准的 Kubernetes 版本升级流程，包含预检查、节点驱逐、升级和验证',
            stages: [
                { id: '1', name: '预检查', type: 'pre_check', config: {}, on_failure: 'abort' },
                { id: '2', name: '节点驱逐', type: 'awx_job', config: { awx_template_name: 'drain-nodes' }, on_failure: 'rollback' },
                { id: '3', name: '人工确认', type: 'manual_gate', config: {}, on_failure: 'pause' },
                { id: '4', name: '版本升级', type: 'awx_job', config: { awx_template_name: 'upgrade-k8s' }, on_failure: 'rollback' },
            ],
            created_at: new Date().toISOString(),
            updated_at: new Date().toISOString(),
        },
        {
            id: '2',
            name: '节点扩容',
            description: '向集群添加新节点的标准流程',
            stages: [
                { id: '1', name: '资源检查', type: 'pre_check', config: {}, on_failure: 'abort' },
                { id: '2', name: '添加节点', type: 'awx_job', config: { awx_template_name: 'add-node' }, on_failure: 'rollback' },
            ],
            created_at: new Date().toISOString(),
            updated_at: new Date().toISOString(),
        },
    ])

    const currentStepIndex = steps.findIndex(s => s.id === currentStep)

    const canProceed = () => {
        switch (currentStep) {
            case 'cluster':
                return selectedCluster !== null
            case 'template':
                return selectedTemplate !== null
            case 'params':
                return true // 参数可选
            case 'review':
                return true
            default:
                return false
        }
    }

    const handleNext = () => {
        const nextIndex = currentStepIndex + 1
        if (nextIndex < steps.length) {
            setCurrentStep(steps[nextIndex].id)
        }
    }

    const handleBack = () => {
        const prevIndex = currentStepIndex - 1
        if (prevIndex >= 0) {
            setCurrentStep(steps[prevIndex].id)
        }
    }

    const handleLaunch = async () => {
        setLoading(true)
        // TODO: Call API to start execution
        await new Promise(resolve => setTimeout(resolve, 1500))
        setLoading(false)
        router.push('/pipelines')
    }

    return (
        <div className="container mx-auto p-6 max-w-4xl space-y-6">
            {/* Header */}
            <div className="flex items-center gap-4">
                <Button variant="ghost" size="icon" onClick={() => router.back()}>
                    <ArrowLeft className="w-5 h-5" />
                </Button>
                <div>
                    <h1 className="text-2xl font-bold">启动流水线</h1>
                    <p className="text-muted-foreground">按步骤配置并启动自动化运维流程</p>
                </div>
            </div>

            {/* Progress Steps */}
            <div className="flex items-center justify-between">
                {steps.map((step, idx) => (
                    <div key={step.id} className="flex items-center">
                        <div className={`
              flex items-center justify-center w-10 h-10 rounded-full border-2 font-medium
              ${idx < currentStepIndex ? 'bg-primary border-primary text-primary-foreground' :
                                idx === currentStepIndex ? 'border-primary text-primary' :
                                    'border-gray-300 text-gray-400'}
            `}>
                            {idx < currentStepIndex ? <CheckCircle className="w-5 h-5" /> : idx + 1}
                        </div>
                        <div className="ml-2 hidden sm:block">
                            <p className={`text-sm font-medium ${idx === currentStepIndex ? 'text-primary' : 'text-muted-foreground'}`}>
                                {step.title}
                            </p>
                        </div>
                        {idx < steps.length - 1 && (
                            <ChevronRight className="w-5 h-5 mx-4 text-gray-300" />
                        )}
                    </div>
                ))}
            </div>

            {/* Step Content */}
            <Card className="min-h-[400px]">
                {/* Step 1: Cluster Selection */}
                {currentStep === 'cluster' && (
                    <>
                        <CardHeader>
                            <CardTitle className="flex items-center gap-2">
                                <Server className="w-5 h-5" />
                                选择目标集群
                            </CardTitle>
                            <CardDescription>
                                选择要执行流水线的目标集群，系统会显示集群当前状态
                            </CardDescription>
                        </CardHeader>
                        <CardContent className="space-y-4">
                            {clusters.map((cluster) => (
                                <button
                                    key={cluster.id}
                                    onClick={() => setSelectedCluster(cluster)}
                                    className={`
                    w-full p-4 rounded-lg border-2 text-left transition-all
                    ${selectedCluster?.id === cluster.id
                                            ? 'border-primary bg-primary/5'
                                            : 'border-transparent bg-muted hover:border-gray-300'}
                  `}
                                >
                                    <div className="flex items-center justify-between">
                                        <div>
                                            <p className="font-medium">{cluster.name}</p>
                                            <p className="text-sm text-muted-foreground">{cluster.description}</p>
                                        </div>
                                        <Badge variant={cluster.status === 'active' ? 'default' : 'secondary'}>
                                            {cluster.status}
                                        </Badge>
                                    </div>
                                </button>
                            ))}
                        </CardContent>
                    </>
                )}

                {/* Step 2: Template Selection */}
                {currentStep === 'template' && (
                    <>
                        <CardHeader>
                            <CardTitle className="flex items-center gap-2">
                                <Play className="w-5 h-5" />
                                选择流水线模板
                            </CardTitle>
                            <CardDescription>
                                选择要执行的标准化运维流程模板
                            </CardDescription>
                        </CardHeader>
                        <CardContent className="space-y-4">
                            {templates.map((template) => (
                                <button
                                    key={template.id}
                                    onClick={() => setSelectedTemplate(template)}
                                    className={`
                    w-full p-4 rounded-lg border-2 text-left transition-all
                    ${selectedTemplate?.id === template.id
                                            ? 'border-primary bg-primary/5'
                                            : 'border-transparent bg-muted hover:border-gray-300'}
                  `}
                                >
                                    <div className="flex items-start justify-between">
                                        <div>
                                            <p className="font-medium">{template.name}</p>
                                            <p className="text-sm text-muted-foreground mt-1">{template.description}</p>
                                            <div className="flex flex-wrap gap-1 mt-2">
                                                {template.stages.map((stage) => (
                                                    <Badge key={stage.id} variant="secondary" className="text-xs">
                                                        {stage.name}
                                                    </Badge>
                                                ))}
                                            </div>
                                        </div>
                                        <Badge variant="outline">{template.stages.length} 阶段</Badge>
                                    </div>
                                </button>
                            ))}
                        </CardContent>
                    </>
                )}

                {/* Step 3: Parameters */}
                {currentStep === 'params' && (
                    <>
                        <CardHeader>
                            <CardTitle className="flex items-center gap-2">
                                <Settings className="w-5 h-5" />
                                配置执行参数
                            </CardTitle>
                            <CardDescription>
                                根据模板要求配置本次执行的参数
                            </CardDescription>
                        </CardHeader>
                        <CardContent className="space-y-6">
                            <div className="space-y-4">
                                <div className="space-y-2">
                                    <Label htmlFor="k8s_version">目标 K8s 版本</Label>
                                    <Select
                                        value={parameters.k8s_version || ''}
                                        onValueChange={(v) => setParameters({ ...parameters, k8s_version: v })}
                                    >
                                        <SelectTrigger>
                                            <SelectValue placeholder="选择版本" />
                                        </SelectTrigger>
                                        <SelectContent>
                                            <SelectItem value="1.28.5">1.28.5</SelectItem>
                                            <SelectItem value="1.29.0">1.29.0</SelectItem>
                                            <SelectItem value="1.29.1">1.29.1</SelectItem>
                                        </SelectContent>
                                    </Select>
                                </div>
                                <div className="space-y-2">
                                    <Label htmlFor="max_unavailable">最大不可用节点数</Label>
                                    <Input
                                        id="max_unavailable"
                                        type="number"
                                        value={parameters.max_unavailable || '1'}
                                        onChange={(e) => setParameters({ ...parameters, max_unavailable: e.target.value })}
                                    />
                                </div>
                                <div className="space-y-2">
                                    <Label htmlFor="notes">备注</Label>
                                    <Input
                                        id="notes"
                                        placeholder="本次执行的备注信息"
                                        value={parameters.notes || ''}
                                        onChange={(e) => setParameters({ ...parameters, notes: e.target.value })}
                                    />
                                </div>
                            </div>
                        </CardContent>
                    </>
                )}

                {/* Step 4: Review */}
                {currentStep === 'review' && (
                    <>
                        <CardHeader>
                            <CardTitle className="flex items-center gap-2">
                                <Eye className="w-5 h-5" />
                                确认执行
                            </CardTitle>
                            <CardDescription>
                                请检查以下配置，确认无误后启动流水线
                            </CardDescription>
                        </CardHeader>
                        <CardContent className="space-y-6">
                            {/* Summary */}
                            <div className="space-y-4">
                                <div className="flex items-center justify-between p-4 bg-muted rounded-lg">
                                    <span className="text-muted-foreground">目标集群</span>
                                    <span className="font-medium">{selectedCluster?.name}</span>
                                </div>
                                <div className="flex items-center justify-between p-4 bg-muted rounded-lg">
                                    <span className="text-muted-foreground">执行模板</span>
                                    <span className="font-medium">{selectedTemplate?.name}</span>
                                </div>
                                <div className="flex items-center justify-between p-4 bg-muted rounded-lg">
                                    <span className="text-muted-foreground">执行阶段</span>
                                    <span className="font-medium">{selectedTemplate?.stages.length} 个阶段</span>
                                </div>
                            </div>

                            <Separator />

                            {/* Warning */}
                            <div className="flex items-start gap-3 p-4 bg-yellow-50 dark:bg-yellow-950 rounded-lg border border-yellow-200 dark:border-yellow-800">
                                <AlertTriangle className="w-5 h-5 text-yellow-600 flex-shrink-0 mt-0.5" />
                                <div>
                                    <p className="font-medium text-yellow-800 dark:text-yellow-200">注意事项</p>
                                    <p className="text-sm text-yellow-700 dark:text-yellow-300">
                                        本次操作将对生产集群执行变更。流水线包含人工审批节点，在关键步骤执行前会等待您的确认。
                                    </p>
                                </div>
                            </div>
                        </CardContent>
                    </>
                )}
            </Card>

            {/* Navigation Buttons */}
            <div className="flex items-center justify-between">
                <Button variant="outline" onClick={handleBack} disabled={currentStepIndex === 0}>
                    <ArrowLeft className="w-4 h-4 mr-2" />
                    上一步
                </Button>

                {currentStep === 'review' ? (
                    <Button onClick={handleLaunch} disabled={loading}>
                        {loading ? (
                            <>
                                <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                                启动中...
                            </>
                        ) : (
                            <>
                                <Play className="w-4 h-4 mr-2" />
                                启动执行
                            </>
                        )}
                    </Button>
                ) : (
                    <Button onClick={handleNext} disabled={!canProceed()}>
                        下一步
                        <ArrowRight className="w-4 h-4 ml-2" />
                    </Button>
                )}
            </div>
        </div>
    )
}

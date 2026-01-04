'use client'

import { useState, useEffect, useCallback, useRef, useMemo } from 'react'
import yaml from 'js-yaml'
import { useRouter, useSearchParams } from 'next/navigation'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Badge } from '@/components/ui/badge'
import { Switch } from '@/components/ui/switch'
import { Separator } from '@/components/ui/separator'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from '@/components/ui/dropdown-menu'
import {
    Plus,
    Trash2,
    GripVertical,
    ChevronUp,
    ChevronDown,
    Loader2,
    Save,
    AlertCircle,
    Play,
    Pause,
    Clock,
    CheckCircle,
    Zap,
    ArrowLeft,
    ArrowDown,
    Settings2,
    Copy,
} from 'lucide-react'
import { toast } from 'sonner'
import { RobustaAPI } from '@/lib/api'
import type {
    PipelineTemplate,
    StageDefinition,
    StageType,
    FailureStrategy,
    StageConfig,
    ParameterBinding,
    ParameterInputType,
    AWXJobTemplate,
} from '@/types/pipeline'

// 阶段类型配置 - 使用 shadcn/ui 原生中性色调设计
const STAGE_TYPES: {
    value: StageType;
    label: string;
    icon: React.ReactNode;
    description: string;
    // 核心颜色 - 只保留必要的区分色
    accentColor: string; // 主题强调色 (用于小圆点等)
    iconClass: string; // 图标颜色
}[] = [
        {
            value: 'awx_job',
            label: 'AWX Job',
            icon: <Zap className="w-4 h-4" />,
            description: '执行 AWX/Ansible 作业',
            accentColor: 'bg-blue-500',
            iconClass: 'text-blue-600 dark:text-blue-400',
        },
        {
            value: 'manual_gate',
            label: '人工审批',
            icon: <Pause className="w-4 h-4" />,
            description: '等待人工确认',
            accentColor: 'bg-amber-500',
            iconClass: 'text-amber-600 dark:text-amber-400',
        },
        {
            value: 'delay',
            label: '延时等待',
            icon: <Clock className="w-4 h-4" />,
            description: '等待指定时间',
            accentColor: 'bg-violet-500',
            iconClass: 'text-violet-600 dark:text-violet-400',
        },
        {
            value: 'pre_check',
            label: '预检查',
            icon: <CheckCircle className="w-4 h-4" />,
            description: '前置条件检查',
            accentColor: 'bg-emerald-500',
            iconClass: 'text-emerald-600 dark:text-emerald-400',
        },
        {
            value: 'post_check',
            label: '后检查',
            icon: <CheckCircle className="w-4 h-4" />,
            description: '后置条件检查',
            accentColor: 'bg-teal-500',
            iconClass: 'text-teal-600 dark:text-teal-400',
        },
        {
            value: 'rollback',
            label: '回滚',
            icon: <AlertCircle className="w-4 h-4" />,
            description: '失败时回滚',
            accentColor: 'bg-rose-500',
            iconClass: 'text-rose-600 dark:text-rose-400',
        },
    ]

// 失败策略
const FAILURE_STRATEGIES: { value: FailureStrategy; label: string }[] = [
    { value: 'abort', label: '终止' },
    { value: 'rollback', label: '回滚' },
    { value: 'pause', label: '暂停' },
    { value: 'continue', label: '继续' },
]

const generateId = () => `stage_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`

const createDefaultStage = (type: StageType = 'awx_job'): StageDefinition => ({
    id: generateId(),
    name: STAGE_TYPES.find(t => t.value === type)?.label || '',
    type,
    config: {},
    on_failure: 'abort',
})

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

const flattenVariables = (obj: any, prefix = ''): Record<string, any> => {
    let result: Record<string, any> = {}
    if (!obj || typeof obj !== 'object') return {}

    Object.keys(obj).forEach(key => {
        const value = obj[key]
        const newKey = prefix ? `${prefix}.${key}` : key

        // 如果是数组，不继续递归，视为列表值
        if (Array.isArray(value)) {
            result[newKey] = value
        }
        // 如果是对象，递归
        else if (value && typeof value === 'object') {
            Object.assign(result, flattenVariables(value, newKey))
        } else {
            result[newKey] = value
        }
    })
    return result
}

// 提取变量发现逻辑到单独的组件，避免 Hook 调用顺序错误
const VariableDiscovery = ({
    awxTemplate,
    parameters,
    onAdd
}: {
    awxTemplate: AWXJobTemplate | undefined,
    parameters: ParameterBinding[],
    onAdd: (key: string, value: any) => void
}) => {
    const newVariables = useMemo(() => {
        if (!awxTemplate?.extra_vars) return {}

        let parsedVars = {}
        try {
            parsedVars = yaml.load(awxTemplate.extra_vars) as any
        } catch {
            try {
                parsedVars = JSON.parse(awxTemplate.extra_vars)
            } catch {
                return {}
            }
        }

        const flattened = flattenVariables(parsedVars)
        const existingKeys = new Set(parameters.map(p => p.name))
        const result: Record<string, any> = {}

        Object.entries(flattened).forEach(([key, val]) => {
            if (!existingKeys.has(key)) {
                result[key] = val
            }
        })
        return result
    }, [awxTemplate, parameters])

    return (
        <>
            {Object.entries(newVariables).map(([key, value]) => {
                const typeLabel = Array.isArray(value) ? 'List' : typeof value
                const valString = typeof value === 'object' ? JSON.stringify(value) : String(value)
                return (
                    <div key={`new-${key}`} className="flex items-center justify-between p-3 rounded-md border border-dashed bg-muted/40 group hover:bg-muted/60 transition-colors">
                        <div className="flex items-center gap-3">
                            <Badge variant="outline" className="text-[10px] border-blue-200 text-blue-600 bg-blue-50/50">New</Badge>
                            <div className="flex flex-col gap-0.5">
                                <span className="text-sm font-medium leading-none">{key}</span>
                                <div className="flex items-center gap-2 text-[10px] text-muted-foreground">
                                    <span>{typeLabel}</span>
                                    <span>•</span>
                                    <span className="truncate max-w-[200px]">Default: {valString}</span>
                                </div>
                            </div>
                        </div>
                        <Button size="sm" variant="ghost" onClick={() => onAdd(key, value)} className="h-7 text-xs hover:bg-background hover:text-primary">
                            <Plus className="w-3.5 h-3.5 mr-1" />
                            添加
                        </Button>
                    </div>
                )
            })}
            {parameters.length === 0 && Object.keys(newVariables).length === 0 && (
                <div className="text-center py-6 rounded-lg border border-dashed bg-muted/20">
                    <p className="text-sm text-muted-foreground">暂无参数</p>
                </div>
            )}
        </>
    )
}

export default function TemplateEditorPage() {
    const router = useRouter()
    const searchParams = useSearchParams()
    const templateId = searchParams.get('id')
    const isEditing = !!templateId

    const [name, setName] = useState('')
    const [description, setDescription] = useState('')
    const [stages, setStages] = useState<StageDefinition[]>([])
    const [selectedStageId, setSelectedStageId] = useState<string | null>(null)
    const [loading, setLoading] = useState(isEditing)
    const [saving, setSaving] = useState(false)
    const [errors, setErrors] = useState<Record<string, string>>({})
    const [awxTemplates, setAwxTemplates] = useState<AWXJobTemplate[]>([])
    // 缓存已获取详情的模板 (包含 extra_vars)
    const [fetchedTemplates, setFetchedTemplates] = useState<Record<number, AWXJobTemplate>>({})
    // 记录正在请求的 ID，防止重复请求
    const requestingRef = useRef<Set<number>>(new Set())

    const fetchAWXTemplateDetails = useCallback(async (id: number) => {
        // 如果已有数据，跳过
        if (fetchedTemplates[id] && fetchedTemplates[id].extra_vars !== undefined) return
        // 如果正在请求中，跳过
        if (requestingRef.current.has(id)) return

        requestingRef.current.add(id)
        try {
            const t = await RobustaAPI.getAWXTemplate(id)
            setFetchedTemplates(prev => ({ ...prev, [t.id]: t }))
        } catch (err) {
            console.error('Failed to fetch template details:', err)
        } finally {
            requestingRef.current.delete(id)
        }
    }, [fetchedTemplates])

    // 加载 AWX 模板列表
    useEffect(() => {
        RobustaAPI.listAWXTemplates()
            .then(setAwxTemplates)
            .catch(err => {
                console.error('Failed to load AWX templates:', err)
                toast.error('获取AWX模板列表失败')
            })
    }, [])

    useEffect(() => {
        if (templateId) {
            setLoading(true)
            RobustaAPI.getPipelineTemplate(templateId)
                .then((template) => {
                    setName(template.name)
                    setDescription(template.description || '')
                    const parsedStages = parseStages(template.stages)
                    setStages(parsedStages)
                    if (parsedStages.length > 0) {
                        setSelectedStageId(parsedStages[0].id)
                    }
                })
                .catch((err) => {
                    console.error('Failed to load template:', err)
                    toast.error('加载任务模板失败')
                    router.push('/tasks')
                })
                .finally(() => setLoading(false))
        } else {
            const defaultStage = createDefaultStage()
            setStages([defaultStage])
            setSelectedStageId(defaultStage.id)
        }
    }, [templateId, router])

    const selectedStage = stages.find(s => s.id === selectedStageId) || null

    // 提取当前选中阶段的 AWX Template ID，避免 useEffect 依赖整个 stages 数组
    const currentAwxTemplateId = (selectedStage?.type === 'awx_job') ? selectedStage.config.awx_template_id : undefined

    // 自动获取选中阶段的 AWX 模板详情
    useEffect(() => {
        if (currentAwxTemplateId) {
            fetchAWXTemplateDetails(currentAwxTemplateId)
        }
    }, [currentAwxTemplateId, fetchAWXTemplateDetails])

    // 在指定位置插入阶段
    const insertStageAt = (index: number, type: StageType) => {
        const newStage = createDefaultStage(type)
        const newStages = [...stages]
        newStages.splice(index, 0, newStage)
        setStages(newStages)
        setSelectedStageId(newStage.id)
    }

    const duplicateStage = (id: string) => {
        const idx = stages.findIndex(s => s.id === id)
        if (idx === -1) return
        const sourceStage = stages[idx]
        const newStage: StageDefinition = {
            ...sourceStage,
            id: generateId(),
            name: `${sourceStage.name} (Copy)`,
        }
        const newStages = [...stages]
        newStages.splice(idx + 1, 0, newStage)
        setStages(newStages)
        setSelectedStageId(newStage.id)
        toast.success('阶段已复制')
    }

    const removeStage = (id: string) => {
        if (stages.length <= 1) {
            toast.error('至少需要一个阶段')
            return
        }
        const idx = stages.findIndex(s => s.id === id)
        const newStages = stages.filter(s => s.id !== id)
        setStages(newStages)
        if (selectedStageId === id) {
            const newIdx = Math.min(idx, newStages.length - 1)
            setSelectedStageId(newStages[newIdx]?.id || null)
        }
    }

    const updateStage = (id: string, updates: Partial<StageDefinition>) => {
        setStages(stages.map(s => s.id === id ? { ...s, ...updates } : s))
    }

    const updateStageConfig = (id: string, configUpdates: Partial<StageConfig>) => {
        setStages(stages.map(s => {
            if (s.id === id) {
                return { ...s, config: { ...s.config, ...configUpdates } }
            }
            return s
        }))
    }

    const moveStage = (id: string, direction: 'up' | 'down') => {
        const idx = stages.findIndex(s => s.id === id)
        if (idx === -1) return
        if (direction === 'up' && idx === 0) return
        if (direction === 'down' && idx === stages.length - 1) return
        const newStages = [...stages]
        const targetIdx = direction === 'up' ? idx - 1 : idx + 1
        const temp = newStages[idx]
        newStages[idx] = newStages[targetIdx]
        newStages[targetIdx] = temp
        setStages(newStages)
    }

    const validate = (): boolean => {
        const newErrors: Record<string, string> = {}
        if (!name.trim()) newErrors.name = '请输入模板名称'
        stages.forEach((stage, idx) => {
            if (!stage.name.trim()) newErrors[`stage_${stage.id}_name`] = `阶段 ${idx + 1} 需要名称`
            if (stage.type === 'awx_job' && !stage.config.awx_template_name?.trim()) {
                newErrors[`stage_${stage.id}_awx`] = 'AWX 模板名称必填'
            }
            if (stage.type === 'delay' && (!stage.config.delay_seconds || stage.config.delay_seconds <= 0)) {
                newErrors[`stage_${stage.id}_delay`] = '延时秒数必填'
            }
        })
        setErrors(newErrors)
        return Object.keys(newErrors).length === 0
    }

    const handleSave = async () => {
        if (!validate()) {
            toast.error('请修正表单错误')
            return
        }
        setSaving(true)
        try {
            const data = {
                name: name.trim(),
                description: description.trim(),
                stages: stages.map(s => ({ ...s, name: s.name.trim() })),
            }
            if (isEditing && templateId) {
                await RobustaAPI.updatePipelineTemplate(templateId, data)
                toast.success('模板已更新')
            } else {
                await RobustaAPI.createPipelineTemplate(data)
                toast.success('模板已创建')
            }
            router.push('/tasks?tab=templates')
        } catch (error) {
            console.error('Failed to save template:', error)
            toast.error(isEditing ? '更新失败' : '创建失败')
        } finally {
            setSaving(false)
        }
    }

    // 添加阶段按钮
    const AddStageButton = ({ insertIndex }: { insertIndex: number }) => (
        <DropdownMenu>
            <DropdownMenuTrigger asChild>
                <div className="flex items-center justify-center py-1 cursor-pointer group">
                    <div className="flex items-center gap-1 px-2 py-1 rounded border border-dashed border-slate-300 dark:border-slate-600 hover:border-primary hover:bg-primary/5 transition-colors">
                        <Plus className="w-3 h-3 text-muted-foreground group-hover:text-primary" />
                        <span className="text-[10px] text-muted-foreground group-hover:text-primary">添加</span>
                    </div>
                </div>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="center" className="w-44">
                {STAGE_TYPES.map(t => (
                    <DropdownMenuItem key={t.value} onClick={() => insertStageAt(insertIndex, t.value)} className="text-xs gap-2">
                        <div className={`w-2 h-2 rounded-full ${t.accentColor}`} />
                        {t.label}
                    </DropdownMenuItem>
                ))}
            </DropdownMenuContent>
        </DropdownMenu>
    )

    // 渲染阶段配置表单
    const renderStageConfig = (stage: StageDefinition) => {
        const renderSpecificConfig = () => {
            switch (stage.type) {
                case 'awx_job':
                    const parameters = stage.config.parameters || []

                    // 解析 extra_vars 并计算新发现的变量
                    const awxTemplate = stage.config.awx_template_id ? fetchedTemplates[stage.config.awx_template_id] : undefined

                    const addParameter = () => {
                        const newParam: ParameterBinding = { name: '', label: '', input_type: 'text', required: false }
                        updateStageConfig(stage.id, { parameters: [...parameters, newParam] })
                    }
                    const addDiscoveredParameter = (key: string, value: any) => {
                        const type = Array.isArray(value) ? 'multi_select' : 'text'
                        const newParam: ParameterBinding = {
                            name: key,
                            label: key.split('.').pop() || key,
                            input_type: type,
                            required: false,
                            default_value: typeof value === 'object' ? JSON.stringify(value) : String(value)
                        }
                        updateStageConfig(stage.id, { parameters: [...parameters, newParam] })
                        toast.success(`已添加参数: ${key}`)
                    }
                    const updateParameter = (idx: number, updates: Partial<ParameterBinding>) => {
                        const newParams = [...parameters]
                        newParams[idx] = { ...newParams[idx], ...updates }
                        updateStageConfig(stage.id, { parameters: newParams })
                    }
                    const removeParameter = (idx: number) => {
                        updateStageConfig(stage.id, { parameters: parameters.filter((_, i) => i !== idx) })
                    }

                    return (
                        <div className="space-y-4">
                            <Card className="shadow-none">
                                <CardHeader className="pb-4">
                                    <CardTitle className="text-sm font-medium">任务配置</CardTitle>
                                </CardHeader>
                                <CardContent className="space-y-4">
                                    <div className="space-y-2">
                                        <Label className="text-sm">AWX 模板名称 *</Label>
                                        <Select
                                            value={stage.config.awx_template_name}
                                            onValueChange={(value) => {
                                                const template = awxTemplates.find(t => t.name === value)
                                                updateStageConfig(stage.id, template ? { awx_template_name: template.name, awx_template_id: template.id } : { awx_template_name: value })
                                            }}
                                        >
                                            <SelectTrigger><SelectValue placeholder="选择 AWX 模板" /></SelectTrigger>
                                            <SelectContent>
                                                {awxTemplates.map((t) => (
                                                    <SelectItem key={t.id} value={t.name}>{t.name} <span className="text-muted-foreground text-xs ml-2">({t.job_type})</span></SelectItem>
                                                ))}
                                            </SelectContent>
                                        </Select>
                                        {/* 触发获取详情 */}
                                        {stage.config.awx_template_id && (
                                            <div className="hidden"></div>
                                        )}
                                    </div>
                                    <div className="flex items-center justify-between p-3 rounded-lg border bg-muted/30">
                                        <div className="space-y-0.5">
                                            <Label className="text-sm">Dry Run (检查模式)</Label>
                                            <p className="text-xs text-muted-foreground">执行检查而不进行实际变更</p>
                                        </div>
                                        <Switch checked={stage.config.dry_run || false} onCheckedChange={(c) => updateStageConfig(stage.id, { dry_run: c })} />
                                    </div>
                                </CardContent>
                            </Card>

                            <Card className="shadow-none">
                                <CardHeader className="pb-4 flex flex-row items-center justify-between space-y-0">
                                    <div className="space-y-1">
                                        <CardTitle className="text-sm font-medium">参数绑定</CardTitle>
                                        <CardDescription className="text-xs">定义执行时需要的输入参数</CardDescription>
                                    </div>
                                    <Button variant="outline" size="sm" onClick={addParameter} className="h-8">
                                        <Plus className="w-3.5 h-3.5 mr-1.5" />
                                        添加参数
                                    </Button>
                                </CardHeader>
                                <CardContent className="space-y-3">
                                    {/* 1. 已配置参数列表 */}
                                    {parameters.map((param, idx) => (
                                        <Card key={`existing-${idx}`} className="shadow-none border bg-card/50 relative group transition-all hover:bg-card">
                                            <CardContent className="p-3 space-y-3">
                                                <div className="absolute right-2 top-2">
                                                    <Button
                                                        variant="ghost" size="icon"
                                                        className="h-6 w-6 text-muted-foreground hover:text-destructive opacity-0 group-hover:opacity-100 transition-opacity"
                                                        onClick={() => removeParameter(idx)}
                                                    >
                                                        <Trash2 className="w-3.5 h-3.5" />
                                                    </Button>
                                                </div>

                                                <div className="grid grid-cols-2 gap-3 pr-6">
                                                    <div className="space-y-1.5">
                                                        <Label className="text-xs text-muted-foreground">参数名 (Key)</Label>
                                                        <Input value={param.name} onChange={(e) => updateParameter(idx, { name: e.target.value })} className="h-8 shadow-none" placeholder="var_name" />
                                                    </div>
                                                    <div className="space-y-1.5">
                                                        <Label className="text-xs text-muted-foreground">显示名称 (Label)</Label>
                                                        <Input value={param.label} onChange={(e) => updateParameter(idx, { label: e.target.value })} className="h-8 shadow-none" placeholder="Friendly Name" />
                                                    </div>
                                                    <div className="space-y-1.5">
                                                        <Label className="text-xs text-muted-foreground">类型</Label>
                                                        <Select value={param.input_type} onValueChange={(v) => updateParameter(idx, { input_type: v as ParameterInputType })}>
                                                            <SelectTrigger className="h-8 shadow-none"><SelectValue /></SelectTrigger>
                                                            <SelectContent>
                                                                <SelectItem value="text">文本输入</SelectItem>
                                                                <SelectItem value="select">单选</SelectItem>
                                                                <SelectItem value="multi_select">多选</SelectItem>
                                                                <SelectItem value="fixed">固定值</SelectItem>
                                                            </SelectContent>
                                                        </Select>
                                                    </div>
                                                    <div className="space-y-1.5">
                                                        <Label className="text-xs text-muted-foreground">默认值</Label>
                                                        <Input value={param.default_value || ''} onChange={(e) => updateParameter(idx, { default_value: e.target.value })} className="h-8 shadow-none" />
                                                    </div>
                                                </div>

                                                {(param.input_type === 'select' || param.input_type === 'multi_select') && (
                                                    <div className="space-y-1.5">
                                                        <Label className="text-xs text-muted-foreground">选项 (逗号分隔)</Label>
                                                        <Input value={(param.options || []).join(', ')} onChange={(e) => updateParameter(idx, { options: e.target.value.split(',').map(s => s.trim()).filter(Boolean) })} className="h-8 shadow-none" placeholder="opt1, opt2" />
                                                    </div>
                                                )}

                                                <div className="flex items-center gap-2 pt-1">
                                                    <Switch id={`req-${idx}`} checked={param.required} onCheckedChange={(c) => updateParameter(idx, { required: c })} className="scale-75" />
                                                    <Label htmlFor={`req-${idx}`} className="text-xs cursor-pointer text-muted-foreground font-normal">必填参数</Label>
                                                </div>
                                            </CardContent>
                                        </Card>
                                    ))}

                                    {/* 2. 新发现变量列表 (蓝色) - 使用单独组件渲染 */}
                                    <VariableDiscovery
                                        awxTemplate={awxTemplate}
                                        parameters={parameters}
                                        onAdd={addDiscoveredParameter}
                                    />
                                </CardContent>
                            </Card>
                        </div>
                    )

                case 'manual_gate':
                    return (
                        <Card className="shadow-none">
                            <CardHeader className="pb-4">
                                <CardTitle className="text-sm font-medium">审批配置</CardTitle>
                            </CardHeader>
                            <CardContent>
                                <div className="space-y-2 max-w-sm">
                                    <Label className="text-sm">超时时间 (分钟)</Label>
                                    <Input type="number" value={stage.config.timeout_minutes || ''} onChange={(e) => updateStageConfig(stage.id, { timeout_minutes: parseInt(e.target.value) || undefined })} placeholder="无限制" />
                                    <p className="text-xs text-muted-foreground">超过此时间未审批将自动按失败处理</p>
                                </div>
                            </CardContent>
                        </Card>
                    )

                case 'delay':
                    return (
                        <Card className="shadow-none">
                            <CardHeader className="pb-4">
                                <CardTitle className="text-sm font-medium">延时配置</CardTitle>
                            </CardHeader>
                            <CardContent>
                                <div className="space-y-2 max-w-sm">
                                    <Label className="text-sm">等待时长 (秒) *</Label>
                                    <div className="relative">
                                        <Clock className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                                        <Input type="number" className="pl-10" value={stage.config.delay_seconds || ''} onChange={(e) => updateStageConfig(stage.id, { delay_seconds: parseInt(e.target.value) || undefined })} placeholder="60" />
                                    </div>
                                </div>
                            </CardContent>
                        </Card>
                    )

                case 'pre_check':
                case 'post_check':
                    return (
                        <Card className="shadow-none">
                            <CardHeader className="pb-4">
                                <CardTitle className="text-sm font-medium">检查规则</CardTitle>
                            </CardHeader>
                            <CardContent className="space-y-4">
                                <div className="space-y-2">
                                    <Label className="text-sm">Prometheus PromQL 查询</Label>
                                    <Textarea value={stage.config.prom_query || ''} onChange={(e) => updateStageConfig(stage.id, { prom_query: e.target.value })} placeholder="up{job='my-target'} == 1" className="font-mono text-sm bg-muted/20" rows={3} />
                                </div>
                                <div className="grid grid-cols-2 gap-4">
                                    <div className="space-y-2">
                                        <Label className="text-sm">比较操作符</Label>
                                        <Select value={stage.config.operator || 'eq'} onValueChange={(v) => updateStageConfig(stage.id, { operator: v as 'eq' | 'ne' | 'gt' | 'lt' })}>
                                            <SelectTrigger><SelectValue /></SelectTrigger>
                                            <SelectContent>
                                                <SelectItem value="eq">等于 (=)</SelectItem>
                                                <SelectItem value="ne">不等于 (≠)</SelectItem>
                                                <SelectItem value="gt">大于 (&gt;)</SelectItem>
                                                <SelectItem value="lt">小于 (&lt;)</SelectItem>
                                            </SelectContent>
                                        </Select>
                                    </div>
                                    <div className="space-y-2">
                                        <Label className="text-sm">期望结果值</Label>
                                        <Input value={stage.config.expected_value || ''} onChange={(e) => updateStageConfig(stage.id, { expected_value: e.target.value })} placeholder="1" />
                                    </div>
                                </div>
                            </CardContent>
                        </Card>
                    )

                default:
                    return <div className="p-4 text-sm text-muted-foreground">此类阶段无需额外配置</div>
            }
        }

        return (
            <Tabs defaultValue="config" className="w-full">
                <TabsList className="mb-4">
                    <TabsTrigger value="config">阶段配置</TabsTrigger>
                    <TabsTrigger value="settings">高级设置</TabsTrigger>
                </TabsList>

                <TabsContent value="config" className="space-y-4">
                    {/* 通用基本信息 */}
                    <Card className="shadow-none">
                        <CardHeader className="pb-4">
                            <CardTitle className="text-sm font-medium">基本信息</CardTitle>
                        </CardHeader>
                        <CardContent className="grid md:grid-cols-2 gap-4">
                            <div className="space-y-2">
                                <Label className="text-sm">阶段名称 *</Label>
                                <Input value={stage.name} onChange={(e) => updateStage(stage.id, { name: e.target.value })} />
                            </div>
                            <div className="space-y-2">
                                <Label className="text-sm">阶段类型</Label>
                                <Select value={stage.type} onValueChange={(v) => updateStage(stage.id, { type: v as StageType, config: {} })}>
                                    <SelectTrigger><SelectValue /></SelectTrigger>
                                    <SelectContent>
                                        {STAGE_TYPES.map(t => (
                                            <SelectItem key={t.value} value={t.value}>
                                                <div className="flex items-center gap-2">
                                                    <div className={`w-2 h-2 rounded-full ${t.accentColor}`} />
                                                    {t.label}
                                                </div>
                                            </SelectItem>
                                        ))}
                                    </SelectContent>
                                </Select>
                            </div>
                        </CardContent>
                    </Card>

                    {/* 类型特定配置 */}
                    {renderSpecificConfig()}
                </TabsContent>

                <TabsContent value="settings" className="space-y-4">
                    <Card className="shadow-none">
                        <CardHeader className="pb-4">
                            <CardTitle className="text-sm font-medium">失败策略</CardTitle>
                            <CardDescription>当此阶段执行失败时系统的行为</CardDescription>
                        </CardHeader>
                        <CardContent>
                            <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
                                {FAILURE_STRATEGIES.map(s => (
                                    <div
                                        key={s.value}
                                        onClick={() => updateStage(stage.id, { on_failure: s.value })}
                                        className={`cursor-pointer rounded-lg border p-3 text-center transition-all hover:bg-accent ${stage.on_failure === s.value ? 'border-primary bg-accent' : 'border-border bg-muted/30'}`}
                                    >
                                        <div className="text-sm font-medium">{s.label}</div>
                                        <div className="text-xs text-muted-foreground mt-0.5">{s.value}</div>
                                    </div>
                                ))}
                            </div>
                        </CardContent>
                    </Card>

                    {stage.type === 'awx_job' && (
                        <div className="p-3 rounded-lg bg-muted/50 text-sm text-muted-foreground border">
                            提示: AWX Job 失败通常意味着 Ansible Playbook 执行返回非零状态码。
                        </div>
                    )}
                </TabsContent>
            </Tabs>
        )
    }

    if (loading) return <div className="flex h-screen items-center justify-center"><Loader2 className="h-8 w-8 animate-spin text-primary" /></div>

    return (
        <div className="h-screen flex flex-col bg-background">
            {/* 顶栏 */}
            <header className="h-14 border-b bg-background px-4 flex items-center justify-between shrink-0 z-10">
                <div className="flex items-center gap-4">
                    <Button variant="ghost" size="sm" onClick={() => router.push('/tasks?tab=templates')} className="text-muted-foreground hover:text-foreground">
                        <ArrowLeft className="w-4 h-4 mr-1" />
                        返回
                    </Button>
                    <Separator orientation="vertical" className="h-5" />
                    <div className="flex items-center gap-3">
                        <span className="font-semibold text-foreground">{name || '未命名模板'}</span>
                        <Badge variant="secondary" className="font-normal text-xs">{isEditing ? '编辑模式' : '创建模式'}</Badge>
                    </div>
                </div>
                <div className="flex items-center gap-2">
                    <Button variant="ghost" size="sm" onClick={() => router.push('/tasks?tab=templates')}>取消</Button>
                    <Button size="sm" onClick={handleSave} disabled={saving}>
                        {saving ? <Loader2 className="w-4 h-4 mr-1.5 animate-spin" /> : <Save className="w-4 h-4 mr-1.5" />}
                        保存
                    </Button>
                </div>
            </header>

            {/* 主体布局 */}
            <div className="flex-1 flex overflow-hidden">

                {/* 左侧：导航与设置 */}
                <div className="w-[300px] bg-muted/30 border-r flex flex-col shrink-0">
                    <div className="p-4 border-b bg-background">
                        <Label className="text-[11px] font-medium text-muted-foreground mb-3 block tracking-wide">模板属性</Label>
                        <div className="space-y-3">
                            <Input value={name} onChange={(e) => setName(e.target.value)} placeholder="输入模板名称..." className="h-9" />
                            <Textarea value={description} onChange={(e) => setDescription(e.target.value)} placeholder="描述 (可选)..." rows={2} className="text-sm resize-none" />
                        </div>
                    </div>

                    <div className="flex-1 flex flex-col min-h-0">
                        <div className="px-4 py-3 flex items-center justify-between border-b bg-background">
                            <span className="text-[11px] font-medium text-muted-foreground tracking-wide">执行流程 ({stages.length})</span>
                            <AddStageButton insertIndex={stages.length} />
                        </div>
                        <ScrollArea className="flex-1">
                            <div className="p-3 space-y-2">
                                {stages.map((stage, idx) => {
                                    const typeConfig = STAGE_TYPES.find(t => t.value === stage.type)
                                    const isSelected = selectedStageId === stage.id
                                    return (
                                        <div
                                            key={stage.id}
                                            onClick={() => setSelectedStageId(stage.id)}
                                            className={`
                                                relative group flex items-start gap-3 p-3 rounded-lg cursor-pointer transition-all border
                                                ${isSelected
                                                    ? 'bg-accent border-border shadow-sm'
                                                    : 'bg-card border-transparent hover:bg-accent/50 hover:border-border/50'}
                                            `}
                                        >

                                            {/* 序号线 */}
                                            <div className="flex flex-col items-center self-stretch gap-1 pt-1">
                                                <div className={`w-5 h-5 rounded-full flex items-center justify-center text-[10px] font-bold transition-colors ${isSelected ? 'bg-primary text-primary-foreground' : 'bg-muted text-muted-foreground'}`}>
                                                    {idx + 1}
                                                </div>
                                                {idx < stages.length - 1 && <div className="w-px flex-1 bg-border my-1" />}
                                            </div>

                                            <div className="flex-1 min-w-0">
                                                <div className="flex items-center justify-between mb-1">
                                                    <span className={`text-sm font-medium truncate ${isSelected ? 'text-foreground' : 'text-muted-foreground group-hover:text-foreground'}`}>
                                                        {stage.name || '未命名阶段'}
                                                    </span>
                                                    {/* 操作栏 */}
                                                    <div className={`flex items-center gap-1 opacity-0 ${isSelected ? 'opacity-100' : 'group-hover:opacity-100'} transition-opacity`}>
                                                        <Button variant="ghost" size="icon" className="h-5 w-5" onClick={(e) => { e.stopPropagation(); moveStage(stage.id, 'up') }} disabled={idx === 0}><ChevronUp className="w-3 h-3" /></Button>
                                                        <Button variant="ghost" size="icon" className="h-5 w-5" onClick={(e) => { e.stopPropagation(); moveStage(stage.id, 'down') }} disabled={idx === stages.length - 1}><ChevronDown className="w-3 h-3" /></Button>
                                                        <Button variant="ghost" size="icon" className="h-5 w-5" onClick={(e) => { e.stopPropagation(); duplicateStage(stage.id) }} title="复制"><Copy className="w-3 h-3" /></Button>
                                                        <Button variant="ghost" size="icon" className="h-5 w-5 text-destructive hover:text-destructive" onClick={(e) => { e.stopPropagation(); removeStage(stage.id) }}><Trash2 className="w-3 h-3" /></Button>
                                                    </div>
                                                </div>
                                                <div className="flex items-center text-xs text-muted-foreground gap-2">
                                                    <Badge variant="secondary" className="h-5 px-1.5 font-normal gap-1">
                                                        <div className={`w-1.5 h-1.5 rounded-full ${typeConfig?.accentColor}`} />
                                                        <span className="text-muted-foreground">{typeConfig?.label}</span>
                                                    </Badge>
                                                    {stage.on_failure !== 'abort' && <Badge variant="outline" className="h-5 px-1 text-[10px]">Fail: {stage.on_failure}</Badge>}
                                                </div>
                                            </div>
                                        </div>
                                    )
                                })}
                                {stages.length === 0 && (
                                    <div className="text-center py-10 text-muted-foreground text-xs">
                                        点击上方 + 添加第一个阶段
                                    </div>
                                )}
                            </div>
                        </ScrollArea>
                    </div>
                </div>

                {/* 中间：配置区域 */}
                <div className="flex-1 overflow-auto bg-muted/10 p-6">
                    {selectedStage ? (
                        <div className="max-w-4xl mx-auto space-y-6">
                            <div className="flex items-start gap-4">
                                <div className={`mt-1 w-12 h-12 rounded-xl flex items-center justify-center bg-muted border border-border ${STAGE_TYPES.find(t => t.value === selectedStage.type)?.iconClass}`}>
                                    {STAGE_TYPES.find(t => t.value === selectedStage.type)?.icon}
                                </div>
                                <div>
                                    <h1 className="text-2xl font-semibold text-foreground tracking-tight">{selectedStage.name || 'New Stage'}</h1>
                                    <p className="text-muted-foreground text-sm">{STAGE_TYPES.find(t => t.value === selectedStage.type)?.description}</p>
                                </div>
                            </div>

                            {/* 配置表单 */}
                            {renderStageConfig(selectedStage)}
                        </div>
                    ) : (
                        <div className="h-full flex flex-col items-center justify-center text-muted-foreground opacity-50">
                            <Settings2 className="w-16 h-16 mb-4 stroke-1" />
                            <p className="text-lg">选择左侧阶段开始配置</p>
                        </div>
                    )}
                </div>

                {/* 右侧：MiniMap / 概览 */}
                <div className="w-[220px] border-l bg-muted/20 flex flex-col shrink-0">
                    <div className="p-3 border-b bg-background text-center font-medium text-[11px] text-muted-foreground tracking-wide">Live Preview</div>
                    <ScrollArea className="flex-1 h-full">
                        <div className="relative flex flex-col items-center w-full py-6 space-y-4">
                            {/* Connecting Line */}
                            <div className="absolute top-10 bottom-10 left-1/2 w-px bg-border -z-10" />

                            {/* Start Node */}
                            <div className="relative group">
                                <div className="w-7 h-7 rounded-full bg-emerald-500 text-white flex items-center justify-center shadow-sm z-10 ring-2 ring-background transition-transform group-hover:scale-110">
                                    <Play className="w-3 h-3 fill-current" />
                                </div>
                                <div className="absolute left-9 top-1/2 -translate-y-1/2 bg-popover text-popover-foreground text-[10px] px-2 py-1 rounded-md shadow-sm border opacity-0 group-hover:opacity-100 transition-opacity whitespace-nowrap z-20 pointer-events-none">
                                    开始
                                </div>
                            </div>

                            {stages.map((stage, idx) => {
                                const typeConfig = STAGE_TYPES.find(t => t.value === stage.type)
                                const isSelected = selectedStageId === stage.id
                                return (
                                    <div
                                        key={stage.id}
                                        className={`
                                            relative z-10 w-44 p-2.5 rounded-lg border transition-all cursor-pointer group
                                            ${isSelected
                                                ? 'bg-background border-primary shadow-sm scale-[1.02] z-20'
                                                : 'bg-card border-border hover:border-primary/40 hover:shadow-sm'}
                                        `}
                                        onClick={() => setSelectedStageId(stage.id)}
                                    >
                                        <div className="flex items-center gap-2 mb-1.5">
                                            <div className={`p-1 rounded ${typeConfig?.iconClass}`}>
                                                {typeConfig?.icon}
                                            </div>
                                            <span className={`text-[11px] font-medium truncate flex-1 ${isSelected ? 'text-foreground' : 'text-muted-foreground group-hover:text-foreground'}`}>
                                                {stage.name}
                                            </span>
                                        </div>
                                        <div className="text-[10px] text-muted-foreground flex justify-between items-center">
                                            <span className="flex items-center gap-1">
                                                <div className={`w-1.5 h-1.5 rounded-full ${typeConfig?.accentColor}`} />
                                                {typeConfig?.label}
                                            </span>
                                            <span className="font-mono text-[9px] opacity-50">#{idx + 1}</span>
                                        </div>

                                        {/* Connector Dot on Left/Right? No, just the card on top of line */}
                                    </div>
                                )
                            })}

                            {/* End Node */}
                            <div className="relative group">
                                <div className="w-7 h-7 rounded-full bg-muted text-muted-foreground flex items-center justify-center shadow-sm z-10 ring-2 ring-background transition-transform group-hover:scale-110">
                                    <CheckCircle className="w-3.5 h-3.5" />
                                </div>
                                <div className="absolute left-9 top-1/2 -translate-y-1/2 bg-popover text-popover-foreground text-[10px] px-2 py-1 rounded-md shadow-sm border opacity-0 group-hover:opacity-100 transition-opacity whitespace-nowrap z-20 pointer-events-none">
                                    结束
                                </div>
                            </div>
                        </div>
                    </ScrollArea>
                </div>
            </div>
        </div>
    )
}

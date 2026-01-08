'use client'

import { useState, useEffect } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import * as z from 'zod'
import { toast } from 'sonner'
import { Loader2, Server, Settings2, Network } from 'lucide-react'
import yaml from 'js-yaml'

import { useDictionaryPreload, CLUSTER_DICTIONARY_CODES } from '@/hooks/useDictionaryPreload'

import { Button } from '@/components/ui/button'
import { DictionarySelect } from '@/components/common/DictionarySelect'
import { DictionaryMultiSelect } from '@/components/common/DictionaryMultiSelect'
import { DictionaryCodeBadge } from '@/components/common/DictionaryCodeBadge'
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle,
} from '@/components/ui/dialog'
import {
    Form,
    FormControl,
    FormField,
    FormItem,
    FormLabel,
    FormMessage,
    FormDescription,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from '@/components/ui/select'
import { YamlEditor } from '@/components/ui/yaml-editor'
import { Separator } from '@/components/ui/separator'
import RobustaAPI from '@/lib/api'
import { Cluster } from '@/types/api'

const formSchema = z.object({
    name: z.string().min(1, '请输入集群名称').max(255, '名称不能超过255个字符'),
    cluster_id: z.string().optional(),
    description: z.string().optional(),
    status: z.enum(['active', 'inactive', 'maintenance']).default('active'),
    kube_config: z.string().optional().refine((val) => {
        if (!val) return true
        try {
            yaml.load(val)
            return true
        } catch (e) {
            return false
        }
    }, '请输入有效的 YAML 格式'),
    prometheus_url: z.string().url('请输入有效的 URL').optional().or(z.literal('')),
    // Navy fields
    cluster_version: z.string().optional(),
    idc: z.string().optional(),
    zone: z.string().optional(),
    flow_type: z.string().optional(),
    purpose: z.string().optional(),
    arch: z.string().optional(),
    priority: z.string().optional(),
    cluster_group: z.string().optional(),
})

type FormValues = z.infer<typeof formSchema>

interface ClusterDialogProps {
    open: boolean
    onOpenChange: (open: boolean) => void
    cluster?: Cluster | null // If present, it's edit mode
    onSuccess: () => void
}

export function ClusterDialog({
    open,
    onOpenChange,
    cluster,
    onSuccess,
}: ClusterDialogProps) {
    const [loading, setLoading] = useState(false)
    const isEdit = !!cluster

    // Preload all required dictionaries in parallel
    useDictionaryPreload([...CLUSTER_DICTIONARY_CODES])

    const form = useForm({
        resolver: zodResolver(formSchema),
        defaultValues: {
            name: '',
            cluster_id: '',
            description: '',
            status: 'active' as const,
            kube_config: '',
            prometheus_url: '',
            // Navy fields
            cluster_version: '',
            idc: '',
            zone: '',
            flow_type: '',
            purpose: '',
            arch: '',
            priority: '0',
            cluster_group: '',
        },
    })

    // Reset form when opening/closing or cluster changes
    useEffect(() => {
        if (open) {
            if (cluster) {
                form.reset({
                    name: cluster.name,
                    cluster_id: cluster.cluster_id || '',
                    description: cluster.description || '',
                    status: cluster.status || 'active',
                    kube_config: cluster.kube_config || '',
                    prometheus_url: cluster.prometheus_url || '',
                    cluster_version: cluster.cluster_version || '',
                    idc: cluster.idc || '',
                    zone: cluster.zone || '',
                    flow_type: cluster.flow_type || '',
                    purpose: cluster.purpose || '',
                    arch: cluster.arch || '',
                    priority: cluster.priority?.toString() || '0',
                    cluster_group: cluster.cluster_group || '',
                })
            } else {
                form.reset({
                    name: '',
                    cluster_id: '',
                    description: '',
                    status: 'active',
                    kube_config: '',
                    prometheus_url: '',
                    cluster_version: '',
                    idc: '',
                    zone: '',
                    flow_type: '',
                    purpose: '',
                    arch: '',
                    priority: '0',
                    cluster_group: '',
                })
            }
        }
    }, [open, cluster, form])

    const formatKubeConfig = (val: string) => {
        if (!val) return
        try {
            const parsed = yaml.load(val)
            const formatted = yaml.dump(parsed)
            form.setValue('kube_config', formatted)
            toast.success('KubeConfig 格式化成功')
        } catch (e) {
            form.setError('kube_config', {
                type: 'manual',
                message: '无法格式化：无效的 YAML'
            })
        }
    }

    async function onSubmit(values: FormValues) {
        setLoading(true)
        try {
            if (isEdit && cluster) {
                await RobustaAPI.updateCluster(cluster.name, {
                    description: values.description,
                    kube_config: values.kube_config,
                    prometheus_url: values.prometheus_url,
                    status: values.status,
                    cluster_version: values.cluster_version,
                    idc: values.idc,
                    zone: values.zone,
                    flow_type: values.flow_type,
                    purpose: values.purpose,
                    arch: values.arch,
                    priority: parseInt(values.priority || '0', 10),
                    cluster_group: values.cluster_group,
                })
                toast.success('集群更新成功')
            } else {
                await RobustaAPI.createCluster({
                    name: values.name,
                    cluster_id: values.cluster_id,
                    description: values.description,
                    kube_config: values.kube_config,
                    prometheus_url: values.prometheus_url,
                    status: values.status,
                    cluster_version: values.cluster_version,
                    idc: values.idc,
                    zone: values.zone,
                    flow_type: values.flow_type,
                    purpose: values.purpose,
                    arch: values.arch,
                    priority: parseInt(values.priority || '0', 10),
                    cluster_group: values.cluster_group,
                })
                toast.success('集群创建成功')
            }
            onSuccess()
            onOpenChange(false)
        } catch (error) {
            console.error(error)
            toast.error(isEdit ? '更新失败' : '创建失败')
        } finally {
            setLoading(false)
        }
    }

    return (
        <Dialog open={open} onOpenChange={onOpenChange}>
            <DialogContent className="sm:max-w-[800px] max-h-[90vh] overflow-hidden flex flex-col p-0">
                <DialogHeader className="px-6 pt-6 pb-2 border-b">
                    <DialogTitle className="flex items-center gap-3 text-xl">
                        <div className="p-2.5 bg-blue-500/10 rounded-xl ring-1 ring-blue-500/20">
                            <Server className="w-5 h-5 text-blue-500" />
                        </div>
                        {isEdit ? '编辑集群' : '创建集群'}
                    </DialogTitle>
                    <DialogDescription className="text-sm mt-1.5 ml-1">
                        {isEdit
                            ? '修改集群的基本信息。'
                            : '添加一个新的Kubernetes集群进行监控。'}
                    </DialogDescription>
                </DialogHeader>
                <Form {...form}>
                    <form onSubmit={form.handleSubmit(onSubmit)} className="flex flex-col flex-1 overflow-hidden">
                        <div className="flex-1 overflow-y-auto px-6 py-4 space-y-4">
                            <div className="grid grid-cols-2 gap-4">
                                <FormField
                                    control={form.control}
                                    name="name"
                                    render={({ field }) => (
                                        <FormItem>
                                            <FormLabel>集群名称</FormLabel>
                                            <FormControl>
                                                <Input placeholder="prod-cluster-01" {...field} value={field.value ?? ""} disabled={isEdit} />
                                            </FormControl>
                                            <FormMessage />
                                        </FormItem>
                                    )}
                                />
                                <FormField
                                    control={form.control}
                                    name="cluster_id"
                                    render={({ field }) => (
                                        <FormItem>
                                            <FormLabel>Cluster ID</FormLabel>
                                            <FormControl>
                                                <Input placeholder="可选 (UUID)" {...field} value={field.value ?? ""} disabled={isEdit} />
                                            </FormControl>
                                            <FormMessage />
                                        </FormItem>
                                    )}
                                />
                            </div>

                            <FormField
                                control={form.control}
                                name="status"
                                render={({ field }) => (
                                    <FormItem>
                                        <FormLabel>状态</FormLabel>
                                        <Select
                                            onValueChange={field.onChange}
                                            defaultValue={field.value}
                                            value={field.value}
                                        >
                                            <FormControl>
                                                <SelectTrigger>
                                                    <SelectValue placeholder="选择状态" />
                                                </SelectTrigger>
                                            </FormControl>
                                            <SelectContent>
                                                <SelectItem value="active">活跃</SelectItem>
                                                <SelectItem value="maintenance">维护中</SelectItem>
                                                <SelectItem value="inactive">离线</SelectItem>
                                            </SelectContent>
                                        </Select>
                                        <FormMessage />
                                    </FormItem>
                                )}
                            />

                            <FormField
                                control={form.control}
                                name="prometheus_url"
                                render={({ field }) => (
                                    <FormItem>
                                        <FormLabel>Prometheus URL</FormLabel>
                                        <FormControl>
                                            <Input placeholder="http://prometheus-server:9090" {...field} value={field.value ?? ""} />
                                        </FormControl>
                                        <FormMessage />
                                    </FormItem>
                                )}
                            />

                            <div className="grid grid-cols-2 gap-4">
                                <FormField
                                    control={form.control}
                                    name="cluster_version"
                                    render={({ field }) => (
                                        <FormItem>
                                            <FormLabel className="flex items-center gap-2">
                                                集群版本
                                                <DictionaryCodeBadge code="cluster_version" />
                                            </FormLabel>
                                            <FormControl>
                                                <DictionarySelect
                                                    code="cluster_version"
                                                    value={field.value}
                                                    onValueChange={field.onChange}
                                                    placeholder="选择版本"
                                                />
                                            </FormControl>
                                            <FormMessage />
                                        </FormItem>
                                    )}
                                />
                                <FormField
                                    control={form.control}
                                    name="idc"
                                    render={({ field }) => (
                                        <FormItem>
                                            <FormLabel className="flex items-center gap-2">
                                                IDC
                                                <DictionaryCodeBadge code="idc" />
                                            </FormLabel>
                                            <FormControl>
                                                <DictionarySelect
                                                    code="idc"
                                                    value={field.value}
                                                    onValueChange={field.onChange}
                                                    placeholder="选择 IDC"
                                                />
                                            </FormControl>
                                            <FormMessage />
                                        </FormItem>
                                    )}
                                />
                            </div>

                            <div className="grid grid-cols-2 gap-4">
                                <FormField
                                    control={form.control}
                                    name="zone"
                                    render={({ field }) => (
                                        <FormItem>
                                            <FormLabel className="flex items-center gap-2">
                                                Zone
                                                <DictionaryCodeBadge code="zone" />
                                            </FormLabel>
                                            <FormControl>
                                                <DictionarySelect
                                                    code="zone"
                                                    value={field.value}
                                                    onValueChange={field.onChange}
                                                    placeholder="选择 Zone"
                                                />
                                            </FormControl>
                                            <FormMessage />
                                        </FormItem>
                                    )}
                                />
                                <FormField
                                    control={form.control}
                                    name="flow_type"
                                    render={({ field }) => (
                                        <FormItem>
                                            <FormLabel className="flex items-center gap-2">
                                                流量类型
                                                <DictionaryCodeBadge code="flow_type" />
                                            </FormLabel>
                                            <FormControl>
                                                <DictionarySelect
                                                    code="flow_type"
                                                    value={field.value}
                                                    onValueChange={field.onChange}
                                                    placeholder="选择类型"
                                                />
                                            </FormControl>
                                            <FormMessage />
                                        </FormItem>
                                    )}
                                />
                            </div>

                            <div className="grid grid-cols-2 gap-4">
                                <FormField
                                    control={form.control}
                                    name="purpose"
                                    render={({ field }) => (
                                        <FormItem>
                                            <FormLabel className="flex items-center gap-2">
                                                用途
                                                <DictionaryCodeBadge code="purpose" />
                                            </FormLabel>
                                            <FormControl>
                                                <DictionaryMultiSelect
                                                    code="purpose"
                                                    value={field.value}
                                                    onValueChange={field.onChange}
                                                    placeholder="选择用途（可多选）"
                                                />
                                            </FormControl>
                                            <FormMessage />
                                        </FormItem>
                                    )}
                                />
                                <FormField
                                    control={form.control}
                                    name="arch"
                                    render={({ field }) => (
                                        <FormItem>
                                            <FormLabel className="flex items-center gap-2">
                                                架构
                                                <DictionaryCodeBadge code="arch" />
                                            </FormLabel>
                                            <FormControl>
                                                <DictionarySelect
                                                    code="arch"
                                                    value={field.value}
                                                    onValueChange={field.onChange}
                                                    placeholder="选择架构"
                                                />
                                            </FormControl>
                                            <FormMessage />
                                        </FormItem>
                                    )}
                                />
                            </div>

                            <div className="grid grid-cols-2 gap-4">
                                <FormField
                                    control={form.control}
                                    name="priority"
                                    render={({ field }) => (
                                        <FormItem>
                                            <FormLabel>优先级</FormLabel>
                                            <FormControl>
                                                <Input
                                                    type="number"
                                                    placeholder="0"
                                                    {...field}
                                                    value={field.value || "0"}
                                                    onChange={(e) => field.onChange(e.target.value)}
                                                />
                                            </FormControl>
                                            <FormMessage />
                                        </FormItem>
                                    )}
                                />
                                <FormField
                                    control={form.control}
                                    name="cluster_group"
                                    render={({ field }) => (
                                        <FormItem>
                                            <FormLabel className="flex items-center gap-2">
                                                集群分组
                                                <DictionaryCodeBadge code="cluster_group" />
                                            </FormLabel>
                                            <FormControl>
                                                <DictionarySelect
                                                    code="cluster_group"
                                                    value={field.value}
                                                    onValueChange={field.onChange}
                                                    placeholder="选择分组"
                                                />
                                            </FormControl>
                                            <FormMessage />
                                        </FormItem>
                                    )}
                                />
                            </div>

                            <FormField
                                control={form.control}
                                name="description"
                                render={({ field }) => (
                                    <FormItem>
                                        <FormLabel>描述</FormLabel>
                                        <FormControl>
                                            <Textarea
                                                placeholder="集群用途描述..."
                                                className="resize-none h-20"
                                                {...field}
                                                value={field.value ?? ""}
                                            />
                                        </FormControl>
                                        <FormMessage />
                                    </FormItem>
                                )}
                            />

                            <FormField
                                control={form.control}
                                name="kube_config"
                                render={({ field }) => (
                                    <FormItem className="flex flex-col flex-1 min-h-0">
                                        <div className="flex justify-between items-center">
                                            <FormLabel>KubeConfig (YAML)</FormLabel>
                                            <div className="flex gap-2">
                                                <Button
                                                    type="button"
                                                    variant="outline"
                                                    size="sm"
                                                    className="h-7 text-xs"
                                                    onClick={() => {
                                                        formatKubeConfig(field.value || '')
                                                    }}
                                                >
                                                    格式化 / 校验
                                                </Button>
                                            </div>
                                        </div>
                                        <YamlEditor
                                            value={field.value ?? ""}
                                            onChange={(value) => field.onChange(value)}
                                            placeholder="# 粘贴 KubeConfig YAML 内容..."
                                            height="400px"
                                        />
                                        <FormDescription>
                                            请粘贴标准的 Kubernetes 配置文件 (Admin KubeConfig)。
                                        </FormDescription>
                                        <FormMessage />
                                    </FormItem>
                                )}
                            />
                        </div>

                        <DialogFooter className="px-6 py-4 border-t bg-background">
                            <Button type="submit" disabled={loading}>
                                {loading && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                                {isEdit ? '保存更改' : '创建'}
                            </Button>
                        </DialogFooter>
                    </form>
                </Form>
            </DialogContent>
        </Dialog>
    )
}

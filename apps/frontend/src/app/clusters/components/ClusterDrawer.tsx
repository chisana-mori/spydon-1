'use client'

import { useState, useEffect } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import * as z from 'zod'
import { toast } from 'sonner'
import { Loader2, Server, Settings2, LayoutDashboard, FileKey2 } from 'lucide-react'
import yaml from 'js-yaml'
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter'
import { oneDark } from 'react-syntax-highlighter/dist/esm/styles/prism'

import { useDictionaryPreload, CLUSTER_DICTIONARY_CODES } from '@/hooks/useDictionaryPreload'

import { Button } from '@/components/ui/button'
import { DictionarySelect } from '@/components/common/DictionarySelect'
import { DictionaryMultiSelect } from '@/components/common/DictionaryMultiSelect'
import { DictionaryCodeBadge } from '@/components/common/DictionaryCodeBadge'
import {
    Sheet,
    SheetContent,
    SheetDescription,
    SheetHeader,
    SheetTitle,
    SheetFooter,
} from '@/components/ui/sheet'
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
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Badge } from '@/components/ui/badge'
import RobustaAPI from '@/lib/api'
import { Cluster } from '@/types/api'
import { ScrollArea } from '@/components/ui/scroll-area'

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

interface ClusterDrawerProps {
    open: boolean
    onClose: () => void
    cluster?: Cluster | null // If present, it's edit mode
    onSuccess: () => void
}

export function ClusterDrawer({
    open,
    onClose,
    cluster,
    onSuccess,
}: ClusterDrawerProps) {
    const [loading, setLoading] = useState(false)
    const [activeTab, setActiveTab] = useState('basic')
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
            setActiveTab('basic')
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
            onClose()
        } catch (error) {
            console.error(error)
            toast.error(isEdit ? '更新失败' : '创建失败')
        } finally {
            setLoading(false)
        }
    }

    const statusValue = form.watch('status')
    const kubeConfigValue = form.watch('kube_config')
    const [apiServer, setApiServer] = useState<string>('')

    useEffect(() => {
        if (!kubeConfigValue) {
            setApiServer('')
            return
        }
        try {
            const parsed: any = yaml.load(kubeConfigValue)
            // standard kubeconfig structure: clusters: [{ cluster: { server: ... } }]
            const server = parsed?.clusters?.[0]?.cluster?.server
            setApiServer(server || '')
        } catch (e) {
            setApiServer('')
        }
    }, [kubeConfigValue])

    return (
        <Sheet open={open} onOpenChange={onClose}>
            <SheetContent className="sm:max-w-[700px] w-full flex flex-col h-full p-0 gap-0 bg-background/95 backdrop-blur-sm">
                <SheetHeader className="px-6 py-4 border-b bg-muted/20">
                    <div className="flex items-center gap-4">
                        <div className="p-3 bg-blue-500/10 rounded-xl ring-1 ring-blue-500/20">
                            <Server className="w-5 h-5 text-blue-500" />
                        </div>
                        <div className="space-y-1">
                            <SheetTitle className="flex items-center gap-3 text-xl">
                                {isEdit ? '编辑集群' : '新建集群'}
                                {isEdit ? (
                                    <Badge
                                        variant={statusValue === 'active' ? 'default' : 'secondary'}
                                        className={
                                            statusValue === 'active'
                                                ? "h-5 px-2 text-[10px] font-medium bg-green-500/10 text-green-600 border-green-500/20 shadow-none hover:bg-green-500/20"
                                                : statusValue === 'maintenance'
                                                    ? "h-5 px-2 text-[10px] font-medium bg-yellow-500/10 text-yellow-600 border-yellow-500/20 shadow-none hover:bg-yellow-500/20"
                                                    : "h-5 px-2 text-[10px] font-medium bg-red-500/10 text-red-500 border-red-500/20 shadow-none hover:bg-red-500/20"
                                        }
                                    >
                                        {statusValue === 'active' ? '活跃' : statusValue === 'maintenance' ? '维护中' : '离线'}
                                    </Badge>
                                ) : (
                                    <Badge className="h-5 px-2 text-[10px] font-medium bg-blue-500/10 text-blue-600 hover:bg-blue-500/20 border-blue-500/20 shadow-none">
                                        NEW
                                    </Badge>
                                )}
                            </SheetTitle>
                            <SheetDescription className="text-xs">
                                {isEdit
                                    ? '修改集群的基本信息和配置。'
                                    : '添加一个新的 Kubernetes 集群进行监控。'}
                            </SheetDescription>
                        </div>
                    </div>
                </SheetHeader>

                <div className="flex-1 overflow-hidden flex flex-col">
                    <Form {...form}>
                        <form
                            id="cluster-form"
                            onSubmit={form.handleSubmit(onSubmit)}
                            className="h-full flex flex-col"
                        >
                            <Tabs value={activeTab} onValueChange={setActiveTab} className="flex-1 flex flex-col overflow-hidden">
                                <div className="px-6 py-2 border-b bg-background">
                                    <TabsList className="grid w-full grid-cols-2 bg-muted/50">
                                        <TabsTrigger value="basic" className="data-[state=active]:bg-background data-[state=active]:shadow-sm">
                                            <LayoutDashboard className="w-4 h-4 mr-2" />
                                            基本信息
                                        </TabsTrigger>
                                        <TabsTrigger value="kubeconfig" className="data-[state=active]:bg-background data-[state=active]:shadow-sm">
                                            <FileKey2 className="w-4 h-4 mr-2" />
                                            KubeConfig
                                        </TabsTrigger>
                                    </TabsList>
                                </div>

                                <TabsContent value="basic" className="flex-1 overflow-y-auto p-6 space-y-6 mt-0">
                                    {/* Basic Info Section */}
                                    <div className="space-y-4">
                                        <div className="grid grid-cols-2 gap-4">
                                            <FormField
                                                control={form.control}
                                                name="name"
                                                render={({ field }) => (
                                                    <FormItem>
                                                        <FormLabel className="text-sm font-medium">集群名称</FormLabel>
                                                        <FormControl>
                                                            <Input
                                                                placeholder="prod-cluster-01"
                                                                {...field}
                                                                value={field.value ?? ""}
                                                                disabled={isEdit}
                                                                className="h-10"
                                                            />
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
                                                        <FormLabel className="text-sm font-medium">Cluster ID</FormLabel>
                                                        <FormControl>
                                                            <Input
                                                                placeholder="可选 (UUID)"
                                                                {...field}
                                                                value={field.value ?? ""}
                                                                disabled={isEdit}
                                                                className="h-10 font-mono text-sm"
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
                                                name="status"
                                                render={({ field }) => (
                                                    <FormItem>
                                                        <FormLabel className="text-sm font-medium">状态</FormLabel>
                                                        <Select
                                                            onValueChange={field.onChange}
                                                            defaultValue={field.value}
                                                            value={field.value}
                                                        >
                                                            <FormControl>
                                                                <SelectTrigger className="h-10">
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
                                                        <FormLabel className="text-sm font-medium">Prometheus URL</FormLabel>
                                                        <FormControl>
                                                            <Input
                                                                placeholder="http://prometheus-server:9090"
                                                                {...field}
                                                                value={field.value ?? ""}
                                                                className="h-10"
                                                            />
                                                        </FormControl>
                                                        <FormMessage />
                                                    </FormItem>
                                                )}
                                            />
                                        </div>
                                    </div>

                                    <div className="h-px bg-border/50" />

                                    {/* Navy Fields Section */}
                                    <div className="space-y-4">
                                        <h3 className="text-sm font-semibold text-muted-foreground flex items-center gap-2">
                                            <Settings2 className="w-4 h-4" />
                                            集群属性
                                        </h3>

                                        <div className="grid grid-cols-2 gap-4">
                                            <FormField
                                                control={form.control}
                                                name="cluster_version"
                                                render={({ field }) => (
                                                    <FormItem>
                                                        <FormLabel className="flex items-center gap-2 text-sm">
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
                                                        <FormLabel className="flex items-center gap-2 text-sm">
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
                                                        <FormLabel className="flex items-center gap-2 text-sm">
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
                                                        <FormLabel className="flex items-center gap-2 text-sm">
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
                                                        <FormLabel className="flex items-center gap-2 text-sm">
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
                                                        <FormLabel className="flex items-center gap-2 text-sm">
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
                                                        <FormLabel className="text-sm font-medium">优先级</FormLabel>
                                                        <FormControl>
                                                            <Input
                                                                type="number"
                                                                placeholder="0"
                                                                {...field}
                                                                value={field.value || "0"}
                                                                onChange={(e) => field.onChange(e.target.value)}
                                                                className="h-10"
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
                                                        <FormLabel className="flex items-center gap-2 text-sm">
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
                                    </div>

                                    <div className="h-px bg-border/50" />

                                    {/* Description */}
                                    <FormField
                                        control={form.control}
                                        name="description"
                                        render={({ field }) => (
                                            <FormItem>
                                                <FormLabel className="text-sm font-medium">描述</FormLabel>
                                                <FormControl>
                                                    <Textarea
                                                        placeholder="集群用途描述..."
                                                        className="resize-none min-h-[80px]"
                                                        {...field}
                                                        value={field.value ?? ""}
                                                    />
                                                </FormControl>
                                                <FormMessage />
                                            </FormItem>
                                        )}
                                    />
                                </TabsContent>

                                <TabsContent value="kubeconfig" className="flex-1 overflow-y-auto p-6 mt-0">
                                    <FormField
                                        control={form.control}
                                        name="kube_config"
                                        render={({ field }) => (
                                            <FormItem className="flex flex-col h-full">
                                                {apiServer && (
                                                    <div className="mb-4 p-3 bg-blue-50/50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg flex items-center justify-between">
                                                        <div className="flex items-center gap-2">
                                                            <div className="h-2 w-2 rounded-full bg-blue-500 animate-pulse" />
                                                            <span className="text-xs font-medium text-blue-700 dark:text-blue-300">API Server Detected</span>
                                                        </div>
                                                        <code className="text-xs font-mono bg-white dark:bg-black/20 px-2 py-1 rounded border border-blue-100 dark:border-blue-800 text-blue-800 dark:text-blue-200">
                                                            {apiServer}
                                                        </code>
                                                    </div>
                                                )}

                                                <div className="flex justify-between items-center mb-2">
                                                    <FormLabel className="text-sm font-medium">KubeConfig (YAML)</FormLabel>
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
                                                <div className="flex-1 flex flex-col gap-4 min-h-[400px]">
                                                    <Textarea
                                                        placeholder="请粘贴标准的 Kubernetes 配置文件 (Admin KubeConfig)..."
                                                        className="font-mono text-xs min-h-[200px]"
                                                        {...field}
                                                        value={field.value ?? ""}
                                                    />
                                                    <div className="flex-1 border rounded-md overflow-hidden bg-[#282c34]">
                                                        <ScrollArea className="h-[300px] w-full">
                                                            <SyntaxHighlighter
                                                                language="yaml"
                                                                style={oneDark}
                                                                customStyle={{
                                                                    margin: 0,
                                                                    padding: '1rem',
                                                                    background: 'transparent',
                                                                    fontSize: '12px',
                                                                }}
                                                                showLineNumbers={true}
                                                                wrapLines={true}
                                                            >
                                                                {field.value || '# Preview will appear here'}
                                                            </SyntaxHighlighter>
                                                        </ScrollArea>
                                                    </div>
                                                </div>
                                                <FormDescription className="mt-2">
                                                    请粘贴标准的 Kubernetes 配置文件 (Admin KubeConfig)。
                                                </FormDescription>
                                                <FormMessage />
                                            </FormItem>
                                        )}
                                    />
                                </TabsContent>
                            </Tabs>
                        </form>
                    </Form>
                </div>

                <SheetFooter className="px-6 py-4 border-t bg-muted/20 sm:space-x-4">
                    <Button variant="outline" onClick={onClose} disabled={loading} className="w-24">
                        取消
                    </Button>
                    <Button
                        type="submit"
                        form="cluster-form"
                        disabled={loading}
                        className="w-24"
                    >
                        {loading && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                        {isEdit ? '保存' : '创建'}
                    </Button>
                </SheetFooter>
            </SheetContent>
        </Sheet>
    )
}

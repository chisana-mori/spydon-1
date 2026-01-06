'use client'

import { useState, useEffect } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import * as z from 'zod'
import { toast } from 'sonner'
import { Loader2, Server, Cpu, Info } from 'lucide-react'

import { Button } from '@/components/ui/button'
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
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { ScrollArea } from "@/components/ui/scroll-area"
import RobustaAPI from '@/lib/api'
import { NavyDevice, DeviceFeatureDetails } from '@/types/navy'

const formSchema = z.object({
    role: z.string().max(255, '角色不能超过255个字符').optional(),
    group: z.string().max(255, '分组不能超过255个字符').optional(),
})

type FormValues = z.infer<typeof formSchema>

interface DeviceDialogProps {
    open: boolean
    onOpenChange: (open: boolean) => void
    device: NavyDevice | null
    onSuccess: () => void
    mode?: 'view' | 'edit'
}

export function DeviceDialog({
    open,
    onOpenChange,
    device,
    onSuccess,
    mode = 'view',
}: DeviceDialogProps) {
    const [loading, setLoading] = useState(false)
    const [featureDetails, setFeatureDetails] = useState<DeviceFeatureDetails | null>(null)
    const [loadingFeatures, setLoadingFeatures] = useState(false)

    const form = useForm<FormValues>({
        resolver: zodResolver(formSchema),
        defaultValues: {
            role: '',
            group: '',
        },
    })

    useEffect(() => {
        if (open && device) {
            form.reset({
                role: device.role || '',
                group: device.group || '',
            })

            // If view mode, also fetch feature details (labels/taints)
            if (mode === 'view' && device.ci_code) {
                fetchFeatures(device.ci_code)
            }
        }
    }, [open, device, form, mode])

    const fetchFeatures = async (ciCode: string) => {
        setLoadingFeatures(true)
        try {
            const data = await RobustaAPI.getBatchDeviceFeatures([ciCode])
            setFeatureDetails(data)
        } catch (err) {
            console.error(err)
        } finally {
            setLoadingFeatures(false)
        }
    }

    async function onSubmit(values: FormValues) {
        if (!device) return
        setLoading(true)
        try {
            if (values.role !== device.role) {
                await RobustaAPI.updateNavyDeviceRole(device.id, values.role || '')
            }
            if (values.group !== device.group) {
                await RobustaAPI.updateNavyDeviceGroup(device.id, values.group || '')
            }
            toast.success('设备更新成功')
            onSuccess()
            onOpenChange(false)
        } catch (error) {
            console.error(error)
            toast.error('更新失败')
        } finally {
            setLoading(false)
        }
    }

    if (!device) return null

    return (
        <Dialog open={open} onOpenChange={onOpenChange}>
            <DialogContent className="sm:max-w-[700px] max-h-[85vh] flex flex-col p-0">
                <DialogHeader className="px-6 pt-6 pb-2">
                    <div className="flex items-center gap-2">
                        <Server className="h-5 w-5 text-primary" />
                        <DialogTitle>设备详情: {device.ci_code}</DialogTitle>
                    </div>
                    <DialogDescription>
                        {device.ip} | {device.idc} {device.room}
                    </DialogDescription>
                </DialogHeader>

                <Tabs defaultValue="basic" className="flex-1 flex flex-col overflow-hidden">
                    <div className="px-6 border-b">
                        <TabsList className="w-full justify-start h-auto p-0 bg-transparent gap-6">
                            <TabsTrigger value="basic" className="rounded-none border-b-2 border-transparent data-[state=active]:border-primary data-[state=active]:bg-transparent px-0 py-2">基本信息</TabsTrigger>
                            <TabsTrigger value="specs" className="rounded-none border-b-2 border-transparent data-[state=active]:border-primary data-[state=active]:bg-transparent px-0 py-2">硬件/软件</TabsTrigger>
                            <TabsTrigger value="features" className="rounded-none border-b-2 border-transparent data-[state=active]:border-primary data-[state=active]:bg-transparent px-0 py-2">K8s 特性</TabsTrigger>
                        </TabsList>
                    </div>

                    <ScrollArea className="flex-1 px-6 py-4">
                        <TabsContent value="basic" className="mt-0 space-y-6">
                            <div className="grid grid-cols-2 gap-x-12 gap-y-6">
                                <div>
                                    <h4 className="text-sm font-medium text-muted-foreground mb-1">设备编码</h4>
                                    <p className="text-sm">{device.ci_code}</p>
                                </div>
                                <div>
                                    <h4 className="text-sm font-medium text-muted-foreground mb-1">IP 地址</h4>
                                    <p className="text-sm font-mono">{device.ip}</p>
                                </div>
                                <div>
                                    <h4 className="text-sm font-medium text-muted-foreground mb-1">CPU 架构</h4>
                                    <p className="text-sm">{device.arch_type}</p>
                                </div>
                                <div>
                                    <h4 className="text-sm font-medium text-muted-foreground mb-1">所属集群</h4>
                                    <p className="text-sm">{device.cluster || '-'}</p>
                                </div>
                                <div>
                                    <h4 className="text-sm font-medium text-muted-foreground mb-1">机房/机柜</h4>
                                    <p className="text-sm">{device.idc} {device.room} / {device.cabinet} ({device.cabinet_no})</p>
                                </div>
                                <div>
                                    <h4 className="text-sm font-medium text-muted-foreground mb-1">基础设施类型</h4>
                                    <p className="text-sm">{device.infra_type}</p>
                                </div>
                            </div>

                            <div className="pt-4 border-t">
                                {mode === 'edit' ? (
                                    <Form {...form}>
                                        <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
                                            <div className="grid grid-cols-2 gap-4">
                                                <FormField
                                                    control={form.control}
                                                    name="role"
                                                    render={({ field }) => (
                                                        <FormItem>
                                                            <FormLabel>设备角色</FormLabel>
                                                            <FormControl>
                                                                <Input placeholder="如: worker, master" {...field} />
                                                            </FormControl>
                                                            <FormMessage />
                                                        </FormItem>
                                                    )}
                                                />
                                                <FormField
                                                    control={form.control}
                                                    name="group"
                                                    render={({ field }) => (
                                                        <FormItem>
                                                            <FormLabel>机器分组/用途</FormLabel>
                                                            <FormControl>
                                                                <Input placeholder="如: 业务组, 工具组" {...field} />
                                                            </FormControl>
                                                            <FormMessage />
                                                        </FormItem>
                                                    )}
                                                />
                                            </div>
                                            <div className="flex justify-end gap-2 pt-2">
                                                <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>取消</Button>
                                                <Button type="submit" disabled={loading}>
                                                    {loading && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                                                    保存更改
                                                </Button>
                                            </div>
                                        </form>
                                    </Form>
                                ) : (
                                    <div className="grid grid-cols-2 gap-x-12 gap-y-6">
                                        <div>
                                            <h4 className="text-sm font-medium text-muted-foreground mb-1">当前角色</h4>
                                            <Badge variant="outline" className="text-primary border-primary/20">{device.role || '未分配'}</Badge>
                                        </div>
                                        <div>
                                            <h4 className="text-sm font-medium text-muted-foreground mb-1">机器分组</h4>
                                            <Badge variant="outline" className="text-orange-600 border-orange-200 bg-orange-50">{device.group || '通用'}</Badge>
                                        </div>
                                    </div>
                                )}
                            </div>
                        </TabsContent>

                        <TabsContent value="specs" className="mt-0 space-y-6">
                            <div className="grid grid-cols-2 gap-x-12 gap-y-6">
                                <div>
                                    <h4 className="text-sm font-medium text-muted-foreground mb-1">操作系统</h4>
                                    <p className="text-sm">{device.os_name} ({device.os_issue})</p>
                                </div>
                                <div>
                                    <h4 className="text-sm font-medium text-muted-foreground mb-1">内核版本</h4>
                                    <p className="text-sm font-mono text-xs">{device.os_kernel}</p>
                                </div>
                                <div>
                                    <h4 className="text-sm font-medium text-muted-foreground mb-1">CPU / 内存</h4>
                                    <p className="text-sm">{device.cpu} Cores / {device.memory} GB</p>
                                </div>
                                <div>
                                    <h4 className="text-sm font-medium text-muted-foreground mb-1">厂商 / 型号</h4>
                                    <p className="text-sm">{device.company} / {device.model}</p>
                                </div>
                                <div>
                                    <h4 className="text-sm font-medium text-muted-foreground mb-1">磁盘数量</h4>
                                    <p className="text-sm">{device.disk_count} 个</p>
                                </div>
                                <div>
                                    <h4 className="text-sm font-medium text-muted-foreground mb-1">网卡速率</h4>
                                    <p className="text-sm">{device.network_speed} Mbps</p>
                                </div>
                                <div className="col-span-2">
                                    <h4 className="text-sm font-medium text-muted-foreground mb-1">磁盘详情</h4>
                                    <p className="text-sm bg-muted p-2 rounded font-mono text-xs">{device.disk_detail || '无数据'}</p>
                                </div>
                            </div>
                        </TabsContent>

                        <TabsContent value="features" className="mt-0 space-y-4">
                            {loadingFeatures ? (
                                <div className="flex items-center justify-center py-12">
                                    <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
                                </div>
                            ) : featureDetails ? (
                                <div className="space-y-6">
                                    <div>
                                        <h4 className="text-sm font-medium mb-3 flex items-center">
                                            <Cpu className="h-4 w-4 mr-2" />
                                            Node Labels
                                        </h4>
                                        <div className="flex flex-wrap gap-2">
                                            {featureDetails.labels.length > 0 ? (
                                                featureDetails.labels.map((l, i) => (
                                                    <Badge key={i} variant="secondary" className="font-normal text-[11px] h-auto py-0.5">
                                                        {l.key}: {l.value}
                                                    </Badge>
                                                ))
                                            ) : (
                                                <p className="text-sm text-muted-foreground italic">无 Label 数据</p>
                                            )}
                                        </div>
                                    </div>
                                    <div>
                                        <h4 className="text-sm font-medium mb-3 flex items-center text-orange-600">
                                            <Info className="h-4 w-4 mr-2" />
                                            Node Taints
                                        </h4>
                                        <div className="flex flex-wrap gap-2">
                                            {featureDetails.taints.length > 0 ? (
                                                featureDetails.taints.map((t, i) => (
                                                    <Badge key={i} variant="outline" className="text-[11px] border-orange-200 text-orange-600 bg-orange-50 h-auto py-0.5">
                                                        {t.key}={t.value}:{t.effect}
                                                    </Badge>
                                                ))
                                            ) : (
                                                <p className="text-sm text-muted-foreground italic">无 Taint 数据</p>
                                            )}
                                        </div>
                                    </div>
                                </div>
                            ) : (
                                <div className="flex flex-col items-center justify-center py-12 text-muted-foreground">
                                    <Info className="h-8 w-8 mb-2 opacity-20" />
                                    <p>该设备暂无关联的 K8s 节点数据</p>
                                </div>
                            )}
                        </TabsContent>
                    </ScrollArea>
                </Tabs>

                {mode === 'view' && (
                    <DialogFooter className="px-6 py-4 border-t bg-muted/30">
                        <Button variant="ghost" onClick={() => onOpenChange(false)}>关闭</Button>
                    </DialogFooter>
                )}
            </DialogContent>
        </Dialog>
    )
}

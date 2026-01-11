'use client'

import { useState } from 'react'
import {
    Sheet,
    SheetContent,
    SheetDescription,
    SheetHeader,
    SheetTitle,
} from '@/components/ui/sheet'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Separator } from '@/components/ui/separator'
import {
    AlertDialog,
    AlertDialogAction,
    AlertDialogCancel,
    AlertDialogContent,
    AlertDialogDescription,
    AlertDialogFooter,
    AlertDialogHeader,
    AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import {
    Server,
    Network,
    Globe,
    Activity,
    Eye,
    EyeOff,
    Trash2,
    CheckCircle,
    AlertCircle,
    X,
    ExternalLink,
} from 'lucide-react'
import type { F5Info } from '@/types/f5'
import RobustaAPI from '@/lib/api'
import { toast } from 'sonner'
import { cn } from '@/lib/utils'

interface F5DetailSheetProps {
    f5: F5Info | null
    open: boolean
    onOpenChange: (open: boolean) => void
    onUpdate: () => void
}

export function F5DetailSheet({ f5, open, onOpenChange, onUpdate }: F5DetailSheetProps) {
    const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)
    const [isDeleting, setIsDeleting] = useState(false)
    const [isTogglingIgnore, setIsTogglingIgnore] = useState(false)

    if (!f5) return null

    // Parse pool members
    const poolMembers = f5.pool_members
        ? f5.pool_members.split(',').map(m => m.trim()).filter(Boolean)
        : []

    // Parse domains
    const domains = f5.domains
        ? f5.domains.split(',').map(d => d.trim()).filter(Boolean)
        : []

    // Get status badge
    const getStatusBadge = (status: string) => {
        const lower = status?.toLowerCase() || ''
        if (lower === 'active' || lower === 'running' || lower === 'online') {
            return <Badge variant="outline" className="gap-1.5 bg-emerald-50/80 text-emerald-700 border-emerald-200/50 dark:bg-emerald-950/30 dark:text-emerald-400 dark:border-emerald-800/50 font-medium shadow-sm"><CheckCircle className="h-3 w-3" />{status}</Badge>
        }
        if (lower === 'inactive' || lower === 'offline' || lower === 'stopped') {
            return <Badge variant="outline" className="gap-1.5 bg-rose-50/80 text-rose-700 border-rose-200/50 dark:bg-rose-950/30 dark:text-rose-400 dark:border-rose-800/50 font-medium shadow-sm"><AlertCircle className="h-3 w-3" />{status}</Badge>
        }
        if (lower === 'maintenance') {
            return <Badge variant="outline" className="gap-1.5 bg-amber-50/80 text-amber-700 border-amber-200/50 dark:bg-amber-950/30 dark:text-amber-400 dark:border-amber-800/50 font-medium shadow-sm">{status}</Badge>
        }
        return <Badge variant="outline" className="bg-slate-50/80 text-slate-600 border-slate-200/50 dark:bg-slate-900/50 dark:text-slate-400 dark:border-slate-700/50 font-medium">{status}</Badge>
    }

    // Toggle ignored status
    const handleToggleIgnore = async () => {
        setIsTogglingIgnore(true)
        try {
            await RobustaAPI.updateF5Info(f5.id, {
                ...f5,
                ignored: !f5.ignored,
            })
            toast.success(f5.ignored ? '已取消忽略' : '已设为忽略')
            onUpdate()
            onOpenChange(false)
        } catch (error) {
            console.error('Failed to toggle ignore:', error)
            toast.error('操作失败')
        } finally {
            setIsTogglingIgnore(false)
        }
    }

    // Delete F5
    const handleDelete = async () => {
        setIsDeleting(true)
        try {
            await RobustaAPI.deleteF5Info(f5.id)
            toast.success('删除成功')
            setDeleteDialogOpen(false)
            onUpdate()
            onOpenChange(false)
        } catch (error) {
            console.error('Failed to delete F5:', error)
            toast.error('删除失败')
        } finally {
            setIsDeleting(false)
        }
    }

    return (
        <>
            <Sheet open={open} onOpenChange={onOpenChange}>
                <SheetContent className="w-full sm:max-w-2xl overflow-y-auto">
                    <SheetHeader className="space-y-3 pb-6 border-b bg-gradient-to-br from-blue-50 to-indigo-50 dark:from-blue-950/30 dark:to-indigo-950/30 -mx-6 px-6 pt-6">
                        <div className="flex items-center gap-4">
                            <div className="p-3 bg-white dark:bg-slate-900 rounded-xl ring-1 ring-slate-200/50 dark:ring-slate-700/50 shadow-sm">
                                <Server className="w-6 h-6 text-blue-600 dark:text-blue-400" />
                            </div>
                            <div className="space-y-1">
                                <SheetTitle className="text-xl font-bold bg-clip-text text-transparent bg-gradient-to-r from-slate-900 to-slate-700 dark:from-slate-100 dark:to-slate-300">{f5.name}</SheetTitle>
                                <SheetDescription className="text-sm flex items-center gap-2">
                                    F5 负载均衡器详细信息
                                    <Badge variant="outline" className="font-mono text-xs bg-slate-100/80 text-slate-600 border-slate-200/50 dark:bg-slate-800/50 dark:text-slate-400 dark:border-slate-700/50">{f5.id}</Badge>
                                </SheetDescription>
                            </div>
                        </div>
                    </SheetHeader>

                    <div className="mt-6 space-y-6">
                        {/* Basic Info */}
                        <Card className="border-slate-200/50 dark:border-slate-700/50 shadow-sm">
                            <CardHeader className="pb-3">
                                <CardTitle className="text-base flex items-center gap-2">
                                    <div className="p-1.5 rounded-lg bg-blue-50/80 text-blue-600 dark:bg-blue-950/30 dark:text-blue-400">
                                        <Network className="h-3.5 w-3.5" />
                                    </div>
                                    基本信息
                                </CardTitle>
                            </CardHeader>
                            <CardContent className="space-y-3">
                                <div className="grid grid-cols-2 gap-4">
                                    <div className="space-y-1">
                                        <div className="text-xs text-muted-foreground uppercase tracking-wide">名称</div>
                                        <div className="font-medium">{f5.name}</div>
                                    </div>
                                    <div className="space-y-1">
                                        <div className="text-xs text-muted-foreground uppercase tracking-wide">VIP</div>
                                        <div className="font-mono font-medium text-sm">{f5.vip}</div>
                                    </div>
                                    <div className="space-y-1">
                                        <div className="text-xs text-muted-foreground uppercase tracking-wide">端口</div>
                                        <div className="font-medium">{f5.port}</div>
                                    </div>
                                    <div className="space-y-1">
                                        <div className="text-xs text-muted-foreground uppercase tracking-wide">AppID</div>
                                        <div className="font-medium">{f5.appid || '-'}</div>
                                    </div>
                                    <div className="space-y-1">
                                        <div className="text-xs text-muted-foreground uppercase tracking-wide">实例组</div>
                                        <div className="font-medium">{f5.instance_group || '-'}</div>
                                    </div>
                                    <div className="space-y-1">
                                        <div className="text-xs text-muted-foreground uppercase tracking-wide">状态</div>
                                        <div>{f5.status ? getStatusBadge(f5.status) : '-'}</div>
                                    </div>
                                </div>
                            </CardContent>
                        </Card>

                        {/* Pool Info */}
                        <Card className="border-slate-200/50 dark:border-slate-700/50 shadow-sm">
                            <CardHeader className="pb-3">
                                <CardTitle className="text-base flex items-center gap-2">
                                    <div className="p-1.5 rounded-lg bg-emerald-50/80 text-emerald-600 dark:bg-emerald-950/30 dark:text-emerald-400">
                                        <Activity className="h-3.5 w-3.5" />
                                    </div>
                                    Pool 信息
                                </CardTitle>
                            </CardHeader>
                            <CardContent className="space-y-3">
                                <div className="grid grid-cols-2 gap-4">
                                    <div className="space-y-1">
                                        <div className="text-xs text-muted-foreground uppercase tracking-wide">Pool 名称</div>
                                        <div className="font-medium">{f5.pool_name || '-'}</div>
                                    </div>
                                    <div className="space-y-1">
                                        <div className="text-xs text-muted-foreground uppercase tracking-wide">Pool 状态</div>
                                        <div>{f5.pool_status ? getStatusBadge(f5.pool_status) : '-'}</div>
                                    </div>
                                </div>

                                {poolMembers.length > 0 && (
                                    <div className="space-y-2">
                                        <div className="text-xs text-muted-foreground uppercase tracking-wide">Pool 成员</div>
                                        <div className="space-y-1.5">
                                            {poolMembers.map((member, idx) => (
                                                <div key={idx} className="text-sm font-mono bg-slate-50/80 px-3 py-2 rounded-lg border border-slate-200/50 dark:bg-slate-900/50 dark:border-slate-700/50">
                                                    {member}
                                                </div>
                                            ))}
                                        </div>
                                    </div>
                                )}
                            </CardContent>
                        </Card>

                        {/* Cluster Info */}
                        {f5.cluster_name && (
                            <Card className="border-slate-200/50 dark:border-slate-700/50 shadow-sm">
                                <CardHeader className="pb-3">
                                    <CardTitle className="text-base flex items-center gap-2">
                                        <div className="p-1.5 rounded-lg bg-purple-50/80 text-purple-600 dark:bg-purple-950/30 dark:text-purple-400">
                                            <Server className="h-3.5 w-3.5" />
                                        </div>
                                        关联集群
                                    </CardTitle>
                                </CardHeader>
                                <CardContent>
                                    <div className="flex items-center justify-between">
                                        <div className="font-medium">{f5.cluster_name}</div>
                                        <Badge variant="outline" className="border-purple-200/50 bg-purple-50/80 text-purple-700 dark:bg-purple-950/30 dark:text-purple-400 dark:border-purple-800/50 font-medium">K8s Cluster</Badge>
                                    </div>
                                </CardContent>
                            </Card>
                        )}

                        {/* Domains */}
                        {domains.length > 0 && (
                            <Card className="border-slate-200/50 dark:border-slate-700/50 shadow-sm">
                                <CardHeader className="pb-3">
                                    <CardTitle className="text-base flex items-center gap-2">
                                        <div className="p-1.5 rounded-lg bg-cyan-50/80 text-cyan-600 dark:bg-cyan-950/30 dark:text-cyan-400">
                                            <Globe className="h-3.5 w-3.5" />
                                        </div>
                                        域名列表
                                    </CardTitle>
                                </CardHeader>
                                <CardContent>
                                    <div className="space-y-2">
                                        {domains.map((domain, idx) => (
                                            <div key={idx} className="flex items-center gap-2 text-sm p-2 rounded-lg bg-slate-50/50 border border-slate-200/30 dark:bg-slate-900/30 dark:border-slate-700/30">
                                                <ExternalLink className="h-3 w-3 text-muted-foreground shrink-0" />
                                                <span className="font-mono text-xs">{domain}</span>
                                            </div>
                                        ))}
                                    </div>
                                </CardContent>
                            </Card>
                        )}

                        {/* Grafana */}
                        {f5.grafana_params && (
                            <Card className="border-slate-200/50 dark:border-slate-700/50 shadow-sm">
                                <CardHeader className="pb-3">
                                    <CardTitle className="text-base flex items-center gap-2">
                                        <div className="p-1.5 rounded-lg bg-orange-50/80 text-orange-600 dark:bg-orange-950/30 dark:text-orange-400">
                                            <Activity className="h-3.5 w-3.5" />
                                        </div>
                                        Grafana 参数
                                    </CardTitle>
                                </CardHeader>
                                <CardContent>
                                    <div className="text-sm font-mono bg-slate-50/80 px-3 py-2 rounded-lg border border-slate-200/50 dark:bg-slate-900/50 dark:border-slate-700/50 break-all">
                                        {f5.grafana_params}
                                    </div>
                                </CardContent>
                            </Card>
                        )}

                        {/* Metadata */}
                        <Card className="border-slate-200/50 dark:border-slate-700/50 shadow-sm">
                            <CardHeader className="pb-3">
                                <CardTitle className="text-base flex items-center gap-2">
                                    <div className="p-1.5 rounded-lg bg-slate-100/80 text-slate-600 dark:bg-slate-800/50 dark:text-slate-400">
                                        <Activity className="h-3.5 w-3.5" />
                                    </div>
                                    元数据
                                </CardTitle>
                            </CardHeader>
                            <CardContent className="space-y-3 text-sm">
                                <div className="flex justify-between items-center">
                                    <span className="text-xs text-muted-foreground uppercase tracking-wide">创建时间</span>
                                    <span className="font-mono text-xs">{f5.created_at || '-'}</span>
                                </div>
                                <Separator className="bg-slate-200/50 dark:bg-slate-700/50" />
                                <div className="flex justify-between items-center">
                                    <span className="text-xs text-muted-foreground uppercase tracking-wide">更新时间</span>
                                    <span className="font-mono text-xs">{f5.updated_at || '-'}</span>
                                </div>
                                <Separator className="bg-slate-200/50 dark:bg-slate-700/50" />
                                <div className="flex justify-between items-center">
                                    <span className="text-xs text-muted-foreground uppercase tracking-wide">忽略状态</span>
                                    {f5.ignored ? (
                                        <Badge variant="outline" className="text-amber-700 bg-amber-50/80 border-amber-200/50 dark:bg-amber-950/30 dark:text-amber-400 dark:border-amber-800/50 font-medium">已忽略</Badge>
                                    ) : (
                                        <Badge variant="outline" className="text-emerald-700 bg-emerald-50/80 border-emerald-200/50 dark:bg-emerald-950/30 dark:text-emerald-400 dark:border-emerald-800/50 font-medium">正常</Badge>
                                    )}
                                </div>
                            </CardContent>
                        </Card>

                        {/* Actions */}
                        <div className="flex gap-2 pt-4">
                            <Button
                                variant={f5.ignored ? "default" : "outline"}
                                onClick={handleToggleIgnore}
                                disabled={isTogglingIgnore}
                                className={cn(
                                    "flex-1 transition-all",
                                    f5.ignored
                                        ? "bg-emerald-600 hover:bg-emerald-700 text-white shadow-sm hover:shadow-md"
                                        : "hover:bg-slate-50/80 border-slate-200/50 dark:hover:bg-slate-800/30 dark:border-slate-700/50"
                                )}
                            >
                                {f5.ignored ? (
                                    <>
                                        <Eye className="h-4 w-4 mr-2" />
                                        取消忽略
                                    </>
                                ) : (
                                    <>
                                        <EyeOff className="h-4 w-4 mr-2" />
                                        设为忽略
                                    </>
                                )}
                            </Button>
                            <Button
                                variant="outline"
                                onClick={() => setDeleteDialogOpen(true)}
                                className="flex-1 hover:bg-rose-50/80 hover:text-rose-700 hover:border-rose-200/50 dark:hover:bg-rose-950/30 dark:hover:text-rose-400 dark:hover:border-rose-800/50 transition-all"
                            >
                                <Trash2 className="h-4 w-4 mr-2" />
                                删除
                            </Button>
                        </div>
                    </div>
                </SheetContent>
            </Sheet>

            {/* Delete Confirmation Dialog */}
            <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
                <AlertDialogContent className="border-slate-200/50 dark:border-slate-700/50">
                    <AlertDialogHeader>
                        <AlertDialogTitle className="text-foreground">确认删除</AlertDialogTitle>
                        <AlertDialogDescription className="text-muted-foreground">
                            确定要删除 F5 <span className="font-mono font-medium text-foreground">"{f5.name}"</span> 吗？此操作不可撤销。
                        </AlertDialogDescription>
                    </AlertDialogHeader>
                    <AlertDialogFooter>
                        <AlertDialogCancel className="hover:bg-slate-50/80 dark:hover:bg-slate-800/30">取消</AlertDialogCancel>
                        <AlertDialogAction
                            onClick={handleDelete}
                            disabled={isDeleting}
                            className="bg-rose-600 hover:bg-rose-700 text-white shadow-sm hover:shadow-md"
                        >
                            {isDeleting ? '删除中...' : '确认删除'}
                        </AlertDialogAction>
                    </AlertDialogFooter>
                </AlertDialogContent>
            </AlertDialog>
        </>
    )
}

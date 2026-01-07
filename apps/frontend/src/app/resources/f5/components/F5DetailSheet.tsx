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
            return <Badge variant="default" className="gap-1 bg-green-600"><CheckCircle className="h-3 w-3" />{status}</Badge>
        }
        if (lower === 'inactive' || lower === 'offline' || lower === 'stopped') {
            return <Badge variant="destructive" className="gap-1"><AlertCircle className="h-3 w-3" />{status}</Badge>
        }
        return <Badge variant="secondary">{status}</Badge>
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
                    <SheetHeader className="space-y-3 pb-6 border-b">
                        <div className="flex items-center gap-4">
                            <div className="p-3 bg-blue-500/10 rounded-xl ring-1 ring-blue-500/20">
                                <Server className="w-5 h-5 text-blue-500" />
                            </div>
                            <div className="space-y-1">
                                <SheetTitle className="text-xl">{f5.name}</SheetTitle>
                                <SheetDescription className="text-sm">
                                    F5 负载均衡器详细信息
                                </SheetDescription>
                            </div>
                        </div>
                    </SheetHeader>

                    <div className="mt-6 space-y-6">
                        {/* Basic Info */}
                        <Card>
                            <CardHeader>
                                <CardTitle className="text-base flex items-center gap-2">
                                    <Network className="h-4 w-4" />
                                    基本信息
                                </CardTitle>
                            </CardHeader>
                            <CardContent className="space-y-3">
                                <div className="grid grid-cols-2 gap-4">
                                    <div>
                                        <div className="text-sm text-muted-foreground">名称</div>
                                        <div className="font-medium">{f5.name}</div>
                                    </div>
                                    <div>
                                        <div className="text-sm text-muted-foreground">VIP</div>
                                        <div className="font-mono font-medium">{f5.vip}</div>
                                    </div>
                                    <div>
                                        <div className="text-sm text-muted-foreground">端口</div>
                                        <div className="font-medium">{f5.port}</div>
                                    </div>
                                    <div>
                                        <div className="text-sm text-muted-foreground">AppID</div>
                                        <div className="font-medium">{f5.appid || '-'}</div>
                                    </div>
                                    <div>
                                        <div className="text-sm text-muted-foreground">实例组</div>
                                        <div className="font-medium">{f5.instance_group || '-'}</div>
                                    </div>
                                    <div>
                                        <div className="text-sm text-muted-foreground">状态</div>
                                        <div>{f5.status ? getStatusBadge(f5.status) : '-'}</div>
                                    </div>
                                </div>
                            </CardContent>
                        </Card>

                        {/* Pool Info */}
                        <Card>
                            <CardHeader>
                                <CardTitle className="text-base flex items-center gap-2">
                                    <Activity className="h-4 w-4" />
                                    Pool 信息
                                </CardTitle>
                            </CardHeader>
                            <CardContent className="space-y-3">
                                <div className="grid grid-cols-2 gap-4">
                                    <div>
                                        <div className="text-sm text-muted-foreground">Pool 名称</div>
                                        <div className="font-medium">{f5.pool_name || '-'}</div>
                                    </div>
                                    <div>
                                        <div className="text-sm text-muted-foreground">Pool 状态</div>
                                        <div>{f5.pool_status ? getStatusBadge(f5.pool_status) : '-'}</div>
                                    </div>
                                </div>

                                {poolMembers.length > 0 && (
                                    <div>
                                        <div className="text-sm text-muted-foreground mb-2">Pool 成员</div>
                                        <div className="space-y-1">
                                            {poolMembers.map((member, idx) => (
                                                <div key={idx} className="text-sm font-mono bg-muted/50 px-3 py-2 rounded">
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
                            <Card>
                                <CardHeader>
                                    <CardTitle className="text-base flex items-center gap-2">
                                        <Server className="h-4 w-4" />
                                        关联集群
                                    </CardTitle>
                                </CardHeader>
                                <CardContent>
                                    <div className="flex items-center justify-between">
                                        <div className="font-medium">{f5.cluster_name}</div>
                                        <Badge variant="outline">K8s Cluster</Badge>
                                    </div>
                                </CardContent>
                            </Card>
                        )}

                        {/* Domains */}
                        {domains.length > 0 && (
                            <Card>
                                <CardHeader>
                                    <CardTitle className="text-base flex items-center gap-2">
                                        <Globe className="h-4 w-4" />
                                        域名列表
                                    </CardTitle>
                                </CardHeader>
                                <CardContent>
                                    <div className="space-y-1">
                                        {domains.map((domain, idx) => (
                                            <div key={idx} className="flex items-center gap-2 text-sm">
                                                <ExternalLink className="h-3 w-3 text-muted-foreground" />
                                                <span className="font-mono">{domain}</span>
                                            </div>
                                        ))}
                                    </div>
                                </CardContent>
                            </Card>
                        )}

                        {/* Grafana */}
                        {f5.grafana_params && (
                            <Card>
                                <CardHeader>
                                    <CardTitle className="text-base">Grafana 参数</CardTitle>
                                </CardHeader>
                                <CardContent>
                                    <div className="text-sm font-mono bg-muted/50 px-3 py-2 rounded break-all">
                                        {f5.grafana_params}
                                    </div>
                                </CardContent>
                            </Card>
                        )}

                        {/* Metadata */}
                        <Card>
                            <CardHeader>
                                <CardTitle className="text-base">元数据</CardTitle>
                            </CardHeader>
                            <CardContent className="space-y-2 text-sm">
                                <div className="flex justify-between">
                                    <span className="text-muted-foreground">创建时间</span>
                                    <span className="font-mono">{f5.created_at || '-'}</span>
                                </div>
                                <Separator />
                                <div className="flex justify-between">
                                    <span className="text-muted-foreground">更新时间</span>
                                    <span className="font-mono">{f5.updated_at || '-'}</span>
                                </div>
                                <Separator />
                                <div className="flex justify-between items-center">
                                    <span className="text-muted-foreground">忽略状态</span>
                                    {f5.ignored ? (
                                        <Badge variant="outline" className="text-yellow-600">已忽略</Badge>
                                    ) : (
                                        <Badge variant="outline" className="text-green-600">正常</Badge>
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
                                className="flex-1"
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
                                variant="destructive"
                                onClick={() => setDeleteDialogOpen(true)}
                                className="flex-1"
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
                <AlertDialogContent>
                    <AlertDialogHeader>
                        <AlertDialogTitle>确认删除</AlertDialogTitle>
                        <AlertDialogDescription>
                            确定要删除 F5 "{f5.name}" 吗？此操作不可撤销。
                        </AlertDialogDescription>
                    </AlertDialogHeader>
                    <AlertDialogFooter>
                        <AlertDialogCancel>取消</AlertDialogCancel>
                        <AlertDialogAction
                            onClick={handleDelete}
                            disabled={isDeleting}
                            className="bg-destructive hover:bg-destructive/90"
                        >
                            {isDeleting ? '删除中...' : '确认删除'}
                        </AlertDialogAction>
                    </AlertDialogFooter>
                </AlertDialogContent>
            </AlertDialog>
        </>
    )
}

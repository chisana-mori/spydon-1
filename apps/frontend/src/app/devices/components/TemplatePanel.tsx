'use client'

import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
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
    FileText,
    Trash2,
    Play,
    ChevronLeft,
    ChevronRight,
    Clock,
    Layers,
    Filter,
} from 'lucide-react'
import RobustaAPI from '@/lib/api'
import { QueryTemplate, FilterGroup } from '@/types/navy'
import { formatDistanceToNow } from 'date-fns'
import { zhCN } from 'date-fns/locale'
import { cn } from '@/lib/utils'

interface TemplatePanelProps {
    onSelectTemplate: (template: QueryTemplate) => void
}

export function TemplatePanel({ onSelectTemplate }: TemplatePanelProps) {
    const queryClient = useQueryClient()
    const [page, setPage] = useState(1)
    const [deleteId, setDeleteId] = useState<number | null>(null)

    // 获取模板列表
    const { data, isLoading } = useQuery({
        queryKey: ['navy-templates', page],
        queryFn: () => RobustaAPI.listNavyTemplates(page, 12),
    })

    // 删除模板
    const deleteMutation = useMutation({
        mutationFn: (id: number) => RobustaAPI.deleteNavyTemplate(id),
        onSuccess: () => {
            toast.success('模板已删除')
            queryClient.invalidateQueries({ queryKey: ['navy-templates'] })
            setDeleteId(null)
        },
        onError: () => {
            toast.error('删除失败')
        },
    })

    const templates = data?.data || []
    const total = data?.pagination?.total || 0
    const totalPages = Math.ceil(total / 12)

    // 统计筛选组中的条件数量
    const countConditions = (groups: FilterGroup[]) => {
        return groups.reduce((sum, g) =>
            // 兼容性逻辑：只要 key 存在
            // 且 isActive / is_active / IsActive 不为 false (如果存在的话)
            sum + g.blocks.filter((b: any) => {
                const hasKey = b.key || b.Key;
                const active = b.isActive ?? b.is_active ?? b.IsActive ?? true;
                return !!hasKey && active !== false;
            }).length, 0)
    }

    if (isLoading) {
        return (
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                {Array.from({ length: 6 }).map((_, i) => (
                    <Card key={i} className="animate-pulse">
                        <CardHeader className="pb-3">
                            <Skeleton className="h-5 w-3/4" />
                            <Skeleton className="h-4 w-1/2 mt-2" />
                        </CardHeader>
                        <CardContent>
                            <Skeleton className="h-8 w-full" />
                        </CardContent>
                    </Card>
                ))}
            </div>
        )
    }

    if (templates.length === 0) {
        return (
            <Card className="border-dashed">
                <CardContent className="flex flex-col items-center justify-center py-12">
                    <div className="p-3 rounded-full bg-muted mb-4">
                        <FileText className="h-8 w-8 text-muted-foreground" />
                    </div>
                    <h3 className="font-medium text-lg mb-1">暂无查询模板</h3>
                    <p className="text-sm text-muted-foreground text-center max-w-sm">
                        使用高级查询功能创建复杂筛选条件后，可以保存为模板供下次快速使用
                    </p>
                </CardContent>
            </Card>
        )
    }

    return (
        <div className="space-y-4">
            {/* 模板网格 */}
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                {templates.map((template, index) => {
                    // Generate a color theme based on the ID or index
                    const colorIndex = (template.id || index) % 5

                    const themeIcon = [
                        "bg-blue-100 dark:bg-blue-900/40 text-blue-600 dark:text-blue-400",
                        "bg-emerald-100 dark:bg-emerald-900/40 text-emerald-600 dark:text-emerald-400",
                        "bg-violet-100 dark:bg-violet-900/40 text-violet-600 dark:text-violet-400",
                        "bg-amber-100 dark:bg-amber-900/40 text-amber-600 dark:text-amber-400",
                        "bg-rose-100 dark:bg-rose-900/40 text-rose-600 dark:text-rose-400",
                    ][colorIndex]

                    return (
                        <Card
                            key={template.id}
                            className={cn(
                                "group transition-all cursor-pointer shadow-sm hover:shadow-md hover:border-black/20 dark:hover:border-white/20"
                            )}
                            onClick={() => onSelectTemplate(template)}
                        >
                            <CardHeader className="pb-3">
                                <div className="flex items-start justify-between">
                                    <div className="flex items-center gap-3">
                                        <div className={cn("p-2 rounded-lg transition-transform group-hover:scale-105", themeIcon)}>
                                            <FileText className="h-5 w-5" />
                                        </div>
                                        <CardTitle className="text-base font-semibold line-clamp-1">
                                            {template.name}
                                        </CardTitle>
                                    </div>
                                    <Button
                                        variant="ghost"
                                        size="icon"
                                        className="h-7 w-7 opacity-0 group-hover:opacity-100 transition-opacity text-muted-foreground hover:text-destructive hover:bg-destructive/10"
                                        onClick={(e) => {
                                            e.stopPropagation()
                                            setDeleteId(template.id!)
                                        }}
                                    >
                                        <Trash2 className="h-4 w-4" />
                                    </Button>
                                </div>
                                {template.description && (
                                    <CardDescription className="line-clamp-2 mt-2 text-xs">
                                        {template.description}
                                    </CardDescription>
                                )}
                            </CardHeader>
                            <CardContent className="pt-0">
                                <div className="flex items-center justify-between mb-3 border-b border-black/5 dark:border-white/5 pb-3">
                                    <div className="flex items-center gap-3 text-xs text-muted-foreground/80">
                                        <span className="flex items-center gap-1.5">
                                            <Layers className="h-3.5 w-3.5" />
                                            {template.groups.length} 组
                                        </span>
                                        <span className="flex items-center gap-1.5">
                                            <Filter className="h-3.5 w-3.5" />
                                            {countConditions(template.groups)} 条件
                                        </span>
                                    </div>
                                    <span className="flex items-center gap-1.5 text-xs text-muted-foreground/60">
                                        <Clock className="h-3.5 w-3.5" />
                                        {template.updated_at ? formatDistanceToNow(new Date(template.updated_at), {
                                            addSuffix: true,
                                            locale: zhCN,
                                        }) : '-'}
                                    </span>
                                </div>

                                {/* 条件预览 */}
                                <div className="flex flex-wrap gap-2">
                                    {template.groups.slice(0, 2).flatMap(g =>
                                        g.blocks
                                            .filter((b: any) => {
                                                const hasKey = b.key || b.Key;
                                                const active = b.isActive ?? b.is_active ?? b.IsActive ?? true;
                                                return !!hasKey && active !== false;
                                            })
                                            .slice(0, 3)
                                            .map((b: any) => (
                                                <Badge
                                                    key={b.id || (b.key || b.Key)}
                                                    variant="secondary"
                                                    className="text-[10px] font-medium border-0 px-2 bg-muted text-muted-foreground"
                                                >
                                                    {b.key || b.Key}
                                                </Badge>
                                            ))
                                    )}
                                    {countConditions(template.groups) > 6 && (
                                        <Badge variant="outline" className="text-[10px] bg-background/50">
                                            +{countConditions(template.groups) - 6}
                                        </Badge>
                                    )}
                                </div>
                            </CardContent>
                        </Card>
                    )
                })}
            </div>

            {/* 分页 */}
            {totalPages > 1 && (
                <div className="flex items-center justify-between">
                    <p className="text-sm text-muted-foreground">
                        共 {total} 个模板
                    </p>
                    <div className="flex items-center gap-2">
                        <Button
                            variant="outline"
                            size="sm"
                            disabled={page <= 1}
                            onClick={() => setPage(p => p - 1)}
                        >
                            <ChevronLeft className="h-4 w-4" />
                            上一页
                        </Button>
                        <span className="text-sm text-muted-foreground">
                            {page} / {totalPages}
                        </span>
                        <Button
                            variant="outline"
                            size="sm"
                            disabled={page >= totalPages}
                            onClick={() => setPage(p => p + 1)}
                        >
                            下一页
                            <ChevronRight className="h-4 w-4" />
                        </Button>
                    </div>
                </div>
            )}

            {/* 删除确认对话框 */}
            <AlertDialog open={deleteId !== null} onOpenChange={() => setDeleteId(null)}>
                <AlertDialogContent>
                    <AlertDialogHeader>
                        <AlertDialogTitle>确认删除模板？</AlertDialogTitle>
                        <AlertDialogDescription>
                            此操作不可撤销，删除后该模板将永久消失。
                        </AlertDialogDescription>
                    </AlertDialogHeader>
                    <AlertDialogFooter>
                        <AlertDialogCancel>取消</AlertDialogCancel>
                        <AlertDialogAction
                            onClick={() => deleteId && deleteMutation.mutate(deleteId)}
                            className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
                        >
                            删除
                        </AlertDialogAction>
                    </AlertDialogFooter>
                </AlertDialogContent>
            </AlertDialog>
        </div>
    )
}

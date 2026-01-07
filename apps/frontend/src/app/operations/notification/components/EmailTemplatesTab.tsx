'use client';

import React, { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
    Plus,
    Search,
    Mail,
    MoreHorizontal,
    Edit,
    Trash,
    Loader2,
    Eye
} from 'lucide-react';
import { EmailTemplate } from '@/types/email';
import { RobustaAPI } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import {
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableHeader,
    TableRow,
} from '@/components/ui/table';
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuLabel,
    DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { Badge } from '@/components/ui/badge';
import { EmailTemplateDrawer } from './EmailTemplateDrawer';
import { toast } from 'sonner';
import {
    AlertDialog,
    AlertDialogAction,
    AlertDialogCancel,
    AlertDialogContent,
    AlertDialogDescription,
    AlertDialogFooter,
    AlertDialogHeader,
    AlertDialogTitle,
} from '@/components/ui/alert-dialog';

export function EmailTemplatesTab() {
    const queryClient = useQueryClient();
    const [keyword, setKeyword] = useState('');
    const [page, setPage] = useState(1);
    const [drawerOpen, setDrawerOpen] = useState(false);
    const [selectedTemplateId, setSelectedTemplateId] = useState<number | null>(null);
    const [deleteId, setDeleteId] = useState<number | null>(null);

    const { data, isLoading, isError } = useQuery({
        queryKey: ['email-templates', page, keyword],
        queryFn: () => RobustaAPI.listEmailTemplates(page, 20, keyword),
    });

    const deleteMutation = useMutation({
        mutationFn: RobustaAPI.deleteEmailTemplate,
        onSuccess: () => {
            toast.success('邮件模板已删除');
            queryClient.invalidateQueries({ queryKey: ['email-templates'] });
            setDeleteId(null);
        },
        onError: (error) => {
            console.error(error);
            toast.error('删除失败');
        },
    });

    const handleEdit = (id: number) => {
        setSelectedTemplateId(id);
        setDrawerOpen(true);
    };

    const handleCreate = () => {
        setSelectedTemplateId(null);
        setDrawerOpen(true);
    };

    const handleConfirmDelete = () => {
        if (deleteId) {
            deleteMutation.mutate(deleteId);
        }
    };

    const templates = data?.data || [];
    const total = data?.pagination?.total || 0;

    return (
        <div className="space-y-6">
            <div className="flex flex-col md:flex-row justify-between items-start md:items-center gap-4">
                {/* Header removed as it will be in the main page */}
            </div>

            {/* Filter Bar */}
            <div className="flex flex-col sm:flex-row gap-4 p-4 rounded-xl border bg-card/50 backdrop-blur-sm shadow-sm justify-between">
                <div className="relative flex-1 max-w-sm">
                    <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
                    <Input
                        placeholder="搜索模板名称或标题..."
                        className="pl-9 bg-background/50 border-transparent focus:border-input transition-all"
                        value={keyword}
                        onChange={(e) => { setKeyword(e.target.value); setPage(1); }}
                    />
                </div>
                <Button onClick={handleCreate} className="shadow-lg shadow-primary/20 hover:shadow-xl hover:shadow-primary/30 transition-all">
                    <Plus className="w-4 h-4 mr-2" /> 新建模板
                </Button>
            </div>

            {/* Data Table */}
            <div className="rounded-xl border bg-card shadow-sm overflow-hidden">
                <Table>
                    <TableHeader className="bg-muted/50">
                        <TableRow>
                            <TableHead className="w-[200px]">模板名称</TableHead>
                            <TableHead>邮件标题</TableHead>
                            <TableHead className="w-[100px]">参数数量</TableHead>
                            <TableHead className="w-[100px]">状态</TableHead>
                            <TableHead className="w-[150px]">更新时间</TableHead>
                            <TableHead className="text-right">操作</TableHead>
                        </TableRow>
                    </TableHeader>
                    <TableBody>
                        {isLoading ? (
                            <TableRow>
                                <TableCell colSpan={6} className="h-32 text-center">
                                    <div className="flex items-center justify-center text-muted-foreground">
                                        <Loader2 className="w-6 h-6 animate-spin mr-2" />
                                        加载中...
                                    </div>
                                </TableCell>
                            </TableRow>
                        ) : isError ? (
                            <TableRow>
                                <TableCell colSpan={6} className="h-32 text-center text-destructive">
                                    加载失败，请重试
                                </TableCell>
                            </TableRow>
                        ) : templates.length === 0 ? (
                            <TableRow>
                                <TableCell colSpan={6} className="h-32 text-center text-muted-foreground">
                                    暂无数据
                                </TableCell>
                            </TableRow>
                        ) : (
                            templates.map((template: EmailTemplate) => (
                                <TableRow key={template.id} className="group hover:bg-muted/30 transition-colors">
                                    <TableCell className="font-medium text-foreground">{template.name}</TableCell>
                                    <TableCell className="text-muted-foreground">{template.title}</TableCell>
                                    <TableCell>
                                        <Badge variant="secondary" className="font-mono">
                                            {template.params?.length || 0}
                                        </Badge>
                                    </TableCell>
                                    <TableCell>
                                        <Badge
                                            variant={template.is_enabled ? "default" : "destructive"}
                                            className={!template.is_enabled
                                                ? "bg-red-500/10 text-red-500 hover:bg-red-500/20 border-red-500/20 shadow-none"
                                                : "bg-green-500/10 text-green-600 hover:bg-green-500/20 border-green-500/20 shadow-none"}
                                        >
                                            {template.is_enabled ? '已启用' : '已禁用'}
                                        </Badge>
                                    </TableCell>
                                    <TableCell className="text-xs text-muted-foreground">
                                        {template.updated_at ? new Date(template.updated_at).toLocaleString('zh-CN', {
                                            month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit'
                                        }) : '-'}
                                    </TableCell>
                                    <TableCell className="text-right">
                                        <DropdownMenu>
                                            <DropdownMenuTrigger asChild>
                                                <Button variant="ghost" size="icon" className="h-8 w-8 p-0">
                                                    <span className="sr-only">打开菜单</span>
                                                    <MoreHorizontal className="h-4 w-4" />
                                                </Button>
                                            </DropdownMenuTrigger>
                                            <DropdownMenuContent align="end">
                                                <DropdownMenuLabel>操作</DropdownMenuLabel>
                                                <DropdownMenuItem onClick={() => handleEdit(template.id)}>
                                                    <Edit className="mr-2 h-4 w-4" /> 编辑
                                                </DropdownMenuItem>
                                                <DropdownMenuItem
                                                    onClick={() => setDeleteId(template.id)}
                                                    className="text-destructive focus:text-destructive"
                                                >
                                                    <Trash className="mr-2 h-4 w-4" /> 删除
                                                </DropdownMenuItem>
                                            </DropdownMenuContent>
                                        </DropdownMenu>
                                    </TableCell>
                                </TableRow>
                            ))
                        )}
                    </TableBody>
                </Table>
            </div>

            <EmailTemplateDrawer
                open={drawerOpen}
                onClose={() => setDrawerOpen(false)}
                templateId={selectedTemplateId}
            />

            <AlertDialog open={!!deleteId} onOpenChange={(val) => !val && setDeleteId(null)}>
                <AlertDialogContent>
                    <AlertDialogHeader>
                        <AlertDialogTitle>确认删除？</AlertDialogTitle>
                        <AlertDialogDescription>
                            此操作将永久删除该邮件模板，且不可恢复。
                        </AlertDialogDescription>
                    </AlertDialogHeader>
                    <AlertDialogFooter>
                        <AlertDialogCancel>取消</AlertDialogCancel>
                        <AlertDialogAction onClick={handleConfirmDelete} className="bg-destructive hover:bg-destructive/90">
                            确认删除
                        </AlertDialogAction>
                    </AlertDialogFooter>
                </AlertDialogContent>
            </AlertDialog>
        </div>
    );
}

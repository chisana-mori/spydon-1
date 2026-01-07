
'use client';

import React, { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
    Plus,
    Search,
    BookOpen,
    MoreHorizontal,
    Edit,
    Trash,
    Loader2
} from 'lucide-react';
import { Dictionary, DictionaryListParams } from '@/types/dictionary';
import { listDictionaries, deleteDictionary } from '@/lib/api/dictionaries';
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
import { DictionaryDrawer } from './components/DictionaryDrawer';
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

export default function DictionariesPage() {
    const queryClient = useQueryClient();
    const [queryParams, setQueryParams] = useState<DictionaryListParams>({
        page: 1,
        page_size: 10,
        keyword: '',
        module: '',
    });
    const [drawerOpen, setDrawerOpen] = useState(false);
    const [selectedDictId, setSelectedDictId] = useState<number | null>(null);
    const [deleteId, setDeleteId] = useState<number | null>(null);

    const { data, isLoading, isError } = useQuery({
        queryKey: ['dictionaries', queryParams],
        queryFn: () => listDictionaries(queryParams),
    });

    // Handle delete
    const deleteMutation = useMutation({
        mutationFn: deleteDictionary,
        onSuccess: () => {
            toast.success('字典已删除');
            queryClient.invalidateQueries({ queryKey: ['dictionaries'] });
            setDeleteId(null);
        },
        onError: (error) => {
            console.error(error);
            toast.error('删除失败');
        },
    });

    const handleEdit = (id: number) => {
        setSelectedDictId(id);
        setDrawerOpen(true);
    };

    const handleCreate = () => {
        setSelectedDictId(null);
        setDrawerOpen(true);
    };

    const handleConfirmDelete = () => {
        if (deleteId) {
            deleteMutation.mutate(deleteId);
        }
    };

    const dictionaries = data?.items || [];
    // If backend returns plain array instead of { items, total }, adjust here.
    // Based on previous API code, it returns data.data which is likely list or object.
    // Assuming object with items for now. If actually array, we need to adapt.
    // Safe check:
    const safeDictionaries = Array.isArray(data) ? data : (data?.items || []);
    const total = Array.isArray(data) ? data.length : (data?.total || 0);

    return (
        <div className="space-y-6">
            <div className="flex flex-col md:flex-row justify-between items-start md:items-center gap-4">
                <div className="flex items-center gap-3">
                    <div className="p-2.5 rounded-xl bg-blue-500/10 text-blue-500 ring-1 ring-blue-500/20">
                        <BookOpen className="h-6 w-6" />
                    </div>
                    <div>
                        <h1 className="text-2xl font-bold tracking-tight">字典管理</h1>
                        <p className="text-sm text-muted-foreground mt-0.5">
                            管理系统中的枚举值、下拉选项及配置参数。
                        </p>
                    </div>
                </div>
                <Button onClick={handleCreate} className="shadow-lg shadow-primary/20 hover:shadow-xl hover:shadow-primary/30 transition-all">
                    <Plus className="w-4 h-4 mr-2" /> 新建字典
                </Button>
            </div>

            {/* Filter Bar */}
            <div className="flex flex-col sm:flex-row gap-4 p-4 rounded-xl border bg-card/50 backdrop-blur-sm shadow-sm">
                <div className="relative flex-1">
                    <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
                    <Input
                        placeholder="搜索字典编码或名称..."
                        className="pl-9 bg-background/50 border-transparent focus:border-input transition-all"
                        value={queryParams.keyword}
                        onChange={(e) => setQueryParams({ ...queryParams, keyword: e.target.value, page: 1 })}
                    />
                </div>
                <div className="w-full sm:w-[200px]">
                    <Input
                        placeholder="按模块筛选..."
                        className="bg-background/50 border-transparent focus:border-input transition-all"
                        value={queryParams.module}
                        onChange={(e) => setQueryParams({ ...queryParams, module: e.target.value, page: 1 })}
                    />
                </div>
            </div>

            {/* Data Table */}
            <div className="rounded-xl border bg-card shadow-sm overflow-hidden">
                <Table>
                    <TableHeader className="bg-muted/50">
                        <TableRow>
                            <TableHead className="w-[200px]">编码 (Code)</TableHead>
                            <TableHead>名称</TableHead>
                            <TableHead>模块</TableHead>
                            <TableHead className="w-[100px]">项数量</TableHead>
                            <TableHead className="w-[100px]">状态</TableHead>
                            <TableHead className="w-[150px]">更新时间</TableHead>
                            <TableHead className="text-right">操作</TableHead>
                        </TableRow>
                    </TableHeader>
                    <TableBody>
                        {isLoading ? (
                            <TableRow>
                                <TableCell colSpan={7} className="h-32 text-center">
                                    <div className="flex items-center justify-center text-muted-foreground">
                                        <Loader2 className="w-6 h-6 animate-spin mr-2" />
                                        加载中...
                                    </div>
                                </TableCell>
                            </TableRow>
                        ) : isError ? (
                            <TableRow>
                                <TableCell colSpan={7} className="h-32 text-center text-destructive">
                                    加载失败，请重试
                                </TableCell>
                            </TableRow>
                        ) : safeDictionaries.length === 0 ? (
                            <TableRow>
                                <TableCell colSpan={7} className="h-32 text-center text-muted-foreground">
                                    暂无数据
                                </TableCell>
                            </TableRow>
                        ) : (
                            safeDictionaries.map((dict: Dictionary) => (
                                <TableRow key={dict.id} className="group hover:bg-muted/30 transition-colors">
                                    <TableCell className="font-mono font-medium text-foreground">{dict.code}</TableCell>
                                    <TableCell>
                                        <div className="flex flex-col">
                                            <span>{dict.name}</span>
                                            {dict.description && (
                                                <span className="text-xs text-muted-foreground truncate max-w-[200px]">
                                                    {dict.description}
                                                </span>
                                            )}
                                        </div>
                                    </TableCell>
                                    <TableCell>
                                        <Badge variant="outline" className="bg-blue-500/10 border-blue-500/20 text-blue-600 hover:bg-blue-500/20">
                                            {dict.module}
                                        </Badge>
                                    </TableCell>
                                    <TableCell>
                                        <Badge variant="secondary" className="font-mono">
                                            {dict.items?.length || 0}
                                        </Badge>
                                    </TableCell>
                                    <TableCell>
                                        <Badge variant={dict.is_enabled ? "default" : "destructive"} className={!dict.is_enabled ? "bg-red-500/10 text-red-500 hover:bg-red-500/20 border-red-500/20 shadow-none" : "bg-green-500/10 text-green-600 hover:bg-green-500/20 border-green-500/20 shadow-none"}>
                                            {dict.is_enabled ? '已启用' : '已禁用'}
                                        </Badge>
                                    </TableCell>
                                    <TableCell className="text-xs text-muted-foreground">
                                        {dict.updated_at ? new Date(dict.updated_at).toLocaleString('zh-CN', {
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
                                                <DropdownMenuItem onClick={() => handleEdit(dict.id)}>
                                                    <Edit className="mr-2 h-4 w-4" /> 编辑
                                                </DropdownMenuItem>
                                                <DropdownMenuItem
                                                    onClick={() => setDeleteId(dict.id)}
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

            <DictionaryDrawer
                open={drawerOpen}
                onClose={() => setDrawerOpen(false)}
                dictionaryId={selectedDictId}
            />

            <AlertDialog open={!!deleteId} onOpenChange={(val) => !val && setDeleteId(null)}>
                <AlertDialogContent>
                    <AlertDialogHeader>
                        <AlertDialogTitle>确认删除？</AlertDialogTitle>
                        <AlertDialogDescription>
                            此操作将永久删除该字典及其所有字典项，且不可恢复。
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

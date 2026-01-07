'use client';

import React, { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Plus, AlertTriangle, Search, MoreHorizontal, Edit, Trash, Loader2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuLabel, DropdownMenuTrigger } from '@/components/ui/dropdown-menu';
import { Badge } from '@/components/ui/badge';
import { toast } from 'sonner';
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from '@/components/ui/alert-dialog';
import { useDictionary } from '@/hooks/useDictionary';
import { api } from '@/lib/api';
import { PaginationControl } from '@/components/common/PaginationControl';
import { TaintSheet } from './components/TaintSheet';

type TaintManagement = {
    id: number;
    key: string;
    value: string;
    effect: string;
    description: string;
    type: string;
    status: number;
};

export default function TaintsPage() {
    const queryClient = useQueryClient();
    const [keyword, setKeyword] = useState('');
    const [deleteId, setDeleteId] = useState<number | null>(null);

    const [page, setPage] = useState(1);
    const [size, setSize] = useState(10);

    const { dict: statusDict } = useDictionary('taint_status');

    const { data, isLoading, isError } = useQuery({
        queryKey: ['taints', keyword, page, size],
        queryFn: async () => {
            const params = new URLSearchParams();
            if (keyword) params.append('keyword', keyword);
            params.append('page', page.toString());
            params.append('size', size.toString());
            const res = await api.get(`/configuration/taints?${params}`);
            return res.data;
        },
    });

    const deleteMutation = useMutation({
        mutationFn: async (id: number) => {
            await api.delete(`/configuration/taints/${id}`);
        },
        onSuccess: () => {
            toast.success('污点已删除');
            queryClient.invalidateQueries({ queryKey: ['taints'] });
            setDeleteId(null);
        },
        onError: () => {
            toast.error('删除失败');
        },
    });

    const handleConfirmDelete = () => {
        if (deleteId) deleteMutation.mutate(deleteId);
    };

    const [sheetOpen, setSheetOpen] = useState(false);
    const [selectedTaint, setSelectedTaint] = useState<TaintManagement | null>(null);

    const handleCreate = () => {
        setSelectedTaint(null);
        setSheetOpen(true);
    };

    const handleEdit = (taint: TaintManagement) => {
        setSelectedTaint(taint);
        setSheetOpen(true);
    };

    const taints: TaintManagement[] = data?.list || [];

    return (
        <div className="space-y-6">
            <div className="flex flex-col md:flex-row justify-between items-start md:items-center gap-4">
                <div className="flex items-center gap-3">
                    <div className="p-2.5 rounded-xl bg-blue-500/10 text-blue-500 ring-1 ring-blue-500/20">
                        <AlertTriangle className="h-6 w-6" />
                    </div>
                    <div>
                        <h1 className="text-2xl font-bold tracking-tight">污点管理</h1>
                        <p className="text-sm text-muted-foreground mt-0.5">管理节点污点配置与调度策略。</p>
                    </div>
                </div>
                <Button onClick={handleCreate} className="shadow-lg shadow-primary/20 hover:shadow-xl hover:shadow-primary/30 transition-all">
                    <Plus className="w-4 h-4 mr-2" /> 新建污点
                </Button>
            </div>

            <div className="flex gap-4 p-4 rounded-xl border bg-card/50 backdrop-blur-sm shadow-sm">
                <div className="relative flex-1 max-w-sm">
                    <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
                    <Input
                        placeholder="搜索Key或描述..."
                        className="pl-9 bg-background/50 border-muted-foreground/20 focus:border-primary/50 transition-colors"
                        value={keyword}
                        onChange={(e) => setKeyword(e.target.value)}
                    />
                </div>
            </div>

            <div className="rounded-xl border bg-card shadow-sm overflow-hidden flex flex-col">
                <div className="flex-1">
                    <Table>
                        <TableHeader className="bg-muted/30">
                            <TableRow className="hover:bg-transparent">
                                <TableHead className="w-[60px]">ID</TableHead>
                                <TableHead>Key</TableHead>
                                <TableHead>Value</TableHead>
                                <TableHead>Effect</TableHead>
                                <TableHead>类型</TableHead>
                                <TableHead className="w-[100px]">状态</TableHead>
                                <TableHead className="text-right">操作</TableHead>
                            </TableRow>
                        </TableHeader>
                        <TableBody>
                            {isLoading ? (
                                <TableRow>
                                    <TableCell colSpan={7} className="h-32 text-center">
                                        <div className="flex items-center justify-center text-muted-foreground">
                                            <Loader2 className="w-6 h-6 animate-spin mr-2" /> 加载中...
                                        </div>
                                    </TableCell>
                                </TableRow>
                            ) : isError || taints.length === 0 ? (
                                <TableRow>
                                    <TableCell colSpan={7} className="h-32 text-center text-muted-foreground">暂无数据</TableCell>
                                </TableRow>
                            ) : (
                                taints.map((taint) => (
                                    <TableRow key={taint.id} className="group hover:bg-blue-500/5 transition-colors">
                                        <TableCell className="font-mono text-xs text-muted-foreground">{taint.id}</TableCell>
                                        <TableCell className="font-medium font-mono text-sm max-w-[200px] truncate text-foreground/90" title={taint.key}>{taint.key}</TableCell>
                                        <TableCell className="font-mono text-sm text-muted-foreground">{taint.value || '-'}</TableCell>
                                        <TableCell>
                                            <Badge variant="outline" className="bg-orange-500/10 text-orange-600 border-orange-500/20 hover:bg-orange-500/20 transition-colors">
                                                {taint.effect}
                                            </Badge>
                                        </TableCell>
                                        <TableCell><span className="text-sm text-muted-foreground">{taint.type || '-'}</span></TableCell>
                                        <TableCell>
                                            <Badge variant="outline" className={
                                                taint.status === 0
                                                    ? "bg-green-500/10 text-green-600 border-green-500/20 hover:bg-green-500/20 shadow-none"
                                                    : "bg-red-500/10 text-red-500 border-red-500/20 hover:bg-red-500/20 shadow-none"
                                            }>
                                                {statusDict?.[taint.status] || '未知'}
                                            </Badge>
                                        </TableCell>
                                        <TableCell className="text-right">
                                            <DropdownMenu>
                                                <DropdownMenuTrigger asChild>
                                                    <Button variant="ghost" size="icon" className="h-8 w-8 p-0">
                                                        <MoreHorizontal className="h-4 w-4" />
                                                    </Button>
                                                </DropdownMenuTrigger>
                                                <DropdownMenuContent align="end">
                                                    <DropdownMenuLabel>操作</DropdownMenuLabel>
                                                    <DropdownMenuItem onClick={() => handleEdit(taint)}>
                                                        <Edit className="mr-2 h-4 w-4" /> 编辑
                                                    </DropdownMenuItem>
                                                    <DropdownMenuItem onClick={() => setDeleteId(taint.id)} className="text-destructive focus:text-destructive">
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
                {data && (
                    <div className="p-4 border-t flex items-center justify-between">
                        <div className="text-sm text-muted-foreground">
                            共 {data.total || 0} 条
                        </div>
                        <PaginationControl
                            total={data.total || 0}
                            page={page}
                            size={size}
                            onPageChange={setPage}
                            onSizeChange={setSize}
                        />
                    </div>
                )}
            </div>

            <TaintSheet open={sheetOpen} onOpenChange={setSheetOpen} taint={selectedTaint} />

            <AlertDialog open={!!deleteId} onOpenChange={(val) => !val && setDeleteId(null)}>
                <AlertDialogContent>
                    <AlertDialogHeader>
                        <AlertDialogTitle>确认删除？</AlertDialogTitle>
                        <AlertDialogDescription>此操作将永久删除该污点配置，且不可恢复。</AlertDialogDescription>
                    </AlertDialogHeader>
                    <AlertDialogFooter>
                        <AlertDialogCancel>取消</AlertDialogCancel>
                        <AlertDialogAction onClick={handleConfirmDelete} className="bg-destructive hover:bg-destructive/90">确认删除</AlertDialogAction>
                    </AlertDialogFooter>
                </AlertDialogContent>
            </AlertDialog>
        </div>
    );
}

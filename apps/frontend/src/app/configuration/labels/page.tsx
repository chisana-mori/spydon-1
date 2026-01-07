'use client';

import React, { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Plus, Tags, Search, MoreHorizontal, Edit, Trash, Loader2 } from 'lucide-react';
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
import { LabelSheet } from './components/LabelSheet';

type LabelManagement = {
    id: number;
    name: string;
    key: string;
    source: number;
    status: number;
};

export default function LabelsPage() {
    const queryClient = useQueryClient();
    const [keyword, setKeyword] = useState('');
    const [deleteId, setDeleteId] = useState<number | null>(null);

    const [page, setPage] = useState(1);
    const [size, setSize] = useState(10);

    const { dict: sourceDict } = useDictionary('label_source');
    const { dict: statusDict } = useDictionary('label_status');

    const { data, isLoading, isError } = useQuery({
        queryKey: ['labels', keyword, page, size],
        queryFn: async () => {
            const params = new URLSearchParams();
            if (keyword) params.append('keyword', keyword);
            params.append('page', page.toString());
            params.append('size', size.toString());
            const res = await api.get(`/configuration/labels?${params}`);
            return res.data; // Return full response { list, total }
        },
    });

    const deleteMutation = useMutation({
        mutationFn: async (id: number) => {
            await api.delete(`/configuration/labels/${id}`);
        },
        onSuccess: () => {
            toast.success('标签已删除');
            queryClient.invalidateQueries({ queryKey: ['labels'] });
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
    const [selectedLabel, setSelectedLabel] = useState<LabelManagement | null>(null);

    const handleCreate = () => {
        setSelectedLabel(null);
        setSheetOpen(true);
    };

    const handleEdit = (label: LabelManagement) => {
        setSelectedLabel(label);
        setSheetOpen(true);
    };

    const labels: LabelManagement[] = data?.list || [];

    return (
        <div className="space-y-6">
            <div className="flex flex-col md:flex-row justify-between items-start md:items-center gap-4">
                <div className="flex items-center gap-3">
                    <div className="p-2.5 rounded-xl bg-blue-500/10 text-blue-500 ring-1 ring-blue-500/20">
                        <Tags className="h-6 w-6" />
                    </div>
                    <div>
                        <h1 className="text-2xl font-bold tracking-tight">标签管理</h1>
                        <p className="text-sm text-muted-foreground mt-0.5">管理节点标签特性配置。</p>
                    </div>
                </div>
                <Button onClick={handleCreate} className="shadow-lg shadow-primary/20 hover:shadow-xl hover:shadow-primary/30 transition-all">
                    <Plus className="w-4 h-4 mr-2" /> 新建标签
                </Button>
            </div>

            <div className="flex gap-4 p-4 rounded-xl border bg-card/50 backdrop-blur-sm shadow-sm">
                <div className="relative flex-1 max-w-sm">
                    <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
                    <Input placeholder="搜索标签名称或Key..." className="pl-9 bg-background/50 border-muted-foreground/20 focus:border-primary/50 transition-colors" value={keyword} onChange={(e) => setKeyword(e.target.value)} />
                </div>
            </div>

            <div className="rounded-xl border bg-card shadow-sm overflow-hidden flex flex-col">
                <div className="flex-1">
                    <Table>
                        <TableHeader className="bg-muted/30">
                            <TableRow className="hover:bg-transparent">
                                <TableHead className="w-[60px]">ID</TableHead>
                                <TableHead>名称</TableHead>
                                <TableHead>Key</TableHead>
                                <TableHead className="w-[120px]">来源</TableHead>
                                <TableHead className="w-[100px]">状态</TableHead>
                                <TableHead className="text-right">操作</TableHead>
                            </TableRow>
                        </TableHeader>
                        <TableBody>
                            {isLoading ? (
                                <TableRow>
                                    <TableCell colSpan={6} className="h-32 text-center">
                                        <div className="flex items-center justify-center text-muted-foreground">
                                            <Loader2 className="w-6 h-6 animate-spin mr-2" /> 加载中...
                                        </div>
                                    </TableCell>
                                </TableRow>
                            ) : isError || labels.length === 0 ? (
                                <TableRow>
                                    <TableCell colSpan={6} className="h-32 text-center text-muted-foreground">暂无数据</TableCell>
                                </TableRow>
                            ) : (
                                labels.map((label) => (
                                    <TableRow key={label.id} className="group hover:bg-blue-500/5 transition-colors">
                                        <TableCell className="font-mono text-xs text-muted-foreground">{label.id}</TableCell>
                                        <TableCell className="font-medium text-foreground/90">{label.name}</TableCell>
                                        <TableCell className="font-mono text-sm text-muted-foreground">{label.key}</TableCell>
                                        <TableCell>
                                            <Badge variant="outline" className="bg-blue-500/10 border-blue-500/20 text-blue-600 hover:bg-blue-500/20 transition-colors">
                                                {sourceDict?.[label.source] || label.source}
                                            </Badge>
                                        </TableCell>
                                        <TableCell>
                                            <Badge variant="outline" className={
                                                label.status === 0
                                                    ? "bg-green-500/10 text-green-600 border-green-500/20 hover:bg-green-500/20 shadow-none"
                                                    : "bg-red-500/10 text-red-500 border-red-500/20 hover:bg-red-500/20 shadow-none"
                                            }>
                                                {statusDict?.[label.status] || '未知'}
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
                                                    <DropdownMenuItem onClick={() => handleEdit(label)}>
                                                        <Edit className="mr-2 h-4 w-4" /> 编辑
                                                    </DropdownMenuItem>
                                                    <DropdownMenuItem onClick={() => setDeleteId(label.id)} className="text-destructive focus:text-destructive">
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

            <LabelSheet open={sheetOpen} onOpenChange={setSheetOpen} label={selectedLabel} />

            <AlertDialog open={!!deleteId} onOpenChange={(val) => !val && setDeleteId(null)}>
                <AlertDialogContent>
                    <AlertDialogHeader>
                        <AlertDialogTitle>确认删除？</AlertDialogTitle>
                        <AlertDialogDescription>此操作将永久删除该标签，且不可恢复。</AlertDialogDescription>
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

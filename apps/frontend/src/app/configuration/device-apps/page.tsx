'use client';

import React, { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Plus, Cpu, Search, MoreHorizontal, Edit, Trash, Loader2, Settings } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuLabel, DropdownMenuTrigger } from '@/components/ui/dropdown-menu';
import { Badge } from '@/components/ui/badge';
import { toast } from 'sonner';
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from '@/components/ui/alert-dialog';
import { useDictionary } from '@/hooks/useDictionary';
import { api } from '@/lib/api';
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { PaginationControl } from '@/components/common/PaginationControl';
import { DeviceAppSheet } from './components/DeviceAppSheet';

type DeviceApp = {
    id: number;
    app_id: string;
    type: number;
    name: string;
    owner: string;
    feature: string;
    description: string;
    status: number;
};

export default function DeviceAppsPage() {
    const queryClient = useQueryClient();
    const [keyword, setKeyword] = useState('');
    const [deleteId, setDeleteId] = useState<number | null>(null);

    const [page, setPage] = useState(1);
    const [size, setSize] = useState(10);
    const [activeType, setActiveType] = useState<string>("all");

    const { dict: typeDict } = useDictionary('device_type');
    const { dict: statusDict } = useDictionary('device_status');

    const { data, isLoading, isError } = useQuery({
        queryKey: ['device-apps', keyword, page, size, activeType],
        queryFn: async () => {
            const params = new URLSearchParams();
            if (keyword) params.append('keyword', keyword);
            if (activeType !== "all") params.append('type', activeType);
            params.append('page', page.toString());
            params.append('size', size.toString());
            const res = await api.get(`/configuration/device-apps?${params}`);
            return res.data;
        },
    });

    const deleteMutation = useMutation({
        mutationFn: async (id: number) => {
            await api.delete(`/configuration/device-apps/${id}`);
        },
        onSuccess: () => {
            toast.success('设备应用已删除');
            queryClient.invalidateQueries({ queryKey: ['device-apps'] });
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
    const [selectedDeviceApp, setSelectedDeviceApp] = useState<DeviceApp | null>(null);

    const handleCreate = () => {
        setSelectedDeviceApp(null);
        setSheetOpen(true);
    };

    const handleEdit = (deviceApp: DeviceApp) => {
        setSelectedDeviceApp(deviceApp);
        setSheetOpen(true);
    };

    const apps: DeviceApp[] = data?.list || [];

    return (
        <div className="space-y-6">
            <div className="flex flex-col md:flex-row justify-between items-start md:items-center gap-4">
                <div className="flex items-center gap-3">
                    <div className="p-2.5 rounded-xl bg-blue-500/10 text-blue-500 ring-1 ring-blue-500/20">
                        <Settings className="h-6 w-6" />
                    </div>
                    <div>
                        <h1 className="text-2xl font-bold tracking-tight">设备应用管理</h1>
                        <p className="text-sm text-muted-foreground mt-0.5">管理设备应用类型及采集状态。</p>
                    </div>
                </div>
                <Button onClick={handleCreate} className="shadow-lg shadow-primary/20 hover:shadow-xl hover:shadow-primary/30 transition-all">
                    <Plus className="w-4 h-4 mr-2" /> 新建设备应用
                </Button>
            </div>

            <Tabs defaultValue="all" value={activeType} onValueChange={setActiveType} className="w-full">
                <TabsList className="bg-muted/50 p-1">
                    <TabsTrigger value="all" className="data-[state=active]:bg-background data-[state=active]:shadow-sm">全部</TabsTrigger>
                    <TabsTrigger value="0" className="data-[state=active]:bg-background data-[state=active]:shadow-sm">设备</TabsTrigger>
                    <TabsTrigger value="1" className="data-[state=active]:bg-background data-[state=active]:shadow-sm">组件</TabsTrigger>
                    <TabsTrigger value="2" className="data-[state=active]:bg-background data-[state=active]:shadow-sm">应用</TabsTrigger>
                </TabsList>
            </Tabs>

            <div className="flex gap-4 p-4 rounded-xl border bg-card/50 backdrop-blur-sm shadow-sm">
                <div className="relative flex-1 max-w-sm">
                    <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
                    <Input
                        placeholder="搜索AppId、名称或负责人..."
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
                                <TableHead className="w-[150px]">AppId</TableHead>
                                <TableHead className="w-[150px]">名称</TableHead>
                                <TableHead className="w-[100px]">类型</TableHead>
                                <TableHead className="w-[120px]">负责人</TableHead>
                                <TableHead>用途</TableHead>
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
                            ) : isError || apps.length === 0 ? (
                                <TableRow>
                                    <TableCell colSpan={7} className="h-32 text-center text-muted-foreground">暂无数据</TableCell>
                                </TableRow>
                            ) : (
                                apps.map((app) => (
                                    <TableRow key={app.id} className="group hover:bg-blue-500/5 transition-colors">
                                        <TableCell className="font-mono text-xs font-medium text-foreground/90">{app.app_id}</TableCell>
                                        <TableCell className="font-medium">{app.name}</TableCell>
                                        <TableCell>
                                            <Badge variant="outline" className="bg-blue-500/10 border-blue-500/20 text-blue-600 hover:bg-blue-500/20 transition-colors">
                                                {typeDict?.[app.type] || app.type}
                                            </Badge>
                                        </TableCell>
                                        <TableCell className="text-sm text-muted-foreground">{app.owner}</TableCell>
                                        <TableCell className="text-sm text-muted-foreground truncate max-w-[200px]">{app.description}</TableCell>
                                        <TableCell>
                                            <Badge variant="outline" className={
                                                app.status === 0
                                                    ? "bg-green-500/10 text-green-600 border-green-500/20 hover:bg-green-500/20 shadow-none"
                                                    : "bg-red-500/10 text-red-500 border-red-500/20 hover:bg-red-500/20 shadow-none"
                                            }>
                                                {statusDict?.[app.status] || '未知'}
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
                                                    <DropdownMenuItem onClick={() => handleEdit(app)}>
                                                        <Edit className="mr-2 h-4 w-4" /> 编辑
                                                    </DropdownMenuItem>
                                                    <DropdownMenuItem onClick={() => setDeleteId(app.id)} className="text-destructive focus:text-destructive">
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

            <DeviceAppSheet open={sheetOpen} onOpenChange={setSheetOpen} deviceApp={selectedDeviceApp} />

            <AlertDialog open={!!deleteId} onOpenChange={(val) => !val && setDeleteId(null)}>
                <AlertDialogContent>
                    <AlertDialogHeader>
                        <AlertDialogTitle>确认删除？</AlertDialogTitle>
                        <AlertDialogDescription>此操作将永久删除该设备应用配置，且不可恢复。</AlertDialogDescription>
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

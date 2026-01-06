import React from 'react';
import {
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableHeader,
    TableRow,
} from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";

import { DrainPodMigrationStatus, DrainPodMigrationInfo } from '@/types/safe-drain';

const StatusBadge: React.FC<{ status: DrainPodMigrationStatus }> = ({ status }) => {
    const variants: Record<DrainPodMigrationStatus, "default" | "secondary" | "destructive" | "outline"> = {
        pending: "outline",
        evicting: "secondary",
        evicted: "secondary",
        creating: "default",
        completed: "default",
        failed: "destructive",
        ignored: "outline",
        timeout: "destructive"
    };

    const colors: Record<DrainPodMigrationStatus, string> = {
        pending: "text-zinc-500",
        evicting: "bg-blue-100 text-blue-800 hover:bg-blue-100 dark:bg-blue-900/30 dark:text-blue-300",
        evicted: "bg-purple-100 text-purple-800 hover:bg-purple-100 dark:bg-purple-900/30 dark:text-purple-300",
        creating: "bg-indigo-100 text-indigo-800 hover:bg-indigo-100 dark:bg-indigo-900/30 dark:text-indigo-300",
        completed: "bg-green-100 text-green-800 hover:bg-green-100 dark:bg-green-900/30 dark:text-green-300",
        failed: "bg-red-100 text-red-800 hover:bg-red-100 dark:bg-red-900/30 dark:text-red-300",
        ignored: "text-zinc-400 bg-zinc-100 dark:bg-zinc-800",
        timeout: "bg-orange-100 text-orange-800 hover:bg-orange-100 dark:bg-orange-900/30 dark:text-orange-300"
    };

    const labels: Record<DrainPodMigrationStatus, string> = {
        pending: "等待中",
        evicting: "驱逐中",
        evicted: "已驱逐",
        creating: "重建中",
        completed: "已迁移",
        failed: "失败",
        ignored: "已忽略",
        timeout: "超时"
    };

    return (
        <Badge variant={variants[status]} className={`${colors[status]} whitespace-nowrap`}>
            {labels[status] || status.toUpperCase()}
        </Badge>
    );
};

export const MigrationsTable: React.FC<{ migrations: DrainPodMigrationInfo[] }> = ({ migrations }) => {

    return (
        <div className="rounded-md border h-full overflow-auto relative bg-white dark:bg-zinc-950">
            <Table>
                <TableHeader className="sticky top-0 bg-zinc-50 dark:bg-zinc-900/50 z-10 shadow-sm">
                    <TableRow>
                        <TableHead className="w-[200px]">原Pod名称</TableHead>
                        <TableHead className="w-[150px]">Namespace</TableHead>
                        <TableHead className="w-[200px]">新Pod名称</TableHead>
                        <TableHead className="w-[120px]">目标节点</TableHead>
                        <TableHead className="w-[100px]">Pod状态</TableHead>
                        <TableHead className="w-[100px]">迁移状态</TableHead>
                        <TableHead className="w-[150px] text-right">信息</TableHead>
                    </TableRow>
                </TableHeader>
                <TableBody>
                    {migrations.length === 0 ? (
                        <TableRow>
                            <TableCell colSpan={7} className="h-24 text-center text-muted-foreground">
                                暂无 Pod 迁移数据
                            </TableCell>
                        </TableRow>
                    ) : (
                        migrations.map((mig) => (
                            <TableRow key={mig.migrationId} className="hover:bg-zinc-50 dark:hover:bg-zinc-900/50">
                                <TableCell className="font-medium max-w-[200px] truncate" title={mig.sourcePod.name}>
                                    {mig.sourcePod.name}
                                </TableCell>
                                <TableCell className="max-w-[150px] truncate" title={mig.sourcePod.namespace}>
                                    {mig.sourcePod.namespace}
                                </TableCell>
                                <TableCell className="max-w-[200px] truncate" title={mig.targetPod?.name}>
                                    {mig.targetPod?.name || '-'}
                                </TableCell>
                                <TableCell className="max-w-[120px] truncate">
                                    {mig.targetPod?.nodeName || '-'}
                                </TableCell>
                                <TableCell>
                                    <span className="text-xs text-muted-foreground">
                                        {mig.targetPod?.phase || mig.sourcePod.phase || '-'}
                                    </span>
                                </TableCell>
                                <TableCell>
                                    <StatusBadge status={mig.status} />
                                </TableCell>
                                <TableCell className="text-right text-sm text-muted-foreground max-w-[150px] truncate" title={mig.errorMessage}>
                                    {mig.errorMessage || '-'}
                                </TableCell>
                            </TableRow>
                        ))
                    )}
                </TableBody>
            </Table>
        </div>
    );
};

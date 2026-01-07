import React, { useState } from 'react';
import {
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableHeader,
    TableRow,
} from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { ChevronDown, ChevronRight, CheckCircle2, Server, Info, Activity } from "lucide-react";
import type { DrainPodMigrationInfo, DrainPodMigrationStatus, DrainStats, SortedMigrations } from '@/types/safe-drain';
import { cn } from "@/lib/utils";

// --- Status Badge 组件 ---
const StatusBadge: React.FC<{ status: DrainPodMigrationStatus }> = ({ status }) => {
    const normalizedStatus = (typeof status === 'string' ? status.toLowerCase() : status) as string;

    const config: Record<string, { className: string; label: string }> = {
        pending: {
            className: "bg-zinc-100 text-zinc-600 border-zinc-200 dark:bg-zinc-800 dark:text-zinc-400 dark:border-zinc-700",
            label: "等待中"
        },
        evicting: {
            className: "bg-blue-50 text-blue-700 border-blue-200 dark:bg-blue-900/20 dark:text-blue-300 dark:border-blue-800 animate-pulse",
            label: "驱逐中"
        },
        evicted: {
            className: "bg-violet-50 text-violet-700 border-violet-200 dark:bg-violet-900/20 dark:text-violet-300 dark:border-violet-800",
            label: "已驱逐"
        },
        creating: {
            className: "bg-sky-50 text-sky-700 border-sky-200 dark:bg-sky-900/20 dark:text-sky-300 dark:border-sky-800",
            label: "重建中"
        },
        completed: {
            className: "bg-emerald-50 text-emerald-700 border-emerald-200 dark:bg-emerald-900/20 dark:text-emerald-300 dark:border-emerald-800",
            label: "已迁移"
        },
        ready: {
            className: "bg-emerald-50 text-emerald-700 border-emerald-200 dark:bg-emerald-900/20 dark:text-emerald-300 dark:border-emerald-800",
            label: "Ready"
        },
        failed: {
            className: "bg-red-50 text-red-700 border-red-200 dark:bg-red-900/20 dark:text-red-300 dark:border-red-800",
            label: "失败"
        },
        ignored: {
            className: "bg-zinc-50 text-zinc-500 border-zinc-200 dark:bg-zinc-900/20 dark:text-zinc-500 dark:border-zinc-800 dashed border",
            label: "已忽略"
        },
        timeout: {
            className: "bg-orange-50 text-orange-700 border-orange-200 dark:bg-orange-900/20 dark:text-orange-300 dark:border-orange-800",
            label: "超时"
        }
    };

    const cfg = config[normalizedStatus] || config['pending'];

    return (
        <Badge variant="outline" className={cn("font-normal border h-5 px-2 text-[10px]", cfg.className)}>
            {cfg.label}
        </Badge>
    );
};

// --- Pod Phase Badge ---
const PhaseBadge: React.FC<{ phase?: string; phaseReason?: string }> = ({ phase, phaseReason }) => {
    const displayPhase = phaseReason || phase || '';
    if (!displayPhase) return <span className="text-xs text-muted-foreground/30">-</span>;

    const isError = /backoff|error|crash|err/i.test(displayPhase);
    const isRunning = displayPhase === 'Running';
    const isPending = displayPhase === 'Pending';
    const isSucceeded = displayPhase === 'Succeeded';

    return (
        <div className={cn(
            "inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-[10px] font-medium border h-5",
            isError ? "bg-red-50 text-red-700 border-red-200 dark:bg-red-900/20 dark:text-red-300 dark:border-red-800" :
                isRunning ? "bg-green-50 text-green-700 border-green-200 dark:bg-green-900/20 dark:text-green-300 dark:border-green-800" :
                    isSucceeded ? "bg-green-50 text-green-700 border-green-200 dark:bg-green-900/20 dark:text-green-300 dark:border-green-800" :
                        isPending ? "bg-amber-50 text-amber-700 border-amber-200 dark:bg-amber-900/20 dark:text-amber-300 dark:border-amber-800" :
                            "bg-zinc-100 text-zinc-700 border-zinc-200 dark:bg-zinc-800 dark:text-zinc-300 dark:border-zinc-700"
        )}>
            <span className={cn(
                "w-1 h-1 rounded-full",
                isError ? "bg-red-500" :
                    isRunning ? "bg-green-500" :
                        isSucceeded ? "bg-green-500" :
                            isPending ? "bg-amber-500" :
                                "bg-zinc-500"
            )} />
            {displayPhase}
        </div>
    );
};

// --- 时间格式化 ---
const formatStartTime = (value?: string) => {
    if (!value) return '';

    const time = new Date(value).getTime();
    if (Number.isNaN(time)) return value || '';

    const dateText = new Date(time).toLocaleString(undefined, {
        hour: 'numeric',
        minute: 'numeric',
        second: 'numeric'
    });

    // 简化的相对时间
    const diff = Math.max(0, Date.now() - time);
    const minute = 60 * 1000;

    if (diff < minute) return '刚刚';
    return dateText;
};

// --- Migration Row 组件 ---
const MigrationRow: React.FC<{ migration: DrainPodMigrationInfo; clusterName?: string }> = ({ migration, clusterName }) => {
    const statusStr = (migration.status as string).toLowerCase();
    const isIgnored = statusStr === 'ignored';

    const getPodLink = (name: string, namespace: string) => {
        if (!clusterName) return null;
        return `/kite/pods/${namespace}/${name}?tab=overview&cluster=${clusterName}`;
    };

    const sourceLink = getPodLink(migration.sourcePod.name, migration.sourcePod.namespace);
    const targetLink = migration.targetPod ? getPodLink(migration.targetPod.name, migration.targetPod.namespace) : null;

    return (
        <TableRow className="group transition-colors hover:bg-muted/50 data-[state=selected]:bg-muted">
            {/* 原 Pod 名称 */}
            <TableCell className="py-2.5 pl-4">
                <div className="flex flex-col gap-1 max-w-[200px]">
                    {sourceLink ? (
                        <a
                            href={sourceLink}
                            target="_blank"
                            rel="noopener noreferrer"
                            className="font-mono text-xs font-medium text-blue-600 dark:text-blue-400 hover:underline truncate"
                            title={`查看 Pod ${migration.sourcePod.name} 详情`}
                            onClick={(e) => e.stopPropagation()}
                        >
                            {migration.sourcePod.name}
                        </a>
                    ) : (
                        <span className="font-mono text-xs font-medium text-foreground truncate" title={migration.sourcePod.name}>
                            {migration.sourcePod.name}
                        </span>
                    )}
                </div>
            </TableCell>

            {/* 新 Pod 名称 */}
            <TableCell className="py-2.5">
                <div className="flex flex-col gap-1 max-w-[200px]">
                    {isIgnored ? (
                        <span className="text-xs text-muted-foreground mdm-text-muted">
                            DaemonSet Pod，无需迁移
                        </span>
                    ) : (
                        targetLink ? (
                            <a
                                href={targetLink}
                                target="_blank"
                                rel="noopener noreferrer"
                                className="font-mono text-xs text-muted-foreground truncate hover:text-blue-600 dark:hover:text-blue-400 hover:underline transition-colors"
                                title={`查看 Pod ${migration.targetPod?.name} 详情`}
                                onClick={(e) => e.stopPropagation()}
                            >
                                {migration.targetPod?.name}
                            </a>
                        ) : (
                            <span className="font-mono text-xs text-muted-foreground truncate group-hover:text-foreground transition-colors" title={migration.targetPod?.name}>
                                {migration.targetPod?.name || (
                                    <span className="opacity-30 p-1">...</span>
                                )}
                            </span>
                        )
                    )}
                </div>
            </TableCell>

            {/* 命名空间 */}
            <TableCell className="py-2.5">
                {isIgnored ? (
                    <span className="text-xs text-muted-foreground/30 pl-1">-</span>
                ) : (
                    <Badge variant="outline" className="text-[10px] text-muted-foreground font-normal bg-muted/30 border-border/50 truncate max-w-[120px] px-1.5 h-5" title={migration.sourcePod.namespace}>
                        {migration.targetPod?.namespace || migration.sourcePod.namespace}
                    </Badge>
                )}
            </TableCell>

            {/* Pod 状态 */}
            <TableCell className="py-2.5">
                {isIgnored ? <span className="text-xs text-muted-foreground/30 pl-2">-</span> : (
                    <PhaseBadge
                        phase={migration.targetPod?.phase}
                        phaseReason={migration.targetPod?.phaseReason}
                    />
                )}
            </TableCell>

            {/* 迁移状态 */}
            <TableCell className="py-2.5">
                <StatusBadge status={migration.status} />
            </TableCell>

            {/* 目标节点 */}
            <TableCell className="py-2.5">
                <div className="flex items-center gap-1.5 max-w-[140px]">
                    {isIgnored ? <span className="text-xs text-muted-foreground/30 pl-2">-</span> : migration.targetPod?.nodeName ? (
                        <>
                            <Server className="h-3 w-3 text-muted-foreground/50" />
                            <span className="text-[11px] text-foreground truncate" title={migration.targetPod.nodeName}>
                                {migration.targetPod.nodeName}
                            </span>
                        </>
                    ) : (
                        <span className="text-xs text-muted-foreground/30 pl-1">...</span>
                    )}
                </div>
            </TableCell>

            {/* 信息/时间 */}
            <TableCell className="text-right py-2.5 pr-4">
                <div className="flex justify-end max-w-[180px] ml-auto">
                    {migration.errorMessage ? (
                        <span className="text-[11px] text-red-500 truncate font-medium" title={migration.errorMessage}>
                            {migration.errorMessage}
                        </span>
                    ) : isIgnored ? (
                        <span className="text-xs text-muted-foreground/30">-</span>
                    ) : (
                        <span className="text-[10px] text-muted-foreground font-mono">
                            {migration.targetPod?.createdAt ? formatStartTime(migration.targetPod.createdAt) : ''}
                        </span>
                    )}
                </div>
            </TableCell>
        </TableRow>
    );
};

// --- 迁移表格内容组件 ---
const MigrationsTableContent: React.FC<{ migrations: DrainPodMigrationInfo[]; emptyMessage?: string; clusterName?: string }> = ({
    migrations,
    emptyMessage = "暂无 Pod 迁移数据",
    clusterName
}) => {
    if (migrations.length === 0) {
        return (
            <TableRow>
                <TableCell colSpan={7} className="h-32 text-center">
                    <div className="flex flex-col items-center justify-center text-muted-foreground/50">
                        <Activity className="h-8 w-8 mb-2 opacity-20" />
                        <span className="text-sm">{emptyMessage}</span>
                    </div>
                </TableCell>
            </TableRow>
        );
    }

    return (
        <>
            {migrations.map((mig) => (
                <MigrationRow
                    key={mig.migrationId || `${mig.sourcePod.namespace}-${mig.sourcePod.name}`}
                    migration={mig}
                    clusterName={clusterName}
                />
            ))}
        </>
    );
};

import {
    Pagination,
    PaginationContent,
    PaginationEllipsis,
    PaginationItem,
    PaginationLink,
    PaginationNext,
    PaginationPrevious,
} from "@/components/ui/pagination";

// --- 简易表格版本 (带分页) ---
export const SimpleMigrationsTable: React.FC<{ migrations: DrainPodMigrationInfo[]; clusterName?: string }> = ({ migrations, clusterName }) => {
    const [currentPage, setCurrentPage] = useState(1);
    const pageSize = 10;
    const totalPages = Math.ceil(migrations.length / pageSize);

    const paginatedMigrations = migrations.slice((currentPage - 1) * pageSize, currentPage * pageSize);

    // 重置页码当数据源变化时
    React.useEffect(() => {
        setCurrentPage(1);
    }, [migrations.length]);

    return (
        <div className="space-y-2">
            <div className="rounded-lg border bg-card text-card-foreground shadow-sm overflow-hidden">
                <div className="relative w-full overflow-auto">
                    <Table>
                        <TableHeader className="bg-muted/40">
                            <TableRow className="hover:bg-transparent border-b border-border/50">
                                <TableHead className="w-[180px] h-9 text-xs font-semibold tracking-tight pl-4">原Pod名称</TableHead>
                                <TableHead className="w-[180px] h-9 text-xs font-semibold tracking-tight">新Pod名称</TableHead>
                                <TableHead className="w-[110px] h-9 text-xs font-semibold tracking-tight">命名空间</TableHead>
                                <TableHead className="w-[100px] h-9 text-xs font-semibold tracking-tight">Pod状态</TableHead>
                                <TableHead className="w-[100px] h-9 text-xs font-semibold tracking-tight">迁移状态</TableHead>
                                <TableHead className="w-[140px] h-9 text-xs font-semibold tracking-tight">目标节点</TableHead>
                                <TableHead className="w-[140px] h-9 text-xs font-semibold tracking-tight text-right pr-4">信息</TableHead>
                            </TableRow>
                        </TableHeader>
                        <TableBody>
                            <MigrationsTableContent migrations={paginatedMigrations} clusterName={clusterName} />
                        </TableBody>
                    </Table>
                </div>
            </div>

            {totalPages > 1 && (
                <div className="flex justify-end px-2">
                    <Pagination className="justify-end w-auto">
                        <PaginationContent className="h-8">
                            <PaginationItem>
                                <Button
                                    variant="ghost"
                                    size="icon"
                                    className="h-7 w-7"
                                    onClick={() => setCurrentPage(p => Math.max(1, p - 1))}
                                    disabled={currentPage === 1}
                                >
                                    <ChevronDown className="h-4 w-4 rotate-90" />
                                </Button>
                            </PaginationItem>

                            <div className="text-xs text-muted-foreground mx-2 flex items-center">
                                第 {currentPage} / {totalPages} 页
                            </div>

                            <PaginationItem>
                                <Button
                                    variant="ghost"
                                    size="icon"
                                    className="h-7 w-7"
                                    onClick={() => setCurrentPage(p => Math.min(totalPages, p + 1))}
                                    disabled={currentPage === totalPages}
                                >
                                    <ChevronDown className="h-4 w-4 -rotate-90" />
                                </Button>
                            </PaginationItem>
                        </PaginationContent>
                    </Pagination>
                </div>
            )}
        </div>
    );
};

// --- 接口定义 ---
interface MigrationsTableProps {
    /** 直接传入迁移数据（不使用 Context） */
    migrations?: DrainPodMigrationInfo[];
    /** 预排序的迁移数据（可选，用于替代 Context） */
    sortedMigrations?: SortedMigrations;
    /** 统计数据（可选，用于判断 isAllIgnored 状态） */
    stats?: DrainStats;
    /** 集群名称，用于生成 Pod 跳转链接 */
    clusterName?: string;
}

// --- 主组件 ---
export const MigrationsTable: React.FC<MigrationsTableProps> = ({
    migrations: propMigrations,
    sortedMigrations: propSortedMigrations,
    stats: propStats,
    clusterName
}) => {
    const [showCompleted, setShowCompleted] = useState(false);

    // 如果直接传入 migrations props，使用简化模式（无 Context 依赖）
    if (propMigrations !== undefined) {
        return <SimpleMigrationsTable migrations={propMigrations} clusterName={clusterName} />;
    }

    // Context handling fallback
    let incompleteMigrations: DrainPodMigrationInfo[];
    let completedMigrations: DrainPodMigrationInfo[];
    let stats: DrainStats;

    if (propSortedMigrations) {
        incompleteMigrations = propSortedMigrations.incompleteMigrations;
        completedMigrations = propSortedMigrations.completedMigrations;
        stats = propStats || { totalPods: 0, migratedPods: 0, failedPods: 0, pendingPods: 0, migratingPods: 0, ignoredPods: 0, pdbCount: 0 };
    } else {
        // eslint-disable-next-line @typescript-eslint/no-require-imports
        const { useSortedMigrations, useStats } = require('./DrainContext');
        const contextSorted = useSortedMigrations();
        const contextStats = useStats();
        incompleteMigrations = contextSorted.incompleteMigrations || [];
        completedMigrations = contextSorted.completedMigrations || [];
        stats = contextStats || {};
    }

    const isEmptyState = incompleteMigrations.length === 0 && completedMigrations.length === 0;
    const isAllIgnored = incompleteMigrations.length === 0 &&
        completedMigrations.length === 0 &&
        stats.totalPods > 0 &&
        stats.ignoredPods > 0;

    return (
        <div className="space-y-4">
            {/* 未完成的迁移 */}
            {incompleteMigrations.length > 0 ? (
                <SimpleMigrationsTable migrations={incompleteMigrations} clusterName={clusterName} />
            ) : isAllIgnored ? (
                <div className="rounded-lg border border-dashed p-8 text-center bg-muted/20">
                    <CheckCircle2 className="h-10 w-10 text-emerald-500/50 mx-auto mb-3" />
                    <p className="text-sm font-medium">所有 Pod 均无需驱逐</p>
                    <p className="text-xs text-muted-foreground mt-1">系统 Pod (DaemonSet 等) 已自动忽略 ({stats.ignoredPods} 个)</p>
                </div>
            ) : isEmptyState ? (
                <div className="rounded-lg border border-dashed p-8 text-center bg-muted/20">
                    <CheckCircle2 className="h-10 w-10 text-emerald-500/50 mx-auto mb-3" />
                    <p className="text-sm font-medium">迁移已完成</p>
                </div>
            ) : null}

            {/* 已完成的迁移（可折叠） */}
            {completedMigrations.length > 0 && (
                <div className="pt-2">
                    <Button
                        variant="ghost"
                        size="sm"
                        className="w-full justify-start text-xs font-medium h-9 px-2 text-muted-foreground hover:bg-muted/50 hover:text-foreground mb-2 transition-colors"
                        onClick={() => setShowCompleted(!showCompleted)}
                    >
                        {showCompleted ? <ChevronDown className="h-3.5 w-3.5 mr-2" /> : <ChevronRight className="h-3.5 w-3.5 mr-2" />}
                        已完成的 Pod ({completedMigrations.length})
                    </Button>

                    {showCompleted && (
                        <SimpleMigrationsTable migrations={completedMigrations} clusterName={clusterName} />
                    )}
                </div>
            )}
        </div>
    );
};

export default MigrationsTable;

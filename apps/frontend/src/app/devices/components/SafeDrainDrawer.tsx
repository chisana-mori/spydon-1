'use client'

import React, { useEffect, useRef, useMemo } from 'react'
import { X, Minus, Loader2, CheckCircle2, XCircle, Terminal, Trash2, ChevronRight } from 'lucide-react'
import { motion, AnimatePresence } from 'framer-motion'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Separator } from '@/components/ui/separator'
import { Badge } from '@/components/ui/badge'
import { Progress } from '@/components/ui/progress'
import { Card, CardHeader, CardContent, CardTitle, CardDescription } from '@/components/ui/card'
import { useSafeDrain, ActiveDrain } from '../context/SafeDrainContext'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import type { DrainPodMigrationStatus, DrainLog, DrainStats, DrainPodMigrationInfo } from '@/types/safe-drain'
import { MigrationsTable } from '../../clusters/components/SafeDrain/MigrationsTable'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { DrainStatsDisplay } from '../../clusters/components/SafeDrain/DrainStatsView'

// 辅助函数：normalize 状态到小写
const normalizeStatus = (status: string) => (status || '').toLowerCase()

// 计算单个 drain 的 stats
const calculateDrainStats = (migrations: DrainPodMigrationInfo[], backendStats?: DrainStats): DrainStats => {
    const total = migrations.length
    const migrated = migrations.filter(m => normalizeStatus(m.status as string) === 'completed').length
    const failed = migrations.filter(m => {
        const s = normalizeStatus(m.status as string)
        return s === 'failed' || s === 'timeout'
    }).length
    const pending = migrations.filter(m => normalizeStatus(m.status as string) === 'pending').length
    const migrating = migrations.filter(m => {
        const s = normalizeStatus(m.status as string)
        return s === 'evicting' || s === 'evicted' || s === 'creating'
    }).length
    const ignored = migrations.filter(m => normalizeStatus(m.status as string) === 'ignored').length

    return {
        totalPods: total,
        migratedPods: migrated,
        failedPods: failed,
        pendingPods: pending,
        migratingPods: migrating,
        ignoredPods: ignored,
        pdbCount: backendStats?.pdbCount ?? 0
    }
}

// 计算进度
const calculateProgress = (stats: DrainStats, drainStatus: string, backendProgress: number): number => {
    const total = stats.totalPods
    if (total === 0) return backendProgress

    const isTerminal = ['completed', 'failed', 'cancelled'].includes(drainStatus)
    const completedCount = stats.migratedPods + stats.ignoredPods
    const failedCount = stats.failedPods

    const done = isTerminal ? (completedCount + failedCount) : completedCount

    return Math.round((done / total) * 100)
}

export function SafeDrainDrawer() {
    const { activeDrains, isDrawerOpen, minimizeDrawer, closeDrawer, clearCompleted, removeDrain, requestCancel } = useSafeDrain()

    // 自动滚动日志到底部
    // 实际上因为可能有多个 drain，自动滚动有点复杂。
    // 我们只针对每个 drain item 内部的 logs 区域做处理，
    // 或者只让他显示最新的几条?

    // 如果没有正在进行的任务且所有任务都完成了/失败了，是否允许关闭？
    // 暂时由用户手动关闭

    return (
        <AnimatePresence>
            {isDrawerOpen && (
                <>
                    {/* Backdorp - 点击背景最小化而不是关闭，或者不做 backdrop */}
                    <motion.div
                        initial={{ opacity: 0 }}
                        animate={{ opacity: 1 }}
                        exit={{ opacity: 0 }}
                        onClick={minimizeDrawer} // 点击背景最小化
                        className="fixed inset-0 bg-black/20 backdrop-blur-sm z-40"
                    />

                    {/* Drawer */}
                    <motion.div
                        initial={{ x: '100%' }}
                        animate={{ x: 0 }}
                        exit={{ x: '100%' }}
                        transition={{ type: 'spring', damping: 25, stiffness: 200 }}
                        className="fixed right-0 top-0 bottom-0 w-[calc(100vw-260px)] bg-background border-l border-border shadow-2xl z-50 flex flex-col"
                    >
                        {/* Header */}
                        <div className="flex items-center justify-between p-4 border-b border-border bg-muted/30">
                            <div className="flex items-center gap-2">
                                <Terminal className="h-5 w-5 text-primary" />
                                <h2 className="font-semibold text-base">Safe Drain 任务</h2>
                                <Badge variant="secondary" className="ml-2">
                                    {activeDrains.length}
                                </Badge>
                            </div>
                            <div className="flex items-center gap-1">
                                <Button variant="ghost" size="icon" className="h-8 w-8" onClick={minimizeDrawer} title="最小化">
                                    <Minus className="h-4 w-4" />
                                </Button>
                                <Button variant="ghost" size="icon" className="h-8 w-8" onClick={closeDrawer} title="关闭">
                                    <X className="h-4 w-4" />
                                </Button>
                            </div>
                        </div>

                        {/* Content */}
                        <ScrollArea className="flex-1 p-4">
                            <div className="space-y-4">
                                <div className="space-y-4">
                                    {activeDrains.length === 0 ? (
                                        <div className="flex flex-col items-center justify-center h-64 text-muted-foreground">
                                            <CheckCircle2 className="h-12 w-12 mb-4 opacity-20" />
                                            <p>暂无活动任务</p>
                                        </div>
                                    ) : (
                                        activeDrains.length === 1 ? (
                                            <DrainItem
                                                drain={activeDrains[0]}
                                                onRemove={() => removeDrain(activeDrains[0].drainID)}
                                                onRequestCancel={() => requestCancel(activeDrains[0].drainID)}
                                            />
                                        ) : (
                                            <Tabs defaultValue="summary" className="w-full">
                                                <TabsList className="w-full justify-start h-auto flex-wrap gap-1 bg-transparent p-0 mb-4">
                                                    <TabsTrigger
                                                        value="summary"
                                                        className="data-[state=active]:bg-primary data-[state=active]:text-primary-foreground data-[state=active]:shadow-sm"
                                                    >
                                                        全量汇总
                                                    </TabsTrigger>
                                                    {activeDrains.map(drain => (
                                                        <TabsTrigger
                                                            key={drain.drainID}
                                                            value={drain.drainID}
                                                            className="data-[state=active]:bg-primary data-[state=active]:text-primary-foreground data-[state=active]:shadow-sm"
                                                        >
                                                            {drain.nodeName}
                                                            {drain.status === 'running' && <Loader2 className="ml-1 h-3 w-3 animate-spin" />}
                                                        </TabsTrigger>
                                                    ))}
                                                </TabsList>

                                                <TabsContent value="summary" className="mt-0">
                                                    <DrainSummary
                                                        drains={activeDrains}
                                                        onClearCompleted={clearCompleted}
                                                    />
                                                </TabsContent>

                                                {activeDrains.map(drain => (
                                                    <TabsContent key={drain.drainID} value={drain.drainID} className="mt-0">
                                                        <DrainItem
                                                            drain={drain}
                                                            onRemove={() => removeDrain(drain.drainID)}
                                                            onRequestCancel={() => requestCancel(drain.drainID)}
                                                        />
                                                    </TabsContent>
                                                ))}
                                            </Tabs>
                                        )
                                    )}
                                </div>
                            </div>
                        </ScrollArea>

                        {/* Footer */}
                        <div className="p-4 border-t border-border bg-muted/10 flex justify-between items-center">
                            <span className="text-xs text-muted-foreground">
                                {activeDrains.filter(d => d.status === 'running').length} 个任务进行中
                            </span>
                            {activeDrains.some(d => d.status !== 'running') && (
                                <Button variant="outline" size="sm" onClick={clearCompleted} className="text-xs h-7">
                                    <Trash2 className="h-3 w-3 mr-2" />
                                    清理已完成
                                </Button>
                            )}
                        </div>
                    </motion.div>
                </>
            )}
        </AnimatePresence>
    )
}

function DrainItem({ drain, onRemove, onRequestCancel }: { drain: ActiveDrain, onRemove: () => void, onRequestCancel: () => void }) {
    const isCompleted = drain.status === 'completed'
    const isFailed = drain.status === 'failed'

    // Process logs: reverse order to show newest at top, limit to 50
    const sortedLogs = [...drain.logs].reverse().slice(0, 50)

    // 辅助函数：normalize 状态到小写
    const normalizeStatus = (status: string) => (status || '').toLowerCase()

    // 从 migrations 数组直接计算 stats，确保与表格数据同步
    const computedStats = useMemo((): DrainStats => {
        const migrations = drain.migrations
        const total = migrations.length
        const migrated = migrations.filter(m => normalizeStatus(m.status as string) === 'completed').length
        const failed = migrations.filter(m => {
            const s = normalizeStatus(m.status as string)
            return s === 'failed' || s === 'timeout'
        }).length
        const pending = migrations.filter(m => normalizeStatus(m.status as string) === 'pending').length
        const migrating = migrations.filter(m => {
            const s = normalizeStatus(m.status as string)
            return s === 'evicting' || s === 'evicted' || s === 'creating'
        }).length
        const ignored = migrations.filter(m => normalizeStatus(m.status as string) === 'ignored').length

        return {
            totalPods: total,
            migratedPods: migrated,
            failedPods: failed,
            pendingPods: pending,
            migratingPods: migrating,
            ignoredPods: ignored,
            pdbCount: drain.stats?.pdbCount ?? 0
        }
    }, [drain.migrations, drain.stats?.pdbCount])

    // 基于 migrations 计算真实进度（如果 migrations 存在）
    const computedProgress = useMemo(() => {
        const total = computedStats.totalPods
        if (total === 0) return drain.progress // 如果没有 migrations 数据，使用后端的 progress

        // 逻辑与 Navy 保持一致：
        // 如果任务结束（completed/failed/cancelled），失败也算入进度条（Done）；否则只计算完成+忽略
        const isTerminal = isCompleted || isFailed || drain.status === 'cancelled'

        const completedCount = computedStats.migratedPods + computedStats.ignoredPods
        const failedCount = computedStats.failedPods

        const done = isTerminal ? (completedCount + failedCount) : completedCount

        return Math.round((done / total) * 100)
    }, [computedStats, drain.progress, isCompleted, isFailed, drain.status])

    // 显示进度：优先使用计算的进度（如果有 migrations），否则使用后端推送的进度
    const displayProgress = drain.migrations.length > 0 ? computedProgress : drain.progress

    // Filter migrations for different views
    // 1. In-progress / Pending / Failed (Tracking) - 未完成的
    const activeMigrations = drain.migrations.filter(m => {
        const s = normalizeStatus(m.status as string)
        return ['pending', 'evicting', 'evicted', 'creating', 'failed', 'timeout'].includes(s)
    })

    // 2. Completed / Ready / Ignored (终态)
    const completedMigrations = drain.migrations.filter(m => {
        const s = normalizeStatus(m.status as string)
        return s === 'completed' || s === 'ignored'
    })

    return (
        <Card className={cn(
            "transition-all shadow-sm",
            isCompleted && "border-emerald-200 dark:border-emerald-900/50",
            isFailed && "border-red-200 dark:border-red-900/50",
            drain.status === 'running' && "border-blue-200 dark:border-blue-900/50"
        )}>
            <CardHeader className="pb-3 bg-muted/5 flex flex-row items-start justify-between space-y-0">
                <div className="space-y-1">
                    <CardTitle className="text-base font-medium flex items-center gap-2">
                        {drain.nodeName}
                        <Badge variant="outline" className="font-normal text-xs text-muted-foreground">
                            {drain.clusterName}
                        </Badge>
                    </CardTitle>
                    <div className="flex items-center gap-2 text-xs text-muted-foreground">
                        {drain.status === 'running' && <Loader2 className="h-3.5 w-3.5 animate-spin text-blue-500" />}
                        {isCompleted && <CheckCircle2 className="h-3.5 w-3.5 text-emerald-500" />}
                        {isFailed && <XCircle className="h-3.5 w-3.5 text-red-500" />}
                        <span>{drain.currentStep}</span>
                    </div>
                </div>

                {/* Actions */}
                <div className="flex items-center gap-2">
                    {drain.status === 'running' ? (
                        <Button variant="destructive" size="sm" className="h-7 text-xs" onClick={onRequestCancel}>
                            取消任务
                        </Button>
                    ) : ((isCompleted || isFailed || drain.status === 'cancelled') && (
                        <Button variant="ghost" size="icon" className="h-7 w-7 text-muted-foreground hover:text-destructive" onClick={onRemove}>
                            <X className="h-4 w-4" />
                        </Button>
                    ))}
                </div>
            </CardHeader>

            <CardContent className="space-y-6 pt-6">
                {/* Progress Bar */}
                <div className="space-y-2">
                    <div className="flex justify-between text-xs text-muted-foreground">
                        <span>总体进度</span>
                        <span className="font-mono">{displayProgress}%</span>
                    </div>
                    <Progress value={displayProgress} className={cn(
                        "h-2",
                        (isCompleted || displayProgress >= 100) && "[&>div]:bg-emerald-500",
                        isFailed && "[&>div]:bg-red-500"
                    )} />
                </div>

                {/* Logs Section - Moved below Progress, Taller, Reverse Order */}
                <div className="space-y-2">
                    <div className="flex items-center justify-between">
                        <h4 className="text-sm font-medium flex items-center gap-2">
                            <Terminal className="h-4 w-4 text-muted-foreground" />
                            实时日志
                        </h4>
                        <span className="text-xs text-muted-foreground">{drain.logs.length} 条记录</span>
                    </div>
                    <div className="rounded-lg border bg-zinc-950 text-zinc-300 font-mono text-xs overflow-hidden shadow-inner">
                        <ScrollArea className="h-[300px] w-full p-4">
                            <div className="space-y-1.5">
                                {sortedLogs.length === 0 ? (
                                    <div className="text-zinc-600 italic text-center py-8">暂无日志...</div>
                                ) : (
                                    sortedLogs.map((log, i) => (
                                        <div key={i} className="flex gap-2.5 hover:bg-white/5 py-0.5 rounded px-1 -mx-1 transition-colors">
                                            <span className="text-zinc-600 shrink-0 select-none w-[70px]">
                                                {new Date(log.timestamp).toLocaleTimeString()}
                                            </span>
                                            <span className={cn(
                                                'shrink-0 w-14 font-bold select-none',
                                                log.level === 'ERROR' ? 'text-red-400' :
                                                    log.level === 'WARN' ? 'text-yellow-400' :
                                                        log.level === 'SUCCESS' ? 'text-emerald-400' : 'text-blue-400'
                                            )}>
                                                {log.level}
                                            </span>
                                            <span className="break-all whitespace-pre-wrap flex-1 opacity-90">{log.message}</span>
                                        </div>
                                    ))
                                )}
                            </div>
                        </ScrollArea>
                    </div>
                </div>

                <Separator />

                {/* Stats & Table - 使用计算的 stats 确保与表格同步 */}
                <div className="space-y-4">
                    {computedStats.totalPods > 0 && <DrainStatsDisplay stats={computedStats} />}

                    {/* Active/Tracking Table */}
                    <div className="space-y-2">
                        <div className="flex items-center gap-2">
                            <Badge variant="outline" className="text-xs bg-muted/50">
                                追踪中 ({activeMigrations.length})
                            </Badge>
                        </div>
                        <MigrationsTable migrations={activeMigrations} clusterName={drain.clusterName} />
                    </div>

                    {/* Completed Table */}
                    {completedMigrations.length > 0 && (
                        <div className="space-y-2 pt-2">
                            <div className="flex items-center gap-2">
                                <Badge variant="outline" className="text-xs bg-emerald-50 text-emerald-700 border-emerald-200 dark:bg-emerald-950/30 dark:text-emerald-400 dark:border-emerald-900/50">
                                    迁移完成 / Ready ({completedMigrations.length})
                                </Badge>
                            </div>
                            <MigrationsTable migrations={completedMigrations} clusterName={drain.clusterName} />
                        </div>
                    )}
                </div>
            </CardContent>
        </Card>
    )
}

function DrainSummary({ drains, onClearCompleted }: { drains: ActiveDrain[], onClearCompleted: () => void }) {
    // 汇总计算
    const summary = useMemo(() => {
        const allMigrations = drains.flatMap(d => d.migrations)

        // 汇总日志
        const allLogs = drains.flatMap(d => d.logs.map(log => ({ ...log, nodeName: d.nodeName })))
        const sortedLogs = allLogs.sort((a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime()).slice(0, 50)

        // 汇总 stats
        const aggregatedStats = drains.reduce((acc, drain) => {
            const stats = calculateDrainStats(drain.migrations, drain.stats)
            return {
                totalPods: acc.totalPods + stats.totalPods,
                migratedPods: acc.migratedPods + stats.migratedPods,
                failedPods: acc.failedPods + stats.failedPods,
                pendingPods: acc.pendingPods + stats.pendingPods,
                migratingPods: acc.migratingPods + stats.migratingPods,
                ignoredPods: acc.ignoredPods + stats.ignoredPods,
                pdbCount: acc.pdbCount + stats.pdbCount
            }
        }, {
            totalPods: 0, migratedPods: 0, failedPods: 0, pendingPods: 0, migratingPods: 0, ignoredPods: 0, pdbCount: 0
        })

        let totalDone = 0
        let totalTotal = 0

        drains.forEach(drain => {
            const stats = calculateDrainStats(drain.migrations, drain.stats)
            const isTerminal = ['completed', 'failed', 'cancelled'].includes(drain.status)
            const completedCount = stats.migratedPods + stats.ignoredPods
            const failedCount = stats.failedPods
            const done = isTerminal ? (completedCount + failedCount) : completedCount

            totalDone += done
            totalTotal += stats.totalPods
        })

        const progress = totalTotal === 0 ? 0 : Math.round((totalDone / totalTotal) * 100)

        // Filter active migrations for the table
        const activeMigrations = allMigrations.filter(m => {
            const s = normalizeStatus(m.status as string)
            return ['pending', 'evicting', 'evicted', 'creating', 'failed', 'timeout'].includes(s)
        })

        // Filter completed migrations for the table
        const completedMigrations = allMigrations.filter(m => {
            const s = normalizeStatus(m.status as string)
            return s === 'completed' || s === 'ignored'
        })

        return { stats: aggregatedStats, progress, activeMigrations, completedMigrations, allMigrations, sortedLogs }
    }, [drains])

    const isAllCompleted = drains.every(d => d.status !== 'running')

    return (
        <Card className="transition-all shadow-sm border-blue-200 dark:border-blue-900/50">
            <CardHeader className="pb-3 bg-muted/5 flex flex-row items-start justify-between space-y-0">
                <div className="space-y-1">
                    <CardTitle className="text-base font-medium flex items-center gap-2">
                        任务汇总
                        <Badge variant="outline" className="font-normal text-xs text-muted-foreground">
                            共 {drains.length} 个节点
                        </Badge>
                    </CardTitle>
                    <CardDescription>
                        {isAllCompleted ? "所有节点驱逐任务已结束" : "正在进行多节点驱逐任务"}
                    </CardDescription>
                </div>

                {drains.some(d => d.status !== 'running') && (
                    <Button variant="outline" size="sm" onClick={onClearCompleted} className="text-xs h-7">
                        <Trash2 className="h-3 w-3 mr-2" />
                        清理已完成
                    </Button>
                )}
            </CardHeader>

            <CardContent className="space-y-6 pt-6">
                {/* Progress Bar */}
                <div className="space-y-2">
                    <div className="flex justify-between text-xs text-muted-foreground">
                        <span>总体进度</span>
                        <span className="font-mono">{summary.progress}%</span>
                    </div>
                    <Progress value={summary.progress} className={cn(
                        "h-2",
                        summary.progress >= 100 && "[&>div]:bg-emerald-500"
                    )} />
                </div>

                {/* Aggregated Logs Section */}
                <div className="space-y-2">
                    <div className="flex items-center justify-between">
                        <h4 className="text-sm font-medium flex items-center gap-2">
                            <Terminal className="h-4 w-4 text-muted-foreground" />
                            实时汇总日志
                        </h4>
                        <span className="text-xs text-muted-foreground">显示最近50条</span>
                    </div>
                    <div className="rounded-lg border bg-zinc-950 text-zinc-300 font-mono text-xs overflow-hidden shadow-inner">
                        <ScrollArea className="h-[300px] w-full p-4">
                            <div className="space-y-1.5">
                                {summary.sortedLogs.length === 0 ? (
                                    <div className="text-zinc-600 italic text-center py-8">暂无日志...</div>
                                ) : (
                                    summary.sortedLogs.map((log, i) => (
                                        <div key={i} className="flex gap-2.5 hover:bg-white/5 py-0.5 rounded px-1 -mx-1 transition-colors">
                                            <span className="text-zinc-600 shrink-0 select-none w-[70px]">
                                                {new Date(log.timestamp).toLocaleTimeString()}
                                            </span>
                                            <Badge variant="outline" className="shrink-0 h-5 px-1 text-[10px] border-zinc-700 text-zinc-400 bg-zinc-900/50">
                                                {log.nodeName}
                                            </Badge>
                                            <span className={cn(
                                                'shrink-0 w-14 font-bold select-none',
                                                log.level === 'ERROR' ? 'text-red-400' :
                                                    log.level === 'WARN' ? 'text-yellow-400' :
                                                        log.level === 'SUCCESS' ? 'text-emerald-400' : 'text-blue-400'
                                            )}>
                                                {log.level}
                                            </span>
                                            <span className="break-all whitespace-pre-wrap flex-1 opacity-90">{log.message}</span>
                                        </div>
                                    ))
                                )}
                            </div>
                        </ScrollArea>
                    </div>
                </div>

                <Separator />

                {/* Stats */}
                <div className="space-y-4">
                    <DrainStatsDisplay stats={summary.stats} />


                    {/* Migrations Table (All Active) */}
                    <div className="space-y-2">
                        <div className="flex items-center gap-2">
                            <Badge variant="outline" className="text-xs bg-muted/50">
                                活动中的 Pod ({summary.activeMigrations.length})
                            </Badge>
                        </div>
                        <div className="rounded-md border">
                            <MigrationsTable migrations={summary.activeMigrations} clusterName="All" />
                        </div>
                    </div>

                    {/* Migrations Table (Completed) */}
                    {summary.completedMigrations.length > 0 && (
                        <div className="space-y-2 pt-2">
                            <div className="flex items-center gap-2">
                                <Badge variant="outline" className="text-xs bg-emerald-50 text-emerald-700 border-emerald-200 dark:bg-emerald-950/30 dark:text-emerald-400 dark:border-emerald-900/50">
                                    迁移完成 / Ready ({summary.completedMigrations.length})
                                </Badge>
                            </div>
                            <div className="rounded-md border">
                                <MigrationsTable migrations={summary.completedMigrations} clusterName="All" />
                            </div>
                        </div>
                    )}
                </div>
            </CardContent>
        </Card>
    )
}

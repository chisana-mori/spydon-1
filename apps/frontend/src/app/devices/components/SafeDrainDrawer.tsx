'use client'

import React, { useEffect, useRef } from 'react'
import { X, Minus, Loader2, CheckCircle2, XCircle, Terminal, Trash2, ChevronRight } from 'lucide-react'
import { motion, AnimatePresence } from 'framer-motion'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Separator } from '@/components/ui/separator'
import { Badge } from '@/components/ui/badge'
import { Progress } from '@/components/ui/progress'
import { useSafeDrain, ActiveDrain } from '../context/SafeDrainContext'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import type { DrainPodMigrationStatus, DrainLog } from '@/types/safe-drain'
import { MigrationsTable } from '../../clusters/components/SafeDrain/MigrationsTable'
import { DrainStatsDisplay } from '../../clusters/components/SafeDrain/DrainStatsView'

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
                                {activeDrains.length === 0 ? (
                                    <div className="flex flex-col items-center justify-center h-64 text-muted-foreground">
                                        <CheckCircle2 className="h-12 w-12 mb-4 opacity-20" />
                                        <p>暂无活动任务</p>
                                    </div>
                                ) : (
                                    activeDrains.map(drain => (
                                        <DrainItem
                                            key={drain.drainID}
                                            drain={drain}
                                            onRemove={() => removeDrain(drain.drainID)}
                                            onRequestCancel={() => requestCancel(drain.drainID)}
                                        />
                                    ))
                                )}
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
    const logsEndRef = useRef<HTMLDivElement>(null)

    useEffect(() => {
        logsEndRef.current?.scrollIntoView({ behavior: 'smooth' })
    }, [drain.logs])

    return (
        <div className={cn(
            "rounded-lg border border-border bg-card overflow-hidden transition-all",
            isCompleted && "border-emerald-200 dark:border-emerald-900",
            isFailed && "border-red-200 dark:border-red-900"
        )}>
            {/* Item Header */}
            <div className="p-3 bg-muted/30 flex items-center justify-between">
                <div className="flex flex-col gap-0.5">
                    <div className="flex items-center gap-2">
                        <span className="font-medium text-sm">{drain.nodeName}</span>
                        <span className="text-[10px] px-1.5 py-0.5 rounded-full bg-muted text-muted-foreground border border-border/50">
                            {drain.clusterName}
                        </span>
                    </div>
                    <span className="text-xs text-muted-foreground flex items-center gap-1.5">
                        {drain.status === 'running' && <Loader2 className="h-3 w-3 animate-spin text-blue-500" />}
                        {isCompleted && <CheckCircle2 className="h-3 w-3 text-emerald-500" />}
                        {isFailed && <XCircle className="h-3 w-3 text-red-500" />}
                        {drain.currentStep}
                    </span>
                </div>
                {/* 操作按钮：运行中可取消；完成/失败可移除 */}
                {drain.status === 'running' ? (
                    <Button variant="destructive" size="sm" className="h-7 text-xs" onClick={onRequestCancel}>
                        取消任务
                    </Button>
                ) : ((isCompleted || isFailed || drain.status === 'cancelled') && (
                    <Button variant="ghost" size="icon" className="h-6 w-6 text-muted-foreground hover:text-destructive" onClick={onRemove}>
                        <X className="h-3 w-3" />
                    </Button>
                ))}
            </div>

            <div className="px-3 pt-2 pb-3 space-y-3">
                {/* Progress Bar */}
                <div className="flex items-center gap-3">
                    <Progress value={drain.progress} className={cn(
                        "h-1.5",
                        isCompleted && "bg-emerald-100 dark:bg-emerald-950 [&>div]:bg-emerald-500",
                        isFailed && "bg-red-100 dark:bg-red-950 [&>div]:bg-red-500"
                    )} />
                    <span className="text-xs font-mono w-8 text-right">{drain.progress}%</span>
                </div>

                {/* Migrations table */}
                {drain.stats && <DrainStatsDisplay stats={drain.stats} />}
                <MigrationsTable migrations={drain.migrations} />

                {/* Logs (aligned with auto-navy LogViewer style) */}
                <div className="border rounded-md bg-zinc-950 p-3 h-[150px] text-xs text-zinc-300 font-mono">
                    <ScrollArea className="h-full">
                        <div className="space-y-1">
                            {drain.logs.map((log, i) => (
                                <div key={i} className="flex gap-2">
                                    <span className="text-zinc-500 shrink-0">{new Date(log.timestamp).toLocaleTimeString()}</span>
                                    <span className={cn(
                                        'shrink-0 w-12 font-bold',
                                        log.level === 'ERROR' ? 'text-red-500' :
                                            log.level === 'WARN' ? 'text-yellow-500' :
                                                log.level === 'SUCCESS' ? 'text-emerald-500' : 'text-blue-500'
                                    )}>
                                        [{log.level}]
                                    </span>
                                    <span className="break-all whitespace-pre-wrap">{log.message}</span>
                                </div>
                            ))}
                            <div ref={logsEndRef} />
                        </div>
                    </ScrollArea>
                </div>
            </div>
        </div>
    )
}

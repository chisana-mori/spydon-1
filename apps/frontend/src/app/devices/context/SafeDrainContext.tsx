'use client'

import React, { createContext, useContext, useState, useCallback, useEffect, useRef } from 'react'
import { toast } from 'sonner'
import { appConfig } from '@/config'
import type { DrainPodMigrationInfo } from '@/types/safe-drain'

// Drain 状态定义（与 auto-navy 对齐）
export type DrainStatus = 'pending' | 'running' | 'completed' | 'failed' | 'cancelled'

export interface DrainLog {
    timestamp: string
    level: 'INFO' | 'WARN' | 'ERROR' | 'SUCCESS'
    message: string
}

export interface ActiveDrain {
    drainID: string
    nodeName: string
    clusterName: string
    status: DrainStatus
    progress: number
    currentStep: string // 当前正在进行的操作描述
    logs: DrainLog[]
    startTime: number
    migrations: DrainPodMigrationInfo[]
    stats?: Stats
}

// 迁移统计
export interface Stats {
    totalPods: number
    migratedPods: number
    failedPods: number
    pendingPods: number
    migratingPods: number
    ignoredPods: number
}

interface SafeDrainContextType {
    activeDrains: ActiveDrain[]
    isDrawerOpen: boolean
    isMinimized: boolean
    addDrain: (drainID: string, nodeName: string, clusterName: string) => void
    removeDrain: (drainID: string) => void
    requestCancel: (drainID: string) => Promise<void>
    minimizeDrawer: () => void
    restoreDrawer: () => void
    closeDrawer: () => void
    clearCompleted: () => void
}

const SafeDrainContext = createContext<SafeDrainContextType | null>(null)

export function useSafeDrain() {
    const context = useContext(SafeDrainContext)
    if (!context) {
        throw new Error('useSafeDrain must be used within a SafeDrainProvider')
    }
    return context
}

export function SafeDrainProvider({ children }: { children: React.ReactNode }) {
    const [activeDrains, setActiveDrains] = useState<ActiveDrain[]>([])
    const [isDrawerOpen, setIsDrawerOpen] = useState(false)
    const [isMinimized, setIsMinimized] = useState(false)

    // SSE 连接管理 ref，避免重连
    const eventSourcesRef = useRef<Map<string, EventSource>>(new Map())

    // 添加新的 Drain 任务
    const addDrain = useCallback((drainID: string, nodeName: string, clusterName: string) => {
        setIsDrawerOpen(true)
        setIsMinimized(false)

        setActiveDrains(prev => {
            // 避免重复添加
            if (prev.some(d => d.drainID === drainID)) return prev
            return [...prev, {
                drainID,
                nodeName,
                clusterName,
                status: 'running',
                progress: 0,
                currentStep: '初始化...',
                logs: [],
                startTime: Date.now(),
                migrations: [],
                stats: { totalPods: 0, migratedPods: 0, failedPods: 0, pendingPods: 0, migratingPods: 0, ignoredPods: 0 }
            }]
        })

        // 建立 SSE 连接
        connectToDrainStream(drainID)
    }, [])

    const removeDrain = useCallback((drainID: string) => {
        setActiveDrains(prev => prev.filter(d => d.drainID !== drainID))
        cleanupEventSource(drainID)
    }, [])

    const requestCancel = useCallback(async (drainID: string) => {
        try {
            const res = await fetch(`${appConfig.apiBaseUrl}/navy/drain/${drainID}/cancel`, { method: 'POST', credentials: 'include' })
            if (!res.ok) {
                const err = await res.text().catch(() => '')
                throw new Error(err || `Cancel request failed (${res.status})`)
            }
            // 记录日志并标记取消（最终状态也会通过 SSE 同步过来）
            setActiveDrains(prev => prev.map(d => d.drainID === drainID ? {
                ...d,
                status: d.status === 'completed' || d.status === 'failed' ? d.status : 'cancelled',
                logs: [...d.logs, { timestamp: new Date().toISOString(), level: 'WARN', message: 'Cancellation requested...' }]
            } : d))
        } catch (e: any) {
            toast.error(`取消失败: ${e.message || e}`)
        }
    }, [])

    const minimizeDrawer = useCallback(() => {
        setIsMinimized(true)
        setIsDrawerOpen(false)
    }, [])

    const restoreDrawer = useCallback(() => {
        setIsMinimized(false)
        setIsDrawerOpen(true)
    }, [])

    const closeDrawer = useCallback(() => {
        setIsDrawerOpen(false)
        setIsMinimized(false)
        // 注意：关闭抽屉并不意味着取消任务，任务可能还在后台运行
        // 如果需要取消任务，应该有专门的取消操作
    }, [])

    const clearCompleted = useCallback(() => {
        setActiveDrains(prev => {
            const running = prev.filter(d => d.status === 'running')
            // 清理已完成任务的 SSE
            prev.filter(d => d.status !== 'running').forEach(d => cleanupEventSource(d.drainID))
            return running
        })
    }, [])

    // 清理单个 SSE
    const cleanupEventSource = (drainID: string) => {
        if (eventSourcesRef.current.has(drainID)) {
            eventSourcesRef.current.get(drainID)?.close()
            eventSourcesRef.current.delete(drainID)
        }
    }

    // 连接 SSE
    const connectToDrainStream = (drainID: string) => {
        if (eventSourcesRef.current.has(drainID)) return

        const url = `${appConfig.apiBaseUrl}/navy/drain/${drainID}/events`
        const es = new EventSource(url, { withCredentials: true })

        es.onmessage = (event) => {
            try {
                // 后端 SSEMessage 结构: { type, message, data, timestamp }
                const payload = JSON.parse(event.data)
                handleDrainEvent(drainID, payload)
            } catch (err) {
                console.error('Failed to parse SSE message', err)
            }
        }

        es.onerror = (err) => {
            console.error(`SSE error for drain ${drainID}`, err)
            es.close()
            eventSourcesRef.current.delete(drainID)
        }

        eventSourcesRef.current.set(drainID, es)
    }

    // 处理事件（适配后端 SSEMessage: { type, message, data, timestamp }）
    const handleDrainEvent = (drainID: string, event: any) => {
        setActiveDrains(prev => prev.map(drain => {
            if (drain.drainID !== drainID) return drain

            const updated = { ...drain }
            const type = event?.type
            const msg = event?.message ?? ''
            const ts = event?.timestamp ?? new Date().toISOString()
            const data = event?.data ?? {}

            switch (type) {
                case 'started': {
                    updated.status = 'running'
                    updated.logs = [...updated.logs, { timestamp: ts, level: 'INFO', message: msg || 'Drain started' }]
                    break
                }
                case 'log': {
                    const level = data?.level || 'INFO'
                    updated.logs = [...updated.logs, {
                        timestamp: ts,
                        level,
                        message: msg,
                    }]
                    // 特殊处理：后端用于 bootstrap 的提示"Drain not found or already completed and cleaned up"
                    if (typeof msg === 'string' && msg.includes('Drain not found or already completed')) {
                        // 对于这种情况，后续不会再有事件，直接结束该任务以免一直停留在“初始化”
                        updated.status = 'completed'
                        updated.progress = 100
                        updated.currentStep = 'Drain completed'
                        cleanupEventSource(drainID)
                    }
                    break
                }
                case 'progress': {
                    const pct = Number(data?.progress ?? data?.percent ?? 0)
                    updated.progress = isNaN(pct) ? 0 : pct
                    updated.currentStep = msg || updated.currentStep
                    if (updated.progress >= 100) {
                        updated.status = 'completed'
                        cleanupEventSource(drainID)
                    }
                    break
                }
                case 'completed': {
                    updated.status = 'completed'
                    updated.progress = 100
                    updated.currentStep = msg || 'Drain completed'
                    updated.logs = [...updated.logs, { timestamp: ts, level: 'SUCCESS', message: updated.currentStep }]
                    cleanupEventSource(drainID)
                    break
                }
                case 'cancelled': {
                    updated.status = 'cancelled'
                    updated.currentStep = msg || 'Drain cancelled'
                    updated.logs = [...updated.logs, { timestamp: ts, level: 'WARN', message: updated.currentStep }]
                    cleanupEventSource(drainID)
                    break
                }
                case 'failed': {
                    updated.status = 'failed'
                    updated.currentStep = msg || 'Drain failed'
                    updated.logs = [...updated.logs, { timestamp: ts, level: 'ERROR', message: updated.currentStep }]
                    cleanupEventSource(drainID)
                    break
                }
                case 'error': {
                    const errMsg = msg || 'Unknown error'
                    updated.status = 'failed'
                    updated.currentStep = `错误: ${errMsg}`
                    updated.logs = [...updated.logs, {
                        timestamp: ts,
                        level: 'ERROR',
                        message: errMsg,
                    }]
                    cleanupEventSource(drainID)
                    break
                }
                case 'migration_update': {
                    const mig = data?.migration as DrainPodMigrationInfo | undefined
                    if (mig) {
                        upsertMigration(updated, mig)
                        computeStats(updated)
                    }
                    break
                }
                case 'pod_eviction_started':
                case 'pod_eviction_succeeded':
                case 'pod_eviction_failed':
                case 'pod_migration_ignored': {
                    const mig = data?.migration as DrainPodMigrationInfo | undefined
                    if (mig) {
                        upsertMigration(updated, mig)
                    }
                    const level = (type === 'pod_eviction_failed') ? 'ERROR' : 'INFO'
                    updated.logs = [...updated.logs, { timestamp: ts, level: level as any, message: msg || type }]
                    computeStats(updated)
                    break
                }
                default: {
                    // 未知事件，忽略
                    break
                }
            }

            return updated
        }))
    }

    // 更新/插入迁移记录
    const upsertMigration = (drain: ActiveDrain, mig: DrainPodMigrationInfo) => {
        const idx = drain.migrations.findIndex(m => m.migrationId === mig.migrationId)
        if (idx >= 0) {
            const copy = [...drain.migrations]
            copy[idx] = { ...copy[idx], ...mig }
            drain.migrations = copy
        } else {
            drain.migrations = [...drain.migrations, mig]
        }
    }

    // 计算统计
    const computeStats = (drain: ActiveDrain) => {
        const total = drain.migrations.length
        const migrated = drain.migrations.filter(m => m.status === 'completed' || m.status === 'evicted').length
        const failed = drain.migrations.filter(m => m.status === 'failed').length
        const pending = drain.migrations.filter(m => m.status === 'pending').length
        const migrating = drain.migrations.filter(m => m.status === 'evicting' || m.status === 'creating').length
        const ignored = drain.migrations.filter(m => m.status === 'ignored').length
        drain.stats = { totalPods: total, migratedPods: migrated, failedPods: failed, pendingPods: pending, migratingPods: migrating, ignoredPods: ignored }
    }

    // 组件卸载时清理所有连接
    useEffect(() => {
        return () => {
            eventSourcesRef.current.forEach(es => es.close())
            eventSourcesRef.current.clear()
        }
    }, [])

    return (
        <SafeDrainContext.Provider value={{
            activeDrains,
            isDrawerOpen,
            isMinimized,
            addDrain,
            removeDrain,
            requestCancel,
            minimizeDrawer,
            restoreDrawer,
            closeDrawer,
            clearCompleted
        }}>
            {children}
        </SafeDrainContext.Provider>
    )
}

'use client'

import React, { createContext, useContext, useState, useCallback, useEffect, useRef } from 'react'
import { toast } from 'sonner'
import { appConfig } from '@/config'
import type { DrainPodMigrationInfo, DrainStats, DrainStatus, DrainLog } from '@/types/safe-drain'

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
    stats?: DrainStats
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

    // Load from localStorage on mount
    useEffect(() => {
        const saved = localStorage.getItem('navy_active_drains')
        if (saved) {
            try {
                const parsed = JSON.parse(saved)
                setActiveDrains(parsed)
                // Reconnect to running drains and load migrations
                parsed.forEach((d: ActiveDrain) => {
                    if (d.status === 'running' || d.status === 'pending') {
                        connectToDrainStream(d.drainID)
                    }
                    // 如果 migrations 为空但 stats 显示有数据，尝试从 API 获取
                    if (d.migrations.length === 0 && (d.stats?.totalPods || 0) > 0) {
                        loadMigrationsFromAPI(d.drainID)
                    }
                })
            } catch (e) {
                console.error('Failed to parse saved drains', e)
            }
        }
    }, [])

    // Save to localStorage on change
    useEffect(() => {
        localStorage.setItem('navy_active_drains', JSON.stringify(activeDrains))
    }, [activeDrains])

    // SSE 连接管理 ref，避免重连
    const eventSourcesRef = useRef<Map<string, EventSource>>(new Map())

    // 添加新的 Drain 任务
    const addDrain = useCallback((drainID: string, nodeName: string, clusterName: string) => {
        setIsDrawerOpen(true)
        setIsMinimized(false)

        setActiveDrains(prev => {
            // 避免重复添加
            if (prev.some(d => d.drainID === drainID)) return prev

            // 清理逻辑：
            // 1. 清理所有已完成、失败、取消的历史任务，避免干扰
            // 2. 清理针对同一节点+集群的旧任务（无论状态如何），确保显示最新的
            const cleanPrev = prev.filter(d => {
                const isTerminal = ['completed', 'failed', 'cancelled', 'timeout'].includes(d.status)
                const isSameTarget = d.nodeName === nodeName && d.clusterName === clusterName

                // 如果是终端状态，或者是同一个目标的旧任务，则移除
                if (isTerminal || isSameTarget) return false
                return true
            })

            return [...cleanPrev, {
                drainID,
                nodeName,
                clusterName,
                status: 'running',
                progress: 0,
                currentStep: '初始化...',
                logs: [],
                startTime: Date.now(),
                migrations: [],
                stats: { totalPods: 0, migratedPods: 0, failedPods: 0, pendingPods: 0, migratingPods: 0, ignoredPods: 0, pdbCount: 0 }
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

    // 从 API 加载迁移数据（作为 SSE 的补充）
    const loadMigrationsFromAPI = async (drainID: string) => {
        try {
            console.log(`[SafeDrain] Loading migrations from API for drain: ${drainID}`)
            const res = await fetch(`${appConfig.apiBaseUrl}/navy/drain/${drainID}/migrations`, {
                credentials: 'include',
            })
            if (!res.ok) {
                console.warn(`[SafeDrain] Failed to load migrations for ${drainID}: ${res.status}`)
                return
            }
            const result = await res.json()
            console.log(`[SafeDrain] API response for ${drainID}:`, result)

            // 后端 API 返回格式: { success, message, data: { drainId, migrations, summary } }
            const migrations = result.data?.migrations || result.migrations || []
            console.log(`[SafeDrain] Extracted ${migrations.length} migrations`)

            if (Array.isArray(migrations) && migrations.length > 0) {
                setActiveDrains(prev => prev.map(drain => {
                    if (drain.drainID !== drainID) return drain
                    // 合并迁移数据
                    const updatedMigrations = [...drain.migrations]
                    migrations.forEach((mig: DrainPodMigrationInfo) => {
                        const idx = updatedMigrations.findIndex(m => m.migrationId === mig.migrationId)
                        if (idx >= 0) {
                            updatedMigrations[idx] = { ...updatedMigrations[idx], ...mig }
                        } else {
                            updatedMigrations.push(mig)
                        }
                    })
                    console.log(`[SafeDrain] Updated drain ${drainID} with ${updatedMigrations.length} migrations`)

                    // 重新计算统计
                    const newDrain = { ...drain, migrations: updatedMigrations }
                    computeStatsForDrain(newDrain)
                    return newDrain
                }))
            }
        } catch (err) {
            console.error(`[SafeDrain] Error loading migrations for ${drainID}:`, err)
        }
    }

    // 为指定的 drain 计算统计（独立函数，用于 API 加载后重新计算）
    const computeStatsForDrain = (drain: ActiveDrain) => {
        const total = drain.migrations.length
        const migrated = drain.migrations.filter(m => m.status === 'completed').length
        const failed = drain.migrations.filter(m => m.status === 'failed' || m.status === 'timeout').length
        const pending = drain.migrations.filter(m => m.status === 'pending').length
        const migrating = drain.migrations.filter(m => m.status === 'evicting' || m.status === 'evicted' || m.status === 'creating').length
        const ignored = drain.migrations.filter(m => m.status === 'ignored').length
        drain.stats = {
            totalPods: total,
            migratedPods: migrated,
            failedPods: failed,
            pendingPods: pending,
            migratingPods: migrating,
            ignoredPods: ignored,
            pdbCount: drain.stats?.pdbCount ?? 0
        }
    }

    // 连接 SSE
    const connectToDrainStream = (drainID: string) => {
        if (eventSourcesRef.current.has(drainID)) return

        const url = `${appConfig.apiBaseUrl}/navy/drain/${drainID}/events`
        const es = new EventSource(url, { withCredentials: true })

        es.onopen = () => {
            // SSE 连接成功后，尝试从 API 加载迁移数据（以防 SSE 事件丢失）
            loadMigrationsFromAPI(drainID)
        }

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
            console.warn(`SSE error for drain ${drainID}, will retry in 2s...`, err)
            es.close()
            eventSourcesRef.current.delete(drainID)

            // 检查是否应该重连（只为运行中的 drain 重连）
            setActiveDrains(prev => {
                const drain = prev.find(d => d.drainID === drainID)
                if (drain && drain.status === 'running') {
                    // 延迟重连，避免立即重连打到错误的后端实例
                    setTimeout(() => {
                        console.log(`[SafeDrain] Reconnecting SSE for ${drainID}...`)
                        connectToDrainStream(drainID)
                    }, 2000)
                }
                return prev
            })
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
                    // 不再根据 progress >= 100 自动完成，因为这可能只是 Eviction 进度。
                    // 只有明确收到 completion 事件才结束。
                    break
                }
                case 'stream_closing': {
                    const reason = data?.reason
                    if (reason !== 'failed' && reason !== 'cancelled') {
                        updated.status = 'completed'
                        updated.progress = 100
                        updated.currentStep = 'Drain operation finished'
                        updated.logs = [...updated.logs, { timestamp: ts, level: 'SUCCESS', message: 'Drain operation finished' }]
                    }
                    cleanupEventSource(drainID)
                    break
                }
                case 'drain_completed': // Handle explicit completion event if backend sends it
                case 'completed': {
                    updated.status = 'completed'
                    updated.progress = 100
                    updated.currentStep = msg || 'Drain completed'
                    updated.logs = [...updated.logs, { timestamp: ts, level: 'SUCCESS', message: updated.currentStep }]
                    cleanupEventSource(drainID)

                    // Auto-remove completed drain after 5 seconds
                    /* setTimeout(() => {
                        removeDrain(drainID)
                    }, 5000) */
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
                case 'pod_snapshot_created': {
                    // 处理初始快照 - 这是填充迁移数据的关键事件
                    const snapshot = data?.snapshot
                    if (snapshot?.pods && Array.isArray(snapshot.pods)) {
                        console.log('[SafeDrain] Processing snapshot pods:', snapshot.pods.length);
                        const initialMigrations: DrainPodMigrationInfo[] = snapshot.pods.map((pod: any) => {
                            const owners = (pod.ownerReferences || pod.OwnerReferences || []) as any[];
                            const hasOwners = Array.isArray(owners) && owners.length > 0;
                            const ownerKinds = hasOwners ? owners.map((o) => (o?.kind || o?.Kind || '') as string) : [];

                            // Debug logs for specific system pods
                            if (pod.name?.includes('kindnet') || pod.name?.includes('kube-proxy')) {
                                console.log(`[SafeDrain] Inspecting system pod: ${pod.name}`, {
                                    owners,
                                    ownerKinds,
                                    hasOwners
                                });
                            }

                            const looksDaemonSet = ownerKinds.some(k => k === 'DaemonSet' || k === 'daemonset');
                            const looksStatefulSet = ownerKinds.some(k => k === 'StatefulSet' || k === 'statefulset');
                            const looksStatic = !hasOwners;
                            const shouldIgnore = looksDaemonSet || looksStatefulSet || looksStatic;

                            if ((pod.name?.includes('kindnet') || pod.name?.includes('kube-proxy')) && !shouldIgnore) {
                                console.warn(`[SafeDrain] System pod ${pod.name} NOT ignored!`, {
                                    looksDaemonSet,
                                    looksStatefulSet,
                                    looksStatic,
                                    ownerKinds
                                });
                            }

                            return {
                                migrationId: `${drainID}-${pod.namespace}-${pod.name}`,
                                drainId: drainID,
                                sourcePod: {
                                    name: pod.name,
                                    namespace: pod.namespace,
                                    nodeName: pod.nodeName,
                                    uid: pod.uid,
                                    phase: pod.phase,
                                },
                                status: shouldIgnore ? 'ignored' : 'pending' as const,
                                startTime: ts,
                            };
                        })
                        updated.migrations = initialMigrations
                        const ignoredCount = initialMigrations.filter(m => m.status === 'ignored').length;
                        updated.stats = {
                            ...updated.stats,
                            totalPods: initialMigrations.length,
                            pendingPods: initialMigrations.length - ignoredCount,
                            ignoredPods: ignoredCount,
                        } as DrainStats
                    }
                    updated.logs = [...updated.logs, { timestamp: ts, level: 'INFO', message: msg || `Snapshot created: ${snapshot?.pods?.length || 0} pods` }]
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
                case 'pod_migration_started':
                case 'pod_migration_completed':
                case 'pod_migration_failed':
                case 'pod_migration_ignored':
                case 'new_pod_detected':
                case 'pod_phase_updated':
                case 'pod_pending':
                case 'new_pod_running':
                case 'new_pod_ready': {
                    const mig = data?.migration as DrainPodMigrationInfo | undefined
                    if (mig) {
                        upsertMigration(updated, mig)
                    }
                    const level = type.includes('failed') ? 'ERROR' :
                        type.includes('completed') || type.includes('ready') || type.includes('running') ? 'SUCCESS' : 'INFO'
                    updated.logs = [...updated.logs, { timestamp: ts, level: level as any, message: msg || type }]
                    computeStats(updated)
                    break
                }
                case 'stats': {
                    if (data) {
                        updated.stats = {
                            ...updated.stats,
                            // Merge stats from server if provided, otherwise logic-computed stats prevail
                            // Assuming server sends authoritative stats
                            totalPods: data.totalPods ?? updated.stats?.totalPods,
                            migratedPods: data.migrated ?? data.completed ?? updated.stats?.migratedPods,
                            failedPods: data.failed ?? updated.stats?.failedPods,
                            pendingPods: data.pending ?? updated.stats?.pendingPods,
                            migratingPods: (data.evicting ?? 0) + (data.creating ?? 0),
                            ignoredPods: data.ignored ?? updated.stats?.ignoredPods,
                            pdbCount: data.pdbCreated ?? updated.stats?.pdbCount ?? 0
                        } as DrainStats
                    }
                    break
                }
                default: {
                    // 未知事件也记录日志便于调试
                    if (msg) {
                        updated.logs = [...updated.logs, { timestamp: ts, level: 'INFO', message: `[${type}] ${msg}` }]
                    }
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

    // 计算统计（大小写不敏感的状态比较）
    const computeStats = (drain: ActiveDrain) => {
        const normalize = (s: string) => (s || '').toLowerCase()
        const total = drain.migrations.length
        const migrated = drain.migrations.filter(m => {
            const s = normalize(m.status as string)
            return s === 'completed'
        }).length
        const failed = drain.migrations.filter(m => {
            const s = normalize(m.status as string)
            return s === 'failed' || s === 'timeout'
        }).length
        const pending = drain.migrations.filter(m => normalize(m.status as string) === 'pending').length
        const migrating = drain.migrations.filter(m => {
            const s = normalize(m.status as string)
            return s === 'evicting' || s === 'evicted' || s === 'creating'
        }).length
        const ignored = drain.migrations.filter(m => normalize(m.status as string) === 'ignored').length
        drain.stats = { totalPods: total, migratedPods: migrated, failedPods: failed, pendingPods: pending, migratingPods: migrating, ignoredPods: ignored, pdbCount: drain.stats?.pdbCount ?? 0 }
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

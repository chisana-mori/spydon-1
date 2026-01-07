/**
 * DrainContext - Safe Drain 状态管理
 *
 * 采用多 Context 分离架构，优化性能：
 * - StatsContext: 统计数据
 * - LogsContext: 日志数据
 * - MigrationsContext: 迁移列表
 * - SortedMigrationsContext: 预排序的迁移数据
 * - ProgressContext: 进度状态
 * - TimerContext: 计时器状态
 * - ActionsContext: 操作函数
 */
import React, {
    createContext,
    useContext,
    useState,
    useEffect,
    useCallback,
    useRef,
    startTransition,
    useMemo
} from 'react';
import { safeDrainAPI } from '@/services/safe-drain-api';
import type {
    DrainPodMigrationInfo,
    DrainStats,
    LogEntry,
    SSEMessage,
    SortedMigrations,
    DrainStatus,
} from '@/types/safe-drain';
import { PodMigrationStatus } from '@/types/safe-drain';

// --- 1. Context 类型定义 ---
interface TimerState {
    startTime: Date | null;
    elapsedTime: number; // seconds
    isRunning: boolean;
    isOvertime: boolean;
}

interface ProgressState {
    progress: number;
    sseStatus: 'connecting' | 'open' | 'retrying' | 'closed';
}

interface Actions {
    startDrain: (cluster: string, node: string) => Promise<void>;
    cancelDrain: () => Promise<void>;
    reset: () => void;
    loadInitialData: (opts?: { quickRetry?: boolean; force?: boolean }) => Promise<void>;
    hasPendingPods: boolean;
}

interface DrainContextProps {
    drainId: string | null;
    status: DrainStatus;
    nodeName: string;
}

// --- 2. 创建所有 Context ---
const StatsContext = createContext<DrainStats | undefined>(undefined);
const LogsContext = createContext<LogEntry[] | undefined>(undefined);
const MigrationsContext = createContext<DrainPodMigrationInfo[] | undefined>(undefined);
const SortedMigrationsContext = createContext<SortedMigrations | undefined>(undefined);
const ProgressContext = createContext<ProgressState | undefined>(undefined);
const TimerContext = createContext<TimerState | undefined>(undefined);
const ActionsContext = createContext<Actions | undefined>(undefined);
const DrainIdContext = createContext<DrainContextProps | undefined>(undefined);

// --- 3. 自定义 Hooks ---
export const useStats = () => {
    const context = useContext(StatsContext);
    if (context === undefined) throw new Error('useStats must be used within a DrainProvider');
    return context;
};

export const useLogs = () => {
    const context = useContext(LogsContext);
    if (context === undefined) throw new Error('useLogs must be used within a DrainProvider');
    return context;
};

export const useMigrations = () => {
    const context = useContext(MigrationsContext);
    if (context === undefined) throw new Error('useMigrations must be used within a DrainProvider');
    return context;
};

export const useSortedMigrations = () => {
    const context = useContext(SortedMigrationsContext);
    if (context === undefined) throw new Error('useSortedMigrations must be used within a DrainProvider');
    return context;
};

export const useProgress = () => {
    const context = useContext(ProgressContext);
    if (context === undefined) throw new Error('useProgress must be used within a DrainProvider');
    return context;
};

export const useTimer = () => {
    const context = useContext(TimerContext);
    if (context === undefined) throw new Error('useTimer must be used within a DrainProvider');
    return context;
};

export const useActions = () => {
    const context = useContext(ActionsContext);
    if (context === undefined) throw new Error('useActions must be used within a DrainProvider');
    return context;
};

export const useDrainId = () => {
    const context = useContext(DrainIdContext);
    if (context === undefined) throw new Error('useDrainId must be used within a DrainProvider');
    return context;
};

// 兼容旧 API 的 useDrain Hook
export const useDrain = () => {
    const stats = useStats();
    const logs = useLogs();
    const migrations = useMigrations();
    const { progress, sseStatus } = useProgress();
    const timer = useTimer();
    const actions = useActions();
    const { drainId, status, nodeName } = useDrainId();

    return {
        drainId,
        status,
        nodeName,
        logs,
        migrations,
        stats,
        progress,
        progressMessage: '',
        isConnected: sseStatus === 'open',
        elapsedTime: timer.elapsedTime,
        ...actions,
    };
};

// --- 4. Helper Constants ---
const MAX_LOGS = 800;
const FLUSH_WINDOW_MS = 250;
const MAX_DRAIN_TIME_MINUTES = 30;
const MAX_DRAIN_TIME_SECONDS = MAX_DRAIN_TIME_MINUTES * 60;

// 状态比较辅助函数 - 避免 TypeScript 类型推断问题
const isStatus = (status: string | undefined, ...targets: string[]): boolean => {
    if (!status) return false;
    const normalized = status.toLowerCase();
    return targets.some(t => t.toLowerCase() === normalized);
};

const isCompleted = (status: string | undefined) => isStatus(status, 'completed');
const isFailed = (status: string | undefined) => isStatus(status, 'failed');
const isIgnored = (status: string | undefined) => isStatus(status, 'ignored');
const isPending = (status: string | undefined) => isStatus(status, 'pending');
const isEvicting = (status: string | undefined) => isStatus(status, 'evicting');
const isEvicted = (status: string | undefined) => isStatus(status, 'evicted');
const isCreating = (status: string | undefined) => isStatus(status, 'creating');
const isMigrating = (status: string | undefined) => isEvicting(status) || isEvicted(status) || isCreating(status);
const isTerminalStatus = (status: string | undefined) => isCompleted(status) || isFailed(status) || isIgnored(status);

// Backoff 工具函数
const nextBackoffDelay = (
    attempt: number,
    opts: { base: number; factor: number; max: number } = { base: 1000, factor: 2, max: 15000 }
) => {
    const delay = Math.min(opts.base * Math.pow(opts.factor, attempt), opts.max);
    return delay + Math.random() * 200; // 添加抖动
};

// --- 5. Provider 组件 ---
interface DrainProviderProps {
    children: React.ReactNode;
}

export const DrainProvider: React.FC<DrainProviderProps> = ({ children }) => {
    // Drain 基础状态
    const [drainId, setDrainId] = useState<string | null>(null);
    const [status, setStatus] = useState<DrainStatus>('pending');
    const [nodeName, setNodeName] = useState<string>('');

    // 数据状态
    const [realtimeLogs, setRealtimeLogs] = useState<LogEntry[]>([]);
    const [migrations, setMigrations] = useState<DrainPodMigrationInfo[]>([]);
    const [stats, setStats] = useState<DrainStats>({
        totalPods: 0,
        migratedPods: 0,
        failedPods: 0,
        pendingPods: 0,
        migratingPods: 0,
        ignoredPods: 0,
        pdbCount: 0
    });
    const [progress, setProgress] = useState(0);
    const [sseStatus, setSseStatus] = useState<'connecting' | 'open' | 'retrying' | 'closed'>('closed');

    // 冻结状态（任务完成后防止数据丢失）
    const [frozen, setFrozen] = useState(false);
    const [frozenMigrations, setFrozenMigrations] = useState<DrainPodMigrationInfo[] | null>(null);

    // Timer 状态
    const [timerState, setTimerState] = useState<TimerState>({
        startTime: null,
        elapsedTime: 0,
        isRunning: false,
        isOvertime: false,
    });

    // Refs
    const eventSourceRef = useRef<EventSource | null>(null);
    const processedMessagesRef = useRef<Set<string>>(new Set());
    const pendingLogsRef = useRef<LogEntry[]>([]);
    const flushTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
    const pendingMigsRef = useRef<Record<string, { payload: unknown; nextStatus?: PodMigrationStatus }>>({});
    const migrationsRef = useRef<DrainPodMigrationInfo[]>([]);
    const timerIntervalRef = useRef<ReturnType<typeof setInterval> | null>(null);
    const freezeStateRef = useRef(false);
    const isTerminalRef = useRef(false);
    const drainFinishedRef = useRef(false);
    const snapshotTotalRef = useRef<number | undefined>(undefined);
    const reconnectTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
    const retryAttemptRef = useRef(0);
    const unmountedRef = useRef(false);

    // 同步 migrationsRef
    useEffect(() => {
        migrationsRef.current = migrations;
    }, [migrations]);



    // --- Timer 管理 ---
    const startTimer = useCallback(() => {
        const now = new Date();
        setTimerState(prev => ({
            ...prev,
            startTime: now,
            isRunning: true,
            elapsedTime: 0,
            isOvertime: false,
        }));

        if (timerIntervalRef.current) {
            clearInterval(timerIntervalRef.current);
        }

        timerIntervalRef.current = setInterval(() => {
            setTimerState(prev => {
                if (!prev.startTime || !prev.isRunning) return prev;
                const elapsed = Math.floor((Date.now() - prev.startTime.getTime()) / 1000);
                const isOvertime = elapsed > MAX_DRAIN_TIME_SECONDS;
                return { ...prev, elapsedTime: elapsed, isOvertime };
            });
        }, 1000);
    }, []);

    const stopTimer = useCallback(() => {
        setTimerState(prev => ({ ...prev, isRunning: false }));
        if (timerIntervalRef.current) {
            clearInterval(timerIntervalRef.current);
            timerIntervalRef.current = null;
        }
    }, []);

    const resetTimer = useCallback(() => {
        setTimerState({
            startTime: null,
            elapsedTime: 0,
            isRunning: false,
            isOvertime: false,
        });
        if (timerIntervalRef.current) {
            clearInterval(timerIntervalRef.current);
            timerIntervalRef.current = null;
        }
    }, []);

    // --- 迁移数据变化时自动计算统计信息 (Navy 侧逻辑) ---
    useEffect(() => {
        const effective = frozenMigrations ?? migrations;
        const totalHint = snapshotTotalRef.current;
        // 优先使用快照总数，如果没有快照数据才使用迁移数据长度
        const total = typeof totalHint === 'number' && totalHint > 0 ? totalHint : effective.length;

        if (total > 0) {
            const completed = effective.filter(m => isCompleted(m.status as string)).length;
            const ignored = effective.filter(m => isIgnored(m.status as string)).length;
            const failed = effective.filter(m => isFailed(m.status as string)).length;

            // 逻辑与 Navy 保持一致：
            // 如果冻结（结束），失败也算入进度条（Done）；否则只计算完成+忽略
            const done = frozen ? (completed + ignored + failed) : (completed + ignored);
            const newProgress = Math.round((done / total) * 100);

            setProgress(newProgress);

            setStats(prev => ({
                ...prev,
                totalPods: total, // 确保使用计算出的正确总数
                migratedPods: completed,
                failedPods: failed,
                pendingPods: effective.filter(m => isPending(m.status as string)).length,
                migratingPods: effective.filter(m => isMigrating(m.status as string)).length,
                ignoredPods: ignored,
            }));

            // 如果有迁移活动但计时器未启动，则启动计时器
            const hasActiveMigrations = effective.some(m =>
                isMigrating(m.status as string) || isPending(m.status as string)
            );
            if (hasActiveMigrations && !timerState.isRunning && !frozen) {
                startTimer();
            }
        }
    }, [migrations, frozenMigrations, frozen, timerState.isRunning, startTimer]);

    // --- 日志处理 ---
    const addLog = useCallback((level: string, message: string, category: string = 'GENERAL', customId?: string) => {
        const messageId = customId || `${level}:${message}:${category}:${Date.now()}`;
        if (processedMessagesRef.current.has(messageId) && customId) {
            return;
        }
        if (customId) processedMessagesRef.current.add(messageId);

        const timestamp = new Date().toLocaleTimeString();
        const logEntry: LogEntry = { id: messageId, timestamp, level, message, category };

        pendingLogsRef.current.push(logEntry);
        scheduleFlushing();
    }, []);

    // --- 迁移数据更新 ---
    const upsertMigration = useCallback((payload: unknown, nextStatus?: PodMigrationStatus) => {
        if (freezeStateRef.current) return;

        const mig = (payload as Record<string, unknown>)?.migration as Record<string, unknown> || {};
        const srcPod = mig?.sourcePod as Record<string, unknown> || mig?.SourcePod as Record<string, unknown> || {};
        const srcName = (srcPod?.name || srcPod?.Name || '') as string;
        const srcNs = (srcPod?.namespace || srcPod?.Namespace || '') as string;
        if (!srcName) return;

        const migrationId = (mig?.migrationId || mig?.MigrationID || `${srcNs}/${srcName}`) as string;

        const existing = pendingMigsRef.current[migrationId];
        if (existing) {
            const mergedStatus = chooseStatus(nextStatus, existing.nextStatus);
            const newHasTarget = !!((mig as Record<string, unknown>)?.targetPod || (mig as Record<string, unknown>)?.TargetPod);
            const oldHasTarget = !!((existing.payload as Record<string, unknown>)?.migration as Record<string, unknown>)?.targetPod;
            pendingMigsRef.current[migrationId] = {
                payload: newHasTarget || !oldHasTarget ? payload : existing.payload,
                nextStatus: mergedStatus,
            };
        } else {
            pendingMigsRef.current[migrationId] = { payload, nextStatus };
        }

        scheduleFlushing();
    }, []);

    // 状态优先级选择
    const chooseStatus = (a?: PodMigrationStatus, b?: PodMigrationStatus): PodMigrationStatus | undefined => {
        if (!a) return b;
        if (!b) return a;
        const rank = (s?: PodMigrationStatus) => {
            switch (s) {
                case PodMigrationStatus.Completed: return 5;
                case PodMigrationStatus.Creating:
                case PodMigrationStatus.Evicted:
                case PodMigrationStatus.Ignored: return 4;
                case PodMigrationStatus.Evicting:
                case PodMigrationStatus.Pending: return 3;
                case PodMigrationStatus.Failed: return 6;
                default: return 0;
            }
        };
        return rank(a) >= rank(b) ? a : b;
    };

    // 批量刷新调度
    const scheduleFlushing = useCallback(() => {
        if (flushTimerRef.current) return;

        flushTimerRef.current = setTimeout(() => {
            flushTimerRef.current = null;

            // Flush Logs
            const newLogs = pendingLogsRef.current;
            pendingLogsRef.current = [];
            if (newLogs.length > 0) {
                setRealtimeLogs(prev => {
                    const merged = [...newLogs, ...prev];
                    return merged.slice(0, MAX_LOGS);
                });
            }

            // Flush Migrations
            const migsDelta = pendingMigsRef.current;
            pendingMigsRef.current = {};
            const keys = Object.keys(migsDelta);

            if (keys.length > 0) {
                startTransition(() => {
                    setMigrations(prev => {
                        const list = [...prev];
                        const idxMap = new Map<string, number>();
                        list.forEach((m, i) => idxMap.set(m.migrationId || `${m.sourcePod?.namespace}/${m.sourcePod?.name}`, i));

                        keys.forEach(k => {
                            const { payload, nextStatus } = migsDelta[k];
                            const mig = (payload as Record<string, unknown>)?.migration as Record<string, unknown> || {};
                            const srcPod = mig?.sourcePod as Record<string, unknown> || mig?.SourcePod as Record<string, unknown> || {};
                            const srcName = (srcPod?.name || srcPod?.Name || '') as string;
                            const srcNs = (srcPod?.namespace || srcPod?.Namespace || '') as string;

                            const tgtPod = mig?.targetPod as Record<string, unknown> || mig?.TargetPod as Record<string, unknown>;
                            const tgtName = tgtPod?.name as string || tgtPod?.Name as string;
                            const tgtNs = (tgtPod?.namespace || tgtPod?.Namespace || srcNs) as string;
                            const tgtNode = (tgtPod?.nodeName || tgtPod?.NodeName || '') as string;
                            const tgtPhase = (tgtPod?.phase || tgtPod?.Phase || '') as string;
                            const tgtPhaseReason = (tgtPod?.phaseReason || tgtPod?.phase_reason || '') as string;

                            const migrationId = (mig?.migrationId || mig?.MigrationID || `${srcNs}/${srcName}`) as string;

                            const idx = idxMap.get(migrationId);
                            if (idx !== undefined) {
                                const cur = list[idx];
                                list[idx] = {
                                    ...cur,
                                    status: nextStatus || cur.status,
                                    targetPod: tgtName ? {
                                        name: tgtName || cur.targetPod?.name || '',
                                        namespace: tgtNs || cur.targetPod?.namespace || cur.sourcePod?.namespace || '',
                                        nodeName: tgtNode || cur.targetPod?.nodeName || '',
                                        uid: cur.targetPod?.uid || '',
                                        phase: tgtPhase || cur.targetPod?.phase || '',
                                        phaseReason: tgtPhaseReason || cur.targetPod?.phaseReason,
                                        createdAt: cur.targetPod?.createdAt || new Date().toISOString(),
                                    } : cur.targetPod,
                                } as DrainPodMigrationInfo;
                            } else {
                                list.push({
                                    drainId: drainId || '',
                                    migrationId,
                                    sourcePod: {
                                        name: srcName,
                                        namespace: srcNs,
                                        nodeName: '',
                                        uid: '',
                                        phase: '',
                                        createdAt: new Date().toISOString(),
                                    },
                                    targetPod: tgtName ? {
                                        name: tgtName,
                                        namespace: tgtNs,
                                        nodeName: tgtNode,
                                        uid: '',
                                        phase: tgtPhase,
                                        phaseReason: tgtPhaseReason,
                                        createdAt: new Date().toISOString(),
                                    } : undefined,
                                    status: nextStatus || PodMigrationStatus.Pending,
                                    startTime: new Date().toISOString(),
                                });
                            }
                        });

                        return list;
                    });
                });
            }
        }, FLUSH_WINDOW_MS);
    }, [drainId]);

    // --- 状态冻结（任务完成时） ---
    const freezeSnapshot = useCallback((snapshotInput?: DrainPodMigrationInfo[]) => {
        if (freezeStateRef.current) return;

        const source = (snapshotInput && snapshotInput.length > 0)
            ? snapshotInput
            : (migrationsRef.current.length > 0 ? migrationsRef.current : []);

        const cloned = source.map(m => ({
            ...m,
            sourcePod: { ...m.sourcePod },
            targetPod: m.targetPod ? { ...m.targetPod } : undefined,
        }));

        freezeStateRef.current = true;
        setFrozen(true);
        setFrozenMigrations(cloned);
        setMigrations(cloned);

        // 关闭 SSE 连接
        if (eventSourceRef.current) {
            try { eventSourceRef.current.close(); } catch { /* ignore */ }
            eventSourceRef.current = null;
        }
        if (reconnectTimerRef.current) {
            clearTimeout(reconnectTimerRef.current);
            reconnectTimerRef.current = null;
        }
        isTerminalRef.current = true;
        setSseStatus('closed');
    }, []);

    // --- SSE 消息处理 ---
    const handleSSEMessage = useCallback((msg: SSEMessage) => {
        try {
            const type = msg?.type;
            const d = msg?.data || {};
            const message = msg?.message || '';

            switch (type) {
                case 'started':
                case 'drain_started':
                    setStatus('running');
                    addLog('INFO', message || 'Drain started');
                    startTimer();
                    break;

                case 'progress': {
                    const total = d?.totalPods;
                    if (typeof total === 'number') {
                        setStats(prev => ({ ...prev, totalPods: total }));
                        snapshotTotalRef.current = total;
                    }
                    // Navy 逻辑：优先使用前端派生进度，忽略后端通过 SSE 发送的 progress
                    // if (typeof d?.progress === 'number') setProgress(d.progress);
                    break;
                }

                case 'stats': {
                    const total = d?.totalPods;
                    if (typeof total === 'number') {
                        snapshotTotalRef.current = total;
                        setStats(prev => ({ ...prev, totalPods: total }));
                    }
                    if (d?.pdbCreated !== undefined) {
                        setStats(prev => ({ ...prev, pdbCount: d.pdbCreated }));
                    }
                    // Navy 逻辑：Stats 详细数据由前端派生，忽略后端 SSE 推送
                    /*
                    if (d?.migrated !== undefined || d?.failed !== undefined) {
                        setStats(prev => ({
                            ...prev,
                            migratedPods: d?.migrated ?? prev.migratedPods,
                            failedPods: d?.failed ?? prev.failedPods,
                            pendingPods: d?.pending ?? prev.pendingPods,
                            migratingPods: (d?.evicting ?? 0) + (d?.creating ?? 0),
                            ignoredPods: d?.ignored ?? prev.ignoredPods,
                        }));
                    }
                    */
                    break;
                }

                case 'pod_snapshot_created': {
                    const snap = d?.snapshot;
                    const total = Array.isArray(snap?.pods) ? snap.pods.length : (snap?.totalPods ?? 0);
                    if (typeof total === 'number' && total > 0) {
                        setStats(prev => ({ ...prev, totalPods: total }));
                        snapshotTotalRef.current = total;
                    }

                    // 从快照初始化迁移数据
                    if (Array.isArray(snap?.pods) && snap.pods.length > 0) {
                        snap.pods.forEach((pod: Record<string, unknown>) => {
                            const owners = (pod.ownerReferences || pod.OwnerReferences || []) as Array<Record<string, unknown>>;
                            const hasOwners = Array.isArray(owners) && owners.length > 0;
                            const ownerKinds = hasOwners ? owners.map((o) => (o?.kind || o?.Kind || '') as string) : [];
                            const looksDaemonSet = ownerKinds.some(k => k === 'DaemonSet');
                            const looksStatefulSet = ownerKinds.some(k => k === 'StatefulSet');
                            const looksStatic = !hasOwners;
                            const shouldIgnore = looksDaemonSet || looksStatefulSet || looksStatic;

                            const migrationData = {
                                migration: {
                                    sourcePod: {
                                        name: pod.name,
                                        namespace: pod.namespace,
                                        nodeName: pod.nodeName,
                                        uid: pod.uid || '',
                                        phase: pod.phase || '',
                                        createdAt: pod.createdAt || new Date().toISOString(),
                                    },
                                    status: shouldIgnore ? 'ignored' : 'pending',
                                    migrationId: `${drainId}-${pod.namespace}-${pod.name}`,
                                },
                            };
                            upsertMigration(migrationData, shouldIgnore ? PodMigrationStatus.Ignored : PodMigrationStatus.Pending);
                        });
                    }
                    addLog('INFO', message || '已创建 Pod 快照');
                    break;
                }

                case 'pod_eviction_started':
                    upsertMigration(d, PodMigrationStatus.Evicting);
                    addLog('INFO', `开始驱逐 Pod: ${d?.migration?.sourcePod?.namespace || ''}/${d?.migration?.sourcePod?.name || ''}`);
                    if (!timerState.isRunning) startTimer();
                    break;

                case 'pod_eviction_succeeded':
                    upsertMigration(d, PodMigrationStatus.Evicted);
                    addLog('SUCCESS', `Pod 驱逐成功: ${d?.migration?.sourcePod?.namespace || ''}/${d?.migration?.sourcePod?.name || ''}`);
                    break;

                case 'pod_eviction_failed':
                    upsertMigration(d, PodMigrationStatus.Failed);
                    addLog('ERROR', message || 'Pod 驱逐失败');
                    break;

                case 'new_pod_detected':
                    upsertMigration(d, PodMigrationStatus.Creating);
                    addLog('INFO', `检测到新 Pod: ${d?.migration?.targetPod?.namespace || ''}/${d?.migration?.targetPod?.name || ''}`);
                    break;

                case 'pod_phase_updated':
                    upsertMigration(d, undefined);
                    break;

                case 'pod_migration_completed':
                    upsertMigration(d, PodMigrationStatus.Completed);
                    addLog('SUCCESS', `Pod 迁移完成: ${d?.migration?.sourcePod?.name || ''} -> ${d?.migration?.targetPod?.name || ''}`);
                    break;

                case 'pod_migration_failed':
                    upsertMigration(d, PodMigrationStatus.Failed);
                    addLog('ERROR', `Pod 迁移失败: ${d?.migration?.sourcePod?.name || ''} - ${d?.migration?.errorMessage || '未知原因'}`);
                    break;

                case 'pod_migration_ignored':
                    upsertMigration(d, PodMigrationStatus.Ignored);
                    addLog('INFO', `Pod 已忽略 (DaemonSet): ${d?.migration?.sourcePod?.namespace || ''}/${d?.migration?.sourcePod?.name || ''}`);
                    break;

                case 'pdb_event': {
                    const action = d?.action;
                    if (action === 'created') {
                        addLog('INFO', `PDB 已创建: ${d?.namespace}/${d?.pdbName}`, 'PDB');
                    } else if (action === 'cleaned') {
                        addLog('SUCCESS', `PDB 已清理: ${d?.namespace}/${d?.pdbName}`, 'PDB');
                    }
                    break;
                }

                case 'stream_closing': {
                    const reason = d?.reason;
                    if (reason === 'failed' || reason === 'cancelled') {
                        setStatus(reason === 'cancelled' ? 'cancelled' : 'failed');
                        addLog('WARN', message || 'Drain 连接将关闭');
                    } else {
                        setStatus('completed');
                        setProgress(100);
                        addLog('SUCCESS', message || 'Drain 已完成');
                    }
                    stopTimer();
                    drainFinishedRef.current = true;
                    freezeSnapshot();
                    break;
                }

                case 'completed':
                case 'drain_completed':
                    drainFinishedRef.current = true;
                    setStatus('completed');
                    setProgress(100);
                    addLog('SUCCESS', '节点 drain 操作完成！');
                    stopTimer();
                    freezeSnapshot();
                    break;

                case 'failed':
                case 'drain_failed':
                    drainFinishedRef.current = true;
                    setStatus('failed');
                    addLog('ERROR', message || '节点 drain 操作失败！');
                    stopTimer();
                    freezeSnapshot();
                    break;

                case 'cancelled':
                case 'drain_cancelled':
                    drainFinishedRef.current = true;
                    setStatus('cancelled');
                    addLog('WARN', message || '节点 drain 操作已取消');
                    stopTimer();
                    freezeSnapshot();
                    break;

                case 'error':
                    addLog('ERROR', message || 'Unknown error');
                    break;

                default:
                    // 未知事件类型
                    break;
            }
        } catch (e) {
            console.error('[SSE] 处理消息异常:', e, msg);
        }
    }, [addLog, drainId, freezeSnapshot, startTimer, stopTimer, timerState.isRunning, upsertMigration]);

    // --- 加载初始数据（从 API 获取迁移列表） ---
    const loadInitialData = useCallback(async (opts?: { quickRetry?: boolean; force?: boolean }) => {
        if (freezeStateRef.current && !opts?.force) return;
        if (!drainId || drainId === 'undefined' || drainId.trim() === '') return;

        try {
            const maxAttempts = opts?.quickRetry ? 3 : 1;
            let attempt = 0;

            while (attempt < maxAttempts) {
                try {
                    const migrationsData = await safeDrainAPI.getDrainMigrations(drainId);
                    const list = (migrationsData.migrations || []).filter(m => !m?.drainId || m.drainId === drainId);

                    if (!freezeStateRef.current) {
                        setMigrations(list);
                    }

                    // 更新统计 - Navy 逻辑：
                    // 这里只需要 setMigrations，useEffect 会自动计算 Stats 和 Progress
                    // 但我们需要确保 snapshotTotalRef 被设置，以便 useEffect 使用正确的总数

                    const total = snapshotTotalRef.current && snapshotTotalRef.current > 0
                        ? snapshotTotalRef.current
                        : list.length;

                    if (snapshotTotalRef.current !== total) {
                        snapshotTotalRef.current = total;
                    }

                    /* 移除手动计算，交由 useEffect 处理
                    const completed = list.filter(m => isCompleted(m.status as string)).length;
                    const ignored = list.filter(m => isIgnored(m.status as string)).length;
                    const failed = list.filter(m => isFailed(m.status as string)).length;

                    setStats(prev => ({
                        ...prev,
                        totalPods: total,
                        migratedPods: completed,
                        failedPods: failed,
                        pendingPods: list.filter(m => isPending(m.status as string)).length,
                        migratingPods: list.filter(m => isMigrating(m.status as string)).length,
                        ignoredPods: ignored,
                    }));

                    if (total > 0) {
                        setProgress(Math.round(((completed + ignored) / total) * 100));
                    }
                    */

                    // 检查是否为终态
                    const isTerminal = total > 0 && list.every(m => isTerminalStatus(m.status as string));

                    if (isTerminal) {
                        addLog('INFO', '检测到任务已完成，将冻结当前状态。');
                        freezeSnapshot(list);
                    }

                    break;
                } catch (error) {
                    attempt++;
                    if (attempt >= maxAttempts) {
                        console.error('加载初始数据失败:', error);
                        addLog('ERROR', '加载初始数据失败');
                    } else {
                        const delay = nextBackoffDelay(attempt - 1, { base: 200, factor: 2, max: 1200 });
                        await new Promise(res => setTimeout(res, delay));
                    }
                }
            }
        } catch (error) {
            console.error('加载初始数据失败:', error);
            addLog('ERROR', '加载初始数据失败');
        }
    }, [drainId, addLog, freezeSnapshot]);

    // --- SSE 连接管理 ---
    const connectSSE = useCallback((id: string) => {
        if (eventSourceRef.current) {
            try { eventSourceRef.current.close(); } catch { /* ignore */ }
        }

        const url = `${process.env.NEXT_PUBLIC_API_URL || '/api'}/v1/navy/drain/${id}/events`;
        setSseStatus('connecting');

        const es = new EventSource(url, { withCredentials: true });

        es.onopen = () => {
            if (!unmountedRef.current) {
                setSseStatus('open');
                retryAttemptRef.current = 0;
                addLog('INFO', 'SSE 连接已建立', 'SYSTEM');
            }
        };

        es.onmessage = (event) => {
            if (!unmountedRef.current) {
                try {
                    const data = JSON.parse(event.data);
                    handleSSEMessage(data);
                } catch (error) {
                    console.error('[SSE] 解析消息失败:', error, event.data);
                }
            }
        };

        es.onerror = async () => {
            if (isTerminalRef.current || unmountedRef.current) {
                try { es.close(); } catch { /* ignore */ }
                eventSourceRef.current = null;
                setSseStatus('closed');
                return;
            }

            try { es.close(); } catch { /* ignore */ }
            eventSourceRef.current = null;
            setSseStatus('retrying');

            addLog('INFO', 'SSE 连接中断，正在通过 API 检查最新状态...', 'SYSTEM');
            await loadInitialData({ force: true });

            if (freezeStateRef.current || unmountedRef.current) return;

            addLog('WARN', 'API 确认任务尚未完成，将按计划重连 SSE...', 'SYSTEM');
            const delay = nextBackoffDelay(retryAttemptRef.current, { base: 1000, factor: 2, max: 15000 });
            retryAttemptRef.current = Math.min(retryAttemptRef.current + 1, 6);

            if (reconnectTimerRef.current) clearTimeout(reconnectTimerRef.current);
            reconnectTimerRef.current = setTimeout(() => {
                if (!isTerminalRef.current && !unmountedRef.current && drainId) {
                    connectSSE(id);
                }
            }, delay);
        };

        eventSourceRef.current = es;

        // 延迟加载初始数据作为备份
        setTimeout(() => {
            if (!freezeStateRef.current) {
                loadInitialData({ quickRetry: true });
            }
        }, 200);
    }, [addLog, handleSSEMessage, loadInitialData, drainId]);

    // --- 公开操作 ---
    const startDrain = async (cluster: string, node: string) => {
        reset();
        setNodeName(node);

        try {
            const res = await safeDrainAPI.startDrain({ clusterName: cluster, nodeName: node });
            setDrainId(res.drainId);
            setStatus('running');
            addLog('INFO', `Drain 已启动. ID: ${res.drainId}`, 'SYSTEM');
            connectSSE(res.drainId);
        } catch (e) {
            setStatus('failed');
            const errorMessage = e instanceof Error ? e.message : 'Failed to start drain';
            addLog('ERROR', errorMessage, 'SYSTEM');
            throw e;
        }
    };

    const cancelDrain = async () => {
        if (!drainId) return;
        try {
            await safeDrainAPI.cancelDrain(drainId);
            addLog('WARN', '取消请求已发送...', 'SYSTEM');
        } catch (e) {
            const errorMessage = e instanceof Error ? e.message : 'Failed to cancel';
            addLog('ERROR', `取消失败: ${errorMessage}`, 'SYSTEM');
        }
    };

    const reset = useCallback(() => {
        if (eventSourceRef.current) {
            try { eventSourceRef.current.close(); } catch { /* ignore */ }
            eventSourceRef.current = null;
        }
        if (reconnectTimerRef.current) {
            clearTimeout(reconnectTimerRef.current);
            reconnectTimerRef.current = null;
        }

        setDrainId(null);
        setStatus('pending');
        setNodeName('');
        setRealtimeLogs([]);
        setMigrations([]);
        setStats({
            totalPods: 0, migratedPods: 0, failedPods: 0, pendingPods: 0,
            migratingPods: 0, ignoredPods: 0, pdbCount: 0
        });
        setProgress(0);
        setSseStatus('closed');
        setFrozen(false);
        setFrozenMigrations(null);
        resetTimer();

        processedMessagesRef.current.clear();
        pendingMigsRef.current = {};
        pendingLogsRef.current = [];
        freezeStateRef.current = false;
        isTerminalRef.current = false;
        drainFinishedRef.current = false;
        snapshotTotalRef.current = undefined;
        retryAttemptRef.current = 0;
    }, [resetTimer]);

    // --- 派生进度计算 ---
    useEffect(() => {
        const effective = frozenMigrations ?? migrations;
        const totalHint = snapshotTotalRef.current;
        const total = typeof totalHint === 'number' && totalHint > 0 ? totalHint : effective.length;

        if (total > 0) {
            const completed = effective.filter(m => isCompleted(m.status as string)).length;
            const ignored = effective.filter(m => isIgnored(m.status as string)).length;
            const failed = effective.filter(m => isFailed(m.status as string)).length;
            const done = frozen ? (completed + ignored + failed) : (completed + ignored);
            const newProgress = Math.round((done / total) * 100);
            setProgress(newProgress);

            setStats(prev => ({
                ...prev,
                totalPods: total,
                migratedPods: completed,
                failedPods: failed,
                pendingPods: effective.filter(m => isPending(m.status as string)).length,
                migratingPods: effective.filter(m => isMigrating(m.status as string)).length,
                ignoredPods: ignored,
            }));
        }
    }, [migrations, frozenMigrations, frozen]);

    // --- 清理 ---
    useEffect(() => {
        return () => {
            unmountedRef.current = true;
            if (eventSourceRef.current) {
                try { eventSourceRef.current.close(); } catch { /* ignore */ }
            }
            if (flushTimerRef.current) clearTimeout(flushTimerRef.current);
            if (timerIntervalRef.current) clearInterval(timerIntervalRef.current);
            if (reconnectTimerRef.current) clearTimeout(reconnectTimerRef.current);
        };
    }, []);

    // --- 最终迁移数据 ---
    const finalMigrations = frozenMigrations ?? migrations;

    // --- 预排序迁移数据 ---
    const sortedMigrations = useMemo((): SortedMigrations => {
        const incompleteMigrations = finalMigrations
            .filter(m => !isCompleted(m.status as string) && !isIgnored(m.status as string))
            .sort((a, b) => {
                const priorityOrder: Record<string, number> = {
                    'failed': 0,
                    'evicting': 1,
                    'creating': 1,
                    'evicted': 1,
                    'pending': 2,
                    'ignored': 3,
                };
                const aPriority = priorityOrder[(a.status as string).toLowerCase()] ?? 99;
                const bPriority = priorityOrder[(b.status as string).toLowerCase()] ?? 99;

                if (aPriority !== bPriority) return aPriority - bPriority;
                return Date.parse(b.startTime) - Date.parse(a.startTime);
            });

        const completedMigrations = finalMigrations
            .filter(m => isCompleted(m.status as string) || isIgnored(m.status as string))
            .sort((a, b) => {
                const ta = Date.parse(a.targetPod?.createdAt || a.startTime);
                const tb = Date.parse(b.targetPod?.createdAt || b.startTime);
                return tb - ta;
            });

        return { incompleteMigrations, completedMigrations };
    }, [finalMigrations]);

    // --- Actions ---
    const actions: Actions = {
        startDrain,
        cancelDrain,
        reset,
        loadInitialData,
        hasPendingPods: finalMigrations.some(m => isPending(m.status as string)),
    };

    const progressState: ProgressState = { progress, sseStatus };
    const drainIdState: DrainContextProps = { drainId, status, nodeName };

    return (
        <DrainIdContext.Provider value={drainIdState}>
            <StatsContext.Provider value={stats}>
                <LogsContext.Provider value={realtimeLogs}>
                    <MigrationsContext.Provider value={finalMigrations}>
                        <SortedMigrationsContext.Provider value={sortedMigrations}>
                            <ProgressContext.Provider value={progressState}>
                                <TimerContext.Provider value={timerState}>
                                    <ActionsContext.Provider value={actions}>
                                        {children}
                                    </ActionsContext.Provider>
                                </TimerContext.Provider>
                            </ProgressContext.Provider>
                        </SortedMigrationsContext.Provider>
                    </MigrationsContext.Provider>
                </LogsContext.Provider>
            </StatsContext.Provider>
        </DrainIdContext.Provider>
    );
};

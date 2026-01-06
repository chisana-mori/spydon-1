import React, { createContext, useContext, useEffect, useRef, useState, useCallback, startTransition } from 'react';
import { DrainPodMigrationInfo, DrainStatus, LogEntry, DrainStats, DrainPodMigrationStatus, SSEMessage } from '@/types/safe-drain';

// --- 1. Define Context Types ---
interface DrainContextType {
    drainId: string | null;
    status: DrainStatus;
    logs: LogEntry[];
    migrations: DrainPodMigrationInfo[];
    stats: DrainStats;
    progress: number;
    progressMessage: string;
    isConnected: boolean;
    startDrain: (cluster: string, node: string) => Promise<void>;
    cancelDrain: () => Promise<void>;
    reset: () => void;
    elapsedTime: number; // For timer
}

const DrainContext = createContext<DrainContextType | undefined>(undefined);

export const useDrain = () => {
    const context = useContext(DrainContext);
    if (!context) {
        throw new Error('useDrain must be used within a DrainProvider');
    }
    return context;
};

// --- 2. Helper Constants & Types ---
const MAX_LOGS = 800;
const FLUSH_WINDOW_MS = 250;

export const DrainProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
    const [drainId, setDrainId] = useState<string | null>(null);
    const [status, setStatus] = useState<DrainStatus>('pending');
    const [isConnected, setIsConnected] = useState(false);

    // Data States
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
    const [progressMessage, setProgressMessage] = useState('');

    // Timer State
    const [startTime, setStartTime] = useState<number | null>(null);
    const [elapsedTime, setElapsedTime] = useState(0);
    const timerIntervalRef = useRef<number | null>(null);

    // Refs for buffering and access in callbacks
    const eventSourceRef = useRef<EventSource | null>(null);
    const processedMessagesRef = useRef<Set<string>>(new Set());
    const pendingLogsRef = useRef<LogEntry[]>([]);
    const flushTimerRef = useRef<NodeJS.Timeout | null>(null);
    const migrationsRef = useRef<DrainPodMigrationInfo[]>([]);
    // Keep a map for efficient updates: migrationId -> index in migrations array is not stable if we filter,
    // so best to rebuild or use a map. Ideally we sync `migrations` state with this ref.

    // Using a ref to track pending migration updates to batch them
    const pendingMigsRef = useRef<Record<string, { payload: any }>>({});

    useEffect(() => {
        migrationsRef.current = migrations;
    }, [migrations]);

    // --- Timer Logic ---
    const startTimer = useCallback(() => {
        if (timerIntervalRef.current) return;
        setStartTime(Date.now());
        timerIntervalRef.current = window.setInterval(() => {
            setElapsedTime(prev => prev + 1);
        }, 1000);
    }, []);

    const stopTimer = useCallback(() => {
        if (timerIntervalRef.current) {
            clearInterval(timerIntervalRef.current);
            timerIntervalRef.current = null;
        }
    }, []);

    const resetTimer = useCallback(() => {
        stopTimer();
        setStartTime(null);
        setElapsedTime(0);
    }, [stopTimer]);


    // --- Log Handling ---
    const addLog = useCallback((level: string, message: string, category: string = 'GENERAL', customId?: string) => {
        const messageId = customId || `${level}:${message}:${category}:${Date.now()}`;
        if (processedMessagesRef.current.has(messageId) && customId) {
            // Only dedup if customId is provided, otherwise allow dupes with different timestamps
            return;
        }
        if (customId) processedMessagesRef.current.add(messageId);

        const timestamp = new Date().toLocaleTimeString();
        const logEntry: LogEntry = { id: messageId, timestamp, level, message, category };

        pendingLogsRef.current.push(logEntry);

        if (!flushTimerRef.current) {
            flushTimerRef.current = setTimeout(() => {
                flushTimerRef.current = null;

                // Flush Logs
                const newLogs = pendingLogsRef.current;
                pendingLogsRef.current = [];
                if (newLogs.length > 0) {
                    setRealtimeLogs(prev => {
                        const merged = [...newLogs, ...prev]; // Newest first
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
                            list.forEach((m, i) => idxMap.set(m.migrationId, i));

                            keys.forEach(k => {
                                const { payload } = migsDelta[k];
                                const migData = payload.migration;
                                if (!migData) return;

                                // Normalize incoming data to internal type
                                const normalized: DrainPodMigrationInfo = {
                                    migrationId: migData.migrationId || migData.MigrationID,
                                    drainId: drainId || '',
                                    sourcePod: {
                                        name: migData.sourcePod?.name || migData.SourcePod?.Name || '',
                                        namespace: migData.sourcePod?.namespace || migData.SourcePod?.Namespace || '',
                                        nodeName: migData.sourcePod?.nodeName || migData.SourcePod?.NodeName || '',
                                        uid: migData.sourcePod?.uid || '',
                                        phase: migData.sourcePod?.phase || '',
                                        // ... other fields if needed
                                    },
                                    targetPod: migData.targetPod ? {
                                        name: migData.targetPod.name || migData.TargetPod?.Name || '',
                                        namespace: migData.targetPod.namespace || migData.TargetPod?.Namespace || '',
                                        nodeName: migData.targetPod.nodeName || migData.TargetPod?.NodeName || '',
                                        uid: migData.targetPod.uid || '',
                                        phase: migData.targetPod.phase || '',
                                    } : undefined,
                                    status: (migData.status || migData.Status || 'pending').toLowerCase() as DrainPodMigrationStatus,
                                    errorMessage: migData.errorMessage || migData.ErrorMessage,
                                    startTime: migData.startTime || migData.StartTime,
                                    evictionTime: migData.evictionTime || migData.EvictionTime,
                                    completionTime: migData.completionTime || migData.CompletionTime,
                                };

                                const idx = idxMap.get(normalized.migrationId);
                                if (idx !== undefined) {
                                    list[idx] = { ...list[idx], ...normalized };
                                } else {
                                    list.push(normalized);
                                }
                            });
                            return list;
                        });

                        // Recalculate derived stats after update if not server-provided
                        // (Optional, backend sends 'stats' event usually)
                    });
                }

            }, FLUSH_WINDOW_MS);
        }
    }, [drainId]);


    // --- SSE Handling ---
    const handleSSEMessage = useCallback((event: MessageEvent) => {
        try {
            const raw = JSON.parse(event.data);
            const type = raw.type;
            const data = raw.data || {};
            const msg = raw.message || '';

            switch (type) {
                case 'started':
                case 'drain_started':
                    setStatus('running');
                    addLog('INFO', msg || 'Drain started');
                    startTimer();
                    break;

                case 'progress':
                    if (typeof data.progress === 'number') setProgress(data.progress);
                    setProgressMessage(msg);
                    break;

                case 'stats':
                    setStats(prev => ({
                        totalPods: data.totalPods ?? prev.totalPods,
                        migratedPods: data.migrated ?? prev.migratedPods,
                        failedPods: data.failed ?? prev.failedPods,
                        pendingPods: data.pending ?? prev.pendingPods,
                        migratingPods: (data.evicting ?? 0) + (data.creating ?? 0),
                        ignoredPods: data.ignored ?? prev.ignoredPods,
                        pdbCount: data.pdbCreated ?? prev.pdbCount
                    }));
                    if (data.totalPods) {
                        // Ensure total pods is consistent
                    }
                    if (data.pdbCreated !== undefined) {
                        // Check for completion? Logic handled in stream_closing or separate event
                    }
                    break;

                case 'pod_snapshot_created': {
                    const snap = data.snapshot;
                    const total = snap?.pods?.length || snap?.totalPods || 0;
                    setStats(prev => ({ ...prev, totalPods: total }));

                    // Initialize migrations from snapshot
                    if (Array.isArray(snap?.pods)) {
                        const initialDetails: DrainPodMigrationInfo[] = snap.pods.map((pod: any) => ({
                            migrationId: `${drainId}-${pod.namespace}-${pod.name}`, // Standardize ID gen
                            drainId: drainId || '',
                            sourcePod: {
                                name: pod.name,
                                namespace: pod.namespace,
                                nodeName: pod.nodeName,
                                uid: pod.uid,
                                phase: pod.phase,
                            },
                            status: 'pending', // Default to pending, backend will update if ignored
                            startTime: new Date().toISOString(),
                        }));
                        setMigrations(initialDetails);
                    }
                    addLog('INFO', `Snapshot created: ${total} pods found`, 'SNAPSHOT');
                    break;
                }

                case 'migration_update':
                case 'pod_eviction_started':
                case 'pod_eviction_succeeded':
                case 'pod_eviction_failed':
                case 'pod_migration_ignored': {
                    const mig = data.migration;
                    if (mig) {
                        const mid = mig.migrationId || mig.MigrationID;
                        if (mid) {
                            pendingMigsRef.current[mid] = { payload: data };
                            // Trigger flush if not already running
                            addLog('DEBUG', '', 'INTERNAL', `trig-${Date.now()}`); // Hack to trigger flush cycle if no logs
                        }
                    }

                    // Specific logging
                    if (type === 'pod_eviction_failed') {
                        addLog('ERROR', msg, 'MIGRATION');
                    } else if (type === 'pod_migration_ignored') {
                        addLog('INFO', msg, 'IGNORED');
                    }
                    break;
                }

                case 'pdb_event': {
                    const action = data.action;
                    if (action === 'created') {
                        addLog('INFO', `PDB Created: ${data.namespace}/${data.pdbName}`, 'PDB');
                        // Stats usually updated via 'stats' event, but can increment here locally if needed
                    } else if (action === 'cleaned') {
                        addLog('SUCCESS', `PDB Cleaned: ${data.namespace}/${data.pdbName}`, 'PDB');
                    }
                    break;
                }

                case 'stream_closing': {
                    const reason = data.reason;
                    if (reason === 'failed' || reason === 'cancelled') {
                        setStatus(reason === 'cancelled' ? 'cancelled' : 'failed');
                        addLog('WARN', msg || 'Drain stream closing (abnormal)', 'SYSTEM');
                    } else {
                        setStatus('completed');
                        setProgress(100);
                        addLog('SUCCESS', msg || 'Drain completed successfully', 'SYSTEM');
                    }
                    stopTimer();
                    break;
                }

                case 'error':
                    addLog('ERROR', msg || 'Unknown error', 'ERROR');
                    break;

                default:
                // console.log('Unknown SSE type', type);
            }

        } catch (e) {
            console.error('SSE Parse Error', e);
        }
    }, [addLog, drainId, startTimer, stopTimer]);


    // --- Connection Logic ---
    const connectSSE = useCallback((id: string) => {
        if (eventSourceRef.current) eventSourceRef.current.close();

        // Use same endpoint structure
        const url = `${process.env.NEXT_PUBLIC_API_URL || '/api'}/v1/navy/drain/${id}/events`;
        const es = new EventSource(url, { withCredentials: true });

        es.onopen = () => {
            setIsConnected(true);
            addLog('INFO', 'Connected to event stream', 'SYSTEM');
        };

        es.onmessage = handleSSEMessage;

        es.onerror = (e) => {
            console.error('SSE Error', e);
            // Don't auto-close, let browser retry. But if it persists, we might want manual intervention.
            // addLog('WARN', 'Connection interrupted, retrying...', 'SYSTEM');
        };

        eventSourceRef.current = es;
    }, [handleSSEMessage, addLog]);


    const startDrain = async (cluster: string, node: string) => {
        reset();
        try {
            const res = await fetch(`${process.env.NEXT_PUBLIC_API_URL || '/api'}/v1/navy/drain/start`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ clusterName: cluster, nodeName: node })
            });

            if (!res.ok) {
                const err = await res.json();
                throw new Error(err.error || 'Failed to start drain');
            }

            const data = await res.json();
            setDrainId(data.drainId);
            setStatus('running'); // Optimistic
            addLog('INFO', `Drain initiated. ID: ${data.drainId}`, 'SYSTEM');
            connectSSE(data.drainId);

        } catch (e: any) {
            setStatus('failed');
            addLog('ERROR', e.message, 'SYSTEM');
            throw e;
        }
    };

    const cancelDrain = async () => {
        if (!drainId) return;
        try {
            await fetch(`${process.env.NEXT_PUBLIC_API_URL || '/api'}/v1/navy/drain/${drainId}/cancel`, {
                method: 'POST'
            });
            addLog('WARN', 'Cancellation requested...', 'SYSTEM');
        } catch (e: any) {
            addLog('ERROR', `Failed to cancel: ${e.message}`, 'SYSTEM');
        }
    };

    const reset = useCallback(() => {
        if (eventSourceRef.current) {
            eventSourceRef.current.close();
            eventSourceRef.current = null;
        }
        setDrainId(null);
        setStatus('pending');
        setRealtimeLogs([]);
        setMigrations([]);
        setStats({
            totalPods: 0, migratedPods: 0, failedPods: 0, pendingPods: 0,
            migratingPods: 0, ignoredPods: 0, pdbCount: 0
        });
        setProgress(0);
        setProgressMessage('');
        setIsConnected(false);
        resetTimer();
        processedMessagesRef.current.clear();
        pendingMigsRef.current = {};
        pendingLogsRef.current = [];
    }, [resetTimer]);

    useEffect(() => {
        return () => {
            if (eventSourceRef.current) eventSourceRef.current.close();
            if (flushTimerRef.current) clearTimeout(flushTimerRef.current);
            if (timerIntervalRef.current) clearInterval(timerIntervalRef.current);
        };
    }, []);

    return (
        <DrainContext.Provider value={{
            drainId, status, logs: realtimeLogs, migrations, stats,
            progress, progressMessage, isConnected, elapsedTime,
            startDrain, cancelDrain, reset
        }}>
            {children}
        </DrainContext.Provider>
    );
};

import React, { createContext, useContext, useEffect, useRef, useState, useCallback } from 'react';
import { DrainPodMigrationInfo, DrainStatus, LogEntry } from '@/types/safe-drain';

interface DrainContextType {
    drainId: string | null;
    status: DrainStatus;
    logs: LogEntry[];
    migrations: DrainPodMigrationInfo[];
    progress: number;
    progressMessage: string;
    isConnected: boolean;
    startDrain: (cluster: string, node: string) => Promise<void>;
    cancelDrain: () => Promise<void>;
    reset: () => void;
}

const DrainContext = createContext<DrainContextType | undefined>(undefined);

export const useDrain = () => {
    const context = useContext(DrainContext);
    if (!context) {
        throw new Error('useDrain must be used within a DrainProvider');
    }
    return context;
};

export const DrainProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
    const [drainId, setDrainId] = useState<string | null>(null);
    const [status, setStatus] = useState<DrainStatus>('pending');
    const [logs, setLogs] = useState<LogEntry[]>([]);
    const [migrations, setMigrations] = useState<DrainPodMigrationInfo[]>([]);
    const [progress, setProgress] = useState(0);
    const [progressMessage, setProgressMessage] = useState('');
    const [isConnected, setIsConnected] = useState(false);

    // Store active connection to close it on cleanup
    const eventSourceRef = useRef<EventSource | null>(null);

    const addLog = useCallback((level: string, message: string) => {
        setLogs(prev => [...prev, { timestamp: Date.now(), level, message }]);
    }, []);

    const handleMessage = useCallback((event: MessageEvent) => {
        try {
            // Check if data is prefixed "data: " which usually browser handles, but if manually parsed...
            // Browser EventSource automatically parses "data: " lines into event.data
            const payload = JSON.parse(event.data);

            switch (payload.type) {
                case 'log':
                    addLog(payload.data?.level || 'INFO', payload.message + (payload.data?.category ? ` [${payload.data.category}]` : ''));
                    if ((payload.message || '').includes('Drain not found or already completed')) {
                        setStatus('completed');
                    }
                    break;
                case 'started':
                    setStatus('running');
                    addLog('INFO', payload.message || 'Drain started');
                    break;
                case 'progress':
                    const p = payload.data?.progress ?? 0;
                    setProgress(p);
                    setProgressMessage(payload.message || '');
                    setStatus(prev => (p >= 100 ? 'completed' : (prev === 'failed' || prev === 'cancelled') ? prev : 'running'));
                    break;
                case 'completed':
                    setProgress(100);
                    setStatus('completed');
                    addLog('SUCCESS', payload.message || 'Drain completed');
                    break;
                case 'failed':
                    setStatus('failed');
                    addLog('ERROR', payload.message || 'Drain failed');
                    break;
                case 'cancelled':
                    setStatus('cancelled');
                    addLog('WARN', payload.message || 'Drain cancelled');
                    break;
                case 'migration_update':
                    if (payload.data?.migration) {
                        setMigrations(prev => {
                            const exists = prev.findIndex(m => m.migrationId === payload.data.migration.migrationId);
                            if (exists >= 0) {
                                const newArr = [...prev];
                                newArr[exists] = payload.data.migration;
                                return newArr;
                            }
                            return [...prev, payload.data.migration];
                        });
                    }
                    if (payload.data?.migration?.status === 'completed' || payload.data?.migration?.status === 'failed') {
                        // could trigger derived state update
                    }
                    break;
                case 'pod_eviction_started':
                case 'pod_eviction_succeeded':
                case 'pod_eviction_failed':
                case 'pod_migration_ignored': {
                    const mig = payload.data?.migration;
                    if (mig) {
                        setMigrations(prev => {
                            const exists = prev.findIndex(m => m.migrationId === mig.migrationId);
                            if (exists >= 0) {
                                const newArr = [...prev];
                                newArr[exists] = mig;
                                return newArr;
                            }
                            return [...prev, mig];
                        });
                    }
                    // also surface as a log line for readability
                    const lvl = payload.data?.level || (payload.type === 'pod_eviction_failed' ? 'ERROR' : 'INFO');
                    addLog(lvl, payload.message || `${payload.type}`);
                    break;
                }
                case 'error':
                    setStatus('failed');
                    addLog('ERROR', payload.message);
                    break;
                default:
                    console.log('Unknown event type', payload);
            }

            // Derive global status from progress/events if needed
            if (payload.message === 'Drain completed successfully') {
                setStatus('completed');
            }

        } catch (e) {
            console.error('Failed to parse SSE message', e);
        }
    }, [addLog]);

    const connectSSE = useCallback((id: string) => {
        if (eventSourceRef.current) {
            eventSourceRef.current.close();
        }

        // Adjust URL to match your backend API prefix
        const url = `${process.env.NEXT_PUBLIC_API_URL || '/api'}/v1/navy/drain/${id}/events`;
        // Note: Using relative path might require proxy setup or full URL

        const es = new EventSource(url); // Add withCredentials if needed via polyfill or native if same origin

        es.onopen = () => {
            setIsConnected(true);
            addLog('INFO', 'Connected to drain event stream');
        };

        es.onmessage = handleMessage;

        es.onerror = (e) => {
            console.error('SSE Error', e);
            setIsConnected(false);
            addLog('ERROR', 'Connection to event stream lost (auto-retrying)');
            // Do NOT call es.close(); native EventSource will auto-reconnect
        };

        eventSourceRef.current = es;
    }, [handleMessage, addLog]);

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
            setStatus('running');
            addLog('INFO', `Drain started: ${data.drainId}`);
            connectSSE(data.drainId);

        } catch (e: any) {
            setStatus('failed');
            addLog('ERROR', e.message);
            throw e;
        }
    };

    const cancelDrain = async () => {
        if (!drainId) return;
        try {
            await fetch(`${process.env.NEXT_PUBLIC_API_URL || '/api'}/v1/navy/drain/${drainId}/cancel`, {
                method: 'POST'
            });
            addLog('WARN', 'Cancellation requested...');
            // Status update will come via SSE usually, but we can optimistically set it
            setStatus('cancelled');
        } catch (e: any) {
            addLog('ERROR', `Failed to cancel: ${e.message}`);
        }
    };

    const reset = () => {
        if (eventSourceRef.current) {
            eventSourceRef.current.close();
            eventSourceRef.current = null;
        }
        setDrainId(null);
        setStatus('pending');
        setLogs([]);
        setMigrations([]);
        setProgress(0);
        setProgressMessage('');
        setIsConnected(false);
    };

    useEffect(() => {
        return () => {
            if (eventSourceRef.current) {
                eventSourceRef.current.close();
            }
        };
    }, []);

    return (
        <DrainContext.Provider value={{
            drainId, status, logs, migrations, progress, progressMessage, isConnected,
            startDrain, cancelDrain, reset
        }}>
            {children}
        </DrainContext.Provider>
    );
};

export type DrainStatus = 'pending' | 'running' | 'completed' | 'failed' | 'cancelled';
export type DrainPodMigrationStatus = 'pending' | 'evicting' | 'evicted' | 'creating' | 'completed' | 'failed' | 'ignored' | 'timeout';

export interface SafeDrainRequest {
    clusterName: string;
    nodeName: string;
    dryRun?: boolean;
    force?: boolean;
    timeoutSeconds?: number;
}

export interface SafeDrainResponse {
    drainId: string;
    message: string;
}

export interface SimplifiedOwnerReference {
    kind: string;
    name: string;
    uid: string;
}

export interface DrainPodInfo {
    name: string;
    namespace: string;
    nodeName: string;
    uid: string;
    phase: string;
    labels?: Record<string, string>;
    annotations?: Record<string, string>;
    ownerReferences?: SimplifiedOwnerReference[];
    createdAt?: string;
}

export interface DrainPodMigrationInfo {
    migrationId: string;
    drainId: string;
    sourcePod: DrainPodInfo;
    targetPod?: DrainPodInfo;
    status: DrainPodMigrationStatus;
    errorMessage?: string;
    startTime: string; // ISO date string
    evictionTime?: string;
    completionTime?: string;
    progress?: number;
    ownerReference?: string;
}

export interface DrainStats {
    totalPods: number;
    migratedPods: number;
    failedPods: number;
    pendingPods: number;
    migratingPods: number;
    ignoredPods: number;
    pdbCount: number;
}

export interface LogEntry {
    timestamp: number | string;
    level: string;
    message: string;
    category?: string;
    id?: string;
}

export type DrainLog = LogEntry;

export interface SSEMessage {
    type: string;
    message: string;
    data?: any;
    timestamp?: number;
}

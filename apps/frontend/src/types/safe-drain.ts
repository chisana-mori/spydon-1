export type DrainStatus = 'pending' | 'running' | 'completed' | 'failed' | 'cancelled';
export type DrainPodMigrationStatus = 'pending' | 'evicting' | 'evicted' | 'creating' | 'completed' | 'failed' | 'ignored';

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
}

export interface LogEntry {
    timestamp: number;
    level: string;
    message: string;
    category?: string;
}

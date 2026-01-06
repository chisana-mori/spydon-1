export type DrainStatus = 'pending' | 'running' | 'completed' | 'failed' | 'cancelled';

// Pod 迁移状态枚举 - 用于类型安全的状态判断
export enum PodMigrationStatus {
    Pending = 'pending',
    Evicting = 'evicting',
    Evicted = 'evicted',
    Creating = 'creating',
    Completed = 'completed',
    Failed = 'failed',
    Ignored = 'ignored',
    Timeout = 'timeout'
}

export type DrainPodMigrationStatus =
    | 'pending' | 'evicting' | 'evicted' | 'creating' | 'completed' | 'failed' | 'ignored' | 'timeout'
    | 'Pending' | 'Eviction In Progress' | 'Eviction Failed' | 'New Pod Pending' | 'Unknown' | 'Completed' | 'Ready';

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
    phaseReason?: string;  // kubectl status reason
    status?: string;       // 容器状态
    ready?: string;        // Ready 状态
    restartCount?: number; // 重启次数
    age?: string;          // Pod 年龄
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

// 预排序的迁移数据接口
export interface SortedMigrations {
    incompleteMigrations: DrainPodMigrationInfo[];
    completedMigrations: DrainPodMigrationInfo[];
}

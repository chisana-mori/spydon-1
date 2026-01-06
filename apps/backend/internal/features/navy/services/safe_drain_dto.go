package services

import (
	"time"
)

// StartDrainRequest 表示启动 drain 的请求参数
type StartDrainRequest struct {
	NodeName string `json:"nodeName" binding:"required" example:"worker-node-01"`
	DryRun   bool   `json:"dryRun" example:"false"`
	Timeout  int    `json:"timeout" example:"300"`
}

// DrainStatusResponse 表示 drain 的状态响应
type DrainStatusResponse struct {
	DrainID     string           `json:"drainId" example:"drain-123456"`
	Status      string           `json:"status" example:"running"`
	Progress    int              `json:"progress" example:"50"`
	CurrentStep string           `json:"currentStep" example:"正在驱逐Pod"`
	TotalPods   int              `json:"totalPods" example:"15"`
	EvictedPods int              `json:"evictedPods" example:"8"`
	FailedPods  []FailedPod      `json:"failedPods,omitempty"`
	StartTime   *time.Time       `json:"startTime,omitempty"`
	EndTime     *time.Time       `json:"endTime,omitempty"`
	NodeName    string           `json:"nodeName" example:"worker-node-01"`
	Message     string           `json:"message,omitempty"`
	Error       string           `json:"error,omitempty"`
	PDBsCreated []string         `json:"pdbsCreated,omitempty"`
	Statistics  *DrainStatistics `json:"statistics,omitempty"`
}

// DrainStatistics 表示 drain 操作统计
type DrainStatistics struct {
	TotalReplicaSets   int                 `json:"totalReplicaSets"`
	ProcessedRS        int                 `json:"processedRS"`
	ReadyRS            int                 `json:"readyRS"`
	FailedRS           int                 `json:"failedRS"`
	RSDetails          []ReplicaSetStatus  `json:"rsDetails,omitempty"`
	PDBStatistics      *PDBStatistics      `json:"pdbStatistics,omitempty"`
	EvictionStatistics *EvictionStatistics `json:"evictionStatistics,omitempty"`
}

// ReplicaSetStatus 表示 RS 状态
type ReplicaSetStatus struct {
	Name          string    `json:"name"`
	Namespace     string    `json:"namespace"`
	TotalPods     int       `json:"totalPods"`
	EvictedPods   int       `json:"evictedPods"`
	FailedPods    int       `json:"failedPods"`
	Status        string    `json:"status"`
	PDBCreated    bool      `json:"pdbCreated"`
	PDBName       string    `json:"pdbName,omitempty"`
	LastUpdate    time.Time `json:"lastUpdate"`
	FailureReason string    `json:"failureReason,omitempty"`
}

// PDBStatistics 表示 PDB 统计
type PDBStatistics struct {
	TotalCreated int      `json:"totalCreated"`
	TotalCleaned int      `json:"totalCleaned"`
	ActivePDBs   []string `json:"activePDBs,omitempty"`
	CleanedPDBs  []string `json:"cleanedPDBs,omitempty"`
}

// EvictionStatistics 表示驱逐统计
type EvictionStatistics struct {
	TotalAttempts       int            `json:"totalAttempts"`
	SuccessfulEvictions int            `json:"successfulEvictions"`
	FailedEvictions     int            `json:"failedEvictions"`
	RetryCount          int            `json:"retryCount"`
	FailureReasons      map[string]int `json:"failureReasons,omitempty"`
	AverageEvictionTime float64        `json:"averageEvictionTime"`
}

// RetryDrainRequest 表示重试 drain 的请求参数
type RetryDrainRequest struct {
	DryRun bool `json:"dryRun" example:"false"`
}

// GenericResponse 表示通用响应
type GenericResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message" example:"操作成功"`
	Data    interface{} `json:"data,omitempty"`
}

// DrainHistoryResponse 表示 drain 历史响应
type DrainHistoryResponse struct {
	List  []DrainStatusResponse `json:"list"`
	Total int64                 `json:"total"`
	Page  int                   `json:"page"`
	Size  int                   `json:"size"`
}

// DrainReportResponse 表示 drain 报告响应
type DrainReportResponse struct {
	DrainID          string               `json:"drainId"`
	NodeName         string               `json:"nodeName"`
	Status           string               `json:"status"`
	StartTime        time.Time            `json:"startTime"`
	EndTime          time.Time            `json:"endTime"`
	Duration         string               `json:"duration"`
	TotalPods        int                  `json:"totalPods"`
	EvictedPods      int                  `json:"evictedPods"`
	FailedPods       []FailedPod          `json:"failedPods"`
	PDBsCreated      []string             `json:"pdbsCreated"`
	PDBsCleaned      []string             `json:"pdbsCleaned"`
	Error            string               `json:"error,omitempty"`
	Snapshot         *PodSnapshotResponse `json:"snapshot,omitempty"`
	PodMigrations    []PodMigrationStatus `json:"podMigrations,omitempty"`
	MigrationSummary *MigrationSummary    `json:"migrationSummary,omitempty"`
}

// PodSnapshot 表示 Pod 快照
type PodSnapshot struct {
	NodeName  string             `json:"nodeName"`
	Timestamp time.Time          `json:"timestamp"`
	Pods      []SafeDrainPodInfo `json:"pods"`
	DrainID   string             `json:"drainId"`
}

// PodSnapshotResponse 表示 Pod 快照响应
type PodSnapshotResponse struct {
	NodeName  string             `json:"nodeName"`
	Timestamp time.Time          `json:"timestamp"`
	PodCount  int                `json:"podCount"`
	Pods      []SafeDrainPodInfo `json:"pods"`
	DrainID   string             `json:"drainId"`
}

// SafeDrainPodInfo 表示 Pod 详情
type SafeDrainPodInfo struct {
	Name            string            `json:"name"`
	Namespace       string            `json:"namespace"`
	Deployment      string            `json:"deployment"`
	ReplicaSet      string            `json:"replicaSet"`
	Status          string            `json:"status"`
	Ready           bool              `json:"ready"`
	Labels          map[string]string `json:"labels"`
	NodeName        string            `json:"nodeName"`
	Phase           string            `json:"phase"`
	ReadyContainers string            `json:"readyContainers"`
	TotalContainers string            `json:"totalContainers"`
	Restarts        string            `json:"restarts"`
	Age             string            `json:"age"`
}

// FailedPod 表示失败的 Pod
type FailedPod struct {
	PodInfo          SafeDrainPodInfo       `json:"podInfo"`
	Error            string                 `json:"error"`
	ErrorType        string                 `json:"errorType"`
	ErrorDetails     map[string]interface{} `json:"errorDetails,omitempty"`
	FailureTimestamp time.Time              `json:"failureTimestamp"`
	RetryCount       int                    `json:"retryCount"`
	CanRetry         bool                   `json:"canRetry"`
	RetrySuggestion  string                 `json:"retrySuggestion"`
}

// DrainRequest 表示内部使用的 drain 请求
type DrainRequest struct {
	NodeName    string
	DrainID     string
	DryRun      bool
	Timeout     time.Duration
	UserID      string
	ClusterID   string
	ClusterName string
}

// DrainState 表示 drain 运行状态
type DrainState struct {
	DrainID     string       `json:"drainId"`
	NodeName    string       `json:"nodeName"`
	Status      string       `json:"status"`
	CurrentStep int          `json:"currentStep"`
	TotalSteps  int          `json:"totalSteps"`
	Snapshot    *PodSnapshot `json:"snapshot,omitempty"`
	CreatedPDBs []string     `json:"createdPdbs"`
	FailedPods  []FailedPod  `json:"failedPods"`
	LastUpdate  time.Time    `json:"lastUpdate"`
	StartTime   time.Time    `json:"startTime"`
	EndTime     *time.Time   `json:"endTime,omitempty"`
	UserID      string       `json:"userId"`
	ClusterName string       `json:"clusterName"`
	Progress    int          `json:"progress"`
	Error       string       `json:"error,omitempty"`
	Message     string       `json:"message"`
}

// DeploymentGroup 表示 Deployment 分组
type DeploymentGroup struct {
	DeploymentName string             `json:"deploymentName"`
	Namespace      string             `json:"namespace"`
	Pods           []SafeDrainPodInfo `json:"pods"`
	PDBName        string             `json:"pdbName,omitempty"`
	PDBCreated     bool               `json:"pdbCreated"`
}

// PDBTrackRecord 表示 PDB 跟踪记录
type PDBTrackRecord struct {
	PDBKey           string    `json:"pdbKey"`
	DrainID          string    `json:"drainId"`
	ClusterName      string    `json:"clusterName"`
	Namespace        string    `json:"namespace"`
	PDBName          string    `json:"pdbName"`
	AssociatedRS     string    `json:"associatedRS"`
	RSUID            string    `json:"rsUid"`
	MinAvailable     string    `json:"minAvailable"`
	Status           string    `json:"status"`
	CreatedAt        time.Time `json:"createdAt"`
	LastScannedAt    time.Time `json:"lastScannedAt"`
	ExpectedDeletion time.Time `json:"expectedDeletion"`
	CleanupAttempts  int       `json:"cleanupAttempts"`
	LastError        string    `json:"lastError,omitempty"`
}

// CleanupResult 表示 PDB 清理结果
type CleanupResult struct {
	PDBKey    string    `json:"pdbKey"`
	Success   bool      `json:"success"`
	Method    string    `json:"method"`
	Error     string    `json:"error,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// EvictionResult 表示驱逐结果
type EvictionResult struct {
	PodKey     string        `json:"podKey"`
	Success    bool          `json:"success"`
	Error      string        `json:"error,omitempty"`
	Duration   time.Duration `json:"duration"`
	Timestamp  time.Time     `json:"timestamp"`
	RetryCount int           `json:"retryCount"`
}

// MigrationResult 表示迁移结果
type MigrationResult struct {
	PodKey       string        `json:"podKey"`
	NodeMigrated string        `json:"nodeMigrated,omitempty"`
	Success      bool          `json:"success"`
	WaitDuration time.Duration `json:"waitDuration"`
	Error        string        `json:"error,omitempty"`
}

// PodMigrationStatus 表示 Pod 迁移状态
type PodMigrationStatus struct {
	OldPodName      string    `json:"oldPodName" example:"nginx-deployment-abc123"`
	OldPodNamespace string    `json:"oldPodNamespace" example:"default"`
	OldNodeName     string    `json:"oldNodeName" example:"worker-node-01"`
	RSName          string    `json:"rsName" example:"nginx-deployment-7d99d8c8df"`
	NewPodName      string    `json:"newPodName,omitempty" example:"nginx-deployment-xyz789"`
	NewPodNamespace string    `json:"newPodNamespace,omitempty" example:"default"`
	NewNodeName     string    `json:"newNodeName,omitempty" example:"worker-node-02"`
	Status          string    `json:"status" example:"migrated"`
	MigrationTime   time.Time `json:"migrationTime,omitempty"`
	NewPodReadyTime time.Time `json:"newPodReadyTime,omitempty"`
	NewPodPhase     string    `json:"newPodPhase,omitempty" example:"Running"`
	FailureReason   string    `json:"failureReason,omitempty"`
	MatchConfidence float64   `json:"matchConfidence" example:"0.95"`
	MatchReason     string    `json:"matchReason" example:"rs_name_time_based"`
}

// MigrationSummary 表示迁移摘要统计
type MigrationSummary struct {
	TotalPods            int     `json:"totalPods" example:"10"`
	TotalMigrated        int     `json:"totalMigrated" example:"8"`
	SuccessfullyMigrated int     `json:"successfullyMigrated" example:"7"`
	FailedMigration      int     `json:"failedMigration" example:"1"`
	PendingMigration     int     `json:"pendingMigration" example:"0"`
	MigrationSuccessRate float64 `json:"migrationSuccessRate" example:"87.5"`
	AverageMigrationTime int64   `json:"averageMigrationTime" example:"120"`
}

// Drain 状态常量
const (
	DrainStatusPending   = "pending"
	DrainStatusRunning   = "running"
	DrainStatusPaused    = "paused"
	DrainStatusCompleted = "completed"
	DrainStatusFailed    = "failed"
	DrainStatusCanceled  = "canceled"
)

// SSE 事件类型常量
const (
	EventDrainStarted   = "drain_started"
	EventDrainCompleted = "drain_completed"
	EventDrainFailed    = "drain_failed"
	EventDrainCanceled  = "drain_canceled"
	EventDrainRetried   = "drain_retried"

	EventStepStarted   = "step_started"
	EventStepCompleted = "step_completed"
	EventStepFailed    = "step_failed"

	EventPodEvictionStarted  = "pod_eviction_started"
	EventPodEvicted          = "pod_evicted"
	EventPodEvictionFailed   = "pod_eviction_failed"
	EventPodReplacementReady = "pod_replacement_ready"

	EventRSProcessingStarted   = "rs_processing_started"
	EventRSProcessingCompleted = "rs_processing_completed"
	EventRSProcessingFailed    = "rs_processing_failed"

	EventPDBCreated        = "pdb_created"
	EventPDBCreationFailed = "pdb_creation_failed"
	EventPDBCleaned        = "pdb_cleaned"
	EventPDBCleanupFailed  = "pdb_cleanup_failed"

	EventStatisticsUpdated = "statistics_updated"
	EventProgressUpdated   = "progress_updated"
	EventProgressUpdate    = "progress_update"

	EventPDBViolationDetected = "pdb_violation_detected"
	EventPDBWaitSuggested     = "pdb_wait_suggested"
	EventPodEvictionBlocked   = "pod_eviction_blocked"
	EventRetryRecommendation  = "retry_recommendation"
	EventDrainRetrying        = "drain_retrying"
	EventSnapshotCreated      = "snapshot_created"
	EventPodEvicting          = "pod_evicting"
	EventAllPodsMigrated      = "all_pods_migrated"
	EventPDBCleaning          = "pdb_cleaning"

	EventHeartbeat      = "heartbeat"
	EventConnectionTest = "connection_test"
)

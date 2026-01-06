package services

import (
	"time"

	corev1 "k8s.io/api/core/v1"
)

// DrainStatus defines the current status of the drain operation
type DrainStatus string

const (
	DrainStatusPending   DrainStatus = "pending"
	DrainStatusRunning   DrainStatus = "running"
	DrainStatusCompleted DrainStatus = "completed"
	DrainStatusFailed    DrainStatus = "failed"
	DrainStatusCancelled DrainStatus = "canceled"
)

// DrainPodMigrationStatus defines the status of a single pod migration
type DrainPodMigrationStatus string

const (
	DrainMigrationPending   DrainPodMigrationStatus = "pending"
	DrainMigrationEvicting  DrainPodMigrationStatus = "evicting"
	DrainMigrationEvicted   DrainPodMigrationStatus = "evicted"
	DrainMigrationCreating  DrainPodMigrationStatus = "creating"
	DrainMigrationCompleted DrainPodMigrationStatus = "completed"
	DrainMigrationFailed    DrainPodMigrationStatus = "failed"
	DrainMigrationIgnored   DrainPodMigrationStatus = "ignored"
)

// SafeDrainRequest represents the request to start a drain
type SafeDrainRequest struct {
	ClusterName string `json:"clusterName" binding:"required"`
	NodeName    string `json:"nodeName" binding:"required"`
	DryRun      bool   `json:"dryRun"`
	Force       bool   `json:"force"`
	// If true, daemonset-managed pods are ignored (not blocking drain)
	IgnoreDaemonSets bool `json:"ignoreDaemonSets"`
	// If true, allow evicting/deleting pods with local storage (emptyDir/hostPath)
	DeleteLocalData bool `json:"deleteLocalData"`
	TimeoutSeconds  int  `json:"timeoutSeconds"`
}

// SafeDrainResponse is the synchronous response to a start request
type SafeDrainResponse struct {
	DrainID string `json:"drainId"`
	Message string `json:"message"`
}

// DrainPodInfo represents a pod involved in the drain
type DrainPodInfo struct {
	Name            string                     `json:"name"`
	Namespace       string                     `json:"namespace"`
	NodeName        string                     `json:"nodeName"`
	UID             string                     `json:"uid"`
	Phase           string                     `json:"phase"`
	Labels          map[string]string          `json:"labels"`
	Annotations     map[string]string          `json:"annotations"`
	OwnerReferences []SimplifiedOwnerReference `json:"ownerReferences"`
}

type SimplifiedOwnerReference struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
	UID  string `json:"uid"`
}

// DrainPodMigrationInfo tracks the migration of a single pod
type DrainPodMigrationInfo struct {
	MigrationID    string                  `json:"migrationId"`
	DrainID        string                  `json:"drainId"`
	SourcePod      DrainPodInfo            `json:"sourcePod"`
	TargetPod      *DrainPodInfo           `json:"targetPod,omitempty"`
	Status         DrainPodMigrationStatus `json:"status"`
	ErrorMessage   string                  `json:"errorMessage,omitempty"`
	StartTime      time.Time               `json:"startTime"`
	EvictionTime   *time.Time              `json:"evictionTime,omitempty"`
	CompletionTime *time.Time              `json:"completionTime,omitempty"`
}

// DrainState represents the overall state of the drain operation
type DrainState struct {
	DrainID     string      `json:"drainId"`
	ClusterName string      `json:"clusterName"`
	NodeName    string      `json:"nodeName"`
	Status      DrainStatus `json:"status"` // pending, running, completed, failed, canceled
	StartTime   time.Time   `json:"startTime"`
	EndTime     *time.Time  `json:"endTime,omitempty"`
	Error       string      `json:"error,omitempty"`
	TotalPods   int         `json:"totalPods"`
}

// SSEMessage represents a message sent to the frontend
type SSEMessage struct {
	Type      string      `json:"type"`
	DrainID   string      `json:"drainId,omitempty"`
	Message   string      `json:"message,omitempty"`
	Data      interface{} `json:"data,omitempty"`
	Timestamp string      `json:"timestamp"` // ISO8601
}

func (p *DrainPodInfo) FromK8sPod(pod *corev1.Pod) {
	p.Name = pod.Name
	p.Namespace = pod.Namespace
	p.NodeName = pod.Spec.NodeName
	p.UID = string(pod.UID)
	p.Phase = string(pod.Status.Phase)
	p.Labels = pod.Labels
	p.Annotations = pod.Annotations
	p.OwnerReferences = make([]SimplifiedOwnerReference, len(pod.OwnerReferences))
	for i, ref := range pod.OwnerReferences {
		p.OwnerReferences[i] = SimplifiedOwnerReference{
			Kind: ref.Kind,
			Name: ref.Name,
			UID:  string(ref.UID),
		}
	}
}

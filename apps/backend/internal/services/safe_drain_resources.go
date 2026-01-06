package services

import (
	"context"
	"fmt"
	"time"

	coordinationv1 "k8s.io/api/coordination/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
)

type DrainResourceManager struct {
	client kubernetes.Interface
}

func NewDrainResourceManager(client kubernetes.Interface) *DrainResourceManager {
	return &DrainResourceManager{client: client}
}

// Lease Management

func (r *DrainResourceManager) GetLeaseName(nodeName string) string {
	// Per user requirement "Application a node lease"
	// We use a predictable name so we can lock the node.
	return fmt.Sprintf("drain-lease-%s", nodeName)
}

func (r *DrainResourceManager) AcquireNodeLease(ctx context.Context, namespace, nodeName, drainID string) (*coordinationv1.Lease, error) {
	leaseName := r.GetLeaseName(nodeName)
	leases := r.client.CoordinationV1().Leases(namespace)

	// Stale threshold: lease older than 1 day is considered stale
	staleThreshold := 24 * time.Hour

	// Check if lease exists
	existing, err := leases.Get(ctx, leaseName, metav1.GetOptions{})
	if err == nil {
		// Lease exists. Check if it's stale (expired).
		isStale := false
		var leaseAge time.Duration

		// Prefer RenewTime for staleness detection, fallback to CreationTimestamp
		if existing.Spec.RenewTime != nil {
			leaseAge = time.Since(existing.Spec.RenewTime.Time)
			isStale = leaseAge > staleThreshold
		} else {
			// Fallback to creation time if RenewTime is not set
			leaseAge = time.Since(existing.CreationTimestamp.Time)
			isStale = leaseAge > staleThreshold
		}

		if isStale {
			// Stale lease detected, clean it up and proceed
			oldHolder := ""
			if existing.Spec.HolderIdentity != nil {
				oldHolder = *existing.Spec.HolderIdentity
			}
			// Note: Using fmt.Sprintf to log since we don't import logger here
			_ = leases.Delete(ctx, leaseName, metav1.DeleteOptions{})
			// Log is handled by caller, we just proceed
			_ = oldHolder // suppress unused warning, caller can log if needed
		} else {
			// Lease is still active
			return nil, fmt.Errorf("drain lease %s already exists (age: %v), node is being drained or unclean shutdown", leaseName, leaseAge.Round(time.Second))
		}
	} else if !apierrors.IsNotFound(err) {
		return nil, err
	}

	// Create Lease with RenewTime for future staleness detection
	now := metav1.NewMicroTime(time.Now())
	lease := &coordinationv1.Lease{
		ObjectMeta: metav1.ObjectMeta{
			Name:      leaseName,
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":       "robusta-safe-drain",
				"app.kubernetes.io/managed-by": "safe-drain-service",
				"drain-id":                     drainID,
				"target-node":                  nodeName,
			},
		},
		Spec: coordinationv1.LeaseSpec{
			HolderIdentity: &drainID,
			RenewTime:      &now, // Set RenewTime for staleness detection
		},
	}

	return leases.Create(ctx, lease, metav1.CreateOptions{})
}

func (r *DrainResourceManager) ReleaseNodeLease(ctx context.Context, namespace, nodeName string) error {
	leaseName := r.GetLeaseName(nodeName)
	err := r.client.CoordinationV1().Leases(namespace).Delete(ctx, leaseName, metav1.DeleteOptions{})
	if apierrors.IsNotFound(err) {
		return nil
	}
	return err
}

// PDB Management

// CreateTemporaryPDB creates a PDB for a group of pods, owned by the Lease.
// This ensures that if the Lease is deleted (drain finishes/cancels), the PDB matches GC's lifecycle.
func (r *DrainResourceManager) CreateTemporaryPDB(ctx context.Context, namespace string, lease *coordinationv1.Lease, podSelector map[string]string, nameSuffix string) (*policyv1.PodDisruptionBudget, error) {
	pdbName := fmt.Sprintf("%s-%s", lease.Name, nameSuffix)

	// Ensure owner reference to Lease
	ownerRef := metav1.OwnerReference{
		APIVersion:         "coordination.k8s.io/v1",
		Kind:               "Lease",
		Name:               lease.Name,
		UID:                lease.UID,
		Controller:         func(b bool) *bool { return &b }(true), // Controller=true means GC will cascade
		BlockOwnerDeletion: func(b bool) *bool { return &b }(true),
	}

	// minAvailable := intstr.FromInt(0) // Safe Drain usually wants to control eviction one by one?
	// Actually, if we want to "allow multi-node simultaneous operation", we rely on Global PDBs usually.
	// But if we are creating a PDB *specifically* for this drain (e.g. for pods that don't have one),
	// usually we set maxUnavailable=0 to FREEZE them from *other* disruptions and control them ourselves?
	// Or we create a PDB to *allow* eviction if none exists?
	//
	// In `auto-navy`, it created PDBs with minAvailable=N-1 (allow 1 disruption).
	// Let's assume we want maxUnavailable=1.
	maxUnavailable := intstr.FromInt(1)

	pdb := &policyv1.PodDisruptionBudget{
		ObjectMeta: metav1.ObjectMeta{
			Name:            pdbName,
			Namespace:       namespace,
			OwnerReferences: []metav1.OwnerReference{ownerRef},
			Labels: map[string]string{
				"created-by": "safe-drain-service",
			},
		},
		Spec: policyv1.PodDisruptionBudgetSpec{
			MaxUnavailable: &maxUnavailable,
			Selector: &metav1.LabelSelector{
				MatchLabels: podSelector,
			},
		},
	}

	return r.client.PolicyV1().PodDisruptionBudgets(namespace).Create(ctx, pdb, metav1.CreateOptions{})
}

// Pod Operations

func (r *DrainResourceManager) ListPodsOnNode(ctx context.Context, nodeName string) ([]corev1.Pod, error) {
	// Field selector spec.nodeName=nodeName
	listOptions := metav1.ListOptions{
		FieldSelector: "spec.nodeName=" + nodeName,
	}
	podList, err := r.client.CoreV1().Pods("").List(ctx, listOptions)
	if err != nil {
		return nil, err
	}
	return podList.Items, nil
}

func (r *DrainResourceManager) EvictPod(ctx context.Context, namespace, name string) error {
	eviction := &policyv1.Eviction{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	// Try v1 eviction
	return r.client.PolicyV1().Evictions(namespace).Evict(ctx, eviction)
}

func (r *DrainResourceManager) GetPod(ctx context.Context, namespace, name string) (*corev1.Pod, error) {
	return r.client.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
}

func (r *DrainResourceManager) IsPodEvicted(pod *corev1.Pod) bool {
	// A pod is considered "evicted" (successfully handled) if it is:
	// 1. Terminating (DeletionTimestamp != nil)
	// 2. Succeeded/Failed phase
	// 3. Or simply gone (handled by caller getting NotFound)
	return pod.DeletionTimestamp != nil || pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed
}

// DeletePod deletes a pod. If force is true, it uses 0 grace period to force delete.
func (r *DrainResourceManager) DeletePod(ctx context.Context, namespace, name string, force bool) error {
	var gp int64 = 30
	if force {
		gp = 0
	}
	return r.client.CoreV1().Pods(namespace).Delete(ctx, name, metav1.DeleteOptions{GracePeriodSeconds: &gp})
}

package services

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type fakeStateManager struct {
	states map[string]*DrainState
	err    error
}

func newFakeStateManager() *fakeStateManager {
	return &fakeStateManager{states: make(map[string]*DrainState)}
}

func (f *fakeStateManager) GetDrainState(drainID string) (*DrainState, error) {
	if f.err != nil {
		return nil, f.err
	}
	state, ok := f.states[drainID]
	if !ok {
		return nil, nil
	}
	return state, nil
}

func (f *fakeStateManager) SaveDrainState(drainID string, state *DrainState) error {
	f.states[drainID] = state
	return nil
}

func (f *fakeStateManager) DeleteDrainState(drainID string) error {
	delete(f.states, drainID)
	return nil
}

func (f *fakeStateManager) IsDrainActive(drainID string) bool {
	_, ok := f.states[drainID]
	return ok
}

func (f *fakeStateManager) UpdateDrainStatus(drainID, status string) error {
	state, ok := f.states[drainID]
	if !ok {
		return fmt.Errorf("drain state not found: %s", drainID)
	}
	state.Status = status
	return nil
}

func TestGroupPodsForEviction(t *testing.T) {
	sds := &SimpleDrainService{}

	pods := []DrainPodInfo{
		{Namespace: "ns1", Name: "pod1", Labels: map[string]string{"pod-template-hash": "hash1"}},
		{Namespace: "ns1", Name: "pod2", Labels: map[string]string{"pod-template-hash": "hash1"}},
		{Namespace: "ns2", Name: "jobpod1", Labels: map[string]string{"job-name": "job1"}},
		{Namespace: "ns3", Name: "indpod"},
	}

	groups := sds.groupPodsForEviction(pods)

	assert.Len(t, groups, 3)
	assert.Len(t, groups["rs-ns1/hash1"], 2)
	assert.Len(t, groups["job-ns2/job1"], 1)
	assert.Len(t, groups["individual-ns3/indpod"], 1)
}

func TestShouldIgnorePod(t *testing.T) {
	sds := &SimpleDrainService{}

	tests := []struct {
		name string
		pod  *DrainPodInfo
		want bool
	}{
		{
			name: "daemonset-owned",
			pod: &DrainPodInfo{
				Namespace:       "default",
				OwnerReferences: []metav1.OwnerReference{{Kind: "DaemonSet"}},
			},
			want: true,
		},
		{
			name: "statefulset-owned",
			pod: &DrainPodInfo{
				Namespace:       "default",
				OwnerReferences: []metav1.OwnerReference{{Kind: "StatefulSet"}},
			},
			want: true,
		},
		{
			name: "static-pod",
			pod: &DrainPodInfo{
				Namespace:       "default",
				OwnerReferences: []metav1.OwnerReference{},
			},
			want: true,
		},
		{
			name: "system-namespace",
			pod: &DrainPodInfo{
				Namespace:       "kube-system",
				OwnerReferences: []metav1.OwnerReference{{Kind: "ReplicaSet"}},
			},
			want: true,
		},
		{
			name: "regular-pod",
			pod: &DrainPodInfo{
				Namespace:       "default",
				OwnerReferences: []metav1.OwnerReference{{Kind: "ReplicaSet"}},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sds.shouldIgnorePod(tt.pod)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsPDBBlockError(t *testing.T) {
	sds := &SimpleDrainService{}

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil-error", nil, false},
		{"disruption-budget", errors.New("cannot evict pod due to disruption budget"), true},
		{"pdb-substring", errors.New("PDB: budget exceeded"), true},
		{"violate-phrase", errors.New("eviction would violate the pod's disruption budget"), true},
		{"other", errors.New("some other error"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sds.isPDBBlockError(tt.err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBuildDetailedEvictionError(t *testing.T) {
	sds := &SimpleDrainService{}
	pod := &DrainPodInfo{Namespace: "ns", Name: "pod1"}
	groupKey := "rs-ns/hash1"

	msg := sds.buildDetailedEvictionError(pod, groupKey, errors.New("disruption budget"), 1, 3)
	assert.Contains(t, msg, "PDB 限制")

	msg = sds.buildDetailedEvictionError(pod, groupKey, errors.New("eviction timeout"), 1, 3)
	assert.Contains(t, msg, "驱逐超时")

	msg = sds.buildDetailedEvictionError(pod, groupKey, errors.New("Forbidden: access denied"), 1, 3)
	assert.Contains(t, msg, "权限不足 (RBAC)")

	msg = sds.buildDetailedEvictionError(pod, groupKey, errors.New("pods \"foo\" not found"), 1, 3)
	assert.Contains(t, msg, "Pod 不存在")

	otherErr := errors.New("unexpected error")
	msg = sds.buildDetailedEvictionError(pod, groupKey, otherErr, 2, 3)
	assert.Contains(t, msg, otherErr.Error())
	assert.Contains(t, msg, "警告: 同一组内多个Pod驱逐失败")
}

func TestExtractDrainIDFromKey(t *testing.T) {
	sds := &SimpleDrainService{}

	assert.Equal(t, "node-1", sds.extractDrainIDFromKey("drain:pdb:node-1"))
	assert.Equal(t, "", sds.extractDrainIDFromKey("drain:pdb:"))
	assert.Equal(t, "", sds.extractDrainIDFromKey("drain:pdb"))
}

func TestFilterExpiredPDBs(t *testing.T) {
	sds := &SimpleDrainService{}
	cutoff := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)

	pdbs := []PDBInfo{
		{Key: "k1", Name: "old", CreatedAt: cutoff.Add(-time.Minute)},
		{Key: "k2", Name: "exact", CreatedAt: cutoff},
		{Key: "k3", Name: "new", CreatedAt: cutoff.Add(time.Minute)},
	}

	expired := sds.filterExpiredPDBs(pdbs, cutoff)

	assert.Len(t, expired, 1)
	assert.Equal(t, "old", expired[0].Name)
}

func TestGetClusterNameForDrain(t *testing.T) {
	sm := newFakeStateManager()
	sds := &SimpleDrainService{stateManager: sm}

	drainID := "drain-1"
	sm.states[drainID] = &DrainState{DrainID: drainID, ClusterName: "cluster-a"}

	assert.Equal(t, "cluster-a", sds.getClusterNameForDrain(drainID))
	assert.Equal(t, "default", sds.getClusterNameForDrain("unknown"))

	sm.err = errors.New("redis down")
	assert.Equal(t, "default", sds.getClusterNameForDrain(drainID))
}

func TestGenerateDrainID(t *testing.T) {
	id1 := generateDrainID()
	id2 := generateDrainID()

	assert.True(t, strings.HasPrefix(id1, "drain-"))
	assert.True(t, strings.HasPrefix(id2, "drain-"))
	assert.NotEqual(t, id1, id2)
}

package services

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	coordinationv1 "k8s.io/api/coordination/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	sharedservices "robusta-web/backend/internal/features/shared/services"
)

type fakeK8sClientFactory struct {
	client kubernetes.Interface
}

func (f *fakeK8sClientFactory) GetClient(clusterName string) (kubernetes.Interface, error) {
	return f.client, nil
}

func (*fakeK8sClientFactory) GetClientFromManager(clusterName string) (kubernetes.Interface, error) {
	return nil, nil
}

func (*fakeK8sClientFactory) GetConfig(clusterName string) (*rest.Config, error) {
	return nil, nil
}

func (*fakeK8sClientFactory) GetManager(clusterName string) (manager.Manager, error) {
	return nil, nil
}

func (*fakeK8sClientFactory) ListClusters() []string {
	return nil
}

func (*fakeK8sClientFactory) IsClusterAvailable(clusterName string) bool {
	return true
}

func (*fakeK8sClientFactory) TestConnection(ctx context.Context, clusterName string) error {
	return nil
}

func newTestDrainResourceManager(client kubernetes.Interface) *DrainResourceManager {
	return &DrainResourceManager{
		clientFactory: &fakeK8sClientFactory{client: client},
		versionCache:  sharedservices.NewClusterVersionCache(time.Minute),
	}
}

func TestEnsureNodeLease_DeniesReuseWithinOneDay(t *testing.T) {
	ctx := context.Background()
	client := fake.NewSimpleClientset()
	drm := newTestDrainResourceManager(client)

	namespace := "ns1"
	nodeName := "node1"
	leaseName := "drain-lease-" + nodeName
	now := time.Now()

	_, err := client.CoordinationV1().Leases(namespace).Create(ctx, &coordinationv1.Lease{
		ObjectMeta: metav1.ObjectMeta{
			Name:              leaseName,
			Namespace:         namespace,
			CreationTimestamp: metav1.NewTime(now.Add(-12 * time.Hour)),
		},
	}, metav1.CreateOptions{})
	assert.NoError(t, err)

	_, err = drm.EnsureNodeLease(ctx, "cluster1", namespace, nodeName, "drain-1")
	assert.Error(t, err)
}

func TestPDBLeaseOwnerRefsAreIsolatedPerNode(t *testing.T) {
	ctx := context.Background()
	client := fake.NewSimpleClientset()
	drm := newTestDrainResourceManager(client)

	ns := "default"
	pdbName := "test-pdb"

	pdb := &policyv1.PodDisruptionBudget{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pdbName,
			Namespace: ns,
			OwnerReferences: []metav1.OwnerReference{{
				Kind: "ReplicaSet",
				Name: "rs-1",
			}},
		},
	}
	_, err := client.PolicyV1().PodDisruptionBudgets(ns).Create(ctx, pdb, metav1.CreateOptions{})
	assert.NoError(t, err)

	leaseA := &coordinationv1.Lease{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "drain-lease-node-a",
			Namespace: ns,
			UID:       types.UID("lease-a-uid"),
		},
	}
	leaseB := &coordinationv1.Lease{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "drain-lease-node-b",
			Namespace: ns,
			UID:       types.UID("lease-b-uid"),
		},
	}

	// Attach both leases
	err = drm.ensureLeaseOwnerRefOnPDB(ctx, "cluster1", ns, pdbName, leaseA)
	assert.NoError(t, err)
	err = drm.ensureLeaseOwnerRefOnPDB(ctx, "cluster1", ns, pdbName, leaseB)
	assert.NoError(t, err)

	// Remove lease A reference and ensure lease B remains
	err = drm.removeLeaseOwnerRefFromPDB(ctx, "cluster1", ns, pdbName, leaseA.Name)
	assert.NoError(t, err)

	finalPDB, err := client.PolicyV1().PodDisruptionBudgets(ns).Get(ctx, pdbName, metav1.GetOptions{})
	assert.NoError(t, err)

	var rsCount, leaseACount, leaseBCount int
	for _, ref := range finalPDB.OwnerReferences {
		switch {
		case ref.Kind == "ReplicaSet":
			rsCount++
		case ref.Kind == "Lease" && ref.Name == leaseA.Name:
			leaseACount++
		case ref.Kind == "Lease" && ref.Name == leaseB.Name:
			leaseBCount++
		}
	}

	assert.Equal(t, 1, rsCount)
	assert.Equal(t, 0, leaseACount)
	assert.Equal(t, 1, leaseBCount)
}

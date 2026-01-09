package nodesync

import (
	"context"
	"testing"

	"robusta-web/backend/internal/db"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// testDevice 测试用简化设备模型
type testDevice struct {
	ID        int    `gorm:"column:id"`
	CICode    string `gorm:"column:ci_code"`
	Cluster   string `gorm:"column:cluster"`
	ClusterID int    `gorm:"column:cluster_id"`
	K8sStatus string `gorm:"column:k8s_status"`
	Role      string `gorm:"column:role"`
}

func (testDevice) TableName() string {
	return "device"
}

// setupTestDB 创建内存 SQLite 测试数据库
func setupTestDB(t *testing.T) *db.Database {
	navyDB, err := db.Initialize(":memory:")
	require.NoError(t, err)

	// 创建 device 表
	err = navyDB.Exec(`
		CREATE TABLE IF NOT EXISTS device (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			ci_code VARCHAR(255),
			ip VARCHAR(50),
			cluster VARCHAR(255) DEFAULT '',
			cluster_id INTEGER DEFAULT 0,
			k8s_status VARCHAR(50) DEFAULT '',
			role VARCHAR(100) DEFAULT '',
			status VARCHAR(50),
			created_at DATETIME,
			updated_at DATETIME
		)
	`).Error
	require.NoError(t, err)

	return navyDB
}

// createTestDevice 创建测试设备
func createTestDevice(t *testing.T, navyDB *db.Database, ciCode string) int {
	result := navyDB.Exec(`
		INSERT INTO device (ci_code, ip, status, created_at, updated_at)
		VALUES (?, ?, ?, datetime('now'), datetime('now'))
	`, ciCode, "192.168.1.1", "active")
	require.NoError(t, result.Error)

	var id int
	navyDB.Raw("SELECT last_insert_rowid()").Scan(&id)
	return id
}

// getTestDevice 查询测试设备
func getTestDevice(t *testing.T, navyDB *db.Database, id int) testDevice {
	var device testDevice
	err := navyDB.First(&device, id).Error
	require.NoError(t, err)
	return device
}

// createTestNode 创建测试 K8s Node
func createTestNode(name string, ready bool, unschedulable bool, roles ...string) *corev1.Node {
	labels := make(map[string]string)
	for _, role := range roles {
		labels["node-role.kubernetes.io/"+role] = ""
	}

	conditions := []corev1.NodeCondition{
		{
			Type:   corev1.NodeReady,
			Status: corev1.ConditionFalse,
		},
	}
	if ready {
		conditions[0].Status = corev1.ConditionTrue
	}

	return &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
		Spec: corev1.NodeSpec{
			Unschedulable: unschedulable,
		},
		Status: corev1.NodeStatus{
			Conditions: conditions,
		},
	}
}

func TestExtractNodeInfo(t *testing.T) {
	tests := []struct {
		name           string
		node           *corev1.Node
		expectedRole   string
		expectedStatus string
	}{
		{
			name:           "Ready worker node",
			node:           createTestNode("node-1", true, false, "worker"),
			expectedRole:   "worker",
			expectedStatus: "Ready",
		},
		{
			name:           "NotReady master node",
			node:           createTestNode("node-2", false, false, "master", "control-plane"),
			expectedRole:   "master,control-plane",
			expectedStatus: "NotReady",
		},
		{
			name:           "Unschedulable node",
			node:           createTestNode("node-3", true, true, "worker"),
			expectedRole:   "worker",
			expectedStatus: "Unschedulable",
		},
		{
			name:           "Node without role",
			node:           createTestNode("node-4", true, false),
			expectedRole:   "",
			expectedStatus: "Ready",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			role, status := ExtractNodeInfo(tt.node)
			if tt.expectedRole != "" {
				assert.NotEmpty(t, role)
			}
			assert.Equal(t, tt.expectedStatus, status)
		})
	}
}

func TestUpdateDeviceFromNode(t *testing.T) {
	ctx := context.Background()
	navyDB := setupTestDB(t)

	ciCode := "test-node-1"
	deviceID := createTestDevice(t, navyDB, ciCode)

	node := createTestNode(ciCode, true, false, "worker")

	err := UpdateDeviceFromNode(ctx, navyDB, "test-cluster", node)
	require.NoError(t, err)

	device := getTestDevice(t, navyDB, deviceID)
	assert.Equal(t, "test-cluster", device.Cluster)
	// ClusterID 不再被更新，应保持默认值
	assert.Equal(t, 0, device.ClusterID)
	assert.Equal(t, "Ready", device.K8sStatus)
	// Role 不再被更新，应保持默认值
	assert.Equal(t, "", device.Role)
}

func TestUpdateDeviceFromNode_DeviceNotFound(t *testing.T) {
	ctx := context.Background()
	navyDB := setupTestDB(t)

	node := createTestNode("non-existent-node", true, false, "worker")

	err := UpdateDeviceFromNode(ctx, navyDB, "test-cluster", node)
	assert.NoError(t, err)
}

func TestClearDeviceClusterInfo(t *testing.T) {
	ctx := context.Background()
	navyDB := setupTestDB(t)

	ciCode := "test-node-clear"
	deviceID := createTestDevice(t, navyDB, ciCode)

	// 先设置集群信息
	navyDB.Exec(`UPDATE device SET cluster = ?, cluster_id = ?, k8s_status = ?, role = ? WHERE id = ?`,
		"old-cluster", 99, "Ready", "worker", deviceID)

	err := ClearDeviceClusterInfo(ctx, navyDB, ciCode)
	require.NoError(t, err)

	device := getTestDevice(t, navyDB, deviceID)
	assert.Empty(t, device.Cluster)
	assert.Equal(t, 0, device.ClusterID)
	assert.Empty(t, device.K8sStatus)
	// Role 不再被清除，应保持原值
	assert.Equal(t, "worker", device.Role)
}

func TestCleanOrphanDevices(t *testing.T) {
	ctx := context.Background()
	navyDB := setupTestDB(t)

	device1ID := createTestDevice(t, navyDB, "node-1")
	device2ID := createTestDevice(t, navyDB, "node-2")
	device3ID := createTestDevice(t, navyDB, "node-3")

	clusterID := 1
	for _, id := range []int{device1ID, device2ID, device3ID} {
		navyDB.Exec(`UPDATE device SET cluster = ?, cluster_id = ?, k8s_status = ? WHERE id = ?`,
			"test-cluster", clusterID, "Ready", id)
	}

	activeNodes := []string{"node-1", "node-2"}
	err := CleanOrphanDevices(ctx, navyDB, clusterID, activeNodes)
	require.NoError(t, err)

	device1 := getTestDevice(t, navyDB, device1ID)
	device2 := getTestDevice(t, navyDB, device2ID)
	assert.Equal(t, "test-cluster", device1.Cluster)
	assert.Equal(t, "test-cluster", device2.Cluster)

	device3 := getTestDevice(t, navyDB, device3ID)
	assert.Empty(t, device3.Cluster)
	assert.Equal(t, 0, device3.ClusterID)
}

func TestCleanOrphanDevices_EmptyActiveNodes(t *testing.T) {
	ctx := context.Background()
	navyDB := setupTestDB(t)

	deviceID := createTestDevice(t, navyDB, "orphan-node")
	clusterID := 1

	navyDB.Exec(`UPDATE device SET cluster = ?, cluster_id = ? WHERE id = ?`,
		"test-cluster", clusterID, deviceID)

	err := CleanOrphanDevices(ctx, navyDB, clusterID, []string{})
	require.NoError(t, err)

	device := getTestDevice(t, navyDB, deviceID)
	assert.Empty(t, device.Cluster)
}

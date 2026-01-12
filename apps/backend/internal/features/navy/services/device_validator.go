package services

import (
	"context"
	"fmt"
	"strings"

	"robusta-web/backend/internal/db"
	"robusta-web/backend/internal/models/navy"
	"robusta-web/backend/internal/pkg/nodesync"
)

// DeviceValidator 设备操作验证器
type DeviceValidator struct {
	navyDB          *db.Database
	nodeSyncManager *nodesync.Manager
}

// NewDeviceValidator 创建验证器
func NewDeviceValidator(
	navyDB *db.Database,
	nodeSyncManager *nodesync.Manager,
) *DeviceValidator {
	return &DeviceValidator{
		navyDB:          navyDB,
		nodeSyncManager: nodeSyncManager,
	}
}

// ValidateDevicesExist 验证设备是否存在于 K8s 集群中（实时检查）
func (v *DeviceValidator) ValidateDevicesExist(ctx context.Context, ciCodes []string, metadata map[string]any) error {
	// 从数据库获取设备信息（主要为了获取集群信息）
	var devices []navy.Device
	if err := v.navyDB.Where("ci_code IN ?", ciCodes).Find(&devices).Error; err != nil {
		return fmt.Errorf("查询设备失败: %w", err)
	}

	// 构建设备 CI Code 到集群的映射
	deviceClusterMap := make(map[string]string)
	for _, d := range devices {
		deviceClusterMap[d.CICode] = d.Cluster
	}

	// 检查是否所有设备都在数据库中且有集群信息
	var missingInDB []string
	var noCluster []string
	for _, code := range ciCodes {
		cluster, exists := deviceClusterMap[code]
		if !exists {
			missingInDB = append(missingInDB, code)
		} else if cluster == "" {
			noCluster = append(noCluster, code)
		}
	}

	if len(missingInDB) > 0 {
		return fmt.Errorf("以下设备不存在于数据库: %v", missingInDB)
	}

	if len(noCluster) > 0 {
		return fmt.Errorf("以下设备未关联到集群: %v", noCluster)
	}

	// 按集群分组设备
	clusterDevices := make(map[string][]string)
	for _, code := range ciCodes {
		cluster := deviceClusterMap[code]
		clusterDevices[cluster] = append(clusterDevices[cluster], code)
	}

	// 从缓存检查每个集群中的节点是否存在
	var notFoundInCluster []string
	for clusterName, deviceCodes := range clusterDevices {
		// 检查每个节点是否存在（使用小写 CI Code 作为节点名，从缓存查询）
		for _, ciCode := range deviceCodes {
			nodeName := strings.ToLower(ciCode)
			node, err := v.nodeSyncManager.GetNode(clusterName, nodeName)
			if err != nil {
				return fmt.Errorf("查询节点 %s 失败: %w", nodeName, err)
			}
			if node == nil {
				notFoundInCluster = append(notFoundInCluster, fmt.Sprintf("%s (集群: %s)", ciCode, clusterName))
			}
		}
	}

	if len(notFoundInCluster) > 0 {
		return fmt.Errorf("以下设备在集群中不存在: %v", notFoundInCluster)
	}

	return nil
}

// ValidateClusterAvailable 验证集群是否可用（K8s 连接测试）
func (v *DeviceValidator) ValidateClusterAvailable(ctx context.Context, ciCodes []string, metadata map[string]any) error {
	var devices []navy.Device
	if err := v.navyDB.Where("ci_code IN ?", ciCodes).Find(&devices).Error; err != nil {
		return fmt.Errorf("查询设备失败: %w", err)
	}

	// 按集群去重
	clusterMap := make(map[string]bool)
	for _, d := range devices {
		if d.Cluster != "" {
			clusterMap[d.Cluster] = true
		}
	}

	// 验证每个集群都可连接
	for clusterName := range clusterMap {
		_, err := v.nodeSyncManager.GetClient(clusterName)
		if err != nil {
			return fmt.Errorf("集群 %s 不可用: %w", clusterName, err)
		}
	}

	return nil
}

// ValidateDrainPrerequisite 验证 Drain 前置条件（节点已 cordoned）
func (v *DeviceValidator) ValidateDrainPrerequisite(ctx context.Context, ciCodes []string, metadata map[string]any) error {
	// 从 metadata 中提取集群信息
	clusterName, _ := metadata["cluster"].(string)
	if clusterName == "" {
		// 如果 metadata 没有集群信息，从数据库查询
		var devices []navy.Device
		if err := v.navyDB.Where("ci_code IN ?", ciCodes).Find(&devices).Error; err != nil {
			return fmt.Errorf("查询设备失败: %w", err)
		}
		if len(devices) == 0 || devices[0].Cluster == "" {
			return fmt.Errorf("无法确定集群信息")
		}
		clusterName = devices[0].Cluster
	}

	// 从缓存检查每个节点的 Unschedulable 状态
	for _, ciCode := range ciCodes {
		nodeName := strings.ToLower(ciCode)
		node, err := v.nodeSyncManager.GetNode(clusterName, nodeName)
		if err != nil {
			return fmt.Errorf("查询节点 %s 失败: %w", nodeName, err)
		}
		if node == nil {
			return fmt.Errorf("节点 %s 不存在", nodeName)
		}

		if !node.Spec.Unschedulable {
			return fmt.Errorf("节点 %s 尚未被标记为不可调度，请先执行 Cordon 操作", nodeName)
		}
	}

	return nil
}
